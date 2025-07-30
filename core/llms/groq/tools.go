package groq

import "context"

type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                    `json:"name"`
		Description string                    `json:"description"`
		Parameters  parameters[ParameterBase] `json:"parameters"`
	} `json:"function"`
	Execute executeFunc `json:"-"`
}

type parameters[T ParameterBase] map[string]T
type ParameterBase struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}
type executeFunc func(ctx context.Context, parameters string) (string, error)

func NewTool(name string, description string, params parameters[ParameterBase], execute executeFunc) Tool {
	return Tool{
		Type: "function",
		Function: struct {
			Name        string                    `json:"name"`
			Description string                    `json:"description"`
			Parameters  parameters[ParameterBase] `json:"parameters"`
		}{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
		Execute: execute,
	}
}
