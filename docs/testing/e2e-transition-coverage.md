# E2E Transition Coverage

## Purpose

This document reports what the repo-level E2E suite explicitly covers today.
It is not a second workflow spec. The normative workflow contract remains in
`docs/specs/state-transitions.md` and `docs/specs/state-model.md`; the checked
catalog behind this report lives in `tests/e2e/coverage_test.go`.

## Coverage Model

The current matrix has 13 transition families and 9 canonical workflow nodes:

- `idle`
- `plan`
- `execution/step-<n>/implement`
- `execution/finalize/review`
- `execution/finalize/fix`
- `execution/finalize/archive`
- `execution/finalize/publish`
- `execution/finalize/await_merge`
- `land`

The suite uses bounded representative routes instead of pretending that a
loop-capable workflow can be exhaustively enumerated. Its model includes:

- multi-step and single-step plans
- mandatory full finalize review
- one blocking finding and an inferred linked repair delta
- an explicit broad-repair reset using `review start --full`
- publish evidence progression into `execution/finalize/await_merge`
- archive and await-merge reopen paths for both finalize repair and a new step
- merge-approved land completion back to `idle`

## Current Explicit Coverage

The repo-level suite has 8 scenario families:

- `review_workflow`: mandatory full review, blocking findings, inferred linked
  delta, explicit full reset, and clean archive coverage
- `archive_reopen_finalize_fix`: archive handoff followed by finalize repair
- `reopen_new_step`: archive handoff followed by a newly tracked step
- `publish_handoff`: publish evidence progression into await-merge
- `lightweight_workflow`: one-step lightweight execution with the same
  mandatory finalize review and publish gates
- `land_workflow`: await-merge through land completion back to idle
- `await_merge_reopen_finalize_fix`: merge-ready invalidation into repair
- `await_merge_reopen_new_step`: merge-ready invalidation into a new step

Current bounded coverage is therefore:

- scenarios: 8
- canonical nodes: 9 / 9
- transition families: 13 / 13

`tests/e2e/coverage_test.go` enforces both that every canonical transition is
mapped to a scenario and that the catalog stays synchronized with the tracked
transition matrix. The scenarios themselves remain the executable evidence;
the catalog is the durable index connecting them to the contract.

## Covered Transition Families

- `idle_to_plan`
- `plan_to_step_implement`
- `step_implement_to_next_step_implement`
- `step_implement_to_finalize_review`
- `finalize_review_to_finalize_fix`
- `finalize_review_to_finalize_archive`
- `finalize_fix_to_finalize_review`
- `finalize_fix_to_new_step_implement`
- `finalize_archive_to_publish`
- `publish_to_await_merge`
- `archived_to_finalize_fix`
- `await_merge_to_land`
- `land_to_idle`

## Remaining E2E Gaps

There are no remaining gaps at the bounded transition-family level.
`tests/resilience/` covers malformed pointers and degraded artifacts, while
`tests/e2e/runstate_concurrency_test.go` covers deterministic archive, reopen,
evidence, status, and lock-contention interleavings. Property-style coverage
for parsing-heavy paths and unbounded route enumeration remain possible future
work rather than release gates.

## Interpretation

State-preserving implementation, validation, reviewer investigation, Closeout
preparation, evidence refresh, and land bookkeeping do not appear as separate
transitions. In particular, step review and controller aggregate are no longer
workflow nodes or commands.
