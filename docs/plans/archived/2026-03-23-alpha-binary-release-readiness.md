---
template_version: 0.2.0
created_at: "2026-03-23T23:28:51+08:00"
source_type: direct_request
source_refs: []
---

# Prepare the first alpha binary release

## Goal

Prepare `superharness` for its first public alpha release as a binary-first
GitHub Release. The result should let an external user download a versioned
archive, verify the published checksum, run `harness --version` against the
released binary, and follow the README to a successful first run without
needing the development-only installer.

This slice should stay deliberately narrow. It should add only the release
infrastructure, versioning, documentation, and public-project guardrails
needed for an honest alpha release. Homebrew, richer release metadata, and
broader distribution polish remain deferred.

## Scope

### In Scope

- Define the first public alpha release shape around GitHub Releases with
  prebuilt archives and checksums.
- Decide and document the supported alpha platform set and contributor build
  baseline.
- Extend version/build metadata so release binaries report a human-meaningful
  release identifier alongside commit-oriented diagnostics.
- Add a repo-owned release build path that can produce deterministic release
  artifacts locally and from GitHub Actions.
- Add CI automation that runs the repository test suite for normal changes.
- Update public docs for install, verification, and first-run guidance for the
  alpha binary release.
- Add a repository `LICENSE` so the public release terms are explicit.

### Out of Scope

- Homebrew or other package-manager distribution.
- Windows release packaging in the first alpha slice.
- Making `go install` or source builds the primary release path for external
  users.
- Optional JSON or richer provenance output for `harness --version`.
- Workflow-state follow-ups unrelated to release readiness, including broader
  PR/CI modeling and deferred UI work.
- Folding issue `#7` into this slice; that fix is expected to land separately
  and must be pulled before execution begins.

## Acceptance Criteria

- [x] The repository documents a first public alpha release contract that uses
      GitHub Releases with prebuilt archives and checksums, and it explicitly
      defers Homebrew.
- [x] Release binaries report a stable human-facing release identifier in
      `harness --version` without regressing the current dev/release diagnostic
      behavior.
- [x] A repo-owned release build path can produce the supported alpha
      artifacts and checksum output deterministically.
- [x] GitHub Actions runs `go test ./...` for ordinary changes and provides a
      documented path to publish release artifacts.
- [x] README and adjacent public docs let a new external user install and
      verify the alpha binary release without relying on
      `scripts/install-dev-harness`.
- [x] The repository includes an explicit `LICENSE` file appropriate for the
      public alpha release.

## Deferred Items

- Homebrew publishing and tap maintenance.
- Windows release artifacts.
- Optional `harness --version --json` support and richer build metadata beyond
  the first release identifier.
- Broader source-install ergonomics beyond documenting the contributor build
  baseline.
- Additional hardening follow-ups such as fuzzing and resilience suites once
  the first alpha release path exists.

## Work Breakdown

### Step 1: Capture the alpha release contract in tracked form

- Done: [x]

#### Objective

Record the accepted discovery decisions for the first public alpha so
execution can proceed from the tracked plan without relying on chat memory.

#### Details

Discovery converged on a binary-first alpha release rather than a source-first
release or a broader distribution push. The first release should center on
GitHub Releases with prebuilt archives plus checksums, while explicitly
deferring Homebrew. The plan must also capture the practical dependency on
issue `#7`: because that fix is landing separately, the executing agent should
fetch the latest remote state before `harness execute start` and treat `#7` as
an upstream baseline update rather than part of this release slice.

#### Expected Files

- `docs/plans/active/2026-03-23-alpha-binary-release-readiness.md`

#### Validation

- The tracked plan records the accepted release shape, non-goals, and the
  dependency on pulling the latest `main` before execution starts.

#### Execution Notes

Discovery completed before planning. The accepted direction is a first public
alpha released through GitHub Releases with prebuilt archives plus checksums,
with Homebrew explicitly deferred. This step also records the execution
constraint that the implementing agent should fetch the latest remote `main`
before starting code changes because issue `#7` is landing separately.

#### Review Notes

NO_STEP_REVIEW_NEEDED: discovery and planning closeout recorded directly in the
tracked plan.

### Step 2: Add release versioning and deterministic packaging

- Done: [x]

#### Objective

Make release binaries identify themselves clearly and add one repo-owned build
path that packages the first alpha artifacts deterministically.

#### Details

Prefer a thin repo-owned release build script that GitHub Actions can call, so
local packaging and remote packaging stay aligned. Extend the existing version
plumbing to surface a release identifier in `harness --version` while keeping
commit and mode diagnostics useful for dogfooding. Artifact naming, archive
layout, and checksum generation should be deterministic and documented so the
first public release does not depend on ad hoc shell history.

#### Expected Files

