package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/cli"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/timeline"
	version "github.com/catu-ai/easyharness/internal/version"
)

func TestPlanTemplateWritesOutputFile(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 17, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(t.TempDir(), "docs/plans/active/2026-03-17-test-plan.md")
	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
		"--source-type", "issue",
		"--source-ref", "#42",
	})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%s", exitCode, stderr.String())
	}
	ensurePlanSizeInFile(t, outputPath, "M")

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Contains(data, []byte("# CLI Generated Plan")) {
		t.Fatalf("generated file missing title:\n%s", data)
	}
}

func TestPlanTemplateDateSeedsCurrentLocalTimeOfDay(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 25, 14, 15, 16, 0, time.FixedZone("CST", 8*60*60))
	}

	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "Date Seeded Plan",
		"--date", "2026-03-20",
	})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "created_at: 2026-03-20T14:15:16+08:00") {
		t.Fatalf("expected date-seeded template to preserve current local time-of-day, got:\n%s", stdout.String())
	}
}

func TestPlanTemplateSizeFlagSeedsExplicitSize(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "Sized Plan",
		"--size", "XL",
	})
	if exitCode != 0 {
		t.Fatalf("expected explicit size template success, got %d: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "size: XL") {
		t.Fatalf("expected explicit size in template, got:\n%s", stdout.String())
	}
}

func TestPlanTemplateDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	home := t.TempDir()
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "Watchlist Exclusion",
	})
	if exitCode != 0 {
		t.Fatalf("plan template failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestVersionFlagPrintsJSONBuildInfo(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	modified := true
	app.Version = func() version.Info {
		return version.Info{
			Version:   "v0.0.0",
			Mode:      "dev",
			Commit:    "abc123",
			GoVersion: "go1.25.0",
			BuildTime: "2026-04-14T12:34:56Z",
			Modified:  &modified,
			Path:      "/tmp/harness",
		}
	}

	exitCode := app.Run([]string{"--version"})
	if exitCode != 0 {
		t.Fatalf("expected version exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr for version output, got %q", stderr.String())
	}
	var payload version.Info
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON version output: %v\n%s", err, stdout.String())
	}
	if payload.Mode != "dev" {
		t.Fatalf("expected mode in version output, got %#v", payload)
	}
	if payload.Commit != "abc123" {
		t.Fatalf("expected commit in version output, got %#v", payload)
	}
	if payload.Path != "/tmp/harness" {
		t.Fatalf("expected dev path in version output, got %#v", payload)
	}
	if payload.Version != "v0.0.0" {
		t.Fatalf("expected version in version output, got %#v", payload)
	}
	if payload.Modified == nil || !*payload.Modified {
		t.Fatalf("expected modified=true in version output, got %#v", payload)
	}
}

func TestVersionFlagOmitsDevOnlyFieldsOutsideDevMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Version = func() version.Info {
		return version.Info{
			Commit: "abc123",
			Mode:   "release",
		}
	}

	exitCode := app.Run([]string{"--version"})
	if exitCode != 0 {
		t.Fatalf("expected version exit code 0, got %d: %s", exitCode, stderr.String())
	}
	var payload version.Info
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON version output: %v\n%s", err, stdout.String())
	}
	if payload.Path != "" {
		t.Fatalf("expected release version output to omit path, got %#v", payload)
	}
	if payload.Modified != nil {
		t.Fatalf("expected release version output to omit modified, got %#v", payload)
	}
}

func TestVersionHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"--version", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected version help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness --version") {
		t.Fatalf("expected version help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for version help, got %q", stdout.String())
	}
}

func TestPlanLintCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 17, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(t.TempDir(), "docs/plans/active/2026-03-17-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	ensurePlanSizeInFile(t, outputPath, "M")

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"plan", "lint", outputPath})
	if exitCode != 0 {
		t.Fatalf("lint command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON lint output: %v\n%s", err, stdout.String())
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
}

func TestPlanTemplateHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "template", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: harness plan template")) {
		t.Fatalf("expected help text, got %s", stderr.String())
	}
}

