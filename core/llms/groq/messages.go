package groq

import (
	"github.com/koscakluka/ema/core/llms"
)

type message struct {
	Role       llms.MessageRole `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall       `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func toMessages(messages []llms.Turn) []message {
	var groqMessages []message
	for _, msg := range messages {
		groqMessages = append(groqMessages, toMessage(msg)...)
	}
	return groqMessages
}

func toMessage(msg llms.Turn) []message {
	var toolCalls []toolCall
	for _, tCall := range msg.ToolCalls {
		toolCalls = append(toolCalls, toolCall{
			ID:   tCall.ID,
			Type: tCall.Type,
			Function: toolCallFunction{
				Name:      tCall.Function.Name,
				Arguments: tCall.Function.Arguments,
			},
		})
	}
	switch msg.Role {
	case llms.MessageRoleSystem:
		return []message{{
			Role:       llms.MessageRoleSystem,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toolCalls,
		}}

	case llms.MessageRoleUser:
		return []message{{
			Role:       llms.MessageRoleUser,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toolCalls,
		}}

	case llms.MessageRoleTool:
		return []message{{
			Role:       llms.MessageRoleTool,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toolCalls,
		}}

	case llms.MessageRoleAssistant:
		return []message{{
			Role:       llms.MessageRoleAssistant,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toolCalls,
		}}
	}
	return []message{}
}
