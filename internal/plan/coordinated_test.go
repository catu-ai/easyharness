package plan_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestCoordinatedPackageLoadsFlatOrderedSubplans(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-coordinated-root.md")
	writeFile(t, rootPath, renderCoordinatedRoot(t, "Coordinated Root"))

	apiPath := mustSubplanPath(t, rootPath, "api")
	api := renderSubplan(t, "API", nil)
	api = strings.ReplaceAll(api, "- Done: [ ]", "- Done: [x]")
	api = strings.Replace(api, "- Validation: PENDING", "- Validation: Focused tests passed.", 1)
	api = strings.Replace(api, "- Delivered: PENDING", "- Delivered: API behavior.", 1)
	writeFile(t, apiPath, api)

	uiPath := mustSubplanPath(t, rootPath, "ui")
	writeFile(t, uiPath, renderSubplan(t, "UI", []string{"api"}))

	pkg, err := plan.LoadCoordinatedPackage(rootPath)
	if err != nil {
		t.Fatalf("LoadCoordinatedPackage: %v", err)
	}
	if len(pkg.Subplans) != 2 || pkg.Subplans[0].ID != "api" || pkg.Subplans[1].ID != "ui" {
		t.Fatalf("unexpected sorted subplans: %#v", pkg.Subplans)
	}
	if !pkg.Subplan("api").Completed() {
		t.Fatal("expected completed API subplan")
	}
	if current := pkg.Subplan("ui").CurrentStep(); current == nil || current.Title != "Step 1: Replace with first step title" {
		t.Fatalf("unexpected UI current step: %#v", current)
	}
	if issues := pkg.DependencyIssues(); len(issues) != 0 {
		t.Fatalf("unexpected dependency issues: %#v", issues)
	}
	if got := pkg.Progress(); got.Total != 2 || got.Completed != 1 || got.Runnable != 1 || got.Waiting != 0 {
		t.Fatalf("unexpected progress: %#v", got)
	}
	if issues := pkg.CompletionIssues(); len(issues) != 1 || issues[0].Path != "subplan.ui" {
		t.Fatalf("unexpected completion issues: %#v", issues)
	}
}

