package support

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompactPlanCompletionHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.md")
	content := `---
template_version: 0.3.0
---

# Fixture

## Work Breakdown

### Step 1: Build

- Done: [ ]
- Outcome: Build the fixture.
- Covers: Criterion 1
- Check: Run the test.

## Validation Strategy

- Run tests.

## Closeout

- Validation: PENDING_UNTIL_ARCHIVE
- Review: PENDING_UNTIL_ARCHIVE
- Delivered: PENDING_UNTIL_ARCHIVE
- Not Delivered: PENDING_UNTIL_ARCHIVE
- Follow-Up Issues: NONE
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	CompleteStep(t, path, 1)
	CompleteCloseout(t, path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "- Done: [x]") {
		t.Fatalf("expected completed compact step, got:\n%s", got)
	}
	if strings.Contains(got, "PENDING_UNTIL_ARCHIVE") {
		t.Fatalf("expected every Closeout placeholder to be replaced, got:\n%s", got)
	}
	for _, field := range []string{"Validation", "Review", "Delivered", "Not Delivered", "Follow-Up Issues"} {
		if !strings.Contains(got, "- "+field+":") {
			t.Fatalf("expected Closeout field %q, got:\n%s", field, got)
		}
	}
}
