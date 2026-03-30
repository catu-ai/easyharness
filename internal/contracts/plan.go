package contracts

type PlanLintResult struct {
	OK                       bool              `json:"ok"`
	Command                  string            `json:"command"`
	Summary                  string            `json:"summary"`
	Artifacts                PlanLintArtifacts `json:"artifacts,omitempty"`
	SupportedTemplateVersion string            `json:"supported_template_version,omitempty"`
	Errors                   []JSONError       `json:"errors,omitempty"`
}

type PlanLintArtifacts struct {
	PlanPath string `json:"plan_path"`
}
