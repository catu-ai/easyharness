package reviewdimensions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestListReturnsBuiltinDimensions(t *testing.T) {
	result := Service{Workdir: t.TempDir()}.List()
	if !result.OK {
		t.Fatalf("expected builtin list success, got %#v", result)
	}
	got := map[string][]string{}
	for _, dimension := range result.Dimensions {
		got[dimension.Name] = dimension.Sources
		if strings.TrimSpace(dimension.Description) == "" {
			t.Fatalf("dimension %q has empty description", dimension.Name)
		}
	}
	for _, name := range []string{"agent-ux", "correctness", "docs-consistency", "evidence-validity", "risk-scan", "tests"} {
		if !reflect.DeepEqual(got[name], []string{SourceBuiltin}) {
			t.Fatalf("expected builtin dimension %q, got %#v", name, result.Dimensions)
		}
	}
}

func TestInstructionsReturnsBuiltinMarkdown(t *testing.T) {
	instructions, warnings, errors := Service{Workdir: t.TempDir()}.Instructions("correctness")
	if len(errors) > 0 {
		t.Fatalf("expected builtin instructions, got errors %#v", errors)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if !strings.Contains(instructions, "Review the change for correctness.") {
		t.Fatalf("unexpected instructions:\n%s", instructions)
	}
	if strings.Contains(instructions, `\"`) {
		t.Fatalf("expected raw markdown, got escaped content:\n%s", instructions)
	}
}

func TestEvidenceValidityInstructionsCoverGoalOrientedReview(t *testing.T) {
	instructions, warnings, errors := Service{Workdir: t.TempDir()}.Instructions("evidence-validity")
	if len(errors) > 0 {
		t.Fatalf("expected evidence-validity instructions, got errors %#v", errors)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	for _, want := range []string{
		"Review the change for evidence validity.",
		"scorecard",
		"rejected hypotheses",
		"residual uncertainty",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("expected instructions to contain %q:\n%s", want, instructions)
		}
	}
}

func TestRepoDimensionsOverrideBuiltins(t *testing.T) {
	root := t.TempDir()
	writeDimension(t, root, ".harness/review/dimensions/tests.md", `---
name: tests
description: Use the repo-specific test policy.
---

Repo-specific test instruction.
`)

	result := Service{Workdir: root}.List()
	if !result.OK {
		t.Fatalf("expected list success, got %#v", result)
	}
	for _, dimension := range result.Dimensions {
		if dimension.Name == "tests" {
			if !reflect.DeepEqual(dimension.Sources, []string{SourceRepo}) || dimension.Description != "Use the repo-specific test policy." {
				t.Fatalf("expected repo override for tests, got %#v", dimension)
			}
			instructions, _, errors := Service{Workdir: root}.Instructions("tests")
			if len(errors) > 0 {
				t.Fatalf("expected repo instructions, got %#v", errors)
			}
			if instructions != "Repo-specific test instruction." {
				t.Fatalf("instructions = %q", instructions)
			}
			return
		}
	}
	t.Fatalf("missing tests dimension in %#v", result.Dimensions)
}

func TestConfiguredDimensionsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".harness/config.yaml", `version: 1
paths:
  review:
    dimensions: config/review-dims
`)
	writeDimension(t, root, "config/review-dims/api-contract.md", `---
name: api-contract
description: Use when checking API contracts.
---

Check the public contract.
`)

	result := Service{Workdir: root}.List()
	if !result.OK {
		t.Fatalf("expected list success, got %#v", result)
	}
	for _, dimension := range result.Dimensions {
		if dimension.Name == "api-contract" && reflect.DeepEqual(dimension.Sources, []string{SourceRepo}) {
			return
		}
	}
	t.Fatalf("expected configured repo dimension, got %#v", result.Dimensions)
}

