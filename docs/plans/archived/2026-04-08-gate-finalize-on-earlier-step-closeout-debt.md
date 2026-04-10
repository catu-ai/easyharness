---
template_version: 0.2.0
created_at: "2026-04-08T22:47:47+08:00"
source_type: direct_request
source_refs: []
size: M
---

# Gate finalize on earlier-step closeout debt

## Goal

Resolve issue #24 by turning missed earlier-step closeout from a reminder-only
signal into a real finalize boundary guard, without discarding the newer
explicit earlier-step repair model. The resulting workflow should still allow
an agent to repair a completed earlier step deliberately, but it should no
longer allow default finalize review or archive to proceed while unresolved
earlier-step closeout debt remains.

This slice should also update the repo-visible wording around step-frontier
semantics so a cold reader no longer assumes `execution/step-k/...` is globally
monotonic. Ordinary frontier progression should remain first-unfinished-step
driven, while explicit earlier-step closeout repair remains the documented
exception.

## Scope

### In Scope

- Clarify the normative workflow/docs so the ordinary step frontier remains
  monotonic, but explicit earlier-step closeout repair is called out as the
  intentional exception.
- Define and implement shared earlier-step closeout-debt detection that command
  gates can rely on consistently.
- Reject default finalize review starts when unresolved earlier-step
  review-complete debt remains.
- Reject archive when unresolved earlier-step review-complete debt remains.
- Preserve the repair path where an explicit step-bound review can be started
  for an earlier completed step to clear the debt.
- Add regression coverage for the new gate behavior and the explicit repair
  escape hatch.

### Out of Scope

- Redefining `execution/finalize/*` so status can never point at finalize before
  the earlier-step debt is repaired.
- Adding a new dedicated retrospective closeout command.
- Changing publish, await-merge, or land semantics beyond whatever archive gate
  fallout is required for correctness.

## Acceptance Criteria

- [x] The specs and tracked wording clearly distinguish the ordinary monotonic
      step frontier from the explicit earlier-step repair exception, so readers
      do not rely on the old global-monotonicity assumption.
- [x] `harness review start` rejects a finalize-bound review when earlier
      completed steps still lack review-complete closeout, and the error points
      the controller toward explicit step repair instead of ambiguous retry
      advice.
- [x] `harness review start` still allows an explicit `step=<i>` repair review
      for an earlier completed step, including from finalize-scope nodes, and a
      later clean repair unblocks the ordinary finalize path.
- [x] `harness archive` rejects unresolved earlier-step closeout debt using the
      same underlying debt rules as the review-start gate.
- [x] Automated tests cover finalize review rejection, explicit earlier-step
      repair allowance, archive rejection, and the post-repair clean path.

## Deferred Items

- Reconsidering whether `harness status` should stop resolving to finalize nodes
  when earlier-step closeout debt exists. This slice will clarify wording and
  enforce command gates, but it will not redesign the derived node model.

## Work Breakdown

### Step 1: Align the workflow contract with explicit earlier-step repair and finalize gates

- Done: [x]

#### Objective

Update the normative docs so they describe the accepted model precisely:
ordinary frontier progression stays first-unfinished-step driven, explicit
earlier-step closeout repair is the intentional exception, and finalize review
plus archive are blocked until earlier-step closeout debt is resolved.

#### Details

