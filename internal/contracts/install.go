package contracts

// InstallResult is the JSON result returned by `harness install`.
type InstallResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Mode indicates whether the command applied changes or only planned them.
	Mode string `json:"mode"`

	// Scope is the resolved install scope.
	Scope string `json:"scope"`

	// Actions lists the planned or executed file-level install actions.
	Actions []InstallAction `json:"actions"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented install preparation or writes.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// InstallAction describes one file-level change planned or applied by
// `harness install`.
type InstallAction struct {
	// Path is the repository-relative file path touched by the action.
	Path string `json:"path"`

	// Kind is the coarse action kind such as create, update, or noop.
	Kind string `json:"kind"`

	// Details is the human-readable explanation of the planned change.
	Details string `json:"details"`
}
