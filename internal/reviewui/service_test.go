package reviewui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestServiceReadSurfacesOneReviewerDecisionFindingsAndCoverage(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-decision.md", "Review decision")
	saveReviewState(t, workdir, planStem, 2, "")
	integrated := reviewAssignment(workdir, planStem, "review-002-delta", "integrated", "integrated")
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{"finding-1"}}
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest: contracts.ReviewManifest{
			RoundID: "review-002-delta", Kind: "delta", AnchorSHA: "base123", ReviewedHeadSHA: "head456",
			Repair: repair, Revision: 2, ReviewTitle: "Finalize repair", PlanPath: relPlanPath, PlanStem: planStem,
			CreatedAt: "2026-07-13T10:00:00Z", Assignments: []contracts.ReviewAssignment{integrated},
		},
		Ledger: contracts.ReviewLedger{
			RoundID: "review-002-delta", Kind: "delta", UpdatedAt: "2026-07-13T10:05:00Z",
			Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(integrated, "submitted", "2026-07-13T10:03:00Z")},
		},
		Submissions: []contracts.ReviewSubmission{
			{RoundID: "review-002-delta", Slot: "integrated", Role: "integrated", By: "reviewer-integrated", SubmittedAt: "2026-07-13T10:03:00Z", Summary: "The repair is ready."},
		},
		Aggregate: &contracts.ReviewAggregate{
			RoundID: "review-002-delta", Kind: "delta", Revision: 2, ReviewTitle: "Finalize repair", ReviewedHeadSHA: "head456", Repair: repair,
			Decision: "changes_requested", AggregatedAt: "2026-07-13T10:06:00Z",
			BlockingFindings:    []contracts.ReviewAggregateFinding{{FindingID: "finding-2", Slot: "integrated", Role: "integrated", Area: "state", Severity: "important", Title: "State mismatch", Details: "The state can drift."}},
			NonBlockingFindings: []contracts.ReviewAggregateFinding{{FindingID: "finding-3", Slot: "integrated", Role: "integrated", Area: "docs", Severity: "minor", Title: "Wording", Details: "The wording can improve."}},
			ResolvedFindingIDs:  []string{"finding-1"}, UnresolvedFindingIDs: []string{"finding-2"},
			UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{{FindingID: "finding-2", Slot: "integrated", Role: "integrated", Area: "state", Severity: "important", Title: "State mismatch", Details: "The state can drift."}},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK || len(result.Rounds) != 1 {
		t.Fatalf("expected one review decision, got %#v", result)
	}
	got := result.Rounds[0]
	if got.Status != "changes_requested" || got.Decision != "changes_requested" || got.CoverageStatus != "blocked" {
		t.Fatalf("expected blocked decision, got %#v", got)
	}
	if got.Reviewer == nil || got.Reviewer.Name != "reviewer-integrated" || got.Reviewer.Summary != "The repair is ready." {
		t.Fatalf("expected the independent integrated reviewer only, got %#v", got.Reviewer)
	}
	if len(got.BlockingFindings) != 1 || got.BlockingFindings[0].FindingID != "finding-2" || got.BlockingFindings[0].Area != "state" {
		t.Fatalf("expected decision findings without assignment topology, got %#v", got.BlockingFindings)
	}
	if got.RepairsRoundID != "review-001-full" || got.ReviewedHeadSHA != "head456" || len(got.ResolvedFindingIDs) != 1 {
		t.Fatalf("expected repair coverage, got %#v", got)
	}
}

func TestServiceReadKeepsOneReviewerPending(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-pending.md", "Pending review")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	integrated := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated")
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest: contracts.ReviewManifest{RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T11:00:00Z", Assignments: []contracts.ReviewAssignment{integrated}},
		Ledger: contracts.ReviewLedger{RoundID: "review-001-full", Kind: "full", UpdatedAt: "2026-07-13T11:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{
			ledgerAssignment(integrated, "pending", ""),
		}},
	})

	got := Service{Workdir: workdir}.Read().Rounds[0]
	if got.Status != "waiting_for_review" || got.Reviewer == nil || got.Reviewer.Status != "pending" {
		t.Fatalf("expected a singular pending reviewer view, got %#v", got)
	}
}

