package orchestration

import (
	"context"
	"sync"

	"log"

	"github.com/koscakluka/ema/core/audio"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/core/speechtotext"
	"github.com/koscakluka/ema/core/texttospeech"
)

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

	llm                    LLM
	speechToTextClient     SpeechToText
	textToSpeechClient     TextToSpeech
	audioInput             AudioInput
	audioOutput            AudioOutput
	interruptionClassifier InterruptionClassifier
}

func NewOrchestrator(opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		AlwaysRecording: false,
		IsRecording:     false,
		IsSpeaking:      false,
		transcripts:     make(chan string, 10), // TODO: Figure out good valiues for this
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.interruptionClassifier == nil {
		o.interruptionClassifier = NewSimpleInterruptionClassifier(o.llm)
	}

	return o
}

type OrchestratorOption func(*Orchestrator)

func WithLLM(client LLM) OrchestratorOption {
	return func(o *Orchestrator) {
		o.llm = client
	}
}

func WithSpeechToTextClient(client SpeechToText) OrchestratorOption {
	return func(o *Orchestrator) {
		o.speechToTextClient = client
	}
}

func WithTextToSpeechClient(client TextToSpeech) OrchestratorOption {
	return func(o *Orchestrator) {
		o.textToSpeechClient = client
		o.IsSpeaking = true
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

func WithInterruptionClassifier(classifier InterruptionClassifier) OrchestratorOption {
	return func(o *Orchestrator) {
		o.interruptionClassifier = classifier
	}
}

func (o *Orchestrator) Orchestrate(ctx context.Context, opts ...OrchestrateOption) {
	options := OrchestrateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if o.textToSpeechClient != nil {
		ttsOptions := []texttospeech.TextToSpeechOption{
			texttospeech.WithAudioCallback(func(audio []byte) {
				if options.onAudio != nil {
					options.onAudio(audio)
				}

				if o.audioOutput == nil {
					return
				}

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

				if options.onAudioEnded != nil {
					options.onAudioEnded(transcript)
				}

				if o.audioOutput == nil {
					return
				}

				if !o.IsSpeaking || o.canceled {
					o.audioOutput.ClearBuffer()
					return
				}

				o.audioOutput.SendAudio([]byte{})
				o.audioOutput.AwaitMark()
			}),
		}
		if o.audioOutput != nil {
			ttsOptions = append(ttsOptions, texttospeech.WithEncodingInfo(o.audioOutput.EncodingInfo()))
		}

		if err := o.textToSpeechClient.OpenStream(context.TODO(), ttsOptions...); err != nil {
			log.Printf("Failed to open deepgram speech stream: %v", err)
		}
	}

	if o.speechToTextClient != nil {
		sttOptions := []speechtotext.TranscriptionOption{
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

				o.SendPrompt(transcript, opts...)
			}),
		}
		if o.audioInput != nil {
			sttOptions = append(sttOptions, speechtotext.WithEncodingInfo(o.audioInput.EncodingInfo()))
		}

		if err := o.speechToTextClient.Transcribe(context.TODO(), sttOptions...); err != nil {
			log.Fatalf("Failed to start transcribing: %v", err)
		}
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

			response, _ := o.llm.Prompt(context.TODO(), transcript,
				llms.WithMessages(messages...),
				llms.WithTools(o.tools...),
				llms.WithStream(
					func(data string) {
						if o.canceled {
							return
						}

						if options.onResponse != nil {
							options.onResponse(data)
						}
						if o.textToSpeechClient != nil {
							if err := o.textToSpeechClient.SendText(data); err != nil {
								log.Printf("Failed to send text to deepgram: %v", err)
							}
						}
					}))

			o.messages = append(o.messages, response...)
			if o.textToSpeechClient != nil {
				if err := o.textToSpeechClient.FlushBuffer(); err != nil {
					log.Printf("Failed to flush buffer: %v", err)
				}
			} else {
				o.activePrompt = nil
				o.canceled = false
				o.promptEnded.Done()

			}
			if options.onResponseEnd != nil {
				options.onResponseEnd()
			}
		}
	}()

	if o.audioInput != nil && o.speechToTextClient != nil {
		o.AlwaysRecording = true
		go func() {
			if err := o.audioInput.Stream(ctx, func(audio []byte) {
				if err := o.speechToTextClient.SendAudio(audio); err != nil {
					log.Printf("Failed to send audio to speech to text client: %v", err)
				}
			}); err != nil {
				log.Printf("Failed to start audio input streaming: %v", err)
			}
		}()
	} else if o.audioInput != nil && o.speechToTextClient == nil {
		log.Println("Warning: skip starting input audio stream: audio input set but speech to text client is not set")
	}
}

type OrchestrateOptions struct {
	onTranscription        func(transcript string)
	onInterimTranscription func(transcript string)
	onSpeakingStateChanged func(isSpeaking bool)
	onResponse             func(response string)
	onResponseEnd          func()
	onCancellation         func()
	onAudio                func(audio []byte)
	onAudioEnded           func(transcript string)
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

func WithAudioCallback(callback func(audio []byte)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onAudio = callback
	}
}

func WithAudioEndedCallback(callback func(transcript string)) OrchestrateOption {
	return func(o *OrchestrateOptions) {
		o.onAudioEnded = callback
	}
}

func (o *Orchestrator) Close() {
	// TODO: Make sure that deepgramClient is closed and no longer transcribing
	// before closing the channel
	close(o.transcripts)
}

func (o *Orchestrator) SendPrompt(prompt string, opts ...OrchestrateOption) {
	options := OrchestrateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if o.activePrompt != nil && !o.interruption {
		o.interruption = true
	}

	passthrough := &prompt
	if o.interruption {
		if o.interruptionClassifier != nil {
			interruption, err := o.interruptionClassifier.Classify(prompt, o.messages, ClassifyWithTools(o.tools))
			if err != nil {
				// TODO: Retry?
				log.Printf("Failed to classify interruption: %v", err)
			} else {
				passthrough, err = o.respondToInterruption(prompt, interruption, options)
				if err != nil {
					log.Printf("Failed to respond to interruption: %v", err)
				}
			}
		}
		o.interruption = false
	}
	if passthrough != nil {
		if options.onTranscription != nil {
			options.onTranscription(prompt)
		}
		o.transcripts <- *passthrough
	}
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
	if o.audioOutput != nil {
		o.audioOutput.ClearBuffer()
	}
}

func (o *Orchestrator) SetAlwaysRecording(isAlwaysRecording bool) {
	o.AlwaysRecording = isAlwaysRecording
}

type LLM interface {
	Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error)
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

type InterruptionClassifier interface {
	Classify(prompt string, history []llms.Message, opts ...ClassifyOption) (interruptionType, error)
}
