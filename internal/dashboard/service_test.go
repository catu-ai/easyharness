package dashboard

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
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/watchlist"
)

func TestReadGroupsReadableWorkspaceStates(t *testing.T) {
	home := t.TempDir()
	active := seedGitWorkspace(t, "active")
	completed := seedGitWorkspace(t, "completed")
	idle := seedGitWorkspace(t, "idle")
	writeWatchlist(t, home, []watchlist.Workspace{
		workspaceRecord(active, "2026-04-22T12:00:00Z"),
		workspaceRecord(completed, "2026-04-22T11:00:00Z"),
		workspaceRecord(idle, "2026-04-22T10:00:00Z"),
	})

	result := Service{
		LookupEnv: easyHome(home),
		ReadStatus: func(path string) contracts.StatusResult {
			switch path {
			case active:
				return statusResult("execution/finalize/await_merge", "Awaiting merge", nil)
			case completed:
				return statusResult("idle", "Landed", &contracts.StatusArtifacts{LastLandedAt: "2026-04-22T09:00:00Z"})
			case idle:
				return statusResult("idle", "No current plan", nil)
			default:
				t.Fatalf("unexpected status path %q", path)
				return contracts.StatusResult{}
			}
		},
	}.Read()

	if !result.OK {
		t.Fatalf("expected dashboard read to succeed, got %#v", result)
	}
	assertGroupPaths(t, result, StateActive, []string{active})
	assertGroupPaths(t, result, StateCompleted, []string{completed})
	assertGroupPaths(t, result, StateIdle, []string{idle})
	activeEntry := findWorkspace(t, result, StateActive, active)
	if activeEntry.CurrentNode != "execution/finalize/await_merge" || activeEntry.DashboardState != StateActive {
		t.Fatalf("unexpected active entry: %#v", activeEntry)
	}
	completedEntry := findWorkspace(t, result, StateCompleted, completed)
	if completedEntry.CurrentNode != "idle" || completedEntry.Artifacts == nil || completedEntry.Artifacts.LastLandedAt == "" {
		t.Fatalf("unexpected completed entry: %#v", completedEntry)
	}
	idleEntry := findWorkspace(t, result, StateIdle, idle)
	if idleEntry.CurrentNode != "idle" || idleEntry.InvalidReason != "" {
		t.Fatalf("unexpected idle entry: %#v", idleEntry)
	}
}

func TestReadSurfacesMissingAndInvalidEntries(t *testing.T) {
	home := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing")
	notGit := filepath.Join(t.TempDir(), "not-git")
	if err := os.MkdirAll(notGit, 0o755); err != nil {
		t.Fatalf("mkdir non-git workspace: %v", err)
	}
	statusError := seedGitWorkspace(t, "status-error")
	writeWatchlist(t, home, []watchlist.Workspace{
		workspaceRecord(missing, "2026-04-22T12:00:00Z"),
		workspaceRecord(notGit, "2026-04-22T11:00:00Z"),
		workspaceRecord(statusError, "2026-04-22T10:00:00Z"),
	})

	result := Service{
		LookupEnv: easyHome(home),
		ReadStatus: func(path string) contracts.StatusResult {
			if path != statusError {
				t.Fatalf("unexpected status path %q", path)
			}
			return contracts.StatusResult{
				OK:      false,
				Command: "status",
				Summary: "Unable to read current worktree state.",
				State:   contracts.StatusState{CurrentNode: "idle"},
				Errors:  []contracts.ErrorDetail{{Path: "state", Message: "boom"}},
			}
		},
	}.Read()

	if !result.OK {
		t.Fatalf("expected dashboard read to succeed, got %#v", result)
	}
	assertGroupPaths(t, result, StateMissing, []string{missing})
	assertGroupPaths(t, result, StateInvalid, []string{notGit, statusError})
	missingEntry := findWorkspace(t, result, StateMissing, missing)
	if missingEntry.InvalidReason != "" || missingEntry.CurrentNode != "" {
		t.Fatalf("unexpected missing entry: %#v", missingEntry)
	}
	notGitEntry := findWorkspace(t, result, StateInvalid, notGit)
	if notGitEntry.InvalidReason != InvalidNotGitWorkspace {
		t.Fatalf("expected not-git invalid reason, got %#v", notGitEntry)
	}
	statusErrorEntry := findWorkspace(t, result, StateInvalid, statusError)
	if statusErrorEntry.InvalidReason != InvalidStatusError || statusErrorEntry.CurrentNode != "idle" {
		t.Fatalf("expected status-error invalid reason with partial node, got %#v", statusErrorEntry)
	}
}

