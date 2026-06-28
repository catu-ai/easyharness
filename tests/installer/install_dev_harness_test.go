//go:build installer_smoke

package installer_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestInstallDevHarnessDefaultsToUserLocalBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	tempHome := t.TempDir()
	firstPathDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(firstPathDir, 0o755); err != nil {
		t.Fatalf("mkdir first PATH dir: %v", err)
	}

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": support.InstallerPath(t, firstPathDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	expectedWrapper := filepath.Join(tempHome, ".local", "bin", "harness")
	retiredGlobalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+expectedWrapper)
	support.RequireFileExists(t, expectedWrapper)
	support.RequireFileMissing(t, filepath.Join(firstPathDir, "harness"))
	support.RequireFileMissing(t, retiredGlobalFallback)

	info, err := os.Lstat(expectedWrapper)
	if err != nil {
		t.Fatalf("lstat wrapper: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to be a wrapper file, not a symlink", expectedWrapper)
	}
}

func TestInstallDevHarnessHelpDoesNotMentionGlobalFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--help",
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness --help failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	if strings.Contains(result.CombinedOutput(), "--global") {
		t.Fatalf("expected help output to omit removed --global flag\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}

func TestInstallDevHarnessRejectsRemovedGlobalFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected removed --global flag to fail\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stderr, "Unknown argument: --global")
}

func TestInstallDevHarnessVerifiesPATHResolvedWrapperWhenInstallDirIsAlreadyOnPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+wrapperPath)
	support.RequireContains(t, result.Stdout, "Verified harness on PATH at "+wrapperPath)
}

func TestInstallDevHarnessWrapperDispatchesToCurrentWorktreeOverStablePathFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	_, nestedDir := newFakeWorktree(t)
	wrapperResult := support.RunCommand(
		t,
		nestedDir,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode != 0 {
		t.Fatalf("wrapper failed with exit %d\nstdout:\n%s\nstderr:\n%s", wrapperResult.ExitCode, wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stdout, "fake worktree harness")
	support.RequireContains(t, wrapperResult.Stdout, "args=status")
	if strings.Contains(wrapperResult.CombinedOutput(), "stable fallback harness") {
		t.Fatalf("expected wrapper to prefer the worktree-local binary over the stable PATH fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}
}

func TestInstallDevHarnessWrapperRequiresStableHarnessOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	wrapperResult := support.RunCommand(
		t,
		otherProject,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": strings.Join([]string{installDir, "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, string(os.PathListSeparator)),
		}),
		wrapperPath,
		"--help",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected wrapper without a stable PATH fallback to fail outside easyharness source trees\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "Could not find an easyharness source tree")
	support.RequireContains(t, wrapperResult.Stderr, "no stable harness binary is available on PATH")
	support.RequireContains(t, wrapperResult.Stderr, "Install the stable easyharness release with Homebrew")
}

func TestInstallDevHarnessWrapperUsesStableHarnessOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	helpResult := support.RunCommand(
		t,
		otherProject,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper stable PATH fallback failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
}

func TestInstallDevHarnessWrapperSkipsOtherManagedWrappersOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	cases := []struct {
		name   string
		script string
	}{
		{
			name:   "marker wrapper",
			script: fakeManagedWrapperScript("unexpected managed wrapper"),
		},
		{
			name:   "legacy wrapper",
			script: fakeLegacyManagedWrapperScript("unexpected legacy managed wrapper"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := support.CopyInstallerFixture(t)
			installDir := filepath.Join(t.TempDir(), "path-bin")
			managedDir := t.TempDir()
			stableDir, _ := newFakeStableHarness(t)
			support.WriteFixtureFile(t, filepath.Join(managedDir, "harness"), tc.script, 0o755)

			result := support.RunCommand(
				t,
				repoRoot,
				support.InstallerEnv(t, map[string]string{
					"HOME": t.TempDir(),
					"PATH": support.InstallerPath(t, installDir, managedDir, stableDir),
				}),
				"/bin/bash",
				filepath.Join(repoRoot, "scripts", "install-dev-harness"),
				"--install-dir", installDir,
			)
			if result.ExitCode != 0 {
				t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
			}

			wrapperPath := filepath.Join(installDir, "harness")
			otherProject := t.TempDir()
			helpResult := support.RunCommand(
				t,
				otherProject,
				support.EnvWithOverrides(t, map[string]string{
					"PATH": support.InstallerPath(t, installDir, managedDir, stableDir),
				}),
				wrapperPath,
				"--help",
			)
			if helpResult.ExitCode != 0 {
				t.Fatalf("wrapper with other managed wrapper on PATH failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
			}

			support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
			if strings.Contains(helpResult.CombinedOutput(), "unexpected managed wrapper") || strings.Contains(helpResult.CombinedOutput(), "unexpected legacy managed wrapper") {
				t.Fatalf("expected wrapper to skip other managed wrappers on PATH\nstdout:\n%s\nstderr:\n%s", helpResult.Stdout, helpResult.Stderr)
			}
		})
	}
}

