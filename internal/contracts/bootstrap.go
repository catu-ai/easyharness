package contracts

// BootstrapResult is the JSON result returned by bootstrap resource commands
// such as `harness init`, `harness skills install`, or
// `harness instructions uninstall`.
type BootstrapResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Mode indicates whether the command applied changes or only planned them.
	Mode string `json:"mode"`

	// Resource identifies the managed bootstrap surface.
	Resource string `json:"resource"`

	// Operation identifies the resource action such as install or uninstall.
	Operation string `json:"operation"`

	// Scope is the resolved resource scope such as repo or user.
	Scope string `json:"scope"`

	// Agent is the resolved agent profile name used for defaults.
	Agent string `json:"agent"`

	// Actions lists the planned or executed path-level actions.
	Actions []BootstrapAction `json:"actions"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented planning or writes.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// BootstrapAction describes one path-level change planned or applied by a
// bootstrap resource command.
type BootstrapAction struct {
	// Path is the file or directory path touched by the action.
	Path string `json:"path"`

	// Kind is the coarse action kind such as create, update, delete, or noop.
	Kind string `json:"kind"`

	// Details is the human-readable explanation of the planned change.
	Details string `json:"details"`
}
