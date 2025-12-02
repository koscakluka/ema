package groq

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/koscakluka/ema-core/core/llms"
)

const (
	envVarApiKeyName = "GROQ_API_KEY"

	defaultModel  = "llama-3.3-70b-versatile"
	defaultPrompt = "You are a helpful assistant, keep the conversation going and answer any questions to the best of your ability. Reply concisely and clearly unless asked to expand on something. If told to not respond, respond with '...'."
)

type Client struct {
	apiKey string

	model        string
	tools        []llms.Tool
	systemPrompt string
}

// NewClient is DEPRECATED, use individual model constructors
func NewClient(opts ...ClientOption) (*Client, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		apiKey:       options.apiKey,
		model:        options.model,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

type ClientOptions struct {
	apiKey       string
	model        string
	tools        []llms.Tool
	systemPrompt string
}

type ClientOption func(*ClientOptions)

// WithModel defines the model that the client will use.
//
// Depricated: use individual model constructors, this one is for backwards
// compatibility and isn't typesafe (i.e. doesn't check for invalid models)
func WithModel(model string) ClientOption {
	return func(c *ClientOptions) {
		c.model = model
	}
}

func WithTools(tools ...llms.Tool) ClientOption {
	return func(c *ClientOptions) {
		c.tools = slices.Clone(tools)
	}
}

func WithSystemPrompt(prompt string) ClientOption {
	return func(c *ClientOptions) {
		c.systemPrompt = prompt
	}
}

func WithAPIKey(apiKey string) ClientOption {
	return func(c *ClientOptions) {
		c.apiKey = apiKey
	}
}

func (c *Client) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, defaultModel, prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Client) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, defaultModel, prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Client) ModelCard() llms.ModelCard {
	card, ok := ModelCards[ChatModel(c.model)]
	if !ok {
		return llms.ModelCard{}
	}

	return card
}

func populateOptions(opts ...ClientOption) (*ClientOptions, error) {
	options := &ClientOptions{
		apiKey: os.Getenv(envVarApiKeyName),

		model:        defaultModel,
		systemPrompt: defaultPrompt,
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.apiKey == "" {
		return nil, fmt.Errorf("groq api key neither found (GROQ_API_KEY) nor provided")
	}

	return options, nil
}

type Llmaa3370BVersatileClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewLlama3370BVersatileClient(opts ...ClientOption) (*Llmaa3370BVersatileClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Llmaa3370BVersatileClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *Llmaa3370BVersatileClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelLlama3370BVersatile), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llmaa3370BVersatileClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelLlama3370BVersatile), prompt, c.systemPrompt, c.tools, opts...)
}

type Llama318BInstructClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewLlama318BInstructClient(opts ...ClientOption) (*Llama318BInstructClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Llama318BInstructClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *Llama318BInstructClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelLlama318BInstant), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llama318BInstructClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelLlama318BInstant), prompt, c.systemPrompt, c.tools, opts...)
}

type GPTOSS20BClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewGPTOSS20BClient(opts ...ClientOption) (*GPTOSS20BClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &GPTOSS20BClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *GPTOSS20BClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelGPTOSS20B), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *GPTOSS20BClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelGPTOSS20B), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *GPTOSS20BClient) PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error {
	_, err := PromptJSONSchema(ctx, c.apiKey, string(ModelGPTOSS20B), prompt, c.systemPrompt, outputSchema, opts...)
	return err
}

type GPTOSS120BClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewGPTOSS120BClient(opts ...ClientOption) (*GPTOSS120BClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &GPTOSS120BClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *GPTOSS120BClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelGPTOSS120B), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *GPTOSS120BClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelGPTOSS120B), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *GPTOSS120BClient) PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error {
	_, err := PromptJSONSchema(ctx, c.apiKey, string(ModelGPTOSS120B), prompt, c.systemPrompt, outputSchema, opts...)
	return err
}

type Llama4Maverick17BInstructClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewLlama4Maverick17BInstructClient(opts ...ClientOption) (*Llama4Maverick17BInstructClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Llama4Maverick17BInstructClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *Llama4Maverick17BInstructClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelLlama4Maverick17BInstruct), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llama4Maverick17BInstructClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelLlama4Maverick17BInstruct), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llama4Maverick17BInstructClient) PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error {
	_, err := PromptJSONSchema(ctx, c.apiKey, string(ModelLlama4Maverick17BInstruct), prompt, c.systemPrompt, outputSchema, opts...)
	return err
}

type Llama4Scout17BInstructClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewLlama4Scout17BInstructClient(opts ...ClientOption) (*Llama4Scout17BInstructClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Llama4Scout17BInstructClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *Llama4Scout17BInstructClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelLlama4Scout17BInstruct), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llama4Scout17BInstructClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelLlama4Scout17BInstruct), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Llama4Scout17BInstructClient) PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error {
	_, err := PromptJSONSchema(ctx, c.apiKey, string(ModelLlama4Scout17BInstruct), prompt, c.systemPrompt, outputSchema, opts...)
	return err
}

type KimiK2Instruct0905Client struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewKimiK2Instruct0905Client(opts ...ClientOption) (*KimiK2Instruct0905Client, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &KimiK2Instruct0905Client{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *KimiK2Instruct0905Client) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelKimiK2Instruct0905), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *KimiK2Instruct0905Client) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelKimiK2Instruct0905), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *KimiK2Instruct0905Client) PromptWithStructure(ctx context.Context, prompt string, outputSchema any, opts ...llms.StructuredPromptOption) error {
	_, err := PromptJSONSchema(ctx, c.apiKey, string(ModelKimiK2Instruct0905), prompt, c.systemPrompt, outputSchema, opts...)
	return err
}

type Qwen332BClient struct {
	apiKey string

	tools        []llms.Tool
	systemPrompt string
}

func NewQwen332BClient(opts ...ClientOption) (*Qwen332BClient, error) {
	options, err := populateOptions(opts...)
	if err != nil {
		return nil, err
	}

	return &Qwen332BClient{
		apiKey:       options.apiKey,
		tools:        options.tools,
		systemPrompt: options.systemPrompt,
	}, nil
}

func (c *Qwen332BClient) Prompt(ctx context.Context, prompt string, opts ...llms.PromptOption) ([]llms.Message, error) {
	return Prompt(ctx, c.apiKey, string(ModelQwen332B), prompt, c.systemPrompt, c.tools, opts...)
}

func (c *Qwen332BClient) PromptWithStream(ctx context.Context, prompt *string, opts ...llms.StreamingPromptOption) llms.Stream {
	return PromptWithStream(ctx, c.apiKey, string(ModelQwen332B), prompt, c.systemPrompt, c.tools, opts...)
}
