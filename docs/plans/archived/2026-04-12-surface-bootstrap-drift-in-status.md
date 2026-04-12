---
template_version: 0.2.0
created_at: "2026-04-12T13:31:37+08:00"
source_type: direct_request
source_refs:
    - chat://current-session
size: M
---

# Surface Idle Bootstrap Drift Reminder In Status

## Goal

Make `harness status` proactively remind the agent about repo-local bootstrap
drift for the current agent's default repository targets only when the
worktree is idle, instead of leaving stale bootstrap assets discoverable only
when a human reruns `harness init` or the narrower bootstrap install commands.

This slice should add one shared bootstrap-drift detector that `status` can
reuse without inventing hidden repo state. The user-visible behavior should
stay conservative and non-blocking: bootstrap drift becomes a soft reminder for
the agent, expressed as warning debt plus a repair-oriented next action, not a
workflow-state transition and not execution debt.

## Scope

### In Scope

- Define the repo-status contract for detecting stale easyharness-managed
  bootstrap instructions or skills under the current agent's default repo
  targets.
- Add a reusable detector for managed bootstrap drift that compares installed
  managed assets against the currently packaged bootstrap assets.
- Surface bootstrap drift through `harness status` warnings and next actions
  only in the `idle` state without changing `state.current_node` or affecting
  later execution flow.
- Cover the new behavior with focused status and bootstrap tests and update any
  normative docs that need to mention proactive `status` surfacing.

### Out of Scope

- Detecting drift for custom bootstrap targets installed earlier with explicit
  `--dir` or `--file` overrides, because `status` does not persist those paths.
- Introducing a new standalone `doctor` or `check` command in this slice.
- Changing `harness init` or resource-install refresh semantics beyond the
  shared comparison logic needed by `status`.
- Surfacing bootstrap drift while a plan is active or execution is already in
  progress.

## Acceptance Criteria

- [x] `harness status` reports a non-blocking warning plus a repair-oriented
      next action when the worktree is `idle` and the default repo `AGENTS.md`
      managed block or any default repo managed skill package is stale relative
      to the running binary's packaged bootstrap assets.
- [x] `harness status` reports no bootstrap-drift warning when the repo has no
      default managed bootstrap assets installed yet, or when the installed
      managed assets already match the packaged assets.
- [x] Bootstrap drift does not change `state.current_node` and does not block
      or redirect later workflow execution; it appears only as idle-state
      warning debt plus an actionable `next_action` such as rerunning
      `harness init --dry-run`.
- [x] The detection logic is shared code rather than a one-off `status`
      content comparison, and automated coverage proves the idle reminder path
      while active-plan status stays unchanged.

## Deferred Items

- Support for status-based inspection of non-default agent targets or explicit
  override paths.
- Richer bootstrap inspection or repair commands if later product work needs
  more than warnings and ordinary next actions.

## Work Breakdown

### Step 1: Define shared bootstrap drift detection for repo status

- Done: [x]

#### Objective

Add a reusable detector that can classify whether the current agent's default
repo bootstrap instructions or managed skills are absent, current, or stale.

#### Details

The detector should reuse the existing ownership and version-marker rules from
bootstrap install instead of inventing a second interpretation. Keep the scope
to the default repo targets that `harness init` would manage for the current
agent, because `status` has no durable record of prior override paths. The
comparison should treat a repository with no managed bootstrap assets as
"nothing to warn about" rather than drift.

#### Expected Files

- `internal/install/**`
- `internal/status/**`
- `docs/specs/bootstrap-install.md`

#### Validation

- Focused unit tests cover fresh, stale, and absent default repo bootstrap
  states.
- Shared detection code can be exercised without shelling out through the CLI.

#### Execution Notes

