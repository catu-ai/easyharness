package contracts

// NextAction describes one concrete follow-up action that the caller should
// consider after reading a command result.
type NextAction struct {
	// Command is the suggested command line to run next when the next step is
	// best expressed as a harness command.
	Command *string `json:"command"`

	// Description explains the suggested next step in plain language.
	Description string `json:"description"`
}

// ErrorDetail describes one machine-readable validation or execution problem in
// a command result.
type ErrorDetail struct {
	// Path identifies the field, section, or artifact path associated with the
	// error.
	Path string `json:"path"`

	// Message is the human-readable explanation of the problem.
	Message string `json:"message"`
}
