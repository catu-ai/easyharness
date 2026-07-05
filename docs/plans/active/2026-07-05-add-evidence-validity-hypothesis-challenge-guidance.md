---
template_version: 0.2.0
created_at: "2026-07-05T20:09:47+08:00"
approved_at: "2026-07-05T20:34:56+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/272
    - https://github.com/catu-ai/easyharness/issues/262
size: S
---

# Add Evidence-Validity And Hypothesis-Challenge Guidance

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add the missing goal-oriented evidence and challenge vocabulary for v0.6.0 so
controllers and reviewers can test conclusions, hypotheses, scorecard movement,
and residual uncertainty, not only implementation diffs.

The finished slice should preserve the existing review model: formal `review`
remains the hard gate for step closeout and finalize readiness, while
`challenge` remains optional advisory intervention during checkpointed
exploration. This slice should add the smallest durable catalog and guidance
changes needed for future goal-oriented plans to request evidence-focused
formal review and optional checkpoint challenge without changing review
aggregation semantics.

## Scope

### In Scope

- Add a built-in `evidence-validity` review dimension for formal review of
  conclusion support, evidence quality, scorecard alignment, rejected
  hypotheses, residual uncertainty, and follow-up handling.
- Document `hypothesis-challenge` as a checkpoint advisory action, not a
  review dimension, without making every checkpoint a review gate.
- Update goal-oriented review/archive guidance to explain when step-closeout or
  finalize review should include evidence or hypothesis-focused review.
- Update controller review orchestration guidance only as needed to keep
  dimension selection deliberate and to point goal-oriented closeout at the
  new dimension when it fits.
- Keep managed skill edits in `assets/bootstrap/` first and sync materialized
  `.agents/skills/` output if any managed skill changes.
- Add focused tests or validation for the built-in dimension catalog behavior.

### Out of Scope

- Redesigning review aggregation or reviewer submission semantics.
- Adding automatic dimension injection or defaulting every goal-oriented review
  to every built-in dimension.
- Implementing plan-scoped dimensions from #263.
- Implementing #273 status or next-action behavior.
- Implementing #274 structural lint coverage.
- Implementing #275 user-facing docs, examples, or tutorials.
- Removing the `goal_oriented` authoring-preview boundary from #270.

## Acceptance Criteria

- [x] `harness review dimensions list` exposes a built-in
  `evidence-validity` dimension with concise controller-facing selection
  guidance.
- [x] `harness review dimensions instructions evidence-validity` gives
  reviewer guidance for testing whether goal-oriented conclusions are supported
  by evidence, scorecards, probes, rejected hypotheses, residuals, and
  follow-up handling.
- [x] The goal-oriented spec distinguishes formal evidence-focused review from
  checkpoint advisory challenge and describes `hypothesis-challenge` as a
  checkpoint action rather than a review dimension or automatic hard gate.
- [x] Review orchestration guidance reminds controllers to choose dimensions
  deliberately and explains when goal-oriented finalize review should consider
  `evidence-validity`.
- [x] Existing review dimensions and repo-defined overrides continue to work.
- [x] The wording avoids narrowing the feature to academic research terminology
  and keeps #273, #274, and #275 explicitly deferred.

## Deferred Items

- #273 owns goal-oriented status and next-action behavior.
- #274 owns full structural lint coverage for goal-oriented plans.
- #275 owns user-facing docs, help text, and examples.
- #263 owns plan-scoped dimensions.

## Work Breakdown

### Step 1: Add the formal evidence review dimension

- Done: [x]

#### Objective

Add `evidence-validity` to the built-in review dimension catalog and update
focused tests for catalog listing and instruction retrieval.

#### Details

The dimension should be useful beyond academic research work. It should tell a
reviewer to check whether accepted conclusions are supported by the approved
scorecard, checkpoint reports, probes or experiments, durable evidence,
rejected hypotheses or candidate directions, residual uncertainty, and
follow-up handling.

Keep the catalog reusable. Do not make the dimension auto-run, plan-scoped, or
special-cased for `workflow_profile: goal_oriented`.

#### Expected Files

- `internal/reviewdimensions/service.go`
- `internal/reviewdimensions/service_test.go`

#### Validation

- `go test ./internal/reviewdimensions`
- `harness review dimensions list` includes `evidence-validity`.
- `harness review dimensions instructions evidence-validity` returns the new
  instruction body.

#### Execution Notes

Added the built-in `evidence-validity` review dimension and focused unit
coverage for listing and instruction retrieval. Rebuilt the dev `harness`
binary after the Go change. Validation passed for `go test
./internal/reviewdimensions`, `harness review dimensions list`, and `harness
review dimensions instructions evidence-validity`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 connects the same catalog behavior into the
goal-oriented spec and controller guidance, so review is more useful after the
behavior and guidance are aligned.

### Step 2: Add goal-oriented challenge and review guidance

- Done: [x]

#### Objective

Update normative and controller-facing guidance so future goal-oriented plans
can request formal evidence-focused review and optional advisory
`hypothesis-challenge` at checkpoint boundaries.

#### Details

Update the goal-oriented workflow spec to replace the #272 placeholder with
clear vocabulary:

- `evidence-validity` is a formal review dimension for step-closeout or
  finalize review when conclusions, synthesis, or decision artifacts need
  evidence-focused scrutiny.
- `hypothesis-challenge` is a checkpoint advisory action, not a review
  dimension. It is considered at checkpoint boundaries when the controller
  needs alternative hypotheses, sharper probes, weak-evidence critique,
  premature-convergence checks, or collective brainstorming.
