---
template_version: 0.2.0
created_at: "2026-03-22T00:00:00+08:00"
source_type: issue
source_refs:
    - '#6'
---

# Add repo-level smoke and review workflow tests

## Goal

Introduce the first repo-level Go test structure under `tests/` so
`superharness` can exercise the real built `harness` binary in smoke and
multi-command workflow scenarios without relying on whichever binary happens to
be on `PATH`.

This slice should prove the proposal in
`docs/specs/proposals/testing-structure.md` is workable without expanding into
a full testing-architecture refactor. The new support helpers should stay
scoped to repo-level suites only, while the first golden end-to-end scenario
covers the review workflow from plan creation through review aggregation.

## Scope

### In Scope

- Add a repo-level `tests/support/` package for binary build, temporary
  workspace setup, command execution, and stable assertion helpers used by
  top-level suites only.
- Add `tests/smoke/` coverage that runs the built `harness` binary for
  `harness --help`, `harness status`, and a minimal
  `plan template -> plan lint` roundtrip.
- Add `tests/e2e/` coverage for the first golden workflow:
  `plan template -> execute start -> review start -> review submit -> review aggregate`.
- Keep the built binary under test aligned with the working tree by compiling
  `./cmd/harness` into a temporary path instead of resolving `harness` from
  `PATH`.
- Record the deferred follow-up coverage that remains intentionally out of
  scope for this first repo-level slice.

### Out of Scope

- Refactoring existing `internal/*` package-local tests to reuse
  `tests/support/` or otherwise changing their helper structure in this slice.
- Adding `tests/resilience/` or broader repo-level suites beyond the first
  smoke package and one review-workflow E2E path.
- Covering lifecycle-heavy flows such as
  `archive -> status -> reopen -> status` or landed-state reporting.
- Adding fuzz tests, wrapper scripts, build tags, or CI wiring changes beyond
  what is needed to run the new Go test packages locally.
- Changing the production CLI contract except for behavior-preserving
  testability adjustments that are strictly required by the new repo-level
  suites.

## Acceptance Criteria

- [x] `tests/support/` exists as a repo-level helper package, is used by the
      new top-level suites, and keeps binary building plus workspace/command
      setup out of the smoke and E2E test bodies.
- [x] `go test ./tests/smoke -count=1` passes with real-binary coverage for
      `harness --help`, `harness status`, and a minimal
      `plan template -> plan lint` roundtrip without depending on `PATH`.
- [x] `go test ./tests/e2e -count=1` passes with one golden review-workflow
      scenario that asserts the key command outputs plus durable review/state
      artifacts after start, submit, and aggregate.
- [x] Existing package-local suites continue to pass without being migrated to
      the new support package, and the plan records explicit deferred follow-up
      coverage for lifecycle, resilience, and fuzzing work.

## Deferred Items

- Add more repo-level E2E coverage for lifecycle-heavy flows such as
  `archive -> status -> reopen -> status` and landed-state reporting after
  `harness land --pr ...` plus `harness land complete`.
- Add `tests/resilience/` with deterministic failure-injection cases such as
  corrupted `.local/harness/current-plan.json`, missing review artifacts, and
  archive rollback failures.
- Evaluate fuzz coverage for parsing-heavy paths such as plan linting and
  review artifact parsing once the repo-level baseline is stable.
- Revisit whether repo-level suites should stay as opt-in package paths or
  join the default `go test ./...` developer path with explicit expectations
  for runtime and caching.
- Revisit package-local helper deduplication separately if `internal/*` test
  repetition becomes expensive enough to justify a dedicated test utility
  layer.

## Work Breakdown

### Step 1: Add repo-level test support helpers

- Done: [x]

#### Objective

Create the first `tests/support/` helpers so repo-level suites can build the
working-tree `harness` binary, create temporary harness workspaces, run
commands, and assert stable results without duplicating shell/process setup in
every test.

#### Details

Keep the support surface intentionally narrow and repo-level in meaning. The
helpers should work for `tests/smoke` and `tests/e2e`, but they should not
become a dumping ground for package-local unit-test helpers or production
dependencies. Prefer generated temporary workspaces over large checked-in
snapshots for this first slice, and make the binary under test come from the
current working tree rather than `PATH`.

#### Expected Files

- `tests/support/binary.go`
- `tests/support/repo.go`
- `tests/support/run.go`
- `tests/support/assert.go`
- `tests/support/plan.go`

