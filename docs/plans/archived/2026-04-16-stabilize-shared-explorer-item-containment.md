---
template_version: 0.2.0
created_at: "2026-04-16T09:06:26+08:00"
approved_at: "2026-04-16T09:08:38+08:00"
source_type: issue
source_refs:
    - '#173'
size: S
---

# Stabilize shared ExplorerItem containment at narrow workbench widths

## Goal

Fix the narrow-width explorer row wrapping bug tracked in issue `#173` without
treating it as a Timeline-only one-off. Today the shared `ExplorerItem` layout
allows compact metadata to break in visually awkward ways under realistic
narrow explorer widths, most notably in Timeline rows where labels such as
`Evidence · rev 1` can leave the revision number stranded on its own line.

The accepted direction is to define a shared containment contract for all
workbench pages that render `ExplorerItem` rows, then let page-specific
Timeline and Review markup use that shared contract to keep compact metadata
readable at the minimum supported explorer width. This should stay a bounded
layout/readability slice rather than turning into a broader explorer redesign.

## Scope

### In Scope

- Define the shared narrow-width containment rules for `ExplorerItem` rows used
  by `Status`, `Timeline`, and `Review`.
- Adjust shared row structure and/or styles so explorer rows stay contained at
  the minimum supported `220px` explorer width.
- Prevent compact metadata tokens such as `rev 1` or compact review status
  labels from breaking into visually awkward fragments.
- Keep `Status`, `Timeline`, and `Review` readable under the same shared
  containment contract, with small page-specific markup changes when needed.
- Add or update focused browser validation so the shared explorer-row contract
  is proven at the narrow-width boundary instead of only by static inspection.
- Refresh the embedded UI bundle under `internal/ui/static/*` after the
  frontend changes land.

### Out of Scope

- Redesigning the overall workbench shell, splitter behavior, or page-level
  navigation.
- Reworking the `Plan` page tree explorer, which does not currently use
  `ExplorerItem`.
- Changing Timeline, Review, or Status data contracts beyond the small label or
  structure adjustments needed to keep compact explorer metadata readable.
- Broader visual polish of unrelated inspector panels, headers, or typography.

## Acceptance Criteria

- [x] Every page that renders shared `ExplorerItem` rows today (`Status`,
      `Timeline`, and `Review`) stays visually contained at the `220px`
      explorer width boundary without row-level horizontal overflow.
- [x] Timeline explorer metadata remains readable at that width, and compact
      labels such as `Evidence · rev 1` do not strand `1` on its own wrapped
      line.
- [x] Review explorer rows keep their compact metadata/status treatment
      readable at the same boundary and do not regress from the earlier narrow
      width stabilization work.
- [x] Status explorer rows continue to read cleanly under the shared contract
      after the containment changes.
- [x] Frontend checks/build pass, the embedded static UI bundle is refreshed,
      and focused browser validation or screenshot evidence covers the shared
      narrow-width contract.

## Deferred Items

- Any broader redesign of explorer-row information hierarchy beyond the shared
  containment and compact-metadata behavior needed for this slice.
- Applying the same treatment to the `Plan` tree explorer unless that page is
  later migrated onto `ExplorerItem`.
- Follow-up cleanup of compact label wording if containment can be fixed
  without changing those labels now.

## Work Breakdown

### Step 1: Define and implement the shared ExplorerItem narrow-width contract

- Done: [x]

#### Objective

Make the shared explorer-row container resilient at the `220px` width boundary
so all current `ExplorerItem` pages inherit a stable containment baseline.

#### Details

Treat this as a shared row-layout contract, not a Timeline-only patch. The
shared component should prefer truncation/containment over awkward fragment
wrapping when space is tight, while still allowing page-specific row markup to
preserve each page's existing information hierarchy. Timeline and Review may
need richer subtitle structure so compact metadata tokens stay grouped, but the
shared row container remains the main contract surface.

#### Expected Files

- `web/src/workbench.tsx`
- `web/src/pages.tsx`
- `web/src/styles.css`
- `web/src/helpers.ts`

#### Validation

- Shared explorer rows remain contained at `220px` without row-level horizontal
  overflow.
- Timeline and Review compact metadata stay readable without orphaned trailing
  fragments.
- Status rows still render cleanly after the shared containment changes.

