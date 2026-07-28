package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestCoordinatedWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	rootRelPath := "docs/plans/active/2026-07-28-coordinated-workflow.md"
	rootPath := workspace.Path(rootRelPath)
	subplansRelDir := "docs/plans/active/supplements/2026-07-28-coordinated-workflow/subplans"
	apiRelPath := subplansRelDir + "/api.md"
	integrationRelPath := subplansRelDir + "/integration.md"

	rootTemplate := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--coordinated",
		"--title", "Coordinated Workflow",
		"--timestamp", "2026-07-28T00:00:00Z",
		"--size", "L",
		"--output", rootRelPath,
	)
	support.RequireSuccess(t, rootTemplate)
	support.RequireNoStderr(t, rootTemplate)

	apiTemplate := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--subplan",
		"--title", "Build API",
		"--output", apiRelPath,
	)
	support.RequireSuccess(t, apiTemplate)
	support.RequireNoStderr(t, apiTemplate)

	integrationTemplate := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--subplan",
		"--title", "Integrate Candidate",
		"--depends-on", "api",
		"--output", integrationRelPath,
	)
	support.RequireSuccess(t, integrationTemplate)
	support.RequireNoStderr(t, integrationTemplate)

	lint := support.Run(t, workspace.Root, "plan", "lint", rootRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	support.ApprovePlan(t, rootPath, "2026-07-28T00:05:00Z")
	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)
	executePayload := support.RequireJSONResult[executeStartResult](t, execute)
	if executePayload.State.CurrentNode != "execution/coordinate" {
		t.Fatalf("execute node = %q, want execution/coordinate", executePayload.State.CurrentNode)
	}

	coordinating := runStatus(t, workspace.Root)
	assertNode(t, coordinating, "execution/coordinate")
	if coordinating.Facts.Subplans.Total != 2 ||
		coordinating.Facts.Subplans.Completed != 0 ||
		coordinating.Facts.Subplans.Runnable != 1 ||
		coordinating.Facts.Subplans.Waiting != 1 {
		t.Fatalf("unexpected coordinated progress: %#v", coordinating.Facts.Subplans)
	}

	waitingCommand := support.Run(t, workspace.Root, "status", "--plan", "integration")
	support.RequireSuccess(t, waitingCommand)
	support.RequireNoStderr(t, waitingCommand)
	waiting := support.RequireJSONResult[statusResult](t, waitingCommand)
	assertNode(t, waiting, "execution/waiting")
	if waiting.Facts.SelectedSubplan.ID != "integration" ||
		len(waiting.Facts.SelectedSubplan.WaitingOn) != 1 ||
		waiting.Facts.SelectedSubplan.WaitingOn[0] != "api" {
		t.Fatalf("unexpected selected waiting subplan: %#v", waiting.Facts.SelectedSubplan)
	}

	completeSubplanFile(t, workspace.Path(apiRelPath))
	runnableCommand := support.Run(t, workspace.Root, "status", "--plan", integrationRelPath)
	support.RequireSuccess(t, runnableCommand)
	support.RequireNoStderr(t, runnableCommand)
	runnable := support.RequireJSONResult[statusResult](t, runnableCommand)
	assertNode(t, runnable, "execution/step-1/implement")

	completeSubplanFile(t, workspace.Path(integrationRelPath))
	support.CheckAllAcceptanceCriteria(t, rootPath)
	preReview := runStatus(t, workspace.Root)
	assertNode(t, preReview, "execution/finalize/review")
	if preReview.Facts.Subplans.Completed != 2 {
		t.Fatalf("expected both subplans complete before review, got %#v", preReview.Facts.Subplans)
	}

	runPassingFinalizeReview(t, workspace)
	support.CompleteCloseout(t, rootPath)
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	archivedRootRelPath := "docs/plans/archived/2026-07-28-coordinated-workflow.md"
	if archivePayload.Artifacts.ToPlanPath != archivedRootRelPath {
		t.Fatalf("unexpected archived root: %#v", archivePayload.Artifacts)
	}
	archivedSubplansDir := workspace.Path("docs/plans/archived/supplements/2026-07-28-coordinated-workflow/subplans")
	support.RequireFileExists(t, filepath.Join(archivedSubplansDir, "api.md"))
	support.RequireFileExists(t, filepath.Join(archivedSubplansDir, "integration.md"))
	if _, err := os.Stat(workspace.Path(subplansRelDir)); !os.IsNotExist(err) {
		t.Fatalf("active subplan package should move on archive, got %v", err)
	}

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "new-step")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if reopenPayload.State.CurrentNode != "execution/coordinate" {
		t.Fatalf("coordinated reopen node = %q, want execution/coordinate", reopenPayload.State.CurrentNode)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/coordinate")
	support.RequireFileExists(t, workspace.Path(apiRelPath))
	support.RequireFileExists(t, workspace.Path(integrationRelPath))

	followUpRelPath := subplansRelDir + "/follow-up.md"
	followUpTemplate := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--subplan",
		"--title", "Reopened Follow-up",
		"--output", followUpRelPath,
	)
	support.RequireSuccess(t, followUpTemplate)
	support.RequireNoStderr(t, followUpTemplate)
	completeSubplanFile(t, workspace.Path(followUpRelPath))
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/review")
}

func completeSubplanFile(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read subplan %s: %v", path, err)
	}
	updated := strings.ReplaceAll(string(content), "- Done: [ ]", "- Done: [x]")
	updated = strings.Replace(updated, "- Validation: PENDING", "- Validation: Focused checks passed.", 1)
	updated = strings.Replace(updated, "- Delivered: PENDING", "- Delivered: The subplan outcome is integrated.", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write completed subplan %s: %v", path, err)
	}
}
