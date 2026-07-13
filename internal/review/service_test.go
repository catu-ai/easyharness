package review_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/review"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestStartCreatesSoleIntegratedFinalizeReviewerWithPlanFocus(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}

	result := svc.Start(review.StartOptions{})
	if !result.OK {
		t.Fatalf("start failed: %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.Reviewer == nil {
		t.Fatalf("expected one reviewer handle: %#v", result.Artifacts)
	}
	if result.Artifacts.RoundID != "review-001-full" || result.Artifacts.ReviewedHeadSHA == "" {
		t.Fatalf("unexpected full round: %#v", result.Artifacts)
	}
	reviewer := result.Artifacts.Reviewer
	if !strings.Contains(reviewer.ReviewFocus, "state and coverage") {
		t.Fatalf("expected automatic plan review focus: %#v", reviewer)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated || state.ActiveReviewRound.Step != nil {
		t.Fatalf("unexpected active review state: state=%#v err=%v", state, err)
	}
}

func TestStartRejectsUnfinishedStepInsteadOfCreatingStepReview(t *testing.T) {
	root, _ := writeExecutingPlan(t, false)
	result := (review.Service{Workdir: root}).Start(review.StartOptions{})
	if result.OK || len(result.Errors) == 0 || result.Errors[0].Path != "plan.steps" {
		t.Fatalf("expected finalize-only rejection: %#v", result)
	}
}

func TestSubmitCompletesReviewAndUpdatesCoverageWithoutAggregateAction(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	start := svc.Start(review.StartOptions{})
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	submit := svc.Submit(start.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{
		Summary:  "Candidate satisfies the complete rubric.",
		Findings: []review.Finding{{Area: "docs-consistency", Severity: "minor", Title: "Small wording improvement", Details: "This does not block the candidate."}},
	}))
	if !submit.OK || submit.Review == nil || submit.Review.Decision != "pass" {
		t.Fatalf("submit did not complete the review: %#v", submit)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || !state.ActiveReviewRound.Aggregated || state.ActiveReviewRound.Decision != "pass" {
		t.Fatalf("unexpected completed state: state=%#v err=%v", state, err)
	}
	if state.FinalizeCoverage == nil || state.FinalizeCoverage.RootRoundID != start.Artifacts.RoundID || state.FinalizeCoverage.CoveredHeadSHA != start.Artifacts.ReviewedHeadSHA {
		t.Fatalf("expected full coverage root: %#v", state.FinalizeCoverage)
	}
	if _, err := os.Stat(roundFile(root, stem, start.Artifacts.RoundID, "aggregate.json")); err != nil {
		t.Fatalf("internal decision artifact missing: %v", err)
	}
}

func TestSubmitRejectsMovedHeadAndLeavesRoundPending(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	start := svc.Start(review.StartOptions{})
	appendCommit(t, root, "candidate moved")

	submit := svc.Submit(start.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{Summary: "Stale result."}))
	if submit.OK || len(submit.Errors) == 0 || submit.Errors[0].Path != "review.reviewed_head_sha" {
		t.Fatalf("expected reviewed-head rejection: %#v", submit)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated {
		t.Fatalf("moved-head failure must leave round pending: state=%#v err=%v", state, err)
	}
}

func TestExistingCoverageInfersLinkedDeltaAndResolvesFinding(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	full := svc.Start(review.StartOptions{})
	blocked := svc.Submit(full.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{
		Summary:  "One repair is required.",
		Findings: []review.Finding{{Area: "correctness", Severity: "important", Title: "Repair required", Details: "The candidate misses an invariant."}},
	}))
	if !blocked.OK || blocked.Review == nil || blocked.Review.Decision != "changes_requested" || len(blocked.Review.UnresolvedFindingIDs) != 1 {
		t.Fatalf("expected blocking full decision: %#v", blocked)
	}
	appendCommit(t, root, "repair")

	delta := svc.Start(review.StartOptions{})
	if !delta.OK || !strings.HasSuffix(delta.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected inferred delta: %#v", delta)
	}
	manifest := readManifest(t, root, stem, delta.Artifacts.RoundID)
	if manifest.AnchorSHA != full.Artifacts.ReviewedHeadSHA || manifest.Repair == nil || manifest.Repair.RoundID != full.Artifacts.RoundID || len(manifest.Repair.FindingIDs) != 1 {
		t.Fatalf("unexpected inferred repair link: %#v", manifest)
	}
	passed := svc.Submit(delta.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{
		Summary:     "The bounded repair resolves the invariant.",
		Resolutions: []review.FindingResolution{{FindingID: blocked.Review.UnresolvedFindingIDs[0], Status: "resolved", Details: "The invariant is now enforced."}},
	}))
	if !passed.OK || passed.Review == nil || passed.Review.Decision != "pass" {
		t.Fatalf("expected clean linked delta: %#v", passed)
	}
	state, _, _ := runstate.LoadState(root, stem)
	if state.FinalizeCoverage == nil || state.FinalizeCoverage.RootRoundID != full.Artifacts.RoundID || state.FinalizeCoverage.TipRoundID != delta.Artifacts.RoundID {
		t.Fatalf("expected continuous full-plus-delta coverage: %#v", state.FinalizeCoverage)
	}
}

