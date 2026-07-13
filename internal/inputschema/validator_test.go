package inputschema_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/inputschema"
)

func TestValidateAcceptsValidReviewSpec(t *testing.T) {
	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"assignments":[{"slot":"integrated","role":"integrated","dimensions":["correctness"],"instructions":"Check correctness."}]
	}`))
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %#v", issues)
	}
}

func TestValidateReportsUnknownFieldPath(t *testing.T) {
	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"assignments":[{"slot":"integrated","role":"integrated","dimensions":["correctness"],"instructions":"Check correctness."}],
		"unexpected":true
	}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "spec.unexpected" {
		t.Fatalf("expected unknown-field path, got %#v", issues)
	}
}

func TestValidateReportsNestedTypePath(t *testing.T) {
	issues := inputschema.Validate("inputs.review.submission", "submission", []byte(`{
		"summary":"Found one issue.",
		"findings":[{"area":"review-contract","severity":1,"title":"Wrong type","details":"Severity must be a string."}]
	}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "submission.findings[0].severity" {
		t.Fatalf("expected nested type path, got %#v", issues)
	}
}

func TestValidateReportsMissingRequiredFieldPath(t *testing.T) {
	issues := inputschema.Validate("inputs.evidence.ci", "input", []byte(`{}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "input.status" {
		t.Fatalf("expected missing required field path, got %#v", issues)
	}
}

func TestValidateDoesNotDependOnRepositoryWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	temp := t.TempDir()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"assignments":[{"slot":"integrated","role":"integrated","dimensions":["correctness"],"instructions":"Check correctness."}]
	}`))
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues away from repo root, got %#v (cwd=%s)", issues, filepath.Clean(temp))
	}
}

func TestValidateSplitsMultipleMissingRequiredFields(t *testing.T) {
	issues := inputschema.Validate("inputs.review.submission", "submission", []byte(`{
		"summary":"Missing fields.",
		"findings":[{"title":"Missing metadata"}]
	}`))
	if len(issues) != 3 {
		t.Fatalf("expected three validation issues, got %#v", issues)
	}
	paths := map[string]bool{}
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	for _, want := range []string{
		"submission.findings[0].area",
		"submission.findings[0].severity",
		"submission.findings[0].details",
	} {
		if !paths[want] {
			t.Fatalf("expected issue path %s, got %#v", want, issues)
		}
	}
}
