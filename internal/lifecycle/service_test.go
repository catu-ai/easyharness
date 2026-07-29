package lifecycle_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
)

func TestArchiveMovesPlanAndUpdatesPointers(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	activeBytes, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read active plan: %v", err)
	}
	writeFile(t, activePath, strings.Replace(string(activeBytes),
		"- Validation: Validated the implementation and command surfaces.",
		"- Validation: Validated the implementation,\n  including command and archive surfaces.", 1))
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
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
	archivedBytes, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("read archived plan: %v", err)
	}
	if !strings.Contains(string(archivedBytes), "size: M") {
		t.Fatalf("expected archived plan to preserve size frontmatter, got:\n%s", archivedBytes)
	}
	if !strings.Contains(string(archivedBytes), "- Validation: Validated the implementation,\n  including command and archive surfaces.") {
		t.Fatalf("expected archive to preserve Closeout continuation prose, got:\n%s", archivedBytes)
	}
	if !strings.HasSuffix(string(archivedBytes), "\n") || strings.HasSuffix(string(archivedBytes), "\n\n") {
		t.Fatalf("expected archived plan to end with exactly one newline, got %q", archivedBytes[len(archivedBytes)-4:])
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current-plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/archived/2026-03-18-archive-smoke.md" {
		t.Fatalf("unexpected current plan: %#v", current)
	}
	if result.Artifacts == nil || result.Artifacts.FromSupplementsPath != "" || result.Artifacts.ToSupplementsPath != "" {
		t.Fatalf("expected no supplements artifacts for markdown-only archive, got %#v", result.Artifacts)
	}
	if !containsActionDescription(result.NextAction, "Wait for explicit human merge approval only after status reaches `execution/finalize/await_merge`") {
		t.Fatalf("expected explicit human-approval archive guidance, got %#v", result.NextAction)
	}
	if !containsActionDescription(result.NextAction, "record publish evidence for the archived candidate while it remains in `execution/finalize/publish`") {
		t.Fatalf("expected publish-phase archive guidance, got %#v", result.NextAction)
	}
	if !containsActionDescription(result.NextAction, "Record CI and sync evidence before expecting the candidate to advance to `execution/finalize/await_merge`") {
		t.Fatalf("expected CI/sync archive guidance, got %#v", result.NextAction)
	}
	if containsActionDescription(result.NextAction, "merge manually from the PR once checks are green") {
		t.Fatalf("unexpected archive guidance that permits merge before approval: %#v", result.NextAction)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil {
		t.Fatalf("unexpected state: %#v", state)
	}
	assertRawStateJSONOmitsKeys(t, filepath.Join(root, ".local", "harness", "plans", "2026-03-18-archive-smoke", "state.json"), "current_node", "plan_path", "plan_stem")
}

func TestArchiveMovesSupplementsDirectoryWithPlanPackage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	activeSupplements := filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke/spec.md")
	writeFile(t, activeSupplements, "# draft spec\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}

	archivedSupplements := filepath.Join(root, "docs/plans/archived/supplements/2026-03-18-archive-smoke/spec.md")
	if _, err := os.Stat(archivedSupplements); err != nil {
		t.Fatalf("expected archived supplements to move, got %v", err)
	}
	if _, err := os.Stat(filepath.Dir(activeSupplements)); !os.IsNotExist(err) {
		t.Fatalf("expected active supplements directory to be removed, got %v", err)
	}
	if result.Artifacts == nil || result.Artifacts.FromSupplementsPath != "docs/plans/active/supplements/2026-03-18-archive-smoke" || result.Artifacts.ToSupplementsPath != "docs/plans/archived/supplements/2026-03-18-archive-smoke" {
		t.Fatalf("unexpected supplements artifacts: %#v", result.Artifacts)
	}
}

func TestArchiveLightweightMovesLocalPlanAndPromptsBreadcrumb(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight.md"
	activePath := writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-lightweight", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}

	archivedRelPath := ".local/harness/plans/archived/2026-03-18-lightweight.md"
	archivedPath := filepath.Join(root, archivedRelPath)
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected local archived path, got %v", err)
	}
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("expected tracked active path to be removed, got %v", err)
	}
	if result.Artifacts == nil || result.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected archived artifact path %q, got %#v", archivedRelPath, result.Artifacts)
	}
	if result.Artifacts.FromSupplementsPath != "" || result.Artifacts.ToSupplementsPath != "" {
		t.Fatalf("expected no supplements artifacts for lightweight archive without supplements, got %#v", result.Artifacts)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance first, got %#v", result.NextAction)
	}
	if !containsActionDescription(result.NextAction, "tracked active-plan removal") {
		t.Fatalf("expected lightweight archive guidance to mention the tracked active-plan removal, got %#v", result.NextAction)
	}
}

func TestArchiveLightweightMovesSupplementsIntoLocalArchivePackage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight.md"
	writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-18-lightweight/spec.md"), "# lightweight draft\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-lightweight", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}
	archivedSupplements := filepath.Join(root, ".local/harness/plans/archived/supplements/2026-03-18-lightweight/spec.md")
	if _, err := os.Stat(archivedSupplements); err != nil {
		t.Fatalf("expected lightweight archived supplements path, got %v", err)
	}
	if result.Artifacts == nil || result.Artifacts.ToSupplementsPath != ".local/harness/plans/archived/supplements/2026-03-18-lightweight" {
		t.Fatalf("unexpected lightweight supplements artifacts: %#v", result.Artifacts)
	}
}

func TestExecuteStartPersistsMilestoneAndPointer(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-smoke.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Smoke"))

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}.ExecuteStart()
	if !result.OK {
		t.Fatalf("expected execute start success, got %#v", result)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("unexpected current plan pointer: %#v", current)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:00:00Z" {
		t.Fatalf("expected execution-start milestone, got %#v", state)
	}
}

func TestPlanApproveRecordsAndRefreshesApprovedAt(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-plan-approve.md"
	planPath := filepath.Join(root, activeRelPath)
	writeFile(t, planPath, stripApprovedAt(buildAwaitingPlan(t, "Plan Approve Refresh")))

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	first := svc.PlanApprove("human")
	if !first.OK {
		t.Fatalf("expected first plan approve success, got %#v", first)
	}
	firstDoc, err := plan.LoadFile(planPath)
	if err != nil {
		t.Fatalf("load plan after first approval: %v", err)
	}
	if firstDoc.Frontmatter.ApprovedAt != "2026-03-18T01:00:00Z" {
		t.Fatalf("expected first approved_at timestamp, got %#v", firstDoc.Frontmatter)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
	}
	second := svc.PlanApprove("human")
	if !second.OK {
		t.Fatalf("expected second plan approve success, got %#v", second)
	}
	if !strings.Contains(second.Summary, "Refreshed") {
		t.Fatalf("expected refreshed approval summary, got %#v", second)
	}
	secondDoc, err := plan.LoadFile(planPath)
	if err != nil {
		t.Fatalf("load plan after second approval: %v", err)
	}
	if secondDoc.Frontmatter.ApprovedAt != "2026-03-18T01:10:00Z" {
		t.Fatalf("expected refreshed approved_at timestamp, got %#v", secondDoc.Frontmatter)
	}
}

