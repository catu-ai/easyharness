package e2e_test

import (
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

const (
	runstateConcurrencyPlanTitle = "Runstate Concurrency Coverage Plan"
	runstateConcurrencyStepOne   = "Prepare the first archived candidate"
	runstateConcurrencyStepTwo   = "Reach merge-ready handoff before reopen"
)

func TestArchivedRunstateInterleavingsIgnoreStaleEvidenceAndFailClearlyUnderLock(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-04-11-runstate-concurrency-coverage.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", runstateConcurrencyPlanTitle,
		"--timestamp", "2026-04-11T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#56",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, runstateConcurrencyPlanTitle, runstateConcurrencyPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToAwaitMergeNode(t, workspace, planPath, runstateConcurrencyStepOne, runstateConcurrencyStepTwo)

	mergeReadyStatus := runStatus(t, workspace.Root)
	assertNode(t, mergeReadyStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen to bump revision to 2, got %#v", reopenPayload)
	}

	support.RewritePlanPreservingFrontmatter(t, planPath, runstateConcurrencyPlanTitle, runstateConcurrencyPlanBody())
	support.CompleteStep(t, planPath, 1)
	support.CompleteStep(t, planPath, 2)
	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/fix")
	if preFinalizeStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix repair cue before the fresh finalize review, got %#v", preFinalizeStatus)
	}

	runPassingFinalizeReview(t, workspace)
	support.CompleteCloseout(t, planPath)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" || archivePayload.Facts.Revision != 2 {
		t.Fatalf("expected second archive to preserve revision 2, got %#v", archivePayload)
	}

	postRearchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postRearchiveStatus, "execution/finalize/publish")
	if postRearchiveStatus.Facts.Evidence.Recorded.Publish.Status != "" ||
		postRearchiveStatus.Facts.Evidence.Recorded.CI.Status != "" ||
		postRearchiveStatus.Facts.Evidence.Recorded.Sync.Status != "" {
		t.Fatalf("expected revision-1 evidence to stay ignored after reopen, got %#v", postRearchiveStatus.Facts.Evidence.Recorded)
	}

	release, err := runstate.AcquireStateMutationLock(workspace.Root, "2026-04-11-runstate-concurrency-coverage")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}

	lockedStatus := support.Run(t, workspace.Root, "status")
	support.RequireExitCode(t, lockedStatus, 1)
	support.RequireNoStderr(t, lockedStatus)
	lockedStatusPayload := support.RequireJSONResult[statusResult](t, lockedStatus)
	if lockedStatusPayload.OK || lockedStatusPayload.Summary != "Another local state mutation is still in progress." {
		t.Fatalf("expected locked status failure, got %#v", lockedStatusPayload)
	}
	if lockedStatusPayload.Artifacts.ProjectRoot == "" || lockedStatusPayload.Artifacts.PlanPath != postRearchiveStatus.Artifacts.PlanPath {
		t.Fatalf("expected repo-facing locked status artifacts, got %#v", lockedStatusPayload.Artifacts)
	}

	lockedCIInput := workspace.WriteJSON(t, "tmp/locked-ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/rev2-locked",
	})
	lockedEvidence := support.Run(t, workspace.Root, "evidence", "submit", "--kind", "ci", "--input", lockedCIInput)
	support.RequireExitCode(t, lockedEvidence, 1)
	support.RequireNoStderr(t, lockedEvidence)
	lockedEvidencePayload := support.RequireJSONResult[evidenceSubmitResult](t, lockedEvidence)
	if lockedEvidencePayload.OK || lockedEvidencePayload.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("expected locked evidence-submit failure, got %#v", lockedEvidencePayload)
	}
	support.RequireFileMissing(t, workspace.Path(".local/harness/plans/2026-04-11-runstate-concurrency-coverage/evidence/ci/ci-002.json"))

	release()

	submitEvidence(t, workspace, "publish", "tmp/rev2-publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/156",
		"branch": "codex/runstate-concurrency-coverage",
		"base":   "main",
		"commit": "def456abc789",
	})
	submitEvidence(t, workspace, "ci", "tmp/rev2-ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/rev2",
	})
	submitEvidence(t, workspace, "sync", "tmp/rev2-sync.json", map[string]any{
		"status":   "fresh",
		"base_ref": "main",
		"head_ref": "codex/runstate-concurrency-coverage",
	})

	finalStatus := runStatus(t, workspace.Root)
	assertNode(t, finalStatus, "execution/finalize/await_merge")
	if finalStatus.Facts.Evidence.Recorded.Publish.Status != "recorded" ||
		finalStatus.Facts.Evidence.Recorded.CI.Status != "success" ||
		finalStatus.Facts.Evidence.Recorded.Sync.Status != "fresh" {
		t.Fatalf("expected revision-2 evidence to drive merge-ready status, got %#v", finalStatus.Facts.Evidence.Recorded)
	}
}

func runstateConcurrencyPlanBody() string {
	return compactPlanFixture(runstateConcurrencyStepOne, runstateConcurrencyStepTwo)
}
