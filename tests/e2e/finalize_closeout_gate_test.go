package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const finalizeGatePlanTitle = "Finalize Closeout Gate Plan"

func TestFinalizeReviewStartAndArchiveRejectEarlierCloseoutDebtWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-04-08-finalize-closeout-gate-e2e.md"
	planPath := workspace.Path(planRelPath)
	planStem := "2026-04-08-finalize-closeout-gate-e2e"

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", finalizeGatePlanTitle,
		"--timestamp", "2026-04-08T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#24",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, finalizeGatePlanTitle, finalizeCloseoutGatePlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)
	support.ApprovePlan(t, planPath, "2026-04-08T00:05:00Z")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	support.CompleteStep(
		t,
		planPath,
		1,
		"Left Step 1 done without step-closeout evidence so the built binary has real earlier-step debt to reject.",
		"Completed review notes without a clean step-closeout review artifact.",
	)
	support.CompleteStep(
		t,
		planPath,
		2,
		"Completed the later step while explicitly recording that no separate step-closeout review was needed for the fixture.",
		"NO_STEP_REVIEW_NEEDED: The later fixture step is only here to move the candidate to the finalize boundary.",
	)

	finalizeReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, finalizeReviewStatus, "execution/finalize/review")

	reviewSpecPath := workspace.WriteJSON(t, "tmp/finalize-gate-review-spec.json", map[string]any{
		"kind": "full",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Verify the built binary rejects finalize review while earlier-step closeout debt remains.",
			},
		},
	})
	reviewStart := support.Run(t, workspace.Root, "review", "start", "--spec", reviewSpecPath)
	support.RequireExitCode(t, reviewStart, 1)
	support.RequireNoStderr(t, reviewStart)
	reviewStartPayload := support.RequireJSONResult[struct {
		OK      bool           `json:"ok"`
		Command string         `json:"command"`
		Summary string         `json:"summary"`
		Errors  []commandError `json:"errors"`
	}](t, reviewStart)
	if reviewStartPayload.OK || reviewStartPayload.Command != "review start" {
		t.Fatalf("expected failed finalize review-start payload, got %#v", reviewStartPayload)
	}
	if reviewStartPayload.Summary != "Review spec does not match the current workflow state." {
		t.Fatalf("unexpected finalize review-start summary: %#v", reviewStartPayload)
	}
	if len(reviewStartPayload.Errors) != 1 || reviewStartPayload.Errors[0].Path != "spec" {
		t.Fatalf("expected one spec-scoped finalize review-start error, got %#v", reviewStartPayload.Errors)
	}
	if !strings.Contains(reviewStartPayload.Errors[0].Message, "Step 1: Earlier step closeout debt") || !strings.Contains(reviewStartPayload.Errors[0].Message, "spec.step=1") {
		t.Fatalf("expected explicit repair guidance in finalize review-start error, got %#v", reviewStartPayload.Errors)
	}

	statePath := workspace.WriteJSON(t, ".local/harness/plans/"+planStem+"/state.json", map[string]any{
		"execution_started_at": "2026-04-08T00:05:00Z",
		"plan_path":            planRelPath,
		"plan_stem":            planStem,
		"revision":             1,
		"active_review_round": map[string]any{
			"round_id":   "review-001-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})
	support.RequireFileExists(t, statePath)

	finalizeArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, finalizeArchiveStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireExitCode(t, archive, 1)
	support.RequireNoStderr(t, archive)
	archivePayload := support.RequireJSONResult[struct {
		OK      bool           `json:"ok"`
		Command string         `json:"command"`
		Summary string         `json:"summary"`
		Errors  []commandError `json:"errors"`
	}](t, archive)
	if archivePayload.OK || archivePayload.Command != "archive" {
		t.Fatalf("expected archive failure payload, got %#v", archivePayload)
	}
	if archivePayload.Summary != "Current plan is not archive-ready." {
		t.Fatalf("unexpected archive summary: %#v", archivePayload)
	}
	foundCloseoutError := false
	for _, issue := range archivePayload.Errors {
		if issue.Path == "plan.steps[0].review_notes" && strings.Contains(issue.Message, "Step 1: Earlier step closeout debt") {
			foundCloseoutError = true
			break
		}
	}
	if !foundCloseoutError {
		t.Fatalf("expected earlier-step closeout archive error among %#v", archivePayload.Errors)
	}
	support.RequireFileExists(t, planPath)
}

func finalizeCloseoutGatePlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise the built binary paths that reject default finalize review start and
archive when an earlier completed step still lacks review-complete closeout.

## Scope

### In Scope

- create a finalize-bound candidate with one earlier-step closeout debt
- assert built-binary finalize review start rejects default finalize binding
- assert built-binary archive rejects the same debt even when finalize review
  state looks clean

### Out of Scope

- repairing the earlier step debt
- archive/publish success after debt repair

## Acceptance Criteria

- [ ] built-binary finalize review start rejects unresolved earlier-step debt
- [ ] built-binary archive rejects unresolved earlier-step debt

## Deferred Items

- NONE.

## Work Breakdown

### Step 1: Earlier step closeout debt

- Done: [ ]

#### Objective

Create one completed earlier step that intentionally lacks review-complete
closeout.

#### Details

NONE

#### Expected Files

- tests/e2e/finalize_closeout_gate_test.go

#### Validation

- Built-binary status resolves to finalize review after both steps are marked
  done.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Later step reaches finalize boundary

- Done: [ ]

#### Objective

Move the candidate to the finalize boundary without introducing additional
closeout debt.

#### Details

NONE

#### Expected Files

- tests/e2e/finalize_closeout_gate_test.go

#### Validation

- Built-binary review start and archive both reject the earlier-step debt.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run the built-binary E2E that asserts finalize review start and archive reject
  earlier-step closeout debt.

## Risks

- Risk: The fixture could accidentally satisfy the debt and stop exercising the
  blocked paths.
  - Mitigation: Assert the exact built-binary failure payloads for both
    commands.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
`)
}
