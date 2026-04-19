---
template_version: 0.2.0
created_at: "2026-04-19T10:43:16+08:00"
approved_at: "2026-04-19T10:46:58+08:00"
source_type: issue
source_refs:
    - '#162'
size: XS
---

# Define the machine-local watchlist contract

## Goal

Define the first machine-local watchlist contract that future dashboard and
registration work can build on without reopening basic storage questions. The
result should let a cold reader point to one user-private watchlist location,
understand the minimal persisted record shape, and see which read-model or UI
fields are intentionally derived instead of stored.

This slice should keep the contract narrow. The watchlist should track
`git-backed workspaces`, not only linked git worktrees, so direct work in a
repository's primary checkout remains a first-class case. The contract should
also stay machine-local: the initial grouping model should treat a local
repository family (primary checkout plus any linked worktrees) as the natural
UI grouping unit without trying to merge separate clones that merely share the
same remote.

## Scope

### In Scope

- Define the machine-local, user-private storage location for the watchlist.
- Define the minimal persisted record for watched `git-backed workspaces`,
  including the durable identity needed for local use.
- Define the terminology split between watched `workspace` records and
  repository-family grouping derived at read time.
- Define which dashboard-facing facts are derived on read, including local
  repository grouping and linked-worktree classification.
- Define the smallest durable metadata needed for remembered-set behavior and
  future stale or missing handling without turning the watchlist into a broad
  read model.
- Define the first membership action boundary so `unwatch` is the one explicit
  way to remove a workspace from the remembered set.

### Out of Scope

- Implementing watchlist writes, registration, or migration behavior.
- Building the dashboard UI or read model beyond the contract detail needed to
  explain persisted-versus-derived fields.
- Supporting non-git directories as watched items in this initial contract.
- Defining remote-URL-based project grouping or cross-clone deduplication.
- Choosing daemon versus on-demand backend architecture beyond what the
  storage contract must assume.

## Acceptance Criteria

- [x] A tracked spec names one machine-local, user-private watchlist location
      and one minimal persisted schema for watched `git-backed workspaces`.
- [x] The contract explicitly states that a watched `workspace` may be either
      a repository's primary checkout or a linked git worktree, and does not
      require linked-worktree-specific metadata for identity.
- [x] The contract distinguishes persisted workspace fields from derived
      dashboard fields, including local repository-family grouping and branch
      or linked-worktree classification.
- [x] The contract defines a minimal persisted workspace record that now
      includes durable time metadata needed for remembered-set behavior and
      later stale handling.
- [x] The contract explicitly defines `unwatch` as the one membership-removal
      action and keeps `active`, `completed`, and `missing` as derived states
      rather than persisted transitions.
- [x] The new documentation is easy to discover from the existing specs index,
      and the tracked plan lints cleanly.

## Deferred Items

- Automatic workspace registration behavior on `harness status` or any other
  command.
- The v1 backend shape for reading or serving the watchlist.
- Read-model details for aggregating live status across watched workspaces.
- Any future stale-item cleanup or GC policy beyond explicit `unwatch`.
- Any later concept of project grouping that merges distinct local clones by
  remote identity.

## Work Breakdown

### Step 1: Define the minimal watched-workspace contract

- Done: [x]

#### Objective

Write the normative contract for the minimal machine-local watchlist record
and its identity model.

#### Details

The contract should intentionally shift the vocabulary from "watched worktree"
to "watched workspace" where that broader term is more accurate. It should
state that the watchlist only covers git-backed workspaces for now, so both a
repository's primary checkout and any linked git worktrees qualify. Identity
should stay machine-local and path-oriented enough for an XS slice: the spec
should define the minimal persisted fields and explain that repository-family
grouping, branch display, and linked-worktree classification are derived at
read time rather than persisted as primary identity.

#### Expected Files

- `docs/specs/watchlist-contract.md`

#### Validation

- A cold reader can identify the watchlist file location and record shape from
  the spec alone.
- The spec makes the git-backed prerequisite explicit without narrowing the
  surface to linked worktrees only.

#### Execution Notes

