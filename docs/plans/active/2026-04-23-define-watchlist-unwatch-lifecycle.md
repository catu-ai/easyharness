---
template_version: 0.2.0
created_at: "2026-04-23T09:36:00+08:00"
approved_at: "2026-04-23T09:38:14+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/166
size: S
---

# Define Watchlist Unwatch Lifecycle

## Goal

Close the completed-versus-hidden lifecycle gap for watched worktrees by
settling on explicit `unwatch` terminology and behavior. A watched workspace
may be classified as `completed` by the dashboard read model, but clearing it
from the dashboard-owned watched set should be an explicit watchlist
membership-removal action, not a hidden dashboard state and not a harness
workflow archive.

This slice should leave future dashboard UI and API work with one durable
contract: `completed` is derived from harness status, while `unwatch` removes a
workspace from the machine-local watchlist without mutating harness workflow
state.

## Scope

### In Scope

- Tighten the watchlist and dashboard specs so #166 consistently uses
  `unwatch` rather than `hidden`, `hide`, or dashboard-local `archive`.
- Confirm that the dashboard `completed` rule remains the current read-model
  rule: readable status, `current_node: "idle"`, and last-landed context.
- Define `unwatch` as an explicit user action that removes a workspace record
  from `watchlist.json`.
- State that `unwatch` is dashboard/watchlist membership behavior only; it
  must not run `harness archive`, alter tracked plans, delete worktrees, or
  change workflow state.
- Add or adjust focused tests if existing docs or contracts are backed by
  generated schema, contract sync, or dashboard/watchlist behavior assertions.
- Leave a GitHub-visible closeout note for #166 explaining the accepted
  terminology and any deferred implementation surface.

### Out of Scope

- Building the dashboard frontend UI.
- Adding the `harness dashboard` CLI entrypoint.
- Implementing a full workspace detail route.
- Automatically removing completed, stale, missing, or invalid workspaces.
- Introducing a persisted `hidden` flag or any separate hidden lifecycle
  state.
- Changing harness workflow semantics, including `harness archive`.
- Deleting local repositories, git worktrees, plan files, or `.local/harness`
  artifacts.

## Acceptance Criteria

- [x] The tracked specs consistently describe `completed` as a derived
      dashboard lifecycle state rather than persisted watchlist state.
- [x] The tracked specs define `unwatch` as explicit watchlist membership
      removal from `watchlist.json`.
- [x] The tracked specs clearly state that `unwatch` does not call or mean
      `harness archive`.
- [x] The tracked specs state that completed workspaces do not automatically
      age out or disappear in v1.
- [x] Any existing references to dashboard-local hide/archive semantics are
      either removed or reframed as historical/deferred wording.
- [x] If implementation work is needed in this slice, focused watchlist or
      dashboard tests prove the new behavior without changing workflow state.
- [x] Issue #166 has a closeout comment or PR note summarizing the accepted
      `unwatch` direction.

## Deferred Items

