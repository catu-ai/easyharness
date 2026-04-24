---
template_version: 0.2.0
created_at: "2026-04-23T23:42:00+08:00"
approved_at: "2026-04-23T23:44:06+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/167
size: M
---

# Implement Watchlist Dashboard Home And Workspace Routing

<!-- This plan uses supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/
to carry the accepted dashboard visual reference and design comments that
future execution should treat as part of the approved plan package. Before
archive, any repository-facing contract changes must be absorbed into tracked
docs or code so the supplement remains backup context rather than a hidden
dependency. -->

## Goal

Turn the machine-local watchlist read model into a usable dashboard product
surface. This slice should add the `harness dashboard` entrypoint, implement
the dashboard home at `/dashboard`, support workspace detail routing under
`/workspace/<workspace_key>/...`, and let the user remove watchlist membership
with `Unwatch` without mutating harness workflow state.

The result should feel like the home/explorer for the same product family as
the existing workspace workbench, not a separate SaaS-style dashboard. The
dashboard home should be a flat recency-sorted list that emphasizes
workspace/folder name, long plan title, current state, and a compact progress
signal while preserving straightforward navigation into the existing dense
single-workspace workbench.

## Scope

### In Scope

- Add a top-level `harness dashboard` command that starts the local UI server
  and opens the machine-local dashboard home at `/dashboard`.
- Redirect `/` to `/dashboard`.
- Implement workspace detail routes under
  `/workspace/<workspace_key>/status|plan|timeline|review` and redirect bare
  `/workspace/<workspace_key>` to `/workspace/<workspace_key>/status`.
- Keep `harness ui` as a compatibility entrypoint in this slice without
  printing a deprecation warning; it should continue to open the current
  workspace, but through the new dashboard-owned workspace route.
- Ensure `harness ui` can reach the current workspace detail by touching the
  current workdir into the machine-local watchlist before opening the route.
- Build the dashboard home as one recency-sorted stacked list using
  `last_seen_at`, not lifecycle-grouped sections.
- Reuse the existing workspace workbench shell for readable watched workspaces
  and add a `Home` affordance in the rail that returns to `/dashboard`.
- Render missing, invalid, and not-currently-watched workspace routes through
  one intentionally minimal degraded page with a return path to the dashboard.
- Wire `Unwatch` as the one dashboard-local write action and keep it scoped to
  watchlist membership removal only.
- Align tracked docs with the accepted home-layout, routing, compatibility, and
  `Unwatch` behavior changes where current wording still implies grouped home
  sections or a deprecation warning in this slice.
- Persist the accepted dashboard design reference and comments in the matching
  tracked supplements package so future execution does not depend on discovery
  chat.

### Out of Scope

- Search, filters, saved views, or any other dashboard browsing controls beyond
  recency order.
- Notifications, background monitoring, filesystem watching, or daemon/server
  reuse across command runs.
- Cross-worktree write actions other than explicit `Unwatch`.
- A richer degraded workspace experience beyond a simple shared minimal page.
- Mobile-specific redesign work beyond choosing a dashboard item structure that
  can collapse vertically without redesigning the whole page.
- Reworking the existing status/plan/timeline/review content model inside the
  workspace workbench.
- New progress semantics such as percent-complete math or speculative
  session/activity counts not backed by a stable product definition.

## Acceptance Criteria

- [x] `harness dashboard` opens the UI at `/dashboard`, and `/` redirects to
      `/dashboard`.
- [x] The dashboard home renders watched workspaces as one stacked list sorted
      by `last_seen_at` descending instead of lifecycle-grouped sections.
- [x] Each dashboard item visibly prioritizes workspace/folder name, plan
      title, status, recency, and compact path/meta information using the
      accepted design direction captured in the supplements package.
- [x] Dashboard items use a fixed-width progress axis whose node count varies
      with the underlying workflow data rather than a generic fixed-slot
      template; raw workflow nodes are available from progress-node
      hover/focus text instead of always-visible card metadata.
