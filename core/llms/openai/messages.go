package openai

import "github.com/koscakluka/ema/core/llms"

type openAIMessage struct {
	Type string `json:"type"`

	Role    llms.MessageRole `json:"role,omitempty"`
	Content string           `json:"content,omitempty"`

	ToolCallID        string `json:"call_id,omitempty"`
	ToolCallName      string `json:"name,omitempty"`
	ToolCallArguments string `json:"arguments,omitempty"`
	ToolCallOutput    string `json:"output,omitempty"`
	ToolCallStatus    string `json:"status,omitempty"`
}

// MessageRoleDeveloper is a special role used by OpenAI that makes a difference
// between OpenAI defined message and developer defined message.
// Developer messages are lower priority than OpenAI (i.e. "System") messages.
const MessageRoleDeveloper llms.MessageRole = "developer"

func toOpenAIMessages(messages []llms.Message) []openAIMessage {
	var openaiMessages []openAIMessage
	for _, msg := range messages {
		openaiMessages = append(openaiMessages, toOpenAIMessage(msg)...)
	}
	return openaiMessages
}

func toOpenAIMessage(msg llms.Message) []openAIMessage {
	switch msg.Role {
	case llms.MessageRoleSystem:
		return []openAIMessage{{
			Type:    "message",
			Role:    MessageRoleDeveloper,
			Content: msg.Content,
		}}

	case llms.MessageRoleUser:
		return []openAIMessage{{
			Type:    "message",
			Role:    llms.MessageRoleUser,
			Content: msg.Content,
		}}

	case llms.MessageRoleTool:
		return []openAIMessage{{
			Type:           "function_call_output",
			ToolCallID:     msg.ToolCallID,
			ToolCallOutput: msg.Content,
		}}

	case llms.MessageRoleAssistant:
		if len(msg.ToolCalls) > 0 {
			oAIMsgs := []openAIMessage{}
			for _, toolCall := range msg.ToolCalls {
				oAIMsgs = append(oAIMsgs, openAIMessage{
					Type:              "function_call",
					ToolCallID:        toolCall.ID,
					ToolCallName:      toolCall.Function.Name,
					ToolCallArguments: toolCall.Function.Arguments,
					ToolCallStatus:    "completed",
				})
			}
			return oAIMsgs
		} else if len(msg.Content) > 0 {
			return []openAIMessage{{
				Type:    "message",
				Role:    llms.MessageRoleAssistant,
				Content: msg.Content,
			}}
		}
	}

	return []openAIMessage{}
}
