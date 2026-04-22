---
template_version: 0.2.0
created_at: "2026-04-22T22:36:21+08:00"
approved_at: "2026-04-22T22:56:56+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/165
size: M
---

# Build Watchlist Dashboard Read Model

## Goal

Build the read-only backend model that turns the machine-local watchlist into
dashboard-ready watched workspace summaries. The model should preserve the raw
harness workflow state for each readable workspace while adding a dashboard
lifecycle state that is honest about active work, completed landed work, idle
but ambiguous workspaces, missing paths, and invalid watched entries.

This slice should give the future dashboard UI one compact API payload to read
without making the frontend reimplement watchlist parsing, status resolution,
recency ordering, or degraded-entry handling.

## Scope

### In Scope

- Add a read-only watchlist reader that loads the machine-local
  `watchlist.json` without mutating harness workflow or watchlist state.
- Add a dashboard read model service that summarizes every watched workspace.
- Reuse the existing `status.Service` for readable watched workspaces so
  `current_node`, summary, warnings, blockers, next actions, and artifacts stay
  consistent with `harness status`.
- Expose both raw `current_node` and a dashboard lifecycle state for each
  watched workspace.
- Classify dashboard states as `active`, `completed`, `idle`, `missing`, or
  `invalid`.
- Treat `execution/finalize/await_merge` and `land` as `active`, because they
  still require human steering or post-merge bookkeeping.
- Treat `current_node: "idle"` with last-landed context as `completed`.
- Treat ordinary `current_node: "idle"` without last-landed context as `idle`
  rather than pretending it is completed.
- Surface invalid watched entries with a reason such as `unreadable`,
  `not_git_workspace`, or `status_error`.
- Update the tracked watchlist/dashboard contract docs so future UI work can
  consume the read model without reverse-engineering Go types or tests.
- Serve the read model through a UI server API endpoint suitable for the future
  dashboard home.
- Add focused tests for watchlist reads, dashboard classification, degraded
  entries, recency ordering, and API behavior.

### Out of Scope

- Building the dashboard frontend UI.
- Adding the `harness dashboard` CLI entrypoint.
- Adding filesystem watching, notifications, or background refresh.
- Adding dashboard-local write actions such as unwatch.
- Changing the persisted watchlist schema.
- Adding compatibility shims for obsolete watchlist or status shapes.

## Acceptance Criteria

- [ ] The dashboard read model reads watched workspaces from the machine-local
      watchlist and returns one compact summary per watched entry.
- [ ] Each readable workspace summary exposes the raw harness `current_node`
      alongside a dashboard lifecycle state.
- [ ] `execution/finalize/await_merge`, `land`, and other non-idle readable
      nodes classify as `active`.
- [ ] Idle workspaces with `artifacts.last_landed_at` classify as `completed`.
- [ ] Idle workspaces without last-landed context classify as `idle`, not
      `completed`.
- [ ] Missing watched paths classify as `missing` and remain visible in the
      model.
- [ ] Existing but invalid watched paths classify as `invalid` with a reason
      that preserves whether the path was unreadable, not git-backed, or failed
      status resolution.
- [ ] The read model is read-only and tests prove it does not touch or rewrite
      the watchlist or workflow state.
- [ ] Dashboard entries are ordered by watchlist recency using `last_seen_at`,
      with deterministic fallback ordering for malformed or equal timestamps.
- [ ] A UI server API endpoint returns the dashboard read model as JSON without
      disturbing the existing `/api/status`, `/api/plan`, `/api/timeline`, or
      `/api/review` endpoints.
- [ ] Tracked specs document the dashboard read model payload boundary,
      `current_node` exposure, dashboard lifecycle states, and invalid reason
      semantics.

## Deferred Items

- Implement the `harness dashboard` command and default `/dashboard` route in a
  follow-up dashboard entrypoint slice.
- Build the frontend dashboard home and workspace navigation in the minimal UI
  slice.
- Add dashboard-local unwatch behavior after the read model and UI shape are
  proven.

## Work Breakdown

### Step 1: Add a read-only watchlist loading API

- Done: [x]

#### Objective

