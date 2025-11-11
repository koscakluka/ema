package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/koscakluka/ema/core/llms"
)

const (
	interruptionClassifierSystemPrompt = `You are a helpful assistant that can classify a prompt type of interruption to the conversation.

A conversation interruption can be classified as one of the following:
- continuation: The interruption is a continuation of the previous sentence/request (e.g. "Tell me about Star Wars.", "Ships design").
- cancellation: Anything that indicates that the response should not be finished. Only used if the interruption cannot be addressed by a listed tool.
- clarification: The interruption is a clarification or restatement of the previous instruction (e.g. "It's actually about the TV show, not the movie").
- ignorable: The interruption is ignorable and should not be responded to.
- repetition: The interruption is a repetition of the previous sentence/request.
- noise: The interruption is noise and should be ignored.
- action: The interruption is a addressable with a listed tool.
- new prompt: The interruption is a new prompt to be responded to that could not be understood as a continuation of the previous sentence

Only respond with the classification of the interruption as JSON: {"classification": "response"}

Accessible tools:
`

	interruptionClassifierStructuredSystemPrompt = `You are a helpful assistant that can classify a prompt type of interruption to the conversation.

A conversation interruption can be classified as one of the following:
- continuation: The interruption is a continuation of the previous sentence/request (e.g. "Tell me about Star Wars.", "Ships design").
- cancellation: Anything that indicates that the response should not be finished. Only used if the interruption cannot be addressed by a listed tool.
- clarification: The interruption is a clarification or restatement of the previous instruction (e.g. "It's actually about the TV show, not the movie").
- ignorable: The interruption is ignorable and should not be responded to.
- repetition: The interruption is a repetition of the previous sentence/request.
- noise: The interruption is noise and should be ignored.
- action: The interruption is a addressable with a listed tool.
- new prompt: The interruption is a new prompt to be responded to that could not be understood as a continuation of the previous sentence

Accessible tools:
`
)

type Classification struct {
	Type string `json:"type" jsonschema:"title=Type,description=The type of interruption" enum:"continuation,clarification,cancellation,ignorable,repetition,noise,action,new prompt"`
}

func classify(interruption llms.InterruptionV0, llm LLM, opts ...ClassifyOption) (*llms.InterruptionV0, error) {
	options := ClassifyOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	switch llm.(type) {
	case LLMWithStructuredPrompt:
		systemPrompt := interruptionClassifierStructuredSystemPrompt
		for _, tool := range options.Tools {
			systemPrompt += fmt.Sprintf("- %s: %s", tool.Function.Name, tool.Function.Description)
		}

		resp := Classification{}
		if err := llm.(LLMWithStructuredPrompt).PromptWithStructure(context.TODO(), interruption.Source,
			&resp,
			llms.WithSystemPrompt(systemPrompt),
			llms.WithTurns(options.History...),
		); err != nil {
			// TODO: Retry?
			return &interruption, err
		}

		interruptionType, err := toInterruptionType(resp.Type)
		if err != nil {
			return nil, err
		}
		interruption.Type = string(interruptionType)
		return &interruption, nil

	case LLMWithGeneralPrompt:
		systemPrompt := interruptionClassifierSystemPrompt
		for _, tool := range options.Tools {
			systemPrompt += fmt.Sprintf("- %s: %s", tool.Function.Name, tool.Function.Description)
		}

		response, _ := llm.(LLMWithGeneralPrompt).Prompt(context.TODO(), interruption.Source,
			llms.WithSystemPrompt(systemPrompt),
			llms.WithTurns(options.History...),
		)

		if len(response.Content) == 0 {
			return nil, fmt.Errorf("no response from interruption classifier")
		}

		var unmarshalledResponse struct {
			Classification string `json:"classification"`
		}
		if err := json.Unmarshal([]byte(response.Content), &unmarshalledResponse); err != nil {
			// TODO: Retry
			return nil, fmt.Errorf("failed to unmarshal interruption classification response: %w", err)
		}

		interruptionType, err := toInterruptionType(unmarshalledResponse.Classification)
		if err != nil {
			return nil, err
		}
		interruption.Type = string(interruptionType)
		return &interruption, nil
	}

	return nil, fmt.Errorf("unknown llm type")
}

func toInterruptionType(classification string) (interruptionType, error) {
	switch classification {
	case "continuation":
		return InterruptionTypeContinuation, nil
	case "clarification":
		return InterruptionTypeClarification, nil
	case "cancellation":
		return InterruptionTypeCancellation, nil
	case "ignorable":
		return InterruptionTypeIgnorable, nil
	case "repetition":
		return InterruptionTypeRepetition, nil
	case "noise":
		return InterruptionTypeNoise, nil
	case "action":
		return InterruptionTypeAction, nil
	case "new prompt":
		return InterruptionTypeNewPrompt, nil
	default:
		return "", fmt.Errorf("unknown interruption type: %s", classification)
	}
}

type ClassifyOption func(*ClassifyOptions)

type ClassifyOptions struct {
	History []llms.Turn
	Tools   []llms.Tool
}

func WithTools(tools []llms.Tool) ClassifyOption {
	return func(o *ClassifyOptions) {
		o.Tools = tools
	}
}

func WithHistory(history []llms.Turn) ClassifyOption {
	return func(o *ClassifyOptions) {
		o.History = history
	}
}
