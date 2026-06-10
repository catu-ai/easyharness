---
template_version: 0.2.0
created_at: "2026-06-10T23:38:23+08:00"
approved_at: "2026-06-10T23:40:39+08:00"
source_type: direct_request
source_refs: []
size: S
---

# Prioritize Active Dashboard Workspaces And Cleanup Degraded Watches

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Make the dashboard home useful when the machine-local watchlist contains many
recent temporary or deleted worktrees. The default home ordering should put
live actionable work first by dashboard lifecycle state, while preserving
recency order inside each lifecycle group.

Add one explicit cleanup affordance for degraded watchlist entries:
`Unwatch missing/invalid`. This remains a user-triggered watchlist membership
write, not automatic garbage collection, and it must not mutate repositories,
git worktrees, tracked plans, or local harness workflow state.

## Scope

### In Scope

- Change dashboard home ordering from global `last_seen_at` recency to
  lifecycle-first ordering: `active`, `completed`, `idle`, `missing`,
  `invalid`, with `last_seen_at` descending inside each lifecycle bucket.
- Keep lifecycle state visible on each dashboard row so the list still explains
  why entries appear where they do.
- Add a dashboard-local API action that removes all currently degraded
  `missing` and `invalid` watched workspace records from the machine-local
  watchlist.
- Add a dashboard home control for the cleanup action, with disabled, empty,
  busy, success-refresh, and error behavior.
- Keep cleanup explicitly scoped to watchlist membership removal through the
  existing `Unwatch` semantics.
- Update focused Go and web tests for ordering, bulk cleanup, request handling,
  and frontend rendering/action behavior.
- Update the watchlist/dashboard contract and README text where they still say
  the dashboard home is globally recency-sorted or only supports per-workspace
  unwatch.

### Out of Scope

- Sort mode pickers, saved views, search, filters, grouping UI, or richer
  dashboard browsing controls.
- Silent automatic garbage collection of missing, invalid, idle, completed, or
  stale watched workspaces.
- Any new persisted watchlist schema fields, tombstones, hidden states, or
  dashboard archive buckets.
- Deleting local checkout directories, removing git worktrees, or mutating
  tracked plan/runtime workflow state.
- Changing how workspaces enter the watchlist or how `last_seen_at` is
  refreshed by core harness commands.

## Acceptance Criteria

- [x] Dashboard home entries render in lifecycle-first order with active work
      ahead of recent degraded entries, while entries within the same lifecycle
      state remain ordered by `last_seen_at` descending with deterministic path
      fallback.
- [x] The dashboard read model/API and frontend helper behavior agree on the
      ordering contract; tests prevent a future frontend re-sort from undoing
      active-first behavior.
- [x] Dashboard home exposes an explicit bulk cleanup action only when at
      least one `missing` or `invalid` entry is present.
- [x] The bulk cleanup action removes all currently degraded watchlist records
      and preserves all active, completed, and idle watched records.
- [x] Bulk cleanup behaves like repeated explicit `Unwatch`: it does not call
      `harness archive`, does not delete checkouts, and does not mutate tracked
      plans or harness runtime workflow state.
- [x] The UI refreshes the dashboard after successful cleanup and surfaces a
      readable error if cleanup fails.
- [x] Existing per-workspace `Unwatch` behavior and degraded workspace route
      behavior continue to work.
- [x] Tracked docs describe the new active-first default and explicit degraded
      cleanup without implying automatic GC or a new hidden/archive lifecycle.

## Deferred Items

- Richer dashboard sort/filter controls such as `Recent`, `Active first`, or
  `Degraded first`.
- Repository-family grouping for primary checkouts and linked worktrees.
- Stale age policies such as "unwatch missing entries older than N days".
- Richer degraded-entry recovery flows beyond watchlist cleanup.

## Work Breakdown

### Step 1: Make dashboard ordering lifecycle-first

- Done: [x]

#### Objective

Move the canonical dashboard home ordering to active-first lifecycle order,
with recency preserved inside each lifecycle state.

#### Details

Today, `internal/dashboard.Service.Read` sorts entries by global
`last_seen_at`, groups them, and `web/src/helpers.ts` flattens the groups and
sorts globally by recency again. That means a recently touched missing temp
worktree can appear above the active plan the human actually needs.

The clean target is one ordering rule shared by backend and frontend:
`active`, `completed`, `idle`, `missing`, `invalid`, then recency inside each
state. Prefer using the backend group order as the canonical API ordering and
remove or replace the frontend global recency re-sort so it cannot undo the
server contract.