Expose a public watchlist read path that resolves the same machine-local home
as `Touch` and loads the persisted watchlist without writing files.

#### Details

Keep the persisted watchlist schema unchanged. The reader should share the
existing home resolution and parse validation behavior where possible, but it
must not acquire write-only behavior, update timestamps, canonicalize entries
by rewriting the file, or silently drop malformed workspace records that the
dashboard should present as invalid entries later.

#### Expected Files

- `internal/watchlist/watchlist.go`
- `internal/watchlist/watchlist_test.go`

#### Validation

- Add or update watchlist tests that cover default and `EASYHARNESS_HOME`
  loading, missing watchlist files, invalid JSON or unsupported versions, and
  no write/timestamp side effects during reads.

#### Execution Notes

Added `watchlist.Service.Read()` as a read-only loader that resolves the same
machine-local home as `Touch` and returns the parsed `watchlist.json` model
without creating files or updating timestamps. Added focused coverage for
default home, `EASYHARNESS_HOME`, missing files, invalid JSON, unsupported
versions, and preserving persisted workspace records as read.

#### Review Notes

`review-001-delta` found one blocking tests gap: existing-file reads did not
prove the file bytes or mtime were preserved. Added
`TestReadExistingWatchlistDoesNotRewriteFile` and reran
`go test ./internal/watchlist -count=1`. Follow-up `review-002-delta` passed
with no findings.

### Step 2: Document the dashboard read model contract

- Done: [x]

#### Objective

Update tracked specs with the read model contract that the implementation and
future dashboard UI should share.

#### Details

Document that the dashboard read model is a read-time projection over
`watchlist.json` plus per-workspace harness status. The docs should state that
readable entries expose raw `current_node` separately from dashboard lifecycle
state; that lifecycle states are `active`, `completed`, `idle`, `missing`, and
`invalid`; that ordinary idle without last-landed context remains `idle`; and
that invalid entries carry a reason such as `unreadable`, `not_git_workspace`,
or `status_error`.

Keep this as contract alignment for the read model, not a broader dashboard UI
spec rewrite.

#### Expected Files

- `docs/specs/watchlist-contract.md`
- `docs/specs/proposals/harness-ui-steering-surface.md` if nearby dashboard
  wording needs to stay aligned

#### Validation

- Reread the changed spec sections against this plan and issue #165.
- Run `git diff --check` for the documentation changes.

#### Execution Notes

Updated the watchlist contract and dashboard steering proposal to document the
read-time dashboard projection over `watchlist.json` plus per-workspace
status. The specs now separate raw `current_node` from dashboard lifecycle
state, define `active`, `completed`, `idle`, `missing`, and `invalid`, and
record invalid reasons such as `unreadable`, `not_git_workspace`, and
`status_error`. Validation: `git diff --check -- docs/specs/watchlist-contract.md docs/specs/proposals/harness-ui-steering-surface.md`.

#### Review Notes

`review-003-delta` found one blocking agent-UX gap: lifecycle states were
documented, but the concrete dashboard payload boundary was still implicit.
Added the top-level result, lifecycle group, and per-workspace entry field
contract to `docs/specs/watchlist-contract.md`, plus proposal wording that the
frontend should consume stable groups rather than derive field names.
Follow-up `review-004-delta` passed with no findings.

### Step 3: Build the dashboard summary service

- Done: [x]

#### Objective

Create a dashboard read model that turns watchlist entries into ordered,
dashboard-ready workspace summaries while preserving raw harness status fields.

#### Details

The service should read the watchlist once, then inspect each watched
workspace. For readable workspaces, call `status.Service{Workdir:
workspace_path}.Read()` and map the returned status into a compact dashboard
entry. Keep `current_node` as a first-class field when status is readable.

Dashboard lifecycle mapping should be:

- `active`: status is readable and `current_node` is anything except `idle`.
- `completed`: status is readable, `current_node` is `idle`, and status
  artifacts include last-landed context.
- `idle`: status is readable, `current_node` is `idle`, and no last-landed
  context is present.
- `missing`: the watched path no longer exists.
- `invalid`: the watched path exists but cannot be treated as a valid readable
  harness workspace.

