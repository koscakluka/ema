package orchestration

import (
	"slices"

	"github.com/koscakluka/ema/core/llms"
)

type Turns []llms.Turn

// Peek returns the last turn in the stored turns
func (t Turns) Peek() *llms.Turn {
	return &t[len(t)-1]
}

// Push adds a new turn to the stored turns
func (t *Turns) Push(turn llms.Turn) {
	*t = append(*t, turn)
}

// Pop removes the last turn from the stored turns, returns nil if empty
func (t *Turns) Pop() *llms.Turn {
	if len(*t) == 0 {
		return nil
	}
	turn := (*t)[len(*t)-1]
	*t = (*t)[:len(*t)-1]
	return &turn
}

// Clear removes all stored turns
func (t *Turns) Clear() {
	*t = (*t)[:0]
}

// Values is an iterator that goes over all the stored turns starting from the
// earliest towards the latest
func (t *Turns) Values(yield func(llms.Turn) bool) {
	for _, turn := range *t {
		if !yield(turn) {
			return
		}
	}
}

// Values is an iterator that goes over all the stored turns starting from the
// latest towards the earliest
func (t *Turns) RValues(yield func(llms.Turn) bool) {
	// TODO: There should be a better way to do this than creating a new
	// method just for reversing the order
	for _, turn := range slices.Backward(*t) {
		if !yield(turn) {
			return
		}
	}
}