- [x] `/workspace/<workspace_key>` redirects to
      `/workspace/<workspace_key>/status`, and readable watched workspaces load
      the existing workbench under the new route family.
- [x] The workspace-detail rail includes `Home`, which returns the user to the
      dashboard home.
- [x] `harness ui` still works in this slice, opens the current workspace via
      the new route family, and does so without printing a deprecation warning.
- [x] Missing, invalid, and unknown workspace keys share one intentionally
      minimal degraded page with a return path to the dashboard; `Unwatch`
      appears only when the workspace is still in the watchlist.
- [x] `Unwatch` removes watchlist membership only and does not call
      `harness archive`, mutate tracked plans, or change workflow state.
- [x] Tracked docs and tests capture the accepted routing, compatibility, and
      dashboard-home behavior without relying on discovery chat.

## Deferred Items

- Search, filtering, and other dashboard browsing affordances.
- Richer degraded-page content or route-specific recovery flows.
- More expressive progress-node hover/focus UI beyond the minimal baseline.
- Any stable aggregate activity metric such as concurrent sessions; this slice
  should prefer already-defined signals like warnings, review state, or step
  counts.
- A later decision on when `harness ui` should begin printing a deprecation
  warning.

## Work Breakdown

### Step 1: Align the tracked dashboard contract and preserve the design baseline

- Done: [x]

#### Objective

Update tracked docs and plan supplements so a future agent can implement the
agreed dashboard without re-reading discovery chat.

#### Details

The tracked design/package for this slice should record the accepted
differences from the older issue/spec wording: the dashboard home is a flat
recency-sorted list rather than grouped lifecycle sections, `Unwatch` is
allowed as the one dashboard-local write action, and `harness ui` stays quiet
in this slice instead of printing a deprecation warning. The supplement should
carry the approved dashboard reference image plus explicit comments describing
what to preserve, what not to literal-copy, and which earlier mockup ideas were
rejected.

#### Expected Files

- `docs/specs/proposals/harness-ui-steering-surface.md`
- `docs/specs/watchlist-contract.md`
- `docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md`
- `docs/plans/active/supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/dashboard-home-reference.png`
- `docs/plans/active/supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/dashboard-home-reference.md`

#### Validation

- Reread the updated spec sections against this plan and confirm that grouped
  home sections and an immediate `harness ui` deprecation warning are no longer
  described as required behavior for this slice.
- Run `git diff --check -- docs/specs/proposals/harness-ui-steering-surface.md docs/specs/watchlist-contract.md docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md docs/plans/active/supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/dashboard-home-reference.md`.

#### Execution Notes

Updated the tracked routing and dashboard-home contract to match the approved
slice: `docs/specs/proposals/harness-ui-steering-surface.md` now describes
`/dashboard`, the `/workspace/<workspace_key>/status|plan|timeline|review`
family, redirect behavior for `/` and bare workspace routes, and the quiet
`harness ui` compatibility path for this release. `docs/specs/watchlist-contract.md`
now records that the stable grouped read model may be flattened by the first
dashboard home into one recency-sorted list. The plan supplement package now
carries the accepted dashboard reference image plus explicit comments about the
stacked-slab layout, the fixed-width variable-density progress axis, the lack
of dashboard left rail, and the rejection of speculative `x sessions` copy.

Validated with a direct reread of the updated spec sections against the active
plan and `git diff --check -- docs/specs/proposals/harness-ui-steering-surface.md docs/specs/watchlist-contract.md docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md docs/plans/active/supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/dashboard-home-reference.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the contract and supplement edits were developed as part
of one integrated dashboard slice, so a separate Step 1 review boundary would
be artificial; the full candidate review will cover these tracked docs together
with the runtime implementation they define.

### Step 2: Add the dashboard entrypoint and route-aware UI server behavior

- Done: [x]

#### Objective

Teach the CLI and UI server to own the machine-local dashboard home, workspace
detail routes, degraded-route handling, and watchlist-backed compatibility
launches from `harness ui`.

#### Details

This step should add the new CLI entrypoint, preserve `harness ui` as a
compatibility command, and give the server enough route-aware backend behavior
to serve dashboard and per-workspace UI resources from one process. The
implementation should prefer read-time resolution from the watchlist/dashboard
model over ad hoc hardcoding, and it should keep `Unwatch` scoped to the
watchlist service. The compatibility path for `harness ui` should touch the
current workdir into the watchlist before opening the matching workspace route
so the new route family stays the single workspace-detail surface.

#### Expected Files

- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/ui/server.go`
- `internal/ui/server_test.go`
- helper files under `internal/ui/` or `internal/dashboard/` if route-key
  lookup or route resource loading needs to be shared cleanly

