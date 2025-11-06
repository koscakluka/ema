package interruptions

import (
	"context"

	emaContext "github.com/koscakluka/ema/core/context"
	"github.com/koscakluka/ema/core/llms"
)

type OrchestratorV0 interface {
	Turns() emaContext.TurnsV0

	CallTool(ctx context.Context, toolCall llms.ToolCall) (*llms.Turn, error)
	CallToolWithPrompt(ctx context.Context, prompt string) error

	QueuePrompt(prompt string)

	CancelTurn()
}
