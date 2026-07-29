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

type SubplanTemplateOptions struct {
	Title     string
	DependsOn []string
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
	if workflowProfile != WorkflowProfileStandard &&
		workflowProfile != WorkflowProfileLightweight &&
		workflowProfile != WorkflowProfileCoordinated {
		return "", fmt.Errorf(
			"workflow profile must be %q, %q, or %q",
			WorkflowProfileStandard,
			WorkflowProfileLightweight,
			WorkflowProfileCoordinated,
		)
	}
	if workflowProfile == WorkflowProfileLightweight {
		if size == "" {
			size = PlanSizeXXS
		}
		if size != PlanSizeXXS {
			return "", fmt.Errorf("lightweight templates must use size %q", PlanSizeXXS)
		}
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
		rendered = strings.Replace(rendered, "- Outcome: Describe the concrete outcome for this step.", "- Outcome: Describe the narrow low-risk change to make.", 1)
		rendered = strings.Replace(rendered, "Describe the validation approach for the whole plan.", "Describe the focused validation needed for this low-risk change.", 1)
		stepTwoMarker := "\n### Step 2: Replace with second step title"
		if start := strings.Index(rendered, stepTwoMarker); start >= 0 {
			if end := strings.Index(rendered[start:], "\n## Validation Strategy"); end >= 0 {
				rendered = rendered[:start] + rendered[start+end:]
			}
		}
	} else if workflowProfile == WorkflowProfileCoordinated {
		rendered = strings.Replace(rendered, "size: "+size, "size: "+size+"\nworkflow_profile: coordinated", 1)
		workBreakdownMarker := "\n## Work Breakdown"
		if start := strings.Index(rendered, workBreakdownMarker); start >= 0 {
			if end := strings.Index(rendered[start:], "\n## Validation Strategy"); end >= 0 {
				rendered = rendered[:start] + rendered[start+end:]
			}
		}
		rendered = strings.Replace(
			rendered,
			"<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,\nabsorb repository-facing normative content into formal tracked locations before\narchive, and record supplement absorption in Closeout. Lightweight plans should\nnormally avoid supplements. -->",
			"<!-- This coordinated root owns flat subplans under\nsupplements/<plan-stem>/subplans/. Human approval applies to this root; agents\nmay revise the child decomposition within its approved boundary. -->",
			1,
		)
	}
	return rendered, nil
}

func RenderSubplanTemplate(opts SubplanTemplateOptions) (string, error) {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Replace With Subplan Title"
	}
	if strings.Contains(title, "\n") {
		return "", fmt.Errorf("title must be a single line")
	}
	for _, dependency := range opts.DependsOn {
		if err := validateSubplanID(dependency); err != nil {
			return "", fmt.Errorf("invalid dependency %q: %w", dependency, err)
		}
	}
	dependencies, err := json.Marshal(opts.DependsOn)
	if err != nil {
		return "", fmt.Errorf("marshal dependencies: %w", err)
	}
	rendered := templateassets.SubplanTemplate()
	rendered = strings.Replace(rendered, "# Replace With Subplan Title", "# "+title, 1)
	rendered = strings.Replace(rendered, "depends_on: []", "depends_on: "+string(dependencies), 1)
	return rendered, nil
}
