package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	versioninfo "github.com/catu-ai/easyharness/internal/version"
)

func testService(root string) Service {
	return Service{
		Workdir: root,
		Version: versioninfo.Info{Version: "v9.9.9", Mode: "release"},
		LookupEnv: func(key string) (string, bool) {
			return "", false
		},
		UserHomeDir: func() (string, error) {
			return filepath.Join(root, "home"), nil
		},
	}
}

func TestInitCreatesManagedInstructionsAndSkills(t *testing.T) {
	root := t.TempDir()

	result := testService(root).Init(Options{})
	if !result.OK {
		t.Fatalf("expected init success, got %#v", result)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	agentsBody := string(agentsData)
	if !strings.Contains(agentsBody, `<!-- easyharness:begin version="v9.9.9" -->`) {
		t.Fatalf("expected versioned managed marker, got:\n%s", agentsBody)
	}
	if !strings.Contains(agentsBody, ".agents/skills") {
		t.Fatalf("expected default repo skills path in managed block, got:\n%s", agentsBody)
	}

	skillData, err := os.ReadFile(filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md"))
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	skillBody := string(skillData)
	if !strings.Contains(skillBody, "easyharness-managed: \"true\"") {
		t.Fatalf("expected managed metadata in skill frontmatter, got:\n%s", skillBody)
	}
	if !strings.Contains(skillBody, "easyharness-version: v9.9.9") {
		t.Fatalf("expected version metadata in skill frontmatter, got:\n%s", skillBody)
	}
}

func TestInitRefreshesManagedBlockWithoutTouchingUserContent(t *testing.T) {
	root := t.TempDir()
	original := strings.Join([]string{
		"# AGENTS.md",
		"",
		"User-owned intro.",
		"",
		"<!-- easyharness:begin -->",
		"old managed content",
		"<!-- easyharness:end -->",
		"",
		"User-owned footer.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := testService(root).Init(Options{})
	if !result.OK {
		t.Fatalf("expected init success, got %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	rendered := string(data)
	if !strings.Contains(rendered, "User-owned intro.") || !strings.Contains(rendered, "User-owned footer.") {
		t.Fatalf("expected user content to survive, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "old managed content") {
		t.Fatalf("expected managed block refresh, got:\n%s", rendered)
	}
	if strings.Count(rendered, "<!-- easyharness:begin") != 1 || strings.Count(rendered, "<!-- easyharness:end -->") != 1 {
		t.Fatalf("expected exactly one managed block, got:\n%s", rendered)
	}
}

func TestInstallSkillsRejectsNonManagedConflicts(t *testing.T) {
	root := t.TempDir()
	conflictPath := filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
		t.Fatalf("mkdir conflict skill dir: %v", err)
	}
	conflictBody := strings.Join([]string{
		"---",
		"name: harness-discovery",
		"description: Custom user-owned skill.",
		"---",
		"",
		"# Custom",
		"",
	}, "\n")
	if err := os.WriteFile(conflictPath, []byte(conflictBody), 0o644); err != nil {
		t.Fatalf("write conflict skill: %v", err)
	}

	result := testService(root).InstallSkills(Options{})
	if result.OK {
		t.Fatalf("expected managed conflict failure, got %#v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected conflict errors, got %#v", result)
	}
}

func TestUninstallSkillsRemovesManagedPackagesButLeavesCustomOnes(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)
	if result := svc.InstallSkills(Options{}); !result.OK {
		t.Fatalf("install skills failed: %#v", result)
	}

	customPath := filepath.Join(root, ".agents/skills/custom/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatalf("mkdir custom skill dir: %v", err)
	}
	customBody := strings.Join([]string{
		"---",
		"name: custom",
		"description: User-owned custom skill.",
		"---",
		"",
		"# Custom",
		"",
	}, "\n")
	if err := os.WriteFile(customPath, []byte(customBody), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	result := svc.UninstallSkills(Options{})
	if !result.OK {
		t.Fatalf("expected uninstall success, got %#v", result)
	}

	if _, err := os.Stat(filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected managed skill removal, err=%v", err)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("expected custom skill to survive, err=%v", err)
	}
}

func TestUninstallInstructionsDeletesFileWhenOnlyManagedContentRemains(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)
	if result := svc.InstallInstructions(Options{}); !result.OK {
		t.Fatalf("install instructions failed: %#v", result)
	}

	result := svc.UninstallInstructions(Options{})
	if !result.OK {
		t.Fatalf("expected uninstall instructions success, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected AGENTS.md deletion, err=%v", err)
	}
}

func TestInstallInstructionsDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()

	result := testService(root).InstallInstructions(Options{DryRun: true})
	if !result.OK {
		t.Fatalf("expected dry-run success, got %#v", result)
	}
	if result.Mode != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid writing AGENTS.md, err=%v", err)
	}
}
