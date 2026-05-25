package contracts

// StatusResult is the JSON result returned by `harness status`.
type StatusResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable explanation of the current state.
	Summary string `json:"summary"`

	// State is the canonical workflow position resolved for the current worktree.
	State StatusState `json:"state"`

	// Facts carries selected high-signal derived details that help explain the
	// current node.
	Facts *StatusFacts `json:"facts,omitempty"`

	// Artifacts points to stable paths or identifiers that help the caller locate
	// related contract artifacts.
	Artifacts *StatusArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Blockers lists state issues that block ordinary execution progression.
	Blockers []ErrorDetail `json:"blockers,omitempty"`

	// Warnings lists non-blocking workflow reminders or ambiguity notes.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented full state resolution.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// StatusState is the canonical workflow node returned by `harness status`.
type StatusState struct {
	// CurrentNode is the resolved v0.2 workflow node for the current worktree.
	CurrentNode string `json:"current_node" jsonschema:"example=execution/step-1/implement"`
}

// StatusFacts carries selected derived details that help interpret a status
// node without dumping the full local state model.
type StatusFacts struct {
	// CurrentStep is the first unfinished plan step when one exists.
	CurrentStep string `json:"current_step,omitempty"`

	// Revision is the current plan-local revision number.
	Revision int `json:"revision,omitempty"`

	// ReopenMode is the active reopen mode when a reopen repair path is in
	// effect.
	ReopenMode string `json:"reopen_mode,omitempty"`

	// ReviewKind is the review kind for the active review round.
	ReviewKind string `json:"review_kind,omitempty"`

	// ReviewTrigger is the derived reason label for the active review round.
	ReviewTrigger string `json:"review_trigger,omitempty"`

	// ReviewTitle is the human-readable title for the active review round when
	// one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// ReviewStatus summarizes the aggregate review state for the current node.
	ReviewStatus string `json:"review_status,omitempty"`

	// ArchiveBlockerCount reports how many archive-readiness blockers are still
	// present.
	ArchiveBlockerCount int `json:"archive_blocker_count,omitempty"`

	// PublishStatus summarizes the latest publish evidence state.
	PublishStatus string `json:"publish_status,omitempty"`

	// PRURL is the currently known pull request URL for the candidate branch.
	PRURL string `json:"pr_url,omitempty"`

	// CIStatus summarizes the latest CI evidence state.
	CIStatus string `json:"ci_status,omitempty"`

	// SyncStatus summarizes the latest remote-sync evidence state.
	SyncStatus string `json:"sync_status,omitempty"`

	// RemoteHandoff is a non-authoritative live observation of the recorded PR
	// handoff target. It may explain remote PR, CI, and sync state, but local
	// durable evidence remains the source of truth for workflow progression.
	RemoteHandoff *StatusRemoteHandoffObservation `json:"remote_handoff,omitempty"`

	// LandPRURL is the pull request URL recorded for the land phase.
	LandPRURL string `json:"land_pr_url,omitempty"`

	// LandCommit is the merge commit or landed commit recorded for the land
	// phase.
	LandCommit string `json:"land_commit,omitempty"`
}

// StatusRemoteHandoffObservation is the live, read-only remote state surfaced
// by `harness status` for an archived candidate with recorded publish PR
// evidence.
type StatusRemoteHandoffObservation struct {
	// Status is the overall remote observation status such as available,
	// degraded, or unavailable.
	Status string `json:"status"`

	// PR summarizes the recorded pull request when it can be observed.
	PR StatusRemotePRObservation `json:"pr"`

	// CI summarizes the live PR check state when it can be observed.
	CI StatusRemoteCIObservation `json:"ci"`

	// Sync summarizes the live PR merge or freshness state when it can be
	// observed.
	Sync StatusRemoteSyncObservation `json:"sync"`

	// Degraded lists non-fatal remote observation failures.
	Degraded []StatusRemoteDegradation `json:"degraded,omitempty"`
}

