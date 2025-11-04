package openai

import (
	"github.com/jinzhu/copier"
	"github.com/koscakluka/ema/core/llms"
)

type openAITool struct {
	Type        string                                  `json:"type"`
	Name        string                                  `json:"name"`
	Description string                                  `json:"description"`
	Parameters  parametersWrapper                       `json:"parameters"`
	Execute     func(parameters string) (string, error) `json:"-"`
}

type parameters[T parameterBase] map[string]T
type parameterBase struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type parametersWrapper struct {
	Properties map[string]parameterBase `json:"properties"`
	Type       string                   `json:"type"`
}

func toOpenAITools(tools []llms.Tool) []openAITool {
	openAITools := []openAITool{}
	for _, tool := range tools {
		openAITool := openAITool{
			Type:        tool.Type,
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  parametersWrapper{Type: "object"},
			Execute:     tool.Execute,
		}
		copier.Copy(&openAITool.Parameters.Properties, tool.Function.Parameters)
		openAITools = append(openAITools, openAITool)
	}
	return openAITools
}
