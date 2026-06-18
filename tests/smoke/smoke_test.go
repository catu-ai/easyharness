package smoke_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/repoconfig"
	"github.com/catu-ai/easyharness/tests/support"
)

type statusResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	Warnings   []string `json:"warnings"`
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

type bootstrapResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Mode      string `json:"mode"`
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
	Scope     string `json:"scope"`
	Agent     string `json:"agent"`
	Actions   []struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	} `json:"actions"`
	Warnings []string `json:"warnings"`
}

func TestHelpShowsTopLevelUsage(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
	support.RequireContains(t, result.CombinedOutput(), "--version       Print JSON build information for the running harness binary")
	support.RequireContains(t, result.CombinedOutput(), "plan template   Render the packaged plan template")
	support.RequireContains(t, result.CombinedOutput(), "plan lint       Validate a tracked plan")
	support.RequireContains(t, result.CombinedOutput(), "plan approve    Record explicit human approval for the current plan")
	support.RequireContains(t, result.CombinedOutput(), "execute start   Record the execution-start milestone")
	support.RequireContains(t, result.CombinedOutput(), "evidence submit Record append-only CI, publish, or sync evidence")
	support.RequireContains(t, result.CombinedOutput(), "review start    Create a deterministic review round")
	support.RequireContains(t, result.CombinedOutput(), "review submit   Record one reviewer submission")
	support.RequireContains(t, result.CombinedOutput(), "review aggregate Aggregate reviewer submissions")
	support.RequireContains(t, result.CombinedOutput(), "review dimensions List and read recommended review dimensions")
	support.RequireContains(t, result.CombinedOutput(), "land            Record merge confirmation and start required post-merge bookkeeping")
	support.RequireContains(t, result.CombinedOutput(), "land complete   Record required post-merge bookkeeping completion")
	support.RequireContains(t, result.CombinedOutput(), "archive         Freeze the current active plan")
	support.RequireContains(t, result.CombinedOutput(), "reopen          Restore the current archived plan")
	support.RequireContains(t, result.CombinedOutput(), "status          Summarize the current plan and local execution state")
	support.RequireContains(t, result.CombinedOutput(), "repo            Manage repo-level easyharness resources")
}

func TestLandHelpShowsRequiredBookkeepingContract(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "land", "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness land --pr <url> [--commit <sha>]")
	support.RequireContains(t, result.CombinedOutput(), "land            Record merge confirmation and enter required post-merge bookkeeping")
	support.RequireContains(t, result.CombinedOutput(), "land complete   Record required post-merge bookkeeping completion and restore idle")
}

func TestLandEntryUsageShowsRequiredBookkeepingContract(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "land")
	support.RequireExitCode(t, result, 2)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness land --pr <url> [--commit <sha>]")
	support.RequireContains(t, result.CombinedOutput(), "Record merge confirmation for the current archived candidate and enter required post-merge bookkeeping.")
}

func TestLandCompleteHelpShowsRequiredBookkeepingContract(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "land", "complete", "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness land complete")
	support.RequireContains(t, result.CombinedOutput(), "Record that required post-merge bookkeeping is complete and restore idle worktree state.")
}

func TestVersionPrintsJSONBuildInfo(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "--version")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	if version := support.RequireVersionField(t, result.Stdout, "version"); version == "" {
		t.Fatalf("expected non-empty release version\noutput:\n%s", result.Stdout)
	}
	if mode := support.RequireVersionField(t, result.Stdout, "mode"); mode != "release" {
		t.Fatalf("expected release mode, got %q\noutput:\n%s", mode, result.Stdout)
	}
	expectedCommit := support.GitHeadCommit(t, support.RepoRoot(t))
	if commit := support.RequireVersionField(t, result.Stdout, "commit"); commit != expectedCommit {
		t.Fatalf("expected release version commit %q, got %q\noutput:\n%s", expectedCommit, commit, result.Stdout)
	}
	if strings.Contains(result.Stdout, `"path"`) {
		t.Fatalf("expected release build version output to omit path, got %q", result.Stdout)
	}
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
	if len(payload.Warnings) != 0 {
		t.Fatalf("expected clean idle workspace to avoid warnings, got %#v", payload.Warnings)
	}
}

