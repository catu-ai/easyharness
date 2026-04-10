---
template_version: 0.2.0
created_at: "2026-03-26T18:21:04+08:00"
source_type: direct_request
source_refs:
    - '#50'
size: XL
---

# Simplify review metadata and infer structural closeout context

## Goal

Reduce the review-spec burden on controller agents by removing agent-authored
structural metadata that harness can already infer from workflow state.

Keep the two important guidance surfaces stable after that reduction:
`harness status` must still warn when a completed step lacks durable
step-closeout coverage, and archive readiness must still require a qualifying
finalize review for the current revision. The new model should also eliminate
the current mismatch where repaired step reviews or finalize reviews can pass
but fail to satisfy closeout because the agent chose a different free-form
`trigger`.

## Scope

### In Scope

- Simplify the review-start contract so controller agents no longer need to
  hand-author structural `trigger`/`target` metadata for ordinary step and
  finalize review.
- Persist internal review structure from workflow state at review-start time,
  including whether the round is step-scoped or finalize-scoped, which step it
  belongs to when applicable, and which revision it belongs to.
- Rework status/archive review satisfaction logic to rely on inferred durable
  review structure instead of free-form structural triggers.
- Remove the legacy agent-authored `trigger`/`target` contract instead of
  carrying compatibility shims for prerelease behavior that we no longer want.
- Add regression coverage for repaired step review, repaired finalize review,
  revision-sensitive finalize requirements, and reopen flows.
- Update CLI/spec/skill guidance so future agents know the reduced review spec
  shape and the new inference rules.

### Out of Scope

- Redesigning reviewer dimensions, reviewer submission payloads, or round ID
  allocation.
- Changing the existing reopen modes or archive/publish/land lifecycle shape.
- Designing a separate advanced review-authoring escape hatch beyond the
  minimal routine contract.

## Acceptance Criteria

- [x] `harness review start` accepts a reduced agent-authored review spec for
      ordinary execution, with structural review identity inferred from the
      current workflow node rather than supplied as free-form `trigger` and
      `target` text.
- [x] Harness persists enough internal review metadata to determine, for every
      round, whether it is step-closeout or finalize review, which step it is
      bound to when applicable, and which revision it belongs to.
- [x] A repaired step review that reruns from the same step still counts as the
      latest step-closeout evidence for that step after it passes; status no
      longer demands an extra closeout round only because the repair round used
      different human-facing labeling.
- [x] Archive readiness and `harness status` both require a passing finalize
      review bound to the current revision, with `revision 1` still requiring
      `full` and later revisions allowing `delta` for narrow finalize-fix
      repairs.
- [x] `harness status` continues to surface the two important guidance layers:
      missing step-closeout debt before later-step/finalize/archive progression,
      and missing or insufficient finalize review before archive.
- [x] Reopen flows stay coherent: both reopen modes advance revision, `new-step`
      still waits for the first new unfinished step, and `finalize-fix`
      continues to require a fresh qualifying finalize review for the new
      revision.
- [x] Focused tests cover the reduced input contract, repaired step review,
      repaired finalize review, and reopen-sensitive review guidance without
      preserving old trigger/target fallback behavior.

## Deferred Items

- Consider whether a later follow-up should rename the remaining human-facing
  review note field from `target` to `summary` after the structural inference
  cutover is stable.
- Consider whether the CLI should eventually expose an explicit advanced escape
  hatch for non-routine review creation outside normal step/finalize nodes.

## Work Breakdown

### Step 1: Shrink the review-start contract

- Done: [x]

#### Objective

Define the reduced review-spec shape and document how harness infers review
structure from workflow state.

#### Details

