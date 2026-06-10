package reviewdimensions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListReturnsBuiltinDimensions(t *testing.T) {
	result := Service{Workdir: t.TempDir()}.List()
	if !result.OK {
		t.Fatalf("expected builtin list success, got %#v", result)
	}
	got := map[string]string{}
	for _, dimension := range result.Dimensions {
		got[dimension.Name] = dimension.Source
		if strings.TrimSpace(dimension.Description) == "" {
			t.Fatalf("dimension %q has empty description", dimension.Name)
		}
	}
	for _, name := range []string{"agent-ux", "correctness", "docs-consistency", "risk-scan", "tests"} {
		if got[name] != SourceBuiltin {
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
			if dimension.Source != SourceRepo || dimension.Description != "Use the repo-specific test policy." {
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
		if dimension.Name == "api-contract" && dimension.Source == SourceRepo {
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
			wantMessage: "field name must use lowercase letters",
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
			wantMessage: "field name must use lowercase letters",
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
