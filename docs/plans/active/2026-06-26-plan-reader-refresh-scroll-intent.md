---
template_version: 0.2.0
created_at: "2026-06-26T16:18:10+08:00"
approved_at: "2026-06-26T16:19:15+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/261
size: S
---

# Plan Reader Refresh Scroll Intent

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Fix the Plan workspace reader so live refreshes do not replay the last selected
heading as an ongoing scroll command. Manual scrolling inside the right-hand
Plan reader should stay under user control across refreshed Plan payloads when
the selected Plan node has not changed.

Keep the useful parts of warm Plan state: the explorer may continue to show the
remembered selected node, and clicking a Plan heading in the explorer should
still navigate the reader to that heading once at the time of selection.

## Scope

### In Scope

- Split Plan reader heading anchor assignment from imperative reader
  navigation in `PlanWorkspace`.
- Ensure selecting a document root, heading, supplement directory, or
  supplement file still updates the inspector and explorer state correctly.
- Add regression coverage for refreshed Plan document payloads preserving
  manual reader scroll when the selected node is unchanged.
- Keep or update the existing Plan UI smoke coverage that verifies an explicit
  heading click scrolls the reader to the selected heading.

### Out of Scope

- Changing the live refresh cadence or `useLiveResource` polling behavior.
- Reworking Plan explorer selection semantics beyond the scroll intent bug.
- Adding browser hash navigation or deep-link support for individual headings.
- Compatibility shims for obsolete Plan UI state shapes.

## Acceptance Criteria

- [x] A Plan heading selection scrolls the reader to that heading once.
- [x] A subsequent live refresh or equivalent rerender with a fresh
      `document`/markdown payload does not snap the reader back to the selected
      heading after the user has manually scrolled away.
- [x] Selecting the document root may still take the reader to the top as an
      explicit navigation action, but refreshes do not repeatedly force that
      top position.
- [x] Supplement file/directory selection remains functional and does not
      trigger markdown reader scroll restoration.
- [x] Automated coverage fails on the old repeated-scroll behavior and passes
      with the new scroll intent boundary.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Separate Scroll Intent From Render Refresh

- Done: [x]

#### Objective

Update `PlanWorkspace` so rendered heading IDs are maintained whenever markdown
content changes, while `scrollTop` changes only for an explicit Plan node
navigation request.

#### Details

The current effect in `web/src/pages.tsx` both assigns heading IDs and scrolls
to `selectedHeading` or top. Because the effect depends on refreshed
`document`/`documentHTML` data, each live Plan poll can rerun the scroll logic
even when the selected node is unchanged.

Prefer a small local state/ref boundary in `PlanWorkspace`: record a navigation
intent when `selectPlanNode` handles an explicit user selection, then consume
that intent after render to perform the one-time scroll. Keep anchor assignment
as its own render-following effect so refreshed markdown still receives stable
heading IDs.

The implementation should preserve the clean target shape rather than adding
compatibility layers. Avoid changes to `useLiveResource` unless execution shows
the bug cannot be fixed at the Plan workspace boundary.

#### Expected Files

- `web/src/pages.tsx`
- `web/src/types.ts` if a state type change is truly needed

#### Validation

- Existing Plan explorer interactions continue to render the right selected
  title/detail.
- Explicit heading and root selections still produce exactly one expected
  reader scroll.
- No refresh-driven effect path directly sets `scrollTop` merely because the
  Plan payload object changed.

#### Execution Notes

Implemented the scroll intent boundary in `PlanWorkspace`: heading ID
assignment now follows markdown render refreshes, while `scrollTop` changes
only when `selectPlanNode` records an explicit navigation request. Added a
local request version so repeat-clicking the same selected node can still issue
a fresh one-time scroll. The implementation stayed inside `web/src/pages.tsx`
and did not require `PlanWorkspaceState` or `useLiveResource` changes.

Validation run:

- `pnpm --dir web test -- main.test.tsx` first failed on the new regression
  test with the old behavior (`scrollTop` changed from 111 to 513 after
  rerender), then passed after the fix.
- `pnpm --dir web test`
- `pnpm --dir web check`

#### Review Notes

`review-001-delta` passed on 2026-06-26 with `correctness` and `tests`
reviewer slots. Both reviewers submitted valid results through
`harness review submit`, both reported no findings, and
`harness review aggregate --round review-001-delta` passed with zero blocking
or non-blocking findings.

### Step 2: Add Regression Coverage And Verify

- Done: [x]

#### Objective

Add focused coverage for the refresh/rerender scroll regression and run the
relevant frontend validation.

#### Details

Prefer a Vitest/jsdom component test around `PlanWorkspace` or the existing App
state tests. The test should select a heading, confirm the initial navigation
intent, manually move the reader scroll position away from that heading, then
rerender with an equivalent-but-new Plan document payload. The assertion should
prove the manual scroll position is preserved unless a new selection is made.

Keep the existing Playwright smoke heading-navigation check intact. Extend the
smoke only if component coverage cannot reliably model the refresh behavior or
if execution changes the browser-only scroll path.

#### Expected Files

- `web/src/main.test.tsx` or `web/src/pages.test.tsx`
- `scripts/ui-playwright-plan-smoke` only if a browser-level check is needed

#### Validation

- Run the relevant frontend unit test command, such as `pnpm --dir web test`
  or a narrower Vitest invocation if supported by the repo.
- Run `pnpm --dir web check`.
- Run `scripts/ui-playwright-plan-smoke` if the smoke script is updated or if
  execution needs browser-level confidence for the scroll behavior.

#### Execution Notes

Added a focused Vitest regression in `web/src/main.test.tsx` that mounts
`PlanWorkspace`, clicks the `Scope` heading, verifies the explicit navigation
scroll, manually changes reader `scrollTop`, then rerenders with an
equivalent fresh `PlanDocument` object. With the old implementation this test
failed because rerender replayed the selected-heading scroll and changed
`scrollTop` from 111 to 513; after the Step 1 fix it passes.

After finalize review, added a second focused regression for the document-root
path: default root selection preserves manual reader `scrollTop` across an
equivalent fresh `PlanDocument` rerender, while an explicit root click still
scrolls the reader back to top once.

Validation run:

- `pnpm --dir web test -- main.test.tsx`
- `pnpm --dir web test`
- `pnpm --dir web check`

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 was the regression coverage half of the same
tightly coupled Step 1 implementation slice already reviewed in
`review-001-delta`. The `tests` reviewer specifically reviewed whether the new
regression proves the refreshed-document scroll replay risk and reported no
findings, so a separate second step review would duplicate that accepted
closeout.

## Validation Strategy

- Focus first on deterministic component coverage for the state/ref boundary:
  explicit selection should scroll once, refreshed equivalent Plan data should
  not scroll again.
- Use TypeScript checking to catch state/type regressions.
- Preserve existing Plan smoke coverage for real browser heading navigation;
  expand it only when needed to cover behavior that jsdom cannot represent.

## Risks

- Risk: The fix could stop legitimate first-time heading navigation after a
  user clicks the explorer.
  - Mitigation: Keep the existing smoke assertion that heading clicks move the
    reader near the selected heading, and add a focused unit assertion for the
    one-time navigation request.
- Risk: A component test may not faithfully model browser scroll layout.
  - Mitigation: Test the intent boundary in Vitest and use the existing
    Playwright smoke path for real layout confidence when execution warrants
    it.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

`review-001-delta` passed with zero findings for the implementation and first
regression test.

`review-002-full` requested changes for two archive-readiness gaps:

- the document-root/default-root refresh replay path lacked explicit
  regression coverage
- the top-level acceptance criteria were still unchecked despite completed
  steps and validation

Both findings were fixed by adding the root/default-root regression test and
checking off the satisfied acceptance criteria. Follow-up review remains
pending.

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
