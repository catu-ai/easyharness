---
template_version: 0.2.0
created_at: "2026-04-19T21:05:00+08:00"
approved_at: "2026-04-19T20:42:25+08:00"
source_type: issue
source_refs:
    - '#163'
size: XS
---

# Define the dashboard entrypoint and migration architecture

## Goal

Define the v1 product and backend shape for the machine-local dashboard so
future implementation work can proceed without reopening command semantics,
routing, or migration questions. The result should tell a cold reader what
`harness dashboard` replaces, how it starts, what its default landing page is,
how workspace detail routing works, and which watchlist signal drives the
dashboard ordering model.

This slice should stay architecture-focused and narrow. It should record the
accepted dashboard behavior for `0.3.0`, the `0.3.x` deprecation path for
`harness ui`, and the minimal watchlist-touching behavior that lets the
dashboard treat `last_seen_at` as the recency signal. It should not implement
the dashboard or expand the watchlist schema just to make routing prettier.

## Scope

### In Scope

- Record `harness dashboard` as the new UI entrypoint targeted for `0.3.0`,
  including its role as the eventual replacement for `harness ui`.
- Define the startup semantics for `harness dashboard`: explicit command
  invocation, on-demand local server, no background daemon, and default landing
  route at `/dashboard`.
- Define the `0.3.x` transition behavior for `harness ui`, including a
  deprecation warning and compatibility redirect into the dashboard-owned
  workspace detail surface.
- Define the workspace route model for dashboard-owned navigation, including
  `/workspace/<workspace_key>` and the decision to derive `workspace_key` from
  canonical `workspace_path` at read time instead of persisting a new watchlist
  field.
- Define the accepted dashboard ordering and recency signal around
  `last_seen_at`, including the expectation that major harness commands refresh
  the field through a shared watchlist writer when they successfully confirm
  the current workspace locally.
- Define the degraded routing behavior for missing, unreadable, or unknown
  watched workspaces and record that `Unwatch` is the only dashboard-local
  write action in v1.

### Out of Scope

- Implementing `harness dashboard`, the watchlist read model, or the dashboard
  UI.
- Reworking the current single-workspace `status`, `plan`, `timeline`, or
  `review` page model beyond recording that the dashboard should reuse it.
- Adding a daemon, broker, singleton server discovery, push transport, file
  watching, or any other background-process architecture.
- Adding `--workspace` direct-open flags or using a raw-path URL scheme for
  workspace routes.
- Expanding the watchlist file schema with a persisted `workspace_id` or any
  other new routing-only field.
- General cross-workspace workflow mutations beyond the one explicit `Unwatch`
  action.

## Acceptance Criteria

- [x] Tracked docs record `harness dashboard` as the explicit `0.3.0` UI
      entrypoint, with on-demand local-server startup semantics and default
      landing at `/dashboard`.
- [x] Tracked docs record the `0.3.x` deprecation path for `harness ui`,
      including console warning behavior and compatibility routing into the
      dashboard-owned workspace detail surface.
- [x] Tracked docs define `/workspace/<workspace_key>` as the workspace route
      model, specify that `workspace_key` is a deterministic read-time
      derivative of canonical `workspace_path`, and reject both raw-path URLs
      and a new persisted watchlist ID for v1.
- [x] Tracked docs define `last_seen_at` as the dashboard recency signal,
      including the expectation that major successful harness commands refresh
      it through a shared watchlist writer rather than ad hoc per-command file
      rewrites.
- [x] Tracked docs describe degraded routing behavior: missing or unreadable
      watched workspaces render a simple degraded page, unknown keys are
      treated as not currently watched, and `Unwatch` is the only
      dashboard-local write action in v1.
- [x] The plan lints cleanly and the resulting documentation is self-contained
      enough that a future agent could implement the dashboard entrypoint
      without relying on discovery chat.

## Deferred Items

- The dashboard watchlist read model and any concrete API payload shape for the
  dashboard home page.
- The concrete implementation of the shared watchlist writer and its locking
  mechanism.
- Any richer workspace-home redesign beyond the existing single-workspace page
  model reused from `harness ui`.
- Route parameters, filters, search, or any future direct-open flags for
  dashboard navigation.
- Push-driven refresh, background monitoring, notifications, or any always-on
  service architecture.
- Any future reason to persist a stable workspace ID that is distinct from the
  canonical path-derived route key.

## Work Breakdown

### Step 1: Record the dashboard command and migration contract

- Done: [x]

#### Objective

Write the tracked product and command contract for `harness dashboard` and the
`harness ui` deprecation window.

#### Details

Capture the accepted product boundary from discovery so future implementation
work does not revisit whether the dashboard is a page inside `harness ui` or a
new top-level command. The docs should make clear that `harness dashboard`
becomes the new UI entrypoint in `0.3.0`, always starts with `/dashboard`,
runs only when the command is invoked, and starts one on-demand local server
per command run. The same step should define `harness ui` as a temporary
compatibility command for `0.3.x`: it prints a deprecation warning, opens the
dashboard-owned workspace detail entry for the current workspace, and does not
introduce singleton-server reuse or any daemon-like behavior.

