package orchestration

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"log"

	"github.com/jinzhu/copier"
	"github.com/koscakluka/ema/core/audio"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/core/speechtotext"
	"github.com/koscakluka/ema/core/texttospeech"
)

type Orchestrator struct {
	IsRecording bool
	IsSpeaking  bool

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

	orchestrateOptions OrchestrateOptions
	config             *Config
}

func NewOrchestrator(opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		IsRecording: false,
		IsSpeaking:  false,
		transcripts: make(chan string, 10), // TODO: Figure out good valiues for this
		config:      &Config{AlwaysRecording: true},
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

func WithConfig(config *Config) OrchestratorOption {
	return func(o *Orchestrator) {
		if config == nil {
			return
		}

		o.config = config
	}
}

func (o *Orchestrator) Orchestrate(ctx context.Context, opts ...OrchestrateOption) {
	o.orchestrateOptions = OrchestrateOptions{}
	for _, opt := range opts {
		opt(&o.orchestrateOptions)
	}

	if o.textToSpeechClient != nil {
		ttsOptions := []texttospeech.TextToSpeechOption{
			texttospeech.WithAudioCallback(func(audio []byte) {
				if o.orchestrateOptions.onAudio != nil {
					o.orchestrateOptions.onAudio(audio)
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

				if o.orchestrateOptions.onAudioEnded != nil {
					o.orchestrateOptions.onAudioEnded(transcript)
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
				if o.orchestrateOptions.onSpeakingStateChanged != nil {
					o.orchestrateOptions.onSpeakingStateChanged(true)
				}
			}),
			speechtotext.WithSpeechEndedCallback(func() {
				if o.orchestrateOptions.onSpeakingStateChanged != nil {
					o.orchestrateOptions.onSpeakingStateChanged(false)
				}
			}),
			speechtotext.WithInterimTranscriptionCallback(func(transcript string) {
				if o.activePrompt != nil && !o.interruption {
					o.interruption = true
				}

				if o.orchestrateOptions.onInterimTranscription != nil {
					o.orchestrateOptions.onInterimTranscription(transcript)
				}
			}),
			speechtotext.WithTranscriptionCallback(func(transcript string) {
				if o.orchestrateOptions.onInterimTranscription != nil {
					o.orchestrateOptions.onInterimTranscription("")
				}

				o.SendPrompt(transcript)
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

			var response []llms.Message
			switch o.llm.(type) {
			case LLMWithStream:
				response, _ = o.processStreaming(ctx, transcript, messages)
			default:
				response = o.processPromptOld(ctx, transcript, messages)
			}

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
			if o.orchestrateOptions.onResponseEnd != nil {
				o.orchestrateOptions.onResponseEnd()
			}
		}
	}()

	if o.audioInput != nil && o.speechToTextClient != nil {
		go func() {
			if fineAudioInput, ok := o.audioInput.(AudioInputFine); ok {
				if o.config.AlwaysRecording {
					if err := fineAudioInput.StartCapture(ctx, func(audio []byte) {
						if err := o.SendAudio(audio); err != nil {
							log.Printf("Failed to send audio to speech to text client: %v", err)
						}
					}); err != nil {
						log.Printf("Failed to start audio input streaming: %v", err)
					}
				}
			} else {
				if err := o.audioInput.Stream(ctx, func(audio []byte) {
					if err := o.SendAudio(audio); err != nil {
						log.Printf("Failed to send audio to speech to text client: %v", err)
					}
				}); err != nil {
					log.Printf("Failed to start audio input streaming: %v", err)
				}
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

func (o *Orchestrator) SendPrompt(prompt string) {
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
				passthrough, err = o.respondToInterruption(prompt, interruption)
				if err != nil {
					log.Printf("Failed to respond to interruption: %v", err)
				}
			}
		}
		o.interruption = false
	}
	if passthrough != nil {
		if o.orchestrateOptions.onTranscription != nil {
			o.orchestrateOptions.onTranscription(prompt)
		}
		o.transcripts <- *passthrough
	}
}

func (o *Orchestrator) SendAudio(audio []byte) error {
	if o.speechToTextClient == nil {
		log.Println("Warning: SendAudio called but speech to text client is not set")
		return nil
	}

	if o.IsRecording || o.config.AlwaysRecording {
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

func (o *Orchestrator) IsAlwaysRecording() bool {
	return o.config.AlwaysRecording
}

func (o *Orchestrator) SetAlwaysRecording(isAlwaysRecording bool) {
	o.config.AlwaysRecording = isAlwaysRecording

	if isAlwaysRecording {
		if fineAudioInput, ok := o.audioInput.(AudioInputFine); ok {
			if err := fineAudioInput.StartCapture(
				context.TODO(),
				func(audio []byte) { o.SendAudio(audio) },
			); err != nil {
				log.Printf("Failed to start audio input: %v", err)
			}
		}
	} else if !o.IsRecording {
		if fineAudioInput, ok := o.audioInput.(AudioInputFine); ok {
			if err := fineAudioInput.StopCapture(); err != nil {
				log.Printf("Failed to stop audio input: %v", err)
			}
		}
	}
}

func (o *Orchestrator) StartRecording() error {
	o.IsRecording = true

	if o.config.AlwaysRecording {
		return nil
	}

	if fineAudioInput, ok := o.audioInput.(AudioInputFine); ok {
		if err := fineAudioInput.StartCapture(
			context.TODO(),
			func(audio []byte) { o.SendAudio(audio) },
		); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (o *Orchestrator) StopRecording() error {
	o.IsRecording = false
	if o.config.AlwaysRecording {
		return nil
	}

	if fineAudioInput, ok := o.audioInput.(AudioInputFine); ok {
		if err := fineAudioInput.StopCapture(); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (o *Orchestrator) Messages() []llms.Message {
	return slices.Clone(o.messages)
}

func (o *Orchestrator) processPromptOld(ctx context.Context, prompt string, messages []llms.Message) []llms.Message {
	response, _ := o.llm.Prompt(ctx, prompt,
		llms.WithMessages(messages...),
		llms.WithTools(o.tools...),
		llms.WithStream(func(data string) {
			if o.canceled {
				return
			}

			if o.orchestrateOptions.onResponse != nil {
				o.orchestrateOptions.onResponse(data)
			}
			if o.textToSpeechClient != nil {
				if err := o.textToSpeechClient.SendText(data); err != nil {
					log.Printf("Failed to send text to deepgram: %v", err)
				}
			}
		}))
	return response
}

func (o *Orchestrator) processStreaming(ctx context.Context, originalPrompt string, messages []llms.Message) ([]llms.Message, error) {
	if o.llm.(LLMWithStream) == nil {
		return nil, fmt.Errorf("LLM does not support streaming")
	}
	llm := o.llm.(LLMWithStream)
	var threadMessages []llms.Message
	if err := copier.Copy(&threadMessages, messages); err != nil {
		log.Printf("Failed to var copy messages: %v", err)
	}

	firstRun := true
	responses := []llms.Message{}
	for {
		var prompt *string
		if firstRun {
			prompt = &originalPrompt
			firstRun = false
		}
		stream := llm.PromptWithStream(context.TODO(), prompt,
			llms.WithMessages(threadMessages...),
			llms.WithTools(o.tools...),
		)

		var response strings.Builder
		toolCalls := []llms.ToolCall{}
		for chunk, err := range stream.Chunks {
			if err != nil {
				// TODO: handle error
				break
			}

			if o.canceled {
				return nil, nil
			}

			switch chunk.(type) {
			// case llms.StreamRoleChunk:
			// case llms.StreamReasoningChunk:
			// case llms.StreamUsageChunk:
			// 	chunk := chunk.(llms.StreamUsageChunk)
			case llms.StreamContentChunk:
				chunk := chunk.(llms.StreamContentChunk)

				response.WriteString(chunk.Content())

				if o.orchestrateOptions.onResponse != nil {
					o.orchestrateOptions.onResponse(chunk.Content())
				}
				if o.textToSpeechClient != nil {
					if err := o.textToSpeechClient.SendText(chunk.Content()); err != nil {
						log.Printf("Failed to send text to deepgram: %v", err)
					}
				}

			case llms.StreamToolCallChunk:
				toolCalls = append(toolCalls, chunk.(llms.StreamToolCallChunk).ToolCall())
			}
		}

		responses = append(responses, llms.Message{
			Role:      llms.MessageRoleAssistant,
			ToolCalls: toolCalls,
		})
		threadMessages = append(threadMessages, llms.Message{
			Role:      llms.MessageRoleAssistant,
			ToolCalls: toolCalls,
		})
		for _, toolCall := range toolCalls {
			response, _ := o.callTool(ctx, toolCall)
			if response != nil {
				threadMessages = append(threadMessages, *response)
				responses = append(responses, *response)
			}
		}

		if len(toolCalls) == 0 {
			responses = append(responses, llms.Message{
				Role:    llms.MessageRoleAssistant,
				Content: response.String(),
			})
			return responses, nil
		}
	}
}

func (o *Orchestrator) callTool(_ context.Context, toolCall llms.ToolCall) (*llms.Message, error) {
	for _, tool := range o.tools {
		if tool.Function.Name == toolCall.Function.Name {
			resp, err := tool.Execute(toolCall.Function.Arguments)
			if err != nil {
				log.Println("Error executing tool:", err)
			}
			return &llms.Message{
				ToolCallID: toolCall.ID,
				Role:       llms.MessageRoleTool,
				Content:    resp,
			}, nil
		}
	}

	return nil, fmt.Errorf("tool not found")
}

type LLM interface {
	Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error)
}

type LLMWithStream interface {
	PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream
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

type AudioInputFine interface {
	StartCapture(ctx context.Context, onAudio func(audio []byte)) error
	StopCapture() error
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
