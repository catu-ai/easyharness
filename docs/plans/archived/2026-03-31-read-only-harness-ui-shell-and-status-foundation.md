---
template_version: 0.2.0
created_at: "2026-03-31T23:26:12+08:00"
source_type: issue
source_refs:
    - '#2'
    - '#70'
    - https://github.com/catu-ai/easyharness/pull/86
size: L
---

# Build the read-only harness UI shell and live status foundation

## Goal

Land the first executable slice of `harness ui` as a self-contained local
workbench that preserves the product promise of "agent executes, human
steers". The first slice should prove the command shape, the embedded-web
distribution model, the thin Go server boundary, and the page-oriented
workbench shell without overcommitting to every UI detail from the current
proposal.

This slice should intentionally stop at a narrow but real milestone: a new
`harness ui` command starts a local read-only UI, serves embedded frontend
assets from the `harness` binary, and renders a live `Status` page backed by
real data. The remaining top-level pages should exist as stable shell routes,
but they may remain structured WIP placeholders until the contracts and
product decisions around their data surfaces are ready.

## Scope

### In Scope

- Add the `harness ui` command with a small, explicit local-server flag set.
- Implement a thin Go UI server that serves embedded frontend assets plus a
  resource-first read-only API surface.
- Add a frontend subproject under `web/` using the accepted build pipeline and
  wire its production build into the Go binary.
- Build the shared UI shell for the left navigation rail and top-level pages:
  `Status`, `Timeline`, `Review`, `Diff`, and `Files`.
- Hook the `Status` page to real data from existing harness-owned sources and
  keep the other pages as clearly marked WIP placeholders.
