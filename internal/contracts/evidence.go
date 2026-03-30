package contracts

type EvidenceResult struct {
	OK         bool               `json:"ok"`
	Command    string             `json:"command"`
	Summary    string             `json:"summary"`
	Artifacts  *EvidenceArtifacts `json:"artifacts,omitempty"`
	NextAction []NextAction       `json:"next_actions"`
	Errors     []JSONError        `json:"errors,omitempty"`
}

type EvidenceArtifacts struct {
	PlanPath       string `json:"plan_path"`
	LocalStatePath string `json:"local_state_path,omitempty"`
	RecordID       string `json:"record_id"`
	RecordPath     string `json:"record_path"`
	Kind           string `json:"kind"`
}

type EvidenceCIInput struct {
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	URL      string `json:"url,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type EvidencePublishInput struct {
	Status string `json:"status"`
	PRURL  string `json:"pr_url,omitempty"`
	Branch string `json:"branch,omitempty"`
	Base   string `json:"base,omitempty"`
	Commit string `json:"commit,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type EvidenceSyncInput struct {
	Status  string `json:"status"`
	BaseRef string `json:"base_ref,omitempty"`
	HeadRef string `json:"head_ref,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type EvidenceCIRecord struct {
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	PlanPath   string `json:"plan_path"`
	PlanStem   string `json:"plan_stem"`
	Revision   int    `json:"revision"`
	RecordedAt string `json:"recorded_at"`
	Status     string `json:"status"`
	Provider   string `json:"provider,omitempty"`
	URL        string `json:"url,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type EvidencePublishRecord struct {
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	PlanPath   string `json:"plan_path"`
	PlanStem   string `json:"plan_stem"`
	Revision   int    `json:"revision"`
	RecordedAt string `json:"recorded_at"`
	Status     string `json:"status"`
	PRURL      string `json:"pr_url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Base       string `json:"base,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type EvidenceSyncRecord struct {
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	PlanPath   string `json:"plan_path"`
	PlanStem   string `json:"plan_stem"`
	Revision   int    `json:"revision"`
	RecordedAt string `json:"recorded_at"`
	Status     string `json:"status"`
	BaseRef    string `json:"base_ref,omitempty"`
	HeadRef    string `json:"head_ref,omitempty"`
	Reason     string `json:"reason,omitempty"`
}