Added `internal/install/drift.go` with a shared repo-bootstrap drift inspector
that reuses the install package's default repo target resolution, managed block
markers, managed skill metadata, and canonical packaged asset rendering. The
detector reports only stale installed managed assets and ignores absent default
bootstrap assets so untouched repositories stay quiet. Focused validation:
`go test ./internal/install ./internal/status -count=1` and
`go test ./tests/smoke -run 'TestStatusReportsIdleWorkspace|TestStatusIdleReportsNonBlockingBootstrapReminderWhenManagedAssetsAreStale' -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The detector and idle reminder were implemented as one
tightly coupled slice, so a separate Step 1 closeout review would split the
same risk surface before the final branch-level review.

### Step 2: Surface bootstrap drift through status warnings and guidance

- Done: [x]

#### Objective

Teach `harness status` to surface idle-only bootstrap drift as warnings and a
repair hint while preserving the existing workflow node semantics.

#### Details

Wire the shared detector into the idle path only. Prefer warning text that
tells the agent the default repo bootstrap assets are stale, that this does
not block future work, and what optional command to run next. Keep
`state.current_node` unchanged and leave active-plan status behavior alone.
Update the status and bootstrap contracts only where the new behavior changes
the normative expectation, and add focused tests for idle reminders plus
active-plan non-regression.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `tests/smoke/**`
- `docs/specs/cli-contract.md`
- `docs/specs/bootstrap-install.md`

#### Validation

- `go test` covers idle reminders, no-warning fresh paths, and active-plan
  non-regression.
- Targeted smoke coverage proves `harness status` recommends optional bootstrap
  refresh after managed version markers are made stale while the repo is idle.

#### Execution Notes

Wired the shared detector into the idle-only `status` path so stale default
repo bootstrap assets now surface as a non-blocking warning plus an optional
`harness init --dry-run` next action, while active plan and execution nodes
stay unchanged. Added status unit coverage for stale idle reminders, fresh idle
silence, and active-plan non-regression; extended smoke coverage for the
end-to-end idle reminder; and updated the CLI/bootstrap specs to document the
idle-only reminder contract. Validation:
`go test ./internal/install ./internal/status -count=1` and
`go test ./tests/smoke -run 'TestStatusReportsIdleWorkspace|TestStatusIdleReportsNonBlockingBootstrapReminderWhenManagedAssetsAreStale' -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 depends directly on Step 1's shared detector and
will be covered by the same final branch-level review, so a separate step
review would be redundant for this M-sized integrated slice.

## Validation Strategy

- Prefer unit coverage for detector edge cases and `status` warning ordering.
- Add or update smoke coverage only for the end-to-end idle status reminder
  surface so the contract is proven through the built binary.
- Re-run any focused bootstrap/status tests plus the relevant smoke test slice
  before closeout.

## Risks

- Risk: `status` could start warning on repositories that never installed
  bootstrap assets, creating noisy false positives.
  - Mitigation: Treat absent default managed assets as "not applied" and do not
    warn until at least one managed default target is present and stale.
- Risk: Status-specific comparison logic could drift from install semantics.
  - Mitigation: Put the comparison behind shared bootstrap code and reuse the
    same managed markers and packaged rendering paths.
- Risk: Adding bootstrap health warnings could crowd out workflow guidance or
  feel like blocking debt.
  - Mitigation: Surface the reminder only while `idle`, make the warning
    explicitly non-blocking, and keep the next action optional and concise.

## Validation Summary

- `go test ./internal/install ./internal/status -count=1` passed after the
  shared detector and idle-only status reminder landed.
- `go test ./internal/status ./tests/smoke -run 'TestStatusIdleSurfaces|TestStatusIdleSkips|TestStatusActive|TestStatusReportsIdleWorkspace|TestStatusIdleReports' -count=1`
  passed after adding isolated stale-surface and active-execution
  non-regression coverage.
- Reviewer follow-up validation in `review-003-full` reported passing focused
  `go test` slices for `./internal/install`, `./internal/status`, and
  `./tests/smoke`.

## Review Summary

- `review-001-full` found 1 blocking and 1 non-blocking coverage finding in
  the `tests` slot; the blocking gap was missing isolated stale-surface
  coverage, and the non-blocking note asked for broader active-path
  non-regression.
- `review-002-delta` reran the `tests` slot against the coverage repair and
  passed cleanly.
- `review-003-full` reran `correctness`, `tests`, and `agent_ux`; the full
  candidate passed with 0 findings after the repair.

## Archive Summary

- Archived At: 2026-04-12T14:01:21+08:00
- Revision: 1
- PR: Not opened yet; this candidate is still branch-local and needs publish
  handoff after archive.
- Ready: The candidate has a clean finalize review (`review-003-full`) after
  one blocking review round and a narrow repair follow-up.
- Merge Handoff: Archive the plan, commit the tracked archive move plus summary
  updates, push `codex/idle-bootstrap-drift-status`, record publish/CI/sync
  evidence, and wait for merge approval once `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added a shared install-layer detector for stale default repo bootstrap assets
  that reuses the same managed block and managed skill rules as `harness init`.
- Taught `harness status` to surface an idle-only, non-blocking reminder plus
  optional `harness init --dry-run` guidance when the default repo `AGENTS.md`
  block or managed skills are stale.
- Added focused install/status unit coverage and smoke coverage for absent,
  fresh, combined-stale, instructions-only stale, skills-only stale, and
  active-path non-regression scenarios.
- Updated the bootstrap and CLI specs so the idle reminder contract is
  documented as optional, agent-facing, and non-blocking.

### Not Delivered

- No support was added for detecting drift in custom bootstrap override paths
  installed with explicit `--dir` or `--file` values.

### Follow-Up Issues

Deferred roadmap items remain intentionally out of scope for this slice:
status inspection for explicit override bootstrap targets and richer bootstrap
inspection or repair commands beyond the idle reminder and ordinary `init`
refresh flow.
