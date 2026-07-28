package plan_test

import (
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestRenderTemplateSeedsFields(t *testing.T) {
	timestamp := time.Date(2026, 3, 17, 13, 50, 0, 0, time.FixedZone("CST", 8*60*60))
	rendered := renderTemplateWithSize(t, plan.TemplateOptions{
		Title:      "Superharness Test Plan",
		Timestamp:  timestamp,
		SourceType: "issue",
		SourceRefs: []string{"#12", "https://example.com/item"},
	}, "M")

	for _, want := range []string{
		"# Superharness Test Plan",
		"created_at: 2026-03-17T13:50:00+08:00",
		"source_type: issue",
		`source_refs: ["#12","https://example.com/item"]`,
		"size: M",
		"template_version: 0.3.0",
		"- Done: [ ]",
		"### Decisions and Constraints",
		"## Review Focus",
		"- Outcome:",
		"- Covers:",
		"## Closeout",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered template missing %q\n%s", want, rendered)
		}
	}
}

func TestRenderTemplateOmitsPrescriptiveStepAndLegacyCloseoutSections(t *testing.T) {
	rendered := renderTemplateWithSize(t, plan.TemplateOptions{Title: "Lean Plan"}, "M")
	for _, unwanted := range []string{
		"#### Objective",
		"#### Details",
		"#### Expected Files",
		"#### Execution Notes",
		"#### Review Notes",
		"PENDING_STEP_EXECUTION",
		"PENDING_STEP_REVIEW",
		"## Validation Summary",
		"## Review Summary",
		"## Archive Summary",
		"## Outcome Summary",
		"- PR:",
		"- Ready:",
		"- Merge Handoff:",
	} {
		if strings.Contains(rendered, unwanted) {
			t.Fatalf("rendered compact template unexpectedly contains %q\n%s", unwanted, rendered)
		}
	}
}

func TestRenderTemplateRejectsMultilineTitle(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "line one\nline two",
	})
	if err == nil {
		t.Fatal("expected multiline title to fail")
	}
}

func TestRenderTemplateLeavesSizePlaceholderWhenOmitted(t *testing.T) {
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "Missing Size Plan",
	})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if !strings.Contains(rendered, "size: REPLACE_WITH_PLAN_SIZE") {
		t.Fatalf("expected explicit size placeholder, got:\n%s", rendered)
	}
}

func TestRenderTemplateRejectsUnsupportedSize(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "Bad Size Plan",
		Size:  "huge",
	})
	if err == nil {
		t.Fatal("expected unsupported size to fail")
	}
}

func TestRenderTemplateRejectsNonCanonicalSizeSpelling(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "Lowercase Size Plan",
		Size:  "xxs",
	})
	if err == nil {
		t.Fatal("expected non-canonical size spelling to fail")
	}
}

func TestRenderTemplateUsesEmptyArrayForMissingSourceRefs(t *testing.T) {
	rendered := renderTemplateWithSize(t, plan.TemplateOptions{
		Title: "Nil Refs Plan",
	}, "M")
	if !strings.Contains(rendered, "source_refs: []") {
		t.Fatalf("expected empty source_refs array, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "size: M") {
		t.Fatalf("expected default plan size, got:\n%s", rendered)
	}
}

func TestRenderTemplateLightweightSeedsWorkflowProfileAndSingleStep(t *testing.T) {
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           "Lightweight Plan",
		WorkflowProfile: plan.WorkflowProfileLightweight,
	})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	for _, want := range []string{
		"workflow_profile: lightweight",
		"size: XXS",
		"### Step 1: Describe the low-risk change",
		"- Outcome: Describe the narrow low-risk change to make.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered lightweight template missing %q\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "### Step 2:") {
		t.Fatalf("expected lightweight template to collapse to a single step\n%s", rendered)
	}
}

func TestRenderTemplateCoordinatedOmitsRootWorkBreakdown(t *testing.T) {
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           "Coordinated Plan",
		Size:            plan.PlanSizeL,
		WorkflowProfile: plan.WorkflowProfileCoordinated,
	})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	for _, want := range []string{
		"workflow_profile: coordinated",
		"supplements/<plan-stem>/subplans/",
		"## Validation Strategy",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered coordinated template missing %q\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "## Work Breakdown") || strings.Contains(rendered, "### Step 1:") {
		t.Fatalf("coordinated root must not contain sequential root steps\n%s", rendered)
	}
}

func TestRenderSubplanTemplateSeedsDependenciesAndOrderedStep(t *testing.T) {
	rendered, err := plan.RenderSubplanTemplate(plan.SubplanTemplateOptions{
		Title:     "UI",
		DependsOn: []string{"api"},
	})
	if err != nil {
		t.Fatalf("RenderSubplanTemplate returned error: %v", err)
	}
	for _, want := range []string{
		"depends_on: [\"api\"]",
		"# UI",
		"## Work Breakdown",
		"### Step 1:",
		"- Validation: PENDING",
		"- Delivered: PENDING",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered subplan template missing %q\n%s", want, rendered)
		}
	}
}

func TestRenderTemplateRejectsGoalOrientedProfile(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           "Goal-Oriented Plan",
		WorkflowProfile: plan.WorkflowProfileGoalOriented,
	})
	if err == nil {
		t.Fatal("expected deferred goal-oriented profile to be rejected")
	}
}

func TestRenderTemplateRejectsLightweightNonXXSSize(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           "Bad Lightweight Plan",
		Size:            "XS",
		WorkflowProfile: plan.WorkflowProfileLightweight,
	})
	if err == nil {
		t.Fatal("expected non-XXS lightweight template to fail")
	}
}

func TestRenderTemplateRejectsLightweightSizeWithWhitespace(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           "Whitespace Lightweight Plan",
		Size:            "XXS ",
		WorkflowProfile: plan.WorkflowProfileLightweight,
	})
	if err == nil {
		t.Fatal("expected whitespace-padded lightweight size to fail")
	}
}

func TestRenderTemplateIncludesSupplementsArchiveGuidance(t *testing.T) {
	rendered := renderTemplateWithSize(t, plan.TemplateOptions{
		Title: "Supplements Guidance Plan",
	}, "M")
	for _, want := range []string{
		"supplements/<plan-stem>/",
		"supplement absorption in Closeout",
		"formal tracked locations",
		"Lightweight plans should\nnormally avoid",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected supplements archive guidance %q in rendered template, got:\n%s", want, rendered)
		}
	}
}

func renderTemplateWithSize(t *testing.T, opts plan.TemplateOptions, size string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(opts)
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if strings.Contains(rendered, "size: "+size) {
		return rendered
	}
	updated := strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: "+size, 1)
	if updated == rendered {
		t.Fatalf("expected rendered template to contain size frontmatter for replacement")
	}
	return updated
}