func TestCoordinatedRootLoadsWithoutWorkBreakdownAndDefersChildReadiness(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-root-readiness.md")
	content := renderCoordinatedRoot(t, "Root Readiness")
	content = checkAllBoxes(content)
	content = strings.ReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Ready.")
	writeFile(t, rootPath, content)

	doc, err := plan.LoadFile(rootPath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if !doc.UsesCoordinatedProfile() {
		t.Fatal("expected coordinated profile")
	}
	if len(doc.Steps) != 0 {
		t.Fatalf("expected no root steps, got %#v", doc.Steps)
	}
	for _, issue := range doc.ArchiveReadinessIssues() {
		if issue.Path == "section.Work Breakdown" {
			t.Fatalf("coordinated root readiness must not require ordinary root steps: %#v", issue)
		}
	}
}

func TestCoordinatedPackageDependencyValidation(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-invalid-graph.md")
	writeFile(t, rootPath, renderCoordinatedRoot(t, "Invalid Graph"))
	writeFile(t, mustSubplanPath(t, rootPath, "api"), renderSubplan(t, "API", []string{"ui"}))
	writeFile(t, mustSubplanPath(t, rootPath, "ui"), renderSubplan(t, "UI", []string{"api", "missing"}))
	writeFile(t, mustSubplanPath(t, rootPath, "self"), renderSubplan(t, "Self", []string{"self"}))

	pkg, err := plan.LoadCoordinatedPackage(rootPath)
	if err != nil {
		t.Fatalf("LoadCoordinatedPackage: %v", err)
	}
	issues := pkg.DependencyIssues()
	for _, want := range []string{"missing dependency", "must not depend on itself", "dependency cycle"} {
		found := false
		for _, issue := range issues {
			if strings.Contains(issue.Message, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q issue in %#v", want, issues)
		}
	}
	if result := plan.LintFile(rootPath); result.OK {
		t.Fatalf("expected package lint to reject invalid dependency graph, got %#v", result)
	}
}

func TestLintFileAcceptsCoordinatedPackageAndDirectSubplan(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-valid-package.md")
	writeFile(t, rootPath, renderCoordinatedRoot(t, "Valid Package"))
	childPath := mustSubplanPath(t, rootPath, "worker")
	writeFile(t, childPath, renderSubplan(t, "Worker", nil))

	if result := plan.LintFile(rootPath); !result.OK {
		t.Fatalf("expected coordinated root lint success, got %#v", result)
	}
	if result := plan.LintFile(childPath); !result.OK {
		t.Fatalf("expected direct subplan lint success, got %#v", result)
	}
}

func TestDirectSubplanLintValidatesItsCoordinatedRoot(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-invalid-root-package.md")
	rootContent := renderCoordinatedRoot(t, "Invalid Root Package")
	rootContent = strings.Replace(rootContent, "## Review Focus\n\n- Review focus\n\n", "", 1)
	writeFile(t, rootPath, rootContent)
	childPath := mustSubplanPath(t, rootPath, "worker")
	writeFile(t, childPath, renderSubplan(t, "Worker", nil))

	result := plan.LintFile(childPath)
	if result.OK {
		t.Fatalf("expected direct child lint to reject invalid root package, got %#v", result)
	}
	assertHasError(t, result, "sections")
}

func TestLintFileRejectsSubplansForStandardRoot(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-standard-root.md")
	writeFile(t, rootPath, mustRenderTemplate(t, "Standard Root"))
	writeFile(t, mustSubplanPath(t, rootPath, "worker"), renderSubplan(t, "Worker", nil))

	result := plan.LintFile(rootPath)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "subplans")
}

func TestLintFileRejectsIncompleteArchivedSubplan(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/archived/2026-07-28-archived-root.md")
	rootContent := renderCoordinatedRoot(t, "Archived Root")
	rootContent = makeArchiveReady(checkAllBoxes(rootContent))
	writeFile(t, rootPath, rootContent)
	writeFile(t, mustSubplanPath(t, rootPath, "worker"), renderSubplan(t, "Worker", nil))

	result := plan.LintFile(rootPath)
	if result.OK {
		t.Fatalf("expected archived package lint failure, got %#v", result)
	}
	assertHasError(t, result, "subplan.worker.result")
}

func TestSubplanPathHelpersRejectNestedPaths(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-path-root.md")
	childPath := mustSubplanPath(t, rootPath, "worker")
	gotRoot, err := plan.RootPathForSubplanPath(childPath)
	if err != nil {
		t.Fatalf("RootPathForSubplanPath: %v", err)
	}
	if gotRoot != rootPath {
		t.Fatalf("expected root %q, got %q", rootPath, gotRoot)
	}
	nested := filepath.Join(plan.SubplansDirForPlanPath(rootPath), "group", "worker.md")
	if _, err := plan.RootPathForSubplanPath(nested); err == nil {
		t.Fatal("expected nested subplan path to fail")
	}
}

func TestCoordinatedPackageRejectsSymlinkedSubplansDirectory(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-symlink-package.md")
	writeFile(t, rootPath, renderCoordinatedRoot(t, "Symlink Package"))

	external := t.TempDir()
	writeFile(t, filepath.Join(external, "worker.md"), renderSubplan(t, "Worker", nil))
	subplansDir := plan.SubplansDirForPlanPath(rootPath)
	if err := os.MkdirAll(filepath.Dir(subplansDir), 0o755); err != nil {
		t.Fatalf("mkdir supplements: %v", err)
	}
	if err := os.Symlink(external, subplansDir); err != nil {
		t.Fatalf("symlink subplans: %v", err)
	}

	if _, err := plan.LoadCoordinatedPackage(rootPath); err == nil || !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("expected symlinked package rejection, got %v", err)
	}
	if result := plan.LintFile(rootPath); result.OK {
		t.Fatalf("expected lint to reject symlinked package, got %#v", result)
	}
}

func TestCoordinatedPackageRejectsSymlinkedSubplanFile(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "docs/plans/active/2026-07-28-symlink-child.md")
	writeFile(t, rootPath, renderCoordinatedRoot(t, "Symlink Child"))

	external := filepath.Join(t.TempDir(), "worker.md")
	writeFile(t, external, renderSubplan(t, "Worker", nil))
	childPath := mustSubplanPath(t, rootPath, "worker")
	if err := os.MkdirAll(filepath.Dir(childPath), 0o755); err != nil {
		t.Fatalf("mkdir subplans: %v", err)
	}
	if err := os.Symlink(external, childPath); err != nil {
		t.Fatalf("symlink child: %v", err)
	}

	if _, err := plan.LoadCoordinatedPackage(rootPath); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlinked child rejection, got %v", err)
	}
}

func renderCoordinatedRoot(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:           title,
		Timestamp:       time.Date(2026, 7, 28, 14, 0, 0, 0, time.FixedZone("SGT", 8*60*60)),
		SourceType:      "direct_request",
		Size:            "L",
		WorkflowProfile: plan.WorkflowProfileCoordinated,
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	return rendered
}

func renderSubplan(t *testing.T, title string, dependencies []string) string {
	t.Helper()
	rendered, err := plan.RenderSubplanTemplate(plan.SubplanTemplateOptions{
		Title:     title,
		DependsOn: dependencies,
	})
	if err != nil {
		t.Fatalf("RenderSubplanTemplate: %v", err)
	}
	return rendered
}

func mustSubplanPath(t *testing.T, rootPath, id string) string {
	t.Helper()
	path, err := plan.SubplanPathForPlan(rootPath, id)
	if err != nil {
		t.Fatalf("SubplanPathForPlan: %v", err)
	}
	return path
}