Added `docs/specs/watchlist-contract.md` as the new normative contract for the
first machine-local watchlist slice. The spec deliberately narrows the watched
unit to `git-backed workspace` so both a repository's primary checkout and any
linked worktrees are first-class cases. It fixes one storage location at
`~/.easyharness/watchlist.json`, defines a minimal JSON file shape with
`version` plus `workspaces[].workspace_path`, and makes canonical absolute path
the initial machine-local identity.

The spec also records the accepted boundary between persisted and derived
fields: repository-family grouping, branch display, and linked-worktree
classification are derived from live Git state instead of stored in the record
itself. This revision reopened the candidate to replace the earlier
`hidden`-style direction with a simpler membership model: the main watchlist
record now carries `watched_at` and `last_seen_at`, while `unwatch` becomes
the single explicit membership-removal action and `active` / `completed` /
`missing` stay derived. This is a docs-only contract change, so no
Red/Green/Refactor loop was needed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 is a documentation-only contract definition and
will receive branch-level review during the ordinary execute closeout flow.

### Step 2: Publish the contract in the docs index and align issue-facing wording

- Done: [x]

#### Objective

Make the new contract easy to discover and ensure the repository-visible
wording matches the accepted discovery decisions.

#### Details

Add the new watchlist contract doc to the specs index, and update any nearby
prose that still frames the minimal contract as linked-worktree-only if that
wording would mislead future readers. Keep the edits narrow: this step is for
discoverability and terminology alignment, not for expanding the feature
scope.

#### Expected Files

- `docs/specs/index.md`
- `docs/specs/watchlist-contract.md`
- another nearby docs file only if minimal wording alignment is needed

#### Validation

- The specs index points clearly to the watchlist contract.
- Terminology stays consistent with the planned model of git-backed
  workspaces, derived repository-family grouping, and deferred dashboard-only
  state.

#### Execution Notes

Updated `docs/specs/index.md` to publish the new watchlist contract alongside
the existing normative specs so future agents can discover it without issue or
chat context. Re-read the nearby spec and proposal surfaces after the index
update and did not find any additional wording that needed immediate alignment:
the new contract already carries the accepted `git-backed workspace`,
repository-family grouping, explicit `unwatch`, and the minimal
`watched_at` / `last_seen_at` metadata direction directly.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This discoverability update is part of the same small
docs slice and will be checked in the normal branch-level review round.

## Validation Strategy

- Re-read the finished contract as if the discovery chat were unavailable and
  verify that the key decisions are self-contained in tracked docs.
- Run `harness plan lint` on this plan before approval and again after plan
  updates during execution.
- During execution, validate the wording against issue `#162` and the umbrella
  watchlist issue so the spec stays aligned with the accepted product shape
  without overcommitting to later implementation details.

## Risks

- Risk: The contract could accidentally overfit to linked git worktrees and
  exclude direct work in a repository's primary checkout.
  - Mitigation: Make `git-backed workspace` the persisted unit explicitly, and
    treat linked-worktree status as derived metadata rather than the base
    identity.
- Risk: The contract could mix UI-only concerns such as `hidden` into the core
  storage schema, making later read-model work heavier than needed.
  - Mitigation: Avoid a separate `hidden` layer in the first contract, keep
    one explicit `unwatch` action instead, and leave lifecycle buckets such as
    `active`, `completed`, and `missing` as derived read-model states.