#### Validation

- Add or update CLI tests for `harness dashboard`, `/` redirection behavior,
  and `harness ui` compatibility routing.
- Add or update UI server tests for dashboard reads, workspace route handling,
  degraded routes, and `Unwatch` request handling.
- Run focused Go tests covering the touched CLI/UI packages.

#### Execution Notes

Added a dashboard-owned launch and routing surface across the CLI and UI
server. `internal/cli/app.go` now exposes `harness dashboard`, keeps
`harness ui` as a quiet compatibility entrypoint, touches the current workdir
into the watchlist before `harness ui` opens `/workspace/<key>/status`, and
lets tests assert the exact browser path via an injected `RunUIServer`
function. `internal/ui/server.go` now redirects `/` to `/dashboard`,
redirects bare `/workspace/<key>` routes to `/workspace/<key>/status`, serves
the SPA shell for dashboard and workspace routes, resolves watched workspaces
by key for per-workspace API reads, and scopes `POST /api/workspace/<key>/unwatch`
to watchlist membership removal. `internal/dashboard/service.go` and
`internal/contracts/dashboard.go` now expose per-workspace lookup data
including workspace name, plan title, status facts, and a truthful progress
axis derived from tracked plan structure plus finalize/await-merge nodes.

Validated with targeted unit coverage in `internal/dashboard/service_test.go`,
`internal/ui/server_test.go`, and `internal/cli/app_test.go`, plus
`go test ./internal/dashboard ./internal/ui ./internal/cli -count=1` and the
later branch-wide `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this CLI/server/runtime wiring is tightly coupled to the
frontend route ownership and is safer to review as one integrated candidate
than as an isolated pre-UI backend slice.

### Step 3: Build the dashboard home, workspace route shell changes, and minimal degraded page

- Done: [x]

#### Objective

Implement the dashboard frontend, reuse the existing workbench under the new
workspace routes, and wire the accepted minimal `Unwatch`/degraded UI.

#### Details

The dashboard home should follow the supplement comments: no left rail, no
outer section box, and no consumer-SaaS card grid. Each item should present
workspace/folder name first, then plan title, then path/meta, then a fixed
width progress axis whose node count comes from actual workflow data rather
than a fake shared slot template. Long plan titles should be allowed to read
as the main secondary content, including a controlled two-line presentation if
needed. The workspace-detail rail should gain `Home`, while the detail content
should otherwise keep the existing status/plan/timeline/review model.

For degraded and unknown workspace routes, keep the page intentionally minimal:
plain summary, route context, `Return to dashboard`, and `Unwatch` only when
the key still resolves to a watched workspace entry.

#### Expected Files

- `web/src/main.tsx`
- `web/src/types.ts`
- `web/src/helpers.ts`
- `web/src/pages.tsx`
- `web/src/workbench.tsx`
- `web/src/styles.css`
- additional `web/src/` files if the route split is cleaner with new
  components

#### Validation

- Add or update frontend tests if the repository already has stable coverage at
  this layer; otherwise ensure `pnpm --dir web build` passes.
- Run targeted server/UI tests that exercise the dashboard home, workspace
  routes, degraded page, and `Unwatch` flow end-to-end through the local UI
  server.
- If practical during execution, capture fresh screenshots proving the final
  dashboard stays aligned with the supplement reference and current workbench
  shell.

#### Execution Notes

Reworked the frontend into a dashboard-owned route model without adding a
router library. `web/src/main.tsx` now parses `/dashboard` and
`/workspace/<key>/<page>` centrally, fetches `/api/dashboard` plus
workspace-keyed resources, routes readable workspaces through the existing
workbench shell, and sends degraded or unknown keys to one minimal fallback
page. `web/src/pages.tsx`, `web/src/helpers.ts`, `web/src/types.ts`, and
`web/src/styles.css` now implement the accepted dashboard home: one recency-
sorted stacked list, no dashboard left rail, folder/workspace name first, plan
title allowed to occupy the main secondary space, path/meta underneath, a
fixed-width progress axis whose node count comes from actual workflow phase
data, raw workflow nodes kept to progress-node hover/focus text, and weak
`Open` / `Unwatch` actions. `web/src/workbench.tsx` now supports a
`Home` rail icon so workspace detail can return directly to `/dashboard`.

Step-closeout `review-001-full` then requested six repairs across correctness,
tests, and repo-facing docs. The repair extracted `useLiveResource` into a
dedicated module and resets resource state on path changes so workspace-route
navigation cannot keep rendering the previous workspace after a missing,
invalid, or unwatched lookup. Dashboard list rows now use a collision-safe
client key derived from `workspace_key`, `workspace_path`, and index. CLI
coverage now proves `harness ui` touches the current workspace into the
machine-local watchlist before opening its keyed route. UI server coverage now
exercises keyed `/plan`, `/timeline`, and `/review` reads in addition to
keyed `/status`. The web package now has targeted Vitest coverage for live-
resource path resets, dashboard progress-node rendering, degraded-page action
behavior, and collision-safe dashboard row keys. Repo-facing docs now present
`harness dashboard` as the dashboard home and `harness ui` as the quiet
current-workspace compatibility entrypoint, and `scripts/sync-contract-artifacts`
was rerun so the generated dashboard schema stays in sync with the updated Go
contract.

Validated with `go test ./internal/dashboard ./internal/ui ./internal/cli -count=1`,
`pnpm --dir web test`, `pnpm --dir web build`,
`scripts/sync-contract-artifacts --check`, and `go test ./... -count=1`.

Delta `review-002-delta` then found one remaining correctness gap: dashboard
collision rows rendered distinctly, but `Unwatch` still targeted only
`/api/workspace/<workspace_key>/unwatch`, so a colliding key could remove a
different watched workspace than the row the user clicked. The repair now
posts the exact `workspace_path` from the selected row, and the server now
resolves unwatch targets fail-safe: when a request omits `workspace_path`, a
colliding key returns an ambiguity error instead of removing an arbitrary
match. Added targeted server coverage for explicit-path unwatchs and helper
coverage for the ambiguous collision case. Revalidated with
`go test ./internal/ui ./internal/cli -count=1`, `pnpm --dir web test`, and
`pnpm --dir web build`.

Delta `review-003-delta` then found two final follow-ups: ambiguous degraded
workspace routes still exposed a non-fail-safe `Unwatch` action, and the web
test suite still lacked coverage that the client sends the selected row's
exact `workspace_path` in the unwatch request body. The repair extracted
explicit workspace-action helpers, made degraded routes hide `Unwatch` when
the workspace invalid reason is `route_key_collision`, and added targeted web
tests for both the request shape and the degraded-route fail-safe behavior.
Revalidated with `pnpm --dir web test`, `pnpm --dir web build`, and
`go test ./internal/ui ./internal/cli -count=1`.

#### Review Notes

`review-001-full` requested six blocking findings across correctness, tests,
and docs consistency: stale workspace-route data could survive key changes,
dashboard collision rows used duplicate client keys, the `harness ui`
watchlist-touch compatibility contract and keyed workspace resource reads
lacked direct coverage, frontend-only degraded/progress behavior had no
targeted tests, and repo-facing docs still described `harness ui` as the
primary UI surface instead of introducing `harness dashboard`.

Those repairs landed and `review-002-delta` then narrowed the remaining gap to
collision-safe unwatch targeting. That repair landed, and `review-003-delta`
then found two final follow-ups: ambiguous degraded routes still exposed
`Unwatch`, and the frontend still lacked direct coverage for the explicit
`workspace_path` request body. Those final repairs landed through
`0dbc9cb`, and `review-004-delta` passed clean with no remaining findings for
this step.

## Validation Strategy

- Lint the tracked plan before approval.
- Recheck the plan plus supplement package as if the chat history were gone and
  confirm a future agent could implement the accepted UX from repository files
  alone.
- During execution, keep validation focused on CLI/UI server tests, dashboard
  route coverage, and a passing `pnpm --dir web build`.
- Before archive, confirm any accepted routing or home-layout contract changes
  have been written back into tracked specs so the supplement remains a design
  aid rather than a hidden dependency.

## Risks

- Risk: The dashboard could drift into a separate visual product and stop
  feeling like the home of the existing workbench.
  - Mitigation: Keep the accepted reference image and comments in the approved
    supplement package, and align any necessary spec wording with that
    baseline.
- Risk: The progress axis could become a misleading fake template rather than a
  truthful rendering of workflow structure.
  - Mitigation: Treat the fixed axis width as a layout constraint only; derive
    node count from actual workflow/plan data and avoid invented percent or
    session math.
- Risk: `harness ui` compatibility could accidentally create a second
  workspace-detail product surface.
  - Mitigation: Route both `harness dashboard` and `harness ui` into the same
    `/workspace/<workspace_key>/...` family and test that compatibility path
    explicitly.

## Validation Summary

- `git diff --check -- docs/specs/proposals/harness-ui-steering-surface.md docs/specs/watchlist-contract.md docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md docs/plans/active/supplements/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing/dashboard-home-reference.md`
- `harness plan lint docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md`
- `go test ./internal/dashboard ./internal/ui ./internal/cli -count=1`
- `pnpm --dir web test`
- `pnpm --dir web build`
- `scripts/sync-contract-artifacts`
- `scripts/sync-contract-artifacts --check`
- `go test ./... -count=1`
- finalize-fix validation:
  - `go test ./internal/dashboard -count=1`
  - `go test ./internal/ui ./internal/cli -count=1`
  - `pnpm --dir web test`
  - `pnpm --dir web build`
- Reopen revision 2 validation:
  - `git diff --check`
  - `go test ./internal/dashboard ./internal/ui ./internal/cli -count=1`
  - `pnpm --dir web test`
  - `pnpm --dir web build`
  - `scripts/sync-contract-artifacts`
  - `scripts/sync-contract-artifacts --check`
  - `harness plan lint docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md`
  - `go test ./... -count=1`
  - Playwright visual/behavior check against `go run ./cmd/harness dashboard --no-open --port 58417`: desktop screenshot at `output/playwright/dashboard-fix-desktop.png`, mobile screenshot at `output/playwright/dashboard-fix-mobile.png`, and a runtime scroll probe confirming `.dashboard-stage` scrolls with `overflow: auto` while desktop `body` remains `overflow: hidden`.
  - post-review focusability repair: `pnpm --dir web test`, `pnpm --dir web build`, `git diff --check`, and `harness plan lint docs/plans/active/2026-04-23-implement-watchlist-dashboard-home-and-workspace-routing.md`
- Reopen revision 3 validation:
  - `pnpm --dir web test`
  - `pnpm --dir web build`
  - Playwright check against `go run ./cmd/harness dashboard --no-open --port 58419` confirmed the custom progress tooltip renders visible text on focus, first/last progress-node tooltip text stays within a 390px viewport, and the dashboard does not gain horizontal overflow.
  - Playwright scroll checks confirmed desktop scrolling moves `.dashboard-stage.scrollTop` while `body` remains hidden, and mobile-width scrolling moves `window.scrollY` while `.dashboard-stage` is `overflow: visible`.
  - `scripts/install-dev-harness`
  - Follow-up scroll recheck against `go run ./cmd/harness dashboard --no-open --port 58421` confirmed the current bundle keeps desktop scrolling on `.dashboard-stage` (`scrollTop=620`, `windowY=0`) and mobile-width scrolling on the document (`windowY=620`, `stageScrollTop=0`) with no horizontal overflow (`scrollWidth == clientWidth` in both layouts).
- Reopen revision 4 validation:
  - `pnpm --dir web test`
  - `pnpm --dir web build`
  - `git diff --check`
  - Playwright check against `go run ./cmd/harness dashboard --no-open --port 58422` confirmed short progress tooltip text is centered near the hovered current node, long edge tooltip text is clamped to the progress-axis boundary without horizontal overflow, and the workspace rail `Home` button now computes to the same transparent, borderless rail-item styling as the link items.
  - `scripts/install-dev-harness`
  - `go test ./internal/ui ./internal/cli -count=1`

## Review Summary

- `review-001-full` requested six blocking findings across correctness, tests,
  and docs consistency for stale route data, collision row keys, missing
  keyed-route and compatibility coverage, missing frontend behavior coverage,
  and stale repo-facing CLI/UI docs.
- `review-002-delta` narrowed the remaining gap to collision-safe unwatch
  targeting.
- `review-003-delta` found two final step-closeout follow-ups: ambiguous
  degraded routes still exposed `Unwatch`, and the frontend lacked direct
  request-body coverage for the explicit `workspace_path` unwatch payload.
- `review-004-delta` passed clean, closing Step 3 review.
- Finalize `review-005-full` found two blocking findings: degraded workspace
  routes surfaced a misleading generic success summary, and README still
  described the dashboard surface as read-only despite shipping the
  dashboard-local `Unwatch` action.
- `review-006-delta` passed clean after reusing degraded workspace summaries at
  the route level and updating README to describe the UI as workflow-safe with
  one dashboard-local write action.
- `review-007-delta` passed clean after the narrowed finalize-summary update
  checked the acceptance criteria, validation/review/outcome summaries, and
  follow-up pointer to `#156`.