func TestInvalidRepoDimensionFilesFailList(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantMessage string
	}{
		{
			name: "malformed frontmatter",
			content: `---
name: [broken
description: Invalid YAML.
---

Instruction.
`,
			wantMessage: "malformed YAML frontmatter",
		},
		{
			name: "missing name",
			content: `---
description: Missing name.
---

Instruction.
`,
			wantMessage: "field name must use lowercase alphanumeric segments",
		},
		{
			name: "missing description",
			content: `---
name: missing-description
---

Instruction.
`,
			wantMessage: "field description must not be empty",
		},
		{
			name: "unsupported field",
			content: `---
name: api-contract
description: Has extra metadata.
use_when: Never.
---

Instruction.
`,
			wantMessage: `unsupported frontmatter field "use_when"`,
		},
		{
			name: "invalid name",
			content: `---
name: Bad Name
description: Invalid name.
---

Instruction.
`,
			wantMessage: "field name must use lowercase alphanumeric segments",
		},
		{
			name: "empty body",
			content: `---
name: empty-body
description: Empty instruction body.
---
`,
			wantMessage: "instruction body must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeDimension(t, root, ".harness/review/dimensions/bad.md", tt.content)

			result := Service{Workdir: root}.List()
			if result.OK {
				t.Fatalf("expected list failure, got %#v", result)
			}
			if len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, tt.wantMessage) {
				t.Fatalf("unexpected errors: %#v", result.Errors)
			}
		})
	}
}

