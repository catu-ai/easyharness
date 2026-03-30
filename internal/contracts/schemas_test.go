package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestGeneratedSchemasMatchCheckedInFiles(t *testing.T) {
	root := repoRoot(t)
	tempDir := t.TempDir()
	if err := contracts.GenerateSchemaFiles(tempDir); err != nil {
		t.Fatalf("generate schemas: %v", err)
	}

	got := schemaFileMap(t, tempDir)
	want := schemaFileMap(t, filepath.Join(root, "docs", "schemas"))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("generated schemas differ from checked-in files")
	}
}

func TestRepresentativePayloadsValidateAgainstCheckedInSchemas(t *testing.T) {
	root := repoRoot(t)
	tests := []struct {
		name   string
		schema string
		value  any
	}{
		{
			name:   "status result",
			schema: "commands/status.result.schema.json",
			value: contracts.StatusResult{
				OK:      true,
				Command: "status",
				Summary: "Plan is ready.",
				State:   contracts.StatusState{CurrentNode: "plan"},
				NextAction: []contracts.NextAction{
					{Description: "Run harness execute start."},
				},
			},
		},
		{
			name:   "review spec",
			schema: "inputs/review.spec.schema.json",
			value: contracts.ReviewSpec{
				Kind: "delta",
				Dimensions: []contracts.ReviewDimension{
					{Name: "correctness", Instructions: "Check contract drift."},
				},
			},
		},
		{
			name:   "publish evidence input",
			schema: "inputs/evidence.publish.schema.json",
			value: contracts.EvidencePublishInput{
				Status: "recorded",
				PRURL:  "https://github.com/catu-ai/easyharness/pull/72",
				Branch: "codex/issue-72-contract-schemas",
				Base:   "main",
			},
		},
		{
			name:   "runstate state artifact",
			schema: "artifacts/runstate.state.schema.json",
			value: contracts.RunstateState{
				ExecutionStartedAt: "2026-03-31T10:00:00+08:00",
				CurrentNode:        "execution/step-1/implement",
				PlanPath:           "docs/plans/active/2026-03-30-generated-contract-schemas.md",
				PlanStem:           "2026-03-30-generated-contract-schemas",
				Revision:           1,
				ActiveReviewRound: &contracts.RunstateReviewRound{
					RoundID:    "review-001-delta",
					Kind:       "delta",
					Step:       intPtr(1),
					Revision:   1,
					Aggregated: false,
				},
			},
		},
		{
			name:   "review manifest artifact",
			schema: "artifacts/review.manifest.schema.json",
			value: contracts.ReviewManifest{
				RoundID:     "review-001-delta",
				Kind:        "delta",
				Step:        intPtr(1),
				Revision:    1,
				ReviewTitle: "Step 1",
				PlanPath:    "docs/plans/active/2026-03-30-generated-contract-schemas.md",
				PlanStem:    "2026-03-30-generated-contract-schemas",
				CreatedAt:   "2026-03-31T10:00:00+08:00",
				Dimensions: []contracts.ReviewManifestSlot{
					{
						Name:           "correctness",
						Slot:           "correctness",
						Instructions:   "Check wire shape.",
						SubmissionPath: ".local/harness/plans/x/reviews/review-001-delta/submissions/correctness.json",
					},
				},
				LedgerPath:  ".local/harness/plans/x/reviews/review-001-delta/ledger.json",
				Aggregate:   ".local/harness/plans/x/reviews/review-001-delta/aggregate.json",
				Submissions: ".local/harness/plans/x/reviews/review-001-delta/submissions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := mustLoadResolvedSchema(t, filepath.Join(root, "docs", "schemas", tt.schema))
			if err := resolved.Validate(toJSONValue(t, tt.value)); err != nil {
				t.Fatalf("validate %s: %v", tt.schema, err)
			}
		})
	}
}

func TestGenerateSchemaFilesRemovesStaleFiles(t *testing.T) {
	tempDir := t.TempDir()
	stalePath := filepath.Join(tempDir, "commands", "stale.schema.json")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatalf("mkdir stale dir: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	if err := contracts.GenerateSchemaFiles(tempDir); err != nil {
		t.Fatalf("generate schemas: %v", err)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale schema file to be removed, got err=%v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func schemaFileMap(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" && filepath.Base(path) != "index.md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

func mustLoadResolvedSchema(t *testing.T, path string) *jsonschema.Resolved {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve schema: %v", err)
	}
	return resolved
}

func toJSONValue(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal value: %v", err)
	}
	return decoded
}

func intPtr(v int) *int {
	return &v
}
