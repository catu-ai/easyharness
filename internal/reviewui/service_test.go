package reviewui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestServiceReadSurfacesReviewerAssignments(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-assignments.md", "Review assignments")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")

	integrated := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated", []contracts.ReviewResolvedDimension{
		resolvedDimension("correctness", "builtin", "Check behavioral correctness."),
		resolvedDimension("tests", "builtin", "Check validation coverage."),
	})
	specialist := reviewAssignment(workdir, planStem, "review-001-full", "review-state", "specialist", []contracts.ReviewResolvedDimension{
		resolvedDimension("risk-scan", "builtin", "Challenge review-state failure modes."),
	})
	specialist.RiskBrief = &contracts.ReviewRiskBrief{
		RiskSurfaces: []string{"review state transitions"},
		Invariants:   []string{"a passing aggregate covers the immutable candidate"},
		FailureModes: []string{"candidate head changes after review"},
	}

	round := reviewRoundFixture{
		Manifest: contracts.ReviewManifest{
			RoundID:     "review-001-full",
			Kind:        "full",
			Revision:    1,
			ReviewTitle: "Finalize candidate",
			PlanPath:    relPlanPath,
			PlanStem:    planStem,
			CreatedAt:   "2026-07-13T10:00:00Z",
			Assignments: []contracts.ReviewAssignment{integrated, specialist},
		},
		Ledger: contracts.ReviewLedger{
			RoundID:   "review-001-full",
			Kind:      "full",
			UpdatedAt: "2026-07-13T10:05:00Z",
			Assignments: []contracts.ReviewLedgerAssignment{
				ledgerAssignment(integrated, "submitted", "2026-07-13T10:04:00Z"),
				ledgerAssignment(specialist, "pending", ""),
			},
		},
		Submissions: []contracts.ReviewSubmission{{
			RoundID:     "review-001-full",
			Slot:        "integrated",
			Role:        "integrated",
			SubmittedAt: "2026-07-13T10:04:00Z",
			Summary:     "Integrated review passed.",
			Findings:    []contracts.ReviewFinding{},
			ExtraFields: map[string]json.RawMessage{
				"worklog":  json.RawMessage(`{"full_plan_read":true,"checked_areas":["internal/review"],"open_questions":["Does repair coverage compose?"],"candidate_findings":["Potential gap"]}`),
				"coverage": json.RawMessage(`{"review_kind":"full","anchor_sha":"base123"}`),
			},
		}},
	}
	writeReviewRound(t, workdir, planStem, round)

	result := Service{Workdir: workdir}.Read()
	if !result.OK || len(result.Rounds) != 1 {
		t.Fatalf("expected one readable round, got %#v", result)
	}
	got := result.Rounds[0]
	if got.Status != "waiting_for_submissions" || got.TotalAssignments != 2 || got.SubmittedAssignments != 1 || got.PendingAssignments != 1 {
		t.Fatalf("expected assignment progress, got %#v", got)
	}
	if len(got.Reviewers) != 2 {
		t.Fatalf("expected two reviewer assignments, got %#v", got.Reviewers)
	}
	if got.Reviewers[0].Role != "integrated" || len(got.Reviewers[0].Dimensions) != 2 || got.Reviewers[0].Worklog == nil {
		t.Fatalf("expected integrated assignment guidance and worklog, got %#v", got.Reviewers[0])
	}
	worklog := got.Reviewers[0].Worklog
	if worklog.FullPlanRead == nil || !*worklog.FullPlanRead || worklog.ReviewKind != "full" || worklog.AnchorSHA != "base123" || len(worklog.CheckedAreas) != 1 || len(worklog.OpenQuestions) != 1 || len(worklog.CandidateFindings) != 1 {
		t.Fatalf("expected normalized progressive worklog, got %#v", worklog)
	}
	if got.Reviewers[1].Role != "specialist" || got.Reviewers[1].RiskBrief == nil || len(got.Reviewers[1].RiskBrief.Invariants) != 1 {
		t.Fatalf("expected specialist risk brief, got %#v", got.Reviewers[1])
	}
	if !strings.Contains(string(got.Reviewers[0].RawSubmission), `"role":"integrated"`) {
		t.Fatalf("expected raw assignment submission, got %s", got.Reviewers[0].RawSubmission)
	}
}