#### Execution Notes

Implemented the shared compact-subtitle contract instead of a Timeline-only
one-off. Timeline metadata now separates the kind label from the revision token
so `rev 1` stays grouped at narrow widths, and Review now renders its compact
metadata/status line through the same shared subtitle row structure rather than
an isolated page-only container. The shared subtitle container also now clamps
its own width/overflow boundaries so `ExplorerItem` rows stay visually bounded
at the `220px` explorer floor. This slice used a real Red/Green browser loop:
the first smoke extension failed on the original Timeline regression by proving
that the `rev` text and revision number landed on different visual lines at
`220px`, then the shared row fix turned that same check green. After archive,
direct human visual feedback on the shipped Timeline explorer exposed one more
real presentation issue: the first repair solved containment but made the row
hierarchy feel too mechanical by splitting `Evidence` and `rev 1` across the
subtitle row while the timestamp still dominated the title line. Revision `2`
reopened in `finalize-fix` mode and retuned Timeline to keep `Evidence · rev 1`
as one left-side metadata string with a weaker right-aligned timestamp, so the
row reads like one dense explorer line again without giving up the shared
containment contract.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the layout change and its shared subtitle structure are
easier to review together with the focused validation updates than as an
artificial intermediate step boundary.

### Step 2: Prove the shared containment contract with browser validation and refreshed assets

- Done: [x]

#### Objective

Back the shared layout change with browser-level evidence at the narrow-width
boundary and refresh the embedded UI bundle that ships with the Go server.

#### Details

Use the existing UI browser validation flow to exercise the current
`ExplorerItem` pages (`Status`, `Timeline`, and `Review`) at the `220px`
explorer width boundary. Prefer deterministic DOM containment checks and saved
screenshots over a code-only claim that the CSS "looks right." Keep this
evidence narrowly focused on the shared explorer-row contract rather than
turning the smoke flow into a full visual regression suite.

#### Expected Files

- `scripts/ui-playwright-smoke`
- `internal/ui/static/index.html`
- `internal/ui/static/assets/*`

#### Validation

- Focused browser validation proves the shared explorer-row contract on
  `Status`, `Timeline`, and `Review` at `220px`.
- `pnpm --dir web check` and `pnpm --dir web build` pass after the layout
  changes.
- The refreshed embedded UI bundle matches the frontend source changes.

#### Execution Notes

Extended `scripts/ui-playwright-smoke` with one shared `220px` explorer-row
contract check that now exercises `Status`, `Timeline`, and `Review` through
the same browser path. The check records DOM containment rather than relying on
static inspection, keeps title ellipsis legal, and preserves a targeted
Timeline-specific assertion that the revision token stays on one line. While
making that shared proof real, the smoke flow also absorbed two small
validation-only hardening fixes that the new path surfaced: waiting for the
shared width persistence effect before asserting on localStorage, and reading
`window.location.origin` directly instead of assuming `URL` exists in every
Playwright execution context. Browser evidence landed in
`output/playwright/harness-ui-smoke-52645-1776302387494631000-18421/`, which
contains `status-explorer-220.png`, `timeline-explorer-220.png`, and
`review-explorer-220.png` from the passing run. Rebuilt the embedded frontend
bundle under `internal/ui/static/*` after the source changes and reran the
shared smoke to green. After the `finalize-fix` reopen for revision `2`, reran
the same browser flow successfully and captured refreshed 220px screenshots
under `output/playwright/harness-ui-smoke-95942-1776303427222054000-25066/`,
including a Timeline explorer screenshot whose subtitle hierarchy now reads as
one left-side metadata string plus a weaker right-side timestamp.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the shared smoke proof and embedded-bundle refresh are
part of the same bounded UI slice and are best reviewed together at finalize.

## Validation Strategy

- Run `pnpm --dir web check`.
- Run `pnpm --dir web build` to refresh and validate the embedded static UI
  bundle.
- Run `git diff --check` to catch formatting or whitespace mistakes.
- Use focused browser validation at the `220px` explorer width boundary for the
  current `ExplorerItem` pages: `Status`, `Timeline`, and `Review`.
- Capture DOM containment evidence and screenshots for the narrow-width state
  so archive-time review does not depend on chat memory.

## Risks

