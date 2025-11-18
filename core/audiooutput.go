package orchestration

func (o *Orchestrator) passSpeechToAudioOutput() {
bufferReadingLoop:
	for audioOrMark := range o.buffer.Audio {
		switch audioOrMark.Type {
		case "audio":
			audio := audioOrMark.Audio
			if o.orchestrateOptions.onAudio != nil {
				o.orchestrateOptions.onAudio(audio)
			}

			if o.audioOutput == nil {
				continue bufferReadingLoop
			}

			if !o.IsSpeaking || (o.turns.activeTurn() != nil && o.turns.activeTurn().Cancelled) {
				o.audioOutput.ClearBuffer()
				break bufferReadingLoop
			}

			o.audioOutput.SendAudio(audio)

		case "mark":
			mark := audioOrMark.Mark
			if o.audioOutput != nil {
				switch o.audioOutput.(type) {
				case AudioOutputV1:
					o.audioOutput.(AudioOutputV1).Mark(mark, func(mark string) {
						o.buffer.MarkPlayed(mark)
					})
				case AudioOutputV0:
					go func() {
						o.audioOutput.(AudioOutputV0).AwaitMark()
						o.buffer.MarkPlayed(mark)
					}()
				}
			} else {
				o.buffer.MarkPlayed(mark)
			}
		}
	}

	defer func() {
		o.finaliseActiveTurn()
		o.promptEnded.Done()
	}()

	if o.orchestrateOptions.onAudioEnded != nil {
		o.orchestrateOptions.onAudioEnded(o.buffer.audioTranscript)
	}

	if o.audioOutput == nil {
		return
	}

	// TODO: Figure out why this is needed
	// o.audioOutput.SendAudio([]byte{})

	if !o.IsSpeaking || (o.turns.activeTurn() != nil && o.turns.activeTurn().Cancelled) {
		o.audioOutput.ClearBuffer()
		return
	}

}
