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

type StructuredPromptOptions struct {
	BaseOptions
	PromptOptions
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

type StructuredPromptOption interface {
	ApplyToStructured(*StructuredPromptOptions)
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

func (f PromptOption) ApplyToStructured(o *StructuredPromptOptions) {
	o.PromptOptions.Messages = o.BaseOptions.Messages
	f(&o.PromptOptions)
	o.BaseOptions.Messages = o.PromptOptions.Messages
}

// WithStream is a PromptOption that sets the stream callback for the prompt.
//
// Deprecated: Use specialized streaming method instead of general one
func WithStream(stream func(string)) PromptOption {
	return func(opts *PromptOptions) {
		opts.Stream = stream
	}
}

// WithSystemPrompt is a PromptOption that sets the system prompt for the
// prompt.
// Repeating this option will overwrite the previous system prompt.
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

// WithMessages is a PromptOption that adds passed messages to the prompt.
// Repeating this option will sequentially add more messages.
func WithMessages(messages ...Message) PromptOption {
	return func(opts *PromptOptions) {
		opts.Messages = append(opts.Messages, messages...)
	}
}

// WithTools is a PromptOption that adds tools to the prompt
//
// This option does nothing for structured prompts, it is depricated for use
// there and will be disabled in the future
func WithTools(tools ...Tool) PromptOption {
	return func(opts *PromptOptions) {
		opts.Tools = append(opts.Tools, tools...)
	}
}

// WithForcedTools is a PromptOption that forces the use of tools in the prompt.
// Note that any tool that is available can be used, not just the ones passed
// into this option.
//
// This option does nothing for structured prompts, it is depricated for use
// there and will be disabled in the future
func WithForcedTools(tools ...Tool) PromptOption {
	return func(opts *PromptOptions) {
		opts.Tools = tools
	}
}
