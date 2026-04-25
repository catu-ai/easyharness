---
template_version: 0.2.0
created_at: "2026-04-25T23:26:35+08:00"
approved_at: "2026-04-25T23:27:52+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/189
size: M
---

# Clarify Status Snapshot and Mutation Coordination

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #189 by turning status resolution and UI/API resource reads into
clear read-only snapshots, while keeping agent-facing `harness status`
conservative through an explicit short settle wait at the CLI boundary.

The end state should remove the confusing split where `status.Service.Read()`
sounds like a plain read but creates or competes for `.state-mutation.lock`.
Mutation locks should belong to mutation coordination, read models should stay
side-effect-free, and successful core CLI commands should continue refreshing
the machine-local watchlist according to the existing watchlist contract.

## Scope

### In Scope

- Update the normative state, CLI, and watchlist docs so future agents can tell
  which reads are pure snapshots, which commands may wait for mutation
  settlement, and which surfaces may refresh watchlist recency.
- Refactor `status` service APIs so status snapshot resolution is read-only and
  does not acquire state mutation locks, write workflow state, or own
  watchlist/timeline side effects.
- Add an explicit CLI `harness status` settle path that briefly waits for an
  in-progress state mutation to release its advisory lock before reading a
  snapshot, then returns a clear busy/error result if the timeout is exceeded.
- Keep mutation commands on the existing fail-fast mutation-lock behavior unless
  a command-specific contract explicitly says otherwise.
- Ensure UI/API/dashboard status and workspace-detail reads use the read-only
  snapshot path and never create `.state-mutation.lock` or touch watchlist
  recency.
- Preserve the accepted watchlist behavior that successful core CLI workflow
  commands, including `harness status`, silently refresh `last_seen_at` through
  the shared best-effort CLI postprocessor.

### Out of Scope

- Redesigning review, evidence, timeline, or watchlist artifact formats.
- Removing `state.json`, `.local/harness/current-plan.json`, or the existing
  mutation lock files.
- Introducing long-running command coordination or background job tracking.
- Changing dashboard product scope beyond keeping its read model pure.
- Adding compatibility shims for the old ambiguous `status.Service.Read()` API.

## Acceptance Criteria

- [x] Specs state that read-model services are snapshot reads: they do not
      acquire mutation locks, write workflow state, append timeline events, or
      touch the watchlist.
- [x] Specs state that CLI `harness status` may wait briefly for a currently
      held state mutation lock before resolving a snapshot, and reports a clear
      busy/error result if the lock remains held after the timeout.
- [x] Specs preserve the distinction that mutation commands fail fast on
      mutation-lock contention, while status checkpoint reads may wait because
      harness commands are not expected to be long-running.
- [x] `status` package exposes a clearly named read-only snapshot API, and
      callers no longer rely on a lock-acquiring `Read()`/`ReadUnlocked()` pair.
- [x] UI and dashboard status reads use the snapshot API and have regression
      coverage proving they do not create `.state-mutation.lock`, mutate
      workflow state, or touch the watchlist.
- [x] CLI `harness status` has regression coverage for the settle behavior:
      it waits through a short held lock when the lock releases, and returns a
      clear contention result when the lock stays held past the timeout.
- [x] Successful core CLI command watchlist registration still works, including
      `harness status`, while UI/API/dashboard polling remains excluded.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Specify read and mutation coordination boundaries

- Done: [x]

#### Objective

Document the durable contract for status snapshots, CLI status settlement,
mutation lock contention, and watchlist touch boundaries.

#### Details

`docs/specs/state-model.md` already defines `state.json` as
control-plane-only and mutation locks as write coordination. Extend it so
read-model services are explicitly snapshot readers rather than lock owners.
Describe the expected reader behavior when a snapshot observes temporarily
incomplete artifact state: surface a degraded result or warning rather than
hiding the possibility behind a read lock.

`docs/specs/cli-contract.md` should carry the user-facing `harness status`
contract: status is an agent-facing checkpoint and may briefly wait for an
active state mutation to settle before resolving the snapshot. If the lock is
still held after the short timeout, status should return a clear busy/error
result instead of reading a likely in-flight state.

`docs/specs/watchlist-contract.md` already says successful core workflow
commands can refresh `last_seen_at` and UI/dashboard reads should not inflate
recency. Cross-reference the read-model purity rule so future UI/API work does
not reintroduce watchlist touches through polling surfaces.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- `docs/specs/watchlist-contract.md`

#### Validation

- The docs describe the read/write boundary without depending on this plan or
  discovery chat.
- `git diff --check` passes for the edited specs.

#### Execution Notes