func TestExecuteStartRejectsUnapprovedPlan(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-unapproved.md"
	writeFile(t, filepath.Join(root, activeRelPath), stripApprovedAt(buildAwaitingPlan(t, "Execute Start Unapproved")))

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start to reject unapproved plan, got %#v", result)
	}
	if result.Summary != "Current plan is waiting for recorded human approval." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) == 0 || result.Errors[0].Path != "frontmatter.approved_at" {
		t.Fatalf("expected approval error path, got %#v", result.Errors)
	}
}

func TestExecuteStartBackfillsLegacyExecutingPlan(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-legacy.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Legacy Execute Start"))
	if _, err := saveLifecycleState(t, root, "2026-03-18-execute-start-legacy", &runstate.State{
		Revision: 2,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-legacy-delta",
			Kind:       "delta",
			Aggregated: false,
		},
	}); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}
	seeded, _, err := runstate.LoadState(root, "2026-03-18-execute-start-legacy")
	if err != nil {
		t.Fatalf("load seeded state: %v", err)
	}
	if seeded == nil || seeded.ExecutionStartedAt != "" {
		t.Fatalf("expected legacy state to start without execution_started_at, got %#v", seeded)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 5, 0, 0, time.UTC)
		},
	}.ExecuteStart()
	if !result.OK {
		t.Fatalf("expected execute start success, got %#v", result)
	}
	if !strings.Contains(result.Summary, "Execution started") {
		t.Fatalf("unexpected summary: %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-legacy")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:05:00Z" {
		t.Fatalf("expected backfilled execution-start milestone, got %#v", state)
	}
	if state.ActiveReviewRound == nil || state.ActiveReviewRound.RoundID != "review-legacy-delta" {
		t.Fatalf("expected legacy executing state to remain otherwise intact, got %#v", state)
	}
}

func TestExecuteStartIsIdempotent(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-idempotent.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Idempotent"))

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
		},
	}
	first := svc.ExecuteStart()
	if !first.OK {
		t.Fatalf("expected first execute start success, got %#v", first)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 20, 0, 0, time.UTC)
	}
	second := svc.ExecuteStart()
	if !second.OK {
		t.Fatalf("expected second execute start success, got %#v", second)
	}
	if !strings.Contains(second.Summary, "already started") {
		t.Fatalf("unexpected second summary: %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-idempotent")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:10:00Z" {
		t.Fatalf("expected original execution-start timestamp to remain, got %#v", state)
	}
}

func TestExecuteStartRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-locked.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Locked"))

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-18-execute-start-locked")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start failure while state lock is held, got %#v", result)
	}
	if result.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Path != "state" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestExecuteStartRollsBackWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-rollback.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Rollback"))
	currentPlanAsDir := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(currentPlanAsDir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start failure, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected state rollback after pointer write failure, got %#v", state)
	}
}

func TestArchiveRejectsMissingCloseoutFields(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")
	content := buildActiveArchiveCandidate(t)
	content = strings.Replace(content, "- Validation: Validated the implementation and command surfaces.\n", "", 1)
	writeFile(t, path, content)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{Workdir: root}
	result := svc.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "section.Closeout")
}

func TestArchivePreflightFailureLeavesPlanAndPointersUntouched(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := filepath.Join(root, activeRelPath)
	content := buildActiveArchiveCandidate(t)
	content = strings.Replace(content, "## Deferred Items\n\n- None.\n", "## Deferred Items\n\n- Deferred follow-up still needs to be written down.\n", 1)
	writeFile(t, activePath, content)
	if _, err := runstate.SaveCurrentPlan(root, activeRelPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "section.Closeout.Follow-Up Issues")

	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active plan to remain after failed archive, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no archived plan to be written, got %v", err)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("expected current plan pointer to remain on active plan, got %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected state to remain after failed archive, got %#v", state)
	}
}

func TestArchiveRollsBackWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	currentPlanAsDir := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(currentPlanAsDir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active plan to remain after rollback, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected archived target to be removed on rollback, got %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected state rollback to preserve local state, got %#v", state)
	}
}

func TestArchiveRestoresActivePlanWhenTimelineAppendFailsAfterCleanup(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		AfterMutation: func(lifecycle.Result) error {
			return errors.New("timeline append failed")
		},
	}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active plan to be restored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected archived target to be removed on rollback, got %v", err)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("expected current plan pointer to be restored, got %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected state to roll back to active state, got %#v", state)
	}
}

func TestArchiveRollbackRestoresSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	activeSupplements := filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke/spec.md")
	writeFile(t, activeSupplements, "# draft spec\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		AfterMutation: func(lifecycle.Result) error {
			return errors.New("timeline append failed")
		},
	}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	if _, err := os.Stat(activeSupplements); err != nil {
		t.Fatalf("expected active supplements to be restored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/supplements/2026-03-18-archive-smoke")); !os.IsNotExist(err) {
		t.Fatalf("expected archived supplements directory to be rolled back, got %v", err)
	}
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
			errorMatch: "complete or replace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
			writeActiveArchiveCandidate(t, root, activeRelPath)
			tc.state.ExecutionStartedAt = "2026-03-18T03:30:00Z"
			if tc.state.ActiveReviewRound == nil && tc.errorPath != "state.active_review_round" {
				tc.state.ActiveReviewRound = &runstate.ReviewRound{
					RoundID:    "review-001-full",
					Kind:       "full",
					Revision:   1,
					Aggregated: true,
					Decision:   "pass",
				}
			}
			if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", tc.state); err != nil {
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