func TestServiceReadSurfacesFindingProvenanceAndRepairResolutions(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-repair.md", "Review repair")
	saveReviewState(t, workdir, planStem, 2, "review-002-delta")

	assignment := reviewAssignment(workdir, planStem, "review-002-delta", "integrated", "integrated", []contracts.ReviewResolvedDimension{
		resolvedDimension("correctness", "builtin+plan", "Verify the targeted repair."),
	})
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{"review-001-full:integrated:1"}}
	round := reviewRoundFixture{
		Manifest: contracts.ReviewManifest{
			RoundID:     "review-002-delta",
			Kind:        "delta",
			AnchorSHA:   "abc123",
			Revision:    2,
			ReviewTitle: "Repair review",
			Repair:      repair,
			PlanPath:    relPlanPath,
			PlanStem:    planStem,
			CreatedAt:   "2026-07-13T11:00:00Z",
			Assignments: []contracts.ReviewAssignment{assignment},
		},
		Ledger: contracts.ReviewLedger{
			RoundID:   "review-002-delta",
			Kind:      "delta",
			UpdatedAt: "2026-07-13T11:05:00Z",
			Assignments: []contracts.ReviewLedgerAssignment{
				ledgerAssignment(assignment, "submitted", "2026-07-13T11:04:00Z"),
			},
		},
		Aggregate: &contracts.ReviewAggregate{
			RoundID:      "review-002-delta",
			Kind:         "delta",
			Revision:     2,
			ReviewTitle:  "Repair review",
			Repair:       repair,
			Decision:     "changes_requested",
			AggregatedAt: "2026-07-13T11:06:00Z",
			BlockingFindings: []contracts.ReviewAggregateFinding{{
				FindingID: "review-002-delta:integrated:1",
				Slot:      "integrated",
				Role:      "integrated",
				Area:      "coverage-chain",
				Severity:  "important",
				Title:     "Repair chain is incomplete",
				Details:   "The repair does not close all referenced findings.",
			}},
			NonBlockingFindings:  []contracts.ReviewAggregateFinding{},
			ResolvedFindingIDs:   []string{"review-001-full:integrated:1"},
			UnresolvedFindingIDs: []string{"review-002-delta:integrated:1"},
		},
		Submissions: []contracts.ReviewSubmission{{
			RoundID:     "review-002-delta",
			Slot:        "integrated",
			Role:        "integrated",
			SubmittedAt: "2026-07-13T11:04:00Z",
			Summary:     "The original finding is fixed, but a new blocker remains.",
			Resolutions: []contracts.ReviewFindingResolution{{
				FindingID: "review-001-full:integrated:1",
				Status:    "resolved",
				Details:   "The targeted behavior now passes.",
			}},
			Findings: []contracts.ReviewFinding{{
				Area:     "coverage-chain",
				Severity: "important",
				Title:    "Repair chain is incomplete",
				Details:  "The repair does not close all referenced findings.",
			}},
		}},
	}
	writeReviewRound(t, workdir, planStem, round)

	result := Service{Workdir: workdir}.Read()
	got := result.Rounds[0]
	if got.RepairsRoundID != "review-001-full" || got.AnchorSHA != "abc123" {
		t.Fatalf("expected repair linkage, got %#v", got)
	}
	if len(got.Reviewers) != 1 || len(got.Reviewers[0].Resolutions) != 1 {
		t.Fatalf("expected reviewer resolution, got %#v", got.Reviewers)
	}
	if len(got.BlockingFindings) != 1 {
		t.Fatalf("expected blocking finding, got %#v", got.BlockingFindings)
	}
	finding := got.BlockingFindings[0]
	if finding.FindingID != "review-002-delta:integrated:1" || finding.Slot != "integrated" || finding.Role != "integrated" || finding.Area != "coverage-chain" {
		t.Fatalf("expected stable finding provenance, got %#v", finding)
	}
}

