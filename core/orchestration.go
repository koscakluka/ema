package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"log"

	emaContext "github.com/koscakluka/ema/core/context"
	"github.com/koscakluka/ema/core/llms"
)

type Orchestrator struct {
	IsRecording bool
	IsSpeaking  bool

	turns Turns

	outputTextBuffer  textBuffer
	outputAudioBuffer audioBuffer
	transcripts       chan string
	promptEnded       sync.WaitGroup

	tools []llms.Tool

	llm                    LLM
	speechToTextClient     SpeechToText
	textToSpeechClient     TextToSpeech
	audioInput             AudioInput
	audioOutput            audioOutput
	interruptionClassifier InterruptionClassifier
	interruptionHandlerV0  InterruptionHandlerV0
	interruptionHandlerV1  InterruptionHandlerV1

	orchestrateOptions OrchestrateOptions
	config             *Config
}

func NewOrchestrator(opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		IsRecording:       false,
		IsSpeaking:        false,
		transcripts:       make(chan string, 10), // TODO: Figure out good valiues for this
		config:            &Config{AlwaysRecording: true},
		turns:             Turns{activeTurnIdx: -1},
		outputTextBuffer:  *newTextBuffer(),
		outputAudioBuffer: *newAudioBuffer(),
	}

	for _, opt := range opts {
		opt(o)
	}

	// TODO: Remove this in a couple of releases
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

func (o *Orchestrator) Close() {
	// TODO: Make sure that deepgramClient is closed and no longer transcribing
	// before closing the channel
	close(o.transcripts)
}

func (o *Orchestrator) Orchestrate(ctx context.Context, opts ...OrchestrateOption) {
	o.orchestrateOptions = OrchestrateOptions{}
	for _, opt := range opts {
		opt(&o.orchestrateOptions)
	}

	o.initTTS()
	o.initSST()

	go func() {
		for transcript := range o.transcripts {
			if o.turns.activeTurn() != nil {
				o.promptEnded.Wait()
			}
			activeTurn := &llms.Turn{
				Role:  llms.TurnRoleAssistant,
				Stage: llms.TurnStagePreparing,
			}
			o.promptEnded.Add(1)

			messages := o.turns
			o.turns.Push(llms.Turn{
				Role:    llms.TurnRoleUser,
				Content: transcript,
			})

			o.outputTextBuffer.Clear()
			o.outputAudioBuffer.Clear()
			go o.passTextToTTS()
			go o.passSpeechToAudioOutput()

			activeTurn.Stage = llms.TurnStageGeneratingResponse
			o.turns.pushActiveTurn(*activeTurn)
			var response *llms.Turn
			switch o.llm.(type) {
			case LLMWithStream:
				response, _ = o.processStreaming(ctx, transcript, messages.turns, &o.outputTextBuffer)
			case LLMWithPrompt:
				response, _ = o.processPromptOld(ctx, transcript, messages.turns, &o.outputTextBuffer)
			default:
				// Impossible state
				continue
			}

			o.outputTextBuffer.ChunksDone()
			o.outputAudioBuffer.ChunksDone()
			activeTurn = o.turns.activeTurn()
			if activeTurn != nil && response != nil {
				activeTurn.Role = response.Role
				activeTurn.Content = response.Content
				activeTurn.ToolCalls = response.ToolCalls
			} else {
				// TODO: Figure out how to handle this case
			}

			if activeTurn != nil && !activeTurn.Cancelled {
				// NOTE: Just in case it wasn't set previously
				activeTurn.Stage = llms.TurnStageSpeaking
				o.turns.updateActiveTurn(*activeTurn)
			}
		}
	}()

	o.initAudioInput()
}

func (o *Orchestrator) SendPrompt(prompt string) {
	o.processUserTurn(prompt)
}

func (o *Orchestrator) SendAudio(audio []byte) error {
	return o.sendAudio(audio)
}

// QueuePrompt immediately queues the prompt for processing after the current
// turn is finished. It bypasses the normal processing pipeline and can be useful
// for handling prompts that are sure to follow up after the current turn.
func (o *Orchestrator) QueuePrompt(prompt string) {
	go o.queuePrompt(prompt)
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
		go func() {
			if err := o.startCapture(); err != nil {
				log.Printf("Failed to start audio input: %v", err)
			}
		}()
	} else if !o.IsRecording {
		if err := o.stopCapture(); err != nil {
			log.Printf("Failed to stop audio input: %v", err)
		}
	}
}

func (o *Orchestrator) StartRecording() error {
	o.IsRecording = true

	if o.config.AlwaysRecording {
		return nil
	}

	return o.startCapture()
}

func (o *Orchestrator) StopRecording() error {
	o.IsRecording = false
	if o.config.AlwaysRecording {
		return nil
	}

	return o.stopCapture()
}

func (o *Orchestrator) Turns() emaContext.TurnsV0 {
	return &o.turns
}

func (o *Orchestrator) processPromptOld(ctx context.Context, prompt string, messages []llms.Turn, buffer *textBuffer) (*llms.Turn, error) {
	if o.llm.(LLMWithPrompt) == nil {
		return nil, fmt.Errorf("LLM does not support prompting")
	}

	response, _ := o.llm.(LLMWithPrompt).Prompt(ctx, prompt,
		llms.WithTurns(messages...),
		llms.WithTools(o.tools...),
		llms.WithStream(buffer.AddChunk),
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

func (o *Orchestrator) processStreaming(ctx context.Context, originalPrompt string, originalTurns []llms.Turn, buffer *textBuffer) (*llms.Turn, error) {
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

			activeTurn := o.turns.activeTurn()
			if activeTurn != nil && activeTurn.Cancelled {
				return nil, nil
			}
			if activeTurn != nil && activeTurn.Stage != llms.TurnStageSpeaking {
				activeTurn.Stage = llms.TurnStageSpeaking
				o.turns.updateActiveTurn(*activeTurn)
			}

			switch chunk.(type) {
			// case llms.StreamRoleChunk:
			// case llms.StreamReasoningChunk:
			// case llms.StreamUsageChunk:
			// 	chunk := chunk.(llms.StreamUsageChunk)
			case llms.StreamContentChunk:
				chunk := chunk.(llms.StreamContentChunk)

				response.WriteString(chunk.Content())
				buffer.AddChunk(chunk.Content())

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
		_, err := o.processStreaming(ctx, prompt, o.turns.turns, newTextBuffer())
		return err

	case LLMWithPrompt:
		_, err := o.processPromptOld(ctx, prompt, o.turns.turns, newTextBuffer())
		return err

	default:
		// Impossible state technically
		return fmt.Errorf("unknown LLM type")
	}

}
