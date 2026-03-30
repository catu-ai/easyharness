package contracts

type LifecycleResult struct {
	OK         bool            `json:"ok"`
	Command    string          `json:"command"`
	Summary    string          `json:"summary"`
	State      LifecycleState  `json:"state"`
	Artifacts  *LifecyclePaths `json:"artifacts,omitempty"`
	NextAction []NextAction    `json:"next_actions"`
	Errors     []JSONError     `json:"errors,omitempty"`
}

type LifecycleState struct {
	PlanStatus string `json:"plan_status"`
	Lifecycle  string `json:"lifecycle"`
	Revision   int    `json:"revision"`
}

type LifecyclePaths struct {
	FromPlanPath    string `json:"from_plan_path"`
	ToPlanPath      string `json:"to_plan_path"`
	LocalStatePath  string `json:"local_state_path,omitempty"`
	CurrentPlanPath string `json:"current_plan_path,omitempty"`
}
