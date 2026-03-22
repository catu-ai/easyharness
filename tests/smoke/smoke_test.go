package smoke_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

type statusResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type lintResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Artifacts struct {
		PlanPath string `json:"plan_path"`
	} `json:"artifacts"`
}

func TestHelpShowsTopLevelUsage(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
	support.RequireContains(t, result.CombinedOutput(), "plan template   Render the packaged plan template")
	support.RequireContains(t, result.CombinedOutput(), "plan lint       Validate a tracked plan")
	support.RequireContains(t, result.CombinedOutput(), "execute start   Record the execution-start milestone")
	support.RequireContains(t, result.CombinedOutput(), "evidence submit Record append-only CI, publish, or sync evidence")
	support.RequireContains(t, result.CombinedOutput(), "review start    Create a deterministic review round")
	support.RequireContains(t, result.CombinedOutput(), "review submit   Record one reviewer submission")
	support.RequireContains(t, result.CombinedOutput(), "review aggregate Aggregate reviewer submissions")
	support.RequireContains(t, result.CombinedOutput(), "land            Record merge confirmation for the archived candidate")
	support.RequireContains(t, result.CombinedOutput(), "land complete   Record post-merge cleanup completion")
	support.RequireContains(t, result.CombinedOutput(), "archive         Freeze the current active plan")
	support.RequireContains(t, result.CombinedOutput(), "reopen          Restore the current archived plan")
	support.RequireContains(t, result.CombinedOutput(), "status          Summarize the current plan and local execution state")
}

func TestStatusReportsIdleWorkspace(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[statusResult](t, result)
	if !payload.OK {
		t.Fatalf("expected ok status payload, got %#v", payload)
	}
	if payload.Command != "status" {
		t.Fatalf("expected status command, got %#v", payload)
	}
	if payload.State.CurrentNode != "idle" {
		t.Fatalf("expected idle state, got %#v", payload)
	}
	if payload.Summary != "No current plan is active in this worktree." {
		t.Fatalf("expected idle summary, got %#v", payload)
	}
	if len(payload.NextAction) == 0 {
		t.Fatalf("expected idle status to include next-action guidance, got %#v", payload)
	}
	if payload.NextAction[0].Command != nil {
		t.Fatalf("expected idle next action to be descriptive only, got %#v", payload)
	}
	if payload.NextAction[0].Description != "Start discovery or create a new tracked plan when the next slice is ready." {
		t.Fatalf("expected idle handoff guidance, got %#v", payload)
	}
}

func TestPlanTemplatePrintsToStdoutByDefault(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", "Stdout Plan",
		"--timestamp", "2026-03-22T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
	)
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	support.RequireContains(t, result.Stdout, "# Stdout Plan")
	support.RequireContains(t, result.Stdout, "created_at: 2026-03-22T00:00:00Z")
	support.RequireContains(t, result.Stdout, "source_type: issue")
	support.RequireContains(t, result.Stdout, "source_refs: [\"#6\"]")
}

func TestSupportRunUsesBuiltBinaryInsteadOfPATH(t *testing.T) {
	workspace := support.NewWorkspace(t)
	poisonDir := workspace.Path("tmp/poison-bin")
	if err := os.MkdirAll(poisonDir, 0o755); err != nil {
		t.Fatalf("mkdir poison dir: %v", err)
	}

	name := "harness"
	script := "#!/bin/sh\necho poisoned harness\nexit 97\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		name += ".exe"
		script = "@echo poisoned harness\r\nexit /b 97\r\n"
		mode = 0o644
	}
	poisonPath := filepath.Join(poisonDir, name)
	if err := os.WriteFile(poisonPath, []byte(script), mode); err != nil {
		t.Fatalf("write poison harness: %v", err)
	}

	// Build once before poisoning PATH so the runner can only succeed by using
	// the cached absolute binary path instead of resolving `harness` from PATH.
	support.BuildBinary(t)
	t.Setenv("PATH", poisonDir)

	result := support.Run(t, workspace.Root, "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
	if result.CombinedOutput() == "poisoned harness\n" || result.CombinedOutput() == "poisoned harness\r\n" {
		t.Fatalf("expected support runner to bypass PATH and invoke the built binary, got %q", result.CombinedOutput())
	}
}

func TestPlanTemplateAndLintRoundTrip(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-22-smoke-plan.md"

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", "Smoke Plan",
		"--timestamp", "2026-03-22T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)

	planPath := workspace.Path(planRelPath)
	support.RequireFileExists(t, planPath)
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read rendered plan: %v", err)
	}
	support.RequireContains(t, string(data), "# Smoke Plan")
	support.RequireContains(t, string(data), "created_at: 2026-03-22T00:00:00Z")
	support.RequireContains(t, string(data), "source_type: issue")
	support.RequireContains(t, string(data), "source_refs: [\"#6\"]")

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	payload := support.RequireJSONResult[lintResult](t, lint)
	if !payload.OK {
		t.Fatalf("expected lint success, got %#v", payload)
	}
	if payload.Command != "plan lint" {
		t.Fatalf("expected lint command, got %#v", payload)
	}
	if payload.Artifacts.PlanPath != planRelPath {
		t.Fatalf("expected lint plan path %q, got %#v", planRelPath, payload)
	}
}
