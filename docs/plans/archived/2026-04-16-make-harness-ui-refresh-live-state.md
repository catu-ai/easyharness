---
template_version: 0.2.0
created_at: "2026-04-16T23:44:00+08:00"
approved_at: "2026-04-16T23:43:19+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/158
size: XS
---

# Make the harness UI refresh live state without manual reload

## Goal

Make the read-only `harness ui` workbench react to harness state changes that
happen outside the currently open browser tab. Today the UI fetches each page's
data only on first load or when navigating into that page, so CLI commands,
background work, or changes made from another window can leave the open UI
showing stale status, timeline, plan, or review data until the user manually
reloads the page.

This slice should keep the current thin read-only architecture and solve the
trust problem with a lighter-weight synchronization model: visible pages should
re-fetch on a short polling cadence, refresh immediately when the tab regains
focus or visibility, and surface a small freshness/disconnected signal in the
existing shell. The goal is to make the UI feel operationally current without
introducing websocket/SSE infrastructure or a second hidden state system.

## Scope

### In Scope

- Add a shared client-side refresh mechanism for the read-only workbench pages
  so `Status`, `Plan`, `Timeline`, and `Review` stop behaving like one-shot
  snapshots.
- Define and implement an explicit freshness target for visible pages:
  automatic refresh within a few seconds while the tab is visible, plus an
  immediate catch-up refresh on focus/visibility regain.
- Surface lightweight freshness state in the shared shell so the user can tell
  whether the UI is updating normally, temporarily stale, or disconnected after
  fetch failures.
- Keep the implementation aligned with the current thin architecture by
  reusing the existing `/api/status`, `/api/plan`, `/api/timeline`, and
  `/api/review` endpoints instead of adding push transport by default.
- Add focused validation that proves external state changes become visible
  without manual browser reload.
- Rebuild the embedded static UI bundle under `internal/ui/static/*`.

### Out of Scope

- Adding websocket, SSE, file-watch, or other push-driven backend
  infrastructure in this slice.
- Redesigning the workbench layout, navigation model, or page information
  hierarchy beyond the small freshness indicator needed for this behavior.
- Converting `harness ui` into an action-triggering surface or changing CLI
  workflow semantics.
- Solving every possible background synchronization concern beyond the main
  read-only workbench pages covered by the current UI.

## Acceptance Criteria

- [x] While a workbench page is open and the tab is visible, external harness
      state changes become visible automatically without a full browser reload.
- [x] When the browser tab regains focus or visibility after being backgrounded,
      the active page refreshes immediately instead of waiting for the next
      polling interval.
- [x] The shared shell surfaces concise freshness state that distinguishes a
      healthy updating state from a stale/disconnected state after fetch
      failures.
- [x] Focused automated validation fails if the UI returns to one-shot snapshot
      behavior for at least one representative external state-change flow.
- [x] `pnpm --dir web check`, `pnpm --dir web build`, and `git diff --check`
      pass after the change.

## Deferred Items

- Evaluating whether some future pages need different polling cadences or
  route-specific refresh policies once this shared baseline exists.
- Push-based transport work if later measurement shows polling plus focus
  refresh is still not responsive enough for real usage.
- Broader stale-state recovery features beyond the shell-level indicator and
  retry behavior needed for this slice.

## Work Breakdown

### Step 1: Add shared live-refresh and freshness state to the workbench shell

- Done: [x]

#### Objective

Turn the current one-shot page fetches into a shared read-only refresh model
with explicit freshness signaling.

#### Details

Keep the change centered in the shared frontend state/loading flow rather than
spreading independent timers across every page component. The workbench should
own a small refresh contract for the active page data: poll while the document
is visible, refresh immediately on focus or visibility regain, avoid overlapping
fetches for the same resource, and record the last successful sync/error state
so the shell can show a concise freshness indicator. Prefer the clean target
design over compatibility shims; do not add optional legacy snapshot behavior.