func TestArchiveRequiresPassingReviewForRevisionOne(t *testing.T) {
	testCases := []struct {
		name       string
		state      *runstate.State
		errorMatch string
	}{
		{
			name:       "missing review",
			state:      &runstate.State{},
			errorMatch: "coverage chain rooted in a full review",
		},
		{
			name: "failed full review still blocks",
			state: &runstate.State{
				ActiveReviewRound: &runstate.ReviewRound{
					RoundID:    "review-001-full",
					Kind:       "full",
					Revision:   1,
					Aggregated: true,
					Decision:   "changes_requested",
				},
			},
			errorMatch: "coverage chain rooted in a full review",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
			writeActiveArchiveCandidate(t, root, activeRelPath)
			tc.state.ExecutionStartedAt = "2026-03-18T03:30:00Z"
			if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", tc.state); err != nil {
				t.Fatalf("save state: %v", err)
			}

			result := lifecycle.Service{Workdir: root}.Archive()
			if result.OK {
				t.Fatalf("expected archive failure, got %#v", result)
			}
			assertErrorPath(t, result.Errors, "state.finalize_coverage")
			assertErrorContains(t, result.Errors, "state.finalize_coverage", tc.errorMatch)
		})
	}
}

func TestArchiveDoesNotInventEarlierStepReviewDebtAfterPassingFinalizeReview(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidateWithCloseoutDebt(t, root, activeRelPath)

	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T03:35:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if !result.OK {
		t.Fatalf("expected finalize coverage to archive without historical step-review debt, got %#v", result)
	}
}

func TestArchiveAllowsPassingDeltaReviewForReopenedRevision(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)

	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T03:55:00Z",
		Revision:           2,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-delta",
			Kind:       "delta",
			Revision:   2,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 0, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success for reopened revision, got %#v", result)
	}
}

func TestArchiveIgnoresEvidenceArtifactsOnceFinalizeReviewPasses(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-archive-smoke", "docs/plans/archived/2026-03-18-archive-smoke.md")

	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T04:05:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 10, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success despite CI/sync signals, got %#v", result)
	}
}

func TestArchiveRejectsLegacyAggregateWithoutFinalizeCoverage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T04:25:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	writeAggregateArtifact(t, root, "2026-03-18-archive-smoke", "review-001-full", map[string]any{
		"decision": "pass",
	})

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 30, 0, 0, time.UTC)
		},
	}.Archive()
	if result.OK {
		t.Fatalf("expected legacy aggregate without coverage to fail closed, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "state.finalize_coverage")
}

func TestReopenMovesArchivedPlanBackToActiveAndResetsSummaries(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
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

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
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
	if !strings.Contains(text, "UPDATE_REQUIRED_AFTER_REOPEN") {
		t.Fatalf("expected reopen update-required markers, got:\n%s", text)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected reopen mode to be recorded, got %#v", state)
	}
	if state.Revision != 2 {
		t.Fatalf("expected revision bump in state, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-archive-smoke.md" {
		t.Fatalf("expected reopened current-plan pointer to move back to active path, got %#v", current)
	}
}

func TestReopenRestoresArchivedPlanWhenTimelineAppendFailsAfterCleanup(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{Workdir: root}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	archivedRelPath := "docs/plans/archived/2026-03-18-archive-smoke.md"
	archivedPath := filepath.Join(root, archivedRelPath)
	reopen := lifecycle.Service{
		Workdir: root,
		AfterMutation: func(lifecycle.Result) error {
			return errors.New("timeline append failed")
		},
	}.Reopen("finalize-fix")
	if reopen.OK {
		t.Fatalf("expected reopen failure, got %#v", reopen)
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected archived plan to be restored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected reopened active target to be removed on rollback, got %v", err)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != archivedRelPath {
		t.Fatalf("expected current plan pointer to be restored to archived path, got %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.Reopen != nil {
		t.Fatalf("expected state to roll back to archived review state, got %#v", state)
	}
}

func TestReopenRollbackRestoresArchivedSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke/spec.md"), "# active draft\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{Workdir: root}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	archivedRelPath := "docs/plans/archived/2026-03-18-archive-smoke.md"
	archivedPath := filepath.Join(root, archivedRelPath)
	archivedSupplements := filepath.Join(root, "docs/plans/archived/supplements/2026-03-18-archive-smoke/spec.md")
	reopen := lifecycle.Service{
		Workdir: root,
		AfterMutation: func(lifecycle.Result) error {
			return errors.New("timeline append failed")
		},
	}.Reopen("finalize-fix")
	if reopen.OK {
		t.Fatalf("expected reopen failure, got %#v", reopen)
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected archived plan to be restored, got %v", err)
	}
	if _, err := os.Stat(archivedSupplements); err != nil {
		t.Fatalf("expected archived supplements to be restored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke")); !os.IsNotExist(err) {
		t.Fatalf("expected reopened active supplements directory to be removed on rollback, got %v", err)
	}
}

func TestReopenRemovesSynthesizedStateWhenTimelineAppendFailsWithoutPriorState(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{Workdir: root}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	statePath := filepath.Join(root, ".local/harness/plans/2026-03-18-archive-smoke/state.json")
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove archived state: %v", err)
	}

	archivedRelPath := "docs/plans/archived/2026-03-18-archive-smoke.md"
	archivedPath := filepath.Join(root, archivedRelPath)
	reopen := lifecycle.Service{
		Workdir: root,
		AfterMutation: func(lifecycle.Result) error {
			return errors.New("timeline append failed")
		},
	}.Reopen("finalize-fix")
	if reopen.OK {
		t.Fatalf("expected reopen failure, got %#v", reopen)
	}
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected archived plan to be restored, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, activeRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected reopened active target to be removed on rollback, got %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("expected synthesized reopened state to be removed, got %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state after failed reopen: %v", err)
	}
	if state != nil {
		t.Fatalf("expected no persisted state after failed reopen rollback, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != archivedRelPath {
		t.Fatalf("expected current plan pointer to be restored to archived path, got %#v", current)
	}
}

func TestReopenLightweightMovesLocalArchiveBackToTrackedActive(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight-reopen.md"
	writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-lightweight-reopen", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
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

	archivedPath := filepath.Join(root, ".local/harness/plans/archived/2026-03-18-lightweight-reopen.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected lightweight archived path to exist before reopen, got %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	reopenedActivePath := filepath.Join(root, activeRelPath)
	if _, err := os.Stat(reopenedActivePath); err != nil {
		t.Fatalf("reopened tracked active path missing: %v", err)
	}
	if _, err := os.Stat(archivedPath); !os.IsNotExist(err) {
		t.Fatalf("expected local lightweight archive to be removed after reopen, got %v", err)
	}
	if lint := plan.LintFile(reopenedActivePath); !lint.OK {
		t.Fatalf("reopened lightweight active plan should lint, got %#v", lint)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-lightweight-reopen")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected reopened lightweight state to point back to tracked active path, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("expected reopened current-plan pointer to move back to tracked active path, got %#v", current)
	}
	if reopen.Artifacts == nil || reopen.Artifacts.FromSupplementsPath != "" || reopen.Artifacts.ToSupplementsPath != "" {
		t.Fatalf("expected no supplements artifacts for markdown-only reopen, got %#v", reopen.Artifacts)
	}
}

func TestReopenLightweightMovesSupplementsBackToTrackedActivePackage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight-reopen.md"
	writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-18-lightweight-reopen/spec.md"), "# lightweight active draft\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-lightweight-reopen", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		Revision:           1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	archive := lifecycle.Service{Workdir: root}.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}
	reopen := lifecycle.Service{Workdir: root}.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}
	reopenedSupplements := filepath.Join(root, "docs/plans/active/supplements/2026-03-18-lightweight-reopen/spec.md")
	if _, err := os.Stat(reopenedSupplements); err != nil {
		t.Fatalf("expected reopened lightweight supplements path, got %v", err)
	}
	if reopen.Artifacts == nil || reopen.Artifacts.FromSupplementsPath != ".local/harness/plans/archived/supplements/2026-03-18-lightweight-reopen" || reopen.Artifacts.ToSupplementsPath != "docs/plans/active/supplements/2026-03-18-lightweight-reopen" {
		t.Fatalf("unexpected reopened lightweight supplements artifacts: %#v", reopen.Artifacts)
	}
}

func TestReopenMovesSupplementsDirectoryBackToActivePlanPackage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	writeFile(t, filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke/spec.md"), "# active draft\n")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		Revision:           1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	archive := lifecycle.Service{Workdir: root}.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	reopen := lifecycle.Service{Workdir: root}.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}
	reopenedSupplements := filepath.Join(root, "docs/plans/active/supplements/2026-03-18-archive-smoke/spec.md")
	if _, err := os.Stat(reopenedSupplements); err != nil {
		t.Fatalf("expected reopened supplements path, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/supplements/2026-03-18-archive-smoke")); !os.IsNotExist(err) {
		t.Fatalf("expected archived supplements directory to be removed after reopen, got %v", err)
	}
	if reopen.Artifacts == nil || reopen.Artifacts.FromSupplementsPath != "docs/plans/archived/supplements/2026-03-18-archive-smoke" || reopen.Artifacts.ToSupplementsPath != "docs/plans/active/supplements/2026-03-18-archive-smoke" {
		t.Fatalf("unexpected reopen supplements artifacts: %#v", reopen.Artifacts)
	}
}

func TestReopenNewStepRecordsModeAndStatusCue(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
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

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("new-step")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}
	if len(reopen.NextAction) == 0 || !strings.Contains(reopen.NextAction[len(reopen.NextAction)-1].Description, "Add a new unfinished step") {
		t.Fatalf("unexpected reopen next actions: %#v", reopen.NextAction)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "new-step" {
		t.Fatalf("expected reopen mode to be recorded, got %#v", state)
	}
	if state.Reopen.BaseStepCount != 2 {
		t.Fatalf("expected reopen to capture original step count, got %#v", state.Reopen)
	}

	statusResult := status.Service{Workdir: root}.Snapshot()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected current node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected status summary: %q", statusResult.Summary)
	}
}

