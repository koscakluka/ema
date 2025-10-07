package llms

type Stream interface {
	Chunks(func(StreamChunk, error) bool)
}

type StreamChunk interface {
	// ID() string
	// Object() string
	// Created() int
	// Model() string
	// SystemFingerprint() string
	FinishReason() *string
}

type StreamRoleChunk interface {
	StreamChunk
	Role() string
}

type StreamReasoningChunk interface {
	StreamChunk
	Reasoning() string
	Channel() string
}

type StreamContentChunk interface {
	StreamChunk
	Content() string
}

type StreamToolCallChunk interface {
	StreamChunk
	ToolCall() ToolCall
}

type StreamUsageChunk interface {
	StreamChunk
	Usage() Usage
}

// TODO: See if this actually makes any sense
// type choiceBase struct {
// 	Index int
// 	// Logprobs any
// 	FinishReason *string
// }

type Usage struct {
	QueueTime               float64
	PromptTokens            int
	PromptTime              float64
	CompletionTokens        int
	CompletionTime          float64
	TotalTokens             int
	TotalTime               float64
	CompletionTokensDetails *CompletionTokensDetails
}

type CompletionTokensDetails struct {
	ReasoningTokens int
}