func TestStatusIdleReportsNonBlockingBootstrapReminderWhenManagedAssetsAreStale(t *testing.T) {
	workspace := support.NewWorkspace(t)

	initResult := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, initResult)
	support.RequireNoStderr(t, initResult)

	staleManagedInstructionsAtPath(t, workspace.Path("AGENTS.md"))
	staleManagedSkillAtPath(t, workspace.Path(".agents/skills/harness-discovery/SKILL.md"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[statusResult](t, result)
	if payload.State.CurrentNode != "idle" {
		t.Fatalf("expected idle state, got %#v", payload)
	}
	if len(payload.Warnings) == 0 || !strings.Contains(strings.Join(payload.Warnings, "\n"), "non-blocking reminder") {
		t.Fatalf("expected non-blocking bootstrap reminder, got %#v", payload.Warnings)
	}
	if len(payload.NextAction) == 0 || payload.NextAction[0].Command == nil || *payload.NextAction[0].Command != "harness repo init --dry-run" {
		t.Fatalf("expected optional refresh command first, got %#v", payload.NextAction)
	}
}

func TestStatusIdleReportsReminderWhenOnlyManagedInstructionsAreStale(t *testing.T) {
	workspace := support.NewWorkspace(t)

	initResult := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, initResult)
	support.RequireNoStderr(t, initResult)

	staleManagedInstructionsAtPath(t, workspace.Path("AGENTS.md"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[statusResult](t, result)
	if len(payload.Warnings) == 0 || !strings.Contains(strings.Join(payload.Warnings, "\n"), "AGENTS.md managed block") {
		t.Fatalf("expected stale instructions reminder, got %#v", payload.Warnings)
	}
}

func TestStatusIdleReportsReminderWhenOnlyManagedSkillsAreStale(t *testing.T) {
	workspace := support.NewWorkspace(t)

	initResult := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, initResult)
	support.RequireNoStderr(t, initResult)

	staleManagedSkillAtPath(t, workspace.Path(".agents/skills/harness-discovery/SKILL.md"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[statusResult](t, result)
	if len(payload.Warnings) == 0 || !strings.Contains(strings.Join(payload.Warnings, "\n"), "managed skill package") {
		t.Fatalf("expected stale managed-skill reminder, got %#v", payload.Warnings)
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

func staleManagedInstructionsAtPath(t *testing.T, agentsPath string) {
	t.Helper()
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	staleAgents := strings.Replace(string(agentsData), `<!-- easyharness:begin version="`, `<!-- easyharness:begin version="stale-`, 1)
	if err := os.WriteFile(agentsPath, []byte(staleAgents), 0o644); err != nil {
		t.Fatalf("write stale AGENTS.md: %v", err)
	}
}

func staleManagedSkillAtPath(t *testing.T, skillPath string) {
	t.Helper()
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	staleSkill := strings.Replace(string(skillData), "easyharness-version:", "easyharness-version: stale-", 1)
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale managed skill: %v", err)
	}
}

func TestInitBootstrapsFreshRepository(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if !payload.OK || payload.Command != "repo init" {
		t.Fatalf("expected init payload, got %#v", payload)
	}
	if payload.Mode != "apply" || payload.Scope != "repo" || payload.Resource != "bootstrap" {
		t.Fatalf("unexpected init mode/scope/resource: %#v", payload)
	}

	agentsPath := workspace.Path("AGENTS.md")
	support.RequireFileExists(t, agentsPath)
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	support.RequireContains(t, string(agentsData), `<!-- easyharness:begin version="`)
	support.RequireContains(t, string(agentsData), "<!-- easyharness:end -->")

	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-execute/SKILL.md"))
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
	configData, err := os.ReadFile(workspace.Path(".harness/config.yaml"))
	if err != nil {
		t.Fatalf("read repo config: %v", err)
	}
	if string(configData) != repoconfig.DefaultContent {
		t.Fatalf("expected canonical repo config, got:\n%s", configData)
	}
}

func TestInitDryRunDoesNotWriteRepositoryFiles(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "init", "--dry-run")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if payload.Mode != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", payload)
	}
	if len(payload.Actions) == 0 {
		t.Fatalf("expected planned actions, got %#v", payload)
	}

	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents"))
	support.RequireFileMissing(t, workspace.Path(".harness/config.yaml"))
}

func TestInitRepeatRunReportsNoopActions(t *testing.T) {
	workspace := support.NewWorkspace(t)

	first := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, first)
	support.RequireNoStderr(t, first)

	second := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, second)
	support.RequireNoStderr(t, second)

	payload := support.RequireJSONResult[bootstrapResult](t, second)
	if !strings.Contains(payload.Summary, "already up to date") {
		t.Fatalf("expected no-op summary, got %#v", payload)
	}
	for _, action := range payload.Actions {
		if action.Kind != "noop" {
			t.Fatalf("expected noop repeat install actions, got %#v", payload.Actions)
		}
	}
}

