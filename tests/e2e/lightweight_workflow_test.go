package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	lightweightWorkflowTitle = "Lightweight Workflow Plan"
	lightweightStepTitle     = "Update the lightweight workflow docs"
)

func TestLightweightWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-31-lightweight-workflow.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", lightweightWorkflowTitle,
		"--timestamp", "2026-03-31T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#69",
		"--lightweight",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RequireFileExists(t, planPath)

	support.RewritePlanPreservingFrontmatter(t, planPath, lightweightWorkflowTitle, lightweightWorkflowPlanBody())
	support.EnsurePlanSize(t, planPath, "XXS")

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")
	support.ApprovePlan(t, planPath, "2026-03-31T00:05:00Z")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q, got %#v", planRelPath, current)
	}

	initialStatus := runStatus(t, workspace.Root)
	assertNode(t, initialStatus, "execution/step-1/implement")
	if initialStatus.Facts.CurrentStep != trackedStepTitle(1, lightweightStepTitle) {
		t.Fatalf("expected current step %q, got %#v", trackedStepTitle(1, lightweightStepTitle), initialStatus)
	}

	support.CompleteStep(t, planPath, 1)
	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	runPassingFinalizeReview(t, workspace)
	support.CompleteCloseout(t, planPath)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", archivePayload)
	}
	assertLifecycleEnvelope(t, archivePayload, "execution/finalize/publish", 1)
	archivedRelPath := ".local/harness/plans/archived/2026-03-31-lightweight-workflow.md"
	if archivePayload.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected archived lightweight path %q, got %#v", archivedRelPath, archivePayload)
	}

	postArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postArchiveStatus, "execution/finalize/publish")
	if len(postArchiveStatus.NextAction) == 0 || !strings.Contains(postArchiveStatus.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance after lightweight archive, got %#v", postArchiveStatus.NextAction)
	}
	publishedHead := workspace.CommitAll(t, "commit lightweight active plan removal")

	submitEvidence(t, workspace, "publish", "tmp/lightweight-publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/109",
		"branch": "codex/e2e-lightweight-workflow",
		"base":   "main",
		"commit": publishedHead,
	})
	submitEvidence(t, workspace, "ci", "tmp/lightweight-ci.json", map[string]any{
		"status":   "not_applied",
		"provider": "github-actions",
		"reason":   "docs-only lightweight candidate",
	})
	submitEvidence(t, workspace, "sync", "tmp/lightweight-sync.json", map[string]any{
		"status": "not_applied",
		"reason": "no remote sync requirement in the test workspace",
	})

	awaitMergeStatus := runStatus(t, workspace.Root)
	assertNode(t, awaitMergeStatus, "execution/finalize/await_merge")
	if !strings.Contains(awaitMergeStatus.Summary, "lightweight path") {
		t.Fatalf("expected await-merge summary to mention lightweight breadcrumb, got %#v", awaitMergeStatus)
	}
	if len(awaitMergeStatus.NextAction) == 0 || !strings.Contains(awaitMergeStatus.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected await-merge breadcrumb guidance, got %#v", awaitMergeStatus.NextAction)
	}

	land := support.Run(t, workspace.Root, "land", "--pr", "https://github.com/catu-ai/easyharness/pull/109", "--commit", publishedHead)
	support.RequireSuccess(t, land)
	support.RequireNoStderr(t, land)
	landPayload := requireLifecycleResult(t, land)
	if !landPayload.OK || landPayload.Command != "land" {
		t.Fatalf("unexpected lightweight land payload: %#v", landPayload)
	}
	assertLifecycleEnvelope(t, landPayload, "land", 1)

	landComplete := support.Run(t, workspace.Root, "land", "complete")
	support.RequireSuccess(t, landComplete)
	support.RequireNoStderr(t, landComplete)
	landCompletePayload := requireLifecycleResult(t, landComplete)
	if !landCompletePayload.OK || landCompletePayload.Command != "land complete" {
		t.Fatalf("unexpected lightweight land-complete payload: %#v", landCompletePayload)
	}
	assertLifecycleEnvelope(t, landCompletePayload, "idle", 1)

	current = support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != "" || current.LastLandedPlanPath != archivedRelPath || current.LastLandedAt == "" {
		t.Fatalf("expected lightweight idle marker with last-landed context, got %#v", current)
	}
	postLandStatus := runStatus(t, workspace.Root)
	assertNode(t, postLandStatus, "idle")
	if postLandStatus.Artifacts.PlanPath != archivedRelPath {
		t.Fatalf("expected lightweight last-landed path in idle status, got %#v", postLandStatus)
	}
}

func lightweightWorkflowPlanBody() string {
	return compactPlanFixture(lightweightStepTitle)
}
