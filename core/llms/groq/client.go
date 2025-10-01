package groq

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/koscakluka/ema/core/llms"
)

const (
	defaultModel  = "llama-3.3-70b-versatile"
	defaultPrompt = "You are a helpful assistant, keep the conversation going and answer any questions to the best of your ability. Reply concisely and clearly unless asked to expand on something. If told to not respond, respond with '...'."

	url = "https://api.groq.com/openai/v1/chat/completions"

	endMessage  = "[DONE]"
	chunkPrefix = "data:"
)

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
	return promptWithStream(ctx, c.apiKey, defaultModel, prompt, c.systemPrompt, c.tools, opts...)
}
