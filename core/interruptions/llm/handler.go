package llm

import (
	"context"

	"github.com/koscakluka/ema/core/interruptions"
	"github.com/koscakluka/ema/core/llms"
)

type InterruptionHandlerWithStructuredPrompt struct {
	llm LLMWithStructuredPrompt
}

func NewInterruptionHandlerWithStructuredPrompt(classificationLLM LLMWithStructuredPrompt) *InterruptionHandlerWithStructuredPrompt {
	handler := &InterruptionHandlerWithStructuredPrompt{
		llm: classificationLLM,
	}
	return handler
}

func (h *InterruptionHandlerWithStructuredPrompt) HandleV0(prompt string, history []llms.Turn, tools []llms.Tool, orchestrator interruptions.OrchestratorV0) error {
	classification, err := classify(prompt, h.llm, WithHistory(history), WithTools(tools))
	if err != nil {
		return err
	}
	return respond(classification, prompt, orchestrator)
}

type LLMWithStructuredPrompt interface {
	PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error
}

type InterruptionHandlerWithGeneralPrompt struct {
	LLM
	llm LLMWithGeneralPrompt
}

func NewInterruptionHandlerWithGeneralPrompt(classificationLLM LLMWithGeneralPrompt) *InterruptionHandlerWithGeneralPrompt {
	handler := &InterruptionHandlerWithGeneralPrompt{
		llm: classificationLLM,
	}
	return handler
}

type LLMWithGeneralPrompt interface {
	LLM
	Prompt(ctx context.Context, prompt string, opts ...llms.GeneralPromptOption) (*llms.Message, error)
}

func (h *InterruptionHandlerWithGeneralPrompt) HandleV0(prompt string, history []llms.Turn, tools []llms.Tool, orchestrator interruptions.OrchestratorV0) error {
	classification, err := classify(prompt, h.llm)
	if err != nil {
		return err
	}
	return respond(classification, prompt, nil)
}

type LLM any