func TestRepoInitPreservesExistingInvalidConfigWithWarning(t *testing.T) {
	workspace := support.NewWorkspace(t)
	configPath := workspace.Path(".harness/config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("version: 2\n"), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	result := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if len(payload.Warnings) == 0 || !strings.Contains(strings.Join(payload.Warnings, "\n"), "unsupported version 2") {
		t.Fatalf("expected invalid config warning, got %#v", payload.Warnings)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(configData) != "version: 2\n" {
		t.Fatalf("expected existing config to be preserved, got:\n%s", configData)
	}
}

func TestRepoConfigInitCreatesCanonicalConfig(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "config", "init")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if !payload.OK || payload.Command != "repo config init" || payload.Resource != "config" {
		t.Fatalf("unexpected config init payload: %#v", payload)
	}
	configData, err := os.ReadFile(workspace.Path(".harness/config.yaml"))
	if err != nil {
		t.Fatalf("read repo config: %v", err)
	}
	if string(configData) != repoconfig.DefaultContent {
		t.Fatalf("expected canonical repo config, got:\n%s", configData)
	}
}

func TestRepoConfigRefreshUpdatesOldDefaultConfig(t *testing.T) {
	workspace := support.NewWorkspace(t)
	configPath := workspace.Path(".harness/config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	result := support.Run(t, workspace.Root, "repo", "config", "refresh")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if !payload.OK || payload.Command != "repo config refresh" || payload.Resource != "config" || payload.Operation != "refresh" {
		t.Fatalf("unexpected config refresh payload: %#v", payload)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read repo config: %v", err)
	}
	if string(configData) != repoconfig.DefaultContent {
		t.Fatalf("expected refreshed canonical repo config, got:\n%s", configData)
	}
}

func TestRepoConfigQueriesResolvedValuesViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	workspace.WriteFile(t, ".harness/config.yaml", []byte(`version: 1
paths:
  plans:
    active: workflow/plans/open
  local_runtime: tmp/harness-runtime
`))

	get := support.Run(t, workspace.Root, "repo", "config", "get", "paths.local_runtime")
	support.RequireSuccess(t, get)
	support.RequireNoStderr(t, get)
	if get.Stdout != "tmp/harness-runtime\n" {
		t.Fatalf("unexpected get stdout:\n%s", get.Stdout)
	}

	list := support.Run(t, workspace.Root, "repo", "config", "list")
	support.RequireSuccess(t, list)
	support.RequireNoStderr(t, list)
	wantList := strings.Join([]string{
		"paths.plans.active=workflow/plans/open",
		"paths.plans.archived=docs/plans/archived",
		"paths.local_runtime=tmp/harness-runtime",
		"paths.review.dimensions=.harness/review/dimensions",
		"",
	}, "\n")
	if list.Stdout != wantList {
		t.Fatalf("unexpected list stdout:\n%s", list.Stdout)
	}

	prefixed := support.Run(t, workspace.Root, "repo", "config", "list", "paths.plans")
	support.RequireSuccess(t, prefixed)
	support.RequireNoStderr(t, prefixed)
	wantPrefixed := strings.Join([]string{
		"paths.plans.active=workflow/plans/open",
		"paths.plans.archived=docs/plans/archived",
		"",
	}, "\n")
	if prefixed.Stdout != wantPrefixed {
		t.Fatalf("unexpected prefixed list stdout:\n%s", prefixed.Stdout)
	}

	objectGet := support.Run(t, workspace.Root, "repo", "config", "get", "paths")
	support.RequireExitCode(t, objectGet, 1)
	if objectGet.Stdout != "" {
		t.Fatalf("expected object get to keep stdout empty, got %q", objectGet.Stdout)
	}
	support.RequireContains(t, objectGet.Stderr, "harness repo config list paths")
}