Updated `docs/specs/state-model.md`, `docs/specs/cli-contract.md`, and
`docs/specs/watchlist-contract.md` to define read-model purity, CLI
`harness status` settle behavior, mutation-command fail-fast lock semantics,
and the CLI-only watchlist refresh boundary. Validation: `git diff --check`
for the edited specs and plan; `harness plan lint
docs/plans/active/2026-04-25-clarify-status-snapshot-and-mutation-coordination.md`.
Follow-up repair after `review-001-delta` clarified that the status settle
check must be passive and non-destructive: no missing lock-file creation, no
mutation-lock ownership as a quiescence probe, and no mutation lock held while
resolving the snapshot.

#### Review Notes

`review-001-delta` found one blocking correctness issue: the initial settle
contract did not explicitly forbid lock-owning or lock-file-creating probes.
The repair added passive/non-destructive settle requirements to the state model
and CLI contract. `review-002-delta` passed with no findings.

### Step 2: Refactor status snapshot APIs

- Done: [x]

#### Objective

Make status resolution a plainly named read-only snapshot service and remove
the ambiguous lock-acquiring `Read()` API shape.

#### Details

Refactor `internal/status` so the primary service entrypoint is named for what
it does, such as `Snapshot()` or `ResolveSnapshot()`. It should not acquire
`.state-mutation.lock`, write `state.json`, append timeline events, touch the
watchlist, or expose an `AfterSuccess` hook.