func TestPlanTemplateLightweightFlagSeedsLocalVariant(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "template", "--title", "Lightweight Plan", "--lightweight"})
	if exitCode != 0 {
		t.Fatalf("expected lightweight template success, got %d: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workflow_profile: lightweight") {
		t.Fatalf("expected workflow_profile in template, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "size: XXS") {
		t.Fatalf("expected lightweight template to emit size XXS, got %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "### Step 2:") {
		t.Fatalf("expected lightweight template to collapse to one step, got %s", stdout.String())
	}
}

func TestPlanTemplateLightweightRejectsNonXXSSize(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "Bad Lightweight Plan",
		"--lightweight",
		"--size", "XS",
	})
	if exitCode != 1 {
		t.Fatalf("expected lightweight/non-XXS mismatch to fail with exit code 1, got %d: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), `lightweight templates must use size "XXS"`) {
		t.Fatalf("expected size mismatch error, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout on error, got %q", stdout.String())
	}
}

func TestRootHelpMentionsVersionFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"--help"})
	if exitCode != 0 {
		t.Fatalf("expected root help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness <command> [subcommand] [flags]") {
		t.Fatalf("expected root help usage, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--version       Print JSON build information for the running harness binary") {
		t.Fatalf("expected root help to mention --version, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "init            Install or refresh the managed bootstrap resources for the current repository") {
		t.Fatalf("expected root help to mention init, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "skills          Manage easyharness skill packages") {
		t.Fatalf("expected root help to mention skills, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "instructions    Manage easyharness instruction files and managed blocks") {
		t.Fatalf("expected root help to mention instructions, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ui              Start the local read-only harness UI workbench") {
		t.Fatalf("expected root help to mention ui, got %q", stderr.String())
	}
}

func TestUIHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"ui", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected ui help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness ui") {
		t.Fatalf("expected ui help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for ui help, got %q", stdout.String())
	}
}

func TestUIRejectsPositionalArguments(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"ui", "extra"})
	if exitCode != 2 {
		t.Fatalf("expected ui positional-arg exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness ui") {
		t.Fatalf("expected ui usage on positional args, got %q", stderr.String())
	}
}

func TestInitCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	exitCode := app.Run([]string{"init", "--dry-run"})
	if exitCode != 0 {
		t.Fatalf("init dry-run failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON init output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "init" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload["mode"] != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", payload)
	}
}

func TestInstructionsInstallCommandWritesManagedAssets(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"instructions", "install"})
	if exitCode != 0 {
		t.Fatalf("instructions install failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON instructions output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "instructions install" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to be written: %v", err)
	}
	assertWatchlistAbsent(t, home)
}

func TestSkillsCommandRejectsInvalidScope(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	exitCode := app.Run([]string{"skills", "install", "--scope", "bogus"})
	if exitCode != 1 {
		t.Fatalf("expected invalid scope exit code 1, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON skills output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "skills install" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected skills failure, got %#v", payload)
	}
}

func TestSkillsHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"skills", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness skills") {
		t.Fatalf("expected skills help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for skills help, got %q", stdout.String())
	}
}

func TestPlanLintHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "lint", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: harness plan lint")) {
		t.Fatalf("expected help text, got %s", stderr.String())
	}
}

func TestStatusCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("status command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON status output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "status" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestStatusCommandTouchesWatchlistForIdleWorkspace(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC)
	}

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("status command failed with %d: %s", exitCode, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(home, ".easyharness", "watchlist.json"))
	if err != nil {
		t.Fatalf("expected watchlist file after status, err=%v", err)
	}
	var payload struct {
		Workspaces []struct {
			WorkspacePath string `json:"workspace_path"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode watchlist: %v\n%s", err, data)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve canonical workspace: %v", err)
	}
	if len(payload.Workspaces) != 1 || payload.Workspaces[0].WorkspacePath != canonicalRoot {
		t.Fatalf("unexpected watchlist payload: %#v", payload)
	}
}

func TestStatusCommandTouchesWatchlistForIdleLinkedWorktree(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	repoRoot := filepath.Join(t.TempDir(), "workspace")
	home := t.TempDir()
	initGitRepoWithCommit(t, repoRoot)
	linked := filepath.Join(t.TempDir(), "linked-worktree")
	runGit(t, repoRoot, "worktree", "add", "-b", "linked-status-branch", linked, "HEAD")
	app.Getwd = func() (string, error) { return linked, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 5, 0, 0, time.UTC)
	}

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("status command failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistContainsWorkspace(t, home, linked)
}

func TestStatusCommandDoesNotTouchWatchlistOutsideGitWorkspace(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("status command failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestPlanLintDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	home := t.TempDir()
	app.UserHomeDir = func() (string, error) { return home, nil }
	outputPath := filepath.Join(t.TempDir(), "docs/plans/active/2026-03-17-test-plan.md")

	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	ensurePlanSizeInFile(t, outputPath, "M")

	stdout.Reset()
	stderr.Reset()
	exitCode := app.Run([]string{"plan", "lint", outputPath})
	if exitCode != 0 {
		t.Fatalf("lint command failed with %d: %s", exitCode, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".easyharness", "watchlist.json")); !os.IsNotExist(err) {
		t.Fatalf("expected plan lint to avoid touching watchlist, err=%v", err)
	}
}

func TestStatusCommandIgnoresWatchlistWriteFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.LookupEnv = func(string) (string, bool) { return "", false }
	app.UserHomeDir = func() (string, error) { return "", errors.New("watchlist-home-boom") }

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("expected status to succeed despite watchlist failure, got %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON status output: %v\n%s", err, stdout.String())
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected status success payload, got %#v", payload)
	}
}

func TestVersionCommandDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	home := t.TempDir()
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"--version"})
	if exitCode != 0 {
		t.Fatalf("version command failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestHelpDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	home := t.TempDir()
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"help"})
	if exitCode != 0 {
		t.Fatalf("help command failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestInitDryRunDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"init", "--dry-run"})
	if exitCode != 0 {
		t.Fatalf("init --dry-run failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestSkillsInstallDryRunDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"skills", "install", "--dry-run"})
	if exitCode != 0 {
		t.Fatalf("skills install --dry-run failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestUIHelpDoesNotTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	home := t.TempDir()
	app.UserHomeDir = func() (string, error) { return home, nil }

	exitCode := app.Run([]string{"ui", "--help"})
	if exitCode != 0 {
		t.Fatalf("ui help failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistAbsent(t, home)
}

func TestExecuteStartCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"execute", "start"})
	if exitCode != 0 {
		t.Fatalf("execute start command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON execute-start output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "execute start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "execution/step-1/implement", 1)

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus execute-start event, got %#v", timelineResult)
	}
	if timelineResult.Events[0].Command != "plan" || timelineResult.Events[1].Command != "execute start" {
		t.Fatalf("unexpected execute-start timeline events: %#v", timelineResult.Events)
	}
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestPlanApproveTouchesWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 14, 55, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Approve Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	ensurePlanSizeInFile(t, outputPath, "M")

	stdout.Reset()
	stderr.Reset()
	if exitCode := app.Run([]string{"plan", "approve", "--by", "human"}); exitCode != 0 {
		t.Fatalf("plan approve failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestExecuteStartIgnoresWatchlistWriteFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.LookupEnv = func(string) (string, bool) { return "", false }
	app.UserHomeDir = func() (string, error) { return "", errors.New("watchlist-home-boom") }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")

	stdout.Reset()
	stderr.Reset()
	exitCode := app.Run([]string{"execute", "start"})
	if exitCode != 0 {
		t.Fatalf("expected execute start to succeed despite watchlist failure, got %d: %s", exitCode, stderr.String())
	}
}

func TestExecuteStartRollsBackWhenTimelineAppendFails(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Generated Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if err := os.MkdirAll(filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/events.jsonl"), 0o755); err != nil {
		t.Fatalf("seed broken event index path: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode := app.Run([]string{"execute", "start"})
	if exitCode != 1 {
		t.Fatalf("expected execute start failure when timeline append fails, got %d", exitCode)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-test-plan")
	if err != nil {
		t.Fatalf("load state after rollback: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "" {
		t.Fatalf("expected execute start rollback to restore pre-start state, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan after rollback: %v", err)
	}
	if current != nil {
		t.Fatalf("expected execute start rollback to restore nil current-plan pointer, got %#v", current)
	}
}

func TestEvidenceSubmitCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"status":"success","provider":"github-actions"}`)
	exitCode := app.Run([]string{"evidence", "submit", "--kind", "ci"})
	if exitCode != 0 {
		t.Fatalf("evidence submit command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON evidence submit output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "evidence submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus evidence event, got %#v", timelineResult)
	}
	if timelineResult.Events[0].Command != "plan" || timelineResult.Events[1].Command != "evidence submit" {
		t.Fatalf("unexpected evidence timeline events: %#v", timelineResult.Events)
	}
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestEvidenceSubmitIgnoresWatchlistWriteFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.LookupEnv = func(string) (string, bool) { return "", false }
	app.UserHomeDir = func() (string, error) { return "", errors.New("watchlist-home-boom") }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"status":"success","provider":"github-actions"}`)
	exitCode := app.Run([]string{"evidence", "submit", "--kind", "ci"})
	if exitCode != 0 {
		t.Fatalf("expected evidence submit to succeed despite watchlist failure, got %d: %s", exitCode, stderr.String())
	}
}

func TestEvidenceSubmitCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"status":"success","unexpected":true}`)
	exitCode := app.Run([]string{"evidence", "submit", "--kind", "ci"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON evidence submit output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "evidence submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "input.unexpected")
}

func TestReviewStartCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Review Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")

	stdout.Reset()
	stderr.Reset()

	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	exitCode := app.Run([]string{"review", "start"})
	if exitCode != 0 {
		t.Fatalf("review start command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review start output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "review start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 3 {
		t.Fatalf("expected plan, execute-start, and review-start events, got %#v", timelineResult)
	}
	last := timelineResult.Events[len(timelineResult.Events)-1]
	if last.Command != "review start" {
		t.Fatalf("unexpected review-start timeline event: %#v", last)
	}
}

func TestReviewCommandsTouchWatchlist(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	anchor := initGitRepoWithCommit(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Touch Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}
	if err := os.Remove(watchlistPathForHome(home)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove watchlist after execute start: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"` + anchor + `","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistContainsWorkspace(t, home, root)
	if err := os.Remove(watchlistPathForHome(home)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove watchlist after review start: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good.","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistContainsWorkspace(t, home, root)
	if err := os.Remove(watchlistPathForHome(home)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove watchlist after review submit: %v", err)
	}

	if exitCode := app.Run([]string{"review", "aggregate", "--round", "review-001-delta"}); exitCode != 0 {
		t.Fatalf("review aggregate failed with %d: %s", exitCode, stderr.String())
	}
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestReviewAggregateIgnoresWatchlistWriteFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	anchor := initGitRepoWithCommit(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Failure Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"` + anchor + `","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good.","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}

	app.LookupEnv = func(string) (string, bool) { return "", false }
	app.UserHomeDir = func() (string, error) { return "", errors.New("watchlist-home-boom") }
	if exitCode := app.Run([]string{"review", "aggregate", "--round", "review-001-delta"}); exitCode != 0 {
		t.Fatalf("expected review aggregate to succeed despite watchlist failure, got %d: %s", exitCode, stderr.String())
	}
}

func TestReviewStartCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Review Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{
		"kind":"delta","anchor_sha":"anchor-sha",
		"dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}],
		"unexpected":true
	}`)
	exitCode := app.Run([]string{"review", "start"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review start output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "review start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "spec.unexpected")
}

func TestReviewSubmitCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}

	assertLastTimelineEventCommand(t, root, "review submit")
}

func TestReviewSubmitCommandDoesNotFailWhenStateMutationLockIsHeld(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Lock Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-18-test-plan")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 0 {
		t.Fatalf("expected review submit success while state lock is held, got %d: %s", exitCode, stderr.String())
	}
}

func TestReviewSubmitCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"findings":[]}`)
	exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review submit output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "review submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "submission.summary")
}

func TestReviewAggregateCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Aggregate Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := app.Run([]string{"review", "aggregate", "--round", "review-001-delta"}); exitCode != 0 {
		t.Fatalf("review aggregate failed with %d: %s", exitCode, stderr.String())
	}

	assertLastTimelineEventCommand(t, root, "review aggregate")
}

func TestReviewSubmitRollsBackWhenTimelineAppendFails(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Rollback Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	approvePlanInFile(t, outputPath, "2026-03-18T14:55:00+08:00")
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","anchor_sha":"anchor-sha","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	eventIndexPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/events.jsonl")
	if err := os.Remove(eventIndexPath); err != nil {
		t.Fatalf("remove seeded event index: %v", err)
	}
	if err := os.MkdirAll(eventIndexPath, 0o755); err != nil {
		t.Fatalf("replace event index with directory: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness", "--by", "reviewer-correctness"}); exitCode != 1 {
		t.Fatalf("expected review submit failure when timeline append fails, got %d", exitCode)
	}

	submissionPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/reviews/review-001-delta/submissions/correctness/submission.json")
	data, err := os.ReadFile(submissionPath)
	if err != nil {
		t.Fatalf("read restored submission skeleton: %v", err)
	}
	var submission struct {
		RoundID     string          `json:"round_id"`
		Slot        string          `json:"slot"`
		Dimension   string          `json:"dimension"`
		SubmittedAt string          `json:"submitted_at"`
		Summary     string          `json:"summary"`
		Findings    []any           `json:"findings"`
		Worklog     json.RawMessage `json:"worklog"`
	}
	if err := json.Unmarshal(data, &submission); err != nil {
		t.Fatalf("unmarshal restored submission skeleton: %v", err)
	}
	if submission.RoundID != "review-001-delta" || submission.Slot != "correctness" || submission.Dimension != "correctness" {
		t.Fatalf("expected submission skeleton identity to be restored, got %#v", submission)
	}
	if submission.SubmittedAt != "" || submission.Summary != "" || len(submission.Findings) != 0 || len(submission.Worklog) == 0 {
		t.Fatalf("expected rollback to restore the starter skeleton, got %#v", submission)
	}
	ledgerPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/reviews/review-001-delta/ledger.json")
	var ledger struct {
		Slots []struct {
			Slot   string `json:"slot"`
			Status string `json:"status"`
		} `json:"slots"`
	}
	ledgerBytes, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger after rollback: %v", err)
	}
	if err := json.Unmarshal(ledgerBytes, &ledger); err != nil {
		t.Fatalf("unmarshal ledger after rollback: %v", err)
	}
	if len(ledger.Slots) != 1 || ledger.Slots[0].Status != "pending" {
		t.Fatalf("expected pending ledger after rollback, got %#v", ledger.Slots)
	}
}

func TestArchiveCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 16, 0, 0, 0, time.UTC)
	}

	relPlanPath := "docs/plans/active/2026-03-18-archive-ready.md"
	writeArchiveReadyPlanForCLI(t, root, relPlanPath)
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-18-archive-ready", &runstate.State{
		ExecutionStartedAt: "2026-03-18T15:00:00Z",
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
	seedPassingFinalizeReviewForCLI(t, root, "2026-03-18-archive-ready", relPlanPath, "review-001-full")

	if exitCode := app.Run([]string{"archive"}); exitCode != 0 {
		t.Fatalf("archive failed with %d: %s", exitCode, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON archive output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "archive" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "execution/finalize/publish", 1)

	assertLastTimelineEventCommand(t, root, "archive")
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestLandCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"})
	if exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "land", 1)
	facts := payload["facts"].(map[string]any)
	if facts["land_pr_url"] != "https://github.com/catu-ai/easyharness/pull/99" {
		t.Fatalf("expected land_pr_url in facts, got %#v", facts)
	}

	assertLastTimelineEventCommand(t, root, "land")
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestReopenNewStepCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "new-step"})
	if exitCode != 0 {
		t.Fatalf("reopen command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "execution/finalize/fix", 2)
	facts := payload["facts"].(map[string]any)
	if facts["reopen_mode"] != "new-step" {
		t.Fatalf("expected new-step reopen_mode in facts, got %#v", facts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "new-step" {
		t.Fatalf("expected reopen mode to persist, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-landed-plan.md" {
		t.Fatalf("expected reopened current-plan pointer to move back to active path, got %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected reopen summary: %q", statusResult.Summary)
	}
	if len(statusResult.NextAction) == 0 || !strings.Contains(statusResult.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected new-step guidance after reopen, got %#v", statusResult.NextAction)
	}
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestReopenFinalizeFixCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 15, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 0 {
		t.Fatalf("reopen command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "execution/finalize/fix", 2)
	facts := payload["facts"].(map[string]any)
	if facts["reopen_mode"] != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen_mode in facts, got %#v", facts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen mode to persist, got %#v", state)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs follow-up fixes") {
		t.Fatalf("unexpected reopen summary: %q", statusResult.Summary)
	}
	if len(statusResult.NextAction) == 0 || !strings.Contains(statusResult.NextAction[0].Description, "review-023-full") && !strings.Contains(strings.ToLower(statusResult.NextAction[0].Description), "review") {
		t.Fatalf("expected finalize-fix guidance after reopen, got %#v", statusResult.NextAction)
	}

	assertLastTimelineEventCommand(t, root, "reopen")
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestReopenCommandRequiresMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen"})
	if exitCode != 2 {
		t.Fatalf("expected missing-mode exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for usage error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen usage text, got %q", stderr.String())
	}
}

func TestReopenCommandRejectsInvalidMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 30, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "bogus"})
	if exitCode != 1 {
		t.Fatalf("expected invalid-mode exit code 1, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected invalid reopen mode to fail, got %#v", payload)
	}
}

func TestReopenCommandRejectsMalformedModeFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--mode"})
	if exitCode != 2 {
		t.Fatalf("expected malformed-mode exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for parse error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag needs an argument: -mode") {
		t.Fatalf("expected parse error for missing mode value, got %q", stderr.String())
	}
}

func TestReopenHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected reopen help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for help, got %q", stdout.String())
	}
}

func TestReopenCommandRejectsExtraPositionalArgs(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix", "extra"})
	if exitCode != 2 {
		t.Fatalf("expected extra-args exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for usage error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen usage text, got %q", stderr.String())
	}
}

func TestReopenCommandReportsGetwdFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Getwd = func() (string, error) {
		return "", errors.New("boom")
	}

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 1 {
		t.Fatalf("expected getwd failure exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout on getwd failure, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "resolve working directory: boom") {
		t.Fatalf("expected getwd failure in stderr, got %q", stderr.String())
	}
}

func TestReopenCommandRejectsLandCleanupInProgress(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 45, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)
	if exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"}); exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 1 {
		t.Fatalf("expected reopen failure during land cleanup, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected reopen failure during land cleanup, got %#v", payload)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK || statusResult.State.CurrentNode != "land" {
		t.Fatalf("expected land status to remain after failed reopen, got %#v", statusResult)
	}
}

func TestLandCompleteCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	home := t.TempDir()
	seedGitWorkspace(t, root)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return home, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)
	if exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"}); exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "complete"})
	if exitCode != 0 {
		t.Fatalf("land complete command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land complete output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land complete" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertLifecycleEnvelope(t, payload, "idle", 1)

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK || statusResult.State.CurrentNode != "idle" {
		t.Fatalf("expected idle status after land complete, got %#v", statusResult)
	}

	assertLastTimelineEventCommand(t, root, "land complete")
	assertWatchlistContainsWorkspace(t, home, root)
}

func TestLandCommandRejectsActivePlanWithoutWritingLandedMarker(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-active-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Active Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/active/2026-03-18-active-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"})
	if exitCode != 1 {
		t.Fatalf("expected land failure exit code, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected ok=false, got %#v", payload)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-active-plan.md" {
		t.Fatalf("expected active current plan to remain, got %#v", current)
	}
	if current.LastLandedPlanPath != "" || current.LastLandedAt != "" {
		t.Fatalf("expected no landed marker on failed command, got %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected active-plan status after failed land, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode == "idle" {
		t.Fatalf("failed land should not switch status to idle: %#v", statusResult)
	}
}

func seedMergeReadyEvidenceForCLI(t *testing.T, root string) {
	t.Helper()
	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 5, 55, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("seed ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("seed sync evidence: %#v", result)
	}
}

func assertLastTimelineEventCommand(t *testing.T, root, command string) {
	t.Helper()
	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) == 0 {
		t.Fatalf("expected timeline events for %q, got %#v", command, timelineResult)
	}
	last := timelineResult.Events[len(timelineResult.Events)-1]
	if last.Command != command {
		t.Fatalf("expected last timeline event %q, got %#v", command, last)
	}
}

func writeArchiveReadyPlanForCLI(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "CLI Archive Ready Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	content := rendered
	content = replaceCLIAll(content, "- Done: [ ]", "- Done: [x]")
	content = replaceCLIAll(content, "- Status: pending", "- Status: completed")
	content = replaceCLIAll(content, "- [ ]", "- [x]")
	content = replaceCLIAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceCLIAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: Archive-ready CLI fixture uses finalize review artifacts.")
	content = replaceCLI(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the slice before archive.")
	content = replaceCLI(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nFull review passed before archive.")
	content = replaceCLI(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.\n- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.")
	content = replaceCLI(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the slice.")
	content = replaceCLI(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.")
	content = replaceCLI(content, "### Follow-Up Issues\n\nPENDING_UNTIL_ARCHIVE", "### Follow-Up Issues\n\nNONE")
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write archive-ready plan: %v", err)
	}
	return path
}

func seedPassingFinalizeReviewForCLI(t *testing.T, root, planStem, relPlanPath, roundID string) {
	t.Helper()
	reviewDir := filepath.Join(root, ".local/harness/plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	manifest := `{
  "round_id": "` + roundID + `",
  "kind": "full",
  "revision": 1,
  "review_title": "Full branch candidate before archive",
  "plan_path": "` + relPlanPath + `",
  "plan_stem": "` + planStem + `"
}`
	if err := os.WriteFile(filepath.Join(reviewDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	aggregate := `{
  "round_id": "` + roundID + `",
  "kind": "full",
  "revision": 1,
  "review_title": "Full branch candidate before archive",
  "decision": "pass",
  "blocking_findings": [],
  "non_blocking_findings": [],
  "aggregated_at": "2026-03-18T15:30:00Z"
}`
	if err := os.WriteFile(filepath.Join(reviewDir, "aggregate.json"), []byte(aggregate), 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func assertCLIErrorPath(t *testing.T, errors []struct {
	Path string `json:"path"`
}, path string) {
	t.Helper()
	for _, issue := range errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected CLI error path %q, got %#v", path, errors)
}

func assertLifecycleEnvelope(t *testing.T, payload map[string]any, wantNode string, wantRevision float64) {
	t.Helper()

	state, ok := payload["state"].(map[string]any)
	if !ok {
		t.Fatalf("expected lifecycle payload state object, got %#v", payload)
	}
	if state["current_node"] != wantNode {
		t.Fatalf("expected current_node %q, got %#v", wantNode, state)
	}
	if _, ok := state["plan_status"]; ok {
		t.Fatalf("expected lifecycle payload to drop plan_status, got %#v", state)
	}
	if _, ok := state["lifecycle"]; ok {
		t.Fatalf("expected lifecycle payload to drop lifecycle, got %#v", state)
	}
	if _, ok := state["revision"]; ok {
		t.Fatalf("expected lifecycle payload to drop state.revision, got %#v", state)
	}
	facts, ok := payload["facts"].(map[string]any)
	if !ok {
		t.Fatalf("expected lifecycle payload facts object, got %#v", payload)
	}
	if facts["revision"] != wantRevision {
		t.Fatalf("expected revision %v in facts, got %#v", wantRevision, facts)
	}
}

func writeArchivedPlanForCLI(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "CLI Landed Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	content := rendered
	content = bytes.NewBufferString(content).String()
	content = replaceCLI(content, "status: active", "status: archived")
	content = replaceCLI(content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval")
	content = replaceCLIAll(content, "- Done: [ ]", "- Done: [x]")
	content = replaceCLIAll(content, "- Status: pending", "- Status: completed")
	content = replaceCLIAll(content, "- [ ]", "- [x]")
	content = replaceCLIAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceCLIAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: Archived CLI fixture uses explicit review-complete closeout.")
	content = replaceCLI(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the slice.")
	content = replaceCLI(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nFull review passed.")
	content = replaceCLI(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- Archived At: 2026-03-18T02:00:00Z\n- Revision: 1\n- PR: NONE\n- Ready: Ready for merge approval.\n- Merge Handoff: Commit and push before merge approval.")
	content = replaceCLI(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the slice.")
	content = replaceCLI(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.")
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write archived plan: %v", err)
	}
	return path
}

func replaceCLI(content, old, new string) string {
	tuned := bytes.Replace([]byte(content), []byte(old), []byte(new), 1)
	return string(tuned)
}

func replaceCLIAll(content, old, new string) string {
	tuned := bytes.ReplaceAll([]byte(content), []byte(old), []byte(new))
	return string(tuned)
}

func ensurePlanSizeInFile(t *testing.T, path, size string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan file: %v", err)
	}
	content := strings.Replace(string(data), "size: REPLACE_WITH_PLAN_SIZE", "size: "+size, 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
}

func approvePlanInFile(t *testing.T, path, approvedAt string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan file: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "approved_at:") {
			lines[i] = "approved_at: " + approvedAt
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
				t.Fatalf("write approved plan file: %v", err)
			}
			return
		}
		if strings.HasPrefix(line, "created_at:") {
			lines = append(lines[:i+1], append([]string{"approved_at: " + approvedAt}, lines[i+1:]...)...)
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
				t.Fatalf("write approved plan file: %v", err)
			}
			return
		}
	}
	t.Fatalf("created_at frontmatter line not found in %s", path)
}

func watchlistPathForHome(home string) string {
	return filepath.Join(home, ".easyharness", "watchlist.json")
}

func assertWatchlistAbsent(t *testing.T, home string) {
	t.Helper()
	if _, err := os.Stat(watchlistPathForHome(home)); !os.IsNotExist(err) {
		t.Fatalf("expected no watchlist at %s, err=%v", watchlistPathForHome(home), err)
	}
}

func assertWatchlistContainsWorkspace(t *testing.T, home, root string) {
	t.Helper()
	data, err := os.ReadFile(watchlistPathForHome(home))
	if err != nil {
		t.Fatalf("read watchlist file: %v", err)
	}
	var payload struct {
		Workspaces []struct {
			WorkspacePath string `json:"workspace_path"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode watchlist file: %v\n%s", err, data)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve canonical workspace: %v", err)
	}
	if len(payload.Workspaces) != 1 || payload.Workspaces[0].WorkspacePath != canonicalRoot {
		t.Fatalf("unexpected watchlist payload: %#v", payload)
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

func initGitRepoWithCommit(t *testing.T, root string) string {
	t.Helper()
	seedGitWorkspace(t, root)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test repo\n"), 0o644); err != nil {
		t.Fatalf("write git fixture file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "test fixture")
	return runGit(t, root, "rev-parse", "HEAD")
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