func TestRepoConfigQueryInvalidConfigWarnsOnStderr(t *testing.T) {
	workspace := support.NewWorkspace(t)
	workspace.WriteFile(t, ".harness/config.yaml", []byte("version: 2\n"))

	result := support.Run(t, workspace.Root, "repo", "config", "list", "paths")
	support.RequireSuccess(t, result)
	want := strings.Join([]string{
		"paths.plans.active=docs/plans/active",
		"paths.plans.archived=docs/plans/archived",
		"paths.local_runtime=.local/harness",
		"paths.review.dimensions=.harness/review/dimensions",
		"",
	}, "\n")
	if result.Stdout != want {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	support.RequireContains(t, result.Stderr, "Ignoring")
	support.RequireContains(t, result.Stderr, "using built-in defaults")
}

func TestReviewDimensionsCatalogViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	workspace.WriteFile(t, ".harness/config.yaml", []byte(`version: 1
paths:
  review:
    dimensions: custom/review-dimensions
`))
	workspace.WriteFile(t, "custom/review-dimensions/api-contract.md", []byte(`---
name: api-contract
description: Use when checking public API contracts.
---

# API Contract

Check the public contract.
`))

	list := support.Run(t, workspace.Root, "review", "dimensions", "list")
	support.RequireSuccess(t, list)
	support.RequireNoStderr(t, list)
	var payload struct {
		OK         bool   `json:"ok"`
		Command    string `json:"command"`
		Dimensions []struct {
			Name         string `json:"name"`
			Source       string `json:"source"`
			Description  string `json:"description"`
			Instructions string `json:"instructions"`
		} `json:"dimensions"`
	}
	if err := json.Unmarshal([]byte(list.Stdout), &payload); err != nil {
		t.Fatalf("decode dimensions list: %v\n%s", err, list.Stdout)
	}
	if !payload.OK || payload.Command != "review dimensions list" {
		t.Fatalf("unexpected list payload: %#v", payload)
	}
	seen := map[string]string{}
	for _, dimension := range payload.Dimensions {
		seen[dimension.Name] = dimension.Source
		if dimension.Instructions != "" {
			t.Fatalf("list leaked instruction body: %#v", dimension)
		}
	}
	if seen["correctness"] != "builtin" || seen["api-contract"] != "repo" {
		t.Fatalf("expected builtin and repo dimensions, got %#v", payload.Dimensions)
	}

	instructions := support.Run(t, workspace.Root, "review", "dimensions", "instructions", "api-contract")
	support.RequireSuccess(t, instructions)
	support.RequireNoStderr(t, instructions)
	if instructions.Stdout != "# API Contract\n\nCheck the public contract.\n" {
		t.Fatalf("unexpected raw instructions:\n%s", instructions.Stdout)
	}

	planPath := workspace.Path("docs/plans/active/2026-06-11-review-dimensions-smoke.md")
	template := support.Run(t, workspace.Root, "plan", "template", "--title", "Review Dimensions Smoke", "--output", planPath)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.ApprovePlan(t, planPath, "2026-06-11T00:00:00+08:00")

	startExecution := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, startExecution)
	support.RequireNoStderr(t, startExecution)

	specPath := workspace.WriteJSON(t, "review-spec.json", map[string]any{
		"kind":         "full",
		"review_title": "Catalog-managed review dimension smoke",
		"dimensions": []map[string]string{
			{
				"name":         "api-contract",
				"instructions": "Run `harness review dimensions instructions api-contract` and follow the returned Markdown instruction.",
			},
		},
	})
	reviewStart := support.Run(t, workspace.Root, "review", "start", "--spec", specPath)
	support.RequireSuccess(t, reviewStart)
	support.RequireNoStderr(t, reviewStart)
	var reviewStartPayload struct {
		OK        bool   `json:"ok"`
		Command   string `json:"command"`
		Artifacts struct {
			RoundID string `json:"round_id"`
			Slots   []struct {
				Name           string `json:"name"`
				Slot           string `json:"slot"`
				Instructions   string `json:"instructions"`
				SubmissionPath string `json:"submission_path"`
			} `json:"slots"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal([]byte(reviewStart.Stdout), &reviewStartPayload); err != nil {
		t.Fatalf("decode review start: %v\n%s", err, reviewStart.Stdout)
	}
	if !reviewStartPayload.OK || reviewStartPayload.Command != "review start" || reviewStartPayload.Artifacts.RoundID != "review-001-full" {
		t.Fatalf("unexpected review start payload: %#v", reviewStartPayload)
	}
	if len(reviewStartPayload.Artifacts.Slots) != 1 {
		t.Fatalf("expected one slot, got %#v", reviewStartPayload.Artifacts.Slots)
	}
	slot := reviewStartPayload.Artifacts.Slots[0]
	if slot.Name != "api-contract" || slot.Slot != "api-contract" {
		t.Fatalf("expected catalog dimension slot, got %#v", slot)
	}
	support.RequireFileExists(t, workspace.Path(slot.SubmissionPath))

	submit := support.RunWithOptions(t, support.RunOptions{
		Workdir: workspace.Root,
		Args: []string{
			"review", "submit",
			"--round", reviewStartPayload.Artifacts.RoundID,
			"--slot", slot.Slot,
			"--by", "reviewer-api-contract",
		},
		Stdin: `{"summary":"Catalog-managed dimension submission path works.","findings":[]}`,
	})
	support.RequireSuccess(t, submit)
	support.RequireNoStderr(t, submit)
}

func TestSkillsInstallRejectsInvalidScopeViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "skills", "install", "--scope", "bogus")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if payload.OK {
		t.Fatalf("expected skills install failure payload, got %#v", payload)
	}
	if payload.Command != "repo skills install" || payload.Scope != "bogus" {
		t.Fatalf("unexpected invalid-scope payload: %#v", payload)
	}
}

func TestInstructionsInstallRejectsDuplicateManagedBlocksViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsPath := workspace.Path("AGENTS.md")
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"<!-- easyharness:begin -->",
		"one",
		"<!-- easyharness:end -->",
		"",
		"<!-- easyharness:begin -->",
		"two",
		"<!-- easyharness:end -->",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := support.Run(t, workspace.Root, "repo", "instructions", "install")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if payload.OK {
		t.Fatalf("expected duplicate-block instructions failure, got %#v", payload)
	}
	if payload.Command != "repo instructions install" || payload.Scope != "repo" {
		t.Fatalf("unexpected duplicate-block payload: %#v", payload)
	}
}

func TestSkillsInstallBootstrapsOnlySkills(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "skills", "install")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if !payload.OK || payload.Scope != "repo" || payload.Resource != "skills" {
		t.Fatalf("unexpected skills-scope payload: %#v", payload)
	}
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-discovery/SKILL.md"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
}

func TestSkillsInstallRecoversAfterApplyWriteFailure(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsRootPath := workspace.Path(".agents")
	if err := os.WriteFile(agentsRootPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking .agents file: %v", err)
	}

	failed := support.Run(t, workspace.Root, "repo", "skills", "install")
	support.RequireExitCode(t, failed, 1)
	support.RequireNoStderr(t, failed)

	failedPayload := support.RequireJSONResult[bootstrapResult](t, failed)
	if failedPayload.OK {
		t.Fatalf("expected apply-mode write failure, got %#v", failedPayload)
	}

	if err := os.Remove(agentsRootPath); err != nil {
		t.Fatalf("remove blocking .agents file: %v", err)
	}

	retry := support.Run(t, workspace.Root, "repo", "skills", "install")
	support.RequireSuccess(t, retry)
	support.RequireNoStderr(t, retry)

	retryPayload := support.RequireJSONResult[bootstrapResult](t, retry)
	if !retryPayload.OK || retryPayload.Scope != "repo" {
		t.Fatalf("expected successful retry payload, got %#v", retryPayload)
	}
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
}

func TestInitRecoversAfterMidFlightFailure(t *testing.T) {
	workspace := support.NewWorkspace(t)
	initial := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, initial)
	support.RequireNoStderr(t, initial)

	blockedSkillPath := workspace.Path(".agents/skills/harness-discovery/SKILL.md")
	skillData, err := os.ReadFile(blockedSkillPath)
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	staleSkill := strings.Replace(string(skillData), "easyharness-version:", "easyharness-version: stale-", 1)
	if err := os.WriteFile(blockedSkillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale managed skill file: %v", err)
	}
	if err := os.Chmod(blockedSkillPath, 0o400); err != nil {
		t.Fatalf("chmod blocked skill file: %v", err)
	}

	failed := support.Run(t, workspace.Root, "repo", "init")
	support.RequireExitCode(t, failed, 1)
	support.RequireNoStderr(t, failed)

	failedPayload := support.RequireJSONResult[bootstrapResult](t, failed)
	if failedPayload.OK {
		t.Fatalf("expected init failure, got %#v", failedPayload)
	}
	support.RequireFileExists(t, workspace.Path("AGENTS.md"))

	if err := os.Chmod(blockedSkillPath, 0o644); err != nil {
		t.Fatalf("chmod blocked skill file: %v", err)
	}

	retry := support.Run(t, workspace.Root, "repo", "init")
	support.RequireSuccess(t, retry)
	support.RequireNoStderr(t, retry)

	retryPayload := support.RequireJSONResult[bootstrapResult](t, retry)
	if !retryPayload.OK || retryPayload.Scope != "repo" {
		t.Fatalf("expected successful init retry payload, got %#v", retryPayload)
	}
	support.RequireFileExists(t, workspace.Path("AGENTS.md"))
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
}

func TestInstructionsInstallRefreshesExistingManagedBlockAndThenNoops(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsPath := workspace.Path("AGENTS.md")
	initial := strings.Join([]string{
		"# AGENTS.md",
		"",
		"Repo-owned intro.",
		"",
		"<!-- easyharness:begin -->",
		"outdated managed content",
		"<!-- easyharness:end -->",
		"",
		"## Repo Rules",
		"",
		"- Keep commits reviewable.",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	refresh := support.Run(t, workspace.Root, "repo", "instructions", "install")
	support.RequireSuccess(t, refresh)
	support.RequireNoStderr(t, refresh)

	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read refreshed AGENTS.md: %v", err)
	}
	agentsBody := string(agentsData)
	support.RequireContains(t, agentsBody, "Repo-owned intro.")
	support.RequireContains(t, agentsBody, "## Repo Rules")
	support.RequireContains(t, agentsBody, "## Harness Working Agreement")
	if strings.Contains(agentsBody, "outdated managed content") {
		t.Fatalf("expected refreshed managed block, got:\n%s", agentsBody)
	}

	second := support.Run(t, workspace.Root, "repo", "instructions", "install")
	support.RequireSuccess(t, second)
	support.RequireNoStderr(t, second)

	payload := support.RequireJSONResult[bootstrapResult](t, second)
	if !strings.Contains(payload.Summary, "already up to date") {
		t.Fatalf("expected noop block rerun summary, got %#v", payload)
	}
	if len(payload.Actions) != 1 || payload.Actions[0].Kind != "noop" {
		t.Fatalf("expected noop block rerun action, got %#v", payload.Actions)
	}
}

func TestInitSupportsExplicitOverrideTargetsViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "repo", "init", "--agent", "claude", "--skills-dir", ".claude/skills", "--instructions-file", "CLAUDE.md")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if !payload.OK || payload.Command != "repo init" || payload.Resource != "bootstrap" {
		t.Fatalf("unexpected init override payload: %#v", payload)
	}
	instructionsPath := workspace.Path("CLAUDE.md")
	skillPath := workspace.Path(".claude/skills/harness-discovery/SKILL.md")
	support.RequireFileExists(t, instructionsPath)
	support.RequireFileExists(t, skillPath)
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))

	instructionsData, err := os.ReadFile(instructionsPath)
	if err != nil {
		t.Fatalf("read custom instructions file: %v", err)
	}
	support.RequireContains(t, string(instructionsData), `<!-- easyharness:begin version="`)

	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read custom skill file: %v", err)
	}
	support.RequireContains(t, string(skillData), "easyharness-version:")

	staleInstructions := strings.Replace(string(instructionsData), `<!-- easyharness:begin version="`, `<!-- easyharness:begin version="stale-`, 1)
	if err := os.WriteFile(instructionsPath, []byte(staleInstructions), 0o644); err != nil {
		t.Fatalf("write stale custom instructions file: %v", err)
	}
	staleSkill := strings.Replace(string(skillData), "easyharness-version:", "easyharness-version: stale-", 1)
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale custom skill file: %v", err)
	}

	refresh := support.Run(t, workspace.Root, "repo", "init", "--agent", "claude", "--skills-dir", ".claude/skills", "--instructions-file", "CLAUDE.md")
	support.RequireSuccess(t, refresh)
	support.RequireNoStderr(t, refresh)

	refreshedInstructions, err := os.ReadFile(instructionsPath)
	if err != nil {
		t.Fatalf("read refreshed custom instructions file: %v", err)
	}
	if strings.Contains(string(refreshedInstructions), `version="stale-`) {
		t.Fatalf("expected custom instructions refresh to replace stale version marker, got:\n%s", refreshedInstructions)
	}

	refreshedSkill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read refreshed custom skill file: %v", err)
	}
	if strings.Contains(string(refreshedSkill), "easyharness-version: stale-") {
		t.Fatalf("expected custom skill refresh to replace stale version marker, got:\n%s", refreshedSkill)
	}
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))
}