func TestReopenMarkersMustBeClearedBeforeRearchive(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
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

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T03:05:00Z",
		Revision:           2,
		Reopen:             &runstate.ReopenState{Mode: "finalize-fix", ReopenedAt: "2026-03-18T03:00:00Z"},
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-delta",
			Kind:       "delta",
			Revision:   2,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save reopened state: %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 4, 0, 0, 0, time.UTC)
	}
	rearchive := svc.Archive()
	if rearchive.OK {
		t.Fatalf("expected rearchive to fail while reopen markers remain, got %#v", rearchive)
	}
	assertErrorPath(t, rearchive.Errors, "section.Closeout")

	activePath := filepath.Join(root, activeRelPath)
	data, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read reopened plan: %v", err)
	}
	cleared := strings.ReplaceAll(string(data), "UPDATE_REQUIRED_AFTER_REOPEN — ", "")
	writeFile(t, activePath, cleared)

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 4, 10, 0, 0, time.UTC)
	}
	rearchive = svc.Archive()
	if !rearchive.OK {
		t.Fatalf("expected rearchive success after clearing markers, got %#v", rearchive)
	}
}

func TestReopenResetsReviewStateAfterArchive(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
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

	if _, err := saveLifecycleState(t, root, "2026-03-18-archive-smoke", &runstate.State{
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save archived state: %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
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
	if state.ActiveReviewRound != nil || state.Land != nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected reopened state to preserve only reopen metadata, got %#v", state)
	}
}

func TestReopenRejectsLandCleanupInProgress(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	reopen := svc.Reopen("finalize-fix")
	if reopen.OK {
		t.Fatalf("expected reopen to fail during land cleanup, got %#v", reopen)
	}
	assertErrorPath(t, reopen.Errors, "state.land")

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.CompletedAt != "" {
		t.Fatalf("expected land cleanup state to remain intact, got %#v", state)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/archived/2026-03-18-landed-plan.md" {
		t.Fatalf("expected archived current plan to remain intact, got %#v", current)
	}
}

func TestLandCompleteWritesIdleMarkerForStatus(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	result := svc.LandComplete()
	if !result.OK {
		t.Fatalf("expected land complete success, got %#v", result)
	}
	if !strings.Contains(result.Summary, "required post-merge bookkeeping completion") {
		t.Fatalf("expected land complete summary to mention required bookkeeping, got %#v", result)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "" || current.LastLandedPlanPath != "docs/plans/archived/2026-03-18-landed-plan.md" {
		t.Fatalf("unexpected current plan marker: %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Snapshot()
	if !statusResult.OK {
		t.Fatalf("expected idle-after-land status, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "idle" {
		t.Fatalf("unexpected current node: %#v", statusResult.State)
	}
}

func TestLandGuidanceMakesPRAndIssueBookkeepingConditional(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !result.OK {
		t.Fatalf("expected land success, got %#v", result)
	}
	if !strings.Contains(result.Summary, "entered required post-merge bookkeeping") {
		t.Fatalf("expected land summary to mention required bookkeeping, got %#v", result)
	}
	if len(result.NextAction) < 2 {
		t.Fatalf("expected land next actions, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "required post-merge bookkeeping") {
		t.Fatalf("expected bookkeeping guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "rely on the forge merge record") {
		t.Fatalf("expected forge-record guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "only for material") {
		t.Fatalf("expected conditional handoff guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "only after any necessary durable handoff") {
		t.Fatalf("expected land complete gate guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "required post-merge bookkeeping completion") {
		t.Fatalf("expected land complete action to mention required bookkeeping completion, got %#v", result.NextAction)
	}
}

func TestLandCompleteRejectsMissingLandEntry(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.LandComplete()
	if result.OK {
		t.Fatalf("expected land complete failure without prior land entry, got %#v", result)
	}
}

func TestLandReadsEvidenceArtifactsWhenStateIsSparse(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.Land("https://github.com/catu-ai/easyharness/pull/99", "")
	if !result.OK {
		t.Fatalf("expected land success from evidence artifacts, got %#v", result)
	}
}

func TestLandRejectsPostArchiveProductCommitOutsideReviewCoverage(t *testing.T) {
	root := t.TempDir()
	archivedRelPath := "docs/plans/archived/2026-03-18-landed-plan.md"
	writeArchivedLandedPlan(t, root, archivedRelPath)
	if _, err := runstate.SaveCurrentPlan(root, archivedRelPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	commitLifecycleCandidate(t, root)
	writeFile(t, filepath.Join(root, "product.go"), "package product\n\nconst Unreviewed = true\n")
	commitLifecycleCandidate(t, root)
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", archivedRelPath)

	result := lifecycle.Service{Workdir: root}.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if result.OK {
		t.Fatalf("expected unreviewed post-archive product change to block land, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "review.coverage")
	assertErrorContains(t, result.Errors, "review.coverage", "product.go")
}

func TestEvaluateArchivedReviewCoverageAllowsUnrelatedBaseSynchronizationWithoutPublishBase(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "base.txt"), "base\n")
	commitLifecycleCandidate(t, root)
	archivedRelPath := "docs/plans/archived/2026-03-18-landed-plan.md"
	writeArchivedLandedPlan(t, root, archivedRelPath)
	publishedHead := commitLifecycleCandidate(t, root)
	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil || state == nil || state.FinalizeCoverage == nil {
		t.Fatalf("load reviewed coverage state: state=%#v err=%v", state, err)
	}

	runGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), runErr, output)
		}
		return strings.TrimSpace(string(output))
	}
	base := runGit("rev-parse", state.FinalizeCoverage.CoveredHeadSHA+"^")
	runGit("branch", "candidate", publishedHead)
	runGit("checkout", "-q", "main")
	runGit("reset", "-q", "--hard", base)
	writeFile(t, filepath.Join(root, "upstream.txt"), "unrelated upstream work\n")
	runGit("add", "upstream.txt")
	runGit("commit", "-q", "-m", "upstream advance")
	currentBase := runGit("rev-parse", "HEAD")
	runGit("update-ref", "refs/remotes/origin/main", currentBase)
	runGit("checkout", "-q", "candidate")
	runGit("merge", "-q", "--no-edit", "main")
	candidateHead := runGit("rev-parse", "HEAD")

	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", archivedRelPath)
	patchLifecycleEvidenceRecord(t, root, "2026-03-18-landed-plan", "publish", map[string]any{"commit": publishedHead})
	patchLifecycleEvidenceRecord(t, root, "2026-03-18-landed-plan", "sync", map[string]any{"status": "fresh", "pr_url": "https://github.com/catu-ai/easyharness/pull/99", "base_commit": currentBase, "head_commit": candidateHead})
	publish, err := evidence.LoadLatestPublish(root, "2026-03-18-landed-plan", 1)
	if err != nil || publish == nil || publish.Base != "" {
		t.Fatalf("expected contract-valid publish evidence without optional base: publish=%#v err=%v", publish, err)
	}
	doc, err := plan.LoadFile(filepath.Join(root, archivedRelPath))
	if err != nil {
		t.Fatalf("load synchronized archived plan: %v", err)
	}
	issues := lifecycle.EvaluateArchivedReviewCoverage(root, "2026-03-18-landed-plan", doc, state)
	if len(issues) != 0 {
		t.Fatalf("expected unrelated base synchronization to preserve review coverage, got %#v", issues)
	}
	patchLifecycleEvidenceRecord(t, root, "2026-03-18-landed-plan", "sync", map[string]any{"pr_url": "https://github.com/catu-ai/easyharness/pull/100"})
	issues = lifecycle.EvaluateArchivedReviewCoverage(root, "2026-03-18-landed-plan", doc, state)
	if len(issues) == 0 {
		t.Fatal("expected mismatched sync PR identity to fail closed")
	}
}

func patchLifecycleEvidenceRecord(t *testing.T, root, planStem, kind string, updates map[string]any) {
	t.Helper()
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "evidence", kind, kind+"-001.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s evidence: %v", kind, err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse %s evidence: %v", kind, err)
	}
	for key, value := range updates {
		record[key] = value
	}
	data, err = json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal %s evidence: %v", kind, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s evidence: %v", kind, err)
	}
}

func TestLandUsesPublishedRebasedCandidateAfterSquashMerge(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "base.txt"), "base\n")
	commitLifecycleCandidate(t, root)
	archivedRelPath := "docs/plans/archived/2026-03-18-landed-plan.md"
	writeArchivedLandedPlan(t, root, archivedRelPath)
	if _, err := runstate.SaveCurrentPlan(root, archivedRelPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	reviewedArchiveHead := commitLifecycleCandidate(t, root)
	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil || state == nil || state.FinalizeCoverage == nil {
		t.Fatalf("load reviewed coverage state: state=%#v err=%v", state, err)
	}

	runGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), runErr, output)
		}
		return strings.TrimSpace(string(output))
	}
	base := runGit("rev-parse", state.FinalizeCoverage.CoveredHeadSHA+"^")
	runGit("branch", "candidate", reviewedArchiveHead)
	runGit("checkout", "-q", "main")
	runGit("reset", "-q", "--hard", base)
	writeFile(t, filepath.Join(root, "upstream.txt"), "unrelated upstream work\n")
	runGit("add", "upstream.txt")
	runGit("commit", "-q", "-m", "upstream advance")
	baseHead := runGit("rev-parse", "HEAD")
	runGit("checkout", "-q", "candidate")
	runGit("rebase", "main")
	publishedHead := runGit("rev-parse", "HEAD")
	runGit("checkout", "-q", "main")
	runGit("merge", "-q", "--squash", publishedHead)
	runGit("commit", "-q", "-m", "squash candidate")
	if ancestor := exec.Command("git", "-C", root, "merge-base", "--is-ancestor", publishedHead, "HEAD").Run(); ancestor == nil {
		t.Fatal("test setup expected squash merge not to retain feature ancestry")
	}

	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", archivedRelPath)
	patchLifecycleEvidenceRecord(t, root, "2026-03-18-landed-plan", "publish", map[string]any{"commit": reviewedArchiveHead})
	patchLifecycleEvidenceRecord(t, root, "2026-03-18-landed-plan", "sync", map[string]any{"status": "fresh", "pr_url": "https://github.com/catu-ai/easyharness/pull/99", "base_commit": baseHead, "head_commit": publishedHead})

	result := lifecycle.Service{Workdir: root}.Land("https://github.com/catu-ai/easyharness/pull/99", runGit("rev-parse", "HEAD"))
	if !result.OK {
		t.Fatalf("expected squash-land recovery from immutable publish evidence, got %#v", result)
	}
}

