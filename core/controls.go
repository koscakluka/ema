package orchestration

func (o *Orchestrator) CancelTurn() {
	// TODO: This could potentially be done directly on the turn instead of
	// as an exposed method
	activeTurn := o.turns.activeTurn()
	if activeTurn != nil && !activeTurn.Cancelled {
		activeTurn.Cancelled = true
		o.turns.updateActiveTurn(*activeTurn)
		if o.orchestrateOptions.onCancellation != nil {
			o.orchestrateOptions.onCancellation()
		}
		o.UnpauseTurn()
	}
}

func (o *Orchestrator) PauseTurn() {
	if o.audioOutput != nil {
		o.audioOutput.ClearBuffer()
	}
	o.buffer.PauseAudio()
}

func (o *Orchestrator) UnpauseTurn() {
	o.buffer.UnpauseAudio()
}
