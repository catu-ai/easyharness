package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	customPathRootsTitle = "Custom Path Roots Plan"
	customPathStepTitle  = "Wire the custom roots"
)

func TestCustomPathRootsWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	workspace.WriteFile(t, ".harness/config.yaml", []byte(`version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness-runtime
`))

	planRelPath := "workflow/plans/open/2026-06-07-custom-path-roots.md"
	planPath := workspace.Path(planRelPath)
	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", customPathRootsTitle,
		"--timestamp", "2026-06-07T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#229",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RequireFileExists(t, planPath)

	support.RewritePlanPreservingFrontmatterWithSize(t, planPath, customPathRootsTitle, customPathRootsPlanBody(), "M")

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")
	if preExecuteStatus.Artifacts.PlanPath != planRelPath {
		t.Fatalf("expected custom active plan path %q, got %#v", planRelPath, preExecuteStatus)
	}

	support.ApprovePlan(t, planPath, "2026-06-07T00:05:00Z")
	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)
	executePayload := requireExecuteStartResult(t, execute)
	expectedStateRelPath := "tmp/harness-runtime/plans/2026-06-07-custom-path-roots/state.json"
	if !strings.HasSuffix(filepath.ToSlash(executePayload.Artifacts.LocalStatePath), expectedStateRelPath) {
		t.Fatalf("expected local state path to end with %q, got %#v", expectedStateRelPath, executePayload)
	}
	support.RequireFileExists(t, workspace.Path(expectedStateRelPath))

	current := support.ReadJSONFile[currentPlan](t, workspace.Path("tmp/harness-runtime/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q, got %#v", planRelPath, current)
	}

	initialStatus := runStatus(t, workspace.Root)
	assertNode(t, initialStatus, "execution/step-1/implement")

	roundID := runPassingDeltaReview(t, workspace, customPathStepTitle, 1)
	reviewManifest := filepath.Join("tmp/harness-runtime/plans/2026-06-07-custom-path-roots/reviews", roundID, "manifest.json")
	support.RequireFileExists(t, workspace.Path(reviewManifest))

	support.CompleteStep(
		t,
		planPath,
		1,
		"Implemented the configured-root workflow fixture.",
		"Clean delta review "+roundID+" passed for the custom-root fixture.",
	)
	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")
	runPassingFinalizeReview(t, workspace)

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	archivedRelPath := "workflow/plans/done/2026-06-07-custom-path-roots.md"
	if archivePayload.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected custom archived path %q, got %#v", archivedRelPath, archivePayload)
	}
	support.RequireFileExists(t, workspace.Path(archivedRelPath))

	postArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postArchiveStatus, "execution/finalize/publish")
	if postArchiveStatus.Artifacts.PlanPath != archivedRelPath {
		t.Fatalf("expected archived status path %q, got %#v", archivedRelPath, postArchiveStatus)
	}

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/229",
		"branch": "codex/custom-harness-path-roots",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/custom-roots",
	})
	submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status":   "fresh",
		"base_ref": "main",
		"head_ref": "codex/custom-harness-path-roots",
	})
	support.RequireFileExists(t, workspace.Path("tmp/harness-runtime/plans/2026-06-07-custom-path-roots/evidence/publish/publish-001.json"))
	support.RequireFileExists(t, workspace.Path("tmp/harness-runtime/plans/2026-06-07-custom-path-roots/evidence/ci/ci-001.json"))
	support.RequireFileExists(t, workspace.Path("tmp/harness-runtime/plans/2026-06-07-custom-path-roots/evidence/sync/sync-001.json"))

	awaitMergeStatus := runStatus(t, workspace.Root)
	assertNode(t, awaitMergeStatus, "execution/finalize/await_merge")
}

func TestCustomPathRootsReopenWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	workspace.WriteFile(t, ".harness/config.yaml", []byte(`version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness-runtime
`))

	planRelPath := "workflow/plans/open/2026-06-07-custom-path-roots-reopen.md"
	planPath := workspace.Path(planRelPath)
	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", customPathRootsTitle,
		"--timestamp", "2026-06-07T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#229",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatterWithSize(t, planPath, customPathRootsTitle, customPathRootsPlanBody(), "M")

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	archivePayload := drivePlanToArchivedPublishNode(t, workspace, planPath, customPathStepTitle)
	archivedRelPath := "workflow/plans/done/2026-06-07-custom-path-roots-reopen.md"
	if archivePayload.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected custom archived path %q, got %#v", archivedRelPath, archivePayload)
	}

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" ||
		reopenPayload.Facts.Revision != 2 ||
		reopenPayload.Facts.ReopenMode != "finalize-fix" ||
		reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected finalize-fix reopen to restore custom active path, got %#v", reopenPayload)
	}

	current := support.ReadJSONFile[currentPlan](t, workspace.Path("tmp/harness-runtime/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected custom current-plan pointer %q after reopen, got %#v", planRelPath, current)
	}
}

func customPathRootsPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise configurable harness path roots through the built binary.

## Scope

### In Scope

- Use configured active, archived, and runtime roots.

### Out of Scope

- Dashboard behavior.

## Acceptance Criteria

- [ ] The custom-root workflow reaches merge-ready handoff.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Wire the custom roots

- Done: [ ]

#### Objective

Drive a standard plan through custom configured roots.

#### Details

NONE

#### Expected Files

- workflow/plans/open/2026-06-07-custom-path-roots.md

#### Validation

- Built-binary e2e coverage.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run the built binary through status, review, archive, and evidence commands.

## Risks

- Risk: Runtime artifacts might still land in the default root.
  - Mitigation: Assert current-plan, state, review, and evidence paths.

## Validation Summary

Custom-root e2e validation is provided by this test.

## Review Summary

Custom-root e2e validation submits and aggregates review artifacts.

## Archive Summary

- PR: test fixture
- Ready: archive-ready test fixture
- Merge Handoff: test fixture records evidence and reaches await_merge

## Outcome Summary

### Delivered

Built-binary custom-root workflow coverage.

### Not Delivered

NONE

### Follow-Up Issues

NONE
`)
}