- Establish testing and validation expectations for both Go/unit coverage and
  comprehensive browser coverage using the
  [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill.
- Plan the expected follow-up issues for deferred page hookups and adjacent UI
  concerns so archive does not leave vague future work.

### Out of Scope

- Any UI-triggered write actions, command execution, or state mutation.
- Full implementation of live `Timeline`, `Review`, `Diff`, or `Files` data
  views.
- Remote PR/CI/sync integration beyond what the existing local harness state
  already exposes.
- Long-lived client-side persistence beyond minimal ephemeral UI state such as
  the selected page or theme-free shell layout.
- Final visual polish, dense interaction tuning, or complete resolution of all
  open proposal questions.

## Acceptance Criteria

- [x] The CLI exposes `harness ui` with the agreed read-only local-workbench
      behavior and minimal flags for `--host`, `--port`, and `--no-open`.
- [x] The shipped `harness` binary remains self-contained: production UI
      assets are embedded into the Go binary, and running `harness ui` does
      not require Node.js, pnpm, or any external frontend runtime.
- [x] The repository gains a clearly isolated `web/` frontend subproject plus
      Go-side UI serving code without muddying the existing repo structure.
- [x] The UI shell renders stable top-level routes for `Status`, `Timeline`,
      `Review`, `Diff`, and `Files`, with `Status` backed by real read-only
      data and the remaining pages clearly labeled as WIP placeholders.
- [x] The `Status` page renders real values for summary, current node, next
      actions, warnings, blockers, and the highest-signal facts/artifacts that
      the current contracts already expose.
- [x] Focused Go/unit tests and comprehensive browser validation both exist for
      the new UI slice, and the plan documents how future execution should use
      the [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md)
      skill with isolated ports/workdirs/artifact paths when parallel browser
      checks use multiple subagents.
- [x] The tracked plan explicitly names the deferred UI follow-up issues that
      must be created or updated before archive if those slices remain
      unimplemented.

## Deferred Items

- Hook real data into `Timeline`, including command trajectory, inputs,
  outputs, and state transitions beyond the status-derived summary.
- Hook real review round data into `Review`, including findings navigation,
  manifest/ledger/submission tabs, and aggregate interpretation.
- Hook worktree diff data and file browsing data into `Diff` and `Files` with
  production-grade viewers rather than WIP placeholders.
- Add a thin persistent global status strip only after the standalone `Status`
  page proves what must stay globally visible.
- Refine the visual system, information density, and navigation affordances
  after the live shell is usable.
- Track binary-size budget and embedded-asset budget as explicit follow-up work
  if the first slice materially changes release size.

## Work Breakdown

### Step 1: Define the `harness ui` contract, repo boundaries, and follow-up issue map

- Done: [x]

#### Objective

Lock the user-facing command shape, runtime constraints, frontend/build
boundary, and named deferred issue buckets before implementation starts.

#### Details

This step should write the accepted discovery outcomes back into tracked docs
or code comments where needed so later execution does not depend on session
memory. The command name should stay `harness ui`, not `web` or `dashboard`,
and the initial flags should stay deliberately small: `--host`, `--port`, and
`--no-open`. The default behavior should bind locally, choose a safe default
port policy, and open the UI automatically unless disabled.

This step should also record the architectural split: `web/` is an isolated
frontend subproject for build-time assets only; Go remains the runtime entry
surface and the single shipped binary. The server/API boundary should stay
resource-first and read-only. Before archive, any remaining deferred product
surface must be turned into concrete GitHub issues rather than left as vague
"later" notes. The expected follow-up issue buckets are:

- live `Timeline` data hookup
- live `Review` data hookup
- live `Diff` and `Files` data hookup
- binary-size and embedded-asset budget follow-up if needed
- visual polish / interaction-density follow-up after real usage

#### Expected Files

- `docs/plans/active/2026-03-31-read-only-harness-ui-shell-and-status-foundation.md`
- `README.md`
- `docs/specs/cli-contract.md`
- Go-side UI contract notes if implementation chooses a dedicated spec or
  package doc

#### Validation

- A future agent can infer the command behavior, repo organization, and API
  stance from tracked material alone.
- The plan clearly names the follow-up issue buckets that must be created or
  updated before archive when those slices remain deferred.

#### Execution Notes

Locked the command and product boundary in tracked docs before the UI runtime
landed. README now lists `harness ui` as a first-class command, explains that
the first slice is read-only, and documents the `pnpm --dir web install &&
pnpm --dir web build` flow for refreshing embedded UI assets. The CLI contract
now includes `harness ui` in the command surface, records it as a plain-text
local-server command rather than a JSON envelope, and defines the initial
`--host`, `--port`, and `--no-open` contract. The plan itself also names the
required follow-up issue buckets for deferred pages, size-budget work, and
visual refinement so archive will not leave vague future work.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this contract/documentation step was developed as part
of one integrated UI-foundation slice and will receive a full finalize review
before archive.

### Step 2: Add the Go UI entrypoint, local server, and embedded asset pipeline

- Done: [x]

#### Objective

Create the runtime backbone for `harness ui`: command parsing, local server
startup, embedded production assets, and a minimal read-only API surface.

#### Details

The Go runtime should remain the product entrypoint. `harness ui` should serve
the embedded frontend build plus the minimal resource APIs needed for the first
slice. The initial API should stay resource-oriented and narrow, with `Status`
as the only page that must consume real data in this phase. The implementation
must keep the shipped artifact self-contained: frontend build output is bundled
into `harness`, and runtime should not shell out to Node/pnpm.

This step should also define the development/build workflow cleanly enough that
contributors can build the frontend assets locally without guessing how they
become embedded. If the chosen implementation introduces a generated asset
directory, keep it obviously derived and easy to rebuild.

#### Expected Files

- `internal/cli/app.go`
- new Go UI package(s) under `internal/`
- frontend asset embedding package(s) or files
- build or helper scripts if needed
- `cmd/harness/*` if the UI entrypoint needs additional glue

#### Validation

- `harness ui --help` explains the intended flags and read-only behavior.
- The UI server can start locally and serve the embedded app shell.
- Production UI assets are built and embedded into the Go binary rather than
  loaded from an external runtime at execution time.

#### Execution Notes

Added `internal/ui/` as the thin Go UI layer. The new server starts from the
`harness ui` command, binds a local address, prints the resolved URL, supports
`--host`, `--port`, and `--no-open`, serves embedded static assets, exposes
`GET /api/status`, and falls back to `index.html` for SPA routes. The status
endpoint now keeps the JSON error body but returns `503 Service Unavailable`
when `harness status` itself fails, so the browser can surface real status-read
errors instead of silently rendering a fake success state. The frontend build
now targets `internal/ui/static/`, which keeps the shipped binary
self-contained while leaving `web/` as a clean build-time source tree only.
Reinstalled the repo-local `harness` binary through `scripts/install-dev-harness`
after the Go CLI changes so direct `harness` commands now reflect the new `ui`
surface.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this runtime/backbone step was developed as part of one
integrated UI-foundation slice and will receive a full finalize review before
archive.

### Step 3: Build the SPA shell and live Status page

- Done: [x]

#### Objective

Ship the multi-page workbench shell and a real `Status` page while leaving the
other top-level pages as intentional WIP placeholders.

#### Details

This step should create the shared shell for the top bar, left rail, route
selection, and page layout. `Status` should render the real harness data that
already exists today: summary, current node, next actions, warnings, blockers,
and the highest-signal facts/artifacts. `Timeline`, `Review`, `Diff`, and
`Files` should exist as real routes with stable layout and clear WIP messaging
so the product skeleton is visible without pretending that the deeper data
contracts are already settled.

The implementation should keep the UI read-only and avoid speculative client
state or elaborate front-end persistence. Minimal shell state such as active
page selection is fine; durable product state remains owned by existing harness
contracts and runtime artifacts.

#### Expected Files

- `web/package.json`
- `web/pnpm-lock.yaml`
- `web/vite.config.*`
- `web/tsconfig.json`
- `web/index.html`
- `web/src/*`
- any Go-side API/view-model files needed for Status

#### Validation

- The shell renders the five accepted top-level pages.
- The Status page displays real harness data from the current worktree.
- The WIP pages are explicit enough that users understand they are structure,
  not broken implementations.

#### Execution Notes

Created the `web/` frontend subproject with `Preact + TypeScript + Vite`,
including the shared shell, left-rail navigation, top-level page routes, and a
flat workbench visual system. `Status` now fetches `/api/status` and renders
summary, `current_node`, next actions, warnings, blockers, facts, artifacts,
and errors defensively. The embedded shell now also injects the current
worktree path and repo label into the top chrome, and the SPA route handling
prefers real pathname routes like `/review` and `/timeline` instead of relying
on hash-only restoration. `Timeline`, `Review`, `Diff`, and `Files` are real
routes with explicit WIP messaging so the shell is stable without pretending
their deeper data hookups are already complete.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this shell/status step was developed as part of one
integrated UI-foundation slice and will receive a full finalize review before
archive.

### Step 4: Validate the UI slice with focused unit coverage and comprehensive browser checks

- Done: [x]

#### Objective

Prove that the first UI slice is reliable enough to continue building on and
capture the testing pattern future UI work should follow.

#### Details

In addition to focused Go/unit coverage for CLI parsing, server behavior, API
responses, and embed/build glue, this step should establish comprehensive
browser validation for the new UI slice using the
[$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill. The
goal is not just one smoke click: the browser checks should exercise startup,
navigation between the five top-level pages, real `Status` rendering, and the
WIP placeholder behavior.

When future execution uses subagents to parallelize browser checks, the plan
must keep the runs isolated. Each parallel run should use distinct listening
ports, distinct browser artifact output paths (for example under
`output/playwright/`), and worktree/runtime inputs that avoid shared mutable
state collisions. Parallel browser validation is allowed, but only with
explicit isolation so one run cannot corrupt another run's screenshots, traces,
or local UI server lifecycle.

#### Expected Files

- Go tests under `internal/` and/or `tests/`
- frontend test/build validation glue if needed
- browser-validation notes or helper scripts if implementation adds them

#### Validation

- Go/unit tests cover the new CLI parsing, server lifecycle, and API behavior.
- Browser validation covers UI startup, route navigation, and real Status page
  rendering using the
  [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill.
- The validation notes make parallel subagent isolation explicit for future
  comprehensive browser checks.

#### Execution Notes

Added focused Go coverage for the UI server and CLI surface, then ran
`go test ./internal/cli ./internal/ui` and the full `go test ./...` suite.
Validated the frontend source with `pnpm --dir web check` and rebuilt the
embedded assets with `pnpm --dir web build`. For comprehensive browser
coverage, added `scripts/ui-playwright-smoke` and exercised the live UI through the
[$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) workflow:
started the repo-local `harness` binary, waited for the live `Status` page to
render the current `/api/status` summary and `current_node`, and confirmed
that `Timeline`, `Review`, `Diff`, and `Files` each resolve to clear WIP
placeholder routes. The same smoke helper now also validates the browser-side
failure path by holding the real plan-local state mutation lock, confirming
that `/api/status` returns `503`, and asserting that the rendered Status page
surfaces the resulting error message. The validation strategy now explicitly
encodes port/output/runtime isolation for future subagents: the smoke helper
defaults to an auto-selected port, supports a workdir override, pins the
repo-local binary under that workdir, and uses a dedicated Playwright session
plus temp cwd so stale listeners or shared scratch state do not hide route or
asset regressions.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this validation step was developed as part of one
integrated UI-foundation slice and will receive a full finalize review before
archive.

## Validation Strategy

- Use focused Go/unit tests for CLI parsing, local server startup/shutdown, API
  responses, and frontend asset embedding behavior.
- Use frontend build verification to ensure the `web/` subproject produces
  deterministic embedded assets for the Go binary.
- Use comprehensive browser validation via the
  [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill to
  exercise the live workbench in a real browser rather than relying only on
  static snapshots.
- When browser checks run in parallel via subagents, isolate ports, output
  directories, traces/screenshots, and any mutable local runtime state to keep
  the runs reproducible.
- Before archive, create or update explicit follow-up GitHub issues for any
  deferred page hookups, size-budget work, or post-MVP UI refinements that
  remain outstanding.

## Risks

- Risk: The embedded-web pipeline could add enough build/process complexity
  that the repo starts to feel split-brained between Go and frontend concerns.
  - Mitigation: Keep `web/` isolated, keep Go as the runtime entrypoint, and
    document the build/embed flow explicitly.
- Risk: The Status page may grow into a page-specific reinterpretation of
  workflow state instead of a richer view over existing contracts.
  - Mitigation: Keep the first live page narrowly grounded in `harness status`
    plus existing facts/artifacts rather than inventing new persistent state.
- Risk: Browser validation can become flaky or collide when multiple agents run
  it in parallel.
  - Mitigation: Use the
    [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill,
    require explicit isolation for ports and artifact paths, and treat
    comprehensive browser checks as first-class validation rather than an
    ad-hoc afterthought.
- Risk: Deferring four of the five pages could leave archive with vague future
  intent instead of durable next work.
  - Mitigation: Require concrete follow-up issue creation or updates before
    archive for each deferred slice that remains undone.

## Validation Summary

Reopen revision 2 revalidated the UI shell after the post-archive visual
feedback loop and the follow-up smoke-helper isolation repair.

- `pnpm --dir web build`
- `pnpm --dir web check`
- `go test ./...`
- `scripts/install-dev-harness` after Go/embed changes so the repo-local
  `harness` binary matched the current worktree
- `bash scripts/ui-playwright-smoke`, which now:
  - rebuilds the embedded UI assets and refreshes the repo-local `harness`
    binary before browser validation
  - defaults to an isolated runtime snapshot instead of mutating the shared
    checkout-local `.local/harness` state
  - verifies healthy live `Status` rendering against `/api/status`
  - verifies left-rail navigation plus deep-link rendering for `Timeline`,
    `Review`, `Diff`, and `Files`
  - verifies lock-induced `503` failure rendering for `Status`
  - verifies the Vite dev-mode mount against a live backend via
    `HARNESS_UI_API_TARGET`

## Review Summary

Revision 2 used two finalize review rounds after the archived candidate was
reopened for UI polish.

- `review-001-full`: changes requested for incorrect worktree metadata, broken
  pathname restoration, and missing repo-local browser validation
- `review-002-full`: changes requested for status failure semantics and weak
  browser-validation coverage/isolation
- `review-003-delta`: passed after the bounded repair for status errors,
  stricter smoke assertions, and Playwright session isolation
- `review-004-full`: changes requested for missing tracked follow-up issue refs
  and missing browser failure-state validation
- `review-005-delta`: passed after adding concrete follow-up issue refs and the
  lock-based browser failure-path check
- `review-006-full`: changes requested for the dev-mode metadata token crash
  and the `/api` namespace leak
- `review-007-full`: changes requested for an undocumented frontend dev
  workflow and missing dev-mode mount validation
- `review-008-full`: changes requested for missing rail-click navigation
  coverage and default parallel-collision risk in the smoke helper
- `review-009-full`: full finalize review passed with no findings
- `review-011-full`: changes requested after reopen for the hardcoded
  `repoName` regression plus missing proof that rebuilt embedded assets and
  helper prerequisites were covered by the validation path
- `review-012-full`: full finalize review found no correctness regressions in
  the refreshed VS Code-like shell, but it did request one more repair so the
  smoke helper would stop using shared checkout-local harness state by default
- `review-013-delta`: passed after isolating the smoke helper's runtime
  workdir while preserving embedded-asset rebuild coverage, failure-path
  coverage, and dev-mode validation

## Archive Summary

- Archived At: 2026-04-01T10:33:36+08:00
- Revision: 2
Revision 2 is archive-ready again after `review-013-delta` passed on
2026-04-01T10:32:18+08:00.

- Reopen Mode: `finalize-fix`
- PR: [#96](https://github.com/catu-ai/easyharness/pull/96)
- Ready: the reopened candidate restored the dynamic repo metadata contract,
  rebuilt the shell into a flatter VS Code-like three-column layout, tightened
  browser validation, and removed shared-checkout runtime-state collisions
  from `scripts/ui-playwright-smoke`.
- Merge Handoff: run `harness archive`, commit the tracked archive move plus
  the refreshed plan summaries and UI/script updates on
  `codex/read-only-harness-ui-shell`, push the branch, refresh PR #96, and
  record publish/CI/sync evidence until `harness status` returns
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

Revision 2 kept the MVP scope the same but materially improved the quality and
resume-ability of the read-only UI slice.

- Added `harness ui` as a read-only local workbench command with `--host`,
  `--port`, and `--no-open`.
- Added `internal/ui/` as the thin Go UI server that serves embedded frontend
  assets, exposes `/api/status`, injects worktree/repo metadata into the shell,
  and returns `503` JSON responses when status resolution fails.
- Added the isolated `web/` frontend subproject with `Preact + TypeScript +
  Vite`, including the shared shell, top rail, page routes, and a live `Status`
  page backed by real `harness status` data.
- Refined the shell into a flatter, VS Code-like three-column workbench with
  an icon-only activity bar, page-specific sidebar, and editor pane whose
  detail view follows the selected `Status` section.
- Restored the metadata contract so the embedded UI continues to inject the
  current workdir and derived repo label while separately branding the product
  as `easyharness`.
- Added explicit WIP pages for `Timeline`, `Review`, `Diff`, and `Files` so
  the workbench structure is visible without pretending their deeper data
  surfaces are already settled.
- Added focused Go coverage for the new CLI/UI surface plus a comprehensive
  Playwright smoke helper that now covers healthy status rendering, lock-based
  failure rendering, SPA rail navigation, WIP deep links, and the Vite dev-mode
  mount path.
- Updated the Playwright smoke helper so each run gets its own browser session
  workdirs and an isolated default runtime snapshot, which keeps future
  parallel browser runs from contending on the shared checkout-local plan lock.
- Documented the frontend build/dev workflow in tracked docs, including the
  repo-local harness rebuild flow and the proxy-compatible `pnpm --dir web
  dev:harness` path.
- Created concrete follow-up GitHub issues for the deferred Timeline, Review,
  Diff/Files, size-budget, and visual-polish slices.

### Not Delivered

- The `Timeline` page still does not render real trajectory data; it remains a
  deliberate WIP placeholder tracked by [#93](https://github.com/catu-ai/easyharness/issues/93).
- The `Review` page still does not render live review-round artifacts or
  findings; that follow-up lives in [#95](https://github.com/catu-ai/easyharness/issues/95).
- The `Diff` and `Files` pages still do not render real worktree/file-browser
  data; that work remains in [#91](https://github.com/catu-ai/easyharness/issues/91).
- Binary-size budgeting and embedded-asset budget enforcement remain deferred
  to [#92](https://github.com/catu-ai/easyharness/issues/92).
- Visual density, chrome refinement, and any eventual global status strip
  remain deferred to [#94](https://github.com/catu-ai/easyharness/issues/94).

### Follow-Up Issues

- [#93](https://github.com/catu-ai/easyharness/issues/93) Hook real timeline
  data into harness UI
- [#95](https://github.com/catu-ai/easyharness/issues/95) Hook review round
  data into harness UI
- [#91](https://github.com/catu-ai/easyharness/issues/91) Hook diff and file
  browser data into harness UI
- [#92](https://github.com/catu-ai/easyharness/issues/92) Track harness UI
  embedded asset and binary size budget
- [#94](https://github.com/catu-ai/easyharness/issues/94) Refine harness UI
  visual density and navigation after MVP usage