The initial target should be intentionally modest and explicit: a short
polling cadence suitable for a local read-only UI, plus immediate focus
catch-up. If a single shared helper/hook can keep the four page fetch flows
consistent, prefer that over page-by-page ad hoc logic. Keep the backend API
shape unchanged unless the freshness UI proves impossible without a tiny
additive field.

#### Expected Files

- `web/src/main.tsx`
- `web/src/helpers.ts`
- `web/src/types.ts`
- `web/src/workbench.tsx`
- `web/src/styles.css`

#### Validation

- With the UI left open on a representative page, external CLI or fixture
  changes become visible automatically within the agreed polling window.
- Refocus/visibility regain triggers an immediate refresh.
- Loading/error transitions do not leave the shell stuck in a misleading
  "fresh" state after failed fetches.

#### Execution Notes

Implemented a shared `useLiveResource` loop in `web/src/main.tsx` so `status`
polls continuously, the active workbench page (`plan`, `timeline`, or
`review`) refreshes while visible, and both respond immediately to
`focus`/`visibilitychange` catch-up events. The hook now prevents overlapping
fetches, preserves last-known-good data on refresh failures, and separates
initial load state from background refreshes so the shell no longer flashes a
full loading state every polling cycle.

Added explicit freshness modeling in `web/src/types.ts` and
`web/src/helpers.ts`, then surfaced it through a new topbar indicator in
`web/src/workbench.tsx` plus matching styling in `web/src/styles.css`. The
indicator now distinguishes healthy live updates, background refreshes, stale
state after failed refreshes, and disconnected first-load failures. Strict
red-first TDD was not practical for this step because the existing browser
smoke harness was already known to stall before this new behavior surface, and
the change spans shared runtime polling plus browser lifecycle events rather
than a narrow pure function. The slice instead validated behavior directly in a
live browser against a temporary harness workdir before review.

Follow-up repairs from `review-001-full` and `review-002-delta` hardened the
catch-up path further: focus/visibility-triggered refreshes now abort an older
in-flight request before starting the replacement fetch, and only the newest
active controller can clear the shared in-flight guard in `finally`. That
prevents both the original skip-on-refocus bug and the later overlap-window
regression found during delta review.

Revision 2 reopened the candidate for a narrow UX polish fix after PR feedback
pointed out that the topbar freshness pill briefly flashed `Updating` on every
fast local refresh. `useLiveResource` now applies a small buffer before showing
the background `Updating` state when prior live data already exists. Fast
successful refreshes therefore stay visually on `Live`, while fast failures now
move directly from `Live` to `Stale` without a hard-to-read intermediate blink.
A focused Playwright follow-up validated both transitions against a live
`harness ui` session and recorded the markers in the tracked browser
supplement.

#### Review Notes

`review-001-full` requested changes for one catch-up correctness gap and two
validation gaps. `review-002-delta` then narrowed the remaining repair scope
to one in-flight-guard bug and one visibility-test isolation bug. After the
final repair, `review-003-delta` passed cleanly with no blocking or
non-blocking findings.

### Step 2: Add regression coverage for external updates and refresh shipped assets

- Done: [x]

#### Objective

Prove the UI no longer requires manual reload for representative state changes,
then rebuild the embedded bundle that `harness ui` serves.

#### Details

Favor targeted validation that actually exercises an open browser against a
changing harness workdir rather than only unit-testing fetch helpers in
isolation. Extend existing UI/browser validation where practical so future UI
work inherits the regression guard. The validation should cover at least one
real "change after page load" flow and one refocus/visibility catch-up flow if
the harness setup allows it. Refresh the embedded static assets only after the
frontend sources and browser validation agree on the new behavior.

If interactive browser evidence is needed to confirm the freshness indicator
reads clearly in practice, record it in a matching tracked supplement package
for this plan instead of leaving it in terminal scrollback.

#### Expected Files

- `scripts/ui-playwright-smoke`
- `internal/ui/static/*`
- `docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/*` (only if needed)

#### Validation

- Automated browser validation catches regression back to manual-reload-only
  behavior.
