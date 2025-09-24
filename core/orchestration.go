package orchestration

import (
	"context"
	"sync"

	"log"

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

	tools []groq.Tool

	speechToTextClient SpeechToText
	textToSpeechClient TextToSpeech
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

	o.tools = []groq.Tool{
		groq.NewTool("recording_control", "Turn on or off sound recording, might be referred to as 'listening'",
			map[string]groq.ParameterBase{
				"is_recording": {Type: "boolean", Description: "Whether to record or not"},
			},
			func(parameters struct {
				IsRecording bool `json:"is_recording"`
			}) (string, error) {
				o.AlwaysRecording = parameters.IsRecording
				return "Success. Respond with a very short phrase", nil
			}),
		groq.NewTool("speaking_control", "Turn off agent's speaking ability. Might be referred to as 'muting'",
			map[string]groq.ParameterBase{
				"is_speaking": {Type: "boolean", Description: "Wheather to speak or not"},
			},
			func(parameters struct {
				IsSpeaking bool `json:"is_speaking"`
			}) (string, error) {
				o.IsSpeaking = parameters.IsSpeaking
				return "Success. Respond with a very short phrase", nil
			}),
	}

	return o
}

type Callbacks struct {
	OnTranscription        func(transcript string)
	OnInterimTranscription func(transcript string)
	OnSpeakingStateChanged func(isSpeaking bool)
	OnResponse             func(response string)
	OnResponseEnd          func()
	OnCancellation         func()
	OnAudio                func(audio []byte)
	OnAudioEnd             func(transcript string)
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

func (o *Orchestrator) ListenForSpeech(ctx context.Context, callbacks Callbacks) {
	client := groq.NewClient()
	if o.speechToTextClient == nil {
		if callbacks.OnAudio != nil {
			log.Println("Warning: onAudio callback set but speech to text client is not set")
		}
		if callbacks.OnAudioEnd != nil {
			log.Println("Warning: onAudioEnd callback set but speech to text client is not set")
		}
	}

	if err := o.textToSpeechClient.OpenStream(context.TODO(),
		texttospeech.WithAudioCallback(func(audio []byte) {
			if !o.IsSpeaking || o.canceled {
				return
			}

			if callbacks.OnAudio != nil {
				callbacks.OnAudio(audio)
			}
		}),
		texttospeech.WithAudioEndedCallback(func(transcript string) {
			o.activePrompt = nil
			o.canceled = false
			o.promptEnded.Done()
			if !o.IsSpeaking || o.canceled {
				return
			}

			if callbacks.OnAudioEnd != nil {
				callbacks.OnAudioEnd(transcript)
			}

		}),
	); err != nil {
		log.Printf("Failed to open deepgram speech stream: %v", err)
	}

	if err := o.speechToTextClient.Transcribe(context.TODO(),
		speechtotext.WithSpeechStartedCallback(func() {
			if callbacks.OnSpeakingStateChanged != nil {
				callbacks.OnSpeakingStateChanged(true)
			}
		}),
		speechtotext.WithSpeechEndedCallback(func() {
			if callbacks.OnSpeakingStateChanged != nil {
				callbacks.OnSpeakingStateChanged(false)
			}
		}),
		speechtotext.WithInterimTranscriptionCallback(func(transcript string) {
			if o.activePrompt != nil && !o.interruption {
				o.interruption = true
			}

			if callbacks.OnInterimTranscription != nil {
				callbacks.OnInterimTranscription(transcript)
			}
		}),
		speechtotext.WithTranscriptionCallback(func(transcript string) {
			if callbacks.OnInterimTranscription != nil {
				callbacks.OnInterimTranscription("")
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
					passthrough, err = o.respondToInterruption(transcript, interruption, callbacks)
					if err != nil {
						log.Printf("Failed to respond to interruption: %v", err)
					}
				}
				o.interruption = false
			}
			if passthrough != nil {
				if callbacks.OnTranscription != nil {
					callbacks.OnTranscription(transcript)
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
						if callbacks.OnResponse != nil {
							callbacks.OnResponse(data)
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
			if callbacks.OnResponseEnd != nil {
				callbacks.OnResponseEnd()
			}
		}
	}()
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

type SpeechToText interface {
	Transcribe(ctx context.Context, opts ...speechtotext.TranscriptionOption) error
	SendAudio(audio []byte) error
}

type TextToSpeech interface {
	OpenStream(ctx context.Context, opts ...texttospeech.TextToSpeechOption) error
	SendText(text string) error
	FlushBuffer() error
}
