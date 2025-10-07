package llms

// PromptOptions is a struct that contains all the options for a prompt. It is
// used as a base for both general and streaming prompt options.
//
// Deprecated: this struct will be removed and replaced with a more specific
// option patterns
type PromptOptions struct {
	Messages        []Message
	Stream          func(string)
	Tools           []Tool
	ForcedToolsCall bool
}

type BaseOptions struct {
	Messages []Message
}

type GeneralPromptOptions struct {
	BaseOptions
	PromptOptions
	Tools           []Tool
	ForcedToolsCall bool
}

type StreamingPromptOptions struct {
	GeneralPromptOptions
}

// PromptOption is a function that can be used to modify the prompt options.
//
// Deprecated: this type will be removed and replaced with a more specific
// option patterns
type PromptOption func(*PromptOptions)

type GeneralPromptOption interface {
	ApplyToGeneral(*GeneralPromptOptions)
}

type StreamingPromptOption interface {
	ApplyToStreaming(*StreamingPromptOptions)
}

func (f PromptOption) ApplyToGeneral(o *GeneralPromptOptions) {
	o.PromptOptions.Messages = o.BaseOptions.Messages
	o.PromptOptions.Tools = o.Tools
	o.PromptOptions.ForcedToolsCall = o.ForcedToolsCall
	f(&o.PromptOptions)
	o.BaseOptions.Messages = o.PromptOptions.Messages
	o.Tools = o.PromptOptions.Tools
	o.ForcedToolsCall = o.PromptOptions.ForcedToolsCall
}

func (f PromptOption) ApplyToStreaming(o *StreamingPromptOptions) {
	o.PromptOptions.Messages = o.GeneralPromptOptions.BaseOptions.Messages
	o.PromptOptions.Tools = o.GeneralPromptOptions.Tools
	o.PromptOptions.ForcedToolsCall = o.GeneralPromptOptions.ForcedToolsCall
	f(&o.PromptOptions)
	o.BaseOptions.Messages = o.PromptOptions.Messages
	o.GeneralPromptOptions.Tools = o.PromptOptions.Tools
	o.GeneralPromptOptions.ForcedToolsCall = o.PromptOptions.ForcedToolsCall
}

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
