---
template_version: 0.2.0
created_at: "2026-06-10T00:00:00Z"
approved_at: "2026-06-10T23:49:08+08:00"
source_type: issue
source_refs:
    - '#232'
size: XS
---

# Cover dashboard custom-root workspace reads

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add focused regression coverage for issue #232 so dashboard and watchlist read
surfaces stay compatible with per-workspace `.harness/config.yaml` path-root
customization.

Discovery found that the current implementation already appears to satisfy the
behavior: `internal/dashboard.Service` reads each watched workspace through
that workspace's `status.Service`, and status/plan/runstate resolution already
honors configured active plan and local runtime roots. A manual probe against
`/api/dashboard` also classified a watched custom-root workspace as active,
reported the custom plan path, and read execution state from the custom local
runtime root without creating `.local/harness`. This slice turns that evidence
into durable tests and issue-closeout evidence.

## Scope

### In Scope

- Add dashboard service regression coverage for watched workspaces with custom
  active plan and custom local runtime roots.
- Add UI/API regression coverage for `/api/dashboard` and the watched workspace
  route using the same kind of custom-root workspace.
- Preserve the machine-local watchlist contract: reads use `EASYHARNESS_HOME`
  or the default easyharness home and do not persist repo-config-derived
  watchlist fields.
- Document validation evidence in the plan closeout so #232 can be closed by
  the PR or follow-up issue comment.

### Out of Scope

- Production behavior changes, unless the new tests reveal a real current
  mismatch with #232.
- Changing the global watchlist storage location through repo config.
- Adding persisted per-workspace customization fields to `watchlist.json`.
- New dashboard UI presentation changes.

## Acceptance Criteria

- [x] A watched git workspace with `.harness/config.yaml` setting
      `paths.plans.active` to a non-default root is classified correctly by
      dashboard reads when its active plan lives under that custom root.
- [x] A watched workspace with `paths.local_runtime` set to a non-default root
      is classified from state under that custom runtime root, with no default
      `.local/harness` state created by dashboard reads.
- [x] Dashboard service output includes the custom repo-relative plan path and
      plan title/progress context for the custom-root active workspace.
- [x] UI/API tests prove `/api/dashboard` and a watched workspace route use the
      same repo-config-aware read path.
- [x] Watchlist storage remains machine-local and non-mutating during dashboard
      reads.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Cover dashboard service custom-root reads

- Done: [x]

#### Objective

Add a focused `internal/dashboard` regression test that exercises the read model
directly with a watched custom-root workspace.

#### Details

Build a temporary git workspace with:

- `.harness/config.yaml` configuring custom active, archived, and local runtime
  roots.
- one active plan under the configured active root.
- execution-start state recorded under the configured local runtime root.
- a watchlist stored under a temporary `EASYHARNESS_HOME`.

The test should call `dashboard.Service{}.Read()` using the existing test
helpers where practical. It should assert that the watched workspace lands in
the `active` group, carries the custom repo-relative plan path, surfaces the
plan title/progress context, and does not create default `.local/harness`
runtime state as a side effect of reading.

#### Expected Files

- `internal/dashboard/service_test.go`

#### Validation

- `go test ./internal/dashboard -count=1`
- The new test fails if dashboard status classification ignores either the
  configured active plan root or the configured local runtime root.

#### Execution Notes

Added `TestReadUsesWorkspaceRepoConfigForCustomRootActiveWorkspace` to
`internal/dashboard/service_test.go`. The test seeds a watched git workspace
with custom active, archived, and local runtime roots; stores the active plan
and execution state under those configured roots; verifies dashboard
classification, custom plan path, title/progress context, and non-mutating
runtime reads; and asserts the default `.local/harness` runtime root remains
absent.

