package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const finalizeGatePlanTitle = "Finalize Closeout Gate Plan"

func TestUnreviewedStepsAdvanceButStartedStepReviewBindsWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-04-08-finalize-closeout-gate-e2e.md"
	planPath := workspace.Path(planRelPath)

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
		"Completed Step 1 without starting an optional step review.",
		"No step review was started; finalize-first review remains the default.",
	)
	assertNode(t, runStatus(t, workspace.Root), "execution/step-2/implement")

	workspace.CommitAll(t, "checkpoint optional step review candidate")
	anchor := currentWorkspaceHead(t, workspace.Root)
	started := startReviewRound(t, workspace, "tmp/optional-step-review.json", map[string]any{
		"step":       1,
		"kind":       "delta",
		"anchor_sha": anchor,
		"assignments": []map[string]any{
			integratedAssignment("integrated", "Check the intentionally selected step boundary.", "correctness"),
		},
	})
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/review")
	submitReviewSlot(t, workspace, started.Artifacts.RoundID, started.Artifacts.Slots[0], "The optional step review found a blocker.", []map[string]any{
		{
			"area":     "step-boundary",
			"severity": "important",
			"title":    "Resolve the intentionally reviewed boundary",
			"details":  "Once started, this optional step review must be resolved before ordinary progression resumes.",
		},
	})
	aggregate := aggregateReviewRound(t, workspace, started.Artifacts.RoundID)
	if aggregate.Review.Decision != "changes_requested" {
		t.Fatalf("expected intentionally started step review to bind after findings, got %#v", aggregate)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/step-1/implement")
}

func finalizeCloseoutGatePlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise the finalize-first contract: an unreviewed completed step advances
normally, while an intentionally started optional step review remains binding.

## Scope

### In Scope

- complete a step without starting review and advance to the next step
- intentionally start a step-bound review from the later frontier
- prove blocking findings keep that explicitly reviewed step current

### Out of Scope

- resolving the intentionally started step review
- finalize review, archive, and publish

## Acceptance Criteria

- [ ] a completed step without review advances to the next unfinished step
- [ ] an intentionally started step review enters the explicit review node
- [ ] blocking findings from that review keep the reviewed step current

## Deferred Items

- NONE.

## Work Breakdown

### Step 1: Advance without optional review

- Done: [ ]

#### Objective

Complete one step without starting an optional review.

#### Details

NONE

#### Expected Files

- tests/e2e/finalize_closeout_gate_test.go

#### Validation

- Built-binary status resolves to the next unfinished step.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Start an intentional step review

- Done: [ ]

#### Objective

Bind an optional review to the earlier step and record blocking findings.

#### Details

NONE

#### Expected Files

- tests/e2e/finalize_closeout_gate_test.go

#### Validation

- Built-binary review state remains bound after blocking findings aggregate.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run the built-binary E2E that contrasts default advancement with an explicitly
  started review boundary.

## Risks

- Risk: Derived status could look only at step completion and ignore an active
  review.
  - Mitigation: Assert both the in-review node and the post-aggregate repair
    node.

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
