package groq

type Tool struct {
	Type     string                                  `json:"type"`
	Function FunctionDescription                     `json:"function"`
	Execute  func(parameters string) (string, error) `json:"-"`
}

type FunctionDescription struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Parameters  parameters[ParameterBase] `json:"parameters"`
}

type parameters[T ParameterBase] map[string]T
type ParameterBase struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}
