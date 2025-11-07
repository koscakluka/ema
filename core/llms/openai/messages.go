package openai

import "github.com/koscakluka/ema/core/llms"

type openAIMessage struct {
	Type messageType `json:"type"`

	Role    messageRole `json:"role,omitempty"`
	Content string      `json:"content,omitempty"`

	ToolCallID        string `json:"call_id,omitempty"`
	ToolCallName      string `json:"name,omitempty"`
	ToolCallArguments string `json:"arguments,omitempty"`
	ToolCallOutput    string `json:"output,omitempty"`
	ToolCallStatus    string `json:"status,omitempty"`
}

type messageRole string

const (
	messageRoleSystem    messageRole = "system"
	messageRoleDeveloper messageRole = "developer"
	messageRoleUser      messageRole = "user"
	messageRoleAssistant messageRole = "assistant"
	messageRoleTool      messageRole = "tool"
)

type messageType string

const (
	messageTypeMessage            messageType = "message"
	messageTypeFunctionCall       messageType = "function_call"
	messageTypeFunctionCallOutput messageType = "function_call_output"
)

func toOpenAIMessages(instructions string, messages []llms.Turn) []openAIMessage {
	openAIMessages := []openAIMessage{}
	if instructions != "" {
		openAIMessages = append(openAIMessages, openAIMessage{
			Role:    messageRoleDeveloper,
			Type:    messageTypeMessage,
			Content: instructions,
		})
	}
	for _, msg := range messages {
		openAIMessages = append(openAIMessages, toOpenAIMessage(msg)...)
	}
	return openAIMessages
}

func toOpenAIMessage(turn llms.Turn) []openAIMessage {
	switch turn.Role {
	case llms.TurnRoleUser:
		return []openAIMessage{{
			Type:    messageTypeMessage,
			Role:    messageRoleUser,
			Content: turn.Content,
		}}

	case llms.TurnRoleAssistant:
		if len(turn.ToolCalls) > 0 {
			oAIMsgs := []openAIMessage{}
			for _, toolCall := range turn.ToolCalls {
				oAIMsgs = append(oAIMsgs, openAIMessage{
					Type:              messageTypeFunctionCall,
					ToolCallID:        toolCall.ID,
					ToolCallName:      toolCall.Name,
					ToolCallArguments: toolCall.Arguments,
					ToolCallStatus:    "completed",
				})
				if toolCall.Response != "" {
					oAIMsgs = append(oAIMsgs, openAIMessage{
						Type:           messageTypeFunctionCallOutput,
						ToolCallID:     toolCall.ID,
						ToolCallOutput: toolCall.Response,
					})
				}
			}
			return oAIMsgs
		}
		if len(turn.Content) > 0 {
			return []openAIMessage{{
				Type:    messageTypeMessage,
				Role:    messageRoleAssistant,
				Content: turn.Content,
			}}
		}
	}

	return []openAIMessage{}
}
