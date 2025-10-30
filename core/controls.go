package orchestration

func (o *Orchestrator) Cancel() {
	if o.activePrompt != nil {
		o.canceled = true
		if o.orchestrateOptions.onCancellation != nil {
			o.orchestrateOptions.onCancellation()
		}
	}
}
