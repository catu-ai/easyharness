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

	// Evidence groups durable recorded evidence and compact read-only remote
	// observation facts for archived-candidate handoff.
	Evidence *StatusEvidenceFacts `json:"evidence,omitempty"`

	// LandPRURL is the pull request URL recorded for the land phase.
	LandPRURL string `json:"land_pr_url,omitempty"`

	// LandCommit is the merge commit or landed commit recorded for the land
	// phase.
	LandCommit string `json:"land_commit,omitempty"`
}

// StatusEvidenceFacts groups archived-candidate handoff evidence.
type StatusEvidenceFacts struct {
	// Recorded is durable local evidence. Recorded evidence is authoritative for
	// workflow-node progression.
	Recorded *StatusRecordedEvidence `json:"recorded,omitempty"`

	// Remote is a compact read-only projection of live remote facts. It explains
	// drift or refresh opportunities but does not advance workflow state.
	Remote *StatusRemoteEvidence `json:"remote,omitempty"`
}

// StatusRecordedEvidence summarizes durable archived-candidate evidence.
type StatusRecordedEvidence struct {
	// Publish is the latest recorded publish evidence when present.
	Publish *StatusRecordedPublishEvidence `json:"publish,omitempty"`

	// CI is the latest recorded CI evidence when present.
	CI *StatusRecordedEvidenceStatus `json:"ci,omitempty"`

	// Sync is the latest recorded sync evidence when present.
	Sync *StatusRecordedEvidenceStatus `json:"sync,omitempty"`
}

// StatusRecordedPublishEvidence summarizes recorded publish evidence.
type StatusRecordedPublishEvidence struct {
	// Status is the recorded publish evidence status.
	Status string `json:"status,omitempty"`

	// PRURL is the recorded pull request URL when one is available.
	PRURL string `json:"pr_url,omitempty"`
}

// StatusRecordedEvidenceStatus summarizes one recorded evidence domain.
type StatusRecordedEvidenceStatus struct {
	// Status is the recorded evidence status for this domain.
	Status string `json:"status,omitempty"`
}

// StatusRemoteEvidence is the compact live, read-only remote state surfaced by
// `harness status` for an archived candidate with recorded publish PR evidence.
type StatusRemoteEvidence struct {
	// Observation describes how complete the live remote observation was, such
	// as complete, partial, or unavailable.
	Observation string `json:"observation,omitempty"`

	// Assessment describes the workflow meaning of the remote facts relative to
	// durable recorded evidence. Commands remain exclusively in next_actions.
	Assessment string `json:"assessment,omitempty"`

	// Message explains the remote evidence relationship in concise human-readable
	// language.
	Message string `json:"message,omitempty"`

	// PR summarizes only the high-signal pull request state needed for workflow
	// guidance.
	PR *StatusRemotePRSummary `json:"pr,omitempty"`

	// CI reports the CI evidence status that refresh would record when it can be
	// observed.
	CI *StatusRemoteEvidenceStatus `json:"ci,omitempty"`

	// Sync reports the sync evidence status that refresh would record when it can
	// be observed.
	Sync *StatusRemoteEvidenceStatus `json:"sync,omitempty"`

	// Degraded lists compact non-fatal remote observation failures.
	Degraded []StatusRemoteDegradation `json:"degraded,omitempty"`
}

// StatusRemotePRSummary summarizes only remote pull request state that can
// affect controller guidance.
type StatusRemotePRSummary struct {
	// State is the provider PR state such as OPEN, CLOSED, or MERGED.
	State string `json:"state,omitempty"`

	// Draft reports whether the observed pull request is a draft.
	Draft bool `json:"draft"`
}

// StatusRemoteEvidenceStatus reports the evidence status that a refresh would
// record for one remote-observed domain.
type StatusRemoteEvidenceStatus struct {
	// Status is the evidence status that refresh would record.
	Status string `json:"status,omitempty"`
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

	// LastLandedAt is the timestamp of the most recent landed plan.
	LastLandedAt string `json:"last_landed_at,omitempty"`
}
