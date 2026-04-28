---
template_version: 0.2.0
created_at: "2026-04-28T15:45:53Z"
approved_at: "2026-04-28T23:48:41+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/172
size: M
---

# Keep Workbench Page State Warm

## Goal

Keep the workbench feeling like one stable UI while users move between Plan,
Timeline, and Review. When a user switches away from one of those pages and
comes back, the page should remember the part they were looking at whenever
that item still exists.

This is a UI continuity pass, not a data-freezing pass. Live data may continue
to refresh, and invalid selections should fall back to a sensible default.

## Scope

### In Scope

- Preserve Plan page selection and expanded tree state across workbench tab
  switches.
- Preserve Timeline selected event and selected detail tab across workbench tab
  switches.
- Preserve Review selected round, selected detail tab, selected artifact, and
  artifact panel visibility across workbench tab switches.
- Keep live data refresh behavior working; do not freeze old payloads just to
  preserve UI selection.
- Fall back cleanly when refreshed data no longer contains the previously
  selected item.
- Add focused frontend tests for switching away and back without losing the
  expected page-local state.

### Out of Scope

- Redesigning the workbench shell or rail.
- Replacing the existing live resource system.
- Persisting state across browser reloads, new tabs, or separate workspaces.
- Stopping all refetching on tab switches.
- Adding new Plan, Timeline, or Review features beyond state continuity.

## Acceptance Criteria

- [x] Plan remembers the selected document heading or supplement node across
      workbench tab switches when the item still exists.
- [x] Plan remembers expanded tree branches across workbench tab switches when
      those branches still exist.
- [x] Timeline remembers the selected event and detail tab across workbench tab
      switches when they still exist.
- [x] Review remembers the selected round, selected detail tab, selected
      artifact, and artifact panel visibility across workbench tab switches
      when they still exist.
- [x] When refreshed data removes a remembered item, the affected page falls
      back to the same kind of default it uses today instead of showing a broken
      selection.
- [x] Live refresh and freshness indicators still reflect current resource
      state instead of treating preserved UI state as preserved data.
- [x] Focused frontend tests cover the continuity behavior and the invalid
      selection fallback.

## Deferred Items

- Persisting workbench page state across browser reloads.
- A broader live refresh policy redesign for inactive pages.
- Search, filtering, or new navigation affordances inside Plan, Timeline, or
  Review.

## Work Breakdown

### Step 1: Define the remembered page state shape

- Done: [x]

#### Objective

Move the remembered Plan, Timeline, and Review UI state to a workbench-level
owner so tab switches no longer destroy it.

#### Details

Keep this state scoped to the current workspace route. The state should describe
where the user is looking, not the loaded API payload. The implementation may
use one small workbench state hook or local state in `App`, but it should avoid
introducing a broad state-management layer.

#### Expected Files

- `web/src/main.tsx`
- `web/src/pages.tsx`
- `web/src/types.ts`

#### Validation

- Typecheck the frontend.
- Confirm Plan, Timeline, and Review can receive controlled state from the
  workbench owner without changing their visible layout.

#### Execution Notes

Added small page-specific state shapes for Plan, Timeline, and Review, then
lifted their ownership into `App` so page components can be controlled across
workbench tab switches. State is scoped to the current workspace key and reset
when the workspace changes.

#### Review Notes

NO_STEP_REVIEW_NEEDED: completed as part of the combined approved slice; the
integrated candidate is covered by the finalize review.

### Step 2: Preserve state while keeping data refresh live

- Done: [x]

#### Objective

Wire Plan, Timeline, and Review so their remembered state survives tab switches
and is corrected only when refreshed data makes the remembered selection
invalid.

#### Details

Do not freeze API payloads as the primary fix. If a page refetches when the user
returns, preserve the user's selection through that refresh when possible. If
the selected item is no longer present, reuse the page's current defaulting
behavior.

#### Expected Files

- `web/src/main.tsx`
- `web/src/pages.tsx`
- `web/src/live-resource.ts`

#### Validation

- Typecheck the frontend.
- Run the focused frontend test suite.
- Manually exercise the UI in a browser if the implementation changes visible
  workbench behavior in a way that unit tests cannot cover well.

#### Execution Notes

Wired Plan, Timeline, and Review to use the remembered state while continuing
to fetch live resources normally. Selection, expanded branches, detail tabs,
and artifact panel state are corrected only after loaded data proves the
remembered id is no longer present.

#### Review Notes

NO_STEP_REVIEW_NEEDED: completed as part of the combined approved slice; the
integrated candidate is covered by the finalize review.

### Step 3: Lock continuity behavior with tests

- Done: [x]

#### Objective

Add focused tests proving that tab switches preserve expected page state and
that missing remembered items fall back cleanly.

#### Details

Prefer small jsdom tests around the frontend components or app shell. Add
browser validation only where jsdom cannot express the routing and interaction
behavior clearly.

#### Expected Files

- `web/src/pages.test.tsx`
- `web/src/live-resource.test.tsx`
- `web/src/main.test.tsx`

#### Validation

- `pnpm --dir web test`
- `pnpm --dir web check`
- `pnpm --dir web build`

#### Execution Notes

Added `web/src/main.test.tsx` coverage for Plan, Timeline, and Review tab
switch continuity, plus fallback behavior for missing remembered review ids and
artifact ids. Review repair added Plan fallback, Timeline fallback,
supplements-only Plan selection, and post-return Plan refetch coverage. Focused
and full frontend validation passed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: completed as part of the combined approved slice; the
integrated candidate is covered by the finalize review.

## Validation Strategy

- Use focused frontend tests for the preserved-state behavior and fallback
  behavior.
- Run the full frontend check/build commands before archive.
- Use browser validation if the implementation affects real routing or visual
  behavior beyond what jsdom tests cover.

## Risks

- Risk: Preserved UI state could accidentally preserve stale data.
  - Mitigation: Keep remembered state limited to selection, expansion, and panel
    choices; keep API payloads and freshness separate.
- Risk: Controlled state could make Plan, Timeline, or Review harder to follow.
  - Mitigation: Keep the state shape small and page-specific instead of adding a
    generic global store.
- Risk: Refreshed data could remove a remembered item.
  - Mitigation: Keep the existing default-selection behavior for invalid
    remembered IDs.

## Validation Summary

Passed:

- `pnpm --dir web test`
- `pnpm --dir web check`
- `pnpm --dir web build`

## Review Summary

Finalize review `review-001-full` requested changes: preserve supplements-only
Plan child selections and expand fallback/refetch test coverage. The repair
fixed the supplements-only Plan fallback path and added focused regression
coverage for Plan, Timeline, and Review fallback plus the Plan refetch claim.
Fresh validation passed; a follow-up finalize review will verify the repaired
candidate.

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
