package plan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestDetectCurrentPathPrefersSingleActivePlanOverArchivedPointer(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs", "plans", "active", "2026-03-18-new-work.md")
	archivedPath := filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md")
	writeTestFile(t, activePath)
	writeTestFile(t, archivedPath)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "archived", "2026-03-17-old-work.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected active plan %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathRejectsRootStemReusedAcrossActiveAndArchivedRoots(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs/plans/active/2026-03-18-reused.md")
	archivedPath := filepath.Join(root, "docs/plans/archived/2026-03-18-reused.md")
	writeTestFile(t, activePath)
	writeTestFile(t, archivedPath)
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-reused.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-18-reused", &runstate.State{
		ExecutionStartedAt: "2026-03-18T12:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save stale state: %v", err)
	}

	if _, err := DetectCurrentPath(root); err == nil || !strings.Contains(err.Error(), "must remain unique") {
		t.Fatalf("expected root stem collision error, got %v", err)
	}
}

func TestDetectCurrentPathErrorsWhenCurrentPointerWouldMaskMultipleActivePlans(t *testing.T) {
	root := t.TempDir()
	activePathA := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	activePathB := filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md")
	writeTestFile(t, activePathA)
	writeTestFile(t, activePathB)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-03-18-second.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	if _, err := DetectCurrentPath(root); err == nil {
		t.Fatal("expected error when current pointer would mask multiple active plans")
	}
}

func TestDetectCurrentPathErrorsWhenArchivedPointerCannotDisambiguateMultipleActivePlans(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md"))
	writeTestFile(t, filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md"))
	writeTestFile(t, filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md"))

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "archived", "2026-03-17-old-work.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	if _, err := DetectCurrentPath(root); err == nil {
		t.Fatal("expected error when archived pointer cannot disambiguate multiple active plans")
	}
}

func TestDetectCurrentPathDoesNotFallBackToArchivedPlanWithoutPointer(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md"))

	_, err := DetectCurrentPath(root)
	if !errors.Is(err, ErrNoCurrentPlan) {
		t.Fatalf("expected ErrNoCurrentPlan, got %v", err)
	}
}

func TestDetectCurrentPathLockedAllowsSameStem(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	writeTestFile(t, activePath)

	got, err := DetectCurrentPathLocked(root, "2026-03-18-first")
	if err != nil {
		t.Fatalf("DetectCurrentPathLocked: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathLockedRejectsStemChange(t *testing.T) {
	root := t.TempDir()
	activePathA := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	activePathB := filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md")
	writeTestFile(t, activePathA)
	writeTestFile(t, activePathB)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-03-18-second.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	if _, err := DetectCurrentPathLocked(root, "2026-03-18-first"); err == nil {
		t.Fatal("expected DetectCurrentPathLocked to reject stem change")
	}
}

func TestDetectCurrentPathPrefersSingleTrackedPlanOverArchivedLightweightPointer(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs", "plans", "active", "2026-03-18-lightweight.md")
	archivedLightweightPath := filepath.Join(root, ".local", "harness", "plans", "archived", "2026-03-17-lightweight.md")
	writeTestFile(t, activePath)
	writeTestFile(t, archivedLightweightPath)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join(".local", "harness", "plans", "archived", "2026-03-17-lightweight.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected tracked active plan %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathUsesConfiguredActivePlanRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFileContent(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness
`)
	activePath := filepath.Join(root, "workflow", "plans", "open", "2026-06-07-custom-root.md")
	writeTestFile(t, activePath)

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected configured active plan %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathIgnoresDefaultRootPointerWhenCustomRootsAreConfigured(t *testing.T) {
	root := t.TempDir()
	writeTestFileContent(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness
`)
	staleDefaultPath := filepath.Join(root, "docs", "plans", "active", "2026-06-07-stale-default.md")
	writeTestFile(t, staleDefaultPath)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-06-07-stale-default.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	_, err := DetectCurrentPath(root)
	if !errors.Is(err, ErrNoCurrentPlan) {
		t.Fatalf("expected ErrNoCurrentPlan for stale default-root pointer, got %v", err)
	}
}

func TestDetectCurrentPathRejectsAbsoluteCurrentPointerOutsideWorkdir(t *testing.T) {
	root := t.TempDir()
	sibling := t.TempDir()
	externalPlan := filepath.Join(sibling, "docs", "plans", "archived", "2026-06-07-external.md")
	writeTestFile(t, externalPlan)

	if _, err := runstate.SaveCurrentPlan(root, externalPlan); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	_, err := DetectCurrentPath(root)
	if !errors.Is(err, ErrNoCurrentPlan) {
		t.Fatalf("expected ErrNoCurrentPlan for absolute external pointer, got %v", err)
	}
}

func TestDetectCurrentPathRejectsRelativeCurrentPointerOutsideWorkdir(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	externalPlan := filepath.Join(parent, "sibling", "docs", "plans", "archived", "2026-06-07-external.md")
	writeTestFile(t, filepath.Join(root, ".keep"))
	writeTestFile(t, externalPlan)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("..", "sibling", "docs", "plans", "archived", "2026-06-07-external.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	_, err := DetectCurrentPath(root)
	if !errors.Is(err, ErrNoCurrentPlan) {
		t.Fatalf("expected ErrNoCurrentPlan for relative external pointer, got %v", err)
	}
}

func TestDetectCurrentPathKeepsArchivedLightweightPointerWhenNoActivePlanExists(t *testing.T) {
	root := t.TempDir()
	archivedLightweightPath := filepath.Join(root, ".local", "harness", "plans", "archived", "2026-03-18-lightweight.md")
	writeTestFile(t, archivedLightweightPath)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join(".local", "harness", "plans", "archived", "2026-03-18-lightweight.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != archivedLightweightPath {
		t.Fatalf("expected archived lightweight plan %s, got %s", archivedLightweightPath, got)
	}
}

func TestArchivePathsUseConfiguredRoots(t *testing.T) {
	root := t.TempDir()
	writeTestFileContent(t, filepath.Join(root, ".harness", "config.yaml"), `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness
`)
	currentPath := filepath.Join(root, "workflow", "plans", "open", "2026-06-07-custom-root.md")

	if got, want := ArchivedPathFor(root, "2026-06-07-custom-root", currentPath, WorkflowProfileStandard), filepath.Join(root, "workflow", "plans", "done", "2026-06-07-custom-root.md"); got != want {
		t.Fatalf("standard archived path = %q, want %q", got, want)
	}
	if got, want := ArchivedPathFor(root, "2026-06-07-custom-root", currentPath, WorkflowProfileLightweight), filepath.Join(root, "tmp", "harness", "plans", "archived", "2026-06-07-custom-root.md"); got != want {
		t.Fatalf("lightweight archived path = %q, want %q", got, want)
	}
}

func TestSupplementsDirForPlanPath(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tracked active plan",
			path: filepath.Join("docs", "plans", "active", "2026-03-18-first.md"),
			want: filepath.Join("docs", "plans", "active", "supplements", "2026-03-18-first"),
		},
		{
			name: "tracked archived plan",
			path: filepath.Join("docs", "plans", "archived", "2026-03-18-first.md"),
			want: filepath.Join("docs", "plans", "archived", "supplements", "2026-03-18-first"),
		},
		{
			name: "local archived lightweight plan",
			path: filepath.Join(".local", "harness", "plans", "archived", "2026-03-18-first.md"),
			want: filepath.Join(".local", "harness", "plans", "archived", "supplements", "2026-03-18-first"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SupplementsDirForPlanPath(tc.path); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	writeTestFileContent(t, path, "# test\n")
}

func writeTestFileContent(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
