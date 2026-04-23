package watchlist

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTouchUsesDefaultEasyharnessHome(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)

	svc := Service{
		UserHomeDir: func() (string, error) { return userHome, nil },
		Now: func() time.Time {
			return time.Date(2026, 4, 19, 1, 2, 3, 0, time.UTC)
		},
	}
	if err := svc.Touch(workdir); err != nil {
		t.Fatalf("touch watchlist: %v", err)
	}

	got := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if got.Version != version {
		t.Fatalf("expected version %d, got %#v", version, got)
	}
	if len(got.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %#v", got.Workspaces)
	}
	canonical, err := filepath.EvalSymlinks(workdir)
	if err != nil {
		t.Fatalf("resolve canonical workspace: %v", err)
	}
	if got.Workspaces[0].WorkspacePath != canonical {
		t.Fatalf("expected workspace path %q, got %#v", workdir, got.Workspaces[0])
	}
}

func TestReadUsesDefaultEasyharnessHome(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")
	seed := File{
		Version: version,
		Workspaces: []Workspace{
			{WorkspacePath: "/tmp/workspace-a", WatchedAt: "2026-04-19T01:00:00Z", LastSeenAt: "2026-04-19T02:00:00Z"},
		},
	}
	writeWatchlistFile(t, watchlistPath, seed)

	got, err := Service{UserHomeDir: func() (string, error) { return userHome, nil }}.Read()
	if err != nil {
		t.Fatalf("read watchlist: %v", err)
	}
	if got.Version != version || len(got.Workspaces) != 1 {
		t.Fatalf("unexpected watchlist: %#v", got)
	}
	if got.Workspaces[0].WorkspacePath != "/tmp/workspace-a" {
		t.Fatalf("expected persisted workspace path to survive read, got %#v", got.Workspaces[0])
	}
}

func TestReadUsesEasyharnessHomeOverride(t *testing.T) {
	userHome := t.TempDir()
	customHome := filepath.Join(t.TempDir(), "custom-home")
	writeWatchlistFile(t, filepath.Join(customHome, "watchlist.json"), File{
		Version: version,
		Workspaces: []Workspace{
			{WorkspacePath: "/tmp/custom", WatchedAt: "2026-04-19T01:00:00Z", LastSeenAt: "2026-04-19T01:00:00Z"},
		},
	})

	svc := Service{
		LookupEnv: func(key string) (string, bool) {
			if key == envHome {
				return customHome, true
			}
			return "", false
		},
		UserHomeDir: func() (string, error) { return userHome, nil },
	}
	got, err := svc.Read()
	if err != nil {
		t.Fatalf("read watchlist: %v", err)
	}
	if len(got.Workspaces) != 1 || got.Workspaces[0].WorkspacePath != "/tmp/custom" {
		t.Fatalf("expected custom home watchlist, got %#v", got)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".easyharness", "watchlist.json")); !os.IsNotExist(err) {
		t.Fatalf("expected default home to remain untouched, err=%v", err)
	}
}

func TestReadMissingWatchlistReturnsEmptyFileWithoutCreatingIt(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")

	got, err := Service{UserHomeDir: func() (string, error) { return userHome, nil }}.Read()
	if err != nil {
		t.Fatalf("read missing watchlist: %v", err)
	}
	if got.Version != version || len(got.Workspaces) != 0 {
		t.Fatalf("expected empty watchlist file model, got %#v", got)
	}
	if _, err := os.Stat(watchlistPath); !os.IsNotExist(err) {
		t.Fatalf("expected read to avoid creating watchlist, err=%v", err)
	}
}

func TestReadExistingWatchlistDoesNotRewriteFile(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")
	writeWatchlistFile(t, watchlistPath, File{
		Version: version,
		Workspaces: []Workspace{
			{WorkspacePath: "/tmp/workspace-a", WatchedAt: "2026-04-19T01:00:00Z", LastSeenAt: "2026-04-19T02:00:00Z"},
		},
	})
	fixedTime := time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC)
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

	if _, err := (Service{UserHomeDir: func() (string, error) { return userHome, nil }}).Read(); err != nil {
		t.Fatalf("read watchlist: %v", err)
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
		t.Fatalf("expected read to preserve watchlist bytes\nbefore:\n%s\nafter:\n%s", beforeData, afterData)
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("expected read to preserve watchlist mtime, got %s want %s", afterInfo.ModTime(), beforeInfo.ModTime())
	}
}

