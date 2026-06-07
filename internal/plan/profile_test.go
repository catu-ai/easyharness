package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathKindForPrefersRepoConfigOverAncestorDefaultRootMarker(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "docs", "plans", "active", "nested-repo")
	writeProfileTestFile(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness-runtime
`)

	staleDefaultPath := filepath.Join(root, "docs", "plans", "active", "2026-06-07-stale-default.md")
	writeProfileTestFile(t, staleDefaultPath, "# stale default root\n")
	configuredActivePath := filepath.Join(root, "workflow", "plans", "open", "2026-06-07-configured.md")
	writeProfileTestFile(t, configuredActivePath, "# configured active root\n")

	if got := PathKindFor(staleDefaultPath); got != "" {
		t.Fatalf("expected stale default-root path to remain unclassified under custom config, got %q", got)
	}
	if got := PathKindFor(configuredActivePath); got != "active" {
		t.Fatalf("expected configured active path kind, got %q", got)
	}
}

func writeProfileTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
