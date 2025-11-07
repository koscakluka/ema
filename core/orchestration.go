package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"log"

	"github.com/koscakluka/ema/core/audio"
	emaContext "github.com/koscakluka/ema/core/context"
	"github.com/koscakluka/ema/core/interruptions"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/core/speechtotext"
	"github.com/koscakluka/ema/core/texttospeech"
)

type Orchestrator struct {
	IsRecording bool
	IsSpeaking  bool

	turns Turns

	transcripts  chan string
	activeTurn   *llms.Turn
	promptEnded  sync.WaitGroup
	interruption bool

	tools []llms.Tool

	llm                    LLM
	speechToTextClient     SpeechToText
	textToSpeechClient     TextToSpeech
	audioInput             AudioInput
	audioOutput            AudioOutput
	interruptionClassifier InterruptionClassifier
	interruptionHandler    InterruptionHandlerV0

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
		switch o.llm.(type) {
		case LLMWithPrompt:
			o.interruptionClassifier = NewSimpleInterruptionClassifier(o.llm.(LLMWithPrompt))
		case InterruptionLLM:
			// HACK: To avoid changing the signature of
			// NewSimpleInterruptionClassifier we pass nil for LLM right now,
			// when we change the whole classifier concept we can change the
			// signature
			o.interruptionClassifier = NewSimpleInterruptionClassifier(nil, ClassifierWithInterruptionLLM(o.llm.(InterruptionLLM)))
		case LLMWithGeneralPrompt:
			// HACK: To avoid changing the signature of
			// NewSimpleInterruptionClassifier we pass nil for LLM right now,
			// when we change the whole classifier concept we can change the
			// signature
			o.interruptionClassifier = NewSimpleInterruptionClassifier(nil, ClassifierWithGeneralPromptLLM(o.llm.(LLMWithGeneralPrompt)))
		}
	}

	return o
}

type OrchestratorOption func(*Orchestrator)

// WithLLM sets the LLM client for the orchestrator.
//
// Deprecated: use WithStreamingLLM instead
func WithLLM(client LLMWithPrompt) OrchestratorOption {
	return func(o *Orchestrator) {
		o.llm = client
	}
}

