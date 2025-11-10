package context

import "github.com/koscakluka/ema/core/llms"

type TurnsV0 interface {
	Push(turn llms.Turn)
	Pop() *llms.Turn

	Clear()

	Values(yield func(llms.Turn) bool)
	RValues(yield func(llms.Turn) bool)
}
