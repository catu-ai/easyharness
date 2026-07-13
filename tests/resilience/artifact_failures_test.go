package resilience_test

import (
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

func TestStatusIgnoresMalformedHistoricalStepReviewArtifactsWithoutActiveBinding(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/active/2026-04-11-resilience-review-artifacts.md"
	writePlanFixture(t, workspace, relPlanPath, "Resilience Review Artifacts", func(content string) string {
		return completeFirstStep(content)
	})
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-review-artifacts", &runstate.State{
		ExecutionStartedAt: "2026-04-11T13:10:00Z",
	})
	writeReviewManifest(t, workspace, "2026-04-11-resilience-review-artifacts", "review-001-delta", map[string]any{
		"review_title": "Step 1: Replace with first step title",
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, workspace, "2026-04-11-resilience-review-artifacts", "review-001-delta", map[string]any{
		"decision": "pass",
	})
	workspace.WriteFile(t, ".local/harness/plans/2026-04-11-resilience-review-artifacts/reviews/review-002-delta/manifest.json", []byte("{not-json"))
	writeReviewAggregate(t, workspace, "2026-04-11-resilience-review-artifacts", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[statusResult](t, result)
	if parsed.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to remain stable, got %#v", parsed)
	}
	if len(parsed.Warnings) != 0 {
		t.Fatalf("expected inactive historical artifacts to create no workflow debt, got %#v", parsed.Warnings)
	}
}

func TestStatusDoesNotTreatMalformedEvidenceAsMergeReady(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/archived/2026-04-11-resilience-evidence-artifacts.md"
	writeArchivedArchiveCandidate(t, workspace, relPlanPath)
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-evidence-artifacts", &runstate.State{
		Revision: 1,
	})
	writePublishRecord(t, workspace, "2026-04-11-resilience-evidence-artifacts", relPlanPath, "publish-001", "https://github.com/catu-ai/easyharness/pull/201", 1)
	writeCIRecord(t, workspace, "2026-04-11-resilience-evidence-artifacts", relPlanPath, "ci-001", "success", 1)
	workspace.WriteFile(t, ".local/harness/plans/2026-04-11-resilience-evidence-artifacts/evidence/sync/sync-001.json", []byte("{not-json"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[statusResult](t, result)
	if parsed.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected malformed evidence to keep publish node, got %#v", parsed)
	}
	if !findWarning(parsed.Warnings, "Unable to read sync evidence") {
		t.Fatalf("expected sync-evidence warning, got %#v", parsed.Warnings)
	}
}