Update the normative docs, CLI help text, and relevant skills so a future
controller agent only needs to choose review breadth (`delta` or `full`), the
review dimensions, and an optional human-readable summary. Fold the discovery
decisions into tracked docs: structural meaning now comes from current node and
revision, not from a free-form trigger string supplied by the agent.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`
- `internal/cli/app.go`

#### Validation

- The docs and help text make the reduced spec shape unambiguous to a cold
  reader.
- The documented rules still explain how routine step-closeout versus finalize
  review is distinguished after trigger removal from agent input.

#### Execution Notes

Updated the CLI help, specs, and review-orchestration guidance so routine
review input is now `kind + dimensions + optional summary/step`. The docs now
say explicitly that harness infers step-bound versus finalize-bound review from
workflow state instead of accepting agent-authored `trigger`/`target`.

#### Review Notes

`review-001-delta` passed with one minor wording finding about legacy
pre-archive language in CLI help. Updated `internal/cli/app.go`, then reran a
narrow `agent_ux` check in `review-002-delta`, which passed cleanly with no
findings.

### Step 2: Persist inferred review structure and reuse it for closeout

- Done: [x]

#### Objective

Teach review start/status/archive to use inferred structural metadata instead
of free-form structural triggers.

#### Details

Bind each round to internal structural facts at creation time: current
revision, finalize versus step scope, and step index when step-scoped. Update
the step-closeout reminder scan, active review context loading, finalize review
satisfaction checks, and archive readiness checks to rely on these inferred
facts rather than `trigger == "step_closeout"` or `trigger == "pre_archive"`.

#### Expected Files

- `internal/review/service.go`
- `internal/runstate/state.go`
- `internal/status/service.go`
- `internal/lifecycle/service.go`

#### Validation

- A repaired step review pass satisfies step-closeout debt without requiring an
  extra controller-authored closeout round.
- A repaired finalize review after reopen satisfies archive readiness for the
  new revision when its kind is sufficient for that revision.

#### Execution Notes

Simplified `review.Spec` and persisted review metadata to use inferred
`step`/`revision` bindings plus a human-readable `summary`. `status` and
`archive` now key off those durable bindings, which fixes the repaired
step-review case and the revision-aware finalize case without asking agents to
choose structural tags.

#### Review Notes

`review-003-delta` checked the inferred `step`/`revision` binding, active
review recovery, and archive-readiness gating. Both `correctness` and
`risk_scan` slots passed cleanly with no findings.

### Step 3: Lock in guidance with regressions

- Done: [x]

#### Objective

Cover the reduced metadata model with focused tests for warning stability and
revision-aware review satisfaction.

#### Details

Add regression tests for: step review fails then repair review passes; finalize
review fails then finalize-fix repair review passes on a later revision; `new-step`
reopen consumes its pending new-step requirement once the first new unfinished
step exists. Remove the old trigger/target compatibility expectations from the
test suite so prerelease coverage matches the reduced contract.

#### Expected Files

- `internal/status/service_test.go`
- `internal/lifecycle/service_test.go`
- `internal/review/service_test.go`

#### Validation

- The new regressions fail against the old trigger-driven behavior and pass
  with the inferred-structure implementation.
- The warning and next-action text remains specific enough for a future agent
  to recover without discovery chat.

#### Execution Notes

Updated unit tests and e2e helpers to use the reduced review spec, added
coverage for inferred structure in review/lifecycle/status, and removed the
old missing-trigger/target fallback expectations that no longer reflect the
desired prerelease behavior.

#### Review Notes

`review-004-delta` found that shared status fixtures still relied on legacy
`trigger`/`target` shims. `review-005-delta` then found the helper layer was
still backfilling `step`/`revision`. Removed both shim layers, updated the
fixtures to express structural metadata directly, reran focused review/lifecycle/status/e2e
validation, and closed the step with `review-006-delta`, which passed cleanly
with no findings.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Run focused Go tests for review start, status, and lifecycle archive/reopen
  behavior.
- Manually inspect representative `harness status` outputs for:
  - missing earlier step closeout
  - repaired step review that now counts
  - revision 1 finalize requiring `full`
  - revision >1 finalize-fix allowing `delta`

## Risks

- Risk: Over-simplifying the internal model could lose the distinction between
  step review and finalize review during reopen or multi-round repair flows.
  - Mitigation: Persist explicit internal step binding plus revision at review
    creation time and cover reopen/fix flows with regression tests.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-26-simplify-review-metadata-and-inference.md`
  passed after closeout updates.
- `go test ./...` passed before step-closeout orchestration, and the final
  focused regression sweep passed with
  `go test ./internal/review ./internal/lifecycle ./internal/status ./tests/e2e`.
- Manual `harness status` checks confirmed the intended guidance transitions:
  step closeout debt, finalize-review gating, archive blockers, and
  revision-aware finalize readiness.

## Review Summary

- Step closeout review passed for Step 1 after one minor wording cleanup in
  `internal/cli/app.go` (`review-002-delta` was the final clean rerun).
- Step 2 closeout passed cleanly in `review-003-delta`.
- Step 3 closeout required two repair loops to remove the remaining fixture
  compatibility shims, then passed cleanly in `review-006-delta`.
- Finalize review passed cleanly in `review-007-full` across `correctness`,
  `tests`, and `docs_consistency`.

## Archive Summary

- Archived At: 2026-03-26T19:20:28+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate has a passing full finalize review for revision 1, all
  tracked steps are complete, and the remaining work is archive/publish
  handoff.
- Merge Handoff: Archive this plan, commit the tracked changes, push
  `codex/review-metadata-inference`, open or update the PR, then record
  publish/CI/sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

- Removed agent-authored structural `target`/`trigger` inputs from routine
  review start and replaced them with harness-inferred step/finalize bindings.
- Persisted durable review structure as `step` plus `revision`, then rewired
  status and archive gating to use that structure instead of free-form tags.
- Updated CLI/spec/skill guidance and rewrote regression fixtures so the
  reduced review metadata contract is exercised directly in tests and e2e
  helpers.

### Not Delivered

- A follow-up rename of the remaining human-facing review summary field was
  intentionally deferred.
- An explicit advanced review-start escape hatch outside routine step/finalize
  nodes was intentionally deferred.

### Follow-Up Issues

- #52 Rename remaining review summary field consistently across local artifacts
  (`https://github.com/catu-ai/microharness/issues/52`)
- #53 Design an explicit advanced review-start escape hatch outside routine
  step/finalize nodes
  (`https://github.com/catu-ai/microharness/issues/53`)
