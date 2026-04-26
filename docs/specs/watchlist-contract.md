# Watchlist Contract

## Purpose

Define the normative machine-local storage contract for the first easyharness
watchlist. This contract exists so later watchlist registration, read-model,
and dashboard work can build on one clear local storage shape instead of
reopening basic questions about what is watched, where it is stored, and which
facts belong in persisted data versus derived UI state.

This spec is intentionally narrow. It defines the minimal persisted watchlist
record for local use. It does not define write triggers, backend shape, or UI
behavior beyond the persisted-versus-derived boundary needed to keep later
slices coherent.

## Watched Unit

The watched unit is a `git-backed workspace`.

A watched workspace is a local filesystem checkout that:

- is backed by a Git working tree
- is intended to run easyharness in that checkout

This definition includes both:

- a repository's primary checkout
- a linked checkout created through `git worktree`

The watchlist contract must not assume that every watched workspace is a
linked worktree. Direct work in a repository's primary checkout is a
first-class case.

This first contract does not support non-git directories as watched items.

## Storage Location

The watchlist file lives at:

- `~/.easyharness/watchlist.json`

If `EASYHARNESS_HOME` is set to a non-empty path:

- an absolute value places the watchlist at
  `$EASYHARNESS_HOME/watchlist.json`
- a relative value is resolved under the current user's home directory before
  writing, so `relative-home` means
  `~/relative-home/watchlist.json`
- a relative value that would escape above the user's home directory, such as
  `../escape`, is invalid and must be rejected

This location is machine-local and user-private. It is not a repository-shared
artifact and must not be written into the repository itself.

The parent directory may later hold other machine-local easyharness files, but
this contract defines only the watchlist file above.

## File Shape

The watchlist file is UTF-8 JSON with one top-level object:

```json
{
  "version": 1,
  "workspaces": [
    {
      "workspace_path": "/absolute/path/to/workspace",
      "watched_at": "2026-04-19T03:00:00Z",
      "last_seen_at": "2026-04-19T03:00:00Z"
    }
  ]
}
```

Contract:

- `version` is required and identifies the watchlist file format version
- `workspaces` is required and contains zero or more watched workspace records
- each `workspace_path` is required
- each `watched_at` is required
- each `last_seen_at` is required
- each `workspace_path` must be an absolute canonical local filesystem path
- `watched_at` records when the workspace first entered the watchlist
- `last_seen_at` records the latest successful watchlist-touching harness
  command that confirmed this workspace locally
- `last_seen_at` is the dashboard's recency signal for ordering watched
  workspaces on the machine-local home page
- duplicate `workspace_path` values are invalid

The minimal persisted workspace record is intentionally small:

- `workspace_path`
- `watched_at`
- `last_seen_at`

This contract does not require any additional persisted per-workspace fields in
the first slice.

Successful core harness workflow commands may refresh `last_seen_at`, not just
explicit watchlist-management commands. The exact command list is an
implementation detail, but the intended shape for the dashboard is that
routine successful workflow confirmations such as `harness status`,
`harness review start`, or lifecycle/evidence commands can keep the recency
signal fresh when they pass through one shared watchlist writer.

Watchlist refresh is a CLI command success side effect, not a read-model side
effect. UI/API/dashboard resource reads, including status polling, must not
touch `watchlist.json` or inflate `last_seen_at`. The dashboard may read the
watchlist and derive workspace status at request time, but recency changes
belong to explicit watchlist-management actions or successful core CLI
workflow commands.

## Path Normalization and Uniqueness

Writers must normalize `workspace_path` before comparing or persisting records.

The normalization contract for this first slice is:

- resolve the path to an absolute local filesystem path before writing
- use one canonical textual path per watched workspace for duplicate detection
- treat the normalized `workspace_path` as the uniqueness key in
  `workspaces[]`

This spec intentionally does not fix every platform-specific normalization
detail yet. Later implementation may need to clarify symlink or case-folding
rules per platform, but it must still preserve one clear rule: repeated
registration of the same local workspace must converge on one canonical
`workspace_path` record rather than creating duplicates. Repeated touch of the
same workspace may refresh `last_seen_at`, but it should preserve the original
`watched_at`.

## Identity Model

For this first machine-local contract, watched-workspace identity is the
canonical absolute `workspace_path`.

This choice is intentionally local and path-oriented:

- it keeps the persisted record small enough for an XS foundation slice
- it supports both primary checkouts and linked worktrees with one model
- it avoids introducing synthetic IDs before later read-model or write-path
  work proves they are necessary

If a workspace moves to a different path, that is a different watched
workspace under this initial contract. The first contract does not attempt to
preserve identity across path moves.

## Dashboard Route Key

The dashboard may expose watched workspaces through a route family such as:

- `/workspace/<workspace_key>`

For v1, `workspace_key` is a read-time derived value, not a persisted
watchlist field.

Contract:

- the route key must be derived deterministically from canonical
  `workspace_path`
- the route key must be opaque enough that the dashboard does not need to
  expose raw absolute paths in URLs
- the watchlist file must not grow a separate persisted route-only
  `workspace_id` field for this first slice
- readers must be able to resolve `workspace_key` back to a watched workspace
  by rereading the current watchlist and deriving keys again from canonical
  `workspace_path`
- if a reader encounters a route-key collision, it must surface an explicit
  error rather than silently choosing one workspace

The exact derivation algorithm is an implementation detail for later work so
long as it remains deterministic for the same canonical `workspace_path`
within a given implementation revision.

## Missing or Unreadable Workspaces

The watchlist is a remembered local set, not a best-effort snapshot of only
currently readable directories.

If a previously watched `workspace_path` later becomes:

- missing
- unreadable
- no longer a valid Git-backed workspace

the watchlist record should remain present until explicit `unwatch` removes
it.

Read-model and UI layers should surface those entries as explicit degraded
states rather than silently dropping them from the watched set.

## Derived Repository Grouping

The watchlist persists watched workspaces, not repository groups.

Repository-family grouping is derived at read time from Git metadata. The
intended local grouping model is:

- a repository's primary checkout and any linked git worktrees belong to the
  same local repository family
- separate local clones remain separate families even when they point to the
  same remote

The exact Git probe is an implementation detail, but the grouping contract
must be consistent with local repository-family identity rather than
remote-URL-based project identity.

This lets the UI treat `workspace` as the base watched unit while still
grouping related local checkouts together.

## Persisted Versus Derived Fields

Persist only the minimal watched-workspace set in the watchlist file.

Persisted:

- `version`
- `workspaces[].workspace_path`
- `workspaces[].watched_at`
- `workspaces[].last_seen_at`

Derived at read time:

- repository root or top-level path
- local repository-family grouping key
- dashboard route key derived from canonical `workspace_path`
- branch name
- whether the workspace is the repository's primary checkout or a linked
  worktree
- whether a watched workspace currently presents as `active`, `completed`,
  `idle`, `missing`, or `invalid`
- invalid reason for degraded read failures, such as `unreadable`,
  `not_git_workspace`, or `status_error`
- live harness status or dashboard summary fields

The contract prefers deriving these facts from the current filesystem and Git
state instead of persisting copies that can drift.

## Membership and User Action

Watchlist membership is binary in this contract:

- a workspace is watched because a record exists in `watchlist.json`
- a workspace is no longer watched once that record is removed from
  `watchlist.json`

This contract does not define a separate dashboard-local `hidden` state.
There is no persisted visibility flag, archive bucket, or secondary lifecycle
layer for clearing rows from the main dashboard.

The explicit user-facing membership-removal action is `unwatch`.

`Unwatch` means:

- remove the selected workspace record from the machine-local watchlist
- preserve unrelated workspace records in the same watchlist file
- leave the watched repository, git worktree, tracked plan files, and
  `.local/harness` workflow artifacts untouched

`Unwatch` does not mean `harness archive`. It must not advance, reopen,
archive, land, or otherwise mutate harness workflow state. It also must not
delete a local checkout or git worktree.

Once a workspace is unwatched, it is absent from ordinary dashboard lifecycle
groups because it is no longer a watched entry. The first dashboard slice does
not need tombstone or history state to distinguish "never watched" from
"previously watched and later unwatched."

## Derived Lifecycle States

The first contract keeps dashboard lifecycle classification derived instead of
persisted.

The dashboard read model is a read-time projection over `watchlist.json` plus
per-workspace harness status. It must not write the watchlist or workflow
state while building dashboard entries.

Readable dashboard entries expose the raw harness `current_node` separately
from the dashboard lifecycle state. The lifecycle state is a compact
dashboard classification, not a replacement for the raw workflow node.

The dashboard lifecycle states are:

- `active`
- `completed`
- `idle`
- `missing`
- `invalid`

These are read-time states for currently watched entries, not membership
transitions stored in the watchlist file.

Lifecycle classification:

- `active`: status is readable and `current_node` is anything except `idle`
- `completed`: status is readable, `current_node` is `idle`, and status
  artifacts include last-landed context
- `idle`: status is readable, `current_node` is `idle`, and no last-landed
  context is present
- `missing`: the watched path no longer exists
- `invalid`: the watched path exists but cannot be treated as a valid readable
  harness workspace

Invalid entries carry a reason such as `unreadable`, `not_git_workspace`,
`status_error`, `malformed_path`, or `route_key_collision`. The reason refines
the `invalid` state; it does not expand the top-level lifecycle enum.

## Dashboard Read Model Payload

The dashboard read model should expose one compact result for the dashboard
home. The concrete API route is implementation-owned, but the payload boundary
should follow this shape:

- `ok`: whether the dashboard read completed without a top-level watchlist
  load failure
