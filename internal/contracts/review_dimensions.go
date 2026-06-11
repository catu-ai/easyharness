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

	// Source identifies where the dimension definition came from.
	Source string `json:"source"`

	// Description is the concise selection guidance for controller agents.
	Description string `json:"description"`
}