func TestInstallDevHarnessWrapperSkipsSymlinkAliasesOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	aliasOneDir := t.TempDir()
	aliasTwoDir := t.TempDir()
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	if err := os.Symlink(wrapperPath, filepath.Join(aliasOneDir, "harness")); err != nil {
		t.Fatalf("create first wrapper alias: %v", err)
	}
	if err := os.Symlink(filepath.Join(aliasOneDir, "harness"), filepath.Join(aliasTwoDir, "harness")); err != nil {
		t.Fatalf("create second wrapper alias: %v", err)
	}

	otherProject := t.TempDir()
	helpResult := support.RunCommandWithTimeout(
		t,
		5*time.Second,
		otherProject,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, aliasOneDir, aliasTwoDir, installDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper with symlink aliases on PATH failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
}

func TestInstallDevHarnessWrapperSkipsRepoLocalDevBinaryOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	devDir, _ := newFakeDevHarness(t)
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, devDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	otherProject := t.TempDir()
	helpResult := support.RunCommand(
		t,
		otherProject,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, devDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper with repo-local dev binary on PATH failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
	if strings.Contains(helpResult.CombinedOutput(), "unexpected dev binary") {
		t.Fatalf("expected wrapper to skip repo-local dev binaries on PATH\nstdout:\n%s\nstderr:\n%s", helpResult.Stdout, helpResult.Stderr)
	}
}