- `internal/version/info.go`
- `internal/version/*_test.go`
- `tests/smoke/smoke_test.go`
- `scripts/build-release`
- optional CLI/help files only if the version contract or root help wording
  needs adjustment

#### Validation

- Add or update targeted tests for release-version reporting without regressing
  current dev/release behavior.
- Run the release build script locally to produce the supported alpha
  artifacts and checksums in a deterministic output directory.
- Execute the current-platform built binary from the release output and verify
  `harness --version` reports the expected release identity.

#### Execution Notes

Added release-facing version metadata so release binaries report a public
release identifier alongside commit and mode, then introduced a
repo-owned `scripts/build-release` path that cross-compiles the supported
alpha targets, stages the binary plus docs, and emits deterministic zip
archives and `SHA256SUMS`. Step-closeout review tightened the smoke contract:
the release smoke now runs two identical builds, compares archive bytes and
checksum manifests across runs, unpacks every supported alpha archive, and
validates the packaged binary metadata for each target while still executing
the host-platform binary for the end-to-end `--version` check. Validation
passed with `go test ./internal/version ./internal/cli ./tests/smoke -count=1`,
`go test ./tests/smoke -count=1`, `bash -n scripts/build-release`,
`scripts/build-release --version v0.1.0-alpha.1 --output-dir .local/release-smoke-manual --platform $(go env GOOS)/$(go env GOARCH)`,
and `go test ./... -count=1`.

#### Review Notes

`review-001-delta` and `review-002-delta` both requested changes in the
`tests` slot because the initial smoke only exercised the host archive and did
not prove deterministic packaging across repeated builds. Expanded
`tests/smoke/release_build_test.go` to inspect every supported alpha archive,
verify archive checksums against actual bytes, and compare outputs from two
identical release builds. Follow-up `review-003-delta` passed with no
remaining findings, then `review-004-delta` reran the `tests` slot with the
durable `step_closeout` trigger so `harness status` could clear the completed
step review reminder cleanly.

### Step 3: Add public-release docs and repository automation

- Done: [x]

#### Objective

Make the repository ready to publish and maintain the first alpha binary
release with explicit docs, CI, and public-project guardrails.

#### Details

Add GitHub Actions for normal test coverage and for the release path defined in
Step 2. Document the supported alpha platforms, contributor Go baseline, and
install/verification steps for the public release. Include a `LICENSE` and any
minimal release-maintainer notes needed to cut the first alpha without hidden
knowledge. Keep the first slice focused on GitHub Releases; do not expand it
into Homebrew or a broader packaging matrix.

#### Expected Files

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `README.md`
- `LICENSE`
- optional release-maintainer doc such as `docs/releasing.md`

#### Validation

- `go test ./...` passes with the new automation and docs in place.
- The release workflow uses the same repo-owned build path as local packaging
  instead of reimplementing release logic inline.
- The README documents how an external user installs, verifies, and runs the
  released binary on the supported alpha platforms.

#### Execution Notes

Added GitHub Actions workflows for ordinary CI and release publication,
documented the public alpha binary install and verification flow in the README,
captured the contributor Go baseline, added a minimal release-maintainer guide,
and included an MIT `LICENSE` for the public release.

#### Review Notes

`review-005-delta` passed cleanly in the `docs_consistency` and `risk_scan`
slots after validating the public alpha docs, release workflow, CI workflow,
and license additions against the tracked binary-first alpha contract.

## Validation Strategy

- Run `harness plan lint docs/plans/active/2026-03-23-alpha-binary-release-readiness.md`
  before execution starts and after any material plan edits.
- During implementation, keep targeted version/smoke coverage green while
  iterating on release metadata and packaging behavior.
- Before archive, run `go test ./...` and a current-platform release smoke
  check against the packaged binary output.
- Before `harness execute start`, fetch the latest remote `main` so the plan
  executes on top of the separately landing `#7` fix if it has merged.

## Risks

- Risk: Release automation drifts between local packaging and GitHub Actions.
  - Mitigation: Use one repo-owned release build path that both humans and the
    workflow invoke.
- Risk: `harness --version` becomes ambiguous once it needs to report both a
  release identifier and the build commit.
  - Mitigation: Keep the output concise but explicit about which field is the
    public release identifier versus the underlying commit or mode diagnostic.
- Risk: Alpha scope expands into Homebrew, Windows, or source-install polish
  and slows the first public release.
  - Mitigation: Keep those items explicitly deferred in scope, acceptance
    criteria, and workflow docs.
- Risk: The upstream `#7` fix lands while this plan is under review, making the
  execution baseline stale.
  - Mitigation: Fetch the latest remote state immediately before execution and
    adjust the branch baseline before code changes begin.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-23-alpha-binary-release-readiness.md`
  passed before execution and again during archive closeout.
- `bash -n scripts/build-release` passed after the deterministic packaging and
  output-directory safety changes.
- `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/release.yml")'`
  validated the release workflow after the prerelease and tag-validation
  updates.
