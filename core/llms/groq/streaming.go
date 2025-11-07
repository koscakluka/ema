package groq

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/internal/utils"
)

func PromptWithStream(
	_ context.Context,
	apiKey string,
	model string,
	prompt *string,
	systemPrompt string,
	baseTools []llms.Tool,
	opts ...llms.StreamingPromptOption,
) *Stream {
	options := llms.StreamingPromptOptions{
		GeneralPromptOptions: llms.GeneralPromptOptions{
			Tools: slices.Clone(baseTools),
		},
	}
	if systemPrompt != "" {
		options.BaseOptions.Messages = append(options.BaseOptions.Messages, llms.Message{
			Role: llms.MessageRoleSystem, Content: systemPrompt,
		})
		options.BaseOptions.Turns = append(options.BaseOptions.Turns, llms.Turn{
			Role: llms.MessageRoleSystem, Content: systemPrompt,
		})
	}
	for _, opt := range opts {
		opt.ApplyToStreaming(&options)
	}

	messages := toMessages(options.BaseOptions.Turns)
	if prompt != nil {
		messages = append(messages, message{
			Role:    llms.MessageRoleUser,
			Content: *prompt,
		})
	}
	log.Println("Messages:", messages)

	var tools []Tool
	if options.GeneralPromptOptions.Tools != nil {
		copier.Copy(&tools, options.GeneralPromptOptions.Tools)
	}

	return &Stream{
		apiKey:   apiKey,
		model:    model,
		tools:    tools,
		messages: messages,
	}

}

type Stream struct {
	apiKey string

	model    string
	tools    []Tool
	messages []message
}

func (s *Stream) Chunks(yield func(llms.StreamChunk, error) bool) {

	var toolChoice *string
	if s.tools != nil {
		toolChoice = utils.Ptr("auto")
	}

	reqBody := requestBody{
		Model:      s.model,
		Messages:   s.messages,
		Stream:     true,
		Tools:      s.tools,
		ToolChoice: toolChoice,
	}

	requestBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		yield(nil, fmt.Errorf("error marshalling JSON: %w", err))
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		yield(nil, fmt.Errorf("error creating HTTP request: %w", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		yield(nil, fmt.Errorf("error sending request: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Retry depending on status, send back a message to the user
		// to indicate that something is going on
		yield(nil, fmt.Errorf("non-OK HTTP status: %s", resp.Status))
		return
	}

	toolCalls := []toolCall{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		chunk := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), chunkPrefix))

		if len(chunk) == 0 {
			continue
		}

		if chunk == endMessage {
			break
		}

		// log.Println("Chunk:", chunk)
		var responseBody streamingResponseBody
		err := json.Unmarshal([]byte(chunk), &responseBody)
		if err != nil {
			if !yield(nil, fmt.Errorf("error unmarshalling JSON: %w", err)) {
				return
			}
			continue
		}
		var finishReason *string
		if len(responseBody.Choices) > 0 {
			delta := responseBody.Choices[0].Delta

			if delta.FinishReason != nil {
				finishReason = delta.FinishReason
			}

			if len(delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, delta.ToolCalls...)
				for _, toolCall := range delta.ToolCalls {
					if !yield(StreamToolCallChunk{
						finishReason: finishReason,
						toolCall: llms.ToolCall{
							ID:   toolCall.ID,
							Type: toolCall.Type,
							Function: llms.ToolCallFunction{
								Name:      toolCall.Function.Name,
								Arguments: toolCall.Function.Arguments,
							},
						},
					}, nil) {
						return
					}
				}
			}

			if delta.Content != "" {
				content := delta.Content
				if !yield(StreamContentChunk{
					finishReason: finishReason,
					content:      content,
				}, nil) {
					return
				}
			}

			if delta.Reasoning != "" {
				reasoning := delta.Reasoning
				if !yield(StreamReasoningChunk{
					finishReason: finishReason,
					reasoning:    reasoning,
					channel:      delta.Channel,
				}, nil) {
					return
				}
			}
		}

		if responseBody.Usage != nil {
			var outputTokensDetails *llms.OutputTokensDetails
			var completionTokensDetails *llms.CompletionTokensDetails
			if responseBody.Usage.CompletionTokensDetails != nil {
				completionTokensDetails = utils.Ptr(llms.CompletionTokensDetails{
					ReasoningTokens: responseBody.Usage.CompletionTokensDetails.ReasoningTokens,
				})
				outputTokensDetails = utils.Ptr(llms.OutputTokensDetails{
					ReasoningTokens: responseBody.Usage.CompletionTokensDetails.ReasoningTokens,
				})
			}

			if !yield(StreamUsageChunk{
				finishReason: finishReason,
				usage: llms.Usage{
					InputTokens:             responseBody.Usage.PromptTokens,
					PromptTokens:            responseBody.Usage.PromptTokens,
					OutputTokens:            responseBody.Usage.CompletionTokens,
					CompletionTokens:        responseBody.Usage.CompletionTokens,
					CompletionTokensDetails: completionTokensDetails,
					OutputTokensDetails:     outputTokensDetails,
					TotalTokens:             responseBody.Usage.TotalTokens,

					QueueTime:      responseBody.Usage.QueueTime,
					PromptTime:     responseBody.Usage.PromptTime,
					CompletionTime: responseBody.Usage.CompletionTime,
					TotalTime:      responseBody.Usage.TotalTime,
				},
			}, nil) {
				return
			}

		}
	}

	if err := scanner.Err(); err != nil {
		yield(nil, fmt.Errorf("error reading streamed response: %w", err))
		return
	}
}

type StreamRoleChunk struct {
	finishReason *string
	role         string
}

func (s StreamRoleChunk) FinishReason() *string {
	return s.finishReason
}

func (s StreamRoleChunk) Role() string {
	return s.role
}

type StreamReasoningChunk struct {
	finishReason *string
	reasoning    string
	channel      string
}

func (s StreamReasoningChunk) FinishReason() *string {
	return s.finishReason
}

func (s StreamReasoningChunk) Reasoning() string {
	return s.reasoning
}

func (s StreamReasoningChunk) Channel() string {
	return s.channel
}

type StreamContentChunk struct {
	finishReason *string
	content      string
}

func (s StreamContentChunk) FinishReason() *string {
	return s.finishReason
}

func (s StreamContentChunk) Content() string {
	return s.content
}

type StreamToolCallChunk struct {
	finishReason *string
	toolCall     llms.ToolCall
}

func (s StreamToolCallChunk) FinishReason() *string {
	return s.finishReason
}

func (s StreamToolCallChunk) ToolCall() llms.ToolCall {
	return s.toolCall
}

type StreamUsageChunk struct {
	finishReason *string
	usage        llms.Usage
}

func (s StreamUsageChunk) FinishReason() *string {
	return s.finishReason
}

func (s StreamUsageChunk) Usage() llms.Usage {
	return s.usage
}
