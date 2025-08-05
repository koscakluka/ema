package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/core/llms/groq"
)

func (o *Orchestrator) respondToInterruption(prompt string, t interruptionType, callbacks Callbacks) (passthrough *string, err error) {
	// TODO: Check if this is still relevant (do we still have an active prompt)
	switch t {
	case InterruptionTypeContinuation:
		o.canceled = true
		if callbacks.OnCancellation != nil {
			callbacks.OnCancellation()
		}
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
		o.canceled = true
		if callbacks.OnCancellation != nil {
			callbacks.OnCancellation()
		}
		return &prompt, nil
		// TODO: Properly passthrough the modified prompt
	case InterruptionTypeCancellation:
		o.canceled = true
		if callbacks.OnCancellation != nil {
			callbacks.OnCancellation()
		}
		return nil, nil
	case InterruptionTypeIgnorable,
		InterruptionTypeRepetition,
		InterruptionTypeNoise:
		return nil, nil
	case InterruptionTypeAction:
		client := groq.NewClient()
		if _, err := client.Prompt(context.TODO(), prompt,
			groq.WithForcedTools(o.tools...),
			groq.WithMessages(o.messages...),
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
)

func (o *Orchestrator) classifyInterruption(prompt string) (interruptionType, error) {
	client := groq.NewClient()

	systemPrompt := interruptionClassifierSystemPrompt
	for _, tool := range o.tools {
		systemPrompt += fmt.Sprintf("- %s: %s", tool.Function.Name, tool.Function.Description)
	}

	response, _ := client.Prompt(context.TODO(), prompt,
		groq.WithSystemPrompt(systemPrompt),
		groq.WithMessages(o.messages...),
	)

	var unmarshalledResponse struct {
		Classification string `json:"classification"`
	}
	if err := json.Unmarshal([]byte(response), &unmarshalledResponse); err != nil {
		// TODO: Retry
		log.Printf("Failed to unmarshal interruption classification response: %v", err)
		return "", nil
	}

	switch unmarshalledResponse.Classification {
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
		return "", fmt.Errorf("unknown interruption type: %s", unmarshalledResponse.Classification)
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
