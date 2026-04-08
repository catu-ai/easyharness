---
template_version: 0.2.0
created_at: "2026-04-08T20:44:26+08:00"
source_type: direct_request
source_refs: []
---

# Clarify explicit step closeout repair semantics

## Goal

Make the `harness review start` contract explicitly describe the already
supported `spec.step` path for step-closeout repair, especially when a later
execution frontier or reopen flow needs to repair an earlier tracked step's
closeout evidence.

This slice is a contract-and-test clarification, not a workflow redesign. The
repository should leave with one clear story about the distinction between
passive earlier-step debt and explicit earlier-step repair: passive warnings
may leave a later frontier stable, but once the controller intentionally starts
an explicit earlier-step repair review, the targeted earlier step becomes the
current review loop until that repair is resolved cleanly.

## Scope

### In Scope

- Update the CLI/spec prose to describe `spec.step` as a formal explicit path
  for step-closeout repair, including reopen-driven repair of earlier steps.
- Document the supported distinction between passive earlier-step closeout debt
  and explicit earlier-step repair, including when status keeps a later
  frontier stable versus when it re-enters the targeted earlier step loop.
- Document the expected behavior when earlier-step closeout repair fails: the
  candidate stays the same overall branch candidate, but the targeted repaired
  step remains current until that earlier-step debt is resolved.
- Add focused automated coverage that locks the current semantics in place for
  explicit earlier-step repair from later-step or finalize contexts.

### Out of Scope

- Changing `review start`, `status`, `reopen`, or aggregate behavior beyond any
  narrow text/test adjustments needed to express the existing semantics.
- Introducing a new CLI flag or alternate advanced-review command shape.
- Reworking the broader review-state model, reviewer orchestration, or archive
  gating rules outside the explicit-step repair clarification.
- Adding compatibility bridges for alternate interpretations of step index or
  workflow rewind semantics.

## Acceptance Criteria

- [x] `docs/specs/cli-contract.md` and any other relevant normative workflow
      prose explicitly describe `spec.step` as a supported explicit path for
      step-closeout repair, including earlier-step repair after later
      progression or reopen.
- [x] The documented workflow semantics make clear when status keeps a later
      frontier stable for passive earlier-step debt and when explicit
      earlier-step repair intentionally re-enters `step-i` as the current
      review loop.
- [x] The documented semantics explain that a failed earlier-step closeout
      review keeps the overall candidate in the same branch scope while
      pinning the repaired step as current until the debt is resolved.
- [x] Focused tests prove that explicit `spec.step=i` review start remains
      available from later-step or finalize contexts and that status guidance
      keeps the current frontier stable while surfacing the earlier-step debt.
- [x] No runtime logic change is required beyond any test fixture or wording
      adjustments needed to lock the existing behavior in place.

## Deferred Items

- Any redesign of how review debt should block or reorder day-to-day work
  within the current frontier beyond the existing status guidance model.
- Any broader cleanup of historical review-manifest warning paths that are not
  necessary to clarify explicit `spec.step` semantics.
- Any future UX refinement that might rename this path or add a dedicated
  command-level affordance for advanced review starts.

## Work Breakdown

### Step 1: Clarify the explicit-step repair contract in specs and workflow prose

- Done: [x]

#### Objective

Update the written contract so a cold reader can understand when and why to
use `spec.step`, what it means for earlier-step closeout repair, and why the
workflow may distinguish passive debt from explicit repair re-entry.

#### Details

