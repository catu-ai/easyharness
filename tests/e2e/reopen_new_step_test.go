package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	reopenNewStepPlanTitle = "Reopen New Step Plan"
	reopenStepOneTitle     = "Finish the first tracked slice"
	reopenStepTwoTitle     = "Finish the second tracked slice"
	reopenStepThreeTitle   = "Finish the original branch candidate"
	reopenStepFourTitle    = "Handle reopened follow-up work"
)

func TestReopenNewStepWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-reopen-new-step.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", reopenNewStepPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, reopenNewStepPlanTitle, reopenNewStepPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)
	support.ApprovePlan(t, planPath, "2026-03-23T00:05:00Z")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	for index := range []string{reopenStepOneTitle, reopenStepTwoTitle, reopenStepThreeTitle} {
		support.CompleteStep(t, planPath, index+1)
	}

	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	passingFinalizeRound := runPassingFinalizeReview(t, workspace)
	support.CompleteCloseout(t, planPath)
	postFinalizeReview := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeReview, "execution/finalize/archive")
	if len(postFinalizeReview.NextAction) == 0 || postFinalizeReview.NextAction[0].Description == "" {
		t.Fatalf("expected archive guidance after %s, got %#v", passingFinalizeRound, postFinalizeReview)
	}

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", archivePayload)
	}
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-reopen-new-step.md" {
		t.Fatalf("expected archived plan path in archive payload, got %#v", archivePayload)
	}

	archivedStatus := runStatus(t, workspace.Root)
	assertNode(t, archivedStatus, "execution/finalize/publish")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "new-step")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "new-step" || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	pendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, pendingNewStepStatus, "execution/finalize/fix")
	if pendingNewStepStatus.Facts.ReopenMode != "new-step" {
		t.Fatalf("expected new-step reopen cue before the new step exists, got %#v", pendingNewStepStatus)
	}
	if !strings.Contains(pendingNewStepStatus.Summary, "needs a new unfinished step") {
		t.Fatalf("expected pending new-step summary, got %#v", pendingNewStepStatus)
	}
	if len(pendingNewStepStatus.NextAction) == 0 || !strings.Contains(pendingNewStepStatus.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected explicit add-step guidance after reopen, got %#v", pendingNewStepStatus)
	}
	stillPendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, stillPendingNewStepStatus, "execution/finalize/fix")

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q after reopen, got %#v", planRelPath, current)
	}

	support.AppendStepBeforeValidationStrategy(t, planPath, reopenedStepFourBody())

	postAppendStatus := runStatus(t, workspace.Root)
	assertNode(t, postAppendStatus, "execution/step-4/implement")
	if postAppendStatus.Facts.CurrentStep != trackedStepTitle(4, reopenStepFourTitle) {
		t.Fatalf("expected newly added step 4 to become current, got %#v", postAppendStatus)
	}
	if postAppendStatus.Facts.ReopenMode != "" {
		t.Fatalf("expected consumed new-step cue to disappear once step 4 exists, got %#v", postAppendStatus)
	}

	stillImplementing := runStatus(t, workspace.Root)
	assertNode(t, stillImplementing, "execution/step-4/implement")

	support.CompleteStep(t, planPath, 4)

	postFourthStepStatus := runStatus(t, workspace.Root)
	assertNode(t, postFourthStepStatus, "execution/finalize/review")
}

func reopenNewStepPlanBody() string {
	return compactPlanFixture(reopenStepOneTitle, reopenStepTwoTitle, reopenStepThreeTitle)
}

func reopenedStepFourBody() string {
	return strings.TrimSpace(fmt.Sprintf(`
### Step 4: %s

- Done: [ ]
- Outcome: Represent the reopened follow-up as a new tracked step.
- Covers: Criterion 1
- Check: Verify status selects the new unfinished step.
`, reopenStepFourTitle))
}