- `review-008-delta` passed clean after structuring the Archive Summary into
  explicit `PR`, `Ready`, and `Merge Handoff` lines that `harness archive`
  expects.
- Finalize `review-009-full` then requested one last docs-consistency repair
  because the closeout narrative omitted the clean pass from
  `review-008-delta`.
- After the merge-ready candidate was reopened for revision 2, the current
  finalize-fix repaired dashboard scroll containment, removed the redundant
  dashboard page header, moved raw `current_node` details into progress-node
  hover/focus text, and split the dashboard progress axis into finer workflow
  phase nodes.
- `review-011-delta` requested one accessibility repair: progress-node raw
  labels were still hover-only because the dots were non-focusable spans.
  The follow-up made each progress node keyboard focusable with an accessible
  label and added frontend coverage for the focusable node contract.
- `review-012-delta` passed clean after verifying the progress-node
  focusability repair closed the `review-011-delta` finding without regressing
  the dashboard card layout or visible metadata contract.
- Revision 3 reopened after human browser testing showed the raw node text was
  still not visibly appearing on hover in the running app. The repair replaced
  reliance on browser-native `title` behavior with a custom CSS tooltip backed
  by each progress node's `data-label`, shown on hover and focus.
- `review-013-delta` confirmed the custom tooltip appeared, then requested an
  edge-placement repair because first/last node tooltips could clip offscreen
  and create mobile horizontal overflow. The follow-up moved the tooltip to an
  axis-level `role="tooltip"` element constrained to the progress axis, and
  also limited dashboard-stage scroll containment to desktop layouts so mobile
  scroll chains to the document instead of being swallowed by a non-scrollable
  stage.
