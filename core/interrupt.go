package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/koscakluka/ema/core/llms"
)

func (o *Orchestrator) respondToInterruption(prompt string, t interruptionType) (passthrough *string, err error) {
	// TODO: Check if this is still relevant (do we still have an active prompt)
	switch t {
	case InterruptionTypeContinuation:
		o.Cancel()
		lastPrompt := -1
		for i := range o.messages {
			if o.messages[i].Role == llms.MessageRoleUser {
				lastPrompt = i
				break
			}
		}
		if lastPrompt == -1 {
			return &prompt, nil
		}
		prompt = o.messages[lastPrompt].Content + " " + prompt
		o.messages = slices.Delete(o.messages, lastPrompt, len(o.messages))
		return &prompt, nil
	case InterruptionTypeClarification:
		o.Cancel()
		return &prompt, nil
		// TODO: Properly passthrough the modified prompt
	case InterruptionTypeCancellation:
		o.Cancel()
		return nil, nil
	case InterruptionTypeIgnorable,
		InterruptionTypeRepetition,
		InterruptionTypeNoise:
		return nil, nil
	case InterruptionTypeAction:
		if _, err := o.llm.Prompt(context.TODO(), prompt,
			llms.WithForcedTools(o.tools...),
			llms.WithMessages(o.messages...),
		); err != nil {
			// TODO: Retry?
			return nil, fmt.Errorf("failed to call tool LLM: %w", err)
		}
		return nil, nil
	case InterruptionTypeNewPrompt:
		// TODO: Consider interrupting the current prompt and asking to continue
		// with it before addressing the new prompt
		return &prompt, nil
	default:
		return &prompt, fmt.Errorf("unknown interruption type: %s", t)
	}
}

type SimpleInterruptionClassifier struct {
	llm   LLM
	tools []llms.Tool
}

func NewSimpleInterruptionClassifier(llm LLM, opts ...InterruptionClassifierOption) *SimpleInterruptionClassifier {
	classifier := &SimpleInterruptionClassifier{
		llm: llm,
	}
	for _, opt := range opts {
		opt(classifier)
	}
	return classifier
}

type InterruptionClassifierOption func(*SimpleInterruptionClassifier)

func ClassifierWithTools(tools []llms.Tool) InterruptionClassifierOption {
	return func(c *SimpleInterruptionClassifier) {
		c.tools = tools
	}
}

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

func (c SimpleInterruptionClassifier) Classify(prompt string, history []llms.Message, opts ...ClassifyOption) (interruptionType, error) {
	options := ClassifyOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	classification := ""
	switch c.llm.(type) {
	case InterruptionLLM:
		systemPrompt := interruptionClassifierStructuredSystemPrompt
		for _, tool := range append(c.tools, options.Tools...) {
			systemPrompt += fmt.Sprintf("- %s: %s", tool.Function.Name, tool.Function.Description)
		}

		resp := Classification{}
		llm := c.llm.(InterruptionLLM)
		if err := llm.PromptWithStructure(context.TODO(), prompt,
			&resp,
			llms.WithSystemPrompt(systemPrompt),
			llms.WithMessages(history...),
		); err != nil {
			return "", err
		}

		classification = resp.Type

	default:
		systemPrompt := interruptionClassifierSystemPrompt
		for _, tool := range append(c.tools, options.Tools...) {
			systemPrompt += fmt.Sprintf("- %s: %s", tool.Function.Name, tool.Function.Description)
		}

		response, _ := c.llm.Prompt(context.TODO(), prompt,
			llms.WithSystemPrompt(systemPrompt),
			llms.WithMessages(history...),
		)

		if len(response) == 0 || len(response[0].Content) == 0 {
			return "", fmt.Errorf("no response from interruption classifier")
		}

		var unmarshalledResponse struct {
			Classification string `json:"classification"`
		}
		if err := json.Unmarshal([]byte(response[len(response)-1].Content), &unmarshalledResponse); err != nil {
			// TODO: Retry
			log.Printf("Failed to unmarshal interruption classification response: %v", err)
			return "", nil
		}
		classification = unmarshalledResponse.Classification
	}

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
	Tools []llms.Tool
}

func ClassifyWithTools(tools []llms.Tool) ClassifyOption {
	return func(o *ClassifyOptions) {
		o.Tools = tools
	}
}

type interruptionType string

const (
	InterruptionTypeContinuation  interruptionType = "continuation"
	InterruptionTypeClarification interruptionType = "clarification"
	InterruptionTypeCancellation  interruptionType = "cancellation"
	InterruptionTypeIgnorable     interruptionType = "ignorable"
	InterruptionTypeRepetition    interruptionType = "repetition"
	InterruptionTypeNoise         interruptionType = "noise"
	InterruptionTypeAction        interruptionType = "action"
	InterruptionTypeNewPrompt     interruptionType = "new prompt"
)

type InterruptionLLM interface {
	PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error
}