func TestDuplicateRepoDimensionFailsList(t *testing.T) {
	root := t.TempDir()
	writeDimension(t, root, ".harness/review/dimensions/one.md", `---
name: api-contract
description: First.
---

Instruction one.
`)
	writeDimension(t, root, ".harness/review/dimensions/two.md", `---
name: api-contract
description: Second.
---

Instruction two.
`)

	result := Service{Workdir: root}.List()
	if result.OK {
		t.Fatalf("expected duplicate failure, got %#v", result)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, "duplicates review dimension name") {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestInstructionsRejectsUnknownDimension(t *testing.T) {
	_, _, errors := Service{Workdir: t.TempDir()}.Instructions("missing-dimension")
	if len(errors) != 1 || !strings.Contains(errors[0].Message, "unknown review dimension") {
		t.Fatalf("unexpected errors: %#v", errors)
	}
}

func TestCurrentPlanGuidanceAppendsToResolvedBaseAndAddsPlanLocalGuidance(t *testing.T) {
	root := t.TempDir()
	planPath := "docs/plans/active/2026-07-13-guidance.md"
	writeFile(t, root, planPath, "# plan\n")
	writeDimension(t, root, "docs/plans/active/supplements/2026-07-13-guidance/review-guidance/correctness.md", `---
name: correctness
description: Check the approved state invariant.
---

The candidate must preserve the approved state invariant.
`)
	writeDimension(t, root, "docs/plans/active/supplements/2026-07-13-guidance/review-guidance/review-state.md", `---
name: review-state
description: Use for the plan-specific review-state contract.
---

Check the plan-specific review-state contract.
`)

	service := Service{Workdir: root}
	result := service.List()
	if !result.OK {
		t.Fatalf("expected plan-scoped list success, got %#v", result)
	}
	byName := map[string]struct {
		sources  []string
		planPath string
		path     string
	}{}
	for _, dimension := range result.Dimensions {
		byName[dimension.Name] = struct {
			sources  []string
			planPath string
			path     string
		}{dimension.Sources, dimension.PlanPath, dimension.Path}
	}
	if got := byName["correctness"]; !reflect.DeepEqual(got.sources, []string{SourceBuiltin, SourcePlan}) || got.planPath != planPath || !strings.HasSuffix(got.path, "/correctness.md") {
		t.Fatalf("unexpected appended correctness provenance: %#v", got)
	}
	if got := byName["review-state"]; !reflect.DeepEqual(got.sources, []string{SourcePlan}) || got.planPath != planPath {
		t.Fatalf("unexpected plan-local provenance: %#v", got)
	}
	instructions, _, errs := service.Instructions("correctness")
	if len(errs) > 0 {
		t.Fatalf("expected appended instructions, got %#v", errs)
	}
	baseIndex := strings.Index(instructions, "Review the change for correctness.")
	planIndex := strings.Index(instructions, "The candidate must preserve the approved state invariant.")
	if baseIndex < 0 || planIndex <= baseIndex || !strings.Contains(instructions, "## Plan-scoped guidance") {
		t.Fatalf("expected base instructions followed by plan guidance:\n%s", instructions)
	}
}

func TestPlanGuidanceAppendsAfterRepoOverride(t *testing.T) {
	root := t.TempDir()
	planPath := "docs/plans/active/2026-07-13-repo-plan.md"
	writeFile(t, root, planPath, "# plan\n")
	writeDimension(t, root, ".harness/review/dimensions/tests.md", `---
name: tests
description: Repo tests.
---

Follow the repo test policy.
`)
	writeDimension(t, root, "docs/plans/active/supplements/2026-07-13-repo-plan/review-guidance/tests.md", `---
name: tests
description: Plan tests.
---

Also test the plan invariant.
`)

	dimension, _, errs := (Service{Workdir: root}).ResolveForPlan(planPath, "tests")
	if len(errs) > 0 {
		t.Fatalf("expected exact-plan resolution, got %#v", errs)
	}
	if !reflect.DeepEqual(dimension.Sources, []string{SourceRepo, SourcePlan}) {
		t.Fatalf("sources = %#v", dimension.Sources)
	}
	if strings.Contains(dimension.Instructions, "unit, integration") || !strings.Contains(dimension.Instructions, "Follow the repo test policy.") || !strings.Contains(dimension.Instructions, "Also test the plan invariant.") {
		t.Fatalf("expected plan guidance appended to repo override:\n%s", dimension.Instructions)
	}
}

func TestResolveForPlanUsesExactPlanInsteadOfCurrentPlan(t *testing.T) {
	root := t.TempDir()
	currentPlan := "docs/plans/active/2026-07-13-current.md"
	exactPlan := "docs/plans/archived/2026-07-12-exact.md"
	writeFile(t, root, currentPlan, "# current\n")
	writeFile(t, root, exactPlan, "# exact\n")
	writeDimension(t, root, "docs/plans/active/supplements/2026-07-13-current/review-guidance/risk-scan.md", `---
name: risk-scan
description: Current plan risks.
---

CURRENT PLAN GUIDANCE
`)
	writeDimension(t, root, "docs/plans/archived/supplements/2026-07-12-exact/review-guidance/risk-scan.md", `---
name: risk-scan
description: Exact plan risks.
---

EXACT PLAN GUIDANCE
`)

	dimension, _, errs := (Service{Workdir: root}).ResolveForPlan(exactPlan, "risk-scan")
	if len(errs) > 0 {
		t.Fatalf("expected exact-plan resolution, got %#v", errs)
	}
	if dimension.PlanPath != exactPlan || !strings.Contains(dimension.Instructions, "EXACT PLAN GUIDANCE") || strings.Contains(dimension.Instructions, "CURRENT PLAN GUIDANCE") {
		t.Fatalf("resolved the wrong plan: %#v\n%s", dimension, dimension.Instructions)
	}
}

func TestArchivedCurrentPlanGuidanceRemainsDiscoverable(t *testing.T) {
	root := t.TempDir()
	planPath := "docs/plans/archived/2026-07-13-archived.md"
	writeFile(t, root, planPath, "# archived\n")
	writeDimension(t, root, "docs/plans/archived/supplements/2026-07-13-archived/review-guidance/archive-risk.md", `---
name: archive-risk
description: Archived plan risk.
---

Check the archived plan risk.
`)
	if _, err := runstate.SaveCurrentPlan(root, planPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := (Service{Workdir: root}).List()
	if !result.OK {
		t.Fatalf("expected archived catalog success, got %#v", result)
	}
	for _, dimension := range result.Dimensions {
		if dimension.Name == "archive-risk" && dimension.PlanPath == planPath && reflect.DeepEqual(dimension.Sources, []string{SourcePlan}) {
			return
		}
	}
	t.Fatalf("missing archived plan guidance in %#v", result.Dimensions)
}

func TestInvalidCurrentPlanGuidanceFailsCatalog(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/plans/active/2026-07-13-invalid.md", "# plan\n")
	writeDimension(t, root, "docs/plans/active/supplements/2026-07-13-invalid/review-guidance/bad.md", `---
name: bad
description: Invalid extra field.
mode: override
---

No override is allowed.
`)

	result := (Service{Workdir: root}).List()
	if result.OK || len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, `unsupported frontmatter field "mode"`) {
		t.Fatalf("expected invalid plan guidance error, got %#v", result)
	}
}

func writeDimension(t *testing.T, root, relPath, content string) {
	t.Helper()
	writeFile(t, root, relPath, content)
}

func writeFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