#### Expected Files

- `docs/specs/proposals/harness-ui-steering-surface.md`
- another nearby tracked doc only if a second durable location is needed for
  the migration contract

#### Validation

- A cold reader can explain the difference between `harness dashboard` and the
  deprecated `harness ui` command from tracked docs alone.
- The docs explicitly reject daemon, singleton-server, and implicit background
  startup behavior for v1.

#### Execution Notes

Updated `docs/specs/proposals/harness-ui-steering-surface.md` to carry the
accepted `0.3.0` dashboard entrypoint direction directly in tracked docs. The
proposal now treats `harness dashboard` as the explicit machine-local UI
command, fixes `/dashboard` as the default landing route, rejects daemon or
singleton-server behavior, and defines `harness ui` as a deprecated
compatibility command through `0.3.x` that opens the dashboard-owned workspace
detail route for the current workspace.

The same proposal revision also records that the dashboard home owns
machine-local workspace selection while selected workspace detail should reuse
the existing dense `Status` / `Plan` / `Timeline` / `Review` workbench model.
This was a docs-only architecture slice, so Red/Green/Refactor TDD was not
practical or necessary; validation relied on direct reread of the tracked doc,
`git diff --check`, and later branch-level review.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 is a tracked-docs architecture update and will
receive branch-level review during the ordinary execute review flow.

### Step 2: Define workspace routing, recency, and degraded-state behavior

- Done: [x]

#### Objective

Write the tracked routing and watchlist semantics that the dashboard depends
on for workspace detail navigation.

#### Details

Record that the dashboard home owns machine-local navigation while workspace
detail should reuse the current single-workspace `status` / `plan` /
`timeline` / `review` model instead of inventing a second workspace-home
shell. Define `/workspace/<workspace_key>` as the route family, where
`workspace_key` is derived deterministically from canonical `workspace_path` at
read time so the watchlist contract can stay minimal. The docs should explain
that the dashboard orders watched workspaces by `last_seen_at`, that major
successful harness commands are expected to refresh that field through a shared
watchlist writer, and that missing or unreadable watched workspaces render a
simple degraded page with `Unwatch` as the one allowed dashboard-local action.
Unknown keys should be treated as "not currently watched" rather than as a
recoverable special case with extra history state.

#### Expected Files

- `docs/specs/watchlist-contract.md`
- `docs/specs/proposals/harness-ui-steering-surface.md`
- another nearby tracked doc only if a small discoverability or terminology
  alignment edit is needed

#### Validation

- The docs clearly separate route-level "not currently watched" handling from
  degraded watched-workspace states such as `missing` or `unreadable`.
- The chosen `workspace_key` story keeps the watchlist schema minimal and
  still lets a future implementation resolve dashboard routes from watchlist
  data alone.

#### Execution Notes

Updated `docs/specs/watchlist-contract.md` to define the dashboard-facing
watchlist semantics that issue `#163` needs before implementation. The spec
now names `last_seen_at` as the machine-local dashboard recency signal, states
that major successful harness commands may refresh it through one shared
watchlist writer, and defines `/workspace/<workspace_key>` as a route family
whose key is derived deterministically from canonical `workspace_path` at read
time instead of persisting a new route-only ID.

The same revision adds explicit degraded-state expectations: watched
workspaces may surface as `unreadable`, missing or unreadable watched entries
should still resolve to a degraded workspace page, unknown keys are treated as
"not currently watched", and `Unwatch` remains the one explicit cleanup action
instead of a broader dashboard mutation surface. This was also a docs-only
change, so no TDD loop was needed; validation relied on direct reread,
cross-checking the proposal wording, and `git diff --check`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 is a tracked-docs contract update and will be
validated through the normal reviewer round for the branch.

## Validation Strategy

- Run `harness plan lint` on this plan before approval and again after any
  plan edits during execution.
- During execution, reread the updated tracked docs as if the discovery chat
  were unavailable and confirm they carry the dashboard command semantics,
  migration path, routing model, recency signal, and degraded-state behavior
  directly.
- Cross-check the updated wording against issue `#163`, the watchlist contract,
  and the existing UI steering proposal so the accepted architecture lands in
  tracked docs instead of staying in issue or chat history alone.

## Risks

- Risk: The documentation could leave enough ambiguity that a future agent
  reopens daemon, singleton-server, or raw-path-routing options during
  implementation.
  - Mitigation: State the rejected alternatives explicitly in the dashboard
    command and route contract rather than implying them only indirectly.
- Risk: `last_seen_at` could become an underspecified signal if the docs do not
  tie it to a shared watchlist writer and successful command confirmation.
  - Mitigation: Record the refresh expectation clearly and defer the concrete
    locking mechanism as an implementation detail rather than leaving the
    signal undefined.