func TestReadOrdersEntriesByRecencyWithDeterministicFallback(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	newest := seedGitWorkspaceAt(t, root, "newest")
	alpha := seedGitWorkspaceAt(t, root, "alpha")
	beta := seedGitWorkspaceAt(t, root, "beta")
	malformed := seedGitWorkspaceAt(t, root, "malformed")
	writeWatchlist(t, home, []watchlist.Workspace{
		workspaceRecord(beta, "2026-04-22T10:00:00Z"),
		workspaceRecord(malformed, "not-a-time"),
		workspaceRecord(newest, "2026-04-22T12:00:00Z"),
		workspaceRecord(alpha, "2026-04-22T10:00:00Z"),
	})

	result := Service{
		LookupEnv:  easyHome(home),
		ReadStatus: func(string) contracts.StatusResult { return statusResult("plan", "Plan exists", nil) },
	}.Read()

	assertGroupPaths(t, result, StateActive, []string{newest, alpha, beta, malformed})
}

func TestReadReturnsTopLevelErrorForUnreadableWatchlist(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "watchlist.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir watchlist dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"version":`), 0o644); err != nil {
		t.Fatalf("write invalid watchlist: %v", err)
	}

	result := Service{LookupEnv: easyHome(home)}.Read()
	if result.OK {
		t.Fatalf("expected unreadable watchlist to fail top-level read, got %#v", result)
	}
	if result.Resource != "dashboard" || len(result.Errors) != 1 {
		t.Fatalf("unexpected top-level error result: %#v", result)
	}
	assertGroupPaths(t, result, StateActive, nil)
}

func TestReadDoesNotRewriteWatchlist(t *testing.T) {
	home := t.TempDir()
	workspace := seedGitWorkspace(t, "idle")
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	watchlistPath := filepath.Join(home, "watchlist.json")
	fixedTime := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	if err := os.Chtimes(watchlistPath, fixedTime, fixedTime); err != nil {
		t.Fatalf("set watchlist timestamp: %v", err)
	}
	beforeInfo, err := os.Stat(watchlistPath)
	if err != nil {
		t.Fatalf("stat watchlist before read: %v", err)
	}
	beforeData, err := os.ReadFile(watchlistPath)
	if err != nil {
		t.Fatalf("read watchlist before read: %v", err)
	}

	result := Service{
		LookupEnv:  easyHome(home),
		ReadStatus: func(string) contracts.StatusResult { return statusResult("idle", "No current plan", nil) },
	}.Read()
	if !result.OK {
		t.Fatalf("dashboard read failed: %#v", result)
	}

	afterInfo, err := os.Stat(watchlistPath)
	if err != nil {
		t.Fatalf("stat watchlist after read: %v", err)
	}
	afterData, err := os.ReadFile(watchlistPath)
	if err != nil {
		t.Fatalf("read watchlist after read: %v", err)
	}
	if string(afterData) != string(beforeData) {
		t.Fatalf("expected dashboard read to preserve watchlist bytes")
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("expected dashboard read to preserve watchlist mtime, got %s want %s", afterInfo.ModTime(), beforeInfo.ModTime())
	}
}

func TestReadUsesDefaultStatusService(t *testing.T) {
	home := t.TempDir()
	workspace := seedGitWorkspace(t, "default-status")
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})

	result := Service{LookupEnv: easyHome(home)}.Read()
	if !result.OK {
		t.Fatalf("dashboard read failed: %#v", result)
	}
	entry := findWorkspace(t, result, StateIdle, workspace)
	if entry.CurrentNode != "idle" || !strings.Contains(entry.Summary, "No current plan is active") {
		t.Fatalf("expected default status service idle entry, got %#v", entry)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".local")); !os.IsNotExist(err) {
		t.Fatalf("expected dashboard read to avoid creating workflow state, err=%v", err)
	}
}

func TestReadUsesDefaultStatusServiceForActiveWorkspaceWithoutMutatingState(t *testing.T) {
	home := t.TempDir()
	workspace := seedGitWorkspace(t, "active-default-status")
	relPlanPath := writeActivePlan(t, workspace)
	if _, err := runstate.SaveCurrentPlan(workspace, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	planStem := strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
	if _, err := runstate.SaveState(workspace, planStem, &runstate.State{
		ExecutionStartedAt: "2026-04-22T09:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	currentPlanPath := filepath.Join(workspace, ".local", "harness", "current-plan.json")
	statePath := filepath.Join(workspace, ".local", "harness", "plans", planStem, "state.json")
	lockPath := filepath.Join(workspace, ".local", "harness", "plans", planStem, ".state-mutation.lock")
	stateFiles := []string{currentPlanPath, statePath}
	fixedTime := time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC)
	before := make(map[string]fileSnapshot, len(stateFiles))
	for _, path := range stateFiles {
		if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
			t.Fatalf("set state timestamp %s: %v", path, err)
		}
		before[path] = snapshotFile(t, path)
	}

	result := Service{LookupEnv: easyHome(home)}.Read()
	if !result.OK {
		t.Fatalf("dashboard read failed: %#v", result)
	}
	entry := findWorkspace(t, result, StateActive, workspace)
	if entry.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected active execution node, got %#v", entry)
	}
	for _, path := range stateFiles {
		assertFileUnchanged(t, path, before[path])
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected dashboard read to avoid creating state lock, err=%v", err)
	}
}