func TestForceFullResetsExistingCoverage(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	first := svc.Start(review.StartOptions{})
	if result := svc.Submit(first.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{Summary: "Pass."})); !result.OK {
		t.Fatalf("first submit failed: %#v", result)
	}
	appendCommit(t, root, "broad redesign")
	reset := svc.Start(review.StartOptions{ForceFull: true})
	if !reset.OK || !strings.HasSuffix(reset.Artifacts.RoundID, "-full") {
		t.Fatalf("expected explicit full reset: %#v", reset)
	}
	manifest := readManifest(t, root, stem, reset.Artifacts.RoundID)
	if manifest.Repair != nil || manifest.AnchorSHA != "" {
		t.Fatalf("full reset must not link prior coverage: %#v", manifest)
	}
}

func writeExecutingPlan(t *testing.T, done bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	stem := "2026-07-13-review-v3"
	doneMark := " "
	if done {
		doneMark = "x"
	}
	content := `---
template_version: 0.3.0
created_at: "2026-07-13T00:00:00Z"
approved_at: "2026-07-13T00:01:00Z"
source_type: direct_request
source_refs: []
size: S
---

# Review V3

## Goal

Exercise the integrated reviewer.

### Decisions and Constraints

- Final review is mandatory.

## Scope

### In Scope

- Review lifecycle.

### Out of Scope

- UI.

## Acceptance Criteria

- [x] Integrated review completes.

## Review Focus

- Challenge review state and coverage.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Build candidate

- Done: [` + doneMark + `]
- Outcome: Candidate is built.
- Covers: Integrated review completes.
- Check: Focused tests pass.

## Validation Strategy

- Run focused tests.

## Closeout

- Validation: Complete.
- Review: Complete.
- Delivered: Complete.
- Not Delivered: None.
- Follow-Up Issues: NONE
- PR: Test only.
- Ready: Yes.
- Merge Handoff: Test only.
`
	planPath := filepath.Join(root, "docs", "plans", "active", stem+".md")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q")
	git(t, root, "config", "user.name", "Codex Test")
	git(t, root, "config", "user.email", "codex@example.com")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "fixture")
	if _, err := runstate.SaveState(root, stem, &runstate.State{ExecutionStartedAt: "2026-07-13T00:02:00Z", Revision: 1}); err != nil {
		t.Fatal(err)
	}
	return root, stem
}

func appendCommit(t *testing.T, root, text string) {
	t.Helper()
	path := filepath.Join(root, "candidate.txt")
	if err := os.WriteFile(path, []byte(text+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", text)
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func readManifest(t *testing.T, root, stem, roundID string) review.Manifest {
	t.Helper()
	data, err := os.ReadFile(roundFile(root, stem, roundID, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest review.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func roundFile(root, stem, roundID, name string) string {
	return filepath.Join(root, ".local", "harness", "plans", stem, "reviews", roundID, name)
}

func jsonBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 13, 1, 2, 3, 0, time.UTC)
}
