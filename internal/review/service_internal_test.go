package review

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestSubmitRollsBackSubmissionAndDecisionWhenStateSaveFails(t *testing.T) {
	root, stem := writeInternalExecutingPlan(t)
	svc := Service{Workdir: root}
	start := svc.Start(StartOptions{})
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	originalSaveState := saveState
	saveState = func(string, string, *runstate.State) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { saveState = originalSaveState })
	result := svc.Submit(start.Artifacts.RoundID, "reviewer-integrated", []byte(`{"summary":"Looks good."}`))
	if result.OK {
		t.Fatalf("expected submit failure: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".local", "harness", "plans", stem, "reviews", start.Artifacts.RoundID, "aggregate.json")); !os.IsNotExist(err) {
		t.Fatalf("decision artifact should roll back, got %v", err)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated {
		t.Fatalf("state should remain pending: state=%#v err=%v", state, err)
	}
}

func writeInternalExecutingPlan(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	stem := "2026-07-13-review-internal"
	path := filepath.Join(root, "docs", "plans", "active", stem+".md")
	content := `---
template_version: 0.3.0
created_at: "2026-07-13T00:00:00Z"
approved_at: "2026-07-13T00:01:00Z"
source_type: direct_request
source_refs: []
size: S
---
# Internal Review
## Goal
Test rollback.
### Decisions and Constraints
- Final review.
## Scope
### In Scope
- Review.
### Out of Scope
- UI.
## Acceptance Criteria
- [x] Review works.
## Review Focus
- State rollback.
## Deferred Items
- None.
## Work Breakdown
### Step 1: Candidate
- Done: [x]
- Outcome: Candidate exists.
- Covers: Review works.
- Check: Test passes.
## Validation Strategy
- Test.
## Closeout
- Validation: Complete.
- Review: Complete.
- Delivered: Complete.
- Not Delivered: None.
- Follow-Up Issues: NONE
- PR: Test.
- Ready: Yes.
- Merge Handoff: Test.
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInternal(t, root, "init", "-q")
	runGitInternal(t, root, "config", "user.name", "Codex Test")
	runGitInternal(t, root, "config", "user.email", "codex@example.com")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInternal(t, root, "add", ".")
	runGitInternal(t, root, "commit", "-qm", "fixture")
	if _, err := runstate.SaveState(root, stem, &runstate.State{ExecutionStartedAt: "2026-07-13T00:02:00Z", Revision: 1}); err != nil {
		t.Fatal(err)
	}
	return root, stem
}

func runGitInternal(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git failed: %v\n%s", err, output)
	}
}