func TestEvaluateArchivedReviewCoverageRejectsUnexpectedArchiveDestination(t *testing.T) {
	root := t.TempDir()
	canonicalRelPath := "docs/plans/archived/2026-03-18-landed-plan.md"
	canonicalPath := writeArchivedLandedPlan(t, root, canonicalRelPath)
	unexpectedPath := filepath.Join(root, "unexpected", "2026-03-18-landed-plan.md")
	if err := os.MkdirAll(filepath.Dir(unexpectedPath), 0o755); err != nil {
		t.Fatalf("mkdir unexpected archive root: %v", err)
	}
	if err := os.Rename(canonicalPath, unexpectedPath); err != nil {
		t.Fatalf("move plan to unexpected archive root: %v", err)
	}
	doc, err := plan.LoadFile(unexpectedPath)
	if err != nil {
		t.Fatalf("load unexpected archived plan: %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load reviewed coverage state: %v", err)
	}
	issues := lifecycle.EvaluateArchivedReviewCoverage(root, "2026-03-18-landed-plan", doc, state)
	assertErrorPath(t, issues, "review.coverage")
	assertErrorContains(t, issues, "review.coverage", "expected the configured archive path")
}

func TestLandRejectsOlderRevisionEvidenceAfterReopen(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := saveLifecycleState(t, root, "2026-03-18-landed-plan", &runstate.State{
		Revision: 1,
	}); err != nil {
		t.Fatalf("save initial state: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := saveLifecycleState(t, root, "2026-03-18-landed-plan", &runstate.State{
		Revision: 2,
	}); err != nil {
		t.Fatalf("save reopened state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.Land("https://github.com/catu-ai/easyharness/pull/99", "")
	if result.OK {
		t.Fatalf("expected older revision evidence to block land, got %#v", result)
	}
}

func TestLandRejectsOverwriteDuringCleanup(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	first := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !first.OK {
		t.Fatalf("expected initial land success, got %#v", first)
	}

	second := svc.Land("https://github.com/catu-ai/easyharness/pull/100", "def456")
	if second.OK {
		t.Fatalf("expected second land entry to fail, got %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.PRURL != "https://github.com/catu-ai/easyharness/pull/99" || state.Land.Commit != "abc123" {
		t.Fatalf("expected original land record to remain intact, got %#v", state)
	}
}

func TestLandAllowsCommitEnrichmentDuringCleanup(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	first := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "")
	if !first.OK {
		t.Fatalf("expected initial land success, got %#v", first)
	}

	second := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !second.OK {
		t.Fatalf("expected commit enrichment success, got %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.PRURL != "https://github.com/catu-ai/easyharness/pull/99" || state.Land.Commit != "abc123" {
		t.Fatalf("expected commit enrichment to update the existing land record, got %#v", state)
	}
}

func TestLandCompleteRollsBackStateWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	writeMergeReadyEvidenceArtifacts(t, root, "2026-03-18-landed-plan", "docs/plans/archived/2026-03-18-landed-plan.md")

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	currentPlanPath := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.Remove(currentPlanPath); err != nil {
		t.Fatalf("remove current-plan file: %v", err)
	}
	if err := os.MkdirAll(currentPlanPath, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := svc.LandComplete()
	if result.OK {
		t.Fatalf("expected land complete failure when current-plan write fails, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.CompletedAt != "" {
		t.Fatalf("expected land state rollback after pointer write failure, got %#v", state)
	}
}

func writeMergeReadyEvidenceArtifacts(t *testing.T, root, planStem, planPath string) {
	t.Helper()
	revision := 1
	if state, _, err := runstate.LoadState(root, planStem); err == nil {
		revision = runstate.CurrentRevision(state)
	}
	type recordFile struct {
		dir     string
		name    string
		payload any
	}
	recordedAt := time.Date(2026, 3, 18, 5, 50, 0, 0, time.UTC).Format(time.RFC3339)
	files := []recordFile{
		{
			dir:  filepath.Join(root, ".local", "harness", "plans", planStem, "evidence", "publish"),
			name: "publish-001.json",
			payload: map[string]any{
				"record_id":   "publish-001",
				"kind":        "publish",
				"plan_path":   planPath,
				"plan_stem":   planStem,
				"revision":    revision,
				"recorded_at": recordedAt,
				"status":      "recorded",
				"pr_url":      "https://github.com/catu-ai/easyharness/pull/99",
				"branch":      "codex/test",
			},
		},
		{
			dir:  filepath.Join(root, ".local", "harness", "plans", planStem, "evidence", "ci"),
			name: "ci-001.json",
			payload: map[string]any{
				"record_id":   "ci-001",
				"kind":        "ci",
				"plan_path":   planPath,
				"plan_stem":   planStem,
				"revision":    revision,
				"recorded_at": recordedAt,
				"status":      "success",
				"provider":    "github-actions",
			},
		},
		{
			dir:  filepath.Join(root, ".local", "harness", "plans", planStem, "evidence", "sync"),
			name: "sync-001.json",
			payload: map[string]any{
				"record_id":   "sync-001",
				"kind":        "sync",
				"plan_path":   planPath,
				"plan_stem":   planStem,
				"revision":    revision,
				"recorded_at": recordedAt,
				"status":      "not_applied",
				"reason":      "generic lifecycle fixture does not model a remote base comparison",
			},
		},
	}
	for _, file := range files {
		if err := os.MkdirAll(file.dir, 0o755); err != nil {
			t.Fatalf("mkdir evidence dir: %v", err)
		}
		data, err := json.Marshal(file.payload)
		if err != nil {
			t.Fatalf("marshal evidence record: %v", err)
		}
		if err := os.WriteFile(filepath.Join(file.dir, file.name), data, 0o644); err != nil {
			t.Fatalf("write evidence record: %v", err)
		}
	}
}

func writeActiveArchiveCandidate(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	writeFile(t, path, buildActiveArchiveCandidate(t))
	return path
}

func TestExecuteStartReturnsCoordinateForCoordinatedRoot(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs/plans/active/2026-07-28-coordinated-start.md")
	writeFile(t, activePath, buildCoordinatedRoot(t, "Coordinated Start", false))

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if !result.OK {
		t.Fatalf("expected coordinated execution start, got %#v", result)
	}
	if result.State.CurrentNode != "execution/coordinate" {
		t.Fatalf("current node = %q, want execution/coordinate", result.State.CurrentNode)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current pointer: %v", err)
	}
	if current.PlanPath != "docs/plans/active/2026-07-28-coordinated-start.md" {
		t.Fatalf("current root pointer = %q", current.PlanPath)
	}
}

func TestExecuteStartRejectsReusedRootStemWithStaleRuntimeState(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-07-28-reused-root.md"
	archivedRelPath := "docs/plans/archived/2026-07-28-reused-root.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildCoordinatedRoot(t, "New Unapproved Root", false))
	writeFile(t, filepath.Join(root, archivedRelPath), buildCoordinatedRoot(t, "Old Archived Root", true))
	if _, err := runstate.SaveCurrentPlan(root, archivedRelPath); err != nil {
		t.Fatalf("save archived pointer: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-07-28-reused-root", &runstate.State{
		ExecutionStartedAt: "2026-07-28T12:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save stale state: %v", err)
	}

	result := (lifecycle.Service{Workdir: root}).ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start to reject reused root identity, got %#v", result)
	}
	assertLifecycleIssueContains(t, result.Errors, "must remain unique")
}

func TestCoordinatedArchiveReadinessRequiresSettledPackage(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs/plans/active/2026-07-28-coordinated-readiness.md")
	writeFile(t, activePath, buildCoordinatedRoot(t, "Coordinated Readiness", true))
	doc, err := plan.LoadFile(activePath)
	if err != nil {
		t.Fatalf("load coordinated root: %v", err)
	}

	issues := lifecycle.EvaluateArchiveReadiness(root, "2026-07-28-coordinated-readiness", doc, &runstate.State{Revision: 1})
	assertLifecycleIssueContains(t, issues, "at least one subplan")

	subplanPath, err := plan.SubplanPathForPlan(activePath, "api")
	if err != nil {
		t.Fatalf("resolve subplan path: %v", err)
	}
	writeFile(t, subplanPath, buildCoordinatedSubplan(t, "API", false))
	issues = lifecycle.EvaluateArchiveReadiness(root, "2026-07-28-coordinated-readiness", doc, &runstate.State{Revision: 1})
	assertLifecycleIssueContains(t, issues, "all subplan steps and Result fields")
}

func TestArchiveAndReopenMoveCompleteCoordinatedPackage(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-07-28-coordinated-archive.md"
	activePath := filepath.Join(root, activeRelPath)
	writeFile(t, activePath, buildCoordinatedRoot(t, "Coordinated Archive", true))
	subplanPath, err := plan.SubplanPathForPlan(activePath, "api")
	if err != nil {
		t.Fatalf("resolve subplan path: %v", err)
	}
	writeFile(t, subplanPath, buildCoordinatedSubplan(t, "API", true))

	planStem := "2026-07-28-coordinated-archive"
	if _, err := saveLifecycleState(t, root, planStem, &runstate.State{
		Revision:           1,
		ExecutionStartedAt: "2026-07-28T10:00:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save coordinated state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC)
		},
	}
	archived := svc.Archive()
	if !archived.OK {
		t.Fatalf("expected coordinated archive success, got %#v", archived)
	}
	archivedPath := filepath.Join(root, "docs/plans/archived/2026-07-28-coordinated-archive.md")
	archivedSubplan, err := plan.SubplanPathForPlan(archivedPath, "api")
	if err != nil {
		t.Fatalf("resolve archived subplan path: %v", err)
	}
	if _, err := os.Stat(archivedSubplan); err != nil {
		t.Fatalf("archived subplan missing: %v", err)
	}
	if _, err := os.Stat(subplanPath); !os.IsNotExist(err) {
		t.Fatalf("active subplan should move with package, got %v", err)
	}

	reopened := svc.Reopen("new-step")
	if !reopened.OK {
		t.Fatalf("expected coordinated reopen success, got %#v", reopened)
	}
	if reopened.State.CurrentNode != "execution/coordinate" {
		t.Fatalf("reopen node = %q, want execution/coordinate", reopened.State.CurrentNode)
	}
	state, _, err := runstate.LoadState(root, planStem)
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.BaseStepCount != 1 {
		t.Fatalf("reopen baseline = %#v, want one coordinated child", state)
	}
	if _, err := os.Stat(subplanPath); err != nil {
		t.Fatalf("reopened subplan missing: %v", err)
	}
	if _, err := os.Stat(archivedSubplan); !os.IsNotExist(err) {
		t.Fatalf("archived subplan should move back with package, got %v", err)
	}
}

func buildCoordinatedRoot(t *testing.T, title string, archiveReady bool) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           title,
		Timestamp:       time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
		SourceType:      "direct_request",
		Size:            "M",
		WorkflowProfile: plan.WorkflowProfileCoordinated,
	})
	if err != nil {
		t.Fatalf("render coordinated root: %v", err)
	}
	rendered = strings.Replace(rendered, "created_at: 2026-07-28T09:00:00Z", "created_at: 2026-07-28T09:00:00Z\napproved_at: 2026-07-28T09:05:00Z", 1)
	if archiveReady {
		rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
		rendered = completeCloseout(rendered)
	}
	return rendered
}

func buildCoordinatedSubplan(t *testing.T, title string, complete bool) string {
	t.Helper()
	rendered, err := plan.RenderSubplanTemplate(plan.SubplanTemplateOptions{Title: title})
	if err != nil {
		t.Fatalf("render coordinated subplan: %v", err)
	}
	if complete {
		rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
		rendered = strings.Replace(rendered, "- Validation: PENDING", "- Validation: Focused tests passed.", 1)
		rendered = strings.Replace(rendered, "- Delivered: PENDING", "- Delivered: Implemented the child outcome.", 1)
	}
	return rendered
}

func assertLifecycleIssueContains(t *testing.T, issues []contracts.ErrorDetail, want string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue.Message, want) {
			return
		}
	}
	t.Fatalf("expected lifecycle issue containing %q, got %#v", want, issues)
}

