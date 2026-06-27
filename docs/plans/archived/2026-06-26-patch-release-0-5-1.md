---
template_version: 0.2.0
created_at: "2026-06-26T17:08:21+08:00"
approved_at: "2026-06-26T17:09:17+08:00"
source_type: direct_request
source_refs:
    - 'patch release after #264/#261'
size: XS
---

# Patch Release 0.5.1

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare a patch release PR for `easyharness` version `0.5.1` so users can get
the Plan reader live-refresh scroll fix from #264/#261 without waiting for a
later minor release.

This is intentionally narrow: it is a release PR, not another product-change
PR. The work uses the standard harness workflow rather than
`workflow_profile: lightweight` because changing `VERSION` is release-safety
work and triggers release automation.

## Scope

### In Scope

- Update the root `VERSION` file from `0.5.0` to `0.5.1`.
- Track the minimal pnpm build approval needed for release validation and make
  installer fixtures copy it.
- Stabilize the release validation path where fresh installer smoke builds
  exposed transient Vite/esbuild service failures.
- Validate the release PR with the repository release validation profile.
- Keep the PR body focused on the patch release purpose and the shipped #261
  fix.
- Archive and publish the release candidate through harness evidence so it can
  wait for merge approval.

### Out of Scope

- Additional product, workflow, or release automation changes.
- Milestone creation or broader minor-release scope shaping.
- Publishing the GitHub release or Homebrew artifacts directly from this
  branch; those are expected to run after the release PR merges.

## Acceptance Criteria

- [x] `VERSION` contains `0.5.1`.
- [x] `scripts/validate-release` passes for the release PR.
- [x] The release PR explains that `0.5.1` ships the Plan reader refresh-scroll
      bug fix from #264/#261.
- [x] Any non-`VERSION` change is directly required for `scripts/validate-release`.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare Patch Release Version

- Done: [x]

#### Objective

Update the repository to represent the `0.5.1` patch release candidate and
validate it with the release profile.

#### Details

The current public release is `v0.5.0`, and the root `VERSION` file currently
contains `0.5.0`. The newly landed #264 fix resolved #261, a user-facing Plan
UI bug where live refresh could snap the reader back to the selected section.
Per `docs/releasing.md`, this is a small repair that fits a patch release.

Only bump `VERSION` to `0.5.1` unless validation proves a directly related
release-readiness repair is required. During execution, release validation
showed fresh installer fixtures need the pnpm build approval for `esbuild`, so
the plan also permits the minimal config/test-fixture update needed to make the
release profile deterministic.

#### Expected Files

- `VERSION`
- `web/pnpm-workspace.yaml`
- `tests/support/smoke.go`
- `tests/release/release_build_test.go`
- `scripts/build-embedded-ui`
- `scripts/validate-release`
- `tests/smoke/build_embedded_ui_test.go`
- `tests/smoke/ci_workflow_test.go`
- `scripts/install-dev-harness`
- `web/package.json`
- `web/vite.config.mjs`

#### Validation

- `scripts/validate-release` passes.
- `git diff` shows no unrelated file changes.

#### Execution Notes

Updated `VERSION` from `0.5.0` to `0.5.1`.

The first release validation attempt exposed a deterministic pnpm build-script
approval gap in fresh installer and release-build fixtures:
`scripts/install-dev-harness` and `scripts/build-release` run `CI=true pnpm
install --frozen-lockfile`, and fresh copied fixtures lacked the generated
`web/pnpm-workspace.yaml` approval for `esbuild`. Tracked that minimal
approval file and copied it in both installer and release-build fixture helpers
so the release profile can validate from fresh checkouts.

Validation run:

- `scripts/validate-release` initially failed on missing fresh-fixture pnpm
  build approval.
- `go test -tags installer_smoke ./tests/installer -count=1` passed after
  adding/copying the approval file.
- `go test -tags release_smoke ./tests/release -count=1` passed after adding
  the release-build fixture copy.
- Finalize review reruns exposed intermittent fresh-checkout Vite/esbuild
  `The service was stopped` / `write EPIPE` failures during installer smoke.
  Updated `scripts/build-embedded-ui` to retry that observed transient failure
  shape once, added smoke coverage for the retry, converted the Vite config
  from TypeScript to native-loadable ESM so config loading no longer depends on
  esbuild in fresh fixtures, and made the release profile run installer smoke
  with `-parallel=1 -timeout=20m` so multiple fresh install/build subprocesses
  do not compete during release validation while retaining enough timeout
  headroom for the serialized smoke package.
- `review-005-full` then found that even native Vite config loading can still
  hit repeated esbuild service failures during the Vite bundle step when every
  installer wrapper smoke case rebuilds the UI in a fresh fixture. The final
  repair makes installer smoke set an explicit test-only environment variable
  to reuse a minimal embedded UI fixture, keeping wrapper/install behavior
  coverage separate from the real UI build coverage already provided by
  `scripts/validate` and release smoke.
- `review-006-delta` passed with one minor coverage note, then a focused
  installer smoke assertion was added to prove the default installer path still
  runs `scripts/build-embedded-ui` when the test-only skip environment variable
  is absent.
- Final `scripts/validate-release` passed on 2026-06-27 after the
  validation-stability repair.

#### Review Notes

`review-001-full` passed on 2026-06-27 with `correctness`, `tests`, and
`risk-scan` reviewer slots. Reviewers reported no findings. The review covered
the `VERSION` bump, pnpm `esbuild` build approval, fresh installer/release
fixture copies, validation evidence, and release-safety scope.