- Rebuilt embedded assets match the updated frontend source.
- Any tracked interactive evidence clearly shows the freshness indicator and
  automatic update behavior.

#### Execution Notes

Extended `scripts/ui-playwright-smoke` with a same-session status-page
regression that mutates underlying harness state after the page is already
open, waits for the open UI to reflect the new node without a reload, and then
asserts the focus-triggered catch-up path from the same browser session.

Rebuilt the embedded UI bundle with `pnpm --dir web build`, refreshed the
repo-local binary with `scripts/install-dev-harness`, and ran a targeted
Playwright validation against a temporary harness runtime. That focused browser
run verified three behaviors directly: automatic polling updates from `land`
to `execution/finalize/await_merge`, a `Stale` topbar state after forced
refresh failures while preserving prior data, and immediate recovery back to
`land` after restoring `fetch` plus dispatching `focus`. The broader
`scripts/ui-playwright-smoke` flow still stalls in an older early-navigation
segment outside this issue's new assertions, so the targeted live-session run
is the authoritative validation evidence for this slice so far.

Later review-driven repairs strengthened that validation story further:
`scripts/ui-playwright-smoke` now isolates the visibility catch-up assertion
with a single-use fetch gate so ordinary polling cannot satisfy the recovery
window by accident, and it adds same-session live-update coverage for the
already-open `Timeline` page after appending a new event. Rebuilt embedded
assets and refreshed the repo-local `harness` binary after each frontend
repair so the served UI matched the reviewed source. Durable evidence for the
command-level checks and the focused browser run now lives in
`docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/command-validation.txt`,
`browser-validation.txt`, and
`validation-evidence.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 remained the validation-and-asset delivery half
of the same bounded slice, and its review-driven repairs were covered by the
same `review-001-full` -> `review-003-delta` closeout sequence tracked in Step
1's review notes.

## Validation Strategy

- Run `pnpm --dir web check`.
- Run `pnpm --dir web build`.
- Run focused browser validation against a live `harness ui` session that
  mutates underlying harness state after the page is already open.
- If needed, run an interactive browser pass to sanity-check freshness copy and
  visual states, and record durable evidence in a tracked supplement.
- Run `scripts/install-dev-harness` after Go-served embedded assets change so
  the repo-local `harness` binary serves the rebuilt UI.
- Run `git diff --check`.
- Record durable command and browser evidence under
  `docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/`
  so later finalize review does not depend on disposable local output.

## Risks

- Risk: Polling plus focus refresh could introduce duplicated requests,
  overlapping fetches, or noisy loading states that make the shell feel more
  jittery instead of more trustworthy.
  - Mitigation: Centralize refresh coordination, suppress overlapping fetches
    for the same resource, and keep the freshness indicator concise instead of
    turning every poll into a visible spinner.
- Risk: Browser automation could accidentally validate only initial load and
  miss the real "state changed after page open" path that issue #158 reports.
  - Mitigation: Use a validation flow that mutates the underlying harness
    state after the browser is already on the page, then assert the visible UI
    changes without a manual reload.

## Validation Summary

- `pnpm --dir web check`, `pnpm --dir web build`, `git diff --check`, and
  `scripts/install-dev-harness` all exited `0`; the durable transcript lives in
  `docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/command-validation.txt`.
- Focused browser validation against a temporary live `harness ui` runtime
  recorded `status-live-update-ok`, `stale-state-ok`,
  `visibility-catchup-ok`, and `timeline-live-update-ok`; the durable output
  lives in
  `docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/browser-validation.txt`.
- A focused reopen follow-up against the revision-2 candidate also recorded
  `fast-success-no-blink-ok` and `fast-failure-no-blink-ok`, proving that the
  buffered freshness indicator no longer flashes `Updating` during fast local
  success/failure paths.
- `docs/plans/active/supplements/2026-04-16-make-harness-ui-refresh-live-state/validation-evidence.md`
  ties those command and browser artifacts back to the candidate behaviors this
  slice claims to validate.
- `scripts/ui-playwright-smoke` now covers same-session status and timeline
  live-update behavior plus an isolated visibility-regain catch-up assertion.
  The broader smoke flow still hits an older unrelated timeline snapshot
  assertion outside this slice, so the tracked focused browser run remains the authoritative
  end-to-end evidence for the final candidate.

## Review Summary

- `review-001-full` requested changes for one catch-up correctness gap and two
  validation gaps in the initial live-refresh implementation.
- `review-002-delta` narrowed the remaining repair scope to the in-flight guard
  bug and the visibility-catch-up test isolation gap after the first repair.
- `review-003-delta` passed cleanly after the catch-up coordination and browser
  validation repairs, closing step review for the implementation slice.
- `review-004-full` requested changes during finalize review because the plan's
  command/browser validation claims were not yet backed by tracked durable
  evidence.
- `review-005-delta` passed cleanly after adding the tracked supplement package
  and explicit plan references for that evidence repair.
- `review-006-full` passed cleanly as the final archive-candidate review, with
  no correctness or tests findings remaining.
- Revision 2 reopened the archived candidate for one narrow UX repair after PR
  feedback about the freshness pill blinking on fast local refreshes.
- `review-007-full` passed for revision 2 with no blocking findings and one
  minor tests note: browser coverage now proves the no-blink fast paths, but it
  still does not positively assert the slower delayed-`Updating` path.

## Archive Summary

- Archived At: 2026-04-17T11:51:29+08:00
- Revision: 2
current reopen repair candidate waiting to be re-archived.
- PR: existing PR `#180` (`https://github.com/catu-ai/easyharness/pull/180`)
  already tracks this candidate and should be updated in place after the
  revision-2 archive move is committed and pushed.