The prose should encode the discovery outcomes from this thread directly in the
repository: earlier-step repair is formally supported, passive debt warnings
do not necessarily change the current node, and explicit earlier-step repair
does intentionally re-enter the targeted step loop while its review is in
flight or non-clean. Update only the docs needed to make those semantics
normative and easy to resume from repository context alone.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/state-transitions.md`
- other nearby workflow docs only if they need narrow consistency updates

#### Validation

- The written contract explicitly names the supported `spec.step` repair path.
- The workflow prose clearly distinguishes passive debt reminders from explicit
  earlier-step repair re-entry.
- Any doc updates remain internally consistent with the current implementation
  and status semantics that will be locked by Step 2 tests.

#### Execution Notes

Updated `docs/specs/cli-contract.md`, `docs/specs/state-model.md`, and
`docs/specs/state-transitions.md` so the repository now says explicitly that
`spec.step` is a supported explicit step-closeout repair path, and clarifies
the real implementation split between passive earlier-step debt warnings and
explicit repair rounds that re-enter the targeted step loop.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is a normative prose clarification only. It
does not change runtime behavior independently of the regression coverage added
in Step 2.

### Step 2: Lock the clarified semantics with focused regression tests

- Done: [x]

#### Objective

Add or update focused tests so the explicit-step repair semantics cannot drift
without deliberate review.

#### Details

Tests should prove the behavior we are documenting rather than inventing a new
one. Cover the important supported cases: explicit earlier-step review start
from a later execution position, explicit earlier-step repair when finalize
context exists, passive debt reminders that keep a later frontier stable, and
explicit repair failures that re-enter the targeted earlier step loop. Prefer
the smallest focused package coverage that directly exercises these semantics.

#### Expected Files

- `internal/review/service_test.go`
- `internal/status/service_test.go`
- `internal/cli/app_test.go` only if CLI-surface coverage is the clearest place
  to lock an explicit-step contract detail

#### Validation

- Focused tests cover explicit `spec.step` use outside the ordinary current
  step path.
- Focused tests cover both frontier-stable passive debt behavior and explicit
  repair-loop re-entry behavior for earlier-step closeout debt.
- The relevant `go test` targets pass cleanly.

#### Execution Notes

Added focused regression coverage in `internal/review/service_test.go` for
explicit `spec.step` review start from a later execution frontier and from
finalize context, plus `internal/status/service_test.go` coverage proving that
a reopened `step-3` frontier can stay stable for passive earlier-step debt, and
that an explicit earlier-step repair review with findings re-enters the
targeted repaired step loop instead of pretending the debt is only passive.
Validation:
`go test ./internal/review ./internal/status`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only adds regression coverage for existing
behavior and does not change production logic.

## Validation Strategy

- Run focused Go tests for the review and status packages, plus CLI tests if
  command-surface coverage changes.
- Re-read the updated spec prose against the implementation to make sure the
  repository describes the semantics the code actually enforces today.
- Use review to confirm the new prose does not accidentally imply workflow
  rewind or a new unsupported advanced-review shape.

## Risks

- Risk: The prose could overstate the supported behavior and imply a broader
  advanced-review feature than the implementation actually provides.
  - Mitigation: keep the wording tightly scoped to explicit step-closeout
    repair and validate it against focused tests and current code paths.
- Risk: Tests could accidentally encode a workflow redesign instead of locking
  today's intended behavior.
  - Mitigation: write the acceptance criteria around existing semantics and
    avoid changing runtime logic in this slice.
- Risk: Different docs could use inconsistent language for repair debt versus
  explicit repair re-entry.
  - Mitigation: update the minimal normative doc set together in one step and
    cross-check terms against implementation-backed tests before approval.

## Validation Summary

- Revalidated the original focused coverage in `internal/review/service_test.go`
  and `internal/status/service_test.go`, then added end-to-end transition
  coverage in `tests/e2e/coverage_test.go` and
  `tests/e2e/explicit_step_repair_test.go` after CI exposed drift between the
  tracked transition matrix and the canonical test catalog.
- Revalidated the repaired candidate with:
  `go test ./tests/e2e/...`
  and
  `go test ./...`
- Re-ran `harness plan lint docs/plans/active/2026-04-08-clarify-explicit-step-closeout-repair-semantics.md`
  and `harness status` after the reopen-driven finalize-fix loop so the plan
  returned to archive-ready state on revision 2.

## Review Summary

- Finalize review `review-001-full` found one blocking tests gap: the failure
  path for explicit earlier-step repair did not yet prove the repaired step
  re-entered the current loop after a non-pass aggregate.
- Finalize review `review-002-full` found two blocking follow-ups: split the
  explicit repair clean/non-clean transition prose in
  `docs/specs/state-transitions.md`, and add in-flight explicit repair status
  coverage.
- Finalize review `review-003-full` found one blocking tests gap and one
  non-blocking suggestion: prove the fixture really enters finalize-fix before
  explicit repair, and suppress passive-debt warnings while an explicit repair
  is already in flight.
- Finalize review `review-004-full` found two final blocking tests gaps: prove
  the finalize-context start test resolves a real finalize-fix node, and prove
  passive-debt reminder prepending still preserves ordinary later-frontier
  continuation guidance.
- Finalize review `review-005-full` passed cleanly with one non-blocking follow-up
  about missing clean-pass regression coverage for explicit repair fallback to
  the ordinary frontier; that gap was deferred to issue `#113`.
- After publish/CI handoff exposed a separate canonical transition-catalog gap,
  the archived candidate was reopened in `finalize-fix` mode for revision 2,
  the missing E2E coverage was added, and finalize review `review-006-full`
  passed cleanly with no blocking or non-blocking findings.

## Archive Summary

- Archived At: 2026-04-08T21:23:00+08:00
- Revision: 2
- PR: `#114` (`https://github.com/catu-ai/easyharness/pull/114`)
- Ready: The reopened finalize-fix candidate has a passing full finalize review
  (`review-006-full`), focused and end-to-end validation are green, and the
  remaining deferred scope is tracked separately in issue `#113`.
- Merge Handoff: Archive the revision-2 candidate, commit the CI-fix follow-up,
  push the branch update to PR `#114`, and refresh CI/sync evidence so the
  candidate can return to merge-ready handoff.

## Outcome Summary

### Delivered

- Clarified the public `harness review start` contract so `spec.step` is
  documented as the supported explicit path for earlier step-closeout repair.
- Updated the workflow specs to distinguish passive earlier-step debt from
  explicit repair loops and to describe how explicit repair interacts with
  ordinary step and finalize transitions.
- Added focused regression coverage for explicit earlier-step repair start from
  later and finalize contexts, passive-debt reminder behavior, in-flight
  explicit repair re-entry, and failed explicit repair re-entry.
- Added canonical transition-catalog coverage and a built-binary E2E scenario
  for explicit earlier-step repair so CI now enforces the same revision-2
  semantics that the tracked specs describe.

### Not Delivered

- No additional behavior redesign was attempted beyond the documentation and
  regression coverage needed to lock the current semantics in place.

### Follow-Up Issues

- #113 Add clean-pass regression coverage for explicit step repair frontier
  fallback (`https://github.com/catu-ai/easyharness/issues/113`)
