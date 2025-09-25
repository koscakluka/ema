package groq

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
