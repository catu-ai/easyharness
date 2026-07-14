package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	publishHandoffPlanTitle = "Publish Handoff Plan"
	publishStepOneTitle     = "Prepare the archived candidate"
	publishStepTwoTitle     = "Finish the branch before publish handoff"
)

func TestPublishHandoffWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-publish-handoff.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", publishHandoffPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, publishHandoffPlanTitle, publishHandoffPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	archivePayload := drivePlanToArchivedPublishNode(t, workspace, planPath, publishStepOneTitle, publishStepTwoTitle)
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-publish-handoff.md" {
		t.Fatalf("expected archived publish-handoff path, got %#v", archivePayload)
	}
	archivedHead := workspace.CommitAll(t, "commit mechanical archive move")
	archivedBase := resolveWorkspaceRevision(t, workspace.Root, archivedHead+"^")

	publish := submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
		"branch": "codex/e2e-lifecycle-handoff-coverage",
		"base":   "main",
	})
	if publish.Artifacts.Kind != "publish" {
		t.Fatalf("expected publish evidence artifacts, got %#v", publish)
	}

	postPublishStatus := runStatus(t, workspace.Root)
	assertNode(t, postPublishStatus, "execution/finalize/publish")
	if postPublishStatus.Facts.Evidence.Recorded.Publish.Status != "recorded" {
		t.Fatalf("expected publish status after publish evidence, got %#v", postPublishStatus)
	}
	if postPublishStatus.Facts.Evidence.Recorded.Publish.PRURL != "https://github.com/catu-ai/easyharness/pull/99" {
		t.Fatalf("expected publish PR URL in status, got %#v", postPublishStatus)
	}

	ci := submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/1",
	})
	if ci.Artifacts.Kind != "ci" {
		t.Fatalf("expected ci evidence artifacts, got %#v", ci)
	}

	postCIStatus := runStatus(t, workspace.Root)
	assertNode(t, postCIStatus, "execution/finalize/publish")
	if postCIStatus.Facts.Evidence.Recorded.CI.Status != "success" {
		t.Fatalf("expected CI success to remain in publish until sync exists, got %#v", postCIStatus)
	}

	sync := submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status":      "fresh",
		"base_ref":    "main",
		"head_ref":    "codex/e2e-lifecycle-handoff-coverage",
		"base_commit": archivedBase,
		"head_commit": archivedHead,
		"pr_url":      "https://github.com/catu-ai/easyharness/pull/99",
	})
	if sync.Artifacts.Kind != "sync" {
		t.Fatalf("expected sync evidence artifacts, got %#v", sync)
	}

	postSyncStatus := runStatus(t, workspace.Root)
	assertNode(t, postSyncStatus, "execution/finalize/await_merge")
	if postSyncStatus.Facts.Evidence.Recorded.Publish.Status != "recorded" ||
		postSyncStatus.Facts.Evidence.Recorded.CI.Status != "success" ||
		postSyncStatus.Facts.Evidence.Recorded.Sync.Status != "fresh" {
		t.Fatalf("expected merge-ready evidence facts after sync, got %#v", postSyncStatus)
	}
}

func TestPostArchiveProductCommitCannotBecomeMergeReadyWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-post-archive-coverage.md"
	planPath := workspace.Path(planRelPath)
	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", "Post Archive Coverage Plan",
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "direct_request",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, "Post Archive Coverage Plan", publishHandoffPlanBody())

	drivePlanToArchivedPublishNode(t, workspace, planPath, publishStepOneTitle, publishStepTwoTitle)
	workspace.CommitAll(t, "commit mechanical archive move")
	workspace.WriteFile(t, "product.go", []byte("package product\n\nconst Unreviewed = true\n"))
	unreviewedHead := workspace.CommitAll(t, "unreviewed post-archive product change")
	reviewedBase := resolveWorkspaceRevision(t, workspace.Root, unreviewedHead+"^^")

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
		"branch": "codex/post-archive-coverage",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status": "success", "provider": "github-actions",
	})
	submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status": "fresh", "base_ref": "main", "head_ref": "codex/post-archive-coverage",
		"base_commit": reviewedBase, "head_commit": unreviewedHead,
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
	})

	result := runStatus(t, workspace.Root)
	assertNode(t, result, "execution/finalize/publish")
	if len(result.Blockers) != 1 || result.Blockers[0].Path != "review.coverage" || !strings.Contains(result.Blockers[0].Message, "product.go") {
		t.Fatalf("expected post-archive product change coverage blocker, got %#v", result.Blockers)
	}
	if len(result.NextAction) != 1 || !strings.Contains(result.NextAction[0].Description, "harness reopen --mode finalize-fix") {
		t.Fatalf("expected reopen-and-review guidance, got %#v", result.NextAction)
	}
}

func publishHandoffPlanBody() string {
	return compactPlanFixture(publishStepOneTitle, publishStepTwoTitle)
}
