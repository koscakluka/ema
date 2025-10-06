package groq

import (
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/internal/utils"
)

type ChatModel llms.ChatModel

const (
	ModelLlama318BInstant    ChatModel = "llama-3.1-8b-instant"
	ModelLlama3370BVersatile ChatModel = "llama-3.3-70b-versatile"
	ModelGPTOSS20B           ChatModel = "openai/gpt-oss-20b"
	ModelGPTOSS120B          ChatModel = "openai/gpt-oss-120b"

	ModelLlama4Maverick17BInstruct ChatModel = "meta-llama/llama-4-maverick-17b-128e-instruct"
	ModelLlama4Scout17BInstruct    ChatModel = "meta-llama/llama-4-scout-17b-16e-instruct"
	ModelKimiK2Instruct0905        ChatModel = "moonshotai/kimi-k2-instruct-0905"
	ModelQwen332B                  ChatModel = "qwen/qwen3-32b"

	// GroqCompound ChatModel = "groq/compound"
	// GroqCompoundMini ChatModel = "groq/compound-mini"

	// ModelLlamaGuard412B ModerationModel = "meta-llama/llama-guard-4-12b"
	// ModelLlamaPromptGuard222M ModerationModel = "meta-llama/llama-prompt-guard-2-22m"
	// ModelLlamaPromptGuard286M ModerationModel = "meta-llama/llama-prompt-guard-2-86m"

)

// ModelCards contain parsed information about the models
// For more information check: https://console.groq.com/docs/models
var ModelCards = map[ChatModel]llms.ModelCard{
	ModelLlama318BInstant: {
		Name:            string(ModelLlama318BInstant),
		ProductionReady: true,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true},
	},
	ModelLlama3370BVersatile: {
		Name:            string(ModelLlama3370BVersatile),
		ProductionReady: true,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true},
	},
	ModelGPTOSS20B: {
		Name:            string(ModelGPTOSS20B),
		ProductionReady: true,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true, Reasoning: []string{"low", "medium", "high"}, DefaultReasoningEffort: "medium"},
	},
	ModelGPTOSS120B: {
		Name:            string(ModelGPTOSS120B),
		ProductionReady: true,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true, Reasoning: []string{"low", "medium", "high"}, DefaultReasoningEffort: "medium"},
	},

	ModelLlama4Maverick17BInstruct: {
		Name:            string(ModelLlama4Maverick17BInstruct),
		ProductionReady: false,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	},
	ModelLlama4Scout17BInstruct: {
		Name:            string(ModelLlama4Scout17BInstruct),
		ProductionReady: false,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	},
	ModelKimiK2Instruct0905: {
		Name:            string(ModelKimiK2Instruct0905),
		ProductionReady: false,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, JSONSchema: true},
	},
	ModelQwen332B: {
		Name:            string(ModelQwen332B),
		ProductionReady: false,
		Capabilities:    llms.Capabilities{ToolCalls: true, JSONMode: true, Reasoning: []string{"none", "default"}, DefaultReasoningEffort: "default", DisableReasoningOption: utils.Ptr("none")},
	},
}