The current repo history already introduced explicit earlier-step repair
semantics on `origin/main`. This step should fold that direction into the local
tracked contracts and make the issue-#24 decision explicit instead of leaving a
reminder-only ambiguity behind. The docs should explain that finalize review is
still a distinct branch-level review, but default finalize progression is not
trusted while an earlier completed step still lacks review-complete closeout.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/state-transitions.md`
- `docs/specs/cli-contract.md`

#### Validation

- The updated specs say when explicit earlier-step repair is allowed and when
  default finalize review/archive must reject unresolved earlier-step debt.
- A cold reader can tell that ordinary step-frontier monotonicity has an
  explicit repair exception instead of inferring a global monotonic rule.

#### Execution Notes

Updated the normative wording to match the accepted workflow model from current
`origin/main`: the ordinary step frontier remains first-unfinished-step driven,
explicit earlier-step closeout repair is the intentional exception, and default
finalize review start plus archive must reject unresolved earlier-step
review-complete debt. The transition catalog wording in `tests/e2e` was synced
to the updated state-transition matrix so the spec drift check stays green.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only tightened workflow wording and the
transition-catalog copy around the Step 2 behavior change; the meaningful risk
surface is covered by the Step 2 review.

### Step 2: Enforce finalize review and archive gates from shared debt detection

- Done: [x]

#### Objective

Implement a single source of truth for unresolved earlier-step closeout debt
and use it to block default finalize review starts and archive while preserving
explicit step-bound repair.

#### Details

Do not duplicate the historical step-closeout scan logic independently across
status, review start, and archive. Refactor or extract the needed detection so
review-start and archive decisions use the same debt semantics that status
already understands. The gate should trigger only when the requested review
would otherwise bind to finalize scope; explicit `step=<i>` repair must still
work even from finalize-scope nodes. Archive should fail with a concrete error
that points at the earlier-step debt instead of surfacing only generic archive
readiness failure.

#### Expected Files

- `internal/review/service.go`
- `internal/lifecycle/service.go`
- `internal/status/service.go`
- `internal/status/service_test.go`

#### Validation

- Targeted unit tests prove finalize review start rejects unresolved debt while
  explicit step repair remains available.
- Archive readiness failures identify earlier-step closeout debt distinctly.
- Existing status reminder behavior remains aligned with the shared debt rules.

#### Execution Notes

Extracted the historical earlier-step closeout scan into
`internal/stepcloseout` so status, review-start binding, and archive readiness
can share one debt detector. Default finalize-bound `harness review start` now
rejects unresolved earlier-step review-complete debt and points the controller
toward explicit `spec.step=<i>` repair, while `harness archive` now fails with
a concrete earlier-step closeout error even after a clean finalize review.

#### Review Notes

`review-001-full` found one blocking regression gap (no built-binary E2E proving
default finalize review start and archive reject earlier-step closeout debt)
and one non-blocking fixture-hygiene issue. The follow-up delta round
`review-002-delta` passed clean after adding the built-binary rejection E2E and
tightening the shared archive-ready fixture helpers.

### Step 3: Lock the repaired workflow in with regression coverage

- Done: [x]

#### Objective

Add or refresh regression coverage so the repo can safely evolve without
reintroducing reminder-only finalize behavior or breaking explicit earlier-step
repair.

#### Details

Cover both the blocked path and the escape hatch. The tests should show that a
default finalize review start is rejected when earlier-step debt remains, an
explicit `step=<i>` repair can still run, a clean repair restores the ordinary
finalize path, and archive remains blocked until that repair is complete.
Choose the smallest combination of unit and e2e coverage that proves the
workflow clearly to future readers.

#### Expected Files

- `internal/review/service_test.go`
- `internal/lifecycle/service_test.go`
- `tests/e2e/`

#### Validation

- `go test ./...` passes.
- New tests fail without the gate behavior and pass with the final
  implementation.

#### Execution Notes

Added focused review and lifecycle regressions for the new gates, updated
archive-ready lifecycle fixtures to carry explicit review-complete closeout,
and synced the E2E transition catalog text with the tightened state-transition
matrix. Finalize follow-up also tightened the built-binary explicit-repair E2E
so it clears real earlier-step debt and proves the ordinary finalize review
path reopens after a clean `spec.step=<i>` repair. `go test ./internal/review
./internal/lifecycle ./internal/status` and `go test ./...` both pass.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This regression-coverage slice landed as the direct
repair for Step 2 review findings, and the resulting tests plus fixture
tightening were covered by the clean `review-002-delta` closeout round.

## Validation Strategy

- Start with focused package tests around review binding, archive readiness, and
  status debt detection while the logic is being reshaped.
- Finish with `go test ./...` so the full workflow contract still holds across
  review, archive, reopen, and merge-handoff surfaces.

## Risks

- Risk: The local worktree is behind the latest `origin/main` wording around
  explicit earlier-step repair, so the implementation could accidentally revert
  or contradict the newer semantics.
  - Mitigation: Reconcile the local docs/tests with the `origin/main` explicit
    repair model before tightening the finalize gates.
- Risk: A naive finalize gate could also block the explicit `step=<i>` repair
  path that is supposed to clear the debt.
  - Mitigation: Gate on the resolved review binding, not on “currently in a
    finalize node” alone, and add direct regression coverage for explicit step
    repair from finalize-scope states.
- Risk: Archive readiness and status could drift if they use slightly different
  debt rules.
  - Mitigation: Reuse one shared debt-detection path instead of copying the
    logic.

## Validation Summary

Implemented shared earlier-step closeout-debt detection for status, review
start, and archive readiness, then verified the final candidate with
`go test ./...` plus focused built-binary E2E runs covering blocked finalize
review start, blocked archive, and the repaired clean path after explicit
`spec.step=<i>` closeout repair.

## Review Summary

Step 2 closeout passed after `review-001-full` raised one blocking E2E gap and
one minor fixture-hygiene issue, both repaired in `review-002-delta`.
Finalize review then found one remaining coverage gap in `review-003-full`; the
bounded proof landed clean in `review-004-delta`, and the archive-candidate
signoff finished clean in `review-005-full`.

## Archive Summary

- Archived At: 2026-04-08T23:30:46+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate now satisfies the acceptance criteria and has a clean
  full finalize review before archive.
- Merge Handoff: Archive this active plan, then commit, push, open a draft PR,
  and record publish/CI/sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

Delivered shared finalize-closeout debt gating across review start, status, and
archive, aligned the workflow/spec wording with the explicit earlier-step
repair exception, and locked the behavior in with unit plus built-binary E2E
coverage for both the blocked and repaired paths.

### Not Delivered

NONE.

### Follow-Up Issues

- #121: Consider blocking finalize node resolution when earlier-step closeout
  debt exists.
