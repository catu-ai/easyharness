package contracts

// ReviewResult is the read-only UI resource returned by `/api/review`.
type ReviewResult struct {
	// OK reports whether review loading succeeded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded review
	// rounds.
	Summary string `json:"summary"`

	// Rounds lists the discovered review rounds for the current plan.
	Rounds []ReviewRoundView `json:"rounds"`

	// Warnings lists non-fatal degraded-state notes for the overall review
	// resource.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented review loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewRoundView is one read-only UI representation of a review round.
type ReviewRoundView struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind,omitempty"`

	// AnchorSHA is the delta-review anchor commit when one is available.
	AnchorSHA string `json:"anchor_sha,omitempty"`

	// ReviewedHeadSHA is the immutable candidate commit captured for this round.
	ReviewedHeadSHA string `json:"reviewed_head_sha,omitempty"`

	// RepairsRoundID is the direct parent coverage round for a repair delta.
	RepairsRoundID string `json:"repairs_round_id,omitempty"`

	// RepairFindingIDs lists the parent findings explicitly targeted by this
	// repair delta.
	RepairFindingIDs []string `json:"repair_finding_ids,omitempty"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision,omitempty"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// Status is the stable UI status label for the round.
	Status string `json:"status,omitempty"`

	// StatusSummary is the concise explanation of the current round status.
	StatusSummary string `json:"status_summary,omitempty"`

	// Decision is the completed review decision when one is known.
	Decision string `json:"decision,omitempty"`

	// CreatedAt is the round creation timestamp when known.
	CreatedAt string `json:"created_at,omitempty"`

	// UpdatedAt is the ledger update timestamp when known.
	UpdatedAt string `json:"updated_at,omitempty"`

	// DecidedAt is the completed review timestamp when known.
	DecidedAt string `json:"decided_at,omitempty"`

	// IsActive reports whether local state still points at this round as active.
	IsActive bool `json:"is_active,omitempty"`

	// Reviewer is the independent integrated reviewer shown for this round.
	Reviewer *ReviewReviewerView `json:"reviewer,omitempty"`

	// BlockingFindings lists the cumulative unresolved blocking findings at this
	// coverage-chain tip, not only findings newly raised by this round.
	BlockingFindings []ReviewFindingView `json:"blocking_findings,omitempty"`

	// NonBlockingFindings lists non-blocking findings when they exist.
	NonBlockingFindings []ReviewFindingView `json:"non_blocking_findings,omitempty"`

	// ResolvedFindingIDs lists repair findings closed by this round.
	ResolvedFindingIDs []string `json:"resolved_finding_ids,omitempty"`

	// UnresolvedFindingIDs lists all blocking findings still open at this
	// coverage-chain tip.
	UnresolvedFindingIDs []string `json:"unresolved_finding_ids,omitempty"`

	// CoverageStatus is clean, blocked, or pending according to the completed
	// coverage-chain state surfaced for this round.
	CoverageStatus string `json:"coverage_status,omitempty"`

	// Warnings lists non-fatal degraded-state notes for the round.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewReviewerView is the independent reviewer result surfaced for a round.
// Internal reviewer orchestration is intentionally not part of the UI resource.
type ReviewReviewerView struct {
	// Name is the reviewer-provided identity label when available.
	Name string `json:"name,omitempty"`

	// Status is pending, submitted, or aborted.
	Status string `json:"status,omitempty"`

	// SubmittedAt is the submission timestamp when known.
	SubmittedAt string `json:"submitted_at,omitempty"`

	// AbortedAt is the timestamp when the unfinished round was explicitly
	// aborted.
	AbortedAt string `json:"aborted_at,omitempty"`

	// Summary is the reviewer's concise assessment.
	Summary string `json:"summary,omitempty"`

	// Warnings lists non-fatal degraded-state notes for the reviewer result.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewFindingView is one decision finding without internal orchestration
// metadata.
type ReviewFindingView struct {
	FindingID string   `json:"finding_id,omitempty"`
	Area      string   `json:"area"`
	Severity  string   `json:"severity"`
	Title     string   `json:"title"`
	Details   string   `json:"details"`
	Locations []string `json:"locations,omitempty"`
}