`review-002-full` requested changes during finalize because reviewer reruns
contradicted the earlier validation evidence with installer-smoke
Vite/esbuild failures. The repair added a bounded transient esbuild retry to
the embedded UI build helper, serialized installer smoke in the release
validation profile, and reran focused plus full release validation before the
next finalize review.

`review-003-full` requested changes because serializing installer smoke without
raising Go's default package timeout could make `scripts/validate-release`
time out before release smoke ran. During the repair, another full release
validation attempt showed the esbuild service failures could repeat even after
one retry, so the final repair also switched Vite config loading to native ESM
with `web/vite.config.mjs`. The serialized installer smoke command now has
explicit `-timeout=20m` headroom, and the release profile was rerun to a clean
pass.

`review-005-full` requested one more change because the exact serialized
installer smoke command could still fail when every installer wrapper test ran
the real Vite bundle in a fresh checkout. The final repair changed installer
smoke to skip the embedded UI rebuild only under the explicit
`EASYHARNESS_TEST_SKIP_EMBEDDED_UI_BUILD=1` test environment and seeded a
minimal embedded UI fixture for Go's embed requirement. The real UI build
remains covered by `scripts/validate` and release smoke.

`review-006-delta` passed with one minor tests finding asking for coverage of
the default installer path when the skip env is absent. Added
`TestInstallDevHarnessRunsEmbeddedUIBuildByDefault` with a fake
`scripts/build-embedded-ui` helper so the default integration path is covered
without reintroducing repeated Vite rebuilds.

## Validation Strategy

- Use `scripts/validate-release` as the release-ready validation profile for
  the `VERSION` PR, as required by `docs/releasing.md`.
- Use harness review to verify the release bump is scoped, validated, and
  consistent with the patch release policy before archive.

## Risks

- Risk: A release PR can trigger tag and release automation after merge.
  - Mitigation: Keep the change limited to `VERSION`, run
    `scripts/validate-release`, and use standard harness review rather than
    lightweight workflow.
- Risk: The patch release may be too narrow or poorly justified.
  - Mitigation: Tie the release PR explicitly to #264/#261 and the patch
    policy in `docs/releasing.md`.

## Validation Summary

Final validation passed on 2026-06-27:

- `pnpm --dir web check`
- `pnpm --dir web build`
- `go test ./tests/smoke -run 'TestBuildEmbeddedUIScript|TestValidationScriptsDefineDevelopmentAndReleaseProfiles' -count=1`
- `go test -tags installer_smoke ./tests/installer -run 'TestInstallDevHarnessRunsEmbeddedUIBuildByDefault|TestInstallDevHarnessDefaultsToUserLocalBin' -count=1 -parallel=1 -timeout=20m`
- `go test -tags installer_smoke ./tests/installer -count=1 -parallel=1 -timeout=20m`
- `go test -tags release_smoke ./tests/release -count=1`
- `scripts/validate-release`
- `harness plan lint docs/plans/active/2026-06-26-patch-release-0-5-1.md`
- `git diff --check origin/main...HEAD`

The final full release profile includes the real embedded UI build through
`scripts/validate`, the serialized installer smoke profile, and release smoke.

## Review Summary

Review history:

- `review-001-full` passed the initial release bump and fresh-fixture pnpm
  approval repair.
- `review-002-full`, `review-003-full`, and `review-005-full` found release
  validation instability in fresh installer smoke. Each finding was repaired
  before continuing.
- `review-004-delta` passed the timeout/native Vite config repair.
- `review-006-delta` passed the installer-smoke UI rebuild separation repair,
  then a minor coverage note was addressed with
  `TestInstallDevHarnessRunsEmbeddedUIBuildByDefault`.
- `review-007-full` passed with `correctness`, `tests`, `risk-scan`, and
  `agent-ux` reviewer slots reporting no findings.

## Archive Summary

- Archived At: 2026-06-27T23:57:57+08:00
- Revision: 1
- PR: Open a patch release PR from `codex/patch-release-0.5.1` to `main` with a
body that explains `0.5.1` ships the Plan reader refresh-scroll bug fix from
#264/#261 and summarizes the release-validation repairs.

- Ready: Yes. The plan steps are complete, acceptance criteria are checked,
`scripts/validate-release` passes, and `review-007-full` passed with no
findings.

- Merge Handoff: After merge, the `VERSION` change is expected to trigger the
tag/release flow for `v0.5.1`; GitHub release/Homebrew publishing remain
post-merge release automation work, not work performed from this branch.

## Outcome Summary

### Delivered

- Prepared the `0.5.1` patch release candidate for the #264/#261 Plan reader
  refresh-scroll fix by updating `VERSION`.
- Added the pnpm `esbuild` build approval and copied it into fresh installer
  and release smoke fixtures.
- Stabilized release validation by using native-loadable Vite config,
  serializing installer smoke with explicit timeout headroom, and separating
  installer wrapper smoke from repeated real Vite rebuilds while preserving
  real UI build coverage in `scripts/validate`.
- Added focused coverage for the transient embedded UI build retry, release
  validation command shape, fresh fixture copy paths, and default
  `install-dev-harness` embedded UI build behavior.
- Left a clear PR handoff for the patch release purpose and post-merge release
  automation boundary.

### Not Delivered

- No additional product fixes beyond the already landed #264/#261 Plan reader
  fix.
- No manual GitHub release, tag publication, or Homebrew publishing from this
  branch.

### Follow-Up Issues

NONE
