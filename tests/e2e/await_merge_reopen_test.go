package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	awaitMergeFinalizeFixPlanTitle = "Await Merge Reopen Finalize Fix Plan"
	awaitMergeFinalizeFixStepOne   = "Prepare the merge-ready candidate"
	awaitMergeFinalizeFixStepTwo   = "Finish the archived handoff before reopen"

	awaitMergeNewStepPlanTitle = "Await Merge Reopen New Step Plan"
	awaitMergeNewStepStepOne   = "Prepare the merge-ready candidate"
	awaitMergeNewStepStepTwo   = "Finish the archived handoff before reopen"
	awaitMergeNewStepStepThree = "Handle merge-ready follow-up as a new step"
)

func TestAwaitMergeReopenFinalizeFixWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-await-merge-reopen-finalize-fix.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", awaitMergeFinalizeFixPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, awaitMergeFinalizeFixPlanTitle, awaitMergeReopenFinalizeFixPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")

	drivePlanToAwaitMergeNode(t, workspace, planPath, awaitMergeFinalizeFixStepOne, awaitMergeFinalizeFixStepTwo)

	preReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, preReopenStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "finalize-fix" || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected await-merge finalize-fix reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	postReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, postReopenStatus, "execution/finalize/fix")
	if postReopenStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen cue after await-merge reopen, got %#v", postReopenStatus)
	}
	if !strings.Contains(postReopenStatus.Summary, "finalize-scope repair") {
		t.Fatalf("expected finalize-fix summary after await-merge reopen, got %#v", postReopenStatus)
	}
}

func TestAwaitMergeReopenNewStepWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-await-merge-reopen-new-step.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", awaitMergeNewStepPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, awaitMergeNewStepPlanTitle, awaitMergeReopenNewStepPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToAwaitMergeNode(t, workspace, planPath, awaitMergeNewStepStepOne, awaitMergeNewStepStepTwo)

	preReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, preReopenStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "new-step")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "new-step" || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected await-merge new-step reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	pendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, pendingNewStepStatus, "execution/finalize/fix")
	if pendingNewStepStatus.Facts.ReopenMode != "new-step" {
		t.Fatalf("expected new-step reopen cue after await-merge reopen, got %#v", pendingNewStepStatus)
	}
	if !strings.Contains(pendingNewStepStatus.Summary, "needs a new unfinished step") {
		t.Fatalf("expected pending new-step summary after await-merge reopen, got %#v", pendingNewStepStatus)
	}

	support.AppendStepBeforeValidationStrategy(t, planPath, awaitMergeNewStepStepThreeBody())

	postAppendStatus := runStatus(t, workspace.Root)
	assertNode(t, postAppendStatus, "execution/step-3/implement")
	if postAppendStatus.Facts.CurrentStep != trackedStepTitle(3, awaitMergeNewStepStepThree) {
		t.Fatalf("expected newly added step 3 to become current after await-merge reopen, got %#v", postAppendStatus)
	}
	if postAppendStatus.Facts.ReopenMode != "" {
		t.Fatalf("expected consumed new-step cue to disappear once step 3 exists, got %#v", postAppendStatus)
	}
}

func awaitMergeReopenFinalizeFixPlanBody() string {
	return compactPlanFixture(awaitMergeFinalizeFixStepOne, awaitMergeFinalizeFixStepTwo)
}

func awaitMergeReopenNewStepPlanBody() string {
	return compactPlanFixture(awaitMergeNewStepStepOne, awaitMergeNewStepStepTwo)
}

func awaitMergeNewStepStepThreeBody() string {
	return `
### Step 3: Handle merge-ready follow-up as a new step

- Done: [ ]
- Outcome: Add the unfinished step required by the await-merge reopen.
- Covers: Criterion 1
- Check: Confirm status moves to this new current step after the append.
`
}
