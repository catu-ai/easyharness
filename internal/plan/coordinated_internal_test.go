package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadCoordinatedPackageRejectsChangingSnapshot(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-changing-package.md")
	writeInternalPlanFile(t, rootPath, renderInternalCoordinatedRoot(t, "Changing Package"))
	firstPath, err := SubplanPathForPlan(rootPath, "first")
	if err != nil {
		t.Fatalf("first subplan path: %v", err)
	}
	writeInternalPlanFile(t, firstPath, renderInternalSubplan(t, "First"))

	originalRead := readSubplanSnapshot
	calls := 0
	readSubplanSnapshot = func(path string) ([]subplanSnapshotFile, error) {
		snapshot, err := originalRead(path)
		calls++
		if calls == 1 && err == nil {
			latePath, pathErr := SubplanPathForPlan(rootPath, "late")
			if pathErr != nil {
				t.Fatalf("late subplan path: %v", pathErr)
			}
			writeInternalPlanFile(t, latePath, renderInternalSubplan(t, "Late"))
		}
		return snapshot, err
	}
	defer func() {
		readSubplanSnapshot = originalRead
	}()

	if _, err := LoadCoordinatedPackage(rootPath); err == nil ||
		!strings.Contains(err.Error(), "changed while it was being read") {
		t.Fatalf("expected unstable package rejection, got %v", err)
	}
}

func renderInternalCoordinatedRoot(t *testing.T, title string) string {
	t.Helper()
	rendered, err := RenderTemplate(TemplateOptions{
		Title:           title,
		Timestamp:       time.Date(2026, 7, 28, 14, 0, 0, 0, time.FixedZone("SGT", 8*60*60)),
		SourceType:      "direct_request",
		Size:            "L",
		WorkflowProfile: WorkflowProfileCoordinated,
	})
	if err != nil {
		t.Fatalf("render coordinated root: %v", err)
	}
	return rendered
}

func renderInternalSubplan(t *testing.T, title string) string {
	t.Helper()
	rendered, err := RenderSubplanTemplate(SubplanTemplateOptions{Title: title})
	if err != nil {
		t.Fatalf("render subplan: %v", err)
	}
	return rendered
}

func writeInternalPlanFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
