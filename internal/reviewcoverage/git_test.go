package reviewcoverage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureCandidateRequiresCommittedCleanGitWorktree(t *testing.T) {
	root := t.TempDir()
	if _, err := CaptureCandidate(root); err == nil || !strings.Contains(err.Error(), "committed HEAD") {
		t.Fatalf("expected non-git rejection, got %v", err)
	}

	initGit(t, root)
	writeFile(t, root, "tracked.txt", "one\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")
	want := git(t, root, "rev-parse", "HEAD")
	candidate, err := CaptureCandidate(root)
	if err != nil {
		t.Fatalf("capture clean candidate: %v", err)
	}
	if candidate.HeadSHA != want {
		t.Fatalf("expected %s, got %#v", want, candidate)
	}

	writeFile(t, root, "tracked.txt", "two\n")
	if _, err := CaptureCandidate(root); err == nil || !strings.Contains(err.Error(), "tracked.txt") {
		t.Fatalf("expected tracked dirt rejection, got %v", err)
	}
}

func TestCaptureCandidateRequiresRuntimeArtifactsToBeIgnored(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	writeFile(t, root, "tracked.txt", "one\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "initial")

	writeFile(t, root, ".local/harness/plans/test/state.json", "{}\n")
	if _, err := CaptureCandidate(root); err == nil || !strings.Contains(err.Error(), ".local/harness") {
		t.Fatalf("expected unignored runtime rejection, got %v", err)
	}
	writeFile(t, root, ".gitignore", ".local/\n")
	git(t, root, "add", ".gitignore")
	git(t, root, "commit", "-m", "ignore runtime")
	if _, err := CaptureCandidate(root); err != nil {
		t.Fatalf("ignored command runtime should not dirty candidate: %v", err)
	}
}

func initGit(t *testing.T, root string) {
	t.Helper()
	git(t, root, "init")
	git(t, root, "config", "user.name", "Review Coverage Test")
	git(t, root, "config", "user.email", "review@example.com")
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, data)
	}
	return strings.TrimSpace(string(data))
}

func writeFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
