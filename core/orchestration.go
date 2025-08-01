package orchestration

import (
	"bytes"
	"context"
	"encoding/binary"

	"log"

	"github.com/koscakluka/ema/core/llms/groq"
	"github.com/koscakluka/ema/core/speechtotext"
	deepgrams2t "github.com/koscakluka/ema/core/speechtotext/deepgram"
	"github.com/koscakluka/ema/core/texttospeech"
	deepgramt2s "github.com/koscakluka/ema/core/texttospeech/deepgram"

	"github.com/gordonklaus/portaudio"
)

const bufferSize = 128

type Orchestrator struct {
	AlwaysRecording bool
	IsRecording     bool
	IsSpeaking      bool
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		AlwaysRecording: true,
		IsRecording:     false,
		IsSpeaking:      true,
	}
}

type Callbacks struct {
	OnTranscription        func(transcript string)
	OnInterimTranscription func(transcript string)
	OnSpeakingStateChanged func(isSpeaking bool)
	OnResponse             func(response string)
}

func (o *Orchestrator) ListenForSpeech(ctx context.Context, callbacks Callbacks) {
	err := portaudio.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize PortAudio: %v", err)
	}
	defer portaudio.Terminate()

	client := groq.NewClient()

	in := make([]int16, bufferSize)
	out := make([]int16, bufferSize)
	stream, err := portaudio.OpenDefaultStream(1, 1, 48000, bufferSize, in, out)
	if err != nil {
		log.Fatalf("Failed to open PortAudio stream: %v", err)
	}
	defer stream.Close()

	voice := deepgramt2s.VoiceAuraAsteria
	deepgramSpeechClient, err := deepgramt2s.NewTextToSpeechClient(context.TODO(), voice)
	if err != nil {
		log.Printf("Failed to create deepgram speech client: %v", err)
	}

	leftoverAudio := make([]byte, bufferSize*2)

	if err := deepgramSpeechClient.OpenStream(context.TODO(),
		texttospeech.WithAudioCallback(func(audio []byte) {
			bufferSize := bufferSize * 2

			if !o.IsSpeaking {
				return
			}

			// PERF: This is just to test this, there is no reason we should
			// kill performance by copying here
			audio = append(leftoverAudio, audio...)
			for i := range len(audio)/bufferSize + 1 {
				if (i+1)*bufferSize > len(audio) {
					leftoverAudio = make([]byte, len(audio)-i*bufferSize)
					copy(leftoverAudio, audio[i*bufferSize:])
					break
				}

				binary.Read(bytes.NewBuffer(audio[i*bufferSize:(i+1)*bufferSize]), binary.LittleEndian, out)
				stream.Write()
			}
		}),
		texttospeech.WithAudioEndedCallback(func(transcript string) {
			if !o.IsSpeaking {
				leftoverAudio = make([]byte, 0)
				return
			}

			bufferSize := bufferSize * 2
			lastAudio := make([]byte, bufferSize)
			for i := range bufferSize {
				lastAudio[i] = 0
			}
			copy(lastAudio, leftoverAudio)
			binary.Write(bytes.NewBuffer(lastAudio), binary.LittleEndian, out)
			stream.Write()
			return
		}),
	); err != nil {
		log.Printf("Failed to open deepgram speech stream: %v", err)
	}

	deepgramClient := deepgrams2t.NewClient(context.TODO())
	if err = deepgramClient.Transcribe(context.TODO(),
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
			if callbacks.OnInterimTranscription != nil {
				callbacks.OnInterimTranscription(transcript)
			}
		}),
		speechtotext.WithTranscriptionCallback(func(transcript string) {
			if callbacks.OnInterimTranscription != nil {
				callbacks.OnInterimTranscription("")
			}
			if callbacks.OnTranscription != nil {
				callbacks.OnTranscription(transcript + "\n")
			}
			flushedOnFinal := false
			client.Prompt(context.TODO(), transcript,
				groq.WithTools(
					groq.NewTool("recording_control", "Turn on or off sound recording, might be referred to as 'listening'",
						map[string]groq.ParameterBase{
							"is_recording": {Type: "boolean", Description: "Whether to record or not"},
						},
						func(parameters struct {
							IsRecording bool `json:"is_recording"`
						}) (string, error) {
							o.AlwaysRecording = parameters.IsRecording
							return "Success", nil
						}),
					groq.NewTool("speaking_control", "Turn off agent's speaking ability. Might be referred to as 'muting'",
						map[string]groq.ParameterBase{
							"is_speaking": {Type: "boolean", Description: "Wheather to speak or not"},
						},
						func(parameters struct {
							IsSpeaking bool `json:"is_speaking"`
						}) (string, error) {
							o.IsSpeaking = parameters.IsSpeaking
							return "Success", nil
						}),
				),
				groq.WithStream(
					func(data string) {
						flushedOnFinal = false
						if callbacks.OnResponse != nil {
							callbacks.OnResponse(data)
						}
						if err := deepgramSpeechClient.SendText(data); err != nil {
							log.Printf("Failed to send text to deepgram: %v", err)
						}
					}))
			if !flushedOnFinal {
				if err := deepgramSpeechClient.FlushBuffer(); err != nil {
					log.Printf("Failed to flush buffer: %v", err)
				}
			}
			if callbacks.OnResponse != nil {
				callbacks.OnResponse("\n")
			}
		}),
	); err != nil {
		log.Fatalf("Failed to start transcribing: %v", err)
	}
	defer deepgramClient.Close()

	log.Println("Starting microphone capture. Speak now...")
	if err := stream.Start(); err != nil {
		log.Fatalf("Failed to start PortAudio stream: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := stream.Read(); err != nil {
				log.Printf("Failed to read from PortAudio stream: %v", err)
			}
			if o.IsRecording || o.AlwaysRecording {
				audioBuffer := bytes.Buffer{}
				binary.Write(&audioBuffer, binary.LittleEndian, in)
				if err := deepgramClient.SendAudio(audioBuffer.Bytes()); err != nil {
					log.Fatalf("Failed to send audio: %v", err)
				}
			}
		}
	}

}
