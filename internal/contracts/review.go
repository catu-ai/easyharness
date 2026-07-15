package contracts

import "encoding/json"

// ReviewStartOptions contains the only controller choice accepted by
// `harness review start`. The ordinary path infers a full root or linked delta
// from current finalize coverage; ForceFull deliberately resets that coverage.
type ReviewStartOptions struct {
	ForceFull bool
}

// ReviewRepairReference identifies the findings targeted by a repair delta.
type ReviewRepairReference struct {
	RoundID    string   `json:"round_id"`
	FindingIDs []string `json:"finding_ids" jsonschema:"minItems=0" easyharness:"no_null"`
}

// ReviewManifest is the command-owned review manifest artifact for one review
// round.
type ReviewManifest struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// AnchorSHA is the controller-chosen git commit anchor recorded for delta
	// review when the round uses one.
	AnchorSHA string `json:"anchor_sha,omitempty"`

	// ReviewedHeadSHA is the command-captured immutable candidate head reviewed
	// by this round.
	ReviewedHeadSHA string `json:"reviewed_head_sha"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// ReviewFocus snapshots the plan's reviewer-specific focus for this round.
	ReviewFocus string `json:"review_focus,omitempty"`

	// Repair identifies the prior round and findings targeted by this delta.
	Repair *ReviewRepairReference `json:"repair,omitempty"`

	// PlanPath is the tracked or archived plan path associated with the round.
	PlanPath string `json:"plan_path"`

	// PlanStem is the durable plan stem associated with the round.
	PlanStem string `json:"plan_stem"`

	// CreatedAt is the round creation timestamp.
	CreatedAt string `json:"created_at"`

	// Assignments lists the materialized reviewer assignments for the round.
	Assignments []ReviewAssignment `json:"assignments"`

	// LedgerPath is the path to the round ledger artifact.
	LedgerPath string `json:"ledger_path"`

	// Aggregate is the path to the round aggregate artifact.
	Aggregate string `json:"aggregate_path"`

	// Submissions is the path to the round submissions directory.
	Submissions string `json:"submissions_dir"`
}

// ReviewAssignment describes one surfaced reviewer assignment.
type ReviewAssignment struct {
	Slot           string `json:"slot"`
	Role           string `json:"role"`
	Instructions   string `json:"instructions"`
	ReviewFocus    string `json:"review_focus,omitempty"`
	SubmissionPath string `json:"submission_path"`
}

// ReviewHandle is the sole integrated reviewer handoff returned by
// `harness review start`. Reviewer topology is fixed rather than selected by
// the controller.
type ReviewHandle struct {
	Instructions   string `json:"instructions"`
	ReviewFocus    string `json:"review_focus,omitempty"`
	SubmissionPath string `json:"submission_path"`
}

// ReviewLedger is the command-owned ledger artifact tracking submission status
// for a review round.
type ReviewLedger struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// UpdatedAt is the timestamp of the most recent ledger update.
	UpdatedAt string `json:"updated_at"`

	// Assignments lists the current state of every reviewer assignment.
	Assignments []ReviewLedgerAssignment `json:"assignments"`
}

// ReviewLedgerAssignment records one reviewer assignment in the ledger.
type ReviewLedgerAssignment struct {
	Slot string `json:"slot"`
	Role string `json:"role"`

	// Status is the current submission status for the slot.
	Status string `json:"status"`

	// SubmissionPath is the path where the slot submission should exist.
	SubmissionPath string `json:"submission_path"`

	// SubmittedAt is the submission timestamp when the slot has been submitted.
	SubmittedAt string `json:"submitted_at,omitempty"`

	// AbortedAt is the timestamp when the unfinished round was explicitly
	// abandoned before submission.
	AbortedAt string `json:"aborted_at,omitempty"`
}

// ReviewSubmissionInput is the JSON input consumed by `harness review submit`.
type ReviewSubmissionInput struct {
	// Summary is the reviewer's concise overall assessment.
	Summary string `json:"summary"`

	// Resolutions records explicit reviewer verdicts for referenced repair
	// findings.
	Resolutions []ReviewFindingResolution `json:"resolutions,omitempty" easyharness:"allow_null"`

	// Findings lists the review findings for the slot.
	Findings []ReviewFinding `json:"findings,omitempty" easyharness:"allow_null"`
}

// ReviewSubmission is the command-owned submission artifact for one reviewer
// slot.
type ReviewSubmission struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Slot is the stable slot identifier.
	Slot string `json:"slot"`

	// Role is the assignment role copied from the round manifest.
	Role string `json:"role"`

	// By is the reviewer-provided identity label for the submitted slot.
	By string `json:"by,omitempty"`

	// SubmittedAt is the submission timestamp.
	SubmittedAt string `json:"submitted_at,omitempty"`

	// Summary is the reviewer's concise overall assessment.
	Summary string `json:"summary,omitempty"`

	Resolutions []ReviewFindingResolution `json:"resolutions,omitempty"`

	// Findings lists the review findings for the slot.
	Findings []ReviewFinding `json:"findings,omitempty"`
}

// ReviewFinding is one review finding in a submission or aggregate.
type ReviewFinding struct {
	// Area is the actionable defect area within the reviewed candidate.
	Area string `json:"area"`

	// Severity is the finding severity label.
	Severity string `json:"severity"`

	// Title is the short human-readable title of the finding.
	Title string `json:"title"`

	// Details is the full review finding explanation.
	Details string `json:"details"`

	// Locations optionally lists lightweight repo-relative source anchors for
	// the finding, such as "path/to/file.go", "path/to/file.go#L123", or
	// "path/to/file.go#L1-L3".
	Locations []string `json:"locations,omitempty"`

	// HasLocations records whether the payload explicitly included the optional
	// locations field so empty arrays can round-trip without being collapsed
	// into omission.
	HasLocations bool `json:"-"`
}

// ReviewFindingResolution records whether one referenced finding is closed.
type ReviewFindingResolution struct {
	FindingID string `json:"finding_id"`
	Status    string `json:"status"`
	Details   string `json:"details"`
}

// ReviewAggregate is the command-owned aggregate artifact for a completed
// review round.
type ReviewAggregate struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// ReviewedHeadSHA repeats the manifest candidate boundary accepted at
	// aggregation time.
	ReviewedHeadSHA string `json:"reviewed_head_sha"`

	Repair *ReviewRepairReference `json:"repair,omitempty"`

	// Decision is the aggregate review decision for the round.
	Decision string `json:"decision"`

	// BlockingFindings lists the findings that currently block progression.
	BlockingFindings []ReviewAggregateFinding `json:"blocking_findings"`

	// NonBlockingFindings lists the findings that were recorded without blocking
	// progression.
	NonBlockingFindings []ReviewAggregateFinding `json:"non_blocking_findings"`

	ResolvedFindingIDs []string `json:"resolved_finding_ids" easyharness:"no_null"`

	UnresolvedFindingIDs []string `json:"unresolved_finding_ids" easyharness:"no_null"`

	// UnresolvedBlockingFindings is the cumulative blocking set at this chain
	// tip, including inherited findings not resolved by this round.
	UnresolvedBlockingFindings []ReviewAggregateFinding `json:"unresolved_blocking_findings" easyharness:"no_null"`

	// AggregatedAt is the aggregate timestamp.
	AggregatedAt string `json:"aggregated_at"`
}

// ReviewAggregateFinding is one aggregate finding annotated with its stable
// identity and reviewer-assignment provenance.
type ReviewAggregateFinding struct {
	FindingID string `json:"finding_id"`

	// Slot is the stable reviewer slot identifier.
	Slot string `json:"slot"`

	Role string `json:"role"`

	Area string `json:"area"`

	// Severity is the finding severity label.
	Severity string `json:"severity"`

	// Title is the short human-readable title of the finding.
	Title string `json:"title"`

	// Details is the full review finding explanation.
	Details string `json:"details"`

	// Locations optionally lists lightweight repo-relative source anchors for
	// the finding, such as "path/to/file.go", "path/to/file.go#L123", or
	// "path/to/file.go#L1-L3".
	Locations []string `json:"locations,omitempty"`

	// HasLocations records whether the payload explicitly included the optional
	// locations field so empty arrays can round-trip without being collapsed
	// into omission.
	HasLocations bool `json:"-"`
}

func (f ReviewFinding) MarshalJSON() ([]byte, error) {
	type payload struct {
		Area      string   `json:"area"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations,omitempty"`
	}
	if f.HasLocations {
		type payloadWithLocations struct {
			Area      string   `json:"area"`
			Severity  string   `json:"severity"`
			Title     string   `json:"title"`
			Details   string   `json:"details"`
			Locations []string `json:"locations"`
		}
		return json.Marshal(payloadWithLocations{
			Area:      f.Area,
			Severity:  f.Severity,
			Title:     f.Title,
			Details:   f.Details,
			Locations: f.Locations,
		})
	}
	return json.Marshal(payload{
		Area:      f.Area,
		Severity:  f.Severity,
		Title:     f.Title,
		Details:   f.Details,
		Locations: f.Locations,
	})
}