func TestInstallDevHarnessVersionReportsDevModeAndPathInsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	versionResult := support.RunCommand(
		t,
		repoRoot,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("wrapper version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}

	if mode := support.RequireVersionField(t, versionResult.Stdout, "mode"); mode != "dev" {
		t.Fatalf("expected dev mode from repo-local worktree binary, got %q\noutput:\n%s", mode, versionResult.Stdout)
	}
	versionSeedBytes, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	expectedVersion := "v" + strings.TrimSpace(string(versionSeedBytes)) + "-dev"
	if version := support.RequireVersionField(t, versionResult.Stdout, "version"); version != expectedVersion {
		t.Fatalf("expected dev version %q, got %q\noutput:\n%s", expectedVersion, version, versionResult.Stdout)
	}
	expectedPath := filepath.Join(repoRoot, ".local", "bin", "harness")
	if resolvedPath, err := filepath.EvalSymlinks(expectedPath); err == nil {
		expectedPath = resolvedPath
	}
	if path := support.RequireVersionField(t, versionResult.Stdout, "path"); path != expectedPath {
		t.Fatalf("expected repo-local dev path %q, got %q\noutput:\n%s", expectedPath, path, versionResult.Stdout)
	}
}

func TestInstallDevHarnessVersionReportsStableModeAndPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	versionResult := support.RunCommand(
		t,
		otherProject,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("wrapper version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}

	if mode := support.RequireVersionField(t, versionResult.Stdout, "mode"); mode != "release" {
		t.Fatalf("expected release mode from stable PATH fallback, got %q\noutput:\n%s", mode, versionResult.Stdout)
	}
	if commit := support.RequireVersionField(t, versionResult.Stdout, "commit"); commit != "stable-test-commit" {
		t.Fatalf("expected stable fallback commit %q, got %q\noutput:\n%s", "stable-test-commit", commit, versionResult.Stdout)
	}
	if strings.Contains(versionResult.Stdout, `"path"`) {
		t.Fatalf("expected stable release fallback to omit path, got %q", versionResult.Stdout)
	}
}

func TestInstallDevHarnessReplacesLegacyManagedWrapperWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	legacyWrapper := `#!/usr/bin/env bash
set -euo pipefail

find_repo_root() {
  local root=""
  if command -v git >/dev/null 2>&1; then
    root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    if [[ -n "${root}" && -f "${root}/scripts/install-dev-harness" && -f "${root}/cmd/harness/main.go" ]]; then
      printf '%s\n' "${root}"
      return 0
    fi
  fi

  local dir="${PWD}"
  while :; do
    if [[ -f "${dir}/scripts/install-dev-harness" && -f "${dir}/cmd/harness/main.go" ]]; then
      printf '%s\n' "${dir}"
      return 0
    fi
    if [[ "${dir}" == "/" ]]; then
      break
    fi
    dir="$(dirname "${dir}")"
  done

  return 1
}

if ! repo_root="$(find_repo_root)"; then
  echo "Could not find a microharness worktree from ${PWD}." >&2
  echo "Run harness from inside a microharness checkout, or call a repo-local binary directly." >&2
  exit 1
fi

binary_path="${repo_root}/.local/bin/harness"
if [[ ! -x "${binary_path}" ]]; then
  echo "No repo-local harness binary found at ${binary_path}." >&2
  echo "Run scripts/install-dev-harness from this worktree first." >&2
  exit 1
fi

exec "${binary_path}" "$@"
`
	support.WriteFixtureFile(t, wrapperPath, legacyWrapper, 0o755)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	refreshed, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read refreshed wrapper: %v", err)
	}
	support.RequireContains(t, string(refreshed), "# easyharness-install-dev-wrapper")
}

func TestInstallDevHarnessReplacesLegacySymlinkedBinaryWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	cases := []struct {
		name       string
		moduleLine string
	}{
		{
			name:       "superharness namespace",
			moduleLine: "module github.com/yzhang1918/superharness\n",
		},
		{
			name:       "personal microharness namespace",
			moduleLine: "module github.com/yzhang1918/microharness\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := support.CopyInstallerFixture(t)
			installDir := filepath.Join(t.TempDir(), "path-bin")
			if err := os.MkdirAll(installDir, 0o755); err != nil {
				t.Fatalf("mkdir install dir: %v", err)
			}

			legacyRoot := t.TempDir()
			for _, dir := range []string{
				filepath.Join(legacyRoot, "scripts"),
				filepath.Join(legacyRoot, "cmd", "harness"),
				filepath.Join(legacyRoot, ".local", "bin"),
			} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("mkdir legacy dir %s: %v", dir, err)
				}
			}
			support.WriteFixtureFile(t, filepath.Join(legacyRoot, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
			support.WriteFixtureFile(t, filepath.Join(legacyRoot, "cmd", "harness", "main.go"), "package main\n", 0o644)
			support.WriteFixtureFile(t, filepath.Join(legacyRoot, "go.mod"), tc.moduleLine, 0o644)
			support.WriteFixtureFile(t, filepath.Join(legacyRoot, ".local", "bin", "harness"), "#!/bin/sh\nexit 0\n", 0o755)

			wrapperPath := filepath.Join(installDir, "harness")
			if err := os.Symlink(filepath.Join(legacyRoot, ".local", "bin", "harness"), wrapperPath); err != nil {
				t.Fatalf("create legacy symlink: %v", err)
			}

			result := support.RunCommand(
				t,
				repoRoot,
				support.InstallerEnv(t, map[string]string{
					"HOME": t.TempDir(),
					"PATH": support.InstallerPath(t),
				}),
				"/bin/bash",
				filepath.Join(repoRoot, "scripts", "install-dev-harness"),
				"--install-dir", installDir,
			)
			if result.ExitCode != 0 {
				t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
			}

			info, err := os.Lstat(wrapperPath)
			if err != nil {
				t.Fatalf("lstat refreshed wrapper: %v", err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("expected refreshed wrapper to replace the legacy symlink")
			}
			refreshed, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read refreshed wrapper: %v", err)
			}
			support.RequireContains(t, string(refreshed), "# easyharness-install-dev-wrapper")
		})
	}
}