- Challenge triggers include competing hypotheses that current probes cannot
  distinguish, evidence too weak for a pending conclusion, scorecard plateau
  before a pivot, high-impact pre-synthesis decisions, explicit plan triggers,
  and human requests.
- Challenge output is recorded in the checkpoint report's `Challenge` field
  when it sharpens the current decision, or in a new checkpoint report when it
  materially changes direction.
- Challenge remains controller input, not a state transition, review round, or
  review aggregate.
- The controller chooses dimensions deliberately; not every built-in dimension
  always runs.
- Goal-oriented finalize review should include evidence/hypothesis-focused
  review when the final synthesis depends on adaptive conclusions, rejected
  alternatives, residual uncertainty, or a durable decision artifact.
- Future status hints for when to consider `hypothesis-challenge` belong to
  #273, not this slice.

If controller skill guidance changes, edit `assets/bootstrap/skills/` and run
`scripts/sync-bootstrap-assets` so `.agents/skills/` stays materialized output.

#### Expected Files

- `docs/specs/goal-oriented-workflow.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`

#### Validation

- Targeted `rg` checks for `evidence-validity`, `hypothesis-challenge`,
  `checkpoint advisory action`, `formal review`, and sibling issue boundaries.
- If bootstrap assets change, `scripts/sync-bootstrap-assets` and inspect the
  synced diff.

#### Execution Notes

Updated the goal-oriented workflow spec to define `hypothesis-challenge` as a
checkpoint advisory action rather than a review dimension, including triggers
and checkpoint-report output. Added `evidence-validity` formal review guidance
for goal-oriented step-closeout and finalize review. Updated managed review
orchestration guidance from `assets/bootstrap/` and synced the materialized
`.agents/skills/` output. Validation passed for targeted `rg` checks,
bootstrap source/output diff, `harness plan lint`, `go test
./internal/reviewdimensions`, and `git diff --check`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 performs integrated validation before finalize
review, and Step 2 was the final implementation/documentation alignment step.

### Step 3: Validate and prepare for review

- Done: [x]

#### Objective

Confirm the candidate is internally consistent, tested, and bounded to #272
before final review and archive.

#### Details

Validate that the built-in dimension catalog works, documentation and managed
skill guidance agree, and the preview boundary from #270 remains intact. Do
not add status, lint, examples, or automatic review behavior while closing this
slice.

If implementation produces concrete follow-up handoff notes, leave concise
comments on the relevant sibling issues before archive. Expected candidates:

- #273 for future status hints that suggest `hypothesis-challenge` at
  checkpoint boundaries.
- #274 for lint boundaries that may inspect structure but must not judge
  evidence quality or hypothesis strength.
- #275 for examples that should show challenge output landing in a checkpoint
  report rather than in a review aggregate.

#### Expected Files

- `docs/plans/active/2026-07-05-add-evidence-validity-hypothesis-challenge-guidance.md`
- Any files changed in Steps 1 and 2

#### Validation

- Run `harness plan lint docs/plans/active/2026-07-05-add-evidence-validity-hypothesis-challenge-guidance.md`.
- Run `go test ./internal/reviewdimensions`.
- Run broader Go validation if the catalog change warrants it.
- Run `git diff --check`.
- Run targeted `rg` checks for the accepted vocabulary and issue boundaries.
- If follow-up issue comments are needed, record the comment URLs or note that
  no concrete external handoff was created.
- Record any blocked checks exactly.

#### Execution Notes

Completed integrated validation and follow-up handoff. Passed `harness plan
lint docs/plans/active/2026-07-05-add-evidence-validity-hypothesis-challenge-guidance.md`,
`go test ./internal/reviewdimensions ./internal/cli`, `go test $(go list ./...
| grep -v '/tests/release$')`, `git diff --check`, CLI smoke checks for
`harness review dimensions list` and `harness review dimensions instructions
evidence-validity`, targeted `rg` checks, and bootstrap source/output diff.
Left follow-up handoff comments on #273, #274, and #275:
https://github.com/catu-ai/easyharness/issues/273#issuecomment-4886073639,
https://github.com/catu-ai/easyharness/issues/274#issuecomment-4886073644,
and https://github.com/catu-ai/easyharness/issues/275#issuecomment-4886073658.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only recorded validation and follow-up
handoff after Steps 1 and 2 completed; the integrated candidate proceeds to
finalize review.

## Validation Strategy

- Lint this active plan before approval and again before archive.
- Use focused unit tests for the built-in review dimension catalog.
- Use CLI smoke checks for `harness review dimensions list` and
  `instructions evidence-validity`.
- Use targeted documentation searches for evidence/challenge vocabulary and
  sibling issue boundaries.
- Run `git diff --check`.
- Run broader repository validation if implementation touches shared behavior
  beyond the review dimension catalog and guidance.

## Risks

- Risk: The new dimension is treated as mandatory for every review.
  - Mitigation: Keep catalog and orchestration wording explicit that
    controllers choose dimensions deliberately.
- Risk: Challenge becomes confused with formal review.
  - Mitigation: Define `hypothesis-challenge` as a checkpoint advisory action
    rather than a review dimension, and preserve formal review as the hard
    gate.
- Risk: The guidance drifts into a narrow research-only posture.
  - Mitigation: Use conclusion, evidence, scorecard, probe, decision, and
    synthesis language instead of academic research framing.
- Risk: This slice accidentally absorbs neighboring v0.6.0 issues.
  - Mitigation: Keep status, lint, docs/examples, and plan-scoped dimensions
    deferred explicitly.

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