#### Validation

- `tests/smoke` and `tests/e2e` can share one helper package without hiding
  important assertions behind opaque abstractions.
- The helper package can build `./cmd/harness`, execute commands against a
  temporary repository, and return stdout, stderr, and exit status in a form
  the suite tests can assert on directly.

#### Execution Notes

Added `tests/support/` as a narrow repo-level helper package for top-level
binary-driven suites only. `binary.go` builds the working-tree `./cmd/harness`
binary into a temporary path, `repo.go` creates temporary workspaces and JSON
fixtures, `run.go` executes commands and captures stdout/stderr/exit status,
`assert.go` provides JSON/file/assertion helpers, and `plan.go` rewrites
generated tracked plans into deterministic repo-level fixtures for workflow
coverage. Validated the helpers by running
`go test ./tests/smoke ./tests/e2e -count=1` and `go test ./...`.

#### Review Notes

`review-011-delta` caught a documentation-only contradiction in the prior step
note, and `review-012-delta` then passed clean for this helper slice. Step 1
now has recorded step-closeout review history that matches the repo-level
workflow contract before archive.

### Step 2: Add real-binary smoke coverage

- Done: [x]

#### Objective

Add a small smoke suite that proves the built `harness` binary starts, reports
help text, handles status in a temporary repo, and can complete a minimal plan
template/lint roundtrip.

#### Details

Keep the smoke suite intentionally small and fast. It should assert stable,
user-visible behavior that is worth guarding at the repository level without
duplicating the package-local CLI tests. The goal is to establish the repo
test structure and binary-execution harness, not to rebuild exhaustive CLI
coverage at a slower layer.

#### Expected Files

- `tests/smoke/smoke_test.go`
- `tests/support/binary.go`
- `tests/support/repo.go`
- `tests/support/run.go`
- `tests/support/assert.go`

#### Validation

- `go test ./tests/smoke -count=1` passes and each smoke case runs the real
  built binary instead of calling internal Go entrypoints directly.
- Smoke cases cover `harness --help`, `harness status`, and
  `plan template -> plan lint` with assertions on stable outputs and exit
  status.

#### Execution Notes

Added `tests/smoke/smoke_test.go` with real-binary coverage for top-level help
output, idle `harness status` behavior in a temporary workspace, and a minimal
`plan template -> plan lint` roundtrip. The smoke cases use `tests/support/`
instead of package-local CLI entrypoints, assert stable command behavior, and
avoid rebuilding a second assertion layer in shell scripts.

#### Review Notes

`review-013-delta` found that the idle `harness status` smoke case did not pin
the stable handoff summary/guidance contract, and `review-014-delta` then found
that root help coverage still under-specified the command surface. After
tightening both smoke assertions, `review-015-delta` confirmed one more root
help command-surface gap. `review-016-delta` then passed clean, so this smoke
slice now has recorded step-closeout review history that matches its repo-level
coverage claims before archive.

### Step 3: Add the first golden review-workflow E2E

- Done: [x]

#### Objective

Prove the repo-level structure can drive one realistic multi-command workflow
by covering `plan template -> execute start -> review start -> review submit -> review aggregate`.

#### Details

Use the simplest valid review scenario that still exercises the command-owned
artifacts and state transitions: generate a plan in a temporary repo, start
execution, start a review round with a minimal valid spec, submit the required
review slot payload, and aggregate the round. Assert the durable artifacts that
matter for future refactors, such as review manifests, submissions, aggregate
results, and state pointers, while avoiding overexpansion into archive/reopen
or resilience behavior in this same slice.

#### Expected Files

- `tests/e2e/review_workflow_test.go`
- `tests/support/binary.go`
- `tests/support/repo.go`
- `tests/support/run.go`
- `tests/support/assert.go`

#### Validation

- `go test ./tests/e2e -count=1` passes with one golden review-workflow test
  that uses the real built binary for every command in the flow.
- The E2E assertions cover the generated plan path, execution start behavior,
  review round creation, review submission persistence, aggregate output, and
  the local artifacts/state that prove the workflow actually completed.

#### Execution Notes

Added `tests/e2e/review_workflow_test.go` to drive the first golden workflow
through the real built binary:
`plan template -> execute start -> review start -> review submit -> review aggregate`.
The test generates a temporary plan, writes JSON inputs for review start and
submit, uses `tests/support/plan.go` to shape deterministic tracked-plan
fixtures, and asserts the resulting manifest, ledger, submission, aggregate,
and local `state.json` artifacts so future workflow refactors keep the durable
review contract intact.