Validation: `go test ./internal/dashboard -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is a narrow service-test addition and will be
covered by the full finalize review for the complete #232 regression coverage
slice.

### Step 2: Cover dashboard UI/API custom-root routes

- Done: [x]

#### Objective

Add HTTP handler coverage proving the dashboard API routes preserve the same
repo-config-aware behavior.

#### Details

Extend `internal/ui/server_test.go` with a custom-root watched workspace
fixture similar to Step 1. Exercise `/api/dashboard` and the matching
`/api/workspace/<key>` route, and optionally `/api/workspace/<key>/status` or
`/api/workspace/<key>/plan` if that keeps the assertions clearer.

The test should keep the API surface read-only: preserve the watchlist file
bytes/mtime when relevant, avoid any mutation of workflow state during reads,
and assert that `EASYHARNESS_HOME` remains the source of watchlist storage.

#### Expected Files

- `internal/ui/server_test.go`

#### Validation

- `go test ./internal/ui -count=1`
- API payload assertions show the custom-root workspace as watched and active
  with the custom repo-relative plan path.

#### Execution Notes

Added `TestNewHandlerDashboardAndWorkspaceRoutesUseCustomWorkspaceRepoConfig`
to `internal/ui/server_test.go`. The test serves `/api/dashboard` and
`/api/workspace/<key>` against a watched custom-root workspace, verifies the
active dashboard state, custom repo-relative plan path, plan title, and current
execution node, and asserts both workflow state and the machine-local
watchlist remain unchanged by reads. The shared UI active-state helper now
derives lock paths through `runstate.PlanRuntimePath` so default and custom
runtime roots are both represented correctly.

Validation: `go test ./internal/ui -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is the companion UI/API regression test for
the same #232 coverage slice and will be covered by the full finalize review
before archive.

## Validation Strategy

- Run focused package tests after each step:
  - `go test ./internal/dashboard -count=1`
  - `go test ./internal/ui -count=1`
- Run the adjacent resolver/read-model packages before archive:
  - `go test ./internal/dashboard ./internal/ui ./internal/status ./internal/plan ./internal/runstate -count=1`
- If the tests uncover a real behavior gap, keep the production fix narrowly
  scoped to the read path that failed and rerun the focused package set.

## Risks

- Risk: The tests duplicate too much custom-root fixture setup and become hard
  to maintain.
  - Mitigation: Reuse existing test helpers when possible and add only small
    local helpers for custom-root setup.
- Risk: The dashboard service and UI tests assert slightly different fixture
  behavior.
  - Mitigation: Keep the workspace shape identical across both tests: custom
    active root, custom archived root, custom local runtime root, active plan,
    execution-start state, and machine-local watchlist entry.
- Risk: New tests accidentally mutate watchlist or runtime state while proving
  read behavior.
  - Mitigation: Snapshot watchlist/runtime files around reads where the test
    touches non-mutating behavior, and assert default `.local/harness` remains
    absent for custom-runtime fixtures.

## Validation Summary

Validated during execution:

- `go test ./internal/dashboard -count=1`
- `go test ./internal/ui -count=1`
- `go test ./internal/dashboard ./internal/ui ./internal/status ./internal/plan ./internal/runstate -count=1`

## Review Summary

Finalize full review `review-001-full` passed with zero blocking findings and
zero non-blocking findings. Reviewer slots covered correctness and test
coverage, including custom active plan roots, custom local runtime reads,
plan-context assertions, and non-mutating dashboard/watchlist behavior.

## Archive Summary

- Archived At: 2026-06-10T23:55:21+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate satisfies all acceptance criteria, passed focused
  validation, and has a clean finalize review.
- Merge Handoff: Commit and push the archive move, open a PR for #232, record
  publish/CI/sync evidence, then wait for explicit human merge approval.

## Outcome Summary

### Delivered

Added dashboard service and UI/API regression coverage proving watched
workspaces with per-workspace custom active plan and local runtime roots remain
readable through dashboard surfaces without mutating watchlist or workflow
state.

### Not Delivered

No production behavior changes were needed; discovery and tests confirmed the
existing read path already honors repo-level path customization for #232.

### Follow-Up Issues

NONE