func (f *ReviewFinding) UnmarshalJSON(data []byte) error {
	type payload struct {
		Area      string   `json:"area"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations"`
	}
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.Severity = decoded.Severity
	f.Area = decoded.Area
	f.Title = decoded.Title
	f.Details = decoded.Details
	f.Locations = decoded.Locations
	_, f.HasLocations = raw["locations"]
	return nil
}

func (f ReviewAggregateFinding) MarshalJSON() ([]byte, error) {
	type payload struct {
		FindingID string   `json:"finding_id"`
		Slot      string   `json:"slot"`
		Role      string   `json:"role"`
		Area      string   `json:"area"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations,omitempty"`
	}
	if f.HasLocations {
		type payloadWithLocations struct {
			FindingID string   `json:"finding_id"`
			Slot      string   `json:"slot"`
			Role      string   `json:"role"`
			Area      string   `json:"area"`
			Severity  string   `json:"severity"`
			Title     string   `json:"title"`
			Details   string   `json:"details"`
			Locations []string `json:"locations"`
		}
		return json.Marshal(payloadWithLocations{
			FindingID: f.FindingID,
			Slot:      f.Slot,
			Role:      f.Role,
			Area:      f.Area,
			Severity:  f.Severity,
			Title:     f.Title,
			Details:   f.Details,
			Locations: f.Locations,
		})
	}
	return json.Marshal(payload{
		FindingID: f.FindingID,
		Slot:      f.Slot,
		Role:      f.Role,
		Area:      f.Area,
		Severity:  f.Severity,
		Title:     f.Title,
		Details:   f.Details,
		Locations: f.Locations,
	})
}

