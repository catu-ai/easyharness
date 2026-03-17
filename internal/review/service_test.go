package review_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/review"
	"github.com/yzhang1918/superharness/internal/runstate"
)

func TestStartCreatesRoundAndUpdatesState(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 123456789, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind:    "delta",
		Target:  "Step 4: Implement the review-round contract",
		Trigger: "step_closeout",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check the state and artifact contract."},
			{Name: "agent_ux", Instructions: "Check that outputs are agent-friendly."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected start success, got %#v", result)
	}
	if result.Artifacts == nil || len(result.Artifacts.Slots) != 2 {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
	if _, err := os.Stat(result.Artifacts.ManifestPath); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestStartRejectsInvalidSpec(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{Workdir: root}
	result := svc.Start(mustJSON(t, map[string]any{
		"kind":       "delta",
		"target":     "",
		"trigger":    "",
		"dimensions": []any{},
	}))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertStartError(t, result, "spec.target")
	assertStartError(t, result, "spec.trigger")
	assertStartError(t, result, "spec.dimensions")
}

func TestSubmitStoresSubmissionAndUpdatesLedger(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind:    "delta",
		Target:  "Step 4",
		Trigger: "step_closeout",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 5, 0, 0, time.UTC)
	}
	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !result.OK {
		t.Fatalf("expected submit success, got %#v", result)
	}
	if _, err := os.Stat(result.Artifacts.SubmissionPath); err != nil {
		t.Fatalf("submission missing: %v", err)
	}
}

func TestSubmitRejectsUnknownSlot(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind:    "delta",
		Target:  "Step 4",
		Trigger: "step_closeout",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "missing", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if result.OK {
		t.Fatalf("expected submit failure, got %#v", result)
	}
	assertSubmitError(t, result, "slot")
}

func TestAggregateRejectsMissingSubmission(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind:    "delta",
		Target:  "Step 4",
		Trigger: "step_closeout",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Aggregate(start.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected aggregate failure, got %#v", result)
	}
	assertAggregateError(t, result, "submissions")
}

func TestAggregateDeltaPassUpdatesState(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind:    "delta",
		Target:  "Step 4",
		Trigger: "step_closeout",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
	}
	result := svc.Aggregate(start.Artifacts.RoundID)
	if !result.OK || result.Review == nil || result.Review.Decision != "pass" {
		t.Fatalf("unexpected aggregate result: %#v", result)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || !state.ActiveReviewRound.Aggregated {
		t.Fatalf("expected aggregated state, got %#v", state)
	}
}

func TestAggregateFullWithBlockingFindings(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind:    "full",
		Target:  "Full branch candidate before archive",
		Trigger: "pre_archive",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Found a blocker.",
		Findings: []review.Finding{
			{
				Severity: "important",
				Title:    "Missing validation",
				Details:  "The archive path is missing one required validation.",
			},
		},
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 12, 0, 0, time.UTC)
	}
	result := svc.Aggregate(start.Artifacts.RoundID)
	if !result.OK || result.Review == nil || result.Review.Decision != "changes_requested" {
		t.Fatalf("unexpected aggregate result: %#v", result)
	}
	if len(result.Review.BlockingFindings) != 1 {
		t.Fatalf("expected one blocking finding, got %#v", result.Review)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Fix the blocking findings before archive") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func writeExecutingPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Review Contract Plan",
		Timestamp:  time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "lifecycle: awaiting_plan_approval", "lifecycle: executing", 1)
	rendered = strings.Replace(rendered, "- Status: pending", "- Status: in_progress", 1)

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return path
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func assertStartError(t *testing.T, result review.StartResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected start error for %s, got %#v", path, result.Errors)
}

func assertSubmitError(t *testing.T, result review.SubmitResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected submit error for %s, got %#v", path, result.Errors)
}

func assertAggregateError(t *testing.T, result review.AggregateResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected aggregate error for %s, got %#v", path, result.Errors)
}