- Wire the new watchlist-level `Service.Unwatch` method into the future
  dashboard/API/UI surface, tracked by
  [#167](https://github.com/catu-ai/easyharness/issues/167).
- Add the frontend control that lets a user unwatch a completed, missing, or
  invalid workspace from the dashboard home or degraded workspace page.

## Work Breakdown

### Step 1: Tighten the completed and unwatch contract

- Done: [x]

#### Objective

Update the durable watchlist/dashboard documentation so a future agent can
understand #166 without reading discovery chat.

#### Details

Keep the existing completed classification rule from the dashboard read model:
`completed` is derived when status is readable, `current_node` is `idle`, and
status artifacts include last-landed context. Replace any remaining dashboard
`hidden`, `hide`, or `archive` language with `unwatch` unless the wording is
explicitly describing a rejected alternative.

`Unwatch` should be specified as membership removal from the machine-local
watchlist. It should not become another lifecycle enum value, because removed
items are no longer watched entries in the dashboard read model.

#### Expected Files

- `docs/specs/watchlist-contract.md`
- `docs/specs/proposals/harness-ui-steering-surface.md`
- `docs/specs/index.md` if the specs index needs a wording update

#### Validation

- `git diff --check -- docs/specs/watchlist-contract.md docs/specs/proposals/harness-ui-steering-surface.md docs/specs/index.md`
- Reread the updated docs and confirm the terms `hidden` and dashboard-local
  `archive` are not used as current behavior for watched workspace clearing.

#### Execution Notes

Updated the watchlist contract and dashboard steering proposal so `completed`
remains a derived dashboard lifecycle state, completed entries stay watched
until explicit `unwatch`, and `unwatch` is defined as watchlist membership
removal only. Validation: `git diff --check -- docs/specs/watchlist-contract.md
docs/specs/proposals/harness-ui-steering-surface.md docs/specs/index.md`;
`rg -n "hidden|hide|dashboard-local archive|age out|age-out|automatic"
docs/specs/watchlist-contract.md
docs/specs/proposals/harness-ui-steering-surface.md`.

#### Review Notes

`review-001-delta` requested one docs-consistency fix: the Missing or
Unreadable Workspaces section still used old deferred membership-removal
wording. Reworded that section to say entries remain until explicit `unwatch`
removes them. `review-002-delta` passed with no findings.

### Step 2: Decide whether this slice needs a tiny unwatch write path

- Done: [x]

#### Objective

Either implement the smallest watchlist-level `unwatch` behavior now or record
the exact follow-up implementation issue if the contract-only closeout is the
right boundary.

#### Details

Prefer implementation only if the existing code shape makes the write path
truly small and isolated: a watchlist service method that removes one canonical
workspace record from `watchlist.json`, preserves unrelated records, uses the
same home resolution and lock/atomic-write discipline as `Touch`, and does not
touch workflow state.

If that expands beyond a small isolated watchlist change, defer it explicitly
and leave #166 as the contract-definition closeout rather than smuggling UI or
CLI work into this plan.

#### Expected Files

- `internal/watchlist/watchlist.go` if implementing
- `internal/watchlist/watchlist_test.go` if implementing
- GitHub issue or PR body note if deferring implementation

#### Validation

- If implementing: `go test ./internal/watchlist -count=1`
- If deferring: a concrete follow-up issue or PR note names the intended
  `unwatch` implementation surface and explains why #166 remains
  contract-only.

#### Execution Notes

Implemented the small isolated watchlist write path in this slice:
`watchlist.Service.Unwatch` removes one selected workspace record from
`watchlist.json`, preserves unrelated records, uses the same home resolution,
lock, and atomic rewrite helpers as `Touch`, supports missing workspace rows
by matching their persisted path, leaves absent records as no-ops without
creating an empty watchlist, and does not touch harness workflow state.
Validation: red `go test ./internal/watchlist -run Unwatch -count=1` failed
before implementation because `Service.Unwatch` did not exist; green
`go test ./internal/watchlist -count=1` passed after implementation and the
no-op refinement.

#### Review Notes

`review-003-delta` requested one tests fix: add `Unwatch` coverage for
`EASYHARNESS_HOME` / configured home resolution. Added
`TestUnwatchUsesEasyharnessHomeOverride`, which proves `Unwatch` removes from
the configured custom watchlist and leaves the default user-home watchlist
untouched. `review-004-delta` passed with no findings.

### Step 3: Close the GitHub-facing handoff

- Done: [x]

#### Objective

Make the accepted `unwatch` decision visible on #166 or in the PR body so the
issue does not depend on hidden chat context.

#### Details

The closeout note should state the final terms plainly:

- `completed` is derived from status returning to `idle` with last-landed
  context.
- completed entries remain watched until explicit user action.
- that action is called `unwatch`.
- `unwatch` removes watchlist membership only and is unrelated to
  `harness archive`.
- v1 has no automatic age-out or garbage collection.

#### Expected Files

- No repository files are required for this step unless the PR body is the
  chosen handoff surface.

#### Validation

- The issue comment or PR note can be read without discovery chat and explains
  why #166 is ready to close or what implementation issue remains.

#### Execution Notes

Posted the GitHub-facing #166 closeout comment documenting the accepted
`unwatch` direction, the derived `completed` rule, the explicit
non-relationship to `harness archive`, the absence of automatic GC, and the
new watchlist-level `Service.Unwatch` write path:
https://github.com/catu-ai/easyharness/issues/166#issuecomment-4301163030

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 only published the already-reviewed decision and
implementation summary to issue #166; it did not change repository behavior or
tracked specs beyond this plan note.

## Validation Strategy

- Lint the tracked plan before approval.
- Run doc whitespace checks for touched markdown specs.
- If a watchlist write path is implemented, run focused watchlist tests and
  include regression cases for preserving unrelated records, idempotent
  removal of absent records, and no workflow-state mutation.
- Before archive, reread the issue and updated specs to confirm #166's
  acceptance criteria are answered by tracked artifacts rather than chat.

## Risks

- Risk: `unwatch` could be confused with workflow archival or worktree
  deletion.
  - Mitigation: Keep the contract wording explicit that `unwatch` only removes
    watchlist membership and does not mutate harness workflow state or files in
    the watched repository.
- Risk: A hidden-state compromise could creep back in as a persisted flag.
  - Mitigation: State that removed items are no longer in the watched set, and
    do not add a `hidden` lifecycle value.
- Risk: The plan could expand into dashboard UI work.
  - Mitigation: Keep UI controls and routing out of scope unless a later
    approved plan takes them on.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-23-define-watchlist-unwatch-lifecycle.md`
  passed after planning, step closeout updates, and final closeout updates.
- `git diff --check -- docs/specs/watchlist-contract.md docs/specs/proposals/harness-ui-steering-surface.md docs/specs/index.md`
  passed for the Step 1 docs contract update.
- `rg -n "membership-removal behavior|hidden state|dashboard-local archive|age out|age-out|automatic"
  docs/specs/watchlist-contract.md docs/specs/proposals/harness-ui-steering-surface.md`
  confirmed the stale membership-removal wording was removed and remaining
  hidden/archive/automatic references are non-goal or rejected-semantics
  wording.
- Red/green watchlist validation was used for Step 2: `go test
  ./internal/watchlist -run Unwatch -count=1` failed before
  `Service.Unwatch` existed, and `go test ./internal/watchlist -count=1`
  passed after implementation and review fixes.
- Final focused validation passed with `go test ./internal/watchlist
  ./internal/dashboard -count=1`; the finalize correctness reviewer also ran
  `go test ./internal/watchlist ./internal/dashboard ./internal/ui -count=1`
  successfully.

## Review Summary

- `review-001-delta` found one docs-consistency issue: a Missing or Unreadable
  Workspaces paragraph still used old deferred membership-removal wording.
- `review-002-delta` passed after that paragraph was changed to explicit
  `unwatch` terminology.
- `review-003-delta` found one tests issue: `Unwatch` lacked
  `EASYHARNESS_HOME` / configured-home coverage.
- `review-004-delta` passed after adding
  `TestUnwatchUsesEasyharnessHomeOverride`.
- `review-005-full` found one archive-readiness issue: final acceptance
  criteria and durable closeout sections were still placeholders. This update
  resolves that closeout gap before the archive retry.

## Archive Summary

- Archived At: pending `harness archive`
- Revision: 1
- PR: not opened yet; publish closeout should create a PR from branch
  `codex/issue-166-unwatch-lifecycle` and include `Closes #166`.
- Ready: Acceptance criteria are satisfied, the watchlist contract and UI
  proposal consistently define `completed` and `unwatch`, the isolated
  watchlist-level `Service.Unwatch` write path is implemented with focused
  tests, issue #166 has a GitHub-visible handoff comment, and the only
  finalize-review finding was this closeout placeholder update.
- Merge Handoff: After archive, commit the tracked plan move, push the branch,
  open the PR, record publish/CI/sync evidence, and stop at merge approval.

## Outcome Summary

### Delivered

- Updated `docs/specs/watchlist-contract.md` so `completed` is clearly a
  derived dashboard lifecycle state, `unwatch` is explicit watchlist
  membership removal, completed/missing/invalid entries remain watched until
  explicit `unwatch`, and v1 has no hidden state, dashboard-local archive
  bucket, or automatic GC.
- Updated `docs/specs/proposals/harness-ui-steering-surface.md` so the
  dashboard proposal uses the same completed/unwatch terminology and states
  that `Unwatch` is not `harness archive`, workflow mutation, or checkout
  deletion.
- Added `watchlist.Service.Unwatch`, reusing watchlist home resolution,
  locking, loading, and atomic-write helpers while preserving unrelated
  records and treating absent records as no-ops.
- Added watchlist tests for removing a selected workspace, preserving
  unrelated records, nested workspace canonicalization, missing watched paths,
  absent-record idempotence, no watchlist creation for absent no-op removal,
  and configured `EASYHARNESS_HOME` resolution.
- Posted the #166 handoff comment:
  https://github.com/catu-ai/easyharness/issues/166#issuecomment-4301163030

### Not Delivered

- Dashboard frontend/API wiring for invoking `Service.Unwatch`; that belongs
  with the dashboard UI follow-up.
- Automatic cleanup, age-out, background monitoring, or garbage collection for
  completed, idle, missing, invalid, or stale watched workspaces.
- Any change to `harness archive`, harness workflow state, checkout deletion,
  tracked plan deletion, or `.local/harness` artifact cleanup semantics.

### Follow-Up Issues

- [#167 Ship a minimal watchlist dashboard UI](https://github.com/catu-ai/easyharness/issues/167)
  should wire a user-facing `Unwatch` control to the new watchlist-level
  removal behavior.