func (f *ReviewAggregateFinding) UnmarshalJSON(data []byte) error {
	type payload struct {
		FindingID string   `json:"finding_id"`
		Slot      string   `json:"slot"`
		Role      string   `json:"role"`
		Area      string   `json:"area"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations"`
	}
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.FindingID = decoded.FindingID
	f.Slot = decoded.Slot
	f.Role = decoded.Role
	f.Area = decoded.Area
	f.Severity = decoded.Severity
	f.Title = decoded.Title
	f.Details = decoded.Details
	f.Locations = decoded.Locations
	_, f.HasLocations = raw["locations"]
	return nil
}

// ReviewStartResult is the JSON result returned by `harness review start`.
type ReviewStartResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the created review artifacts for the round.
	Artifacts *ReviewStartArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewStartArtifacts lists the review artifacts created by
// `harness review start`.
type ReviewStartArtifacts struct {
	// ProjectRoot is the repository root that anchors surfaced repo-facing
	// paths.
	ProjectRoot string `json:"project_root"`

	// PlanPath is the current plan path associated with the review round.
	PlanPath string `json:"plan_path"`

	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// ReviewedHeadSHA is the clean committed candidate boundary captured by the
	// command and recorded in the review manifest.
	ReviewedHeadSHA string `json:"reviewed_head_sha"`

	// Reviewer is the sole integrated reviewer handle for the round.
	Reviewer *ReviewHandle `json:"reviewer,omitempty"`
}

// ReviewSubmitResult is the JSON result returned by `harness review submit`.
type ReviewSubmitResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the created submission artifacts.
	Artifacts *ReviewSubmitArtifacts `json:"artifacts,omitempty"`

	// Review is the completed review decision derived from the sole submission.
	Review *ReviewAggregate `json:"review,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewSubmitArtifacts lists the artifacts touched by `harness review submit`.
type ReviewSubmitArtifacts struct {
	// ProjectRoot is the repository root that anchors surfaced repo-facing
	// paths.
	ProjectRoot string `json:"project_root"`

	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// SubmissionPath is the path to the created submission artifact.
	SubmissionPath string `json:"submission_path"`
}

// ReviewAbortResult is the JSON result returned by `harness review abort`.
type ReviewAbortResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts identifies the preserved round and updated ledger.
	Artifacts *ReviewAbortArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewAbortArtifacts identifies the unfinished round preserved by abort.
type ReviewAbortArtifacts struct {
	// ProjectRoot is the repository root that anchors surfaced repo-facing
	// paths.
	ProjectRoot string `json:"project_root"`

	// PlanPath is the current plan associated with the review round.
	PlanPath string `json:"plan_path"`

	// RoundID is the stable identifier of the aborted round.
	RoundID string `json:"round_id"`

	// LedgerPath is the updated round ledger that records aborted assignments.
	LedgerPath string `json:"ledger_path"`
}
