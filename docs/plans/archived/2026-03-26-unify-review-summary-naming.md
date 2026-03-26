---
template_version: 0.2.0
created_at: "2026-03-26T00:00:00Z"
source_type: issue
source_refs:
    - '#52'
---

# Unify human-facing review title naming

## Goal

Remove the remaining mixed naming for the human-readable review text so a cold
reader no longer has to translate between `summary`, `review_target`, and
older "target" wording.

Choose one stable human-facing name for that field and apply it consistently
across status facts, local review artifacts, docs, and regression coverage
without reintroducing structural ambiguity.

## Scope

### In Scope

- Pick the canonical user-facing name for the human-readable review title.
- Rename status facts and any related helper/test contracts to that canonical
  name.
- Align local review artifact structs, CLI/spec docs, skills, and tests with
  the chosen name where they expose the human-facing field.
- Update regression coverage so the renamed field is exercised end to end.

### Out of Scope

- Reworking the structural review model introduced in issue #50 follow-up.
- Adding new advanced review-start escape hatches outside routine
  step/finalize flows.
- Changing reviewer submission `summary`, which is already a distinct concept.

## Acceptance Criteria

- [x] One canonical name is used for the human-facing review title across
      status facts, local review artifacts, docs, and tests.
- [x] `harness status` no longer exposes the mixed `review_target` naming when
      referring to the human-readable review title.
- [x] The renamed field stays clearly separated from structural review facts
      like step binding, revision, and derived review trigger.
- [x] Unit and e2e coverage are updated to assert the renamed field directly,
      without leaving mixed-name compatibility shims in shared helpers.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Choose and document the canonical name

- Done: [x]

#### Objective

Decide the stable human-facing field name and fold that decision into the
tracked docs and plan context before code changes begin.

#### Details

This step should answer one key design question cleanly: whether the
human-readable review text should standardize on `review_title` or another
explicit name. The decision should optimize for clarity against
reviewer-submission summaries and avoid reviving the older `target`
terminology.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/plans/active/2026-03-26-unify-review-summary-naming.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`

#### Validation

- The plan and specs state `review_title` as the canonical name and explain
  why it is distinct
  from structural review metadata and reviewer submission summaries.
- A future execution agent could implement the rename from the tracked plan
  alone.

#### Execution Notes

Settled the canonical human-facing field name on `review_title` after
clarifying that the field is a review title, not a structural step binding and
not a generic summary. Updated the tracked plan language, CLI contract, and
review orchestration references to use that name explicitly and to keep it
separate from command/status summaries and reviewer submission summaries.

This slice changed tracked docs and review-related contracts rather than
introducing a new algorithm, so focused contract regression coverage was more
appropriate than strict Red/Green/Refactor TDD. Validation so far:
`harness plan lint docs/plans/active/2026-03-26-unify-review-summary-naming.md`
and focused Go coverage after the runtime rename landed.

#### Review Notes

Clean delta review `review-001-delta` passed on the `docs_consistency` slot
with no findings. The reviewer confirmed that `review_title` is used
consistently for the human-facing review title and remains distinct from
command/status summaries and reviewer submission `summary`.

### Step 2: Apply the rename through status and artifacts

- Done: [x]

#### Objective

Rename the runtime/user-facing field everywhere it materially surfaces.

#### Details

Touch the narrowest set of runtime structs and outputs that expose the
human-readable review title. Keep structural fields (`step`, `revision`,
derived trigger) unchanged, but make the human-facing review title use the new
name consistently in status facts and local review artifact readers/writers.

#### Expected Files

- `internal/status/service.go`
- `internal/review/service.go`
- `internal/runstate/state.go`
- `internal/cli/app.go`

#### Validation

- `harness status` and the affected local artifact structs expose
  `review_title` consistently.
- The rename does not break step/finalize gating or reopen/archive behavior.

#### Execution Notes

Renamed the runtime-facing review field from mixed `summary`/`review_target`
wording to `review_title` across review specs, manifests, aggregates,
runstate helpers, status facts, and CLI help text. Structural review fields
(`step`, `revision`, `kind`, derived trigger) stayed unchanged; only the
human-facing review title contract moved.

Because the direct `harness` command had been rebuilt from earlier code, this
step also reran `scripts/install-dev-harness` before relying on live CLI
output. Validation after the rename: `go test ./internal/review
./internal/runstate ./internal/status ./internal/cli ./tests/e2e/...` and
`harness status`, which now exposes `facts.review_title` on the active review
path.

#### Review Notes

Clean delta review `review-002-delta` passed on the `correctness` slot with
no findings. The reviewer confirmed that `review_title` now flows through
review specs, manifests, aggregates, runstate readers, status facts, and CLI
help without regressing step/finalize gating.

### Step 3: Update regression coverage and fixture contracts

- Done: [x]

#### Objective

Make the tests speak `review_title` directly and remove any mixed-name test
contracts that would undermine the rename.

#### Details

Update unit tests, shared e2e helpers, and any coverage tables or assertions
that still refer to the old name. Prefer direct fixtures over compatibility
translation in shared helpers so the regression suite proves the new naming
end to end.

#### Expected Files

- `internal/status/service_test.go`
- `internal/status/service_internal_test.go`
- `tests/e2e/helpers_test.go`
- `tests/e2e/review_workflow_test.go`

#### Validation

- Focused unit and e2e coverage passes with the renamed field.
- Shared helpers no longer preserve the old mixed-name contract where
  `review_title` should be expressed directly.

#### Execution Notes

Updated the shared status/e2e fixture contracts to speak `review_title`
directly, including status facts, persisted review manifests/aggregates, and
end-to-end assertions that exercise step review, finalize review, and repair
loops. Also refreshed the reviewer-facing docs so controller/reviewer prompts
now talk about review titles instead of review targets when they refer to the
human-facing review label.

Validation for this slice relied on the focused suite named in the plan:
`go test ./internal/review ./internal/runstate ./internal/status ./internal/cli
./tests/e2e/...`. Those tests now pass against the renamed fixture and output
contracts without any compatibility translation layer in the shared helpers.

#### Review Notes

Clean delta review `review-003-delta` passed on the `tests` slot with no
findings. The reviewer confirmed that the shared helpers, unit fixtures, and
e2e assertions now use `review_title` directly and do not leave an old-name
compatibility shim behind.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Run focused Go tests for status, review, and e2e review workflow coverage.
- Inspect representative `harness status` output to confirm the human-facing
  review title uses `review_title` consistently.

## Risks

- Risk: A naive rename could blur the difference between review-round summary
  and reviewer-submission summary.
  - Mitigation: Document that distinction explicitly in the plan and specs
    before touching runtime output, and keep structural review facts unchanged.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-26-unify-review-summary-naming.md`
  passed after the `review_title` contract was finalized in the tracked plan.
