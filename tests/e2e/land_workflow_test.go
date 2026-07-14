package e2e_test

import (
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	landWorkflowPlanTitle = "Land Workflow Plan"
	landStepOneTitle      = "Prepare the archived candidate for land"
	landStepTwoTitle      = "Finish merge-ready handoff setup"
)

func TestLandWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-land-workflow.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", landWorkflowPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, landWorkflowPlanTitle, landWorkflowPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToArchivedPublishNode(t, workspace, planPath, landStepOneTitle, landStepTwoTitle)

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
		"branch": "codex/e2e-lifecycle-handoff-coverage",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/2",
	})
	submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status": "not_applied",
		"reason": "fixture does not model a remote base comparison",
	})

	preLandStatus := runStatus(t, workspace.Root)
	assertNode(t, preLandStatus, "execution/finalize/await_merge")

	land := support.Run(t, workspace.Root, "land", "--pr", "https://github.com/catu-ai/easyharness/pull/99", "--commit", "abc123")
	support.RequireSuccess(t, land)
	support.RequireNoStderr(t, land)
	landPayload := requireLifecycleResult(t, land)
	if !landPayload.OK || landPayload.Command != "land" {
		t.Fatalf("unexpected land payload: %#v", landPayload)
	}
	assertLifecycleEnvelope(t, landPayload, "land", 1)
	if landPayload.Facts.LandPRURL != "https://github.com/catu-ai/easyharness/pull/99" || landPayload.Facts.LandCommit != "abc123" {
		t.Fatalf("expected land facts in payload, got %#v", landPayload)
	}

	inLandStatus := runStatus(t, workspace.Root)
	assertNode(t, inLandStatus, "land")
	if inLandStatus.Facts.LandPRURL != "https://github.com/catu-ai/easyharness/pull/99" {
		t.Fatalf("expected land PR URL in status, got %#v", inLandStatus)
	}
	if len(inLandStatus.NextAction) < 2 || inLandStatus.NextAction[1].Command == nil || *inLandStatus.NextAction[1].Command != "harness land complete" {
		t.Fatalf("expected land-complete guidance in land status, got %#v", inLandStatus)
	}

	stillInLandStatus := runStatus(t, workspace.Root)
	assertNode(t, stillInLandStatus, "land")

	landComplete := support.Run(t, workspace.Root, "land", "complete")
	support.RequireSuccess(t, landComplete)
	support.RequireNoStderr(t, landComplete)
	landCompletePayload := requireLifecycleResult(t, landComplete)
	if !landCompletePayload.OK || landCompletePayload.Command != "land complete" {
		t.Fatalf("unexpected land-complete payload: %#v", landCompletePayload)
	}
	assertLifecycleEnvelope(t, landCompletePayload, "idle", 1)

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != "" || current.LastLandedPlanPath != "docs/plans/archived/2026-03-23-land-workflow.md" || current.LastLandedAt == "" {
		t.Fatalf("expected idle marker with last-landed context, got %#v", current)
	}

	postLandStatus := runStatus(t, workspace.Root)
	assertNode(t, postLandStatus, "idle")
	if postLandStatus.Artifacts.PlanPath != "docs/plans/archived/2026-03-23-land-workflow.md" {
		t.Fatalf("expected last-landed path in idle status, got %#v", postLandStatus)
	}
}

func landWorkflowPlanBody() string {
	return compactPlanFixture(landStepOneTitle, landStepTwoTitle)
}