func TestServiceReadSurfacesAbortedRoundHistory(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-aborted.md", "Aborted review")
	saveReviewState(t, workdir, planStem, 1, "")
	integrated := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated")
	abortedAt := "2026-07-13T11:02:00Z"
	assignment := ledgerAssignment(integrated, "aborted", "")
	assignment.AbortedAt = abortedAt
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest: contracts.ReviewManifest{RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T11:00:00Z", Assignments: []contracts.ReviewAssignment{integrated}},
		Ledger:   contracts.ReviewLedger{RoundID: "review-001-full", Kind: "full", UpdatedAt: abortedAt, Assignments: []contracts.ReviewLedgerAssignment{assignment}},
	})

	got := Service{Workdir: workdir}.Read().Rounds[0]
	if got.Status != "aborted" || got.IsActive || got.Reviewer == nil || got.Reviewer.Status != "aborted" || got.Reviewer.AbortedAt != abortedAt {
		t.Fatalf("expected preserved aborted review history, got %#v", got)
	}
}

func TestServiceReadArchivedPlanVisibilityAndLandHiding(t *testing.T) {
	t.Run("visible while waiting for merge", func(t *testing.T) {
		workdir := t.TempDir()
		relPlanPath, planStem := seedArchivedPlan(t, workdir, "2026-07-13-review-archived.md", "Archived review")
		saveReviewState(t, workdir, planStem, 1, "")
		assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated")
		writeReviewRound(t, workdir, planStem, passingRound(relPlanPath, planStem, "review-001-full", assignment))
		result := Service{Workdir: workdir}.Read()
		if len(result.Rounds) != 1 || result.Rounds[0].Status != "pass" {
			t.Fatalf("expected archived review visibility, got %#v", result)
		}
	})

	t.Run("hidden during post-merge bookkeeping", func(t *testing.T) {
		workdir := t.TempDir()
		_, planStem := seedArchivedPlan(t, workdir, "2026-07-13-review-land.md", "Landed review")
		state := &runstate.State{Revision: 1, Land: &runstate.LandState{LandedAt: "2026-07-13T15:00:00Z"}}
		if _, err := runstate.SaveState(workdir, planStem, state); err != nil {
			t.Fatalf("save state: %v", err)
		}
		result := Service{Workdir: workdir}.Read()
		if len(result.Rounds) != 0 || !strings.Contains(result.Summary, "hidden") {
			t.Fatalf("expected review data hidden during land cleanup, got %#v", result)
		}
	})
}

