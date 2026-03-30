package contracts

type StatusResult struct {
	OK         bool          `json:"ok"`
	Command    string        `json:"command"`
	Summary    string        `json:"summary"`
	State      StatusState   `json:"state"`
	Facts      *StatusFacts  `json:"facts,omitempty"`
	Artifacts  *StatusOutput `json:"artifacts,omitempty"`
	NextAction []NextAction  `json:"next_actions"`
	Blockers   []JSONError   `json:"blockers,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	Errors     []JSONError   `json:"errors,omitempty"`
}

type StatusState struct {
	CurrentNode string `json:"current_node"`
}

type StatusFacts struct {
	CurrentStep         string `json:"current_step,omitempty"`
	Revision            int    `json:"revision,omitempty"`
	ReopenMode          string `json:"reopen_mode,omitempty"`
	ReviewKind          string `json:"review_kind,omitempty"`
	ReviewTrigger       string `json:"review_trigger,omitempty"`
	ReviewTitle         string `json:"review_title,omitempty"`
	ReviewStatus        string `json:"review_status,omitempty"`
	ArchiveBlockerCount int    `json:"archive_blocker_count,omitempty"`
	PublishStatus       string `json:"publish_status,omitempty"`
	PRURL               string `json:"pr_url,omitempty"`
	CIStatus            string `json:"ci_status,omitempty"`
	SyncStatus          string `json:"sync_status,omitempty"`
	LandPRURL           string `json:"land_pr_url,omitempty"`
	LandCommit          string `json:"land_commit,omitempty"`
}

type StatusOutput struct {
	PlanPath           string `json:"plan_path,omitempty"`
	LocalStatePath     string `json:"local_state_path,omitempty"`
	ReviewRoundID      string `json:"review_round_id,omitempty"`
	CIRecordID         string `json:"ci_record_id,omitempty"`
	PublishRecordID    string `json:"publish_record_id,omitempty"`
	SyncRecordID       string `json:"sync_record_id,omitempty"`
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`
	LastLandedAt       string `json:"last_landed_at,omitempty"`
}