- `review-014-delta` passed clean after rechecking the axis-level tooltip,
  edge-node containment, and split desktop/mobile scroll behavior. A final
  Playwright scroll probe against the current bundle confirmed desktop wheel
  input scrolls the dashboard stage and mobile-width wheel input scrolls the
  document without introducing horizontal overflow.
- Revision 4 reopened after human visual feedback questioned the split scroll
  model, noted that tooltip placement still felt too far left for short labels,
  and caught the workspace rail `Home` button using the browser's default gray
  button surface. The repair documents the scroll model in CSS, changes
  progress tooltips to follow the hovered/focused node while clamping inside
  the axis for long labels, and resets rail item button styling to match the
  link items.
- `review-015-delta` passed clean after checking the follow-hover tooltip
  placement, edge/long-label clamping, scroll-model documentation, and rail
  `Home` button styling reset.

## Archive Summary

- Archived At: 2026-04-24T21:22:18+08:00
- Revision: 4
- PR: https://github.com/catu-ai/easyharness/pull/191
- Ready: Acceptance criteria remain satisfied after the revision 2 UI feedback
  repair and the revision 3 custom tooltip fix. Step 3 closeout passed through
  `review-004-delta`; finalize follow-ups through `review-010-full` produced
  the original merge-ready candidate; revision 2 passed `review-012-delta`;
  revision 3 passed `review-014-delta` after the edge-tooltip and mobile-scroll
  repair; revision 4 passed `review-015-delta` after the follow-hover tooltip
  placement, rail `Home` styling reset, and documented scroll-model repair.
