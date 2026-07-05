package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	templateassets "github.com/catu-ai/easyharness/assets/templates"
)

const (
	placeholderTitle     = "Replace With Plan Title"
	placeholderTimestamp = "REPLACE_WITH_RFC3339_TIMESTAMP"
)

type TemplateOptions struct {
	Title           string
	Timestamp       time.Time
	SourceType      string
	SourceRefs      []string
	Size            string
	WorkflowProfile string
}

func RenderTemplate(opts TemplateOptions) (string, error) {
	template := templateassets.PlanTemplate()
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = placeholderTitle
	}

	if strings.Contains(title, "\n") {
		return "", fmt.Errorf("title must be a single line")
	}

	if opts.Timestamp.IsZero() {
		opts.Timestamp = time.Now()
	}

	sourceType := strings.TrimSpace(opts.SourceType)
	if sourceType == "" {
		sourceType = "direct_request"
	}
	if opts.SourceRefs == nil {
		opts.SourceRefs = []string{}
	}
	size := opts.Size
	workflowProfile := normalizeWorkflowProfile(opts.WorkflowProfile)
	if workflowProfile != WorkflowProfileStandard && workflowProfile != WorkflowProfileLightweight && workflowProfile != WorkflowProfileGoalOriented {
		return "", fmt.Errorf("workflow profile must be %q, %q, or %q", WorkflowProfileStandard, WorkflowProfileLightweight, WorkflowProfileGoalOriented)
	}
	if workflowProfile == WorkflowProfileLightweight {
		if size == "" {
			size = PlanSizeXXS
		}
		if size != PlanSizeXXS {
			return "", fmt.Errorf("lightweight templates must use size %q", PlanSizeXXS)
		}
	}
	if workflowProfile == WorkflowProfileGoalOriented && size == "" {
		size = PlanSizeM
	}
	if size == "" {
		size = placeholderPlanSize
	}
	if size != placeholderPlanSize && !isSupportedPlanSize(size) {
		return "", fmt.Errorf("size must be one of %s", strings.Join(supportedPlanSizes, ", "))
	}

	sourceRefsJSON, err := json.Marshal(opts.SourceRefs)
	if err != nil {
		return "", fmt.Errorf("marshal source refs: %w", err)
	}

	timestamp := opts.Timestamp.Format(time.RFC3339)

	rendered := template
	rendered = strings.Replace(rendered, "# "+placeholderTitle, "# "+title, 1)
	rendered = strings.ReplaceAll(rendered, placeholderTimestamp, timestamp)
	rendered = strings.Replace(rendered, "source_type: direct_request", "source_type: "+sourceType, 1)
	rendered = strings.Replace(rendered, "source_refs: []", "source_refs: "+string(sourceRefsJSON), 1)
	rendered = strings.Replace(rendered, "size: "+placeholderPlanSize, "size: "+size, 1)
	if workflowProfile == WorkflowProfileLightweight {
		rendered = strings.Replace(rendered, "size: "+size, "size: "+size+"\nworkflow_profile: lightweight", 1)
		rendered = strings.Replace(rendered, "### Step 1: Replace with first step title", "### Step 1: Describe the low-risk change", 1)
		rendered = strings.Replace(rendered, "Describe the concrete outcome for this step.", "Describe the narrow low-risk change to make.", 1)
		rendered = strings.Replace(rendered, "Describe the step-specific details, tradeoffs, or constraints that do not fit\nin the one-line objective. Write `NONE` if the objective is already enough.", "Explain why this change qualifies for the lightweight path and note any constraints. Write `NONE` if the objective already says enough.", 1)
		rendered = strings.Replace(rendered, "Describe the validation approach for the whole plan.", "Describe the focused validation needed for this low-risk change.", 1)
		stepTwoMarker := "\n### Step 2: Replace with second step title"
		if start := strings.Index(rendered, stepTwoMarker); start >= 0 {
			if end := strings.Index(rendered[start:], "\n## Validation Strategy"); end >= 0 {
				rendered = rendered[:start] + rendered[start+end:]
			}
		}
	}
	if workflowProfile == WorkflowProfileGoalOriented {
		rendered = renderGoalOrientedTemplate(rendered, size)
	}

	return rendered, nil
}

