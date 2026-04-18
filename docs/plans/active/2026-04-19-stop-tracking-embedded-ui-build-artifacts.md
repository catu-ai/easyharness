---
template_version: 0.2.0
created_at: "2026-04-19T00:02:15+08:00"
approved_at: "2026-04-19T00:05:11+08:00"
source_type: direct_request
source_refs: []
size: L
---

# Stop Tracking Embedded UI Build Artifacts

## Goal

Shift the repository from checking in generated UI bundles under
`internal/ui/static/` to treating the frontend as a normal build input. After
this change, UI pull requests should primarily review source edits under
`web/` plus any supporting scripts/tests/docs, while the generated embedded
assets are rebuilt locally during developer bootstrap and rebuilt in CI and
release automation before Go compilation.

The shipped `harness` binary should remain self-contained for end users. The
contract change applies to repository contributors and automation, not to
people installing released binaries.

## Scope

### In Scope

- Stop tracking generated bundle files under `internal/ui/static/` and replace
  them with a generated-at-build-time contract suitable for `//go:embed`.
- Update developer bootstrap so `scripts/install-dev-harness` prepares frontend
  dependencies/build output and fails with actionable guidance when `node` or
  `pnpm` is missing.
- Update CI and release automation so frontend assets are built before Go
  tests/builds consume embedded UI assets.
- Update docs and smoke/workflow tests so the new contract is explicit and
  remains stable.

### Out of Scope

- Replacing Vite, Preact, or the current embedded-UI architecture.
- Making `harness ui` itself auto-install Node dependencies or silently rebuild
  the frontend on demand.
- Broader frontend refactors unrelated to the artifact-tracking contract.

## Acceptance Criteria

- [x] The repository no longer tracks generated UI bundle files under
      `internal/ui/generated/build/`, and UI changes are reviewed primarily via
      source files plus workflow/script changes.
- [x] `scripts/install-dev-harness` prepares the embedded UI assets as part of
      developer bootstrap and stops with clear installation guidance when
      required frontend tooling is missing.
- [x] CI and release automation build the frontend before `go test` or release
      packaging so the released `harness` binary still embeds working UI
      assets.
- [x] Documentation and smoke/workflow tests describe and validate the new
      developer/build contract without relying on hidden chat context.

## Deferred Items

- Automatic fallback rebuilds from inside `harness ui` itself if generated UI
  assets are missing at runtime.

## Work Breakdown

### Step 1: Establish the generated embedded-UI contract

- Done: [x]

#### Objective

Define one clean repository contract for embedded UI assets: frontend source is
tracked, generated bundles are not, and Go build/test flows consume generated
assets from a known location.

#### Details

Remove the current practice of checking minified Vite output into git while
preserving the existing embedded UI serving model. The execution path should
keep one canonical build output location for the embedded assets, update git
tracking/ignore rules accordingly, and leave future agents with a clear answer
for how a clean checkout becomes buildable again.

#### Expected Files

- `.gitignore`
- `internal/ui/server.go`
- `internal/ui/generated/`
- `web/vite.config.ts`

#### Validation

- A clean worktree no longer shows generated UI bundle files as tracked source.
- The chosen output location still works with the embedded server contract once
  assets have been generated.
- Any tracked-file deletions or ignore rules are deliberate and documented in
  the plan execution notes.

#### Execution Notes

Moved the Vite build output from tracked `internal/ui/static/` into generated
`internal/ui/generated/build/`, switched `go:embed` to the generated root, and
left a tracked README anchor so clean checkouts still have a stable embed path.
TDD note: this slice changed a shared workflow contract across build scripts,
fixtures, docs, and embedded assets, so strict Red/Green by micro-step was not
practical; focused validation and repo-level smoke coverage were used instead.
After `review-001-full` requested changes, the repair pass moved the
Playwright smoke scripts onto `scripts/build-embedded-ui`, added an explicit
`node` preflight ahead of `pnpm`, cleared inherited generated assets from the
installer/release checkout fixtures, and tightened workflow smoke assertions so
they verify build-before-test and build-before-package ordering directly.

#### Review Notes

PASSED via `review-001-full` follow-up repair and `review-002-delta` on
2026-04-19. The initial full closeout review requested five blocking fixes
around shared builder usage, missing-`node` guidance, clean-checkout smoke
proof, and workflow-order assertions; the repair commit addressed those items
and the delta follow-up passed with zero findings.

### Step 2: Update local developer bootstrap and failure modes

- Done: [x]

#### Objective

Make `scripts/install-dev-harness` the supported local bootstrap path for both
the Go binary and generated embedded UI assets.

#### Details

The installer should prepare frontend dependencies/build output before building
the repo-local `harness` binary, and it should stop early with concise guidance
when required tools such as `node` or `pnpm` are unavailable. Any installer
tests that currently assume Go-only bootstrap need to absorb the frontend build
contract and cover the new failure messaging.

#### Expected Files

