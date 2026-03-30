package contracts

// JSONError is a compact machine-readable error entry returned by commands.
type JSONError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// NextAction is a suggested next step for the operator or controller agent.
type NextAction struct {
	Command     *string `json:"command"`
	Description string  `json:"description"`
}