// StatusRemotePRObservation summarizes the recorded pull request observed by
// `harness status`.
type StatusRemotePRObservation struct {
	// Status is the PR observation status such as available or unavailable.
	Status string `json:"status"`

	// URL is the recorded pull request URL.
	URL string `json:"url,omitempty"`

	// Number is the pull request number when available.
	Number int `json:"number,omitempty"`

	// State is the provider PR state such as OPEN, CLOSED, or MERGED.
	State string `json:"state,omitempty"`

	// IsDraft reports whether the observed pull request is a draft.
	IsDraft bool `json:"is_draft,omitempty"`

	// MergeStateStatus is the provider merge-state status when available.
	MergeStateStatus string `json:"merge_state_status,omitempty"`

	// Mergeable is the provider mergeability value when available.
	Mergeable string `json:"mergeable,omitempty"`

	// ReviewDecision is the provider review decision when available.
	ReviewDecision string `json:"review_decision,omitempty"`

	// HeadRefName is the pull request head branch name when available.
	HeadRefName string `json:"head_ref_name,omitempty"`

	// HeadRefOID is the pull request head commit OID when available.
	HeadRefOID string `json:"head_ref_oid,omitempty"`

	// BaseRefName is the pull request base branch name when available.
	BaseRefName string `json:"base_ref_name,omitempty"`

	// Degraded explains why the PR could not be observed.
	Degraded *StatusRemoteDegradation `json:"degraded,omitempty"`
}

// StatusRemoteCIObservation summarizes live remote CI/check state for a
// recorded pull request.
type StatusRemoteCIObservation struct {
	// Status is the CI observation status such as available or unavailable.
	Status string `json:"status"`

	// EvidenceStatus is the local evidence status that `harness evidence
	// refresh` would record if this remote observation were refreshed now.
	EvidenceStatus string `json:"evidence_status,omitempty"`

	// Checks lists compact remote check rows.
	Checks []StatusRemoteCheckRun `json:"checks,omitempty"`

	// Degraded explains why CI checks could not be observed.
	Degraded *StatusRemoteDegradation `json:"degraded,omitempty"`
}

// StatusRemoteSyncObservation summarizes live remote merge/sync state for a
// recorded pull request.
type StatusRemoteSyncObservation struct {
	// Status is the sync observation status such as available or unavailable.
	Status string `json:"status"`

	// EvidenceStatus is the local evidence status that `harness evidence
	// refresh` would record if this remote observation were refreshed now.
	EvidenceStatus string `json:"evidence_status,omitempty"`

	// MergeState is the provider merge-state value used for sync classification.
	MergeState string `json:"merge_state,omitempty"`

	// Degraded explains why sync state could not be observed.
	Degraded *StatusRemoteDegradation `json:"degraded,omitempty"`
}

// StatusRemoteCheckRun is a compact provider check row surfaced through
// `harness status`.
type StatusRemoteCheckRun struct {
	// Name is the check run name when available.
	Name string `json:"name,omitempty"`

	// Workflow is the workflow name when available.
	Workflow string `json:"workflow,omitempty"`

	// Bucket is the provider's coarse check bucket when available.
	Bucket string `json:"bucket,omitempty"`

	// State is the provider's check state when available.
	State string `json:"state,omitempty"`

	// Link is the provider URL for the check run when available.
	Link string `json:"link,omitempty"`
}

// StatusRemoteDegradation explains a non-fatal live remote observation
// failure.
type StatusRemoteDegradation struct {
	// Code is a stable degradation code such as gh_missing or pr_unreadable.
	Code string `json:"code"`

	// Message is a concise human-readable explanation.
	Message string `json:"message,omitempty"`
}

// StatusArtifacts lists stable paths or identifiers related to the current
// status result.
type StatusArtifacts struct {
	// ProjectRoot is the repository root that anchors surfaced repo-facing
	// paths.
	ProjectRoot string `json:"project_root,omitempty"`

	// PlanPath is the active, archived, or last-landed plan path relevant to the
	// current status resolution.
	PlanPath string `json:"plan_path,omitempty" jsonschema:"example=docs/plans/active/2026-03-31-centralize-contract-schemas-and-generated-reference-docs.md"`

	// SupplementsPath is the companion supplements directory for the current
	// plan package when one exists.
	SupplementsPath string `json:"supplements_path,omitempty" jsonschema:"example=docs/plans/active/supplements/2026-03-31-centralize-contract-schemas-and-generated-reference-docs"`

	// ReviewRoundID is the active review round identifier when review is in
	// flight.
	ReviewRoundID string `json:"review_round_id,omitempty"`

	// ReviewSlots lists the active round's reviewer-owned slot handles when
	// review is in flight.
	ReviewSlots []ReviewSlot `json:"review_slots,omitempty"`

	// CIRecordID is the latest CI evidence record identifier when known.
	CIRecordID string `json:"ci_record_id,omitempty"`

	// PublishRecordID is the latest publish evidence record identifier when
	// known.
	PublishRecordID string `json:"publish_record_id,omitempty"`

	// SyncRecordID is the latest sync evidence record identifier when known.
	SyncRecordID string `json:"sync_record_id,omitempty"`

	// LastLandedAt is the timestamp of the most recent landed plan.
	LastLandedAt string `json:"last_landed_at,omitempty"`
}