- `scripts/install-dev-harness`
- `tests/smoke/install_dev_harness_test.go`
- `docs/development.md`

#### Validation

- Running `scripts/install-dev-harness` in a prepared environment produces both
  embedded UI assets and a working repo-local `harness` binary.
- Missing-tool failures point contributors at the required `node`/`pnpm`
  installation step instead of failing later in `go build`.
- Documentation matches the new bootstrap order.

#### Execution Notes

Added `scripts/build-embedded-ui` as the shared local build entrypoint and made
`scripts/install-dev-harness` call it before rebuilding the repo-local binary.
Installer smoke fixtures now copy the frontend build inputs explicitly instead
of dragging local `node_modules` state into the test worktree.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This automation slice was implemented and validated as
part of the same integrated generated-artifact contract change covered by the
Step 1 closeout review.

### Step 3: Move CI and release automation onto the same frontend-first build path

- Done: [x]

#### Objective

Teach repository automation to install frontend tooling, build the embedded UI
assets, and only then run Go tests or release packaging.

#### Details

The CI workflow and release workflow should gain the Node/pnpm setup required
to build `web/` deterministically from the committed lockfile, then run the
existing Go test/build stages against generated embedded assets. Release
packaging should continue producing a self-contained binary for end users, and
workflow-oriented tests should be updated if they assert the old Go-only shape.

#### Expected Files

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `scripts/build-release`
- `tests/smoke/homebrew_formula_test.go`
- `tests/smoke/release_build_test.go`

#### Validation

- CI builds frontend assets before `go test ./...`.
- Release automation builds frontend assets before release archives are
  produced.
- Existing workflow tests or new assertions verify the presence of the new
  frontend setup/build stages.

#### Execution Notes

Updated CI and release automation to install Node.js, enable Corepack, build
the embedded UI assets, and only then run Go tests or release packaging.
Release-workflow smoke assertions and checkout helpers now follow the same
frontend-first path.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This automation slice was implemented and validated as
part of the same integrated generated-artifact contract change covered by the
Step 1 closeout review.

### Step 4: Prove the repository works without tracked bundle artifacts

- Done: [x]

#### Objective

Close the slice with repo-visible validation that the new contract works in the
same ways contributors and automation will use it.

#### Details

Validation should cover the frontend type/build path, local bootstrap, Go test
coverage, and at least one browser-level smoke path that exercises the embedded
UI served by the rebuilt binary. The final repository state should make it easy
for future contributors to understand that generated UI assets come from the
bootstrap/build pipeline rather than from checked-in bundle files.

#### Expected Files

- `docs/development.md`
- `scripts/ui-playwright-smoke`
- `docs/plans/active/2026-04-19-stop-tracking-embedded-ui-build-artifacts.md`

#### Validation

- `pnpm --dir web check` and the chosen frontend build command pass.
- `scripts/install-dev-harness` passes and produces a working local binary.
- `go test ./... -count=1` passes after the generated-asset transition.
- A UI smoke path against the rebuilt binary passes without relying on tracked
  bundle artifacts.

#### Execution Notes

Validated the generated-asset contract with `scripts/build-embedded-ui`,
`scripts/install-dev-harness`, `pnpm --dir web check`, `go test ./... -count=1`,
`scripts/build-release --version "v$(cat VERSION)" --output-dir
.local/release-ui-artifacts-check --platform "$(go env GOOS)/$(go env GOARCH)"`,
and `scripts/ui-playwright-smoke`. The post-review repair pass revalidated with
`go test ./tests/smoke -count=1`, `scripts/ui-playwright-smoke`, and
`go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This validation slice is itself the repo-level proof for
the integrated generated-artifact contract change covered by the Step 1
closeout review.

## Validation Strategy

- Use the committed `web/pnpm-lock.yaml` as the single frontend dependency
  contract in local bootstrap and automation.
- Validate both the frontend source path (`pnpm --dir web check` plus frontend
  build) and the embedded-binary path (`scripts/install-dev-harness`,
  `go test ./... -count=1`, targeted UI smoke).
- Update workflow-oriented tests when automation shape changes so later agents
  get direct failure signals if the CI/release contract regresses.

## Risks

- Risk: A clean checkout may become confusing if contributors run Go commands
  before the frontend build prerequisites are satisfied.
  - Mitigation: make `scripts/install-dev-harness` the explicit local bootstrap
    path, emit actionable missing-tool/build guidance, and update developer
    docs to state the new contract clearly.
- Risk: CI/release could drift from local bootstrap and produce binaries with
  stale or missing embedded assets.
  - Mitigation: align local bootstrap, CI, and release around the same
    frontend-first build sequence and add workflow/test coverage for that
    sequence.
- Risk: Removing tracked bundles could accidentally break the current embed
  path or leave partially tracked generated files behind.
  - Mitigation: keep one canonical generated output location, remove tracked
    artifacts deliberately, and validate the rebuilt binary through UI smoke
    coverage before closeout.

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
