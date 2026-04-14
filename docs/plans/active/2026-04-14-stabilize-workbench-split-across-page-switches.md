---
template_version: 0.2.0
created_at: "2026-04-14T21:04:35+08:00"
approved_at: "2026-04-14T21:15:24+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/168
size: XS
---

# Keep the harness UI explorer and inspector split stable across page switches

## Goal

Make the read-only `harness ui` workbench keep the user-adjusted
explorer/inspector split visually stable when moving between compatible pages.
Today each page persists explorer width under its own storage key, so a resize
on one page can snap back to a different stored width or the page default on
navigation. The result is a noticeable moving-divider effect that makes the
shared workbench shell feel less intentional.

This slice should preserve the current workbench structure and page-specific
content while treating the splitter width as one shared preference owned by
the workbench shell itself. Any page that reuses that shell should inherit the
same persisted split behavior by default; page-specific widths should require
an explicit incompatibility decision rather than being the baseline.

## Scope

### In Scope

- Define one shared persisted width preference at the workbench-shell level so
  compatible page switches do not reposition the divider.
- Keep the existing drag and keyboard resize behavior intact after the
  persistence change.
- Add focused Playwright coverage that proves a resize on one workbench page is
  still reflected after navigating to another compatible page.
- Run an interactive Playwright-backed browser check so the lived-in shell
  experience is confirmed visually, not only by scripted assertions.
- Rebuild the embedded static UI bundle under `internal/ui/static/*`.

### Out of Scope

- Redesigning the workbench shell, divider visuals, or responsive mobile
  collapse behavior.
- Introducing different remembered widths per page family or per route as the
  default behavior.
- Changing non-workbench pages or broader navigation behavior.
- Adding migration or compatibility fallback logic for older storage keys
  beyond what is necessary for the clean target design.

## Acceptance Criteria

- [x] After the user resizes the explorer on one workbench page, navigating to
      another page that reuses the same workbench shell keeps the divider in
      the same place instead of reverting to another page-specific width.
- [x] Pointer drag and keyboard resize behavior still update the shared width
      preference and keep the divider accessible through its existing
      separator control.
- [x] Programmatic Playwright coverage fails if cross-page navigation
      reintroduces the moving-divider regression.
- [x] An interactive Playwright-backed browser pass confirms the shell still
      feels visually stable while navigating between workbench pages.
- [x] The frontend sources typecheck and the embedded UI bundle rebuilds
      cleanly after the change.

## Deferred Items

- Remembering different preferred widths for a future page only if that page
  eventually proves incompatible with the shared workbench shell contract.
- Any visual polish beyond eliminating the unstable divider movement.

## Work Breakdown

### Step 1: Share the persisted splitter width across workbench pages

- Done: [x]

#### Objective

Replace the per-page persisted explorer width behavior with one shared
preference owned by the workbench shell.

#### Details

Keep the change centered in the shared workbench shell so page components stay
simple and the persistence policy is obvious to future readers. The fix should
preserve current width clamping, default rendering, and storage-denied
fallbacks, while removing the page-to-page storage key split that causes the
visible jump. Treat shared persistence as the default semantic for any page
using this shell; do not leave the contract implicit in scattered page wiring.
Prefer the clean end-state over compatibility bridges.

#### Expected Files

- `web/src/workbench.tsx`
- `web/src/pages.tsx`

#### Validation

- Switching among `Status`, `Plan`, `Timeline`, and `Review` after a resize
  keeps the divider aligned to the shared stored width.
- Keyboard and pointer resizing still update the persisted width normally.

#### Execution Notes

Reproduced the bug first with direct Playwright browser automation against the
live `harness ui`: after resizing `Timeline` to `392px`, switching to
`Review` snapped the frame back to `304px` while separate
`harness-ui:explorer-width:timeline` and `...:review` localStorage keys were
created. Followed Red/Green/Refactor by first extending
`scripts/ui-playwright-smoke` with a cross-page regression assertion for the
shared-shell expectation, then centralized splitter persistence in
`WorkbenchFrame` under one shell-owned key,
`harness-ui:workbench-explorer-width`, with a shared default width of `304px`.
Removed per-page storage/default width wiring from `web/src/pages.tsx` so any
page reusing the workbench shell now inherits the same persistence behavior by
default. Pointer drag and keyboard resizing were preserved unchanged.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the persistence policy change and its wiring belong in
one small shared-shell slice.

### Step 2: Add cross-page regression coverage and refresh shipped assets

- Done: [x]

#### Objective

Prove the splitter stays stable across page switches with both scripted and
interactive Playwright validation, then ship the updated UI bundle.

#### Details

Extend the existing workbench Playwright smoke flow instead of creating a new
test harness. The regression check should resize on one compatible workbench
page, navigate to another compatible page, and assert that the stored width
and rendered grid remain aligned. Also run an interactive Playwright browser
session to confirm the shell reads as one stable frame while content changes
between pages. Rebuild the embedded static bundle after the frontend change so
the Go binary serves the fixed UI.

#### Expected Files

- `scripts/ui-playwright-smoke`
- `scripts/ui-playwright-review-smoke`
- `internal/ui/static/*`

#### Validation

- The targeted smoke assertion catches width divergence across compatible page
  navigation.
- The interactive Playwright check confirms the divider does not visibly jump
  during real page switches.
- The rebuilt static assets match the updated frontend sources.

#### Execution Notes

Updated the existing Playwright smoke coverage to assert that a resized
Timeline splitter survives navigation to `Review`, round-trips back without
jumping, and no legacy page-specific width keys are recreated. Updated the
review-specific smoke script to read the same shared shell key. Rebuilt the
frontend with `pnpm --dir web check` and `pnpm --dir web build`, then refreshed
the repo-local `harness` binary with `scripts/install-dev-harness` so the live
UI served the updated embedded assets. Validation used direct Playwright CLI
flows in two modes: a programmatic run confirmed `Timeline` width `392px`
persisted unchanged on `Review` with only the shared key present, and a headed
interactive run resized `Timeline` to `376px`, navigated via the real `Review`
rail link, and confirmed the `Review` frame still rendered `376px 8px ...`
from the same shared stored width. A full end-to-end rerun of
`scripts/ui-playwright-smoke` still stalled in its broader fixture capture
phase after logging `capturing live status pages`, so the targeted shared-shell
assertion was additionally validated directly through Playwright CLI against
the live app.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the regression coverage and asset refresh are the
delivery half of the same bounded UI fix.

## Validation Strategy

- Run `pnpm --dir web check`.
- Run `pnpm --dir web build`.
- Run the targeted programmatic workbench Playwright smoke flow that exercises
  splitter persistence and cross-page navigation.
- Run an interactive Playwright browser session across the main workbench
  pages to confirm the shell remains visually stable while only the content
  changes.
- Run `git diff --check`.

## Risks

- Risk: Some pages may have been implicitly relying on different default widths
  even though they share the same workbench shell.
  - Mitigation: Make shared persistence the explicit shell contract, then
    validate the main workbench routes with both scripted and interactive
    Playwright checks so any real incompatibility is surfaced immediately.
- Risk: A localStorage assertion alone could miss a layout bug where the DOM
  still renders a different divider position after navigation.
  - Mitigation: Make the smoke coverage check both the persisted width value
    and the rendered workbench frame measurement after route changes.

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