- `go test ./internal/review ./internal/runstate ./internal/status
  ./internal/cli ./tests/e2e/...` passed after the runtime/docs/test rename
  landed.
- Live `harness status` output during step and finalize review now exposes
  `facts.review_title`, confirming the direct CLI contract matches the updated
  unit and e2e coverage.

## Review Summary

- Step closeout review `review-001-delta` passed on `docs_consistency` with no
  findings.
- Step closeout review `review-002-delta` passed on `correctness` with no
  findings.
- Step closeout review `review-003-delta` passed on `tests` with no findings.
- Finalize full review `review-004-full` passed on `correctness` and
  `docs_consistency` with no findings.

## Archive Summary

- Archived At: 2026-03-26T21:22:33+08:00
- Revision: 1
- PR: NONE. Publish evidence should record the PR URL after archive.
- Ready: `review-004-full` passed as the structural `pre_archive` gate, all
  tracked steps are complete, the acceptance criteria are satisfied, and the
  candidate now leaves the repository with one canonical human-facing review
  title: `review_title`.
- Merge Handoff: Run `harness archive`, commit the archive move plus the
  tracked runtime/docs/test updates, push the branch, open the PR, then record
  publish, CI, and sync evidence until the candidate reaches merge approval.

## Outcome Summary

### Delivered

- Renamed the review-round human-facing field to `review_title` across review
  specs, manifests, aggregates, runstate helpers, status facts, and CLI help.
- Updated the tracked CLI contract and review/reviewer guidance so they all
  describe the same `review_title` concept and keep it distinct from command
  or submission summaries.
- Refreshed shared fixtures and regression coverage so unit tests and e2e
  helpers assert `review_title` directly with no compatibility shim.

### Not Delivered

- No advanced escape hatch for non-routine review starts was added here; that
  remains deferred to issue `#53`.

### Follow-Up Issues

- #53: Design an explicit advanced review-start escape hatch outside routine
  step/finalize nodes.
