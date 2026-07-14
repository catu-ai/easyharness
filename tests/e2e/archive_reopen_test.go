package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	archiveReopenPlanTitle = "Archive Reopen Finalize Fix Plan"
	archiveReopenStepOne   = "Prepare the archived candidate"
	archiveReopenStepTwo   = "Finish the original branch candidate"
)

func TestArchiveReopenFinalizeFixWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-archive-reopen-finalize-fix.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", archiveReopenPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, archiveReopenPlanTitle, archiveReopenPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	archivePayload := drivePlanToArchivedPublishNode(t, workspace, planPath, archiveReopenStepOne, archiveReopenStepTwo)
	assertLifecycleEnvelope(t, archivePayload, "execution/finalize/publish", 1)
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-archive-reopen-finalize-fix.md" {
		t.Fatalf("expected archived path in archive payload, got %#v", archivePayload)
	}

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "finalize-fix" || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected finalize-fix reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	postReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, postReopenStatus, "execution/finalize/fix")
	if postReopenStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen cue after reopen, got %#v", postReopenStatus)
	}
	if !strings.Contains(postReopenStatus.Summary, "finalize-scope repair") {
		t.Fatalf("expected finalize-fix summary after reopen, got %#v", postReopenStatus)
	}

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q after finalize-fix reopen, got %#v", planRelPath, current)
	}

	statePath := workspace.Path(".local/harness/plans/2026-03-23-archive-reopen-finalize-fix/state.json")
	state := support.ReadJSONFile[runState](t, statePath)
	if state.FinalizeCoverage.TipRoundID == "" || state.FinalizeCoverage.CoveredHeadSHA == "" || state.FinalizeCoverage.Revision != 1 {
		t.Fatalf("expected reopen to preserve prior finalize coverage, got %#v", state.FinalizeCoverage)
	}

	workspace.WriteFile(t, "candidate/reopened-narrow-fix.txt", []byte("narrow revision-two repair\n"))
	start := startReviewRound(t, workspace, false)
	if !strings.HasSuffix(start.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected reopen repair to infer a linked delta, got %#v", start)
	}
	submission := submitReview(t, workspace, start.Artifacts.RoundID, "integrated-reviewer", "The reopened narrow delta is clean.", nil, nil)
	if submission.Review.Decision != "pass" {
		t.Fatalf("expected reopened narrow delta to extend prior full coverage, got %#v", submission)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/archive")
}

func archiveReopenPlanBody() string {
	return compactPlanFixture(archiveReopenStepOne, archiveReopenStepTwo)
}
