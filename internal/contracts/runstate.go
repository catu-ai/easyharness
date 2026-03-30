package contracts

type RunstateCurrentPlan struct {
	PlanPath           string `json:"plan_path,omitempty"`
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`
	LastLandedAt       string `json:"last_landed_at,omitempty"`
}

type RunstateState struct {
	ExecutionStartedAt string                `json:"execution_started_at,omitempty"`
	CurrentNode        string                `json:"current_node,omitempty"`
	PlanPath           string                `json:"plan_path,omitempty"`
	PlanStem           string                `json:"plan_stem,omitempty"`
	Revision           int                   `json:"revision,omitempty"`
	Reopen             *RunstateReopenState  `json:"reopen,omitempty"`
	ActiveReviewRound  *RunstateReviewRound  `json:"active_review_round,omitempty"`
	LatestEvidence     *RunstateEvidenceSet  `json:"latest_evidence,omitempty"`
	Land               *RunstateLandState    `json:"land,omitempty"`
	LatestCI           *RunstateCIState      `json:"latest_ci,omitempty"`
	Sync               *RunstateSyncState    `json:"sync,omitempty"`
	LatestPublish      *RunstatePublishState `json:"latest_publish,omitempty"`
}

type RunstateReopenState struct {
	Mode          string `json:"mode"`
	ReopenedAt    string `json:"reopened_at,omitempty"`
	BaseStepCount int    `json:"base_step_count,omitempty"`
}

type RunstateReviewRound struct {
	RoundID    string `json:"round_id"`
	Kind       string `json:"kind"`
	Step       *int   `json:"step,omitempty"`
	Revision   int    `json:"revision,omitempty"`
	Aggregated bool   `json:"aggregated"`
	Decision   string `json:"decision,omitempty"`
}

type RunstateEvidenceSet struct {
	CI      *RunstateEvidencePointer `json:"ci,omitempty"`
	Publish *RunstateEvidencePointer `json:"publish,omitempty"`
	Sync    *RunstateEvidencePointer `json:"sync,omitempty"`
}

type RunstateEvidencePointer struct {
	Kind       string `json:"kind"`
	RecordID   string `json:"record_id"`
	Path       string `json:"path"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

type RunstateLandState struct {
	PRURL       string `json:"pr_url,omitempty"`
	Commit      string `json:"commit,omitempty"`
	LandedAt    string `json:"landed_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type RunstateCIState struct {
	SnapshotID string `json:"snapshot_id"`
	Status     string `json:"status"`
}

type RunstateSyncState struct {
	Freshness string `json:"freshness"`
	Conflicts bool   `json:"conflicts"`
}

type RunstatePublishState struct {
	AttemptID string `json:"attempt_id"`
	PRURL     string `json:"pr_url"`
}
