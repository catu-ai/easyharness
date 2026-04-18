package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestBuildEmbeddedUIScriptFailsWithActionableMessageWhenNodeIsMissingButPnpmExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("embedded UI build smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	fakeBin := newBuildEmbeddedUITestBin(t)
	writeFixtureFile(t, filepath.Join(fakeBin, "pnpm"), "#!/bin/sh\nprintf 'unexpected pnpm invocation\\n' >&2\nexit 99\n", 0o755)

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, map[string]string{
			"PATH": fakeBin,
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "build-embedded-ui"),
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected build-embedded-ui to fail without node\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stderr, "Node.js is required to build embedded UI assets.")
	support.RequireContains(t, result.Stderr, "Install Node.js and pnpm, then rerun this command.")
	if strings.Contains(result.CombinedOutput(), "unexpected pnpm invocation") {
		t.Fatalf("expected node preflight to fail before pnpm runs\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}

func TestBuildEmbeddedUIScriptFailsWithActionableMessageWhenPnpmIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("embedded UI build smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	fakeBin := newBuildEmbeddedUITestBin(t)
	writeFixtureFile(t, filepath.Join(fakeBin, "node"), "#!/bin/sh\nexit 0\n", 0o755)

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, map[string]string{
			"PATH": fakeBin,
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "build-embedded-ui"),
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected build-embedded-ui to fail without pnpm\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stderr, "pnpm is required to build embedded UI assets.")
	support.RequireContains(t, result.Stderr, "Install Node.js and pnpm, then rerun this command.")
}

func newBuildEmbeddedUITestBin(t *testing.T) string {
	t.Helper()

	fakeBin := t.TempDir()
	dirnamePath, err := exec.LookPath("dirname")
	if err != nil {
		t.Fatalf("find dirname on PATH: %v", err)
	}
	if err := os.Symlink(dirnamePath, filepath.Join(fakeBin, "dirname")); err != nil {
		t.Fatalf("symlink dirname helper: %v", err)
	}
	return fakeBin
}