Use an `invalid_reason` or equivalent field to distinguish `unreadable`,
`not_git_workspace`, `status_error`, and other concrete invalid cases without
expanding the top-level dashboard state enum.

#### Expected Files

- `internal/dashboard/service.go`
- `internal/dashboard/service_test.go`
- `internal/contracts/dashboard.go`
- `internal/contracts/registry.go`
- generated schema files if the repository's schema generation expects the new
  contract to be registered

#### Validation

- Add focused dashboard service tests for active, completed, idle, missing,
  not-git, unreadable, and status-error entries.
- Cover recency ordering by `last_seen_at` and deterministic fallback ordering
  for equal, empty, or malformed timestamps.
- Cover that the service remains read-only by checking watchlist and workflow
  state files are not created or modified during reads.

#### Execution Notes

Added `internal/dashboard` with a read-only service that loads the watchlist,
probes watched paths, reuses `status.Service` for readable Git workspaces,
classifies entries into stable dashboard lifecycle groups, and preserves raw
`current_node` on readable status entries. Added `internal/contracts`
dashboard types, registered the UI resource schema, synced generated contract
artifacts, and aligned the spec field name with the existing UI `resource`
pattern. Validation: `go test ./internal/dashboard ./internal/watchlist ./internal/contractsync -count=1`; `scripts/sync-contract-artifacts --check`; `git diff --check`.

#### Review Notes

`review-005-delta` found that default dashboard reads used the locking status
path for active workspaces, that active default-status read-only behavior was
untested, and that Git probe failures could be classified as
`not_git_workspace` instead of `unreadable`. Switched the default status path
to `ReadUnlocked()`, added active-workspace no-mutation coverage, and refined
Git probe failure classification. Validation:
`go test ./internal/dashboard ./internal/watchlist ./internal/contractsync -count=1`.

### Step 4: Serve the dashboard model through the UI backend

- Done: [ ]

#### Objective

Expose the dashboard read model through a JSON endpoint that the future
dashboard home can consume.

#### Details

Add a small endpoint such as `GET /api/dashboard` to the existing UI server.
The endpoint should return the dashboard read model and should not trigger
watchlist writes. Keep existing workbench endpoints unchanged. This step does
not need to add frontend rendering, route handling, or the `harness dashboard`
command.

#### Expected Files

- `internal/ui/server.go`
- `internal/ui/server_test.go`
- `web/src/types.ts` only if the current frontend type surface already tracks
  backend response types for unused endpoints

#### Validation

- Add server tests that prove `GET /api/dashboard` returns dashboard JSON,
  rejects non-GET methods, returns degraded entries instead of dropping them,
  and avoids watchlist writes.
- Run the focused Go tests for watchlist, dashboard, and UI server packages.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run `harness plan lint` before execution approval.
- During execution, use focused package tests first:
  `go test ./internal/watchlist ./internal/dashboard ./internal/ui -count=1`.
- Run broader Go coverage if contract or status integration changes spill into
  shared packages: `go test ./internal/... -count=1`.
- If generated schema or contract docs change, run the repository's existing
  schema generation or validation command identified during implementation.

## Risks

- Risk: Dashboard lifecycle states could blur with harness workflow
  `current_node` states.
  - Mitigation: Keep `current_node` exposed directly and make dashboard state a
    separate presentation lifecycle field with explicit tests for ambiguous
    idle behavior.
- Risk: The dashboard service could accidentally mutate the watchlist or
  workflow state while reading many workspaces.
  - Mitigation: Use read-only APIs and add regression tests that inspect file
    absence or modification times around dashboard reads.
- Risk: Invalid watched entries could be silently dropped by convenience
  filtering.
  - Mitigation: Test missing, unreadable, not-git, and status-error entries as
    first-class returned summaries with reasons.
- Risk: Multi-worktree status reads could be slow or expose inconsistent
  partial state.
  - Mitigation: Keep the first model simple and synchronous, return per-entry
    degraded status for failures, and defer caching or background refresh until
    dashboard UI behavior proves it is needed.

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
