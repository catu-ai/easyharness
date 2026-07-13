package contracts

import "encoding/json"

// ReviewSpec is the JSON input consumed by `harness review start`.
type ReviewSpec struct {
	// Step is the tracked plan step number when the review is step-scoped.
	Step *int `json:"step,omitempty"`

	// Kind is the review kind, such as delta or full.
	Kind string `json:"kind"`

	// AnchorSHA is the controller-chosen git commit anchor for delta review.
	// Delta review expects this to resolve to a real commit.
	AnchorSHA string `json:"anchor_sha,omitempty"`

	// ReviewTitle is the human-readable title for finalize or custom review
	// rounds.
	ReviewTitle string `json:"review_title,omitempty"`

	// Repair identifies an earlier round and findings that this delta review
	// intends to resolve.
	Repair *ReviewRepairReference `json:"repair,omitempty"`

	// Assignments lists the explicit reviewer-owned submission assignments.
	Assignments []ReviewAssignmentSpec `json:"assignments" jsonschema:"minItems=1" easyharness:"no_null"`
}

// ReviewRepairReference identifies the findings targeted by a repair delta.
type ReviewRepairReference struct {
	RoundID    string   `json:"round_id"`
	FindingIDs []string `json:"finding_ids" jsonschema:"minItems=0" easyharness:"no_null"`
}

// ReviewAssignmentSpec is one controller-selected reviewer assignment.
type ReviewAssignmentSpec struct {
	Slot         string           `json:"slot"`
	Role         string           `json:"role"`
	Dimensions   []string         `json:"dimensions" jsonschema:"minItems=1" easyharness:"no_null"`
	Instructions string           `json:"instructions"`
	RiskBrief    *ReviewRiskBrief `json:"risk_brief,omitempty"`
}

// ReviewRiskBrief gives a specialist a concrete risk surface to challenge.
type ReviewRiskBrief struct {
	RiskSurfaces []string `json:"risk_surfaces" jsonschema:"minItems=1" easyharness:"no_null"`
	Invariants   []string `json:"invariants" jsonschema:"minItems=1" easyharness:"no_null"`
	FailureModes []string `json:"failure_modes,omitempty" easyharness:"allow_null"`
}