#### Expected Files

- `internal/dashboard/service.go`
- `internal/dashboard/service_test.go`
- `web/src/helpers.ts`
- `web/src/pages.test.tsx`

#### Validation

- Add or update dashboard service tests proving a newer `missing` entry sorts
  after older active/completed/idle entries.
- Add or update web helper/page tests proving flattened dashboard entries keep
  lifecycle-first order instead of global recency order.

#### Execution Notes

Added backend coverage for lifecycle group order and frontend coverage for
flattening dashboard groups without re-sorting globally by recency. Updated
`dashboardWorkspaces` to preserve the API group order, leaving recency sorting
within lifecycle groups to the dashboard read model.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step was completed as part of one cohesive
dashboard slice and is covered by the final full review plus focused
dashboard ordering tests.

### Step 2: Add explicit degraded bulk unwatch API

- Done: [x]

#### Objective

Expose a dashboard-local write action that removes all currently `missing` and
`invalid` watched workspace records.

#### Details

The action should classify the current dashboard read model, select entries
whose `dashboard_state` is `missing` or `invalid`, and remove those exact
watchlist records. It should preserve all other records and remain idempotent
when there are no degraded records.

The implementation may use repeated existing `watchlist.Service.Unwatch`
calls or a focused watchlist service helper if that keeps locking and tests
cleaner. It must handle paths that no longer exist, malformed absolute-path
records surfaced as invalid, and route-key collisions without silently
touching unrelated records.

#### Expected Files

- `internal/ui/server.go`
- `internal/ui/server_test.go`
- `internal/watchlist/watchlist.go`
- `internal/watchlist/watchlist_test.go`

#### Validation

- Add server tests for the bulk endpoint proving missing/invalid entries are
  removed and active/completed/idle entries remain.
- Add watchlist tests only if a new watchlist helper is introduced.
- Preserve existing tests that prove UI read endpoints do not rewrite
  `watchlist.json`.

#### Execution Notes

Added `watchlist.Service.UnwatchWorkspacePaths` for exact persisted-path
removal and a dashboard-local `POST /api/dashboard/unwatch-degraded` endpoint.
The endpoint derives degraded entries from the dashboard read model and removes
only `missing` and `invalid` watchlist records.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step is covered by the final full review plus
focused watchlist and UI server tests for exact-path removal and degraded-only
bulk cleanup.

### Step 3: Wire the dashboard cleanup control

- Done: [x]

#### Objective

Add the visible dashboard home affordance for explicit degraded cleanup and
wire it to the new API action.

#### Details

The control should be present only on the dashboard home and should read as an
explicit unwatch operation, not cleanup by automation. A label such as
`Unwatch missing/invalid` is preferred because it matches the existing
membership-removal terminology and keeps the destructive boundary narrow.

The UI should disable the action while a cleanup is in flight, hide or disable
it when no degraded entries are present, refresh the dashboard after success,
and surface an error near the dashboard home if the request fails. Do not add
sort controls or a dashboard toolbar framework beyond what this action needs.

#### Expected Files

- `web/src/main.tsx`
- `web/src/pages.tsx`
- `web/src/workspace-actions.ts`
- `web/src/pages.test.tsx`
- `web/src/styles.css`

#### Validation

- Add frontend tests proving the cleanup control appears when degraded entries
  exist, is absent or disabled when none exist, calls the bulk endpoint, and
  preserves existing per-row `Unwatch` behavior.
- Run the existing web test suite for affected components.

#### Execution Notes

Added the `Unwatch missing/invalid` dashboard home affordance, busy/error
state, action request helper, and App-level bulk cleanup fetch path. Successful
cleanup refreshes the dashboard; failed cleanup is shown inline on the
dashboard page.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step is covered by the final full review plus
focused web tests for action rendering, request construction, and App-level
failure handling.

### Step 4: Align docs and visual validation

- Done: [x]

#### Objective

Update durable documentation and verify the dashboard still renders cleanly
after the ordering and cleanup changes.

#### Details

The watchlist/dashboard contract currently says the first dashboard home may
flatten entries into one recency-sorted list and that v1 has no automatic GC.
Revise that wording to keep "no automatic GC" intact while describing the new
explicit degraded cleanup action and active-first default ordering.

The README should continue to explain that dashboard-local writes are scoped
to watchlist membership, now noting both per-workspace `Unwatch` and the
explicit degraded cleanup action.

#### Expected Files