func TestReadReturnsParseErrorForInvalidWatchlist(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")
	if err := os.MkdirAll(filepath.Dir(watchlistPath), 0o755); err != nil {
		t.Fatalf("mkdir watchlist dir: %v", err)
	}
	if err := os.WriteFile(watchlistPath, []byte(`{"version":`), 0o644); err != nil {
		t.Fatalf("write invalid watchlist: %v", err)
	}

	_, err := Service{UserHomeDir: func() (string, error) { return userHome, nil }}.Read()
	if err == nil || !strings.Contains(err.Error(), "parse watchlist.json") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestReadReturnsErrorForUnsupportedVersion(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")
	writeWatchlistFile(t, watchlistPath, File{Version: version + 1})

	_, err := Service{UserHomeDir: func() (string, error) { return userHome, nil }}.Read()
	if err == nil || !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
}

func TestTouchUsesEasyharnessHomeOverride(t *testing.T) {
	customHome := filepath.Join(t.TempDir(), "custom-home")
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)

	svc := Service{
		LookupEnv: func(key string) (string, bool) {
			if key == envHome {
				return customHome, true
			}
			return "", false
		},
		UserHomeDir: func() (string, error) { return t.TempDir(), nil },
	}
	if err := svc.Touch(workdir); err != nil {
		t.Fatalf("touch watchlist: %v", err)
	}

	if _, err := os.Stat(filepath.Join(customHome, "watchlist.json")); err != nil {
		t.Fatalf("expected watchlist in custom home, err=%v", err)
	}
}

func TestTouchUsesRelativeEasyharnessHomeOverrideUnderUserHome(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)
	firstCwd := t.TempDir()
	secondCwd := t.TempDir()
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalCwd); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()

	svc := Service{
		LookupEnv: func(key string) (string, bool) {
			if key == envHome {
				return "relative-home", true
			}
			return "", false
		},
		UserHomeDir: func() (string, error) { return userHome, nil },
	}
	for _, cwd := range []string{firstCwd, secondCwd} {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("chdir %q: %v", cwd, err)
		}
		if err := svc.Touch(workdir); err != nil {
			t.Fatalf("touch watchlist from %q: %v", cwd, err)
		}
	}

	got := readWatchlistFile(t, filepath.Join(userHome, "relative-home", "watchlist.json"))
	if len(got.Workspaces) != 1 {
		t.Fatalf("expected one workspace in stable relative override root, got %#v", got.Workspaces)
	}
}

func TestTouchRejectsRelativeEasyharnessHomeOverrideThatEscapesUserHome(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)

	svc := Service{
		LookupEnv: func(key string) (string, bool) {
			if key == envHome {
				return "../escape", true
			}
			return "", false
		},
		UserHomeDir: func() (string, error) { return userHome, nil },
	}
	if err := svc.Touch(workdir); err == nil || !strings.Contains(err.Error(), "escapes user home") {
		t.Fatalf("expected escaping relative override error, got %v", err)
	}
}

func TestTouchPreservesWatchedAtAndRefreshesLastSeenAt(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	svc.Now = func() time.Time { return time.Date(2026, 4, 19, 1, 0, 0, 0, time.UTC) }
	if err := svc.Touch(workdir); err != nil {
		t.Fatalf("initial touch: %v", err)
	}
	initial := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))

	svc.Now = func() time.Time { return time.Date(2026, 4, 19, 2, 0, 0, 0, time.UTC) }
	if err := svc.Touch(workdir); err != nil {
		t.Fatalf("second touch: %v", err)
	}
	updated := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if len(updated.Workspaces) != 1 {
		t.Fatalf("expected one workspace after repeated touch, got %#v", updated.Workspaces)
	}
	if updated.Workspaces[0].WatchedAt != initial.Workspaces[0].WatchedAt {
		t.Fatalf("expected watched_at to stay stable, got %#v -> %#v", initial.Workspaces[0], updated.Workspaces[0])
	}
	if updated.Workspaces[0].LastSeenAt == initial.Workspaces[0].LastSeenAt {
		t.Fatalf("expected last_seen_at to refresh, got %#v -> %#v", initial.Workspaces[0], updated.Workspaces[0])
	}
}

