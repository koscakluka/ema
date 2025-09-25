package llms

import (
	"encoding/json"
	"fmt"
)

type Tool struct {
	Type     string
	Function struct {
		Name        string
		Description string
		Parameters  parameters[ParameterBase]
	}
	Execute func(parameters string) (string, error)
}

type parameters[T ParameterBase] map[string]T
type ParameterBase struct {
	Type        string
	Description string
}

func NewTool[T any](name string, description string, params parameters[ParameterBase], execute func(T) (string, error)) Tool {
	return Tool{
		Type: "function",
		Function: struct {
			Name        string
			Description string
			Parameters  parameters[ParameterBase]
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