func saveLifecycleState(t *testing.T, root, planStem string, state *runstate.State) (string, error) {
	t.Helper()
	if state != nil && state.ActiveReviewRound != nil && state.ActiveReviewRound.Aggregated && state.ActiveReviewRound.Decision == "pass" {
		head := commitLifecycleCandidate(t, root)
		reviewPlanPath := ""
		if detected, err := plan.DetectCurrentPath(root); err == nil {
			if rel, relErr := filepath.Rel(root, detected); relErr == nil {
				reviewPlanPath = filepath.ToSlash(rel)
			}
		}
		revision := runstate.CurrentRevision(state)
		tipID := state.ActiveReviewRound.RoundID
		rootID := tipID
		rootHead := head
		if state.ActiveReviewRound.Kind == "delta" {
			rootID = "review-001-full"
			rootRevision := revision - 1
			if rootRevision < 1 {
				rootRevision = 1
			}
			writeLifecycleReviewRound(t, root, planStem, contracts.ReviewManifest{
				RoundID: rootID, Kind: "full", ReviewedHeadSHA: rootHead, Revision: rootRevision, PlanPath: reviewPlanPath,
			}, cleanLifecycleAggregate(rootID, "full", rootHead, rootRevision, nil))
			repair := &contracts.ReviewRepairReference{RoundID: rootID, FindingIDs: []string{}}
			writeLifecycleReviewRound(t, root, planStem, contracts.ReviewManifest{
				RoundID: tipID, Kind: "delta", AnchorSHA: rootHead, ReviewedHeadSHA: head, Revision: revision, Repair: repair, PlanPath: reviewPlanPath,
			}, cleanLifecycleAggregate(tipID, "delta", head, revision, repair))
		} else {
			writeLifecycleReviewRound(t, root, planStem, contracts.ReviewManifest{
				RoundID: tipID, Kind: "full", ReviewedHeadSHA: head, Revision: revision, PlanPath: reviewPlanPath,
			}, cleanLifecycleAggregate(tipID, "full", head, revision, nil))
		}
		state.FinalizeCoverage = &runstate.FinalizeCoverage{
			RootRoundID: rootID, TipRoundID: tipID, CoveredHeadSHA: head, Revision: revision,
		}
	}
	return runstate.SaveState(root, planStem, state)
}