func TestServiceReadRejectsLegacyDimensionOwnedArtifacts(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-legacy.md", "Legacy review")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	roundDir := runstate.ReviewRoundDir(workdir, planStem, "review-001-full")
	writeJSONFile(t, filepath.Join(roundDir, "manifest.json"), map[string]any{
		"round_id": "review-001-full", "kind": "full", "revision": 1,
		"plan_path": relPlanPath, "plan_stem": planStem, "created_at": "2026-07-13T12:00:00Z",
		"ledger_path": filepath.Join(roundDir, "ledger.json"), "aggregate_path": filepath.Join(roundDir, "aggregate.json"),
		"submissions_dir": filepath.Join(roundDir, "submissions"),
		"dimensions":      []map[string]any{{"name": "Correctness", "slot": "correctness"}},
	})
	writeJSONFile(t, filepath.Join(roundDir, "ledger.json"), map[string]any{
		"round_id": "review-001-full", "kind": "full", "updated_at": "2026-07-13T12:01:00Z",
		"slots": []map[string]any{{"name": "Correctness", "slot": "correctness", "status": "pending"}},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK || len(result.Rounds) != 1 || result.Rounds[0].Status != "degraded" {
		t.Fatalf("expected legacy artifacts to degrade instead of being adapted, got %#v", result)
	}
	joined := strings.Join(result.Rounds[0].Warnings, " ")
	if !strings.Contains(joined, "assignments") {
		t.Fatalf("expected missing assignment warning, got %q", joined)
	}
}

func TestServiceReadRecoversAssignmentFromSubmissionOnly(t *testing.T) {
	workdir := t.TempDir()
	_, planStem := seedActivePlan(t, workdir, "2026-07-13-review-submission-only.md", "Submission only")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	roundDir := runstate.ReviewRoundDir(workdir, planStem, "review-001-full")
	writeJSONFile(t, filepath.Join(roundDir, "submissions", "integrated", "submission.json"), contracts.ReviewSubmission{
		RoundID:     "review-001-full",
		Slot:        "integrated",
		Role:        "integrated",
		SubmittedAt: "2026-07-13T12:10:00Z",
		Summary:     "Recovered from the assignment submission.",
		Findings:    []contracts.ReviewFinding{},
	})

	result := Service{Workdir: workdir}.Read()
	got := result.Rounds[0]
	if got.Status != "degraded" || got.TotalAssignments != 1 || got.SubmittedAssignments != 1 || got.PendingAssignments != 0 {
		t.Fatalf("expected conservative submission-only recovery, got %#v", got)
	}
	if len(got.Reviewers) != 1 || got.Reviewers[0].Role != "integrated" || got.Reviewers[0].Summary == "" {
		t.Fatalf("expected recovered assignment provenance, got %#v", got.Reviewers)
	}
}

func TestServiceReadKeepsUnknownAssignmentStatusConservative(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-unknown-status.md", "Unknown assignment status")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated", []contracts.ReviewResolvedDimension{
		resolvedDimension("correctness", "builtin", "Check behavior."),
	})
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest: contracts.ReviewManifest{
			RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem,
			CreatedAt: "2026-07-13T13:00:00Z", Assignments: []contracts.ReviewAssignment{assignment},
		},
		Ledger: contracts.ReviewLedger{
			RoundID: "review-001-full", Kind: "full", UpdatedAt: "2026-07-13T13:01:00Z",
			Assignments: []contracts.ReviewLedgerAssignment{{
				Slot: "integrated", Role: "integrated", Status: "mystery", SubmissionPath: assignment.SubmissionPath,
			}},
		},
		Submissions: []contracts.ReviewSubmission{{
			RoundID: "review-001-full", Slot: "integrated", Role: "integrated", SubmittedAt: "2026-07-13T13:02:00Z", Summary: "Submitted.",
		}},
	})

	got := Service{Workdir: workdir}.Read().Rounds[0]
	if got.Status != "waiting_for_submissions" || got.SubmittedAssignments != 0 || got.PendingAssignments != 1 {
		t.Fatalf("expected unknown assignment status to remain pending, got %#v", got)
	}
	if len(got.Reviewers[0].Warnings) == 0 || !strings.Contains(strings.Join(got.Reviewers[0].Warnings, " "), "unknown assignment status") {
		t.Fatalf("expected a conservative status warning, got %#v", got.Reviewers[0])
	}
}

