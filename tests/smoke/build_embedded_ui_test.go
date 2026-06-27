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

	repoRoot := support.CopyInstallerFixture(t)
	fakeBin := newBuildEmbeddedUITestBin(t)
	support.WriteFixtureFile(t, filepath.Join(fakeBin, "pnpm"), "#!/bin/sh\nprintf 'unexpected pnpm invocation\\n' >&2\nexit 99\n", 0o755)

	result := support.RunCommand(
		t,
		repoRoot,
		support.EnvWithOverrides(t, map[string]string{
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

	repoRoot := support.CopyInstallerFixture(t)
	fakeBin := newBuildEmbeddedUITestBin(t)
	support.WriteFixtureFile(t, filepath.Join(fakeBin, "node"), "#!/bin/sh\nexit 0\n", 0o755)

	result := support.RunCommand(
		t,
		repoRoot,
		support.EnvWithOverrides(t, map[string]string{
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

func TestBuildEmbeddedUIScriptRetriesTransientEsbuildServiceFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("embedded UI build smoke tests require a POSIX shell")
	}

	repoRoot := support.CopyInstallerFixture(t)
	fakeBin := newBuildEmbeddedUITestBin(t)
	callLog := filepath.Join(t.TempDir(), "pnpm-calls.log")
	buildState := filepath.Join(t.TempDir(), "pnpm-build-state")
	support.WriteFixtureFile(t, filepath.Join(fakeBin, "node"), "#!/bin/sh\nexit 0\n", 0o755)
	support.WriteFixtureFile(t, filepath.Join(fakeBin, "pnpm"), `#!/bin/sh
set -eu

printf '%s\n' "$*" >> "${PNPM_CALL_LOG}"
case "$1" in
  install)
    exit 0
    ;;
  build)
    if [ ! -f "${PNPM_BUILD_STATE}" ]; then
      printf 'seen\n' > "${PNPM_BUILD_STATE}"
      printf 'Error: The service was stopped: write EPIPE\n' >&2
      exit 1
    fi
    printf 'vite build ok\n'
    exit 0
    ;;
esac

printf 'unexpected pnpm invocation: %s\n' "$*" >&2
exit 99
`, 0o755)

	result := support.RunCommand(
		t,
		repoRoot,
		support.EnvWithOverrides(t, map[string]string{
			"PATH":             fakeBin,
			"PNPM_CALL_LOG":    callLog,
			"PNPM_BUILD_STATE": buildState,
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "build-embedded-ui"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("expected build-embedded-ui retry to pass\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	callLogBytes, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatalf("read pnpm call log: %v", err)
	}
	if buildCalls := strings.Count(string(callLogBytes), "build\n"); buildCalls != 2 {
		t.Fatalf("expected exactly two pnpm build calls, got %d\nlog:\n%s", buildCalls, string(callLogBytes))
	}
	support.RequireContains(t, result.Stderr, "Embedded UI build hit a transient esbuild service failure; retrying once.")
	support.RequireContains(t, result.Stdout, "vite build ok")
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