func TestTouchConvergesSymlinkedWorkspacePaths(t *testing.T) {
	userHome := t.TempDir()
	target := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, target)
	symlink := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	if err := svc.Touch(symlink); err != nil {
		t.Fatalf("touch symlinked path: %v", err)
	}
	if err := svc.Touch(target); err != nil {
		t.Fatalf("touch resolved path: %v", err)
	}

	got := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if len(got.Workspaces) != 1 {
		t.Fatalf("expected one converged workspace, got %#v", got.Workspaces)
	}
	canonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolve canonical workspace: %v", err)
	}
	if got.Workspaces[0].WorkspacePath != canonical {
		t.Fatalf("expected canonical target path %q, got %#v", target, got.Workspaces[0])
	}
}

func TestTouchConcurrentWritesPreserveUnrelatedRecords(t *testing.T) {
	userHome := t.TempDir()
	root := t.TempDir()
	first := filepath.Join(root, "workspace-a")
	second := filepath.Join(root, "workspace-b")
	for _, path := range []string{first, second} {
		seedGitWorkspace(t, path)
	}

	var wg sync.WaitGroup
	for i, path := range []string{first, second} {
		wg.Add(1)
		go func(idx int, workdir string) {
			defer wg.Done()
			svc := Service{
				UserHomeDir: func() (string, error) { return userHome, nil },
				Now: func() time.Time {
					return time.Date(2026, 4, 19, 3, idx, 0, 0, time.UTC)
				},
			}
			if err := svc.Touch(workdir); err != nil {
				t.Errorf("touch %q: %v", workdir, err)
			}
		}(i, path)
	}
	wg.Wait()

	got := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if len(got.Workspaces) != 2 {
		t.Fatalf("expected both workspaces after concurrent touches, got %#v", got.Workspaces)
	}
}

func TestTouchCoalescesNonAdjacentDuplicateWorkspaceRecords(t *testing.T) {
	userHome := t.TempDir()
	watchlistPath := filepath.Join(userHome, ".easyharness", "watchlist.json")
	first := filepath.Join(t.TempDir(), "workspace-a")
	second := filepath.Join(t.TempDir(), "workspace-b")
	for _, path := range []string{first, second} {
		seedGitWorkspace(t, path)
	}
	canonicalFirst, err := filepath.EvalSymlinks(first)
	if err != nil {
		t.Fatalf("resolve canonical first workspace: %v", err)
	}
	canonicalSecond, err := filepath.EvalSymlinks(second)
	if err != nil {
		t.Fatalf("resolve canonical second workspace: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(watchlistPath), 0o755); err != nil {
		t.Fatalf("mkdir watchlist dir: %v", err)
	}
	seed := File{
		Version: version,
		Workspaces: []Workspace{
			{WorkspacePath: canonicalFirst, WatchedAt: "2026-04-19T01:00:00Z", LastSeenAt: "2026-04-19T01:00:00Z"},
			{WorkspacePath: canonicalSecond, WatchedAt: "2026-04-19T01:05:00Z", LastSeenAt: "2026-04-19T01:05:00Z"},
			{WorkspacePath: canonicalFirst, WatchedAt: "2026-04-19T01:10:00Z", LastSeenAt: "2026-04-19T01:10:00Z"},
		},
	}
	payload, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed watchlist: %v", err)
	}
	if err := os.WriteFile(watchlistPath, payload, 0o644); err != nil {
		t.Fatalf("write seed watchlist: %v", err)
	}

	svc := Service{
		UserHomeDir: func() (string, error) { return userHome, nil },
		Now: func() time.Time {
			return time.Date(2026, 4, 19, 2, 0, 0, 0, time.UTC)
		},
	}
	if err := svc.Touch(first); err != nil {
		t.Fatalf("touch watchlist: %v", err)
	}

	got := readWatchlistFile(t, watchlistPath)
	if len(got.Workspaces) != 2 {
		t.Fatalf("expected duplicate workspace records to coalesce, got %#v", got.Workspaces)
	}
	if got.Workspaces[0].WorkspacePath != canonicalFirst || got.Workspaces[1].WorkspacePath != canonicalSecond {
		t.Fatalf("unexpected coalesced order/content: %#v", got.Workspaces)
	}
	if got.Workspaces[0].WatchedAt != "2026-04-19T01:00:00Z" {
		t.Fatalf("expected earliest watched_at to survive, got %#v", got.Workspaces[0])
	}
	if got.Workspaces[0].LastSeenAt != "2026-04-19T02:00:00Z" {
		t.Fatalf("expected refreshed last_seen_at on merged workspace, got %#v", got.Workspaces[0])
	}
	if got.Workspaces[1].LastSeenAt != "2026-04-19T01:05:00Z" {
		t.Fatalf("expected unrelated workspace to remain unchanged, got %#v", got.Workspaces[1])
	}
}