func TestServiceReadArchivedPlanVisibilityAndLandHiding(t *testing.T) {
	t.Run("visible while waiting for merge", func(t *testing.T) {
		workdir := t.TempDir()
		relPlanPath, planStem := seedArchivedPlan(t, workdir, "2026-07-13-review-archived.md", "Archived review")
		saveReviewState(t, workdir, planStem, 1, "")
		assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated", []contracts.ReviewResolvedDimension{
			resolvedDimension("correctness", "builtin", "Check behavior."),
		})
		writeReviewRound(t, workdir, planStem, reviewRoundFixture{
			Manifest:    contracts.ReviewManifest{RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T14:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
			Ledger:      contracts.ReviewLedger{RoundID: "review-001-full", Kind: "full", UpdatedAt: "2026-07-13T14:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "submitted", "2026-07-13T14:01:00Z")}},
			Aggregate:   &contracts.ReviewAggregate{RoundID: "review-001-full", Kind: "full", Revision: 1, Decision: "pass", AggregatedAt: "2026-07-13T14:02:00Z", BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{}},
			Submissions: []contracts.ReviewSubmission{{RoundID: "review-001-full", Slot: "integrated", Role: "integrated", SubmittedAt: "2026-07-13T14:01:00Z", Summary: "Passed."}},
		})

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
			t.Fatalf("expected review data to be hidden during land cleanup, got %#v", result)
		}
	})
}