#### Review Notes

Finalize full reviews previously forced more faithful step-review closeout,
multi-slot aggregate gating assertions, deterministic fixture shaping, and
finalize-review state persistence checks for this E2E slice. `review-017-delta`
then exposed two more state-machine gaps around stale step-review metadata and
failed finalize-aggregate persistence. After fixing those behaviors and
assertions, `review-018-delta` passed clean, so this E2E slice now has recorded
step-closeout review history that matches its workflow contract before archive.

## Validation Strategy

- Run `harness plan lint` on this tracked plan before execution starts and
  whenever scope wording changes.
- During implementation, keep the repo-level suites individually runnable with
  `go test ./tests/smoke -count=1` and `go test ./tests/e2e -count=1`.
- Before archive, run `go test ./...` to confirm the new top-level packages do
  not regress the existing package-local test suite.

## Risks

- Risk: The new repo-level helpers could grow into a second general-purpose
  test framework or start overlapping with package-local test utilities.
  - Mitigation: Keep `tests/support/` narrowly focused on real-binary repo
    suites and leave `internal/*` helper deduplication deferred to separate
    follow-up work.
- Risk: Repo-level tests could become slow or flaky if they over-assert help
  text formatting or rebuild the binary unnecessarily for every command.
  - Mitigation: Keep smoke coverage intentionally small, centralize binary
    build/caching logic in support helpers, and assert durable contract fields
    or artifacts rather than incidental formatting.

## Validation Summary

- `go test ./tests/smoke -count=1`
- `go test ./tests/e2e -count=1`
- `go test ./internal/status -count=1`
- `go test ./...`

## Review Summary

- Step 1 helper slice: `review-011-delta` caught a stale closeout note, and
  `review-012-delta` passed after the step history was corrected.
- Step 2 smoke slice: `review-013-delta`, `review-014-delta`, and
  `review-015-delta` progressively tightened idle-status and root-help
  coverage; `review-016-delta` passed clean.
- Step 3 E2E slice: repeated finalize full reviews exposed workflow-state
  gaps, `review-017-delta` isolated the stale step-review/finalize persistence
  issues, and `review-018-delta` passed after those fixes landed.
- Full-candidate review: repeated full rounds through `review-020-full` drove
  the remaining archive-candidate assertion fixes, and `review-021-full`
  passed as the structural `pre_archive` gate with one non-blocking handoff
  note about documenting `tests/support/plan.go` more explicitly.

## Archive Summary

- Archived At: 2026-03-22T13:24:54+08:00
- Revision: 1
- PR: not created yet; publish evidence will record the PR URL after archive.
- Ready: structural `pre_archive` review passed clean; remaining work is the
  archive move plus publish/CI/sync evidence for merge readiness.
- Merge Handoff: after archive, open the PR, record publish/CI/sync evidence,
  and keep deferred testing scope linked through `#6` and `#22`.

## Outcome Summary

### Delivered

- Added repo-level `tests/support/` helpers that build the working-tree
  `harness` binary, create temporary workspaces, run commands, and provide
  reusable assertion helpers for top-level suites.
- Added `tests/smoke/` coverage for root help, idle `harness status`, default
  stdout `plan template`, and `plan template --output -> plan lint`.
- Added a golden `tests/e2e/` review-workflow test that exercises step-level
  delta review, multi-slot finalize review, missing-submission aggregate
  failure, manifest/submission/ledger persistence, and finalize-review status
  behavior with the real built binary.
- Fixed `harness status` so stale step-closeout review metadata no longer leaks
  into finalize nodes after a clean step review.

### Not Delivered

- Additional repo-level lifecycle E2E coverage such as
  `archive -> status -> reopen -> status` or land-state flows.
- `tests/resilience/` deterministic failure-injection coverage.
- Fuzz coverage for parsing-heavy plan/review paths.
- Any refactor of existing `internal/*` package-local test helpers.

### Follow-Up Issues

- `#6` Track the remaining repo-level testing expansion, including lifecycle,
  resilience, fuzzing, and other broader integration coverage.
- `#22` Clarify automatic review progression and closeout expectations in
  `AGENTS.md` and the harness execute skill.