- Risk: Repository grouping could become ambiguous if the contract tries to
  solve remote-based project identity too early.
  - Mitigation: Limit the initial grouping story to local repository families
    and defer cross-clone grouping to a later explicit design slice.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-19-define-machine-local-watchlist-contract.md`
  passed before approval, after the initial closeout notes were filled in, and
  again after the revision-2 and revision-3 repairs reopened the candidate.
- Direct reread of `docs/specs/watchlist-contract.md` confirmed the tracked
  spec itself carries the machine-local file location, the minimal persisted
  schema, the path-based identity model, the derived repository-family
  grouping boundary, and the remembered-set semantics for watched workspaces.
- Direct reread after revision 2 confirmed the contract now also fixes the
  missing foundation-level invariants around path normalization and
  uniqueness, degraded retention for missing or unreadable workspaces,
  crash-safe/concurrent write expectations, and no-silent-drop behavior.
- Direct reread after revision 3 confirmed the persisted record now includes
  `watched_at` and `last_seen_at`, the contract uses explicit `unwatch`
  instead of a separate `hidden` state, and `active` / `completed` /
  `missing` remain derived read-time states.
- Direct reread of `docs/specs/index.md` confirmed the new contract is
  discoverable from the existing specs index without requiring issue or chat
  context.
- `review-001-full` passed with 0 findings across the `correctness` and
  `docs_consistency` dimensions.
- `review-002-delta` passed with 0 findings across the same dimensions after
  the narrow revision-2 contract repair.
- `review-003-delta` passed with 1 non-blocking docs-consistency finding, then
  prompted this narrow revision-3 summary refresh so the plan matches the
  latest contract direction.
- `review-004-delta` passed with 1 non-blocking docs-consistency finding, then
  prompted one final execution-note wording fix so the tracked plan no longer
  uses residual hidden-era language.
- `review-005-delta` passed with 0 findings after that last wording repair.

## Review Summary

- `review-001-full`: finalize review passed with 0 findings across the
  `correctness` and `docs_consistency` dimensions.
- `review-002-delta`: reopen follow-up review passed with 0 findings after the
  narrow contract repair strengthened invariants without expanding the
  persisted schema.
- `review-003-delta`: reopen follow-up review passed with 1 non-blocking
  docs-consistency finding asking the active plan summaries to catch up with
  the revision-3 `unwatch` plus `watched_at` / `last_seen_at` contract
  direction.
- `review-004-delta`: reopen follow-up review passed with 1 non-blocking
  docs-consistency finding pointing out one last hidden-era sentence in the
  execution notes.
- `review-005-delta`: final docs-consistency follow-up passed with 0 findings
  after the execution-note wording repair.

## Archive Summary

- Archived At: 2026-04-19T16:29:56+08:00
- Revision: 3
- PR: https://github.com/catu-ai/easyharness/pull/184
- Ready: Revision 3 keeps the remembered-set contract intentionally small
  while adding the minimal durable time metadata (`watched_at`,
  `last_seen_at`), replacing `hidden` with explicit `unwatch`, and keeping
  `active` / `completed` / `missing` derived; the earlier non-blocking
  docs-consistency findings have been repaired and `review-005-delta` passed
  clean.
- Merge Handoff: Re-archive the candidate, commit the tracked re-archive move
  plus the revision-3 contract refinements on `codex/define-watchlist-contract`,
  push the branch to refresh PR #184, and refresh publish/CI/sync evidence
  until `harness status` returns to `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added `docs/specs/watchlist-contract.md` as the normative machine-local
  watchlist contract for the first watchlist foundation slice.
- Defined the watched unit as a `git-backed workspace`, explicitly covering
  both a repository's primary checkout and linked git worktrees.
- Fixed one machine-local, user-private file location at
  `~/.easyharness/watchlist.json` and one minimal persisted JSON shape with
  `version`, `workspace_path`, `watched_at`, and `last_seen_at`.
- Defined canonical absolute `workspace_path` as the first machine-local
  identity and kept repository-family grouping, branch, and linked-worktree
  classification as derived read-time facts.
- Published the new contract in `docs/specs/index.md` so future agents can
  discover it without relying on issue or chat history.
- Strengthened the foundation-level contract for follow-up watchlist issues by
  defining normalization and uniqueness expectations for `workspace_path`,
  explicit retention of missing or unreadable watched workspaces, crash-safe /
  concurrent-write integrity expectations for future writers, and one explicit
  membership-removal action (`unwatch`) instead of a separate `hidden` state.

### Not Delivered

- No watchlist write path, dashboard read model, backend implementation shape,
  or UI behavior was implemented in this slice.
- No automatic GC policy or stale-item cleanup flow was implemented in this
  slice beyond the contract hooks created by `last_seen_at` and explicit
  `unwatch`.

### Follow-Up Issues

- `#163` Decide the v1 backend shape for the watchlist dashboard.
- `#164` Silently register worktrees in the watchlist on harness status.
- `#165` Build a watchlist-backed dashboard read model.
- `#166` Define completed and unwatch lifecycle for watched worktrees.
- `#167` Ship a minimal watchlist dashboard UI.
