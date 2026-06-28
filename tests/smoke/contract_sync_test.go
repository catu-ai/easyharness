package smoke_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestSyncContractArtifactsCheckPassesForCurrentRepo(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	result := support.RunCommand(t, repoRoot, nil, filepath.Join(repoRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode != 0 {
		t.Fatalf("sync-contract-artifacts --check exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "Contract schemas are in sync.") {
		t.Fatalf("unexpected check output:\n%s", result.CombinedOutput())
	}
}

func TestSyncContractArtifactsCheckIgnoresTransientFrontendDependencyDirs(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cloneRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cloneRoot, 0o755); err != nil {
		t.Fatalf("mkdir clone root: %v", err)
	}
	copyCurrentRepo(t, repoRoot, cloneRoot)

	transientDir := filepath.Join(cloneRoot, "web", "node_modules", ".pnpm", "vite_tmp_missing")
	if err := os.MkdirAll(transientDir, 0o755); err != nil {
		t.Fatalf("mkdir transient dependency dir: %v", err)
	}
	if err := os.Chmod(transientDir, 0); err != nil {
		t.Fatalf("chmod transient dependency dir: %v", err)
	}
	defer func() {
		if err := os.Chmod(transientDir, 0o755); err != nil && !os.IsNotExist(err) {
			t.Fatalf("restore transient dependency dir permissions: %v", err)
		}
	}()

	result := support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode != 0 {
		t.Fatalf("sync-contract-artifacts --check should ignore transient frontend dependency dirs, exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "Contract schemas are in sync.") {
		t.Fatalf("unexpected check output:\n%s", result.CombinedOutput())
	}
}

func TestSyncContractArtifactsCheckFailsOnStaleGeneratedFiles(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cloneRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cloneRoot, 0o755); err != nil {
		t.Fatalf("mkdir clone root: %v", err)
	}
	copyCurrentRepo(t, repoRoot, cloneRoot)

	stalePath := filepath.Join(cloneRoot, "schema", "index.json")
	if err := os.WriteFile(stalePath, []byte("{\"stale\":true}\n"), 0o644); err != nil {
		t.Fatalf("write stale schema: %v", err)
	}

	result := support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode == 0 {
		t.Fatalf("expected stale generated file check to fail:\n%s", result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "stale generated file") {
		t.Fatalf("expected stale-file error, got:\n%s", result.CombinedOutput())
	}

	result = support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"))
	if result.ExitCode != 0 {
		t.Fatalf("sync-contract-artifacts repair run exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}

	result = support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode != 0 {
		t.Fatalf("post-repair sync-contract-artifacts --check exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "Contract schemas are in sync.") {
		t.Fatalf("unexpected post-repair check output:\n%s", result.CombinedOutput())
	}
}

func TestSyncContractArtifactsCheckFailsOnDeprecatedGeneratedDocs(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cloneRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cloneRoot, 0o755); err != nil {
		t.Fatalf("mkdir clone root: %v", err)
	}
	copyCurrentRepo(t, repoRoot, cloneRoot)

	stalePath := filepath.Join(cloneRoot, "docs", "reference", "contracts", "README.md")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatalf("mkdir stale docs dir: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("# stale generated docs\n"), 0o644); err != nil {
		t.Fatalf("write stale docs: %v", err)
	}

	result := support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode == 0 {
		t.Fatalf("expected deprecated generated docs check to fail:\n%s", result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "unexpected generated file") {
		t.Fatalf("expected unexpected-file error, got:\n%s", result.CombinedOutput())
	}

	result = support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"))
	if result.ExitCode != 0 {
		t.Fatalf("sync-contract-artifacts cleanup run exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}

	result = support.RunCommand(t, cloneRoot, nil, filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	if result.ExitCode != 0 {
		t.Fatalf("post-repair sync-contract-artifacts --check exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "Contract schemas are in sync.") {
		t.Fatalf("unexpected post-repair check output:\n%s", result.CombinedOutput())
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected deprecated generated docs to be removed, got err=%v", err)
	}
}

func copyCurrentRepo(t *testing.T, src, dst string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), support.DefaultCommandTimeout)
	defer cancel()

	archive := exec.CommandContext(
		ctx,
		"tar",
		"-cf",
		"-",
		"--exclude=.git",
		"--exclude=.local",
		"--exclude=web/node_modules",
		"--exclude=internal/ui/generated/build",
		".",
	)
	archive.Dir = src
	extract := exec.CommandContext(ctx, "tar", "-xf", "-", "-C", dst)

	var archiveStderr bytes.Buffer
	var extractStderr bytes.Buffer
	archive.Stderr = &archiveStderr
	extract.Stderr = &extractStderr

	pipe, err := archive.StdoutPipe()
	if err != nil {
		t.Fatalf("archive stdout pipe: %v", err)
	}
	extract.Stdin = pipe

	if err := archive.Start(); err != nil {
		t.Fatalf("start archive: %v", err)
	}
	if err := extract.Start(); err != nil {
		t.Fatalf("start extract: %v", err)
	}
	if err := archive.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("archive repo timed out after %s: %v\nstderr:\n%s", support.DefaultCommandTimeout, err, archiveStderr.String())
		}
		t.Fatalf("archive repo: %v\nstderr:\n%s", err, archiveStderr.String())
	}
	if err := extract.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("extract repo timed out after %s: %v\nstderr:\n%s", support.DefaultCommandTimeout, err, extractStderr.String())
		}
		t.Fatalf("extract repo: %v\nstderr:\n%s", err, extractStderr.String())
	}
}
