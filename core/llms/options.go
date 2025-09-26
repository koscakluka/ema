package llms

type PromptOptions struct {
	Messages        []Message
	Stream          func(string)
	Tools           []Tool
	ForcedToolsCall bool
}

type PromptOption func(*PromptOptions)

func WithStream(stream func(string)) PromptOption {
	return func(opts *PromptOptions) {
		opts.Stream = stream
	}
}

func WithSystemPrompt(prompt string) PromptOption {
	return func(opts *PromptOptions) {
		if len(opts.Messages) == 0 {
			opts.Messages = append(opts.Messages, Message{
				Role:    MessageRoleSystem,
				Content: prompt,
			})
		} else if opts.Messages[0].Role == MessageRoleSystem {
			opts.Messages[0].Content = prompt
		} else {
			opts.Messages = append([]Message{{
				Role:    MessageRoleSystem,
				Content: prompt,
			}}, opts.Messages...)
		}
	}
}

func WithMessages(messages ...Message) PromptOption {
	return func(opts *PromptOptions) {
		opts.Messages = append(opts.Messages, messages...)
	}
}

func WithTools(tools ...Tool) PromptOption {
	return func(opts *PromptOptions) {
		opts.Tools = append(opts.Tools, tools...)
	}
}

func WithForcedTools(tools ...Tool) PromptOption {
	return func(opts *PromptOptions) {
		opts.Tools = tools
	}
}
