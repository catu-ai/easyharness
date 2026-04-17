# Validation Evidence

## Command Validation

Durable command output for the candidate now lives in
`command-validation.txt`. That transcript records:

- `pnpm --dir web check` exit `0`
- `pnpm --dir web build` exit `0`, including the rebuilt embedded asset names
  `index-BU_Nt-jg.js` and `index-DW5YH7Wr.css`
- `git diff --check` exit `0`
- `scripts/install-dev-harness` exit `0`

## Browser Validation

Durable focused browser evidence now lives in `browser-validation.txt`. That
same-session temporary-runtime run recorded these pass markers:

- `status-live-update-ok`
- `stale-state-ok`
- `visibility-catchup-ok`
- `timeline-live-update-ok`
- `fast-success-no-blink-ok`
- `fast-failure-no-blink-ok`

Those markers correspond to the final candidate behavior:

- the already-open `Status` page updates itself when the underlying harness
  node changes
- the topbar freshness state becomes `Stale` when refreshes fail while prior
  data remains visible
- the visibility-regain catch-up path succeeds with a single-use fetch gate so
  ordinary polling cannot satisfy that assertion by accident
- the already-open `Timeline` page also updates without a manual reload after a
  new event is appended
- quick successful focus-triggered refreshes stay on `Live` instead of briefly
  flashing `Updating`
- quick failed focus-triggered refreshes move directly from `Live` to `Stale`
  without a transient `Updating` blink

The broader `scripts/ui-playwright-smoke` script still hits an older unrelated
timeline snapshot assertion outside this fix, so the focused live-session
markers above are the authoritative validation evidence for the reopen repair.
