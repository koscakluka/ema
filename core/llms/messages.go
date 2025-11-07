package llms

// Message is a single message in a conversation, but actually it represents a
// response from an LLM. It is an alias for Response for backwards compatibility.
//
// Deprecated: Use Response instead
type Message Response

// Response is a single response from an LLM
type Response struct {
	Content   string
	ToolCalls []ToolCall

	// ToolCallID is the ID of the tool call that this response is responding to
	//
	// Deprecated: LLM should never respond to a tool call, this is only here
	// for backwards compatibility
	ToolCallID string
	// Role describes who the response (previously Message) is from
	//
	// Deprecated: Response always comes from the assistant, so the role would
	// always be the same assistant
	Role MessageRole
}

// Turn is a single turn taken in the conversation.
type Turn struct {
	Role TurnRole

	// Content is the content of the turn
	// In user's turn it is the prompt,
	// in assistant's turn it is the response
	Content   string
	ToolCalls []ToolCall

	// ToolCallID is the ID of the tool call that this turn is responding to
	//
	// Deprecated: The response is now a ToolCall property, this is only here
	// for backwards compatibility
	ToolCallID string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
	Response  string

	// Type is the type of tool call, e.g. function call
	//
	// Deprecated: All tool calls are function calls for us, no need to specify
	Type string
	// Function is the description of the tool call
	//
	// Deprecated: Use ToolCall Name and Arguments properties instead
	Function ToolCallFunction
}

// ToolCallFunction is a description of a tool call
//
// Deprecated: Use ToolCall Name and Arguments properties instead
type ToolCallFunction struct {
	Name      string
	Arguments string
}

// MessageRole describes who is the message from
//
// Deprecated: This is kept for backwards compatibility, but it will not be
// used anymore, llms should generate their own messages and message roles
// based on Turns content
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type TurnRole string

const (
	TurnRoleUser      TurnRole = "user"
	TurnRoleAssistant TurnRole = "assistant"
)
