package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

//go:generate go run ../../cmd/schemagen --output-dir ../../docs/schemas

type SchemaSpec struct {
	Path string
	Type reflect.Type
}

func SchemaSpecs() []SchemaSpec {
	specs := []SchemaSpec{
		{Path: "artifacts/evidence.ci.record.schema.json", Type: typeOf[EvidenceCIRecord]()},
		{Path: "artifacts/evidence.publish.record.schema.json", Type: typeOf[EvidencePublishRecord]()},
		{Path: "artifacts/evidence.sync.record.schema.json", Type: typeOf[EvidenceSyncRecord]()},
		{Path: "artifacts/review.aggregate.schema.json", Type: typeOf[ReviewAggregate]()},
		{Path: "artifacts/review.ledger.schema.json", Type: typeOf[ReviewLedger]()},
		{Path: "artifacts/review.manifest.schema.json", Type: typeOf[ReviewManifest]()},
		{Path: "artifacts/review.submission.schema.json", Type: typeOf[ReviewSubmission]()},
		{Path: "artifacts/runstate.current-plan.schema.json", Type: typeOf[RunstateCurrentPlan]()},
		{Path: "artifacts/runstate.state.schema.json", Type: typeOf[RunstateState]()},
		{Path: "commands/evidence.submit.result.schema.json", Type: typeOf[EvidenceResult]()},
		{Path: "commands/lifecycle.result.schema.json", Type: typeOf[LifecycleResult]()},
		{Path: "commands/plan.lint.result.schema.json", Type: typeOf[PlanLintResult]()},
		{Path: "commands/review.aggregate.result.schema.json", Type: typeOf[ReviewAggregateResult]()},
		{Path: "commands/review.start.result.schema.json", Type: typeOf[ReviewStartResult]()},
		{Path: "commands/review.submit.result.schema.json", Type: typeOf[ReviewSubmitResult]()},
		{Path: "commands/status.result.schema.json", Type: typeOf[StatusResult]()},
		{Path: "inputs/evidence.ci.schema.json", Type: typeOf[EvidenceCIInput]()},
		{Path: "inputs/evidence.publish.schema.json", Type: typeOf[EvidencePublishInput]()},
		{Path: "inputs/evidence.sync.schema.json", Type: typeOf[EvidenceSyncInput]()},
		{Path: "inputs/review.spec.schema.json", Type: typeOf[ReviewSpec]()},
		{Path: "inputs/review.submission.schema.json", Type: typeOf[ReviewSubmissionInput]()},
	}
	slices.SortFunc(specs, func(a, b SchemaSpec) int {
		switch {
		case a.Path < b.Path:
			return -1
		case a.Path > b.Path:
			return 1
		default:
			return 0
		}
	})
	return specs
}

func GenerateSchemaFiles(outputDir string) error {
	specs := SchemaSpecs()
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("reset schema output dir: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create schema output dir: %w", err)
	}
	for _, spec := range specs {
		schema, err := jsonschema.ForType(spec.Type, nil)
		if err != nil {
			return fmt.Errorf("generate schema for %s: %w", spec.Path, err)
		}
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal schema for %s: %w", spec.Path, err)
		}
		target := filepath.Join(outputDir, spec.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	return writeSchemaIndex(outputDir, specs)
}

func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func writeSchemaIndex(outputDir string, specs []SchemaSpec) error {
	indexPath := filepath.Join(outputDir, "index.md")
	var builder strings.Builder
	builder.WriteString("# Generated Schemas\n\n")
	builder.WriteString("This file is generated from the Go-owned contract types in `internal/contracts/`.\n")
	builder.WriteString("Do not edit the schema files or this index by hand.\n\n")
	builder.WriteString("Regenerate them with:\n\n")
	builder.WriteString("```bash\nscripts/update-schemas\n```\n\n")

	groups := []struct {
		label  string
		prefix string
	}{
		{label: "Commands", prefix: "commands/"},
		{label: "Inputs", prefix: "inputs/"},
		{label: "Artifacts", prefix: "artifacts/"},
	}
	for _, group := range groups {
		builder.WriteString("## " + group.label + "\n\n")
		for _, spec := range specs {
			if !strings.HasPrefix(spec.Path, group.prefix) {
				continue
			}
			builder.WriteString("- [" + spec.Path + "](./" + spec.Path + ")\n")
		}
		builder.WriteString("\n")
	}

	return os.WriteFile(indexPath, []byte(builder.String()), 0o644)
}