func cleanLifecycleAggregate(roundID, kind, head string, revision int, repair *contracts.ReviewRepairReference) contracts.ReviewAggregate {
	return contracts.ReviewAggregate{
		RoundID: roundID, Kind: kind, ReviewedHeadSHA: head, Revision: revision, Repair: repair, Decision: "pass",
		BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{},
		ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{},
	}
}

func writeLifecycleReviewRound(t *testing.T, root, planStem string, manifest contracts.ReviewManifest, aggregate contracts.ReviewAggregate) {
	t.Helper()
	dir := runstate.ReviewRoundDir(root, planStem, manifest.RoundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir review round: %v", err)
	}
	for name, value := range map[string]any{"manifest.json": manifest, "aggregate.json": aggregate} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func commitLifecycleCandidate(t *testing.T, root string) string {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return strings.TrimSpace(string(output))
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); os.IsNotExist(err) {
		run("init", "-q", "-b", "main")
		run("config", "user.name", "Codex Test")
		run("config", "user.email", "codex@example.com")
		writeFile(t, filepath.Join(root, ".gitignore"), ".local/\n")
	}
	run("add", ".")
	check := exec.Command("git", "-C", root, "diff", "--cached", "--quiet")
	if err := check.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			t.Fatalf("inspect candidate fixture: %v", err)
		}
		run("commit", "-q", "-m", "archive candidate")
	}
	return run("rev-parse", "HEAD")
}

