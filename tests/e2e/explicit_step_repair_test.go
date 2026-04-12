package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	explicitRepairPlanTitle = "Explicit Step Repair Plan"
	explicitRepairStepOne   = "Establish earlier closeout target"
	explicitRepairStepTwo   = "Keep later frontier active"
)

func TestExplicitStepRepairTransitionsWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-04-08-explicit-step-repair-e2e.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", explicitRepairPlanTitle,
		"--timestamp", "2026-04-08T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#53",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, explicitRepairPlanTitle, explicitStepRepairPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)
	support.ApprovePlan(t, planPath, "2026-04-08T00:05:00Z")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/implement")

	support.CompleteStep(
		t,
		planPath,
		1,
		"Established the earlier closeout target and intentionally left real earlier-step closeout debt for later explicit repair rounds to clear.",
		"Completed review notes without review-complete closeout evidence so explicit earlier-step repair must clear the debt before finalize can proceed.",
	)

	stepTwoStatus := runStatus(t, workspace.Root)
	assertNode(t, stepTwoStatus, "execution/step-2/implement")
	anchorSHA := currentWorkspaceHead(t, workspace.Root)

	failingRepair := startReviewRound(t, workspace, "tmp/explicit-step1-repair-fail.json", map[string]any{
		"step":       1,
		"kind":       "delta",
		"anchor_sha": anchorSHA,
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Exercise explicit earlier-step repair from the later frontier.",
			},
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/review")
	submitReviewSlot(t, workspace, failingRepair.Artifacts.RoundID, failingRepair.Artifacts.Slots[0], "Explicit repair still needs work.", []map[string]any{
		{
			"severity": "important",
			"title":    "Earlier step needs one more pass",
			"details":  "Keep the repaired step current until the debt is resolved.",
		},
	})
	failingRepairAggregate := aggregateReviewRound(t, workspace, failingRepair.Artifacts.RoundID)
	if failingRepairAggregate.Review.Decision != "changes_requested" {
		t.Fatalf("expected explicit earlier-step repair to fail first, got %#v", failingRepairAggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/implement")

	cleanRepair := startReviewRound(t, workspace, "tmp/explicit-step1-repair-pass.json", map[string]any{
		"step":       1,
		"kind":       "delta",
		"anchor_sha": anchorSHA,
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Exercise clean explicit earlier-step repair returning to the later frontier.",
			},
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/review")
	submitReviewSlot(t, workspace, cleanRepair.Artifacts.RoundID, cleanRepair.Artifacts.Slots[0], "Explicit earlier-step repair is now clean.", nil)
	cleanRepairAggregate := aggregateReviewRound(t, workspace, cleanRepair.Artifacts.RoundID)
	if cleanRepairAggregate.Review.Decision != "pass" {
		t.Fatalf("expected explicit earlier-step repair to pass on rerun, got %#v", cleanRepairAggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/step-2/implement")

	support.CheckAllAcceptanceCriteria(t, planPath)
	support.CompleteStep(
		t,
		planPath,
		2,
		"Verified later-frontier execution and explicit earlier-step repair transitions through the built binary.",
		"NO_STEP_REVIEW_NEEDED: This E2E fixture uses explicit earlier-step repair rounds and finalize review directly instead of ordinary step-2 closeout review.",
	)

	finalizeReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, finalizeReviewStatus, "execution/finalize/review")

	finalizeFailure := startReviewRound(t, workspace, "tmp/finalize-failure.json", map[string]any{
		"kind": "full",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Prove the ordinary finalize path is restored after explicit earlier-step repair clears the debt.",
			},
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/review")
	submitReviewSlot(t, workspace, finalizeFailure.Artifacts.RoundID, finalizeFailure.Artifacts.Slots[0], "Finalize review found one follow-up.", []map[string]any{
		{
			"severity": "important",
			"title":    "Need finalize-fix state",
			"details":  "Drive the fixture into finalize fix before one more explicit repair start.",
		},
	})
	finalizeFailureAggregate := aggregateReviewRound(t, workspace, finalizeFailure.Artifacts.RoundID)
	if finalizeFailureAggregate.Review.Decision != "changes_requested" {
		t.Fatalf("expected repaired candidate to allow a default finalize review start before moving into finalize fix, got %#v", finalizeFailureAggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/fix")

	finalizeFixRepair := startReviewRound(t, workspace, "tmp/finalize-fix-explicit-step1.json", map[string]any{
		"step":       1,
		"kind":       "delta",
		"anchor_sha": anchorSHA,
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Exercise explicit earlier-step repair from finalize fix.",
			},
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/review")
	if !strings.HasSuffix(finalizeFixRepair.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected explicit repair from finalize fix to use delta fixture round id, got %#v", finalizeFixRepair)
	}
	submitReviewSlot(t, workspace, finalizeFixRepair.Artifacts.RoundID, finalizeFixRepair.Artifacts.Slots[0], "Earlier-step repair from finalize fix is clean.", nil)
	finalizeFixRepairAggregate := aggregateReviewRound(t, workspace, finalizeFixRepair.Artifacts.RoundID)
	if finalizeFixRepairAggregate.Review.Decision != "pass" {
		t.Fatalf("expected explicit repair from finalize fix to pass, got %#v", finalizeFixRepairAggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/review")

	finalizeReviewRepair := startReviewRound(t, workspace, "tmp/finalize-review-explicit-step1.json", map[string]any{
		"step": 1,
		"kind": "full",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Exercise explicit earlier-step repair from finalize review after the ordinary finalize path has been restored.",
			},
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/review")
	submitReviewSlot(t, workspace, finalizeReviewRepair.Artifacts.RoundID, finalizeReviewRepair.Artifacts.Slots[0], "Earlier-step repair from finalize review is clean.", nil)
	finalizeReviewRepairAggregate := aggregateReviewRound(t, workspace, finalizeReviewRepair.Artifacts.RoundID)
	if finalizeReviewRepairAggregate.Review.Decision != "pass" {
		t.Fatalf("expected finalize-review explicit repair to pass, got %#v", finalizeReviewRepairAggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/review")
}

func explicitStepRepairPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise the built-binary transitions for real explicit earlier-step closeout
repair from a later execution frontier, prove that the repaired candidate can
re-enter ordinary finalize review, and cover explicit repair from finalize
review and finalize fix.

## Scope

### In Scope

- leave real earlier-step closeout debt behind after a completed earlier step
- start an explicit earlier-step repair from a later unfinished step
- cover failing and passing explicit repair aggregates
- prove a clean explicit repair restores the normal finalize review path
- re-enter explicit repair from finalize review and finalize fix

### Out of Scope

- archive and publish handoff for this fixture
- reviewer-subagent orchestration

## Acceptance Criteria

- [ ] later-frontier explicit repair can fail and pin the earlier step
- [ ] later-frontier explicit repair can pass and return to the ordinary frontier
- [ ] a clean explicit repair clears real earlier-step debt so ordinary finalize review can start
- [ ] finalize review and finalize fix can both start explicit earlier-step repair

## Deferred Items

- NONE.

## Work Breakdown

### Step 1: Establish earlier closeout target

- Done: [ ]

#### Objective

Create the earlier completed step that later explicit repair rounds will target.

#### Details

NONE

#### Expected Files

- tests/e2e/explicit_step_repair_test.go

#### Validation

- Built-binary status reaches execution/step-2/implement.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Keep later frontier active

- Done: [ ]

#### Objective

Exercise the explicit earlier-step repair transitions from later-step and
finalize contexts.

#### Details

NONE

#### Expected Files

- tests/e2e/explicit_step_repair_test.go

#### Validation

- Built-binary review start and aggregate transitions match the tracked spec.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run the built-binary E2E that exercises explicit earlier-step repair paths.

## Risks

- Risk: The fixture could drift from the tracked state-transition contract.
  - Mitigation: Assert the key nodes directly through built-binary harness status.

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
