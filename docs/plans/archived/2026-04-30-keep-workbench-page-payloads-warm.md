---
template_version: 0.2.0
created_at: "2026-04-30T00:18:22+08:00"
approved_at: "2026-04-30T00:20:14+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/205
size: XS
---

# Keep Workbench Page Payloads Warm

## Goal

Keep the last successful Plan, Timeline, and Review workbench payloads visible
across tab switches so returning to an inactive page does not briefly show an
empty or first-load state before the next refresh completes.

The intended design is a small stale-while-revalidate extension to the existing
front-end live resource hook: distinguish a resource that is invalid and should
be cleared from a resource that is still valid but paused because its page is
inactive.

## Scope

### In Scope

- Add an explicit live-resource lifecycle shape that separates resource identity
  from refresh activity.
- Keep inactive Plan, Timeline, and Review resource data in memory while their
  workspace remains the same and readable.
- Refresh a retained page payload in the background when the user returns to
  that page.
- Preserve stale/error freshness semantics when a background refresh fails.
- Reset retained page payloads when the resource identity changes, including
  workspace changes.
- Add focused hook and workbench integration tests for the no-empty-flicker
  behavior.

### Out of Scope

- Introducing TanStack Query or another general query/cache library.
- Adding cross-resource cache maps or persistence across reloads, browser tabs,
  or sessions.
- Keeping inactive tabs mounted in the DOM.
- Polling inactive Plan, Timeline, or Review pages.
- Redesigning the full workbench live-refresh policy or freshness UI.
- Changing backend API contracts.

## Acceptance Criteria

- [x] Plan, Timeline, and Review keep their last successful payload visible
      while switching away and back within the same workspace.
- [x] Returning to a retained page triggers an immediate background refresh
      without clearing the existing payload.
- [x] If the background refresh fails after a successful payload exists, the old
      payload remains visible and the resource reports a stale/error state.
- [x] First-load failure without any retained payload still renders the existing
      disconnected/empty behavior.
- [x] Changing workspace or resource identity clears retained Plan, Timeline,
      and Review payloads so data cannot bleed between workspaces.
- [x] Inactive Plan, Timeline, and Review resources do not keep polling.
- [x] Tests cover the hook lifecycle and the App-level tab-switch behavior.

## Deferred Items

- Broader workbench live-refresh transport changes such as websocket or SSE
  updates.
- Persistence of workbench page payloads across browser reloads or new tabs.

## Work Breakdown

### Step 1: Split Live Resource Identity From Refresh Activity

- Done: [x]

#### Objective

Update `useLiveResource` so a resource can be valid but paused, while preserving
the existing clear-on-invalid behavior for identity changes.

#### Details

Prefer a compact API such as:

```ts
useLiveResource({
  resource: workspaceReadable
    ? { key: `workspace:${workspaceKey}:plan`, path: `/api/workspace/${workspaceKey}/plan` }
    : null,
  mode: "live" | "paused",
  formatError,
})
```

`resource: null` means the resource is invalid and should clear data. A changed
`resource.key` also clears data. `mode: "paused"` means stop polling and focus
refresh listeners but keep the last successful payload and current freshness.
`mode: "live"` means fetch now and install the usual polling/focus refresh
behavior. When returning to live mode with retained data, keep rendering that
data while the refresh is in flight.

Keep the hook implementation small. A full query abstraction or broad reducer
rewrite is unnecessary unless the current effect becomes harder to reason about
after the lifecycle split.

#### Expected Files

- `web/src/live-resource.ts`
- `web/src/types.ts` if a shared resource type is useful

#### Validation

- Add or update `web/src/live-resource.test.tsx` tests for paused retention,
  paused no-polling, live resume refresh, stale-on-resume-failure, first-load
  failure, and key-change clearing.

#### Execution Notes

Implemented `useLiveResource` around explicit `resource` identity plus
`live`/`paused` refresh mode. `resource: null` and changed resource keys clear
data, paused mode tears down refresh activity while preserving successful data,
and live resume refreshes retained data in the background. Added hook tests for
paused retention, no polling while paused, live resume refresh, stale-on-resume
failure, first-load disconnected behavior, and invalid-resource clearing.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 were implemented and validated as one
cohesive XS UI resource lifecycle slice; the complete candidate will receive
full finalize review.

### Step 2: Apply Paused Resources To Workbench Pages

- Done: [x]

#### Objective

Use the new lifecycle shape for Plan, Timeline, and Review so inactive pages
pause refresh without losing payloads.