- `scripts/build-release --version v0.1.0-alpha.1 --output-dir .local/release-smoke-manual --platform $(go env GOOS)/$(go env GOARCH)`
  passed, and the packaged host binary reported the expected version, mode,
  and commit metadata.
- `go test ./internal/version ./internal/cli ./tests/smoke -count=1`,
  `go test ./tests/smoke -count=1`, repeated release-smoke runs with
  `go test ./tests/smoke -run 'TestBuildRelease' -count=5`, parallel
  release-smoke invocations in the shared worktree, and `go test ./... -count=1`
  all passed after the final reviewer-driven fixes.
- Revision-2 PR comment follow-up also passed `go test ./tests/smoke -run 'TestBuildRelease' -count=3`
  and `go test ./... -count=1` after tightening output-directory guardrails,
  adding spaced-output checksum coverage, and re-tracking the reopened active
  plan.

## Review Summary

- `review-001-delta` and `review-002-delta` tightened Step 2 coverage around
  deterministic packaging and archive inspection; `review-004-delta` then
  cleared the durable Step 2 closeout reminder with a clean `step_closeout`
  pass.
- `review-005-delta` passed cleanly for the Step 3 docs, CI, release
  workflow, and license slice.
- Finalize review was intentionally iterative: `review-006-full` and
  `review-007-delta` fixed prerelease publishing semantics, `review-008-full`
  fixed tag-validation, Linux checksum docs, and the untracked active plan,
  `review-009-full` fixed stale-output reuse, `review-010-full` exposed the
  reviewer-parallel release-smoke collision, and `review-011-full` tightened
  output-directory safety.
- `review-012-full` passed with zero blocking or non-blocking findings after
  the final repo-scoped output-directory guard and unique smoke-output fixes.
- After PR review feedback reopened the archived candidate in revision 2,
  `review-013-full` flagged that the active plan move had not yet been
  re-tracked in git, and `review-014-full` passed cleanly after restoring the
  tracked active-plan move and landing the `scripts/build-release` safety and
  spaced-path checksum fixes.

## Archive Summary

- Archived At: 2026-03-24T08:38:25+08:00
- Revision: 2
- PR: https://github.com/yzhang1918/superharness/pull/43
- Ready: The candidate now has clean step-closeout and finalize review,
  deterministic release packaging, prerelease-aware GitHub Release
  publication, existing-tag validation for manual dispatch, repo-scoped
  output-directory safety, destructive-directory rejection for repo-owned
  non-artifact paths, space-safe checksum generation, and stable release smoke
  coverage even when reviewer subagents run checks in parallel.
- Merge Handoff: Run `harness archive`, commit the archive move with the
  release-readiness changes, push the updated `codex/alpha-binary-release-readiness`
  branch to refresh PR #43, and then record publish, CI, and sync evidence
  for revision 2 through `harness evidence submit` until status reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added release-facing version metadata so `harness --version` reports a
  stable public release identifier alongside commit and mode diagnostics.
- Added a repo-owned `scripts/build-release` path plus smoke coverage for
  deterministic packaging, checksum emission, archive inspection, reused
  output-directory cleanup, and unsafe output-directory rejection.
- Tightened the release build path so output directories must stay under
  repo-owned `dist/` or `.local/` subtrees, reject parent-directory escapes or
  destructive repo paths, and generate `SHA256SUMS` without breaking when the
  repo or output path contains spaces.
- Added CI and release GitHub Actions workflows that reuse the repo-owned
  packaging path, validate release tags, and publish prerelease-tagged alpha
  releases correctly.
- Documented the binary-first public alpha contract in the README and release
  guide, including macOS/Linux checksum verification, supported targets, and
  contributor Go baseline expectations.
- Added an MIT `LICENSE` and kept the tracked active plan current through the
  multi-round finalize review and archive-readiness repairs.

### Not Delivered

- Homebrew distribution and tap maintenance remain deferred from the first
  public alpha.
- Windows release artifacts are still outside the first alpha packaging
  matrix.
- Optional `harness --version --json` support and richer provenance metadata
  remain deferred beyond the first release identifier.
- Broader source-install ergonomics and longer-term hardening work remain
  backlog items after this alpha slice.

### Follow-Up Issues

- `#42` tracks Homebrew distribution for public releases.
- `#41` tracks Windows release artifacts in the public release flow.
- `#32` tracks optional `harness --version --json` support and richer build
  metadata.
- `#8` and `#31` track the contributor/source-install follow-up around Go
  baseline and installer ergonomics.
- `#36` and `#37` track fuzz/property coverage and broader resilience
  hardening after the first alpha release path lands.