func TestInstallDevHarnessWrapperDoesNotUseStablePathFallbackInsideSourceTreeWithoutLocalBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	t.Parallel()

	repoRoot := support.CopyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := support.RunCommand(
		t,
		repoRoot,
		support.InstallerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	_, nestedDir := newFakeWorktreeWithoutLocalBinary(t)
	wrapperPath := filepath.Join(installDir, "harness")
	wrapperResult := support.RunCommand(
		t,
		nestedDir,
		support.EnvWithOverrides(t, map[string]string{
			"PATH": support.InstallerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected source-tree invocation without local binary to fail\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "No repo-local harness binary found at ")
	support.RequireContains(t, wrapperResult.Stderr, filepath.Join(".local", "bin", "harness"))
	if strings.Contains(wrapperResult.CombinedOutput(), "stable fallback harness") {
		t.Fatalf("expected source-tree invocation to refuse the stable PATH fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}
}

func newFakeWorktree(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "scripts"),
		filepath.Join(root, "cmd", "harness"),
		filepath.Join(root, ".local", "bin"),
		filepath.Join(root, "nested", "dir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	support.WriteFixtureFile(t, filepath.Join(root, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
	support.WriteFixtureFile(t, filepath.Join(root, "cmd", "harness", "main.go"), "package main\n", 0o644)
	support.WriteFixtureFile(t, filepath.Join(root, "go.mod"), "module github.com/catu-ai/easyharness\n", 0o644)
	support.WriteFixtureFile(
		t,
		filepath.Join(root, ".local", "bin", "harness"),
		"#!/bin/sh\nprintf 'fake worktree harness\\n'\nprintf 'args=%s\\n' \"$*\"\n",
		0o755,
	)

	return root, filepath.Join(root, "nested", "dir")
}

func newFakeWorktreeWithoutLocalBinary(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "scripts"),
		filepath.Join(root, "cmd", "harness"),
		filepath.Join(root, "nested", "dir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	support.WriteFixtureFile(t, filepath.Join(root, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
	support.WriteFixtureFile(t, filepath.Join(root, "cmd", "harness", "main.go"), "package main\n", 0o644)
	support.WriteFixtureFile(t, filepath.Join(root, "go.mod"), "module github.com/catu-ai/easyharness\n", 0o644)

	return root, filepath.Join(root, "nested", "dir")
}

func newFakeStableHarness(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "harness")
	support.WriteFixtureFile(
		t,
		path,
		`#!/bin/sh
set -eu

case "${1:-}" in
  --help)
    printf 'stable fallback harness help\n'
    ;;
  --version)
    printf '{"version":"0.0.0","mode":"release","commit":"stable-test-commit"}\n'
    ;;
  *)
    printf 'stable fallback harness\n'
    printf 'args=%s\n' "$*"
    ;;
esac
`,
		0o755,
	)
	return dir, path
}

func fakeManagedWrapperScript(marker string) string {
	return "#!/bin/sh\n" +
		"# easyharness-install-dev-wrapper\n" +
		"printf '" + marker + "\\n'\n"
}

func fakeLegacyManagedWrapperScript(marker string) string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n\n" +
		"find_repo_root() {\n" +
		"  printf 'legacy-wrapper-root\\n'\n" +
		"}\n" +
		"# cmd/harness/main.go\n\n" +
		"printf '" + marker + "\\n'\n"
}

func newFakeDevHarness(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "harness")
	support.WriteFixtureFile(
		t,
		path,
		`#!/bin/sh
set -eu

case "${1:-}" in
  --help)
    printf 'unexpected dev binary\n'
    ;;
  --version)
    printf '{"version":"dev-test","mode":"dev","commit":"dev-test-commit","path":"%s"}\n' "$0"
    ;;
  *)
    printf 'unexpected dev binary\n'
    printf 'args=%s\n' "$*"
    ;;
esac
`,
		0o755,
	)
	return dir, path
}
