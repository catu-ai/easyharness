package bootstrapsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/install"
)

func TestSyncRefreshesManagedOutputsFromBootstrapAssets(t *testing.T) {
	root := t.TempDir()

	agents := strings.Join([]string{
		"# AGENTS.md",
		"",
		"Repo-specific intro.",
		"",
		"<!-- easyharness:begin -->",
		"stale managed content",
		"<!-- easyharness:end -->",
		"",
		"Repo-specific footer.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	skillPath := filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	staleSkill := strings.Join([]string{
		"---",
		"name: harness-discovery",
		"description: Run interactive, Socratic pre-implementation discovery for medium/large or ambiguous work in a harness-driven repository by clarifying goals, constraints, tradeoffs, and workflow direction before planning or execution. Use this whenever the next move is unclear, the user needs help choosing an approach, or archived work may need to reopen.",
		"---",
		"",
		"# Stale",
		"",
	}, "\n")
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale skill: %v", err)
	}

	if _, err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	rendered := string(agentsData)
	if !strings.Contains(rendered, "Repo-specific intro.") || !strings.Contains(rendered, "Repo-specific footer.") {
		t.Fatalf("expected repo-specific AGENTS content to survive, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "stale managed content") {
		t.Fatalf("expected managed block refresh, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, `<!-- easyharness:begin version="dev" -->`) {
		t.Fatalf("expected versioned managed block marker, got:\n%s", rendered)
	}

	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if !strings.Contains(string(skillData), "easyharness-managed: \"true\"") {
		t.Fatalf("expected managed skill metadata, got:\n%s", skillData)
	}
}

func TestCheckReportsDriftWhenManagedOutputsAreStale(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# AGENTS.md\n\nstale\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	skillPath := filepath.Join(root, ".agents/skills/harness-reviewer/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	staleSkill := strings.Join([]string{
		"---",
		"name: harness-reviewer",
		"description: Use when acting as a dedicated reviewer subagent for one assigned harness review slot in an existing review round and you need to inspect the change, write structured findings, and submit them through `harness review submit`. This skill is only for reviewer subagents, not for the controller agent.",
		"---",
		"",
		"# Stale",
		"",
	}, "\n")
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale skill: %v", err)
	}

	result, err := Check(root)
	if err == nil {
		t.Fatalf("expected drift error, got success %#v", result)
	}
	driftErr, ok := err.(*DriftError)
	if !ok {
		t.Fatalf("expected DriftError, got %T: %v", err, err)
	}
	if len(driftErr.Actions) == 0 {
		t.Fatalf("expected drift actions, got %#v", driftErr)
	}
}

func TestCheckReportsStaleManagedSkillPackages(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	orphanPath := filepath.Join(root, ".agents/skills/orphan/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o755); err != nil {
		t.Fatalf("mkdir orphan dir: %v", err)
	}
	orphanBody := strings.Join([]string{
		"---",
		"name: orphan",
		"description: stale easyharness-managed skill.",
		"metadata:",
		"  easyharness-managed: \"true\"",
		"  easyharness-version: dev",
		"---",
		"",
		"# Orphan",
		"",
	}, "\n")
	if err := os.WriteFile(orphanPath, []byte(orphanBody), 0o644); err != nil {
		t.Fatalf("write orphan skill: %v", err)
	}

	_, err := Check(root)
	if err == nil {
		t.Fatalf("expected drift error for orphaned file")
	}
	driftErr, ok := err.(*DriftError)
	if !ok {
		t.Fatalf("expected DriftError, got %T: %v", err, err)
	}

	found := false
	for _, action := range driftErr.Actions {
		if action.Path == ".agents/skills/orphan/SKILL.md" && action.Kind == install.ActionDelete {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected orphaned file drift action, got %#v", driftErr.Actions)
	}
}

func TestSyncRemovesStaleManagedSkillPackages(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	orphanPath := filepath.Join(root, ".agents/skills/orphan/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o755); err != nil {
		t.Fatalf("mkdir orphan dir: %v", err)
	}
	orphanBody := strings.Join([]string{
		"---",
		"name: orphan",
		"description: stale easyharness-managed skill.",
		"metadata:",
		"  easyharness-managed: \"true\"",
		"  easyharness-version: dev",
		"---",
		"",
		"# Orphan",
		"",
	}, "\n")
	if err := os.WriteFile(orphanPath, []byte(orphanBody), 0o644); err != nil {
		t.Fatalf("write orphan skill: %v", err)
	}

	result, err := Sync(root)
	if err != nil {
		t.Fatalf("sync with orphan: %v", err)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("expected orphaned file removal, err=%v", err)
	}

	found := false
	for _, action := range result.Actions {
		if action.Path == ".agents/skills/orphan/SKILL.md" && action.Kind == install.ActionDelete {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected delete action in sync result, got %#v", result.Actions)
	}
}