Update internal callers to use the new snapshot API. UI, dashboard, timeline
hooks, and other read-model callers should all consume the same pure snapshot
path. Because this repository does not preserve compatibility for intermediate
internal APIs by default, remove the confusing `Read()`/`ReadUnlocked()` pair
rather than keeping wrappers that invite future misuse.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/dashboard/service.go`
- `internal/cli/timeline_events.go`
- other direct `status.Service` callers found by `rg`

#### Validation

- `rg "ReadUnlocked|status.Service\\{[^}]*\\}\\.Read\\(" internal` finds no
  remaining status service calls.
- Focused status tests prove snapshot resolution still derives the same nodes
  without writing cache fields into `state.json`.

#### Execution Notes

Replaced the ambiguous status `Read()`/`ReadUnlocked()` pair with a single
read-only `status.Service.Snapshot()` API, removed the status-level
`AfterSuccess` hook, and updated CLI timeline snapshots, dashboard status
aggregation, UI status endpoints, and direct tests/callers. Validation:
`rg` confirmed no status `Read()`/`ReadUnlocked()` callers remain; `go test
./internal/status ./internal/cli ./internal/ui ./internal/dashboard
./internal/lifecycle ./internal/review`.

#### Review Notes

`review-003-delta` passed with correctness and tests slots, both with no
findings.

### Step 3: Add CLI status settle behavior

- Done: [x]

#### Objective

Move conservative mutation coordination to the `harness status` command
boundary by waiting briefly for held state mutation locks before reading a
snapshot.

#### Details

Add a runstate helper that can detect and wait for an actually held advisory
state mutation lock for the current plan. Do not use lock-file existence as the
signal because lock files may remain on disk after the lock is released. The
helper should use the same plan-local lock primitive or a compatible
non-destructive advisory-lock probe.

`harness status` should detect the current plan, wait a short bounded interval
for any held state mutation lock to release, and then call the pure status
snapshot API. If the lock stays held past the timeout, return a clear status
result explaining that another local state mutation is still in progress.

Keep mutation commands fail-fast on lock contention. This settle behavior is
for checkpoint reads, not for competing workflow mutations.

#### Expected Files

- `internal/runstate/state.go`
- `internal/runstate/state_test.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/status/service.go`

#### Validation

- Add tests where `harness status` succeeds after a held lock is released within
  the settle timeout.
- Add tests where `harness status` returns the busy/error result when the lock
  remains held.
- Existing mutation-command lock contention tests continue to prove fail-fast
  behavior.

#### Execution Notes

Added runstate helpers for passive state-mutation lock probing and bounded
settle waits. `harness status` now waits briefly for a held state mutation
lock before calling the pure status snapshot resolver, returns a clear busy
status if the lock remains held, and does not create `.state-mutation.lock`
when no lock file exists. Adjusted the specs to describe the implementable
non-destructive probe boundary. Validation: `git diff --check`; `go test
./internal/runstate ./internal/cli ./internal/status`.

#### Review Notes

`review-004-delta` passed with correctness and tests slots, both with no
findings.

### Step 4: Lock UI/API and watchlist boundaries with tests

- Done: [x]

#### Objective

Prove the issue #189 surfaces are fixed and the accepted watchlist recency
contract still holds.

#### Details

Update UI handler status routes so top-level `/api/status` and workspace-detail
`/api/workspace/<key>/status` use the pure snapshot API. Add active-plan
regressions that snapshot workflow files before the request and assert that the
request does not create `.state-mutation.lock`, rewrite `state.json`, rewrite
`current-plan.json`, or touch the machine-local watchlist.

Also preserve positive CLI watchlist coverage for successful core workflow
commands, especially `harness status`, so read-model purity is not confused
with removing the intentional command-level recency refresh.

#### Expected Files

- `internal/ui/server.go`
- `internal/ui/server_test.go`
- `internal/cli/app_test.go`
- `internal/dashboard/service_test.go`
- `docs/plans/active/2026-04-25-clarify-status-snapshot-and-mutation-coordination.md`

#### Validation

- Focused UI tests fail if status API polling creates `.state-mutation.lock` or
  touches `watchlist.json`.
- Focused CLI tests fail if successful `harness status` stops refreshing the
  watchlist.
- The final test run covers at least `./internal/status`, `./internal/runstate`,
  `./internal/cli`, `./internal/ui`, `./internal/dashboard`, and
  `./internal/watchlist`.

#### Execution Notes

Added active-plan UI/API regressions for top-level `/api/status` and
workspace-detail `/api/workspace/<key>/status`. The tests snapshot
`current-plan.json`, plan-local `state.json`, and the machine-local
watchlist when present, then assert status polling does not rewrite those
files or create `.state-mutation.lock`. Validation: `go test ./internal/ui
-count=1`; `go test ./internal/status ./internal/runstate ./internal/cli
./internal/ui ./internal/dashboard ./internal/watchlist -count=1`; `git diff
--check`.

#### Review Notes

`review-005-delta` passed with correctness and tests slots, both with no
findings.

## Validation Strategy

- Run `harness plan lint` on this plan before approval.
- During execution, run focused Go tests for each touched package as behavior
  lands.
- Before archive, run the combined focused suite:
  `go test ./internal/status ./internal/runstate ./internal/cli ./internal/ui ./internal/dashboard ./internal/watchlist -count=1`.
- Run `git diff --check`.
- If status service API renaming affects generated schemas or docs, run the
  relevant contract-sync check named by the failing tests or nearby docs.

## Risks

- Risk: Removing the lock-acquiring status read could expose transient
  multi-file mutation snapshots to CLI users.
  - Mitigation: Keep the pure snapshot API for read models, but add bounded
    settle behavior specifically to `harness status` before it resolves the
    snapshot.
- Risk: The status API rename could miss a caller and leave a hidden
  lock-acquiring path alive.
  - Mitigation: Use `rg` checks and package tests to prove the old
    `Read()`/`ReadUnlocked()` pair is gone from status callers.
- Risk: Watchlist recency behavior could be accidentally removed while making
  read services pure.
  - Mitigation: Keep watchlist touch at the CLI successful core-command
    postprocessor boundary and preserve positive/negative watchlist tests.
- Risk: Timeout-based status settle behavior could make tests flaky.
  - Mitigation: Keep the wait helper injectable or use deterministic short
    lock-release coordination in tests rather than relying on wall-clock races.

## Validation Summary

- `git diff --check`
- `harness plan lint
  docs/plans/active/2026-04-25-clarify-status-snapshot-and-mutation-coordination.md`
- `go test ./internal/status ./internal/runstate ./internal/cli ./internal/ui
  ./internal/dashboard ./internal/watchlist -count=1`
- Step-focused validation also covered `./internal/lifecycle` and
  `./internal/review` after the status snapshot API rename.

## Review Summary

- Step reviews: `review-001-delta` found and `review-002-delta` verified the
  passive/non-destructive settle-contract repair; `review-003-delta`,
  `review-004-delta`, and `review-005-delta` passed with no findings.
- Finalize reviews: `review-006-full` found one blocking tests gap in the
  positive CLI settle test. The repair made the test prove `harness status`
  stays blocked while the state mutation lock is held and succeeds after
  release. `review-007-full` passed with correctness and tests slots, both
  with no findings.

## Archive Summary

- Archived At: 2026-04-26T00:01:44+08:00
- Revision: 1
- PR: To be created after archive from this candidate branch.
- Ready: The tracked steps are complete, the repair finalize review passed,
  focused validation is green, and no follow-up issues are required.
- Merge Handoff: After archive, commit the archive move, push the branch, open
  the PR, record publish/CI/sync evidence, and wait for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Defined the durable read/write contract for status snapshots, CLI status
  settlement, mutation-command lock contention, and watchlist ownership in the
  state, CLI, and watchlist specs.
- Replaced the ambiguous `status.Service.Read()`/`ReadUnlocked()` API with
  read-only `status.Service.Snapshot()` callers and removed the status-level
  success hook.
- Added passive runstate lock probing and bounded CLI `harness status` settle
  behavior that waits briefly for held mutation locks and reports a clear busy
  status on timeout.
- Added regression coverage for snapshot read purity, CLI settle wait/busy
  behavior, UI/API/dashboard no-mutation reads, and intentional CLI
  watchlist refresh.

### Not Delivered

- No compatibility wrappers for the removed internal status read API.
- No redesign of review, evidence, timeline, watchlist artifact formats, or
  long-running command coordination.

### Follow-Up Issues

NONE
