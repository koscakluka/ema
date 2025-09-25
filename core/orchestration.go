package orchestration

import (
	"context"
	"sync"

	"log"

	"github.com/koscakluka/ema/core/audio"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/core/llms/groq"
	"github.com/koscakluka/ema/core/speechtotext"
	"github.com/koscakluka/ema/core/texttospeech"
)

const bufferSize = 128

type Orchestrator struct {
	AlwaysRecording bool
	IsRecording     bool
	IsSpeaking      bool

	messages []llms.Message

	transcripts  chan string
	activePrompt *string
	promptEnded  sync.WaitGroup
	interruption bool
	canceled     bool

	tools []llms.Tool

	speechToTextClient SpeechToText
	textToSpeechClient TextToSpeech
	audioInput         AudioInput
	audioOutput        AudioOutput
}

func NewOrchestrator(opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		AlwaysRecording: true,
		IsRecording:     false,
		IsSpeaking:      true,
		transcripts:     make(chan string, 10), // TODO: Figure out good valiues for this
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

type OrchestratorOption func(*Orchestrator)

func WithSpeechToTextClient(client SpeechToText) OrchestratorOption {
	return func(o *Orchestrator) {
		o.speechToTextClient = client
	}
}

func WithTextToSpeechClient(client TextToSpeech) OrchestratorOption {
	return func(o *Orchestrator) {
		o.textToSpeechClient = client
	}
}

func WithAudioInput(client AudioInput) OrchestratorOption {
	return func(o *Orchestrator) {
		o.audioInput = client
	}
}

func WithAudioOutput(client AudioOutput) OrchestratorOption {
	return func(o *Orchestrator) {
		o.audioOutput = client
	}
}

func WithTools(tools ...llms.Tool) OrchestratorOption {
	return func(o *Orchestrator) {
		o.tools = tools
	}
}

func WithOrchestrationTools() OrchestratorOption {
	return func(o *Orchestrator) {
		o.tools = append(o.tools, orchestrationTools(o)...)
	}
}

func (o *Orchestrator) Orchestrate(ctx context.Context, opts ...OrchestrateOption) {
	options := OrchestrateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	client := groq.NewClient()

	if err := o.textToSpeechClient.OpenStream(context.TODO(),
		texttospeech.WithEncodingInfo(o.audioOutput.EncodingInfo()),
		texttospeech.WithAudioCallback(func(audio []byte) {
			if !o.IsSpeaking || o.canceled {
				o.audioOutput.ClearBuffer()
				return
			}

			o.audioOutput.SendAudio(audio)
		}),
		texttospeech.WithAudioEndedCallback(func(transcript string) {
			defer func() {
				o.activePrompt = nil
				o.canceled = false
				o.promptEnded.Done()
			}()
			if !o.IsSpeaking || o.canceled {
				o.audioOutput.ClearBuffer()
				return
			}

			o.audioOutput.SendAudio([]byte{})
			o.audioOutput.AwaitMark()
		}),
	); err != nil {
		log.Printf("Failed to open deepgram speech stream: %v", err)
	}

	if err := o.speechToTextClient.Transcribe(context.TODO(),
		speechtotext.WithEncodingInfo(o.audioInput.EncodingInfo()),
		speechtotext.WithSpeechStartedCallback(func() {
			if options.onSpeakingStateChanged != nil {
				options.onSpeakingStateChanged(true)
			}
		}),
		speechtotext.WithSpeechEndedCallback(func() {
			if options.onSpeakingStateChanged != nil {
				options.onSpeakingStateChanged(false)
			}
		}),
		speechtotext.WithInterimTranscriptionCallback(func(transcript string) {
			if o.activePrompt != nil && !o.interruption {
				o.interruption = true
			}

			if options.onInterimTranscription != nil {
				options.onInterimTranscription(transcript)
			}
		}),
		speechtotext.WithTranscriptionCallback(func(transcript string) {
			if options.onInterimTranscription != nil {
				options.onInterimTranscription("")
			}

			if o.activePrompt != nil && !o.interruption {
				o.interruption = true
			}
			passthrough := &transcript
			if o.interruption {
				interruption, err := o.classifyInterruption(transcript)
				if err != nil {
					// TODO: Retry?
					log.Printf("Failed to classify interruption: %v", err)
				} else {
					passthrough, err = o.respondToInterruption(transcript, interruption, options)
					if err != nil {
						log.Printf("Failed to respond to interruption: %v", err)
					}
				}
				o.interruption = false
			}
			if passthrough != nil {
				if options.onTranscription != nil {
					options.onTranscription(transcript)
				}
				o.transcripts <- *passthrough
			}

		}),
	); err != nil {
		log.Fatalf("Failed to start transcribing: %v", err)
	}

	go func() {
		for transcript := range o.transcripts {
			if o.activePrompt != nil {
				o.promptEnded.Wait()
			}
			o.activePrompt = &transcript
			o.promptEnded.Add(1)

			messages := o.messages
			o.messages = append(o.messages, llms.Message{
				Role:    llms.MessageRoleUser,
				Content: transcript,
			})

			response, _ := client.Prompt(context.TODO(), transcript,
				groq.WithMessages(messages...),
				groq.WithTools(o.tools...),
				groq.WithStream(
					func(data string) {
						if options.onResponse != nil {
							options.onResponse(data)
						}
						if err := o.textToSpeechClient.SendText(data); err != nil {
							log.Printf("Failed to send text to deepgram: %v", err)
						}
					}))

			o.messages = append(o.messages, llms.Message{
				Role:    llms.MessageRoleAssistant,
				Content: response,
			})
			if err := o.textToSpeechClient.FlushBuffer(); err != nil {
				log.Printf("Failed to flush buffer: %v", err)
			}
			if options.onResponseEnd != nil {
				options.onResponseEnd()
			}
		}
	}()

	go func() {
		if err := o.audioInput.Stream(ctx, func(audio []byte) {
			if err := o.speechToTextClient.SendAudio(audio); err != nil {
				log.Printf("Failed to send audio to speech to text client: %v", err)
			}
		}); err != nil {
			log.Printf("Failed to start audio input streaming: %v", err)
		}
	}()
}

type OrchestrateOptions struct {
	onTranscription        func(transcript string)
	onInterimTranscription func(transcript string)
	onSpeakingStateChanged func(isSpeaking bool)
	onResponse             func(response string)
	onResponseEnd          func()
	onCancellation         func()
}

type OrchestrateOption func(*OrchestrateOptions)

func WithTranscriptionCallback(callback func(transcript string)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onTranscription = callback
	}
}

