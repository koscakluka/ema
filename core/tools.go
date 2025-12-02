package orchestration

import (
	"github.com/koscakluka/ema-core/core/llms"
)

func orchestrationTools(o *Orchestrator) []llms.Tool {
	return []llms.Tool{
		llms.NewTool("recording_control", "Turn on or off sound recording, might be referred to as 'listening'",
			map[string]llms.ParameterBase{
				"is_recording": {Type: "boolean", Description: "Whether to record or not"},
			},
			func(parameters struct {
				IsRecording bool `json:"is_recording"`
			}) (string, error) {
				o.SetAlwaysRecording(parameters.IsRecording)
				return "Success. Respond with a very short phrase", nil
			}),
		llms.NewTool("speaking_control", "Turn off agent's speaking ability. Might be referred to as 'muting'",
			map[string]llms.ParameterBase{
				"is_speaking": {Type: "boolean", Description: "Wheather to speak or not"},
			},
			func(parameters struct {
				IsSpeaking bool `json:"is_speaking"`
			}) (string, error) {
				o.SetSpeaking(parameters.IsSpeaking)
				return "Success. Respond with a very short phrase", nil
			}),
	}
}
