package contracts

// CurrentPlanFile is the worktree-level pointer file under
// `.local/harness/current-plan.json`.
type CurrentPlanFile struct {
	// PlanPath is the current active or archived plan path when work is in
	// flight.
	PlanPath string `json:"plan_path,omitempty"`

	// LastLandedPlanPath is the most recent landed plan path when the worktree is
	// otherwise idle.
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`

	// LastLandedAt is the timestamp of the most recent landed plan.
	LastLandedAt string `json:"last_landed_at,omitempty"`
}

// LocalStateFile is the plan-local runtime state cache under
// `.local/harness/plans/<plan-stem>/state.json`.
type LocalStateFile struct {
	// ExecutionStartedAt is the execution-start timestamp for the plan.
	ExecutionStartedAt string `json:"execution_started_at,omitempty"`

	// CurrentNode is the cached canonical workflow node when one has been
	// resolved.
	CurrentNode string `json:"current_node,omitempty"`

	// PlanPath is the current tracked or archived plan path associated with this
	// state file.
	PlanPath string `json:"plan_path,omitempty"`

	// PlanStem is the durable plan stem associated with this state file.
	PlanStem string `json:"plan_stem,omitempty"`

	// Revision is the current plan-local revision number.
	Revision int `json:"revision,omitempty"`

	// Reopen records the active reopen repair state when one exists.
	Reopen *ReopenState `json:"reopen,omitempty"`

	// ActiveReviewRound records the current active review round when review is in
	// flight.
	ActiveReviewRound *ReviewRoundState `json:"active_review_round,omitempty"`

	// LatestEvidence records the latest evidence pointers tracked in the state
	// file.
	LatestEvidence *EvidenceSetState `json:"latest_evidence,omitempty"`

	// Land records the current land state when merge cleanup is in flight.
	Land *LandState `json:"land,omitempty"`

	// LatestCI is the transitional cached CI state retained while status still
	// reads legacy evidence hints directly from state.json.
	LatestCI *LegacyCIState `json:"latest_ci,omitempty"`

	// Sync is the transitional cached sync state retained while status still
	// reads legacy hints directly from state.json.
	Sync *LegacySyncState `json:"sync,omitempty"`

	// LatestPublish is the transitional cached publish state retained while
	// status still reads legacy hints directly from state.json.
	LatestPublish *LegacyPublishState `json:"latest_publish,omitempty"`
}

// ReopenState records the active reopen repair state.
type ReopenState struct {
	// Mode is the active reopen mode.
	Mode string `json:"mode"`

	// ReopenedAt is the reopen timestamp.
	ReopenedAt string `json:"reopened_at,omitempty"`

	// BaseStepCount is the number of plan steps that existed before the reopen.
	BaseStepCount int `json:"base_step_count,omitempty"`
}

// ReviewRoundState records the current active review round in the local state
// file.
type ReviewRoundState struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision,omitempty"`

	// Aggregated reports whether the round has already been aggregated.
	Aggregated bool `json:"aggregated"`

	// Decision is the aggregate review decision when one is known.
	Decision string `json:"decision,omitempty"`
}

// EvidenceSetState records the latest evidence pointers tracked in the local
// state file.
type EvidenceSetState struct {
	// CI points to the latest CI evidence record when one exists.
	CI *EvidencePointerState `json:"ci,omitempty"`

	// Publish points to the latest publish evidence record when one exists.
	Publish *EvidencePointerState `json:"publish,omitempty"`

	// Sync points to the latest sync evidence record when one exists.
	Sync *EvidencePointerState `json:"sync,omitempty"`
}

// EvidencePointerState points to one evidence record from the local state file.
type EvidencePointerState struct {
	// Kind is the evidence kind.
	Kind string `json:"kind"`

	// RecordID is the stable identifier of the evidence record.
	RecordID string `json:"record_id"`

	// Path is the path to the evidence record artifact.
	Path string `json:"path"`

	// RecordedAt is the evidence record timestamp when one is tracked.
	RecordedAt string `json:"recorded_at,omitempty"`
}

// LandState records the current land state in the local state file.
type LandState struct {
	// PRURL is the pull request URL recorded for the land phase.
	PRURL string `json:"pr_url,omitempty"`

	// Commit is the merge commit or landed commit recorded for the land phase.
	Commit string `json:"commit,omitempty"`

	// LandedAt is the timestamp when the land command recorded merge completion.
	LandedAt string `json:"landed_at,omitempty"`

	// CompletedAt is the timestamp when land cleanup completed.
	CompletedAt string `json:"completed_at,omitempty"`
}

// LegacyCIState is the transitional CI cache retained in the local state file
// while status still reads legacy evidence hints directly.
type LegacyCIState struct {
	// SnapshotID is the legacy CI snapshot identifier.
	SnapshotID string `json:"snapshot_id"`

	// Status is the cached legacy CI status.
	Status string `json:"status"`
}

// LegacySyncState is the transitional sync cache retained in the local state
// file while status still reads legacy hints directly.
type LegacySyncState struct {
	// Freshness is the cached sync freshness label.
	Freshness string `json:"freshness"`

	// Conflicts reports whether the cached sync state observed conflicts.
	Conflicts bool `json:"conflicts"`
}

// LegacyPublishState is the transitional publish cache retained in the local
// state file while status still reads legacy hints directly.
type LegacyPublishState struct {
	// AttemptID is the legacy publish attempt identifier.
	AttemptID string `json:"attempt_id"`

	// PRURL is the cached publish pull request URL.
	PRURL string `json:"pr_url"`
}
