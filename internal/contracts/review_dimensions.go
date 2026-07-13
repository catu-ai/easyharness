package contracts

// ReviewDimensionsListResult is the JSON result returned by
// `harness review dimensions list`.
type ReviewDimensionsListResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Dimensions lists the discovered review dimensions available to controller
	// agents.
	Dimensions []ReviewDimensionMetadata `json:"dimensions" easyharness:"no_null"`

	// Warnings lists non-blocking config issues that did not prevent catalog
	// discovery.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewDimensionMetadata describes one dimension in the controller-facing
// catalog.
type ReviewDimensionMetadata struct {
	// Name is the stable review dimension identifier, using lowercase
	// alphanumeric segments separated by single hyphens.
	Name string `json:"name"`

	// Sources preserves the ordered provenance of the resolved guidance. A
	// same-name plan fragment follows its builtin or repo base source.
	Sources []string `json:"sources" easyharness:"no_null"`

	// Path identifies the repo-relative file that supplied the resolved
	// repo-defined or plan-scoped guidance. Built-in-only guidance omits it.
	Path string `json:"path,omitempty"`

	// PlanPath identifies the current plan whose package supplied additive
	// guidance. Guidance with no plan-scoped fragment omits it.
	PlanPath string `json:"plan_path,omitempty"`

	// Description is the concise selection guidance for controller agents.
	Description string `json:"description"`
}
