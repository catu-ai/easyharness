package plan_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestLintFileAcceptsValidActivePlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Valid Active Plan")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileAcceptsDoneMarkers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-done-marker-plan.md")
	content := mustRenderTemplate(t, "Done Marker Plan")
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileRequiresCompactPlanSectionsAndStepFields(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-compact-contract.md")
	content := mustRenderTemplate(t, "Compact Contract")
	content = strings.Replace(content, "### Decisions and Constraints\n\n- Decision or constraint\n\n", "", 1)
	content = strings.Replace(content, "## Review Focus\n\n- Review focus\n\n", "", 1)
	content = strings.Replace(content, "- Outcome: Describe the concrete outcome for this step.\n", "", 1)
	content = strings.Replace(content, "- Covers: Criterion 1\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected compact contract violations, got %#v", result)
	}
	assertHasError(t, result, "section.Goal")
	assertHasError(t, result, "sections")
	assertHasError(t, result, "step.Step 1: Replace with first step title.outcome")
	assertHasError(t, result, "step.Step 1: Replace with first step title.covers")
}

func TestLintFileRejectsLegacyStepSubsections(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-legacy-step-details.md")
	content := mustRenderTemplate(t, "Legacy Step Details")
	content = strings.Replace(content, "- Covers: Criterion 1", "- Covers: Criterion 1\n\n#### Details\n\nImplementation recipe.", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected legacy step subsection to fail, got %#v", result)
	}
	assertHasError(t, result, "step.Step 1: Replace with first step title")
}

func TestLintFileRejectsLegacyRuntimeFrontmatter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-legacy-frontmatter-plan.md")
	content := mustRenderTemplate(t, "Legacy Runtime Frontmatter")
	content = strings.Replace(content, "template_version: 0.3.0\n", "status: active\nlifecycle: awaiting_plan_approval\nrevision: 1\nupdated_at: 2026-03-17T14:00:00+08:00\ntemplate_version: 0.3.0\n", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.status")
	assertHasError(t, result, "frontmatter.lifecycle")
	assertHasError(t, result, "frontmatter.revision")
	assertHasError(t, result, "frontmatter.updated_at")
}

func TestLintFileRejectsMissingSize(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-missing-size-plan.md")
	content := mustRenderTemplate(t, "Missing Size")
	content = strings.Replace(content, "size: M\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected missing size to fail, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.size")
}

func TestLintFileRejectsUnsupportedSize(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-unsupported-size-plan.md")
	content := mustRenderTemplate(t, "Unsupported Size")
	content = strings.Replace(content, "size: M", "size: HUGE", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected unsupported size to fail, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.size")
}

func TestLintFileRejectsNonCanonicalSizeSpelling(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-lowercase-size-plan.md")
	content := mustRenderTemplate(t, "Lowercase Size")
	content = strings.Replace(content, "size: M", "size: xxs", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lowercase size to fail, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.size")
}

func TestLintFileRejectsMissingDeferredItemsSection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Invalid Active Plan")
	content = strings.Replace(content, "## Deferred Items\n\n- None.\n\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "sections")
}

func TestLintFileRejectsMissingAcceptanceCriteriaSectionWithoutPanic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Missing Acceptance Criteria")
	content = strings.Replace(content, "## Acceptance Criteria\n\n- [ ] Criterion 1\n- [ ] Criterion 2\n\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Acceptance Criteria")
}

func TestLintFileRejectsArchivedPlanWithPlaceholders(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Placeholder Plan")
	content = strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")
	content = checkAllBoxes(content)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Closeout")
}

func TestLintFileRejectsArchivedPlanWithUncheckedDoneMarker(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-archived-done-plan.md")
	content := mustRenderTemplate(t, "Archived Done Plan")
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 1)
	content = makeArchiveReady(checkAllBoxes(content))
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "step.Step 2: Replace with second step title.done")
}

func TestLintFileRejectsLegacyStepStatusMarkers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-mixed-step-encodings.md")
	content := mustRenderTemplate(t, "Mixed Step Encodings")
	content = strings.Replace(content, "- Done: [ ]", "- Status: in_progress", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "step.Step 1: Replace with first step title")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutCloseoutWithoutPanic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Missing Closeout")
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = makeArchiveReady(checkAllBoxes(strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")))
	if start := strings.Index(content, "## Closeout\n"); start >= 0 {
		content = content[:start]
	}
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Closeout")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutFollowUpIssue(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Deferred Item Plan")
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = makeArchiveReady(checkAllBoxes(strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")))
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Closeout.Follow-Up Issues")
}

func TestLintFileAcceptsHistoricalTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Historical Template Version")
	content = strings.Replace(content, "template_version: 0.3.0", "template_version: 0.0.1", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected older template_version to remain valid, got %#v", result)
	}
}

func TestLintFileRejectsFutureTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Future Template Version")
	content = strings.Replace(content, "template_version: 0.3.0", "template_version: 9.9.9", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "frontmatter.template_version")
}