func WithInterimTranscriptionCallback(callback func(transcript string)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onInterimTranscription = callback
	}
}

func WithSpeakingStateChangedCallback(callback func(isSpeaking bool)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onSpeakingStateChanged = callback
	}
}

func WithResponseCallback(callback func(response string)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onResponse = callback
	}
}

func WithResponseEndCallback(callback func()) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onResponseEnd = callback
	}
}

func WithCancellationCallback(callback func()) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onCancellation = callback
	}
}

func (o *Orchestrator) Close() {
	// TODO: Make sure that deepgramClient is closed and no longer transcribing
	// before closing the channel
	close(o.transcripts)
}

func (o *Orchestrator) SendAudio(audio []byte) error {
	if o.speechToTextClient == nil {
		log.Println("Warning: SendAudio called but speech to text client is not set")
		return nil
	}

	if o.IsRecording || o.AlwaysRecording {
		return o.speechToTextClient.SendAudio(audio)
	}

	return nil
}

func (o *Orchestrator) SetSpeaking(isSpeaking bool) {
	o.IsSpeaking = isSpeaking
	o.audioOutput.ClearBuffer()
}

func (o *Orchestrator) SetAlwaysRecording(isAlwaysRecording bool) {
	o.AlwaysRecording = isAlwaysRecording
}

type SpeechToText interface {
	Transcribe(ctx context.Context, opts ...speechtotext.TranscriptionOption) error
	SendAudio(audio []byte) error
}

type TextToSpeech interface {
	OpenStream(ctx context.Context, opts ...texttospeech.TextToSpeechOption) error
	SendText(text string) error
	FlushBuffer() error
}

type AudioInput interface {
	EncodingInfo() audio.EncodingInfo
	Stream(ctx context.Context, onAudio func(audio []byte)) error
	Close()
}

type AudioOutput interface {
	EncodingInfo() audio.EncodingInfo
	SendAudio(audio []byte) error
	AwaitMark() error
	ClearBuffer()
}
