package llm

import (
	"context"
	"fmt"

	"github.com/koscakluka/ema/core/interruptions"
	"github.com/koscakluka/ema/core/llms"
)

func respond(t interruptionType, prompt string, o interruptions.OrchestratorV0) error {
	switch t {
	case InterruptionTypeContinuation:
		o.CancelTurn()
		found := -1
		count := 0
		for turn := range o.Turns().RValues {
			if turn.Role == llms.MessageRoleUser {
				found = count
				break
			}
			count++
		}

		if found == -1 {
			return nil
		}

		for range found {
			o.Turns().Pop()
		}

		lastUserTurn := o.Turns().Pop()
		if lastUserTurn != nil {
			o.QueuePrompt(lastUserTurn.Content + " " + prompt)
		} else {
			o.QueuePrompt(prompt)
		}
		return nil

	case InterruptionTypeClarification:
		o.CancelTurn()
		o.QueuePrompt(prompt)
		return nil

	case InterruptionTypeCancellation:
		o.CancelTurn()
		return nil

	case InterruptionTypeIgnorable,
		InterruptionTypeRepetition,
		InterruptionTypeNoise:
		return nil

	case InterruptionTypeAction:
		return o.CallToolWithPrompt(context.TODO(), prompt)

	case InterruptionTypeNewPrompt:
		o.QueuePrompt(prompt)
		return nil

	default:
		return fmt.Errorf("unknown interruption type: %s", t)
	}
}
