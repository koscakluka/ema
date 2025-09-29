package groq

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/koscakluka/ema/core/llms"
	"github.com/koscakluka/ema/internal/utils"
)

const (
	defaultModel  = "llama-3.3-70b-versatile"
	defaultPrompt = "You are a helpful assistant, keep the conversation going and answer any questions to the best of your ability. Reply concisely and clearly unless asked to expand on something. If told to not respond, respond with '...'."

	url = "https://api.groq.com/openai/v1/chat/completions"

	endMessage  = "[DONE]"
	chunkPrefix = "data:"
)

type message struct {
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Role       llms.MessageRole `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []toolCall       `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type requestBody struct {
	Model      string    `json:"model"`
	Messages   []message `json:"messages"`
	Stream     bool      `json:"stream"`
	ToolChoice *string   `json:"tool_choice,omitempty"`
	Tools      []Tool    `json:"tools,omitempty"`
}

type responseBody struct {
	Choices []struct {
		Delta struct {
			Content   string     `json:"content"`
			ToolCalls []toolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

type Client struct {
	apiKey string

	model        string
	tools        []llms.Tool
	systemPrompt llms.Message
}

func NewClient(opts ...ClientOption) (*Client, error) {
	client := Client{
		model: defaultModel,
		systemPrompt: llms.Message{
			Role:    llms.MessageRoleSystem,
			Content: defaultPrompt,
		},
	}

	for _, opt := range opts {
		opt(&client)
	}

	if client.apiKey == "" {
		apiKey, ok := os.LookupEnv("GROQ_API_KEY")
		if !ok {
			return nil, fmt.Errorf("groq api key neither found (GROQ_API_KEY) nor provided")
		}

		client.apiKey = apiKey
	}

	return &client, nil
}

type ClientOption func(*Client)

func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

func WithTools(tools ...llms.Tool) ClientOption {
	return func(c *Client) {
		c.tools = slices.Clone(tools)
	}
}

func WithSystemPrompt(prompt string) ClientOption {
	return func(c *Client) {
		c.systemPrompt = llms.Message{
			Role:    llms.MessageRoleSystem,
			Content: prompt,
		}
	}
}

func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func (c *Client) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	options := llms.PromptOptions{
		Messages: []llms.Message{c.systemPrompt},
		Tools:    slices.Clone(c.tools),
	}
	for _, opt := range opts {
		opt(&options)
	}

	var messages []message
	copier.Copy(&messages, options.Messages)
	messages = append(messages, message{
		Role:    llms.MessageRoleUser,
		Content: prompt,
	})

	var toolChoice *string
	var tools []Tool
	if options.Tools != nil {
		toolChoice = utils.Ptr("auto")

		if options.ForcedToolsCall {
			toolChoice = utils.Ptr("required")
		}
		copier.Copy(&tools, options.Tools)
	}

	responses := []message{}

	for {
		reqBody := requestBody{
			Model:      c.model,
			Messages:   messages,
			Stream:     true,
			Tools:      tools,
			ToolChoice: toolChoice,
		}

		requestBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshalling JSON: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
		if err != nil {
			return nil, fmt.Errorf("error creating HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// TODO: Retry depending on status, send back a message to the user
			// to indicate that something is going on
			log.Println("Non-OK HTTP status:", resp.Status)
		}

		toolCalls := []toolCall{}
		var response strings.Builder
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
			var responseBody responseBody
			err := json.Unmarshal([]byte(chunk), &responseBody)
			if err != nil {
				log.Println("Error unmarshalling JSON:", err)
				continue
			}
			if len(responseBody.Choices) == 0 {
				continue
			}
			if len(responseBody.Choices[0].Delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, responseBody.Choices[0].Delta.ToolCalls...)
			}

			content := responseBody.Choices[0].Delta.Content
			response.WriteString(content)
			if options.Stream != nil {
				options.Stream(content)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Println("Error reading streamed response:", err)
		}

		msg := message{
			Role:      llms.MessageRoleAssistant,
			Content:   response.String(),
			ToolCalls: toolCalls,
		}
		messages = append(messages, msg)
		responses = append(responses, msg)
		if len(toolCalls) == 0 {
			llmResponses := []llms.Message{}
			copier.Copy(&llmResponses, responses)
			return llmResponses, nil
		}

		for _, toolCall := range toolCalls {
			for _, tool := range options.Tools {
				if tool.Function.Name == toolCall.Function.Name {
					resp, err := tool.Execute(toolCall.Function.Arguments)
					if err != nil {
						log.Println("Error executing tool:", err)
					}
					msg := message{
						ToolCallID: toolCall.ID,
						Role:       llms.MessageRoleTool,
						Content:    resp,
					}
					messages = append(messages, msg)
					responses = append(responses, msg)
				}
			}

		}

	}
}