// ReviewResolvedDimension snapshots one explicitly assigned guidance fragment.
type ReviewResolvedDimension struct {
	Name         string   `json:"name"`
	Sources      []string `json:"sources" jsonschema:"minItems=1" easyharness:"no_null"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions"`
	PlanPath     string   `json:"plan_path,omitempty"`
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
	Slot           string                    `json:"slot"`
	Role           string                    `json:"role"`
	Dimensions     []ReviewResolvedDimension `json:"dimensions" jsonschema:"minItems=1" easyharness:"no_null"`
	Instructions   string                    `json:"instructions"`
	RiskBrief      *ReviewRiskBrief          `json:"risk_brief,omitempty"`
	SubmissionPath string                    `json:"submission_path"`
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

	// ExtraFields preserves reviewer-owned progressive worklog fields that are
	// not part of the canonical aggregate contract.
	ExtraFields map[string]json.RawMessage `json:"-"`
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

	// ExtraFields preserves reviewer-owned progressive worklog fields that are
	// not part of the canonical aggregate contract.
	ExtraFields map[string]json.RawMessage `json:"-"`
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

func (s ReviewSubmissionInput) MarshalJSON() ([]byte, error) {
	payload := reviewSubmissionInputPayload{
		Summary:     s.Summary,
		Resolutions: s.Resolutions,
		Findings:    s.Findings,
	}
	return marshalWithExtraFields(payload, s.ExtraFields)
}

func (s *ReviewSubmissionInput) UnmarshalJSON(data []byte) error {
	var payload reviewSubmissionInputPayload
	extraFields, err := unmarshalWithExtraFields(data, &payload, reviewSubmissionInputKnownFields)
	if err != nil {
		return err
	}
	s.Summary = payload.Summary
	s.Resolutions = payload.Resolutions
	s.Findings = payload.Findings
	for key := range reviewSubmissionInputIgnoredExtraFields {
		delete(extraFields, key)
	}
	if len(extraFields) == 0 {
		extraFields = nil
	}
	s.ExtraFields = extraFields
	return nil
}

func (s ReviewSubmission) MarshalJSON() ([]byte, error) {
	payload := reviewSubmissionPayload{
		RoundID:     s.RoundID,
		Slot:        s.Slot,
		Role:        s.Role,
		By:          s.By,
		SubmittedAt: s.SubmittedAt,
		Summary:     s.Summary,
		Resolutions: s.Resolutions,
		Findings:    s.Findings,
	}
	return marshalWithExtraFields(payload, s.ExtraFields)
}

func (s *ReviewSubmission) UnmarshalJSON(data []byte) error {
	var payload reviewSubmissionPayload
	extraFields, err := unmarshalWithExtraFields(data, &payload, reviewSubmissionKnownFields)
	if err != nil {
		return err
	}
	s.RoundID = payload.RoundID
	s.Slot = payload.Slot
	s.Role = payload.Role
	s.By = payload.By
	s.SubmittedAt = payload.SubmittedAt
	s.Summary = payload.Summary
	s.Resolutions = payload.Resolutions
	s.Findings = payload.Findings
	s.ExtraFields = extraFields
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

	// Assignments lists the materialized reviewer assignments created for the round.
	Assignments []ReviewAssignment `json:"assignments"`
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

	// Slot is the stable reviewer slot identifier.
	Slot string `json:"slot"`

	// SubmissionPath is the path to the created submission artifact.
	SubmissionPath string `json:"submission_path"`
}

// ReviewAggregateResult is the JSON result returned by
// `harness review aggregate`.
type ReviewAggregateResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the updated aggregate artifacts.
	Artifacts *ReviewAggregateArtifacts `json:"artifacts,omitempty"`

	// Review is the aggregate decision payload when aggregation succeeded.
	Review *ReviewAggregate `json:"review,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewAggregateArtifacts lists the artifacts touched by
// `harness review aggregate`.
type ReviewAggregateArtifacts struct {
	// ProjectRoot is the repository root that anchors surfaced repo-facing
	// paths.
	ProjectRoot string `json:"project_root"`

	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`
}

type reviewSubmissionInputPayload struct {
	Summary     string                    `json:"summary"`
	Resolutions []ReviewFindingResolution `json:"resolutions,omitempty"`
	Findings    []ReviewFinding           `json:"findings,omitempty"`
}

type reviewSubmissionPayload struct {
	RoundID     string                    `json:"round_id"`
	Slot        string                    `json:"slot"`
	Role        string                    `json:"role"`
	By          string                    `json:"by,omitempty"`
	SubmittedAt string                    `json:"submitted_at,omitempty"`
	Summary     string                    `json:"summary,omitempty"`
	Resolutions []ReviewFindingResolution `json:"resolutions,omitempty"`
	Findings    []ReviewFinding           `json:"findings,omitempty"`
}

var reviewSubmissionInputKnownFields = map[string]bool{
	"summary":     true,
	"resolutions": true,
	"findings":    true,
}

var reviewSubmissionInputIgnoredExtraFields = map[string]bool{
	"round_id":     true,
	"slot":         true,
	"role":         true,
	"by":           true,
	"submitted_at": true,
}

var reviewSubmissionKnownFields = map[string]bool{
	"round_id":     true,
	"slot":         true,
	"role":         true,
	"by":           true,
	"submitted_at": true,
	"summary":      true,
	"resolutions":  true,
	"findings":     true,
}

func unmarshalWithExtraFields(data []byte, payload any, knownFields map[string]bool) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, payload); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	extraFields := make(map[string]json.RawMessage)
	for key, value := range raw {
		if knownFields[key] {
			continue
		}
		extraFields[key] = append(json.RawMessage(nil), value...)
	}
	if len(extraFields) == 0 {
		return nil, nil
	}
	return extraFields, nil
}

func marshalWithExtraFields(payload any, extraFields map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(extraFields) == 0 {
		return data, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for key, value := range extraFields {
		raw[key] = append(json.RawMessage(nil), value...)
	}
	return json.Marshal(raw)
}
