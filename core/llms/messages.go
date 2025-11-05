package llms

// Message is a single message in a conversation, but actually it represents a
// single turn. It is just an alias for Turn for backwards compatibility.
//
// Deprecated: use Turn instead
type Message Turn

// Turn is a single turn taken in the conversation.
type Turn struct {
	Role       MessageRole
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}

type ToolCall struct {
	ID       string
	Type     string
	Function ToolCallFunction
}

type ToolCallFunction struct {
	Name      string
	Arguments string
}

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)
