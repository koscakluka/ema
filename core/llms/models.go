package llms

type Model string

type ChatModel Model

type ModerationModel Model

type ModelCard struct {
	Name            string
	Capabilities    Capabilities
	ProductionReady bool
}

type Capabilities struct {
	ToolCalls bool

	JSONMode   bool
	JSONSchema bool

	Caching bool

	Reasoning              []string
	DefaultReasoningEffort string
	DisableReasoningOption *string
}
