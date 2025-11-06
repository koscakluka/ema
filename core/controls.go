package orchestration

func (o *Orchestrator) CancelTurn() {
	// TODO: This could potentially be done directly on the turn instead of
	// as an exposed method
	if o.activePrompt != nil {
		o.canceled = true
		if o.orchestrateOptions.onCancellation != nil {
			o.orchestrateOptions.onCancellation()
		}
	}
}

// Cancel is an alias for CancelTurn
//
// Deprecated: use CancelTurn instead
func (o *Orchestrator) Cancel() {
	o.CancelTurn()
}