func TestReadSurfacesGitProbeFailuresAsUnreadable(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})

	result := Service{
		LookupEnv: easyHome(home),
		CheckGitWorkspace: func(path string) error {
			if path != workspace {
				t.Fatalf("unexpected git probe path %q", path)
			}
			return errors.New("inspect git workspace: permission denied")
		},
	}.Read()

	entry := findWorkspace(t, result, StateInvalid, workspace)
	if entry.InvalidReason != InvalidUnreadable {
		t.Fatalf("expected unreadable invalid reason, got %#v", entry)
	}
}

func statusResult(node, summary string, artifacts *contracts.StatusArtifacts) contracts.StatusResult {
	return contracts.StatusResult{
		OK:        true,
		Command:   "status",
		Summary:   summary,
		State:     contracts.StatusState{CurrentNode: node},
		Artifacts: artifacts,
		NextAction: []contracts.NextAction{
			{Command: nil, Description: "Keep going."},
		},
	}
}

type fileSnapshot struct {
	data    []byte
	modTime time.Time
}

func snapshotFile(t *testing.T, path string) fileSnapshot {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file %s: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return fileSnapshot{data: data, modTime: info.ModTime()}
}

func assertFileUnchanged(t *testing.T, path string, before fileSnapshot) {
	t.Helper()
	after := snapshotFile(t, path)
	if string(after.data) != string(before.data) {
		t.Fatalf("expected %s bytes to remain unchanged", path)
	}
	if !after.modTime.Equal(before.modTime) {
		t.Fatalf("expected %s mtime to remain unchanged, got %s want %s", path, after.modTime, before.modTime)
	}
}

func writeActivePlan(t *testing.T, root string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Dashboard Active Plan",
		Timestamp:  time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
		Size:       "M",
	})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	relPath := filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-04-22-dashboard-active-plan.md"))
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath
}

func writeWatchlist(t *testing.T, home string, workspaces []watchlist.Workspace) {
	t.Helper()
	data, err := json.MarshalIndent(watchlist.File{Version: 1, Workspaces: workspaces}, "", "  ")
	if err != nil {
		t.Fatalf("marshal watchlist: %v", err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir watchlist home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "watchlist.json"), data, 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
}

func workspaceRecord(path, seenAt string) watchlist.Workspace {
	return watchlist.Workspace{
		WorkspacePath: path,
		WatchedAt:     "2026-04-22T09:00:00Z",
		LastSeenAt:    seenAt,
	}
}

func easyHome(home string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		if key == "EASYHARNESS_HOME" {
			return home, true
		}
		return "", false
	}
}

func seedGitWorkspace(t *testing.T, name string) string {
	t.Helper()
	return seedGitWorkspaceAt(t, t.TempDir(), name)
}

func seedGitWorkspaceAt(t *testing.T, parent, name string) string {
	t.Helper()
	root := filepath.Join(parent, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir git workspace: %v", err)
	}
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Codex Test")
	runGit(t, root, "config", "user.email", "codex@example.com")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func assertGroupPaths(t *testing.T, result Result, state string, want []string) {
	t.Helper()
	for _, group := range result.Groups {
		if group.State != state {
			continue
		}
		got := make([]string, 0, len(group.Workspaces))
		for _, workspace := range group.Workspaces {
			got = append(got, workspace.WorkspacePath)
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("unexpected %s group\n got: %#v\nwant: %#v", state, got, want)
		}
		return
	}
	t.Fatalf("missing group %q", state)
}

func findWorkspace(t *testing.T, result Result, state, path string) Workspace {
	t.Helper()
	for _, group := range result.Groups {
		if group.State != state {
			continue
		}
		for _, workspace := range group.Workspaces {
			if workspace.WorkspacePath == path {
				return workspace
			}
		}
	}
	t.Fatalf("missing workspace %q in state %q", path, state)
	return Workspace{}
}

func TestReadSurfacesStatErrorsAsUnreadable(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "unreadable")
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})

	result := Service{
		LookupEnv: easyHome(home),
		Stat: func(path string) (os.FileInfo, error) {
			if path != workspace {
				t.Fatalf("unexpected stat path %q", path)
			}
			return nil, errors.New("permission denied")
		},
	}.Read()

	entry := findWorkspace(t, result, StateInvalid, workspace)
	if entry.InvalidReason != InvalidUnreadable {
		t.Fatalf("expected unreadable invalid reason, got %#v", entry)
	}
}