- Ready: Revision 2 keeps the accepted live-refresh behavior, rebuilds the
  embedded bundle with the buffered freshness-indicator UX repair, refreshes the
  tracked command/browser evidence, and passes `review-007-full` with one minor
  non-blocking coverage note.
- Merge Handoff: Re-archive the revision-2 candidate, commit the tracked plan
  move plus refreshed closeout notes, push branch
  `codex/issue-158-live-ui-refresh` to update PR `#180`, then refresh publish,
  CI, and sync evidence until `harness status` returns to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- The workbench now uses a shared live-refresh loop that keeps `/api/status`
  and the active page (`plan`, `timeline`, or `review`) current while the tab
  is visible, plus an immediate catch-up refresh on focus or visibility regain.
- The shared shell now surfaces explicit freshness state so operators can tell
  when the UI is updating normally, background-refreshing, stale after refresh
  failures, or disconnected on first-load failure.
- The catch-up path now aborts superseded in-flight requests safely and keeps
  the latest request responsible for clearing the shared in-flight guard.
- Revision 2 adds a small activity buffer before switching an already-live
  freshness pill into `Updating`, so fast local refreshes stay visually stable
  on `Live` and fast failures move directly to `Stale`.
- `scripts/ui-playwright-smoke` and the tracked supplement evidence now prove
  same-session status/timeline updates, stale-state signaling, and
  visibility-regain recovery without a manual page reload.
- The embedded static UI bundle under `internal/ui/static/` was rebuilt to ship
  the reviewed frontend behavior through `harness ui`.

### Not Delivered

- Route-specific polling cadences, backoff tuning, or other per-page refresh
  policy refinements beyond the shared baseline shipped here.
- Push-driven transport such as websocket or SSE updates; this slice stays with
  polling plus focus/visibility catch-up.
- Broader stale-state recovery affordances beyond the shell-level freshness
  indicator and the retry behavior already needed for this slice.
- A dedicated positive browser assertion for the slower delayed-`Updating` path
  is still missing; `review-007-full` recorded that as a minor non-blocking
  validation gap.

### Follow-Up Issues

- Issue `#179`: tune workbench live-refresh policy after issue `#158` rollout,
  including polling cadence evaluation, possible route-specific refresh policy
  changes, and any evidence-based push/stale-recovery follow-up.