- Merge Handoff: Run `harness archive`, commit the tracked archive move, push
  branch `codex/issue-167-dashboard-ui`, refresh PR #191, and then record
  publish/CI/sync evidence until `harness status` returns to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added `harness dashboard`, `/dashboard`, and the dashboard-owned
  `/workspace/<workspace_key>/{status|plan|timeline|review}` route family with
  `/` and bare-workspace redirects.
- Preserved `harness ui` as a quiet compatibility entrypoint that touches the
  current workspace into the machine-local watchlist before opening the new
  workspace route family.
- Shipped the machine-local dashboard home as a recency-sorted stacked list
  with accepted design-reference supplements, progress-axis metadata, dense
  workbench continuity, and a `Home` rail action back to the dashboard.
- Added shared degraded workspace handling plus dashboard-local `Unwatch`
  behavior that removes watchlist membership without mutating tracked workflow
  state, including collision-safe fail-safe behavior.
- Updated specs, README, generated dashboard contract schema, and focused
  frontend/server/CLI validation so the shipped behavior is discoverable from
  repository files alone.
- In revision 2, tightened the dashboard home after human visual testing:
  desktop scrolling now belongs to the dashboard stage, the redundant
  `Machine-local home` / `Dashboard` content header is gone, progress nodes
  now represent finer workflow phases such as `execution/step-k/implement`,
  and raw workflow node text is available from node hover/focus affordances
  instead of always-visible card metadata. The post-review accessibility
  follow-up also makes those progress nodes keyboard focusable.
- In revision 3, replaced the unreliable browser-native `title` tooltip with a
  visible custom tooltip that appears on hover and focus, then adjusted the
  tooltip placement and scroll containment so edge nodes remain readable and
  mobile-width scrolling uses the document rather than a non-scrollable
  dashboard stage.
- In revision 4, refined the progress tooltip into a follow-hover/focus
  placement model that clamps within the progress axis for long or edge labels,
  made the rail `Home` button visually consistent with neighboring rail links,
  and documented the intended wide-vs-narrow scroll ownership in CSS.

### Not Delivered

- Search, filtering, or saved dashboard views.
- Richer degraded-page recovery flows beyond the intentionally minimal page.
- A richer progress inspector beyond minimal hover/focus text, or any stable
  aggregate activity metric such as concurrent sessions.
- A future decision on when `harness ui` should begin printing a deprecation
  warning.

### Follow-Up Issues

- Existing dashboard umbrella issue `#156` remains the durable follow-up for
  later dashboard expansion work such as search/filtering, richer degraded
  recovery flows, richer progress affordances, and future `harness ui`
  deprecation timing outside this shipped slice.