func writeLightweightActiveArchiveCandidate(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	content := strings.Replace(buildActiveArchiveCandidate(t), "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	content = strings.Replace(content, "size: M", "size: XXS", 1)
	writeFile(t, path, content)
	return path
}

func writeArchivedLandedPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Landed Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	rendered = strings.Replace(rendered, "## Closeout\n", "## Closeout\n\n- Archived At: 2026-03-18T02:00:00Z\n- Revision: 1", 1)
	rendered = completeCloseout(rendered)
	activeRelPath := strings.Replace(relPath, "/archived/", "/active/", 1)
	activePath := filepath.Join(root, activeRelPath)
	writeFile(t, activePath, rendered)
	planStem := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	if _, err := saveLifecycleState(t, root, planStem, &runstate.State{
		Revision:           1,
		ExecutionStartedAt: "2026-03-18T01:00:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID: "review-001-full", Kind: "full", Revision: 1, Aggregated: true, Decision: "pass",
		},
	}); err != nil {
		t.Fatalf("seed landed-plan review coverage: %v", err)
	}
	writeFile(t, path, rendered)
	if err := os.Remove(activePath); err != nil {
		t.Fatalf("remove reviewed active plan after archive fixture move: %v", err)
	}
	return path
}

func buildActiveArchiveCandidate(t *testing.T) string {
	t.Helper()
	rendered := buildAwaitingPlan(t, "Archive Smoke")
	rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	return completeCloseout(rendered)
}

func writeActiveArchiveCandidateWithCloseoutDebt(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	rendered := buildAwaitingPlan(t, "Archive Smoke")
	rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	rendered = completeCloseout(rendered)
	writeFile(t, path, rendered)
	return path
}

func completeCloseout(rendered string) string {
	replacements := map[string]string{
		"- Validation: PENDING_UNTIL_ARCHIVE":    "- Validation: Validated the implementation and command surfaces.",
		"- Review: PENDING_UNTIL_ARCHIVE":        "- Review: No unresolved blocking review findings remain.",
		"- Delivered: PENDING_UNTIL_ARCHIVE":     "- Delivered: Delivered the planned CLI slice.",
		"- Not Delivered: PENDING_UNTIL_ARCHIVE": "- Not Delivered: NONE.",
	}
	for old, replacement := range replacements {
		rendered = strings.Replace(rendered, old, replacement, 1)
	}
	return rendered
}

func buildAwaitingPlan(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	return strings.Replace(rendered, "created_at: 2026-03-18T02:00:00Z", "created_at: 2026-03-18T02:00:00Z\napproved_at: 2026-03-18T02:05:00Z", 1)
}

func stripApprovedAt(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "approved_at:") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
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

func writeAggregateArtifact(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir aggregate dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal aggregate payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aggregate.json"), data, 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func containsActionDescription(actions []lifecycle.NextAction, snippet string) bool {
	for _, action := range actions {
		if strings.Contains(action.Description, snippet) {
			return true
		}
	}
	return false
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

func assertRawStateJSONOmitsKeys(t *testing.T, path string, keys ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw state json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse raw state json: %v", err)
	}
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected raw state json to omit %q, got %#v", key, payload)
		}
	}
}