func renderGoalOrientedTemplate(rendered string, size string) string {
	rendered = strings.Replace(rendered, "size: "+size, "size: "+size+"\nworkflow_profile: goal_oriented", 1)
	rendered = strings.Replace(rendered, "Describe the intended outcome in one or two short paragraphs.", goalOrientedGoal, 1)
	rendered = strings.Replace(rendered, "### In Scope\n\n- Item", "### In Scope\n\n- Define the objective, scorecard, hypotheses or candidate directions, checkpoint cadence, evidence requirements, challenge triggers, stopping conditions, and final synthesis.\n- Keep tracked checkpoint reports inside the adaptive step instead of turning each checkpoint into a separate harness workflow step.\n- Treat this as a recognized preview workflow profile for v0.6.0 authoring; full execution support is still being completed.", 1)
	rendered = strings.Replace(rendered, "### Out of Scope\n\n- Item", "### Out of Scope\n\n- Full status next-action support, archive/reopen profile preservation, challenge/review protocol, UI rendering, and structural lint enforcement.\n- A hypothesis state machine or separate workflow engine.\n- Changes to default standard or lightweight behavior.", 1)
	rendered = strings.Replace(rendered, "- [ ] Criterion 1\n- [ ] Criterion 2", goalOrientedAcceptanceCriteria, 1)
	rendered = strings.Replace(rendered, "- None.", "- Full execution support remains in the v0.6.0 follow-up implementation slices for challenge/review guidance, status next actions, structural lint coverage, docs/examples, archive/reopen behavior, and UI rendering.", 1)
	rendered = strings.Replace(rendered, "### Step 1: Replace with first step title", "### Step 1: Frame objective and scorecard", 1)
	rendered = strings.Replace(rendered, "Describe the concrete outcome for this step.", "Define the adaptive objective, success scorecard, initial hypotheses or candidate directions, evidence requirements, challenge triggers, checkpoint cadence, and stopping conditions.", 1)
	rendered = strings.Replace(rendered, "Describe the step-specific details, tradeoffs, or constraints that do not fit\nin the one-line objective. Write `NONE` if the objective is already enough.", goalOrientedStepOneDetails, 1)
	rendered = strings.Replace(rendered, "- `path/to/file`", "- The tracked plan body or approved plan package artifacts that define the objective, scorecard, and evidence surfaces.", 1)
	rendered = strings.Replace(rendered, "- Describe how the agent will know this step is complete.\n- Mention the automated tests that should be added or updated if this step\n  changes behavior.", "- The objective, scorecard, hypotheses or candidate directions, checkpoint cadence, challenge triggers, evidence requirements, and stopping conditions are explicit enough for a future agent to resume without hidden chat context.", 1)
	rendered = strings.Replace(rendered, "### Step 2: Replace with second step title", "### Step 2: Run adaptive exploration", 1)
	rendered = strings.Replace(rendered, "Describe the concrete outcome for this step.", "Run bounded probes against the scorecard and write tracked checkpoint reports when the decision space changes meaningfully.", 1)
	rendered = strings.Replace(rendered, "Describe the step-specific details, tradeoffs, or constraints that do not fit\nin the one-line objective. Write `NONE` if the objective is already enough.", goalOrientedStepTwoDetails, 1)
	rendered = strings.Replace(rendered, "- `path/to/file`", "- The tracked plan body, especially the step-local `#### Checkpoint Reports` section.\n- Approved durable support artifacts only when the plan explicitly needs them.", 1)
	rendered = strings.Replace(rendered, "- Describe how the agent will know this step is complete.\n- Mention the automated tests that should be added or updated if this step\n  changes behavior.", "- Checkpoint reports explain decision movement against the scorecard, including evidence pointers and residual uncertainty.\n- The controller can decide whether to continue probing, request challenge, synthesize, or ask the human about scope.", 1)
	rendered = strings.Replace(rendered, "- The controller can decide whether to continue probing, request challenge, synthesize, or ask the human about scope.\n\n#### Execution Notes", "- The controller can decide whether to continue probing, request challenge, synthesize, or ask the human about scope.\n"+goalOrientedCheckpointReports+"#### Execution Notes", 1)
	rendered = strings.Replace(rendered, "\n## Validation Strategy", goalOrientedStepThree+"\n## Validation Strategy", 1)
	rendered = strings.Replace(rendered, "- Describe the validation approach for the whole plan.", "- Validate conclusions against the success scorecard, tracked checkpoint reports, durable evidence, and final synthesis.\n- Run focused automated or manual validation that matches the objective; record evidence pointers rather than raw logs.\n- Before archive, ensure accepted conclusions, rejected hypotheses, residual uncertainty, and follow-up issues are reflected in the final synthesis and outcome summaries.", 1)
	rendered = strings.Replace(rendered, "- Risk: Describe the main risk.\n  - Mitigation: Describe how the work reduces or contains it.", goalOrientedRisks, 1)
	return rendered
}

