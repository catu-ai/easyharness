package review_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

func TestStartRejectsUncheckedAcceptanceBeforeCreatingReviewState(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	planPath := filepath.Join(root, "docs", "plans", "active", stem+".md")
	content, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	content = bytes.Replace(content, []byte("- [x] Integrated review completes."), []byte("- [ ] Integrated review completes."), 1)
	if err := os.WriteFile(planPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	result := (review.Service{Workdir: root}).Start(review.StartOptions{})
	if result.OK || len(result.Errors) != 1 || result.Errors[0].Path != "plan.acceptance" {
		t.Fatalf("expected unchecked-acceptance rejection: %#v", result)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil {
		t.Fatal(err)
	}
	if state != nil && state.ActiveReviewRound != nil {
		t.Fatalf("rejected review must not create round state: %#v", state.ActiveReviewRound)
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

func TestGeneratedSubmissionSkeletonIsValidSubmitInput(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	start := svc.Start(review.StartOptions{})
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	manifest := readManifest(t, root, stem, start.Artifacts.RoundID)
	submissionPath := manifest.Assignments[0].SubmissionPath
	data, err := os.ReadFile(submissionPath)
	if err != nil {
		t.Fatal(err)
	}
	var editable map[string]any
	if err := json.Unmarshal(data, &editable); err != nil {
		t.Fatalf("generated submission skeleton is not reviewer input: %v\n%s", err, data)
	}
	if len(editable) != 3 {
		t.Fatalf("generated skeleton must contain only reviewer input fields: %#v", editable)
	}
	editable["summary"] = "The generated skeleton can be edited and submitted directly."
	edited := jsonBytes(t, editable)
	if err := os.WriteFile(submissionPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	submit := svc.Submit(start.Artifacts.RoundID, "reviewer-integrated", edited)
	if !submit.OK || submit.Review == nil || submit.Review.Decision != "pass" {
		t.Fatalf("edited generated skeleton did not submit: %#v", submit)
	}
	storedData, err := os.ReadFile(submissionPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored review.Submission
	if err := json.Unmarshal(storedData, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.RoundID != start.Artifacts.RoundID || stored.Slot != "integrated" || stored.Role != "integrated" || stored.By != "reviewer-integrated" || stored.SubmittedAt == "" {
		t.Fatalf("command-owned metadata was not added after validation: %#v", stored)
	}
}

func TestSubmitRejectsCompletedRoundWithoutChangingArtifacts(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	start := svc.Start(review.StartOptions{})
	first := svc.Submit(start.Artifacts.RoundID, "reviewer-first", jsonBytes(t, review.SubmissionInput{
		Summary: "The candidate passes.",
	}))
	if !first.OK || first.Review == nil || first.Review.Decision != "pass" {
		t.Fatalf("first submit failed: %#v", first)
	}

	manifest := readManifest(t, root, stem, start.Artifacts.RoundID)
	paths := []string{
		manifest.Assignments[0].SubmissionPath,
		manifest.LedgerPath,
		manifest.Aggregate,
		filepath.Join(root, ".local", "harness", "plans", stem, "state.json"),
	}
	before := make([][]byte, len(paths))
	for i, path := range paths {
		var err error
		before[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact before repeated submit: %v", err)
		}
	}

	repeated := svc.Submit(start.Artifacts.RoundID, "reviewer-second", jsonBytes(t, review.SubmissionInput{
		Summary:  "Attempt to replace the completed decision.",
		Findings: []review.Finding{{Area: "correctness", Severity: "important", Title: "Replacement finding", Details: "Must not be recorded."}},
	}))
	if repeated.OK || repeated.Summary != "Review round is already complete." || len(repeated.Errors) != 1 || repeated.Errors[0].Path != "round" || !strings.Contains(repeated.Errors[0].Message, "complete and immutable") {
		t.Fatalf("expected immutable completed-round rejection: %#v", repeated)
	}
	for i, path := range paths {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read artifact after repeated submit: %v", err)
		}
		if !bytes.Equal(after, before[i]) {
			t.Fatalf("repeated submit changed %s", path)
		}
	}
}

func TestConcurrentSubmitAllowsExactlyOneDecision(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	start := svc.Start(review.StartOptions{})
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	inputs := [][]byte{
		jsonBytes(t, review.SubmissionInput{Summary: "Passing concurrent decision."}),
		jsonBytes(t, review.SubmissionInput{
			Summary:  "Blocking concurrent decision.",
			Findings: []review.Finding{{Area: "correctness", Severity: "important", Title: "Concurrent finding", Details: "Only valid if this submit wins."}},
		}),
	}
	results := make([]review.SubmitResult, len(inputs))
	ready := make(chan struct{})
	var workers sync.WaitGroup
	for i := range inputs {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-ready
			results[i] = svc.Submit(start.Artifacts.RoundID, "reviewer-concurrent-"+string(rune('a'+i)), inputs[i])
		}(i)
	}
	close(ready)
	workers.Wait()

	successes := 0
	for _, result := range results {
		if result.OK {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful concurrent decision, got %#v", results)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || !state.ActiveReviewRound.Aggregated {
		t.Fatalf("expected one completed round: state=%#v err=%v", state, err)
	}
	manifest := readManifest(t, root, stem, start.Artifacts.RoundID)
	data, err := os.ReadFile(manifest.Assignments[0].SubmissionPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored review.Submission
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Summary == "Passing concurrent decision." && state.ActiveReviewRound.Decision != "pass" {
		t.Fatalf("stored passing submission and decision disagree: submission=%#v state=%#v", stored, state.ActiveReviewRound)
	}
	if stored.Summary == "Blocking concurrent decision." && state.ActiveReviewRound.Decision != "changes_requested" {
		t.Fatalf("stored blocking submission and decision disagree: submission=%#v state=%#v", stored, state.ActiveReviewRound)
	}
	if stored.Summary != "Passing concurrent decision." && stored.Summary != "Blocking concurrent decision." {
		t.Fatalf("stored submission came from neither concurrent request: %#v", stored)
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

func TestAbortUnfinishedRoundPreservesCoverageAndAllowsReplacement(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	full := svc.Start(review.StartOptions{})
	if result := svc.Submit(full.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{Summary: "Pass."})); !result.OK {
		t.Fatalf("complete full review: %#v", result)
	}
	appendCommit(t, root, "follow-up candidate")
	delta := svc.Start(review.StartOptions{})
	if !delta.OK || !strings.HasSuffix(delta.Artifacts.RoundID, "-delta") {
		t.Fatalf("start delta: %#v", delta)
	}

	aborted := svc.Abort(delta.Artifacts.RoundID)
	if !aborted.OK || aborted.Artifacts == nil || aborted.Artifacts.RoundID != delta.Artifacts.RoundID {
		t.Fatalf("abort unfinished round: %#v", aborted)
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound != nil {
		t.Fatalf("abort must clear only the active pointer: state=%#v err=%v", state, err)
	}
	if state.FinalizeCoverage == nil || state.FinalizeCoverage.TipRoundID != full.Artifacts.RoundID {
		t.Fatalf("abort must preserve prior finalize coverage: %#v", state.FinalizeCoverage)
	}
	manifest := readManifest(t, root, stem, delta.Artifacts.RoundID)
	ledger, err := os.ReadFile(manifest.LedgerPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored review.Ledger
	if err := json.Unmarshal(ledger, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Assignments) != 1 || stored.Assignments[0].Status != "aborted" || stored.Assignments[0].AbortedAt != fixedNow().Format(time.RFC3339) {
		t.Fatalf("abort fact was not preserved in the ledger: %#v", stored)
	}
	if _, err := os.Stat(roundFile(root, stem, delta.Artifacts.RoundID, "manifest.json")); err != nil {
		t.Fatalf("abort must preserve round artifacts: %v", err)
	}

	replacement := svc.Start(review.StartOptions{})
	if !replacement.OK || !strings.HasSuffix(replacement.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected replacement review after abort: %#v", replacement)
	}
}

func TestAbortRejectsNonActiveMismatchedAndCompletedRounds(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow}
	if result := svc.Abort("review-001-full"); result.OK || result.Summary != "No active review round can be aborted." {
		t.Fatalf("expected no-active rejection: %#v", result)
	}
	active := svc.Start(review.StartOptions{})
	if result := svc.Abort("review-999-full"); result.OK || result.Summary != "The requested review round is not active." {
		t.Fatalf("expected mismatched-round rejection: %#v", result)
	}
	stateBefore, _, _ := runstate.LoadState(root, stem)
	if stateBefore == nil || stateBefore.ActiveReviewRound == nil || stateBefore.ActiveReviewRound.RoundID != active.Artifacts.RoundID {
		t.Fatalf("mismatched abort changed active state: %#v", stateBefore)
	}
	if result := svc.Submit(active.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{Summary: "Pass."})); !result.OK {
		t.Fatalf("complete review: %#v", result)
	}
	if result := svc.Abort(active.Artifacts.RoundID); result.OK || result.Summary != "Completed review rounds are immutable." {
		t.Fatalf("expected completed-round rejection: %#v", result)
	}
	stateAfter, _, _ := runstate.LoadState(root, stem)
	if stateAfter == nil || stateAfter.ActiveReviewRound == nil || !stateAfter.ActiveReviewRound.Aggregated || stateAfter.FinalizeCoverage == nil {
		t.Fatalf("completed abort changed review state: %#v", stateAfter)
	}
}

func TestAbortTimelineFailureRollsBackLedgerAndState(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	svc := review.Service{Workdir: root, Now: fixedNow, AfterAbort: func(review.AbortResult) error {
		return errors.New("timeline unavailable")
	}}
	active := svc.Start(review.StartOptions{})
	manifest := readManifest(t, root, stem, active.Artifacts.RoundID)
	ledgerBefore, err := os.ReadFile(manifest.LedgerPath)
	if err != nil {
		t.Fatal(err)
	}

	result := svc.Abort(active.Artifacts.RoundID)
	if result.OK || len(result.Errors) == 0 || result.Errors[0].Path != "timeline" {
		t.Fatalf("expected timeline rollback failure: %#v", result)
	}
	ledgerAfter, err := os.ReadFile(manifest.LedgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ledgerAfter, ledgerBefore) {
		t.Fatal("timeline failure did not restore the review ledger")
	}
	state, _, err := runstate.LoadState(root, stem)
	if err != nil || state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.RoundID != active.Artifacts.RoundID {
		t.Fatalf("timeline failure did not restore active state: state=%#v err=%v", state, err)
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

func TestStartFallsBackToFullAfterRebaseRewritesReviewedAncestry(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	appendCommit(t, root, "candidate under review")
	svc := review.Service{Workdir: root, Now: fixedNow}
	full := svc.Start(review.StartOptions{})
	if !full.OK {
		t.Fatalf("start full review: %#v", full)
	}
	if result := svc.Submit(full.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{Summary: "Pass."})); !result.OK {
		t.Fatalf("submit full review: %#v", result)
	}

	reviewedHead := full.Artifacts.ReviewedHeadSHA
	base := git(t, root, "rev-parse", reviewedHead+"^")
	git(t, root, "branch", "candidate", reviewedHead)
	git(t, root, "checkout", "-qb", "upstream", base)
	if err := os.WriteFile(filepath.Join(root, "upstream.txt"), []byte("upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "upstream.txt")
	git(t, root, "commit", "-qm", "advance upstream")
	git(t, root, "checkout", "-q", "candidate")
	git(t, root, "rebase", "upstream")
	if ancestor := gitIsAncestor(t, root, reviewedHead, "HEAD"); ancestor {
		t.Fatal("test setup expected rebase to rewrite reviewed ancestry")
	}

	replacement := svc.Start(review.StartOptions{})
	if !replacement.OK || !strings.HasSuffix(replacement.Artifacts.RoundID, "-full") {
		t.Fatalf("expected automatic full review after rewritten ancestry: %#v", replacement)
	}
	manifest := readManifest(t, root, stem, replacement.Artifacts.RoundID)
	if manifest.Repair != nil || manifest.AnchorSHA != "" {
		t.Fatalf("rewritten ancestry must establish a fresh full root: %#v", manifest)
	}
}

func TestStartDoesNotSilentlyResetUnresolvedFindingsAfterRewrittenAncestry(t *testing.T) {
	root, stem := writeExecutingPlan(t, true)
	appendCommit(t, root, "candidate under review")
	svc := review.Service{Workdir: root, Now: fixedNow}
	full := svc.Start(review.StartOptions{})
	blocked := svc.Submit(full.Artifacts.RoundID, "reviewer-integrated", jsonBytes(t, review.SubmissionInput{
		Summary:  "One repair is required.",
		Findings: []review.Finding{{Area: "correctness", Severity: "important", Title: "Repair required", Details: "The candidate misses an invariant."}},
	}))
	if !blocked.OK || blocked.Review == nil || len(blocked.Review.UnresolvedFindingIDs) != 1 {
		t.Fatalf("expected unresolved finding: %#v", blocked)
	}

	reviewedHead := full.Artifacts.ReviewedHeadSHA
	base := git(t, root, "rev-parse", reviewedHead+"^")
	git(t, root, "branch", "candidate", reviewedHead)
	git(t, root, "checkout", "-qb", "upstream", base)
	if err := os.WriteFile(filepath.Join(root, "upstream.txt"), []byte("upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "upstream.txt")
	git(t, root, "commit", "-qm", "advance upstream")
	git(t, root, "checkout", "-q", "candidate")
	git(t, root, "rebase", "upstream")

	replacement := svc.Start(review.StartOptions{})
	if replacement.OK || !strings.Contains(replacement.Summary, "cannot automatically reset unresolved") {
		t.Fatalf("expected safe pre-round failure: %#v", replacement)
	}
	roundEntries, err := os.ReadDir(filepath.Join(root, ".local", "harness", "plans", stem, "reviews"))
	if err != nil {
		t.Fatal(err)
	}
	if len(roundEntries) != 1 {
		t.Fatalf("failed inference created review artifacts: %#v", roundEntries)
	}
	explicit := svc.Start(review.StartOptions{ForceFull: true})
	if !explicit.OK || !strings.HasSuffix(explicit.Artifacts.RoundID, "-full") {
		t.Fatalf("expected explicit whole-candidate replacement to remain available: %#v", explicit)
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

func gitIsAncestor(t *testing.T, root, ancestor, descendant string) bool {
	t.Helper()
	command := exec.Command("git", "-C", root, "merge-base", "--is-ancestor", ancestor, descendant)
	err := command.Run()
	if err == nil {
		return true
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false
	}
	t.Fatalf("git merge-base --is-ancestor %s %s: %v", ancestor, descendant, err)
	return false
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
