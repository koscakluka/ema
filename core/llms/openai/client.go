package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/koscakluka/ema/core/llms"
)

const (
	envVarApiKeyName    = "OPENAI_API_KEY"
	envVarOrgIdName     = "OPENAI_ORG_ID"
	envVarProjectIdName = "OPENAI_PROJECT_ID"

	defaultPrompt = "You are a helpful assistant, keep the conversation going and answer any questions to the best of your ability. Reply concisely and clearly unless asked to expand on something. If told to not respond, respond with '...'."
)

type baseClient[T any] struct {
	apiKey    string
	orgId     string
	projectId string

	model        ChatModel
	modelVersion T

	systemPrompt string
}

func newBase[T any](model ChatModel, defaultModelVersion T, opts ...BaseOption[T]) (*baseClient[T], error) {
	options := &baseClient[T]{
		apiKey:    os.Getenv(envVarApiKeyName),
		orgId:     os.Getenv(envVarOrgIdName),
		projectId: os.Getenv(envVarProjectIdName),

		model:        model,
		modelVersion: defaultModelVersion,

		systemPrompt: defaultPrompt,
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.apiKey == "" {
		return nil, fmt.Errorf("openai api key neither found (OPENAI_API_KEY) nor provided")
	}

	return options, nil
}

type BaseOption[T any] func(*baseClient[T])

func WithSystemPrompt[T any](prompt string) BaseOption[T] {
	return func(c *baseClient[T]) {
		c.systemPrompt = prompt
	}
}

func WithAPIKey[T any](apiKey string) BaseOption[T] {
	return func(c *baseClient[T]) {
		c.apiKey = apiKey
	}
}

func WithOrganisationID[T any](orgId string) BaseOption[T] {
	return func(c *baseClient[T]) {
		c.orgId = orgId
	}
}

func WithProjectID[T any](projectId string) BaseOption[T] {
	return func(c *baseClient[T]) {
		c.projectId = projectId
	}
}

func WithModelVersion[T any](modelVersion T) BaseOption[T] {
	return func(c *baseClient[T]) {
		c.modelVersion = modelVersion
	}
}

type GPT4oClient struct{ baseClient[GPT4oVersion] }

func NewGPT4oClient(opts ...BaseOption[GPT4oVersion]) (*GPT4oClient, error) {
	base, err := newBase(ModelGPT4o, defaultGPT4oVersion, opts...)
	if err != nil {
		return nil, err
	}

	return &GPT4oClient{baseClient: *base}, nil
}

func (c *GPT4oClient) Prompt(ctx context.Context, prompt string, opts ...llms.GeneralPromptOption) (*llms.Message, error) {
	return Prompt(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}

func (c *GPT4oClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}

type GPT41Client struct{ baseClient[GPT41Version] }

func NewGPT41Client(opts ...BaseOption[GPT41Version]) (*GPT41Client, error) {
	base, err := newBase(ModelGPT41, defaultGPT41Version, opts...)
	if err != nil {
		return nil, err
	}

	return &GPT41Client{baseClient: *base}, nil
}

func (c *GPT41Client) Prompt(ctx context.Context, prompt string, opts ...llms.GeneralPromptOption) (*llms.Message, error) {
	return Prompt(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}

func (c *GPT41Client) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}

type GPT5NanoClient struct{ baseClient[GPT5NanoVersion] }

func NewGPT5NanoClient(opts ...BaseOption[GPT5NanoVersion]) (*GPT5NanoClient, error) {
	base, err := newBase(ModelGPT5Nano, defaultGPT5NanoVersion, opts...)
	if err != nil {
		return nil, err
	}

	return &GPT5NanoClient{baseClient: *base}, nil
}

func (c *GPT5NanoClient) Prompt(ctx context.Context, prompt string, opts ...llms.GeneralPromptOption) (*llms.Message, error) {
	return Prompt(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}

func (c *GPT5NanoClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, buildModelString(c.model, string(c.modelVersion)), prompt, c.systemPrompt, opts...)
}