const goalOrientedGoal = `Describe the concrete objective or question to answer, the success scorecard that decides whether the work is complete, and why the path must adapt through hypotheses, probes, checkpoint reports, and final synthesis.

` + "`workflow_profile: goal_oriented`" + ` is a recognized preview workflow profile defined for v0.6.0 authoring; full execution support is still being completed. Use it when the objective is clear but the execution path cannot honestly be reduced to a fixed linear checklist.`

const goalOrientedAcceptanceCriteria = `- [ ] The objective and success scorecard are explicit enough to guide adaptive work.
- [ ] Initial hypotheses or candidate directions are named, along with the evidence needed to compare them.
- [ ] Checkpoint cadence, challenge triggers, stopping conditions, and final synthesis expectations are defined.
- [ ] Tracked checkpoint reports live under adaptive steps and do not automatically become separate harness workflow steps.`

const goalOrientedStepOneDetails = `Capture the goal-oriented authoring contract before probing starts:

- Objective:
- Success Scorecard:
- Hypotheses / Candidate Directions:
- Checkpoint Cadence:
- Challenge Triggers:
- Evidence Requirements:
- Stopping Conditions:

Keep this step focused on framing. Do not use it as a hidden execution log.`

const goalOrientedStepTwoDetails = `Run the probe/checkpoint loop inside this adaptive step:

1. Pick the next bounded hypothesis, candidate direction, or comparison.
2. Run the smallest useful probe or experiment.
3. Compare observed signal with the success scorecard.
4. Write a tracked checkpoint report when the decision space changes.
5. Decide whether to continue, pivot, request challenge, synthesize, or ask the human about scope.

Checkpoint reports stay inside adaptive steps. Do not create one harness step per model turn, probe, or checkpoint report.`

const goalOrientedCheckpointReports = `
#### Checkpoint Reports

##### CP1 - Replace with checkpoint title

Trigger:

Hypotheses:

Probe:

Observed Result:

Scorecard Movement:

Decision / Next Mutation:

Residuals:

Evidence:

`

const goalOrientedStepThree = `
### Step 3: Synthesize and close out

- Done: [ ]

#### Objective

Write the final synthesis and prepare ordinary validation, review, archive, and follow-up handoff.

#### Details

Final Synthesis:

- Accepted Conclusions:
- Rejected Hypotheses / Candidate Directions:
- Residual Uncertainty:
- Evidence:
- Follow-Up / Deferred:

The synthesis should absorb durable learning from checkpoint reports without duplicating raw process logs.

#### Expected Files

- The tracked plan body or the approved deliverable named by the plan.

#### Validation

- The final synthesis is supported by the success scorecard, checkpoint reports, and durable evidence.
- Follow-up issues or deferred items are explicit when uncertainty remains out of scope.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

`

const goalOrientedRisks = `- Risk: Adaptive work drifts into unbounded exploration.
  - Mitigation: Use the scorecard, checkpoint cadence, and stopping conditions to decide when to synthesize, defer, or ask the human about scope.
- Risk: Checkpoint reports become raw logs.
  - Mitigation: Record decision movement, evidence pointers, and residuals; keep raw drafts local or summarize them into approved artifacts.
- Risk: The preview profile is mistaken for complete execution support.
  - Mitigation: Treat this as a recognized preview workflow profile for authoring while full execution support is still being completed.`
