package groq

import (
	"encoding/json"
	"fmt"
)

type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                    `json:"name"`
		Description string                    `json:"description"`
		Parameters  parameters[ParameterBase] `json:"parameters"`
	} `json:"function"`
	Execute func(parameters string) (string, error) `json:"-"`
}

type parameters[T ParameterBase] map[string]T
type ParameterBase struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func NewTool[T any](name string, description string, params parameters[ParameterBase], execute func(T) (string, error)) Tool {
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
		Execute: func(parameters string) (string, error) {
			var unmarshalledParameters T
			if err := json.Unmarshal([]byte(parameters), &unmarshalledParameters); err != nil {
				return "Invalid parameters format", fmt.Errorf("error unmarshalling JSON: %w", err)
			}
			return execute(unmarshalledParameters)
		},
	}
}
