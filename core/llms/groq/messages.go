package groq

import (
	"github.com/koscakluka/ema/core/llms"
)

type message struct {
	Role       messageRole `json:"role"`
	Content    string      `json:"content"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall  `json:"tool_calls,omitempty"`
}

type messageRole string

const (
	messageRoleSystem    messageRole = "system"
	messageRoleUser      messageRole = "user"
	messageRoleAssistant messageRole = "assistant"
	messageRoleTool      messageRole = "tool"
)

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func toMessages(instructions string, turns []llms.Turn) []message {
	messages := []message{}
	if instructions != "" {
		messages = append(messages, message{
			Role:    messageRoleSystem,
			Content: instructions,
		})
	}
	for _, turn := range turns {
		switch turn.Role {
		case llms.TurnRoleUser:
			messages = append(messages, message{
				Role:    messageRoleUser,
				Content: turn.Content,
			})

		case llms.TurnRoleAssistant:
			if len(turn.ToolCalls) > 0 {
				msg := message{Role: messageRoleAssistant}
				responseMsgs := []message{}
				for _, tCall := range turn.ToolCalls {
					msg.ToolCalls = append(msg.ToolCalls, toolCall{
						ID:   tCall.ID,
						Type: "function",
						Function: toolCallFunction{
							Name:      tCall.Name,
							Arguments: tCall.Arguments,
						},
					})
					if tCall.Response != "" {
						responseMsgs = append(responseMsgs, message{
							Role:       messageRoleTool,
							Content:    tCall.Response,
							ToolCallID: tCall.ID,
						})
					}
				}

				messages = append(messages, msg)
				messages = append(messages, responseMsgs...)
			}
			if len(turn.Content) > 0 {
				messages = append(messages, message{
					Role:    messageRoleAssistant,
					Content: turn.Content,
				})
			}
		}
	}
	return messages
}
