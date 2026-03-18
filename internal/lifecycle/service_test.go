package lifecycle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/lifecycle"
	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/runstate"
)

func TestArchiveMovesPlanAndUpdatesPointers(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath: activeRelPath,
		PlanStem: "2026-03-18-archive-smoke",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	result := svc.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}

	archivedPath := filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived path missing: %v", err)
	}
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("expected active path to be removed, got %v", err)
	}
	if lint := plan.LintFile(archivedPath); !lint.OK {
		t.Fatalf("archived plan should lint, got %#v", lint)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current-plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/archived/2026-03-18-archive-smoke.md" {
		t.Fatalf("unexpected current plan: %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.PlanPath != "docs/plans/archived/2026-03-18-archive-smoke.md" {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestArchiveRejectsMissingArchiveSummaryFields(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")
	content := buildActiveArchiveCandidate(t)
	content = strings.Replace(content, "- PR: NONE\n", "", 1)
	writeFile(t, path, content)

	svc := lifecycle.Service{Workdir: root}
	result := svc.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "section.Archive Summary")
}

func TestArchiveRejectsUnresolvedLocalState(t *testing.T) {
	testCases := []struct {
		name       string
		state      *runstate.State
		errorPath  string
		errorMatch string
	}{
		{
			name: "active review round",
			state: &runstate.State{
				ActiveReviewRound: &runstate.ReviewRound{RoundID: "review-001-full", Kind: "full", Aggregated: false},
			},
			errorPath:  "state.active_review_round",
			errorMatch: "aggregate or clear",
		},
		{
			name: "non-green ci",
			state: &runstate.State{
				LatestCI: &runstate.CIState{SnapshotID: "ci-1", Status: "pending"},
			},
			errorPath:  "state.latest_ci",
			errorMatch: "not archive-ready",
		},
		{
			name: "stale sync",
			state: &runstate.State{
				Sync: &runstate.SyncState{Freshness: "stale"},
			},
			errorPath:  "state.sync",
			errorMatch: "refresh remote state",
		},
		{
			name: "merge conflicts",
			state: &runstate.State{
				Sync: &runstate.SyncState{Freshness: "fresh", Conflicts: true},
			},
			errorPath:  "state.sync",
			errorMatch: "resolve merge conflicts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
			writeActiveArchiveCandidate(t, root, activeRelPath)
			tc.state.PlanPath = activeRelPath
			tc.state.PlanStem = "2026-03-18-archive-smoke"
			if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", tc.state); err != nil {
				t.Fatalf("save state: %v", err)
			}

			result := lifecycle.Service{Workdir: root}.Archive()
			if result.OK {
				t.Fatalf("expected archive failure, got %#v", result)
			}
			assertErrorPath(t, result.Errors, tc.errorPath)
			assertErrorContains(t, result.Errors, tc.errorPath, tc.errorMatch)
		})
	}
}

func TestReopenMovesArchivedPlanBackToActiveAndResetsSummaries(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen()
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	activePath := filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("reopened active path missing: %v", err)
	}
	if lint := plan.LintFile(activePath); !lint.OK {
		t.Fatalf("reopened active plan should lint, got %#v", lint)
	}
	data, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read reopened plan: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "revision: 2") {
		t.Fatalf("expected revision bump, got:\n%s", text)
	}
	if !strings.Contains(text, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE") {
		t.Fatalf("expected Archive Summary reset, got:\n%s", text)
	}
	if !strings.Contains(text, "### Follow-Up Issues\n\nNONE") {
		t.Fatalf("expected follow-up reset, got:\n%s", text)
	}
}

func TestReopenClearsStaleCIAndSyncSignals(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath: activeRelPath,
		PlanStem: "2026-03-18-archive-smoke",
		LatestCI: &runstate.CIState{SnapshotID: "ci-1", Status: "success"},
		Sync:     &runstate.SyncState{Freshness: "fresh", Conflicts: false},
	}); err != nil {
		t.Fatalf("save initial state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:          "docs/plans/archived/2026-03-18-archive-smoke.md",
		PlanStem:          "2026-03-18-archive-smoke",
		ActiveReviewRound: &runstate.ReviewRound{RoundID: "review-001-full", Kind: "full", Aggregated: true},
		LatestCI:          &runstate.CIState{SnapshotID: "ci-2", Status: "failed"},
		Sync:              &runstate.SyncState{Freshness: "stale", Conflicts: true},
	}); err != nil {
		t.Fatalf("save archived state: %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen()
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected reopened state")
	}
	if state.ActiveReviewRound != nil || state.LatestCI != nil || state.Sync != nil {
		t.Fatalf("expected reopened state to clear stale review/ci/sync signals, got %#v", state)
	}
}

func writeActiveArchiveCandidate(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	writeFile(t, path, buildActiveArchiveCandidate(t))
	return path
}

func buildActiveArchiveCandidate(t *testing.T) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Archive Smoke",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "lifecycle: awaiting_plan_approval", "lifecycle: executing", 1)
	rendered = strings.ReplaceAll(rendered, "- Status: pending", "- Status: completed")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_EXECUTION", "Completed execution notes.")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_REVIEW", "Completed review notes.")
	rendered = strings.Replace(rendered, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the implementation and command surfaces.", 1)
	rendered = strings.Replace(rendered, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nNo unresolved blocking review findings remain.", 1)
	rendered = strings.Replace(rendered, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.\n- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.", 1)
	rendered = strings.Replace(rendered, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the planned CLI slice.", 1)
	rendered = strings.Replace(rendered, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.", 1)
	return rendered
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func assertErrorPath(t *testing.T, issues []lifecycle.CommandError, path string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected error for %s, got %#v", path, issues)
}

func assertErrorContains(t *testing.T, issues []lifecycle.CommandError, path, fragment string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Path == path && strings.Contains(issue.Message, fragment) {
			return
		}
	}
	t.Fatalf("expected error for %s containing %q, got %#v", path, fragment, issues)
}
