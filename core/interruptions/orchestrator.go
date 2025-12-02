package interruptions

import (
	"context"

	emaContext "github.com/koscakluka/ema-core/core/context"
)

type OrchestratorV0 interface {
	Turns() emaContext.TurnsV0

	CallTool(ctx context.Context, prompt string) error

	QueuePrompt(prompt string)

	CancelTurn()
}
