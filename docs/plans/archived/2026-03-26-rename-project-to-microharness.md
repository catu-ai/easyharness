---
template_version: 0.2.0
created_at: "2026-03-26T09:53:35+08:00"
source_type: direct_request
source_refs:
    - 'issue #45'
---

# Rename the project and repository to microharness

## Goal

Rename the public project from `superharness` to `microharness` before more
distribution and namespace choices harden around the current branding. This
slice should align the GitHub repository name, the Go module path, release
asset naming, and the durable docs/spec text around `microharness` while
intentionally preserving `harness` as the CLI command.

The outcome should be a coherent public identity: the repository and module
path move to `github.com/yzhang1918/microharness`, release archives are named
for `microharness`, and the docs clearly explain that users still run the
`harness` binary after unpacking or installing the tool.

## Scope

### In Scope

- Rename the GitHub repository branding from `superharness` to
  `microharness`.
- Update the tracked docs, release docs, README, and other live public-facing
  references to use `microharness`.
- Move the Go module path and all in-repo imports from
  `github.com/yzhang1918/superharness` to
  `github.com/yzhang1918/microharness`.
- Update release packaging, workflow metadata, and smoke coverage so archives
  are published as `microharness_<version>_<goos>_<goarch>.zip` while the
  packaged executable remains `harness`.
- Publish and verify a follow-up prerelease from the renamed repository so the
  recommended public test artifact matches the new project name.

### Out of Scope

- Changing the CLI command from `harness` to `microharness`.
- Adding Homebrew distribution or changing the deferred Homebrew strategy.
- Moving the repository into a GitHub organization.
- Rewriting historical archived plans, historical releases, or existing tags to
  remove old-name references retroactively.
- Adding Windows artifacts, macOS notarization, or other new distribution
  channels unrelated to the rename itself.

## Acceptance Criteria

- [x] The GitHub repository, live tracked docs, and module path all align on
      `microharness`, with no stale `superharness` references left in current
      codepaths or live operator docs except where historical context is
      intentionally preserved.
- [x] `go.mod` and all in-repo imports use
      `github.com/yzhang1918/microharness`, and the repository still builds and
      tests successfully.
- [x] Release packaging, workflow docs, and smoke coverage publish
      `microharness_*` archives while preserving `harness` as the packaged CLI
      executable and documenting that distinction clearly.
- [x] A new prerelease from the renamed repository is published and verified as
      the recommended artifact for external testing after the rename.

## Deferred Items

- `#42` Homebrew distribution and any tap naming/install flow.
- `#44` Whether the repository should later move into a dedicated GitHub
  organization.
- Any future revisit of the CLI command name; this slice intentionally keeps
  `harness`.
- Cleanup of historical archived docs, old releases, or issue text that still
  mention `superharness` as past context.

## Work Breakdown

### Step 1: Lock the rename contract and live-doc boundaries

- Done: [x]

#### Objective

Turn the discovery decisions into a durable rename contract so the execution
work does not drift on scope or accidentally rename the CLI command.

#### Details

This step should make the live naming boundaries explicit: the project/repo and
module path become `microharness`, the binary stays `harness`, historical
references are left alone when they are part of archived context, and Homebrew
plus org migration remain deferred. The step should also identify the current
live references that must move together so a future agent does not have to
rediscover rename scope from chat history.

#### Expected Files

