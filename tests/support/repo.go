package support

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type Workspace struct {
	Root string
}

func NewWorkspace(t *testing.T) *Workspace {
	t.Helper()
	return &Workspace{Root: t.TempDir()}
}

func (w *Workspace) Path(rel string) string {
	if rel == "" {
		return w.Root
	}
	return filepath.Join(w.Root, filepath.FromSlash(rel))
}

func (w *Workspace) WriteJSON(t *testing.T, rel string, value any) string {
	t.Helper()

	path := w.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", rel, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func (w *Workspace) WriteFile(t *testing.T, rel string, data []byte) string {
	t.Helper()

	path := w.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return path
}

// CommitAll initializes a deterministic fixture repository when needed and
// commits the current tracked candidate. Review specs, runtime state, and
// reviewer submissions live under ignored paths so review start and aggregate
// exercise the same clean-worktree boundary as a real repository.
func (w *Workspace) CommitAll(t *testing.T, message string) string {
	t.Helper()

	gitDir := w.Path(".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		w.WriteFile(t, ".gitignore", []byte(".local/\ntmp/\n"))
		runGit(t, w.Root, "init", "--quiet")
		runGit(t, w.Root, "config", "user.name", "easyharness tests")
		runGit(t, w.Root, "config", "user.email", "tests@easyharness.local")
	} else if err != nil {
		t.Fatalf("stat fixture git directory: %v", err)
	}

	runGit(t, w.Root, "add", "--all")
	staged := exec.Command("git", "-C", w.Root, "diff", "--cached", "--quiet")
	if err := staged.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("inspect staged fixture changes: %v", err)
		}
		runGit(t, w.Root, "commit", "--quiet", "-m", message)
	} else if _, err := os.Stat(filepath.Join(gitDir, "HEAD")); os.IsNotExist(err) {
		runGit(t, w.Root, "commit", "--quiet", "--allow-empty", "-m", message)
	}

	output := runGit(t, w.Root, "rev-parse", "HEAD")
	return strings.TrimSpace(output)
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