- `docs/specs/watchlist-contract.md`
- `README.md`
- `scripts/ui-playwright-smoke`

#### Validation

- Run `harness plan lint` for this plan before approval.
- Run focused Go tests for dashboard, UI server, and watchlist changes.
- Run focused web tests for helpers/pages/actions.
- Run or update the dashboard path in `scripts/ui-playwright-smoke` so the
  active-first list and cleanup affordance are covered visually enough for
  this UI change.

#### Execution Notes

Updated README and the watchlist contract to describe lifecycle-first
dashboard ordering and explicit degraded cleanup as user-triggered `Unwatch`
behavior. Rebuilt embedded UI assets. Did not extend `scripts/ui-playwright-smoke`
because its current fixture is workspace-workbench oriented and dashboard
cleanup is covered by focused Go/web tests without introducing global
watchlist fixture coupling.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this documentation and validation alignment is covered
by the final full review plus plan lint, focused Go tests, web tests, TypeScript
build, and embedded UI build.

## Validation Strategy

- `harness plan lint docs/plans/active/2026-06-10-prioritize-active-dashboard-workspaces-and-cleanup-degraded-watches.md`
- `go test ./internal/dashboard ./internal/ui ./internal/watchlist`
- `pnpm --dir web test -- --run`
- `scripts/ui-playwright-smoke` or an equivalent focused dashboard smoke path
  if the full script is too broad for the execution environment

## Risks

- Risk: Bulk cleanup could remove a record that is temporarily unreadable but
  still important.
  - Mitigation: Make the action explicit, label it with `Unwatch`, scope it to
    currently derived `missing` and `invalid`, and leave active/completed/idle
    entries untouched.
- Risk: Backend and frontend ordering could drift if both sort independently.
  - Mitigation: Keep the API group order canonical and add frontend tests that
    prevent reintroducing a global recency sort.
- Risk: The cleanup affordance could be confused with `harness archive` or git
  worktree deletion.
  - Mitigation: Reuse existing `Unwatch` terminology and update docs to state
    that cleanup only removes machine-local watchlist membership.

## Validation Summary

- `harness plan lint docs/plans/active/2026-06-10-prioritize-active-dashboard-workspaces-and-cleanup-degraded-watches.md`
- `go test ./internal/dashboard ./internal/ui ./internal/watchlist`
- `pnpm --dir web test -- --run`
- `pnpm --dir web exec tsc -p tsconfig.json --noEmit`
- `scripts/build-embedded-ui`
- `go test ./...`

## Review Summary

Finalize review `review-001-full` passed clean with 0 blocking and 0
non-blocking findings. The `correctness` reviewer found no issues in dashboard
ordering, bulk degraded unwatch, exact-path watchlist removal, or UI action
handling. The `tests-docs` reviewer found no gaps in the focused coverage or
tracked docs for active-first ordering, explicit degraded cleanup, and the
no-automatic-GC boundary.

## Archive Summary

- Archived At: 2026-06-10T23:55:55+08:00
- Revision: 1
- PR: not created yet; publish evidence will record the PR URL after archive.
- Ready: The candidate satisfies the tracked acceptance criteria, focused Go
  and web validation passed, `go test ./...` passed, embedded UI assets were
  rebuilt, and `review-001-full` passed clean.
- Merge Handoff: Archive the plan, commit the archive move and code changes,
  push `codex/dashboard-active-first-cleanup`, open a draft PR, record
  publish/CI/sync evidence, and wait for explicit human merge approval.

## Outcome Summary

### Delivered

- Dashboard home flattening now preserves lifecycle-first API group order, so
  active/completed/idle work appears before missing/invalid entries while
  recency remains the within-state ordering signal.
- Added exact persisted-path watchlist removal through
  `watchlist.Service.UnwatchWorkspacePaths`.
- Added `POST /api/dashboard/unwatch-degraded`, which removes only currently
  derived `missing` and `invalid` watched records.
- Added the dashboard `Unwatch missing/invalid` affordance with busy state,
  inline error handling, and success refresh behavior.
- Updated README and the watchlist contract to document active-first ordering
  and explicit degraded cleanup without introducing automatic GC, hidden
  state, or workflow mutation.

### Not Delivered

- Sort mode pickers, saved views, search, filters, repository-family grouping,
  stale-age cleanup policies, and richer degraded recovery flows remain out of
  scope.

### Follow-Up Issues

No follow-up issue was created. The deferred items are optional future
dashboard enhancements, not durable scope required by this slice.