func TestServiceReadOrdersActiveAndHistoricalRounds(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-ordering.md", "Review ordering")
	saveReviewState(t, workdir, planStem, 3, "review-002-delta")
	for _, item := range []struct {
		roundID  string
		kind     string
		revision int
	}{
		{roundID: "review-001-full", kind: "full", revision: 1},
		{roundID: "review-002-delta", kind: "delta", revision: 2},
		{roundID: "review-010-full", kind: "full", revision: 3},
	} {
		assignment := reviewAssignment(workdir, planStem, item.roundID, "integrated", "integrated", []contracts.ReviewResolvedDimension{
			resolvedDimension("correctness", "builtin", "Check behavior."),
		})
		writeReviewRound(t, workdir, planStem, reviewRoundFixture{
			Manifest: contracts.ReviewManifest{RoundID: item.roundID, Kind: item.kind, Revision: item.revision, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T16:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
			Ledger:   contracts.ReviewLedger{RoundID: item.roundID, Kind: item.kind, UpdatedAt: "2026-07-13T16:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "pending", "")}},
		})
	}

	rounds := Service{Workdir: workdir}.Read().Rounds
	if len(rounds) != 3 || rounds[0].RoundID != "review-002-delta" || rounds[1].RoundID != "review-010-full" || rounds[2].RoundID != "review-001-full" {
		t.Fatalf("expected active-first then descending sequence order, got %#v", rounds)
	}
}

func TestServiceReadDegradesDamagedCoreArtifacts(t *testing.T) {
	tests := []struct {
		name           string
		damage         func(t *testing.T, roundDir string)
		warning        string
		expectStatus   string
		expectDecision string
	}{
		{
			name: "missing ledger",
			damage: func(t *testing.T, roundDir string) {
				t.Helper()
				if err := os.Remove(filepath.Join(roundDir, "ledger.json")); err != nil {
					t.Fatalf("remove ledger: %v", err)
				}
			},
			warning:        "Ledger is missing",
			expectStatus:   "degraded",
			expectDecision: "pass",
		},
		{
			name: "malformed ledger",
			damage: func(t *testing.T, roundDir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(roundDir, "ledger.json"), []byte("{not-json"), 0o644); err != nil {
					t.Fatalf("damage ledger: %v", err)
				}
			},
			warning:        "not valid JSON",
			expectStatus:   "degraded",
			expectDecision: "pass",
		},
		{
			name: "unreadable aggregate",
			damage: func(t *testing.T, roundDir string) {
				t.Helper()
				path := filepath.Join(roundDir, "aggregate.json")
				if err := os.Remove(path); err != nil {
					t.Fatalf("remove aggregate: %v", err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatalf("replace aggregate with directory: %v", err)
				}
			},
			warning:        "Unable to read aggregate",
			expectStatus:   "degraded",
			expectDecision: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workdir := t.TempDir()
			relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-damage.md", "Review damage")
			saveReviewState(t, workdir, planStem, 1, "review-001-full")
			assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated", []contracts.ReviewResolvedDimension{
				resolvedDimension("correctness", "builtin", "Check behavior."),
			})
			writeReviewRound(t, workdir, planStem, reviewRoundFixture{
				Manifest:    contracts.ReviewManifest{RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T17:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
				Ledger:      contracts.ReviewLedger{RoundID: "review-001-full", Kind: "full", UpdatedAt: "2026-07-13T17:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "submitted", "2026-07-13T17:01:00Z")}},
				Aggregate:   &contracts.ReviewAggregate{RoundID: "review-001-full", Kind: "full", Revision: 1, Decision: "pass", AggregatedAt: "2026-07-13T17:02:00Z", BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{}},
				Submissions: []contracts.ReviewSubmission{{RoundID: "review-001-full", Slot: "integrated", Role: "integrated", SubmittedAt: "2026-07-13T17:01:00Z", Summary: "Passed."}},
			})
			roundDir := runstate.ReviewRoundDir(workdir, planStem, "review-001-full")
			tc.damage(t, roundDir)

			got := Service{Workdir: workdir}.Read().Rounds[0]
			if got.Status != tc.expectStatus || got.Decision != tc.expectDecision {
				t.Fatalf("expected damaged round to preserve decision conservatively, got %#v", got)
			}
			if !strings.Contains(strings.Join(got.Warnings, " "), tc.warning) {
				t.Fatalf("expected warning %q, got %#v", tc.warning, got.Warnings)
			}
		})
	}
}

func TestServiceReadDegradesMalformedReviewerWorklogConservatively(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-07-13-review-worklog.md", "Review worklog")
	saveReviewState(t, workdir, planStem, 1, "review-001-delta")
	assignment := reviewAssignment(workdir, planStem, "review-001-delta", "integrated", "integrated", []contracts.ReviewResolvedDimension{
		resolvedDimension("risk-scan", "builtin", "Check degraded parsing."),
	})
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest: contracts.ReviewManifest{RoundID: "review-001-delta", Kind: "delta", AnchorSHA: "anchor123", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T18:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
		Ledger:   contracts.ReviewLedger{RoundID: "review-001-delta", Kind: "delta", UpdatedAt: "2026-07-13T18:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "submitted", "2026-07-13T18:01:00Z")}},
		Submissions: []contracts.ReviewSubmission{{
			RoundID: "review-001-delta", Slot: "integrated", Role: "integrated", SubmittedAt: "2026-07-13T18:01:00Z", Summary: "Partially readable.",
			ExtraFields: map[string]json.RawMessage{
				"worklog":  json.RawMessage(`{"full_plan_read":"yes","checked_areas":["web/src/pages.tsx"],"open_questions":"bad","candidate_findings":["Candidate"]}`),
				"coverage": json.RawMessage(`{"review_kind":7,"anchor_sha":"worklog-anchor"}`),
			},
		}},
	})

	reviewer := Service{Workdir: workdir}.Read().Rounds[0].Reviewers[0]
	if reviewer.Worklog == nil || reviewer.Worklog.FullPlanRead != nil || reviewer.Worklog.ReviewKind != "" || reviewer.Worklog.AnchorSHA != "worklog-anchor" || len(reviewer.Worklog.CheckedAreas) != 1 || len(reviewer.Worklog.CandidateFindings) != 1 {
		t.Fatalf("expected partial worklog recovery, got %#v", reviewer.Worklog)
	}
	if !strings.Contains(strings.Join(reviewer.Warnings, " "), "malformed") {
		t.Fatalf("expected malformed worklog warnings, got %#v", reviewer.Warnings)
	}
}

func TestServiceReadUsesConfiguredRuntimeAndRepoFacingArtifactPaths(t *testing.T) {
	workdir := t.TempDir()
	writeJSONFile(t, filepath.Join(workdir, ".harness", "config.yaml"), map[string]any{
		"version": 1,
		"paths": map[string]any{
			"plans":         map[string]any{"active": "workflow/plans/open", "archived": "workflow/plans/done"},
			"local_runtime": "tmp/harness-runtime",
		},
	})
	relPlanPath, planStem := seedPlan(t, workdir, "workflow/plans/open/2026-07-13-configured-review.md", "Configured review")
	saveReviewState(t, workdir, planStem, 1, "review-001-full")
	assignment := reviewAssignment(workdir, planStem, "review-001-full", "integrated", "integrated", []contracts.ReviewResolvedDimension{
		resolvedDimension("correctness", "builtin", "Check behavior."),
	})
	writeReviewRound(t, workdir, planStem, reviewRoundFixture{
		Manifest:    contracts.ReviewManifest{RoundID: "review-001-full", Kind: "full", Revision: 1, PlanPath: relPlanPath, PlanStem: planStem, CreatedAt: "2026-07-13T19:00:00Z", Assignments: []contracts.ReviewAssignment{assignment}},
		Ledger:      contracts.ReviewLedger{RoundID: "review-001-full", Kind: "full", UpdatedAt: "2026-07-13T19:01:00Z", Assignments: []contracts.ReviewLedgerAssignment{ledgerAssignment(assignment, "submitted", "2026-07-13T19:01:00Z")}},
		Submissions: []contracts.ReviewSubmission{{RoundID: "review-001-full", Slot: "integrated", Role: "integrated", SubmittedAt: "2026-07-13T19:01:00Z", Summary: "Passed."}},
	})

	result := Service{Workdir: workdir}.Read()
	if result.Artifacts == nil || result.Artifacts.PlanPath != relPlanPath || len(result.Rounds) != 1 {
		t.Fatalf("expected configured review source, got %#v", result)
	}
	path := result.Rounds[0].Reviewers[0].SubmissionPath
	if filepath.IsAbs(path) || !strings.HasPrefix(path, "tmp/harness-runtime/") {
		t.Fatalf("expected repo-facing configured artifact path, got %q", path)
	}
	if strings.Contains(string(result.Rounds[0].Reviewers[0].RawSubmission), "submission_path") {
		t.Fatalf("expected raw submission artifact to stay sanitized, got %s", result.Rounds[0].Reviewers[0].RawSubmission)
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

func resolvedDimension(name, source, instructions string) contracts.ReviewResolvedDimension {
	return contracts.ReviewResolvedDimension{
		Name: name, Sources: []string{source}, Description: "Guidance for " + name + ".", Instructions: instructions,
	}
}

func reviewAssignment(workdir, planStem, roundID, slot, role string, dimensions []contracts.ReviewResolvedDimension) contracts.ReviewAssignment {
	return contracts.ReviewAssignment{
		Slot: slot, Role: role, Dimensions: dimensions, Instructions: "Review the assigned candidate.",
		SubmissionPath: roundArtifactPath(workdir, planStem, roundID, filepath.Join("submissions", slot, "submission.json")),
	}
}

func ledgerAssignment(assignment contracts.ReviewAssignment, status, submittedAt string) contracts.ReviewLedgerAssignment {
	return contracts.ReviewLedgerAssignment{
		Slot: assignment.Slot, Role: assignment.Role, Status: status, SubmittedAt: submittedAt, SubmissionPath: assignment.SubmissionPath,
	}
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
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: title})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	return relPlanPath, strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
}

func saveReviewState(t *testing.T, workdir, planStem string, revision int, activeRoundID string) {
	t.Helper()
	state := &runstate.State{Revision: revision}
	if activeRoundID != "" {
		state.ActiveReviewRound = &runstate.ReviewRound{RoundID: activeRoundID, Kind: "full", Revision: revision}
	}
	if _, err := runstate.SaveState(workdir, planStem, state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func roundArtifactPath(workdir, planStem, roundID, suffix string) string {
	return filepath.Join(runstate.ReviewRoundDir(workdir, planStem, roundID), suffix)
}
