package contracts

type ReviewSpec struct {
	Step        *int              `json:"step,omitempty"`
	Kind        string            `json:"kind"`
	ReviewTitle string            `json:"review_title,omitempty"`
	Dimensions  []ReviewDimension `json:"dimensions"`
}

type ReviewDimension struct {
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
}

type ReviewManifest struct {
	RoundID     string               `json:"round_id"`
	Kind        string               `json:"kind"`
	Step        *int                 `json:"step,omitempty"`
	Revision    int                  `json:"revision"`
	ReviewTitle string               `json:"review_title,omitempty"`
	PlanPath    string               `json:"plan_path"`
	PlanStem    string               `json:"plan_stem"`
	CreatedAt   string               `json:"created_at"`
	Dimensions  []ReviewManifestSlot `json:"dimensions"`
	LedgerPath  string               `json:"ledger_path"`
	Aggregate   string               `json:"aggregate_path"`
	Submissions string               `json:"submissions_dir"`
}

type ReviewManifestSlot struct {
	Name           string `json:"name"`
	Slot           string `json:"slot"`
	Instructions   string `json:"instructions"`
	SubmissionPath string `json:"submission_path"`
}

type ReviewLedger struct {
	RoundID   string             `json:"round_id"`
	Kind      string             `json:"kind"`
	UpdatedAt string             `json:"updated_at"`
	Slots     []ReviewLedgerSlot `json:"slots"`
}

type ReviewLedgerSlot struct {
	Name           string `json:"name"`
	Slot           string `json:"slot"`
	Status         string `json:"status"`
	SubmissionPath string `json:"submission_path"`
	SubmittedAt    string `json:"submitted_at,omitempty"`
}

type ReviewSubmissionInput struct {
	Summary  string          `json:"summary"`
	Findings []ReviewFinding `json:"findings"`
}

type ReviewSubmission struct {
	RoundID     string          `json:"round_id"`
	Slot        string          `json:"slot"`
	Dimension   string          `json:"dimension"`
	SubmittedAt string          `json:"submitted_at"`
	Summary     string          `json:"summary"`
	Findings    []ReviewFinding `json:"findings"`
}

type ReviewFinding struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Details  string `json:"details"`
}

type ReviewAggregate struct {
	RoundID             string                   `json:"round_id"`
	Kind                string                   `json:"kind"`
	Step                *int                     `json:"step,omitempty"`
	Revision            int                      `json:"revision"`
	ReviewTitle         string                   `json:"review_title,omitempty"`
	Decision            string                   `json:"decision"`
	BlockingFindings    []ReviewAggregateFinding `json:"blocking_findings"`
	NonBlockingFindings []ReviewAggregateFinding `json:"non_blocking_findings"`
	AggregatedAt        string                   `json:"aggregated_at"`
}

type ReviewAggregateFinding struct {
	Slot      string `json:"slot"`
	Dimension string `json:"dimension"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	Details   string `json:"details"`
}

type ReviewStartResult struct {
	OK         bool                  `json:"ok"`
	Command    string                `json:"command"`
	Summary    string                `json:"summary"`
	Artifacts  *ReviewStartArtifacts `json:"artifacts,omitempty"`
	NextAction []NextAction          `json:"next_actions"`
	Errors     []JSONError           `json:"errors,omitempty"`
}

type ReviewStartArtifacts struct {
	PlanPath       string               `json:"plan_path"`
	LocalStatePath string               `json:"local_state_path"`
	RoundID        string               `json:"round_id"`
	ManifestPath   string               `json:"manifest_path"`
	LedgerPath     string               `json:"ledger_path"`
	AggregatePath  string               `json:"aggregate_path"`
	Slots          []ReviewManifestSlot `json:"slots"`
}

type ReviewSubmitResult struct {
	OK         bool                   `json:"ok"`
	Command    string                 `json:"command"`
	Summary    string                 `json:"summary"`
	Artifacts  *ReviewSubmitArtifacts `json:"artifacts,omitempty"`
	NextAction []NextAction           `json:"next_actions"`
	Errors     []JSONError            `json:"errors,omitempty"`
}

type ReviewSubmitArtifacts struct {
	RoundID        string `json:"round_id"`
	Slot           string `json:"slot"`
	SubmissionPath string `json:"submission_path"`
	LedgerPath     string `json:"ledger_path"`
}

type ReviewAggregateResult struct {
	OK         bool                      `json:"ok"`
	Command    string                    `json:"command"`
	Summary    string                    `json:"summary"`
	Artifacts  *ReviewAggregateArtifacts `json:"artifacts,omitempty"`
	Review     *ReviewAggregate          `json:"review,omitempty"`
	NextAction []NextAction              `json:"next_actions"`
	Errors     []JSONError               `json:"errors,omitempty"`
}

type ReviewAggregateArtifacts struct {
	RoundID        string `json:"round_id"`
	AggregatePath  string `json:"aggregate_path"`
	LocalStatePath string `json:"local_state_path"`
}