- Risk: A shared `ExplorerItem` change could accidentally degrade a page that
  was not part of the original Timeline repro.
  - Mitigation: Treat `Status`, `Timeline`, and `Review` as one shared contract
    surface and validate all three at the same `220px` boundary.
- Risk: Fixing awkward wrapping by over-compressing labels could make explorer
  rows technically contained but less readable.
  - Mitigation: Preserve the existing page-specific information hierarchy where
    possible and prefer grouped compact tokens plus truncation over overly
    cryptic relabeling.

## Validation Summary

- `pnpm --dir web check`
- `pnpm --dir web build`
- `git diff --check`
- `scripts/ui-playwright-smoke`
- Initial passing 220px browser evidence from
  `output/playwright/harness-ui-smoke-52645-1776302387494631000-18421/`,
  including `status-explorer-220.png`, `timeline-explorer-220.png`, and
  `review-explorer-220.png`
- Refreshed revision `2` 220px browser evidence from
  `output/playwright/harness-ui-smoke-95942-1776303427222054000-25066/`,
  including `status-explorer-220.png`, `timeline-explorer-220.png`, and
  `review-explorer-220.png`

## Review Summary

Finalize full review `review-001-full` passed cleanly at revision `1` with no
blocking or non-blocking findings. Reviewer slot `correctness` confirmed the
shared ExplorerItem containment change stayed semantically sound across
`Status`, `Timeline`, and `Review`, and reviewer slot `tests` reran
`scripts/ui-playwright-smoke` to confirm the widened 220px browser proof stayed
green after the validation hardening. After direct human visual feedback
reopened the candidate in `finalize-fix` mode, finalize full review
`review-002-full` also passed cleanly at revision `2` with no findings.
Reviewer slot `correctness` confirmed the visual-polish repair restored the
Timeline hierarchy without breaking the shared containment contract, and
reviewer slot `tests` confirmed the adjusted Timeline assertion plus rebuilt UI
bundle stayed aligned with the passing browser proof.

## Archive Summary

- Archived At: 2026-04-16T09:46:26+08:00
- Revision: 2
This candidate still resolves issue #173 through one shared ExplorerItem
narrow-width contract, but revision `2` retunes the Timeline hierarchy after
human visual feedback. Timeline now keeps `Evidence · rev 1` together on the
left side of the subtitle row while demoting the timestamp to a weaker
right-aligned position, so the row reads like one dense explorer line without
backsliding on the shared `220px` containment proof across `Status`,
`Timeline`, and `Review`.

- PR: pending post-archive publish handoff from branch
  `codex/issue-173-shared-explorer-containment`
- Ready: All acceptance criteria remain satisfied after the `finalize-fix`
  reopen, `review-002-full` passed with no findings, and focused browser
  validation plus refreshed 220px screenshots match the revision `2`
  candidate behavior.
- Merge Handoff: Run `harness archive`, commit the tracked archive move plus
  closeout notes, push branch `codex/issue-173-shared-explorer-containment`,
  open or update the PR for issue #173, and record publish/CI/sync evidence
  until `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Shared compact subtitle-row containment in `web/src/styles.css` plus the
  Timeline and Review explorer markup updates in `web/src/pages.tsx`.
- Timeline metadata splitting in `web/src/helpers.ts` so the revision token can
  stay grouped without changing the event title or inspector behavior.
- A widened shared browser proof in `scripts/ui-playwright-smoke` that checks
  220px explorer-row containment for `Status`, `Timeline`, and `Review`, while
  also hardening the existing width-persistence assertions against timing and
  runtime-environment issues.
- Refreshed embedded UI bundle artifacts under `internal/ui/static/*`.
- A revision `2` Timeline visual-polish repair that keeps `Evidence · rev 1`
  as one left-side metadata string and moves the timestamp into a weaker
  right-aligned subtitle position.

### Not Delivered

- No redesign of the overall workbench shell, page headers, or `Plan` tree
  explorer.
- No broader explorer-row wording cleanup beyond the compact-token containment
  needed for this slice.

### Follow-Up Issues

- No new follow-up issue was created in this slice. Deferred items remain
  intentionally out of scope: broader explorer-row visual redesign, migrating
  the `Plan` tree onto `ExplorerItem`, and any later copy cleanup for compact
  labels if a future workbench polish pass wants it.