func TestTouchUsesGitWorkspaceRootForNestedDirectory(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, root)
	nested := filepath.Join(root, "docs", "plans")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested workspace path: %v", err)
	}

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	if err := svc.Touch(nested); err != nil {
		t.Fatalf("touch nested path: %v", err)
	}

	got := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if len(got.Workspaces) != 1 {
		t.Fatalf("expected one workspace, got %#v", got.Workspaces)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve canonical root: %v", err)
	}
	if got.Workspaces[0].WorkspacePath != canonicalRoot {
		t.Fatalf("expected git workspace root %q, got %#v", canonicalRoot, got.Workspaces[0])
	}
}

func TestTouchReturnsErrNotGitWorkspaceOutsideGitCheckout(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	err := svc.Touch(workdir)
	if !errors.Is(err, ErrNotGitWorkspace) {
		t.Fatalf("expected ErrNotGitWorkspace, got %v", err)
	}
}

func TestTouchRejectsFakeGitMarker(t *testing.T) {
	userHome := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(filepath.Join(workdir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir fake git marker: %v", err)
	}

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	err := svc.Touch(workdir)
	if !errors.Is(err, ErrNotGitWorkspace) {
		t.Fatalf("expected fake git marker to be rejected, got %v", err)
	}
}

func TestTouchRegistersLinkedGitWorktree(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, root)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("root repo\n"), 0o644); err != nil {
		t.Fatalf("write root repo file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "fixture")

	linked := filepath.Join(t.TempDir(), "linked-worktree")
	runGit(t, root, "worktree", "add", "-b", "linked-branch", linked, "HEAD")

	svc := Service{UserHomeDir: func() (string, error) { return userHome, nil }}
	if err := svc.Touch(linked); err != nil {
		t.Fatalf("touch linked worktree: %v", err)
	}

	got := readWatchlistFile(t, filepath.Join(userHome, ".easyharness", "watchlist.json"))
	if len(got.Workspaces) != 1 {
		t.Fatalf("expected one linked worktree record, got %#v", got.Workspaces)
	}
	canonicalLinked, err := filepath.EvalSymlinks(linked)
	if err != nil {
		t.Fatalf("resolve linked worktree: %v", err)
	}
	if got.Workspaces[0].WorkspacePath != canonicalLinked {
		t.Fatalf("expected linked worktree path %q, got %#v", canonicalLinked, got.Workspaces[0])
	}
}

func readWatchlistFile(t *testing.T, path string) File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read watchlist file: %v", err)
	}
	var decoded File
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode watchlist file: %v\n%s", err, data)
	}
	return decoded
}

func writeWatchlistFile(t *testing.T, path string, file File) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir watchlist dir: %v", err)
	}
	payload, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatalf("marshal watchlist file: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write watchlist file: %v", err)
	}
}

func seedGitWorkspace(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir git workspace root %q: %v", root, err)
	}
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Codex Test")
	runGit(t, root, "config", "user.email", "codex@example.com")
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