func TestSkillsAndInstructionsInstallSupportUserScopeViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	codexHome := workspace.Path("tmp/codex-home")

	skillsResult := support.RunWithOptions(t, support.RunOptions{
		Workdir: workspace.Root,
		Args:    []string{"repo", "skills", "install", "--scope", "user"},
		Env:     []string{"CODEX_HOME=" + codexHome},
	})
	support.RequireSuccess(t, skillsResult)
	support.RequireNoStderr(t, skillsResult)

	skillsPayload := support.RequireJSONResult[bootstrapResult](t, skillsResult)
	if !skillsPayload.OK || skillsPayload.Command != "repo skills install" || skillsPayload.Scope != "user" {
		t.Fatalf("unexpected user-scope skills payload: %#v", skillsPayload)
	}
	support.RequireFileExists(t, filepath.Join(codexHome, "skills/harness-discovery/SKILL.md"))
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))

	instructionsResult := support.RunWithOptions(t, support.RunOptions{
		Workdir: workspace.Root,
		Args:    []string{"repo", "instructions", "install", "--scope", "user"},
		Env:     []string{"CODEX_HOME=" + codexHome},
	})
	support.RequireSuccess(t, instructionsResult)
	support.RequireNoStderr(t, instructionsResult)

	instructionsPayload := support.RequireJSONResult[bootstrapResult](t, instructionsResult)
	if !instructionsPayload.OK || instructionsPayload.Command != "repo instructions install" || instructionsPayload.Scope != "user" {
		t.Fatalf("unexpected user-scope instructions payload: %#v", instructionsPayload)
	}
	support.RequireFileExists(t, filepath.Join(codexHome, "AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))

	skillsUninstall := support.RunWithOptions(t, support.RunOptions{
		Workdir: workspace.Root,
		Args:    []string{"repo", "skills", "uninstall", "--scope", "user"},
		Env:     []string{"CODEX_HOME=" + codexHome},
	})
	support.RequireSuccess(t, skillsUninstall)
	support.RequireNoStderr(t, skillsUninstall)

	instructionsUninstall := support.RunWithOptions(t, support.RunOptions{
		Workdir: workspace.Root,
		Args:    []string{"repo", "instructions", "uninstall", "--scope", "user"},
		Env:     []string{"CODEX_HOME=" + codexHome},
	})
	support.RequireSuccess(t, instructionsUninstall)
	support.RequireNoStderr(t, instructionsUninstall)
	support.RequireFileMissing(t, filepath.Join(codexHome, "skills/harness-discovery/SKILL.md"))
	support.RequireFileMissing(t, filepath.Join(codexHome, "AGENTS.md"))
}

func TestSkillsAndInstructionsInstallSupportExplicitOverrideTargetsViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	skillsDir := ".claude/skills"
	instructionsFile := "CLAUDE.md"
	skillPath := workspace.Path(filepath.Join(skillsDir, "harness-discovery/SKILL.md"))
	instructionsPath := workspace.Path(instructionsFile)

	skillsInstall := support.Run(t, workspace.Root, "repo", "skills", "install", "--agent", "claude", "--dir", skillsDir)
	support.RequireSuccess(t, skillsInstall)
	support.RequireNoStderr(t, skillsInstall)
	support.RequireFileExists(t, skillPath)
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))

	instructionsInstall := support.Run(t, workspace.Root, "repo", "instructions", "install", "--agent", "claude", "--file", instructionsFile, "--dir", skillsDir)
	support.RequireSuccess(t, instructionsInstall)
	support.RequireNoStderr(t, instructionsInstall)
	support.RequireFileExists(t, instructionsPath)

	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read override skill file: %v", err)
	}
	instructionsData, err := os.ReadFile(instructionsPath)
	if err != nil {
		t.Fatalf("read override instructions file: %v", err)
	}
	support.RequireContains(t, string(skillData), "easyharness-version:")
	support.RequireContains(t, string(instructionsData), `<!-- easyharness:begin version="`)

	staleSkill := strings.Replace(string(skillData), "easyharness-version:", "easyharness-version: stale-", 1)
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale override skill file: %v", err)
	}
	staleInstructions := strings.Replace(string(instructionsData), `<!-- easyharness:begin version="`, `<!-- easyharness:begin version="stale-`, 1)
	if err := os.WriteFile(instructionsPath, []byte(staleInstructions), 0o644); err != nil {
		t.Fatalf("write stale override instructions file: %v", err)
	}

	skillsRefresh := support.Run(t, workspace.Root, "repo", "skills", "install", "--agent", "claude", "--dir", skillsDir)
	support.RequireSuccess(t, skillsRefresh)
	support.RequireNoStderr(t, skillsRefresh)

	instructionsRefresh := support.Run(t, workspace.Root, "repo", "instructions", "install", "--agent", "claude", "--file", instructionsFile, "--dir", skillsDir)
	support.RequireSuccess(t, instructionsRefresh)
	support.RequireNoStderr(t, instructionsRefresh)

	refreshedSkill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read refreshed override skill file: %v", err)
	}
	if strings.Contains(string(refreshedSkill), "easyharness-version: stale-") {
		t.Fatalf("expected override skill refresh to replace stale version marker, got:\n%s", refreshedSkill)
	}

	refreshedInstructions, err := os.ReadFile(instructionsPath)
	if err != nil {
		t.Fatalf("read refreshed override instructions file: %v", err)
	}
	if strings.Contains(string(refreshedInstructions), `version="stale-`) {
		t.Fatalf("expected override instructions refresh to replace stale version marker, got:\n%s", refreshedInstructions)
	}

	skillsUninstall := support.Run(t, workspace.Root, "repo", "skills", "uninstall", "--agent", "claude", "--dir", skillsDir)
	support.RequireSuccess(t, skillsUninstall)
	support.RequireNoStderr(t, skillsUninstall)

	instructionsUninstall := support.Run(t, workspace.Root, "repo", "instructions", "uninstall", "--agent", "claude", "--file", instructionsFile)
	support.RequireSuccess(t, instructionsUninstall)
	support.RequireNoStderr(t, instructionsUninstall)
	support.RequireFileMissing(t, skillPath)
	support.RequireFileMissing(t, instructionsPath)
	support.RequireFileMissing(t, workspace.Path(".agents/skills"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
}

func TestInstructionsInstallRejectsOutOfOrderManagedMarkersViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsPath := workspace.Path("AGENTS.md")
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"<!-- easyharness:end -->",
		"",
		"out of order",
		"",
		"<!-- easyharness:begin -->",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write malformed AGENTS.md: %v", err)
	}

	result := support.Run(t, workspace.Root, "repo", "instructions", "install")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[bootstrapResult](t, result)
	if payload.OK {
		t.Fatalf("expected malformed marker failure, got %#v", payload)
	}
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
		"--size", "M",
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
	support.RequireContains(t, string(data), "size: M")

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