- `docs/plans/active/2026-03-26-rename-project-to-microharness.md`
- `README.md`
- `docs/releasing.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- The tracked plan and live docs describe the same rename contract.
- The live docs explicitly state that `microharness` is the project name while
  `harness` remains the executable name.
- Deferred follow-ups for Homebrew and org migration remain named clearly
  instead of being implied.

#### Execution Notes

Locked the live rename contract around `microharness` in the tracked plan and
updated the live repo docs/specs to match it: the project, repo, release
assets, and module path move to `microharness`, while the executable remains
`harness`. The README, AGENTS contract, release guide, and live spec text now
describe that boundary explicitly and leave Homebrew plus org migration
deferred.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this contract/doc step was implemented together with the
broader Step 2 rename pass, so a separate docs-only delta review would be
redundant.

### Step 2: Rename the codebase, module path, and packaging metadata

- Done: [x]

#### Objective

Update the repository-owned code, imports, and packaging outputs so the
 codebase is internally consistent with the new `microharness` identity while
preserving the `harness` binary interface.

#### Details

This step covers the mechanical rename work inside the repository: change
`go.mod`, update Go imports, refresh repo URLs, update release/package naming,
and keep tests aligned with the new expectations. Validation should target
live codepaths and docs rather than historical archived content, so grep-based
checks should allow clearly intentional old-name references under archived
plans or prior release discussion. If build or release smoke tests currently
assert `superharness_*` archive names, they should move to the new package name
without changing the executable inside the archive.

#### Expected Files

- `go.mod`
- `cmd/harness/main.go`
- `internal/**/*.go`
- `scripts/build-release`
- `tests/smoke/release_build_test.go`
- `README.md`
- `docs/releasing.md`
- `.github/workflows/*.yml`

#### Validation

- Update or add targeted checks that enforce the new module path and release
  asset naming.
- Run `go test ./... -count=1`.
- Run a repo-scoped search to confirm live references moved to `microharness`
  while only intentional historical references remain under archived/history
  content.
- Run `scripts/build-release --version <candidate> --output-dir .local/... --platform $(go env GOOS)/$(go env GOARCH)`
  and verify the produced archive name uses `microharness_...` while the
  unpacked executable is still `harness`.

#### Execution Notes

Renamed the live codebase and packaging metadata from `superharness` to
`microharness`: `go.mod` and in-repo imports now use
`github.com/yzhang1918/microharness`, release archives/package roots are named
`microharness_<version>_<goos>_<goarch>`, and the installer/tests/docs keep the
binary name as `harness`. Focused validation passed with `go test
./tests/smoke -run 'TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary|TestInstallDevHarness' -count=1`,
`scripts/build-release --version v0.1.0-alpha.3 --output-dir
.local/release-rename-check --platform $(go env GOOS)/$(go env GOARCH)`,
`unzip -l` on the generated host archive showing a `microharness_*` package
root with `harness` inside, and `go test ./... -count=1`. After
`review-001-delta` requested follow-up, the installer regained takeover support
for legacy symlink installs that still point at an old `superharness` checkout,
release smoke now runs `status` from the unpacked host archive instead of only
`--version`, and binary metadata assertions now check the renamed module and
main-package paths directly. The review-fix validation passed with `go test
./tests/smoke -run 'TestInstallDevHarness(ReplacesLegacySymlinkedBinaryWithoutForce|ReplacesLegacyManagedWrapperWithoutForce)|TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary' -count=1`
and a fresh `go test ./... -count=1`.

#### Review Notes

`review-001-delta` requested changes on three points: the installer stopped
treating old symlink-based `superharness` installs as managed during upgrade,
release smoke only exercised `--version` from the unpacked archive, and the
validation layer did not assert the renamed module path directly. Commit
`9a31320` fixed those gaps, `review-002-delta` passed cleanly as the review-fix
rerun, and `review-003-delta` then passed as the fresh `step_closeout` review
with no remaining findings, closing Step 2.

### Step 3: Rename the GitHub repository and publish the renamed prerelease

- Done: [x]

#### Objective

Finish the public rename by aligning the remote repository identity and the
recommended test release with the new project name.

#### Details

Execution should coordinate the actual GitHub repository rename with the
already-updated tracked docs and module path so the public URLs settle around
`microharness` before Homebrew or org work begins. The release follow-up should
publish a new alpha from the renamed repository instead of rewriting existing
`v0.1.0-alpha.1`/`alpha.2` artifacts. The final validation should prove that
the renamed release assets, checksums, and version output all match the new
repo identity while the binary name remains `harness`.

#### Expected Files

- GitHub repository metadata for `yzhang1918/microharness`
- GitHub prerelease/tag metadata for the next alpha after `v0.1.0-alpha.2`
- `README.md`
- `docs/releasing.md`

#### Validation

- The GitHub repository URL resolves under `yzhang1918/microharness`.
- The release workflow succeeds for the post-rename prerelease tag.
- Downloaded release assets use `microharness_*` naming, checksum verification
  passes, and the unpacked executable still reports the expected version via
  `./harness --version`.
- The README and release guide point external testers at the renamed release
  path rather than the old project name.

#### Execution Notes

Renamed the GitHub repository from `yzhang1918/superharness` to
`yzhang1918/microharness`, updated the local `origin` remote to the renamed
repo URL, pushed `codex/rename-to-microharness`, and opened PR #48:
`https://github.com/yzhang1918/microharness/pull/48`. Published
`v0.1.0-alpha.3` from commit `cd6a8efdacc7feeb5c9f94ad4c32735145946454`; the
Release workflow succeeded at run `23574207229`, and PR #48 now shows passing
`Build and Publish Release Assets` plus both `Go Test` checks. External
verification for the renamed release is recorded under
`.local/harness/plans/2026-03-26-rename-project-to-microharness/release-verification/v0.1.0-alpha.3-cd6a8ef/`:
the downloaded host archive is named
`microharness_v0.1.0-alpha.3_darwin_arm64.zip`, `SHA256SUMS` matches the local
SHA256, `zipinfo -l` shows a `microharness_v0.1.0-alpha.3_darwin_arm64/`
package root with `harness` inside, and the unpacked binary reports
`version: v0.1.0-alpha.3`, `mode: release`, and
`commit: cd6a8efdacc7feeb5c9f94ad4c32735145946454`.

#### Review Notes

`review-004-delta` passed cleanly with no findings after verifying the renamed
GitHub repo identity, PR/release URLs, downloaded `alpha.3` proof artifacts,
and the preserved `harness` executable name inside the `microharness_*`
release archives.

## Validation Strategy

- Run `harness plan lint` before execution starts and again after any material
  plan edits.
- Validate the rename in layers: docs/contract first, then module/import and
  package naming, then the actual repository/release surfaces.
- Use targeted search checks to distinguish intentional historical
  `superharness` references from accidental live drift.
- Re-verify the published GitHub Release after the repo rename so external
  users see the new identity end-to-end, not just in local builds.

## Risks

- Risk: Module-path rename and repo rename could drift, leaving code, docs, and
  public URLs out of sync.
  - Mitigation: Make the repo URL and module path explicit acceptance criteria
    and validate both before archive.
- Risk: Renaming the project could accidentally rename the CLI command too,
  creating unnecessary compatibility churn.
  - Mitigation: Keep the `harness` executable boundary explicit in scope,
    acceptance criteria, docs, and release smoke checks.
- Risk: Historical docs, archived plans, or old releases may still contain
  `superharness`, making search-based validation noisy.
  - Mitigation: Use targeted live-surface searches and document that historical
    references are intentionally preserved.
- Risk: Publishing from the renamed repository could still leave external users
  following the old `alpha.2` artifact names or URLs.
  - Mitigation: Cut a new prerelease from the renamed repo and make it the
    clearly documented recommended test target.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-26-rename-project-to-microharness.md`
  passed when the plan was created, and the tracked plan remained lint-valid as
  step closeout notes were updated through finalize.
- The rename implementation passed focused validation with `go test
  ./tests/smoke -run 'TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary|TestInstallDevHarness' -count=1`,
  a host-platform `scripts/build-release --version v0.1.0-alpha.3 --output-dir
  .local/release-rename-check --platform $(go env GOOS)/$(go env GOARCH)`, and
  `unzip -l` confirming `microharness_*` archive naming with `harness` inside.
- After `review-001-delta` requested follow-up, the repair batch passed `go
  test ./tests/smoke -run 'TestInstallDevHarness(ReplacesLegacySymlinkedBinaryWithoutForce|ReplacesLegacyManagedWrapperWithoutForce)|TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary' -count=1`
  and `go test ./... -count=1`.
- The renamed prerelease shipped successfully via Release workflow run
  `23574207229`, and PR #48 checks show passing `Build and Publish Release
  Assets` plus both `Go Test` runs.
- External release proof for `v0.1.0-alpha.3` is stored under
  `.local/harness/plans/2026-03-26-rename-project-to-microharness/release-verification/v0.1.0-alpha.3-cd6a8ef/`,
  including `release-view.json`, `SHA256SUMS`, the downloaded
  `microharness_v0.1.0-alpha.3_darwin_arm64.zip`, local checksum verification,
  `zipinfo.txt`, and `version.txt` showing `version: v0.1.0-alpha.3`,
  `mode: release`, and `commit: cd6a8efdacc7feeb5c9f94ad4c32735145946454`.

## Review Summary

- `review-001-delta` requested changes on three points in Step 2: legacy
  symlink installs from old `superharness` checkouts were no longer treated as
  managed, release smoke only exercised `--version` from unpacked archives, and
  the validation layer did not assert the renamed module path directly.
- Commit `9a31320` fixed those gaps, `review-002-delta` passed cleanly as the
  review-fix rerun, and `review-003-delta` then passed as the fresh
  `step_closeout` review for Step 2.
- `review-004-delta` passed cleanly for Step 3 after checking the renamed
  GitHub repository identity, PR/release URLs, downloaded `alpha.3` proof
  artifacts, and the preserved `harness` executable name inside the
  `microharness_*` archives.
- Finalize review `review-005-full` then passed with zero blocking or
  non-blocking findings across the `correctness`, `tests`, `docs_consistency`,
  and `risk_scan` slots, clearing the full branch candidate for archive.

## Archive Summary

- Archived At: 2026-03-26T10:29:07+08:00
- Revision: 1
- PR: https://github.com/yzhang1918/microharness/pull/48
- Ready: The branch candidate now aligns the project, repository, module path,
  docs, installer, and release packaging on `microharness` while preserving
  `harness` as the executable, publishes and externally verifies
  `v0.1.0-alpha.3`, and carries clean Step 2/Step 3 delta review plus a clean
  `review-005-full` finalize pass.
- Merge Handoff: Archive the plan, commit the archive move and closeout
  summaries, push the latest branch tip to refresh PR #48, then record publish,
  CI, and sync evidence for the archived candidate until `harness status`
  returns `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Renamed the public project and repository identity from `superharness` to
  `microharness` across the live README, AGENTS contract, release guide, and
  live specs.
- Moved the Go module path and in-repo imports to
  `github.com/yzhang1918/microharness`.
- Updated release packaging, installer logic, and regression coverage so public
  archives are named `microharness_*`, the packaged executable stays `harness`,
  and legacy installer takeover still works for managed installs created before
  the rename.
- Renamed the GitHub repository to `yzhang1918/microharness`, opened PR #48 in
  the renamed repo, published `v0.1.0-alpha.3`, and recorded durable external
  proof for the renamed prerelease assets and binary identity.

### Not Delivered

- Homebrew distribution remains deferred to `#42`.
- Moving the project into a dedicated GitHub organization remains deferred to
  `#44`.
- The CLI command name remains `harness`; this slice intentionally did not
  rename it to `microharness`.
- Historical archived plans, old releases, and other legacy context still
  mentioning `superharness` were intentionally not rewritten.

### Follow-Up Issues

- `#42` tracks Homebrew distribution and tap naming/install work after the
  rename lands.
- `#44` tracks whether the renamed project should later move into a dedicated
  GitHub organization.
- CLI command renaming remains intentionally deferred with no active follow-up
  issue because the accepted direction for this slice is to keep `harness`.
- Historical old-name references remain intentionally preserved in archived
  plans and prior releases; no cleanup slice is planned right now.