func WithStreamingLLM(client LLMWithStream) OrchestratorOption {
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

// WithInterruptionClassifier sets the interruption classifier that is used
// internally to classify interruptions types so orchestrator can respond to
// them.
//
// Deprecated: use WithInterruptionHandler instead
func WithInterruptionClassifier(classifier InterruptionClassifier) OrchestratorOption {
	return func(o *Orchestrator) {
		o.interruptionClassifier = classifier
	}
}

func WithInterruptionHandlerV0(handler InterruptionHandlerV0) OrchestratorOption {
	return func(o *Orchestrator) {
		o.interruptionHandler = handler
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

				if !o.IsSpeaking || (o.activeTurn != nil && o.activeTurn.Cancelled) {
					o.audioOutput.ClearBuffer()
					return
				}

				o.audioOutput.SendAudio(audio)
			}),
			texttospeech.WithAudioEndedCallback(func(transcript string) {
				defer func() {
					o.activeTurn = nil
					o.promptEnded.Done()
				}()

				if o.orchestrateOptions.onAudioEnded != nil {
					o.orchestrateOptions.onAudioEnded(transcript)
				}

				if o.audioOutput == nil {
					return
				}

				if !o.IsSpeaking || (o.activeTurn != nil && o.activeTurn.Cancelled) {
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
				if o.activeTurn != nil && !o.interruption {
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
			if o.activeTurn != nil {
				o.promptEnded.Wait()
			}
			o.activeTurn = &llms.Turn{
				Role: llms.TurnRoleAssistant,
			}
			o.promptEnded.Add(1)

			messages := o.turns
			o.turns = append(o.turns, llms.Turn{
				Role:    llms.TurnRoleUser,
				Content: transcript,
			})

			var response *llms.Turn
			switch o.llm.(type) {
			case LLMWithStream:
				response, _ = o.processStreaming(ctx, transcript, messages)
			case LLMWithPrompt:
				response, _ = o.processPromptOld(ctx, transcript, messages)
			default:
				// Impossible state
				continue
			}

			if response != nil {
				o.activeTurn = response
			} else {
				// TODO: Figure out how to handle this case
			}

			o.turns = append(o.turns, *o.activeTurn)
			if o.textToSpeechClient != nil {
				if err := o.textToSpeechClient.FlushBuffer(); err != nil {
					log.Printf("Failed to flush buffer: %v", err)
				}
			} else {
				o.activeTurn = nil
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
	if o.activeTurn != nil && !o.interruption {
		o.interruption = true
	}

	passthrough := &prompt
	if o.interruption {
		if o.interruptionHandler != nil {
			if err := o.interruptionHandler.HandleV0(prompt, o.turns, o.tools, o); err != nil {
				log.Printf("Failed to handle interruption: %v", err)
			} else {
				o.interruption = false
				return
			}
		} else if o.interruptionClassifier != nil {
			interruption, err := o.interruptionClassifier.Classify(prompt, llms.ToMessages(o.turns), ClassifyWithTools(o.tools))
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
		o.queuePrompt(*passthrough)
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

// QueuePrompt immediately queues the prompt for processing after the current
// turn is finished. It bypasses the normal processing pipeline and can be useful
// for handling prompts that are sure to follow up after the current turn.
func (o *Orchestrator) QueuePrompt(prompt string) {
	go o.queuePrompt(prompt)
}

func (o *Orchestrator) queuePrompt(prompt string) {
	if o.orchestrateOptions.onTranscription != nil {
		o.orchestrateOptions.onTranscription(prompt)
	}
	o.transcripts <- prompt
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
	return llms.ToMessages(o.turns)
}

func (o *Orchestrator) Turns() emaContext.TurnsV0 {
	return &o.turns
}

func (o *Orchestrator) processPromptOld(ctx context.Context, prompt string, messages []llms.Turn) (*llms.Turn, error) {
	if o.llm.(LLMWithPrompt) == nil {
		return nil, fmt.Errorf("LLM does not support prompting")
	}

	response, _ := o.llm.(LLMWithPrompt).Prompt(ctx, prompt,
		llms.WithTurns(messages...),
		llms.WithTools(o.tools...),
		llms.WithStream(func(data string) {
			if o.activeTurn != nil && o.activeTurn.Cancelled {
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
		}),
	)

	turns := llms.ToTurns(response)
	if len(turns) == 0 {
		log.Println("Warning: no turns returned for assistants turn")
		return nil, nil
	} else if len(turns) > 1 {
		log.Println("Warning: multiple turns returned for assistants turn")
	}
	return &turns[0], nil
}

func (o *Orchestrator) processStreaming(ctx context.Context, originalPrompt string, originalTurns []llms.Turn) (*llms.Turn, error) {
	if o.llm.(LLMWithStream) == nil {
		return nil, fmt.Errorf("LLM does not support streaming")
	}
	llm := o.llm.(LLMWithStream)

	firstRun := true
	assistantTurn := llms.Turn{Role: llms.TurnRoleAssistant}
	for {
		var prompt *string
		turns := originalTurns
		if firstRun {
			prompt = &originalPrompt
			firstRun = false
		} else {
			turns = append(turns, assistantTurn)
		}

		stream := llm.PromptWithStream(context.TODO(), prompt,
			llms.WithTurns(turns...),
			llms.WithTools(o.tools...),
		)

		var response strings.Builder
		toolCalls := []llms.ToolCall{}
		for chunk, err := range stream.Chunks {
			if err != nil {
				// TODO: handle error
				break
			}

			if o.activeTurn != nil && o.activeTurn.Cancelled {
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

		for _, toolCall := range toolCalls {
			response, _ := o.CallTool(ctx, toolCall)
			if response != nil {
				toolCall.Response = response.Content
			}
			assistantTurn.ToolCalls = append(assistantTurn.ToolCalls, toolCall)
		}

		if len(toolCalls) == 0 {
			assistantTurn.Content = response.String()
			return &assistantTurn, nil
		}
	}
}

func (o *Orchestrator) CallTool(_ context.Context, toolCall llms.ToolCall) (*llms.Turn, error) {
	toolName := toolCall.Name
	toolArguments := toolCall.Arguments
	if toolCall.Name == "" {
		toolName = toolCall.Function.Name
	}
	if toolCall.Arguments == "" {
		toolArguments = toolCall.Function.Arguments
	}
	for _, tool := range o.tools {
		if tool.Function.Name == toolName {
			resp, err := tool.Execute(toolArguments)
			if err != nil {
				log.Println("Error executing tool:", err)
			}
			return &llms.Turn{
				ToolCallID: toolCall.ID,
				Role:       llms.TurnRoleAssistant,
				Content:    resp,
			}, nil
		}
	}

	return nil, fmt.Errorf("tool not found")
}

func (o *Orchestrator) CallToolWithPrompt(ctx context.Context, prompt string) error {
	switch o.llm.(type) {
	case LLMWithStream:
		_, err := o.processStreaming(ctx, prompt, o.turns)
		return err

	case LLMWithPrompt:
		_, err := o.processPromptOld(ctx, prompt, o.turns)
		return err

	default:
		// Impossible state technically
		return fmt.Errorf("unknown LLM type")
	}
}

type LLM any

// Deprecated: use LLMWithGeneralPrompt instead
type LLMWithPrompt interface {
	LLM
	Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error)
}

type LLMWithGeneralPrompt interface {
	LLM
	Prompt(ctx context.Context, prompt string, opts ...llms.GeneralPromptOption) (*llms.Message, error)
}

type LLMWithStream interface {
	LLM
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

type InterruptionHandlerV0 interface {
	HandleV0(prompt string, turns []llms.Turn, tools []llms.Tool, orchestrator interruptions.OrchestratorV0) error
}