func TestLintFileRejectsInvalidFilename(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/not-a-valid-name.md")
	content := mustRenderTemplate(t, "Bad Filename")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "path")
}

func TestLintFileRejectsPlanMarkdownInsideSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/supplements/2026-03-17-bad-place.md")
	content := mustRenderTemplate(t, "Bad Supplements Placement")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "path")
}

func TestLintFileRejectsDefaultRootWhenCustomPlanRootsAreConfigured(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness
`)
	path := filepath.Join(root, "docs/plans/active/2026-03-17-stale-default.md")
	content := mustRenderTemplate(t, "Stale Default Root")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure for stale default-root plan, got %#v", result)
	}
	assertHasError(t, result, "path")
}

func TestLintFileAcceptsConfiguredArchivedPlanRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness
`)
	path := filepath.Join(root, "workflow/plans/done/2026-03-17-archived-custom-root.md")
	content := mustRenderTemplate(t, "Archived Custom Root")
	content = makeArchiveReady(checkAllBoxes(strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")))
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected configured archived lint success, got %#v", result)
	}
}

func TestLintFileAllowsValidPlanWhenAncestorDirectoryIsNamedSupplements(t *testing.T) {
	root := filepath.Join(t.TempDir(), "supplements-parent", "project")
	path := filepath.Join(root, "docs/plans/active/2026-03-17-valid-plan.md")
	content := mustRenderTemplate(t, "Valid Plan Under Supplements-Named Ancestor")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileRejectsSupplementsPathWhenItIsAFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Supplements Path Must Be Directory")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-has-supplements"), "not a directory")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileRejectsSupplementsParentPathWhenItIsAFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Supplements Parent Must Be Directory")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements"), "not a directory")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileRejectsArchivedSupplementsParentPathWhenItIsAFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Archived Supplements Parent Must Be Directory")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/archived/supplements"), "not a directory")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileAcceptsMatchingSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Matching Supplements Directory")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-has-supplements/spec.md"), "# draft\n")

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileAcceptsValidPlanReviewGuidance(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-guidance.md")
	writeFile(t, path, mustRenderTemplate(t, "Plan Review Guidance"))
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-guidance/review-guidance/state-risk.md"), `---
name: state-risk
description: Check the plan-specific state risk.
---

Verify the plan-specific state invariant.
`)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected valid plan guidance to lint, got %#v", result)
	}
}

func TestLintFileAcceptsRelativePlanPathWithMatchingSupplements(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-17-relative-guidance.md"
	path := filepath.Join(root, filepath.FromSlash(relPath))
	writeFile(t, path, mustRenderTemplate(t, "Relative Plan Review Guidance"))
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-relative-guidance/review-guidance/state-risk.md"), `---
name: state-risk
description: Check the plan-specific state risk.
---

Verify the plan-specific state invariant.
`)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	result := plan.LintFile(relPath)
	if !result.OK {
		t.Fatalf("expected relative plan path with matching supplements to lint, got %#v", result)
	}
}

func TestLintFileRejectsInvalidPlanReviewGuidance(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-guidance.md")
	writeFile(t, path, mustRenderTemplate(t, "Invalid Plan Review Guidance"))
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-guidance/review-guidance/state-risk.md"), `---
name: state-risk
description: Invalid override metadata.
mode: override
---

Override the base guidance.
`)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected invalid plan guidance to fail lint, got %#v", result)
	}
	assertHasError(t, result, "supplements.review-guidance.state-risk.md")
}

func TestLintFileRejectsNestedPlanReviewGuidance(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-guidance.md")
	writeFile(t, path, mustRenderTemplate(t, "Nested Plan Review Guidance"))
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-guidance/review-guidance/nested/state-risk.md"), `---
name: state-risk
description: Nested guidance.
---

This nesting is not supported.
`)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected nested plan guidance to fail lint, got %#v", result)
	}
	assertHasError(t, result, "supplements.review-guidance.nested")
}

func TestLintFileRejectsPlanReviewGuidanceFilenameMismatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-guidance.md")
	writeFile(t, path, mustRenderTemplate(t, "Mismatched Plan Review Guidance"))
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-guidance/review-guidance/wrong-name.md"), `---
name: state-risk
description: Check the plan-specific state risk.
---

Verify the plan-specific state invariant.
`)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected mismatched guidance filename to fail lint, got %#v", result)
	}
	assertHasError(t, result, "supplements.review-guidance.wrong-name.md")
}

func TestLintFileRejectsConflictingSupplementsRootForSameStem(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Conflicting Supplements Root")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-has-supplements/spec.md"), "# active draft\n")
	writeFile(t, filepath.Join(root, "docs/plans/archived/supplements/2026-03-17-has-supplements/spec.md"), "# stale archived draft\n")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileRejectsConflictingSupplementsFileForSameStem(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-has-supplements.md")
	content := mustRenderTemplate(t, "Conflicting Supplements File")
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/archived/supplements/2026-03-17-has-supplements"), "stale file")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileAcceptsMatchingArchivedSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-has-supplements.md")
	content := makeArchiveReady(checkAllBoxes(strings.ReplaceAll(mustRenderTemplate(t, "Archived Matching Supplements Directory"), "- Done: [ ]", "- Done: [x]")))
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/archived/supplements/2026-03-17-has-supplements/spec.md"), "# archived draft\n")

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileRejectsArchivedPlanWhenSupplementsRemainInActiveRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-has-supplements.md")
	content := makeArchiveReady(checkAllBoxes(strings.ReplaceAll(mustRenderTemplate(t, "Archived Conflicting Supplements Root"), "- Done: [ ]", "- Done: [x]")))
	writeFile(t, path, content)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-17-has-supplements/spec.md"), "# stale active draft\n")

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "supplements")
}

func TestLintFileAcceptsTrackedActiveLightweightPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Lightweight Tracked Plan")
	content = strings.Replace(content, "size: M", "size: XXS", 1)
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected tracked lightweight lint success, got %#v", result)
	}
}

func TestLintFileRejectsGoalOrientedProfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-goal-oriented-plan.md")
	content := mustRenderTemplate(t, "Goal-Oriented Preview Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: goal_oriented", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected deferred goal-oriented profile to fail, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.workflow_profile")
}

func TestLintFileAcceptsArchivedLightweightLocalPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local/harness/plans/archived/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Archived Lightweight Plan")
	content = strings.Replace(content, "size: M", "size: XXS", 1)
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 3)
	content = strings.ReplaceAll(content, "- [ ]", "- [x]")
	content = makeArchiveReady(content)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected archived lightweight lint success, got %#v", result)
	}
}

func TestLintFileRejectsNonXXSLightweightPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-bad-lightweight-plan.md")
	content := mustRenderTemplate(t, "Bad Lightweight Plan")
	content = strings.Replace(content, "size: M", "size: XS", 1)
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected non-XXS lightweight plan to fail, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.size")
}

func TestLintFileRejectsLightweightActivePlanUnderLocalPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local/harness/plans/2026-03-17-lightweight-plan/active/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Bad Local Active Plan")
	content = strings.Replace(content, "size: M", "size: XXS", 1)
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "path")
}

func TestLintFileRejectsArchivedGoalOrientedPreviewPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-goal-oriented-plan.md")
	content := makeArchiveReady(checkAllBoxes(mustRenderTemplate(t, "Archived Goal-Oriented Plan")))
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: goal_oriented", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected archived goal-oriented preview lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.workflow_profile")
}

func TestLintFileRejectsUnsupportedWorkflowProfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-bad-profile.md")
	content := mustRenderTemplate(t, "Bad Profile Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: risky", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.workflow_profile")
}

func TestLintFileRejectsExplicitStandardProfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-standard-profile.md")
	content := mustRenderTemplate(t, "Explicit Standard Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: standard", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected explicit standard profile lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.workflow_profile")
}

func TestLintFileRejectsInvalidStepHeading(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Bad Step Heading")
	content = strings.Replace(content, "### Step 1: Replace with first step title", "### Step banana", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Work Breakdown")
}

func TestLintResultJSONRoundTrip(t *testing.T) {
	result := plan.LintFile(filepath.Join(t.TempDir(), "missing.md"))
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal lint result: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected JSON output")
	}
}

func mustRenderTemplate(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 17, 14, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		SourceType: "direct_request",
		Size:       "M",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	return rendered
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func checkAllBoxes(content string) string {
	content = strings.ReplaceAll(content, "- [ ]", "- [x]")
	return content
}

func makeArchiveReady(content string) string {
	content = strings.Replace(content, "## Closeout\n", "## Closeout\n\n- Archived At: 2026-03-17T15:00:00+08:00\n- Revision: 1\n", 1)
	content = strings.Replace(content, "- Validation: PENDING_UNTIL_ARCHIVE", "- Validation: Validated the planned slice.", 1)
	content = strings.Replace(content, "- Review: PENDING_UNTIL_ARCHIVE", "- Review: No unresolved blocking findings remain.", 1)
	content = strings.Replace(content, "- Delivered: PENDING_UNTIL_ARCHIVE", "- Delivered: Shipped the planned slice.", 1)
	content = strings.Replace(content, "- Not Delivered: PENDING_UNTIL_ARCHIVE", "- Not Delivered: NONE.", 1)
	content = strings.Replace(content, "- PR: PENDING_UNTIL_ARCHIVE", "- PR: NONE", 1)
	content = strings.Replace(content, "- Ready: PENDING_UNTIL_ARCHIVE", "- Ready: The candidate satisfies the acceptance criteria.", 1)
	content = strings.Replace(content, "- Merge Handoff: PENDING_UNTIL_ARCHIVE", "- Merge Handoff: Await explicit merge approval.", 1)
	return content
}

func assertHasError(t *testing.T, result plan.LintResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected error for %s, got %#v", path, result.Errors)
}