#### Details

Status should keep its current live behavior while a workspace is readable
because the topbar depends on it across workbench pages. Plan, Timeline, and
Review should use the workspace/page identity as the resource key and switch
between `live` and `paused` based on the active workbench page.

Do not add App-level shadow caches for these payloads unless the hook approach
cannot stay simple. The resource hook should own payload retention so future
pages can opt into the same lifecycle without duplicating state.

#### Expected Files

- `web/src/main.tsx`

#### Validation

- Add or update `web/src/main.test.tsx` tests proving Plan, Timeline, and
  Review render retained payloads immediately after tab switches and still
  refresh in the background.
- Include a workspace/resource identity reset assertion at either hook or App
  level.

#### Execution Notes

Migrated App resource calls to the new identity/mode API. Dashboard, workspace
route, and status resources remain live while valid. Plan, Timeline, and Review
resources keep workspace-scoped identities and switch between live and paused
based on the active workbench page. Added App integration tests that hold the
return refresh pending and verify retained Plan, Timeline, and Review payloads
render immediately with Updating freshness.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The App migration depends directly on the Step 1 hook
API and was validated in the same focused test/build loop; the complete
candidate will receive full finalize review.

## Validation Strategy

- Run the targeted front-end tests:
  `pnpm --dir web test -- live-resource.test.tsx main.test.tsx`.
- Run the full front-end test suite when the targeted behavior is stable:
  `pnpm --dir web test`.
- Run the front-end type/build check:
  `pnpm --dir web build`.
- Run `harness status` before step closeout and before archive/finalize
  decisions.

## Risks

- Risk: Retained data could briefly show under the wrong workspace if the reset
  boundary is too loose.
  - Mitigation: Treat `resource.key` changes and `resource: null` as hard reset
    events, and cover this with tests.
- Risk: The hook API could become too abstract for a small UI.
  - Mitigation: Keep the API limited to resource identity plus live/paused
    refresh mode, and avoid cache maps, persistence, or dependency-heavy query
    abstractions.
- Risk: Paused resources could accidentally keep polling.
  - Mitigation: Ensure timers and focus/visibility listeners are installed only
    in live mode, and test that paused mode does not fetch on interval.

## Validation Summary

- `pnpm --dir web test -- live-resource.test.tsx main.test.tsx` passed with
  25 tests.
- `pnpm --dir web test` passed with 25 tests across 4 files.
- `pnpm --dir web build` passed, including TypeScript and Vite production
  build.
- Finalize reviewers independently reran the same targeted, full, and build
  validation commands.

## Review Summary

- `review-001-full` passed with 0 blocking findings and 0 non-blocking
  findings.
- Correctness review found no lifecycle, stale-data leak, transition, or
  freshness/error regressions.
- Tests review found the hook and App coverage focused and sufficient for the
  inactive payload retention behavior.

## Archive Summary

- Archived At: 2026-04-30T00:30:39+08:00
- Revision: 1
- PR: NONE. The candidate has not been pushed or opened as a PR yet.
- Ready: Acceptance criteria are satisfied locally, both tracked steps are done,
  validation is green, and `review-001-full` passed cleanly.
- Merge Handoff: Archive the plan, commit the tracked archive move, push branch
  `codex/keep-workbench-page-payloads-warm`, open a PR for issue #205, then
  record publish, CI, and sync evidence before waiting for human merge approval.

## Outcome Summary

### Delivered

- `useLiveResource` now models resource identity separately from refresh
  activity with `resource` plus `live`/`paused` mode.
- Plan, Timeline, and Review workbench payloads stay warm across tab switches
  within the same readable workspace and refresh in the background on return.
- Stale/error behavior keeps retained data visible after failed resumed
  refreshes, while first-load failures still use disconnected empty behavior.
- Resource invalidation and key changes clear retained data to prevent
  cross-workspace payload bleed.
- Tests now cover hook lifecycle edges and App-level pending-refresh tab
  switches for Plan, Timeline, and Review.

### Not Delivered

- No websocket/SSE or broader live-refresh policy redesign was added.
- No payload persistence across reloads, browser tabs, or sessions was added.
- No backend API contract changed.

### Follow-Up Issues

- #179 tracks broader workbench live-refresh policy tuning and any future
  transport or stale-state UX expansion.
- #206 tracks persistence of workbench page position across reloads and new
  tabs; payload persistence remains out of scope for this slice unless that
  future issue expands to include it.
