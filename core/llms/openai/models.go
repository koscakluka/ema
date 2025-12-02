package openai

import (
	"github.com/koscakluka/ema-core/core/llms"
)

const (
	defaultGPT4oVersion    = GPT4oVersion20240806
	defaultGPT41Version    = GPT41Version20250414
	defaultGPT5NanoVersion = GPT5NanoVersion20250807
)

type ChatModel llms.ChatModel

type (
	GPT4oVersion    string
	GPT41Version    string
	GPT5NanoVersion string
)

const (
	ModelGPT4o           ChatModel    = "gpt-4o"
	GPT4oVersion20240513 GPT4oVersion = "2024-05-13"
	GPT4oVersion20240806 GPT4oVersion = "2024-08-06"
	GPT4oVersion20241120 GPT4oVersion = "2024-11-20"

	ModelGPT41           ChatModel    = "gpt-4.1"
	GPT41Version20250414 GPT41Version = "2025-04-14"

	ModelGPT5Nano           ChatModel       = "gpt-5-nano"
	GPT5NanoVersion20250807 GPT5NanoVersion = "2025-08-07"
)

// ModelCards contain parsed information about the models
// For more information check: https://console.groq.com/docs/models
var ModelCards = map[ChatModel]llms.ModelCard{
	// ModelLlama318BInstant: {
	// 	Name:            string(ModelLlama318BInstant),
	// 	ProductionReady: true,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true},
	// },
	// ModelLlama3370BVersatile: {
	// 	Name:            string(ModelLlama3370BVersatile),
	// 	ProductionReady: true,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true},
	// },
	// ModelGPTOSS20B: {
	// 	Name:            string(ModelGPTOSS20B),
	// 	ProductionReady: true,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true, Reasoning: []string{"low", "medium", "high"}, DefaultReasoningEffort: "medium"},
	// },
	// ModelGPTOSS120B: {
	// 	Name:            string(ModelGPTOSS120B),
	// 	ProductionReady: true,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true, Reasoning: []string{"low", "medium", "high"}, DefaultReasoningEffort: "medium"},
	// },
	//
	// ModelLlama4Maverick17BInstruct: {
	// 	Name:            string(ModelLlama4Maverick17BInstruct),
	// 	ProductionReady: false,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	// },
	// ModelLlama4Scout17BInstruct: {
	// 	Name:            string(ModelLlama4Scout17BInstruct),
	// 	ProductionReady: false,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	// },
	// ModelKimiK2Instruct0905: {
	// 	Name:            string(ModelKimiK2Instruct0905),
	// 	ProductionReady: false,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	// },
	// ModelQwen332B: {
	// 	Name:            string(ModelQwen332B),
	// 	ProductionReady: false,
	// 	Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, Reasoning: []string{"none", "default"}, DefaultReasoningEffort: "default", DisableReasoningOption: utils.Ptr("none")},
	// },
}

func buildModelString(model ChatModel, version string) string {
	return string(model) + "-" + version
}