func TestServiceReadOrdersActiveAndHistoricalRounds(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-ordering.md", "Review ordering")
	saveReviewState(t, workdir, planStem, 3, "review-002-delta")
	for _, roundID := range []string{"review-001-full", "review-002-delta", "review-010-full"} {
		assignment := reviewAssignment(workdir, planStem, roundID, "integrated", "integrated")
		writeReviewRound(t, workdir, planStem, reviewRoundFixture{
			Manifest: contracts.ReviewManifest{RoundID: roundID, Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T16:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
			Ledger:   contracts.ReviewLedger{RoundID: roundID, Kind: "full", UpdatedAt: "2026-07-13T16:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "pending", "")}},
		})
	}
	rounds := Service{Workdir: workdir}.Read().Rounds
	if len(rounds) != 3 || rounds[0].RoundID != "review-002-delta" || rounds[1].RoundID != "review-010-full" || rounds[2].RoundID != "review-001-full" {
		t.Fatalf("expected active-first descending order, got %#v", rounds)
	}
}

func TestServiceReadDegradesDamagedDecisionArtifact(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-damage.md", "Review damage")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated")
	writeReviewRound(t, workdir, planStem, passingRound(relPlanPath, planStem, "review-001-full", assignment))
	decisionPath := filepath.Join(runstate.ReviewRoundDir(workdir, planStem, "review-001-full"), "aggregate.json")
	if err := os.WriteFile(decisionPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("damage decision: %v", err)
	}
	got := Service{Workdir: workdir}.Read().Rounds[0]
	if got.Status != "degraded" || !strings.Contains(strings.Join(got.Warnings, " "), "Decision is not valid JSON") {
		t.Fatalf("expected conservative damaged decision, got %#v", got)
	}
}

func TestServiceReadReturnsEmptyWithoutReviewRounds(t *testing.T) {
	workdir := t.TempDir()
	seedActivePlan(t, workdir, "2026-07-13-review-empty.md", "Empty review")
	result := Service{Workdir: workdir}.Read()
	if !result.OK || len(result.Rounds) != 0 || !strings.Contains(result.Summary, "No review rounds") {
		t.Fatalf("expected an empty readable review resource, got %#v", result)
	}
}

type reviewRoundFixture struct {
	Manifest    contracts.ReviewManifest
	Ledger      contracts.ReviewLedger
	Aggregate   *contracts.ReviewAggregate
	Submissions []contracts.ReviewSubmission
}

func passingRound(relPlanPath, planStem, roundID string, assignment contracts.ReviewAssignment) reviewRoundFixture {
	return reviewRoundFixture{
		Manifest:    contracts.ReviewManifest{RoundID: roundID, Kind: "full", ReviewedHeadSHA: "head123", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T14:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
		Ledger:      contracts.ReviewLedger{RoundID: roundID, Kind: "full", UpdatedAt: "2026-07-13T14:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "submitted", "2026-07-13T14:01:00Z")}},
		Aggregate:   &contracts.ReviewAggregate{RoundID: roundID, Kind: "full", Revision: 1, ReviewedHeadSHA: "head123", Decision: "pass", AggregatedAt: "2026-07-13T14:02:00Z", BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{}},
		Submissions: []contracts.ReviewSubmission{{RoundID: roundID, Slot: "integrated", Role: "integrated", By: "reviewer", SubmittedAt: "2026-07-13T14:01:00Z", Summary: "Passed."}},
	}
}

func reviewAssignment(workdir, planStem, roundID, slot, role string) contracts.ReviewAssignment {
	return contracts.ReviewAssignment{
		Slot: slot, Role: role,
		Instructions: "Review the candidate.", SubmissionPath: roundArtifactPath(workdir, planStem, roundID, filepath.Join("submissions", slot, "submission.json")),
	}
}

func ledgerAssignment(assignment contracts.ReviewAssignment, status, submittedAt string) contracts.ReviewLedgerAssignment {
	return contracts.ReviewLedgerAssignment{Slot: assignment.Slot, Role: assignment.Role, Status: status, SubmittedAt: submittedAt, SubmissionPath: assignment.SubmissionPath}
}

func writeReviewRound(t *testing.T, workdir, planStem string, fixture reviewRoundFixture) {
	t.Helper()
	roundDir := runstate.ReviewRoundDir(workdir, planStem, fixture.Manifest.RoundID)
	fixture.Manifest.LedgerPath = filepath.Join(roundDir, "ledger.json")
	fixture.Manifest.Aggregate = filepath.Join(roundDir, "aggregate.json")
	fixture.Manifest.Submissions = filepath.Join(roundDir, "submissions")
	writeJSONFile(t, filepath.Join(roundDir, "manifest.json"), fixture.Manifest)
	writeJSONFile(t, filepath.Join(roundDir, "ledger.json"), fixture.Ledger)
	if fixture.Aggregate != nil {
		writeJSONFile(t, filepath.Join(roundDir, "aggregate.json"), fixture.Aggregate)
	}
	for _, submission := range fixture.Submissions {
		writeJSONFile(t, filepath.Join(roundDir, "submissions", submission.Slot, "submission.json"), submission)
	}
}

func seedActivePlan(t *testing.T, workdir, filename, title string) (string, string) {
	t.Helper()
	return seedPlan(t, workdir, filepath.Join("docs/plans/active", filename), title)
}

func seedArchivedPlan(t *testing.T, workdir, filename, title string) (string, string) {
	t.Helper()
	return seedPlan(t, workdir, filepath.Join("docs/plans/archived", filename), title)
}

func seedPlan(t *testing.T, workdir, relPlanPath, title string) (string, string) {
	t.Helper()
	planPath := filepath.Join(workdir, relPlanPath)
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	content := `---
title: "` + title + `"
status: active
size: S
---

# ` + title + `
`
	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	return filepath.ToSlash(relPlanPath), strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
}

func saveReviewState(t *testing.T, workdir, planStem string, revision int, activeRoundID string) {
	t.Helper()
	state := &runstate.State{Revision: revision}
	if activeRoundID != "" {
		state.ActiveReviewRound = &runstate.ReviewRound{RoundID: activeRoundID}
	}
	if _, err := runstate.SaveState(workdir, planStem, state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func roundArtifactPath(workdir, planStem, roundID, suffix string) string {
	return filepath.Join(runstate.ReviewRoundDir(workdir, planStem, roundID), suffix)
}