- Risk: Missing and unknown workspace states could blur together and produce a
  confusing detail-page experience.
  - Mitigation: Document degraded watched-workspace pages separately from
    "not currently watched" routing outcomes, and keep `Unwatch` as the one
    explicit cleanup action for degraded watched entries.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-19-define-dashboard-entrypoint-and-migration-architecture.md`
  passed before approval and again after execution notes and archive-ready
  summaries were filled in.
- `git diff --check` passed after the tracked proposal and watchlist-contract
  edits landed.
- Direct reread of `docs/specs/proposals/harness-ui-steering-surface.md`
  confirmed the tracked proposal now carries the accepted `harness dashboard`
  entrypoint, `/dashboard` default landing route, `0.3.x` deprecation path for
  `harness ui`, and the reuse of the existing workspace workbench model under
  `/workspace/<workspace_key>`.
- Direct reread of `docs/specs/watchlist-contract.md` confirmed the tracked
  contract now treats `last_seen_at` as the dashboard recency signal, defines
  the route-key derivation and collision expectations, and records degraded
  routing outcomes for missing, unreadable, and no-longer-watched workspaces.
- Cross-check against issue `#163` and the existing UI steering proposal
  confirmed that the accepted dashboard entrypoint, migration, routing, and
  `Unwatch` cleanup decisions now live in tracked docs rather than only in chat
  or issue discussion.

## Review Summary

- `review-001-full` covered the finalize candidate across the `correctness`
  and `docs_consistency` dimensions.
- Reviewer slot `correctness` submitted by `reviewer-correctness` found 0
  issues after checking the active plan, the updated proposal, the updated
  watchlist contract, and nearby spec surfaces for contract conflicts.
- Reviewer slot `docs_consistency` submitted by `reviewer-docs-consistency`
  found 0 issues after checking that the plan, proposal, and contract tell one
  coherent implementation story for dashboard entrypoint, route semantics,
  recency, and degraded workspace handling.
- `harness review aggregate --round review-001-full` passed cleanly with 0
  blocking and 0 non-blocking findings.

## Archive Summary

- Archived At: 2026-04-19T20:47:44+08:00
- Revision: 1
- PR: NONE
- Ready: Revision 1 satisfies the tracked acceptance criteria for issue
  `#163`, documents `harness dashboard` as the `0.3.0` UI entrypoint, keeps
  `harness ui` as a deprecated `0.3.x` compatibility command, reuses the
  existing workspace workbench model under `/workspace/<workspace_key>`, and
  records `last_seen_at`, degraded routing, and `Unwatch` semantics in the
  watchlist contract; `review-001-full` passed cleanly.
- Merge Handoff: Archive this candidate, commit the tracked archive move plus
  the proposal/contract updates on `codex/define-dashboard-entrypoint`, push
  the branch, open or refresh the PR, and record publish/CI/sync evidence
  until `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Updated `docs/specs/proposals/harness-ui-steering-surface.md` to define
  `harness dashboard` as the explicit `0.3.0` UI entrypoint, default
  `/dashboard` landing route, and machine-local home for watched workspaces.
- Recorded the `0.3.x` migration path where `harness ui` remains available as
  a deprecated compatibility command that opens the dashboard-owned workspace
  detail route for the current workspace.
- Documented that selected workspace detail should reuse the existing dense
  `Status`, `Plan`, `Timeline`, and `Review` workbench model instead of adding
  a second workspace-home shell.
- Extended `docs/specs/watchlist-contract.md` so `last_seen_at` is the
  dashboard recency signal and major successful harness commands are expected
  to refresh it through one shared watchlist writer.
- Defined `/workspace/<workspace_key>` as a deterministic read-time route key
  derived from canonical `workspace_path`, while explicitly rejecting raw-path
  URLs and a new persisted watchlist route ID for v1.
- Recorded degraded routing expectations for missing, unreadable, and unknown
  watched workspaces, and kept `Unwatch` as the one explicit dashboard-local
  cleanup action.

### Not Delivered

- No `harness dashboard` command, server, read model, or UI implementation was
  built in this slice.
- No concrete shared watchlist-writer implementation or locking mechanism was
  added in this slice beyond documenting the contract expectation.
- No direct-open flags, search/filter behavior, push-driven refresh, or
  broader dashboard mutation surface was added in this slice.

### Follow-Up Issues

- `#164` Silently register worktrees in the watchlist on harness status.
  This remains the main follow-up for the shared watchlist-touching write path
  that will refresh `last_seen_at` and keep the dashboard list current.
- `#165` Build a watchlist-backed dashboard read model.
  This follow-up should realize the machine-local dashboard home and the
  recency-ordered watched-workspace list documented here.
- `#167` Ship a minimal watchlist dashboard UI.
  This follow-up should implement the new `harness dashboard` entrypoint and
  the dashboard-owned workspace navigation described in this slice.