- `resource`: stable UI resource label such as `dashboard`
- `summary`: concise human-readable result summary
- `groups`: dashboard lifecycle groups in stable order
- `errors`: top-level watchlist or read-model errors when the watched set
  cannot be loaded at all

The `groups` field is a stable classification/result shape for API consumers.
The first dashboard home UI may flatten entries from those groups into one
recency-sorted watched-workspace list instead of rendering grouped sections,
so long as lifecycle state remains visible on each entry.

Each group contains:

- `state`: one of `active`, `completed`, `idle`, `missing`, or `invalid`
- `workspaces`: watched workspace entries in dashboard recency order

Each workspace entry contains:

- `workspace_key`: dashboard route key derived from canonical
  `workspace_path`
- `workspace_path`: canonical watched path from the watchlist record
- `watched_at`: timestamp from the watchlist record
- `last_seen_at`: timestamp from the watchlist record, used as the primary
  dashboard recency signal
- `dashboard_state`: same lifecycle value as the containing group
- `invalid_reason`: present only when `dashboard_state` is `invalid`
- `current_node`: raw harness workflow node for readable status entries
- `summary`: compact row/card summary; for readable entries this should come
  from harness status
- `next_actions`: compact pass-through of the most relevant status next
  actions for readable entries
- `warnings`, `blockers`, and `errors`: compact pass-through or degraded-entry
  diagnostics for the watched workspace
- `artifacts`: stable status artifact handles needed for dashboard navigation
  or display

Readable entries should omit `invalid_reason`. Missing entries do not have
`current_node` because there is no readable workspace status. Invalid entries
may omit `current_node` unless a partial status result produced a trustworthy
raw node before failing.

In particular:

- a harness plan moving through `archive` or back to `idle` does not remove
  the workspace from the watchlist
- ordinary idle without last-landed context remains `idle`; it must not be
  presented as `completed`
- completed watched workspaces remain visible until explicit `unwatch`
- deleting the local directory does not remove the workspace from the
  watchlist by itself; it instead becomes a `missing` watched workspace until
  explicit `unwatch` removes it
- a permissions, Git probe, or status failure may surface as `invalid` without
  removing the workspace from the watchlist
- a malformed non-absolute `workspace_path` must surface as `invalid` before
  any filesystem, Git, or status probe is attempted
- a duplicate or otherwise colliding `workspace_key` must surface explicit
  per-entry collision diagnostics rather than silently routing to one workspace

## No Automatic GC In V1

This first contract does not define silent automatic garbage collection.

The watchlist is a remembered local set, not an auto-pruned mirror of the
current filesystem. The combination of `last_seen_at`, derived `missing`
or `invalid` status, and explicit `unwatch` behavior is enough for this
contract.

Later work may add user-facing cleanup or stale-item policies, but v1 should
not silently discard watched entries just because they have gone idle,
invalid, or missing.

## Dashboard Routing Outcomes

For dashboard workspace-detail routing:

- a watched workspace that is now `missing` or `invalid` should still
  resolve to an explicit degraded workspace page rather than being silently
  dropped or redirected away
- a route key that does not match any current watched workspace should be
  treated as "not currently watched"
- the first dashboard slice does not need extra watchlist history state just
  to distinguish "never watched" from "used to be watched and later unwatched"

If a workspace has been unwatched, its old route key should resolve the same
way as any other key that does not match the current watchlist.

## Write Expectations

This spec does not define which command writes the watchlist, but any future
writer must preserve basic local integrity expectations:

- writes must not silently drop unrelated existing workspace records
- duplicate registration attempts must converge on one record per normalized
  `workspace_path`
- rewriting an existing watched record should preserve `watched_at`
- rewriting an existing watched record may refresh `last_seen_at` when the
  watchlist-touching command successfully confirms the workspace
- major core workflow commands should refresh `last_seen_at` through one
  shared watchlist writer rather than ad hoc per-command file mutation paths
- persistence should use crash-safe replacement rather than partial in-place
  writes when the file is rewritten
- concurrent write paths must avoid last-writer-wins corruption that would
  lose another workspace record

Because the watchlist is a machine-local dashboard/index aid rather than the
workflow control plane, watchlist persistence should be treated as a
best-effort side effect. A watchlist write failure should not by itself change
the success/failure result of an otherwise successful core workflow command.

The exact file-locking or mutation-coordination mechanism is an implementation
detail for later work, but these integrity expectations are part of the
watchlist contract because silent registration during commands such as
`harness status` will depend on them.

## Non-Goals

This spec does not:

- define when or how workspaces are added to the watchlist
- define daemon versus on-demand backend architecture
- define dashboard UI rendering beyond the persisted-versus-derived boundary
- merge separate local clones into one project because they share a remote
- support non-git watched directories in the first slice
- define any dashboard-local `hidden` state or secondary visibility layer
  beyond explicit `unwatch`
- define automatic cleanup for completed, idle, missing, invalid, or stale
  watched workspaces
