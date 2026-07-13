package contracts

import "encoding/json"

// ReviewResult is the read-only UI resource returned by `/api/review`.
type ReviewResult struct {
	// OK reports whether review loading succeeded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded review
	// rounds.
	Summary string `json:"summary"`

	// Artifacts points to the current-plan review artifact paths used to build
	// this response.
	Artifacts *ReviewArtifacts `json:"artifacts,omitempty"`

	// Rounds lists the discovered review rounds for the current plan.
	Rounds []ReviewRoundView `json:"rounds"`

	// Warnings lists non-fatal degraded-state notes for the overall review
	// resource.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented review loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewArtifacts points to the current plan review sources.
type ReviewArtifacts struct {
	// PlanPath is the current plan path associated with the review data.
	PlanPath string `json:"plan_path,omitempty"`

	// ActiveRoundID is the currently active review round when one exists in
	// local state.
	ActiveRoundID string `json:"active_round_id,omitempty"`
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

	// Decision is the aggregate decision when one is known.
	Decision string `json:"decision,omitempty"`

	// CreatedAt is the round creation timestamp when known.
	CreatedAt string `json:"created_at,omitempty"`

	// UpdatedAt is the ledger update timestamp when known.
	UpdatedAt string `json:"updated_at,omitempty"`

	// AggregatedAt is the aggregate timestamp when known.
	AggregatedAt string `json:"aggregated_at,omitempty"`

	// IsActive reports whether local state still points at this round as active.
	IsActive bool `json:"is_active,omitempty"`

	// TotalAssignments is the number of reviewer assignments in the round.
	TotalAssignments int `json:"total_assignments,omitempty"`

	// SubmittedAssignments is the number of assignments with a submission.
	SubmittedAssignments int `json:"submitted_assignments,omitempty"`

	// PendingAssignments is the number of assignments still waiting on a
	// submission.
	PendingAssignments int `json:"pending_assignments,omitempty"`

	// Reviewers lists the reviewer-centric assignment views for the round.
	Reviewers []ReviewAssignmentView `json:"reviewers,omitempty"`

	// BlockingFindings lists aggregate blocking findings when they exist.
	BlockingFindings []ReviewAggregateFinding `json:"blocking_findings,omitempty"`

	// NonBlockingFindings lists aggregate non-blocking findings when they
	// exist.
	NonBlockingFindings []ReviewAggregateFinding `json:"non_blocking_findings,omitempty"`

	// Artifacts lists supporting raw review artifacts for the round.
	Artifacts []ReviewArtifactView `json:"artifacts,omitempty"`

	// Warnings lists non-fatal degraded-state notes for the round.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewAssignmentView is one reviewer-centric view of a materialized
// assignment plus any submission returned for it.
type ReviewAssignmentView struct {
	// Slot is the stable assignment identifier.
	Slot string `json:"slot"`

	// Role is integrated or specialist.
	Role string `json:"role"`

	// Dimensions are the snapshotted guidance fragments assigned to the
	// reviewer.
	Dimensions []ReviewResolvedDimension `json:"dimensions,omitempty"`

	// RiskBrief is present only for specialist assignments.
	RiskBrief *ReviewRiskBrief `json:"risk_brief,omitempty"`

	// Instructions is the explicit reviewer handoff for this slot when it is
	// available in surfaced review data.
	Instructions string `json:"instructions,omitempty"`

	// Status is the current submission status label for the slot.
	Status string `json:"status,omitempty"`

	// SubmissionPath is the path to the submission artifact for this slot.
	SubmissionPath string `json:"submission_path,omitempty"`

	// SubmittedAt is the submission timestamp when one is known.
	SubmittedAt string `json:"submitted_at,omitempty"`

	// Summary is the reviewer's concise overall assessment when one was
	// submitted.
	Summary string `json:"summary,omitempty"`

	// Resolutions records explicit repair-finding verdicts from this reviewer.
	Resolutions []ReviewFindingResolution `json:"resolutions,omitempty"`

	// Findings lists the slot-level review findings when one was submitted.
	Findings []ReviewFinding `json:"findings,omitempty"`

	// Worklog contains normalized progressive reviewer-worklog fields derived
	// from the submission payload when available.
	Worklog *ReviewWorklogView `json:"worklog,omitempty"`

	// RawSubmission preserves the raw reviewer submission payload for secondary
	// UI inspection.
	RawSubmission json.RawMessage `json:"raw_submission,omitempty"`

	// Warnings lists non-fatal degraded-state notes for this slot.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewWorklogView contains normalized reviewer-progress fields derived from
// the richer submission payload.
type ReviewWorklogView struct {
	// ReviewKind is the reviewer-reported kind from the coverage payload when
	// available.
	ReviewKind string `json:"review_kind,omitempty"`

	// AnchorSHA is the reviewer-reported anchor SHA from the coverage payload
	// when available.
	AnchorSHA string `json:"anchor_sha,omitempty"`

	// FullPlanRead reports whether the reviewer marked the full active plan as
	// read.
	FullPlanRead *bool `json:"full_plan_read,omitempty"`

	// CheckedAreas lists the files, checkpoints, or areas the reviewer says it
	// already covered.
	CheckedAreas []string `json:"checked_areas,omitempty"`

	// OpenQuestions lists unresolved review trails still in flight.
	OpenQuestions []string `json:"open_questions,omitempty"`

	// CandidateFindings lists provisional findings captured during review before
	// they were finalized into the canonical findings payload.
	CandidateFindings []string `json:"candidate_findings,omitempty"`
}

// ReviewArtifactView is one raw supporting artifact view for a review round.
type ReviewArtifactView struct {
	// Label is the stable display label for the artifact.
	Label string `json:"label"`

	// Path is the repo-facing artifact path when one exists.
	Path string `json:"path,omitempty"`

	// Status reports whether the artifact is available, missing, or invalid.
	Status string `json:"status,omitempty"`

	// Summary is the concise artifact-state explanation.
	Summary string `json:"summary,omitempty"`

	// ContentType reports how Content should be rendered in the UI resource.
	ContentType string `json:"content_type,omitempty"`

	// Content is the resolved artifact file payload for UI tabs when readable.
	Content json.RawMessage `json:"content,omitempty"`
}
