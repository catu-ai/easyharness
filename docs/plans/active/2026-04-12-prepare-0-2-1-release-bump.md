---
template_version: 0.2.0
created_at: "2026-04-12T14:16:13+08:00"
approved_at: "2026-04-12T14:23:43+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Prepare the 0.2.1 release bump

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare the dedicated release change for `easyharness` version `0.2.1` so the
repository-facing release surface is internally consistent and the normal
VERSION-driven automation can publish the release after merge to `main`.

This slice should stay narrow and release-focused: update the tracked version
entry point, convert non-essential release examples away from the live current
version so future bumps do not require broad example churn, run focused
validation, and leave the result ready for the dedicated release PR path rather
than bypassing the documented workflow with a manual tag push.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.2.0` to `0.2.1`.
- Replace non-essential release-facing examples and help text that currently
  hard-code `0.2.0`/`v0.2.0` with stable pseudo values so future release bumps
  do not need to touch them.
- Update focused tests or fixtures where they currently couple generic example
  wording to the live release line instead of exercising actual current-version
  repository state.
- Run targeted validation for the changed release/version surfaces.
- Leave the repository state ready for a dedicated release branch or PR that
  can merge to `main` and trigger the existing automation chain.

### Out of Scope

- Changing the release workflow design, tag automation, or Homebrew publishing
  behavior.
- Replacing real semantic version fields such as `template_version` or other
  contract/schema markers with fake placeholder values.
- Editing archived plans or historical docs whose `0.2.0` references describe
  past work rather than the current release path.
- Writing a changelog, release notes, or broader release-policy guidance.
- Bypassing the release-PR convention with a direct manual publish path.

## Acceptance Criteria

- [ ] `VERSION` is `0.2.1`, while non-essential release docs/examples/help text
      no longer need to hard-code the current live release number just to
      explain the workflow.
- [ ] Focused validation for the changed release/version surfaces passes,
      including proving that `scripts/read-release-version --tag` resolves the
      new `v0.2.1` tag correctly.
- [ ] The resulting change is scoped as a dedicated release update that is
      ready to move through the repository's normal release PR and merge flow.

## Deferred Items

- Any follow-up changelog or announcement packaging for `0.2.1`.
- Any automation that bumps `VERSION` again after the release ships.

## Work Breakdown

### Step 1: Separate real release state from generic release examples

- Done: [ ]

#### Objective

Update the repository so the true current release state moves to `0.2.1`, but
generic release examples stop tracking the live release number.

#### Details

Keep this step disciplined and avoid broad search-and-replace. `VERSION`
should reflect the real release target `0.2.1`. By contrast, live
release-facing docs, workflow/help examples, and tests/fixtures that are only
illustrating command shape should switch to stable pseudo values such as
`0.0.0` / `v0.0.0` or another clearly generic placeholder. Historical
artifacts, archived plans, and real schema/template version markers should stay
unchanged.

#### Expected Files

- `VERSION`
- `README.md`
- `docs/releasing.md`
- focused release/version helpers or tests if their sample values are stale

#### Validation

- Re-read the updated release docs and README release section for internal
  consistency around the `VERSION -> merge -> auto tag -> Release workflow`
  path.
- Confirm the chosen pseudo example form is obviously generic and does not
  collide with places that must continue to express real current-version state.
- Update automated coverage only where a changed fixture or example previously
  tracked the current release line by accident.

#### Execution Notes

Updated the real release entry point in `VERSION` to `0.2.1`, then separated
generic release examples from the live release line by switching release docs,
workflow/help text, and generic fixture values from `0.2.0` / `v0.2.0` to the
stable pseudo examples `0.0.0` / `v0.0.0`. Left `template_version` and other
schema/template markers untouched because they carry real contract meaning
rather than release-example copy. TDD was not the right fit here because this
step changed documentation/example text and existing test fixtures rather than
introducing new runtime behavior; focused validation passed with
`scripts/read-release-version --tag`, `go test ./internal/cli -count=1`, and
`go test ./tests/smoke -run 'TestReleaseDocsPresentStableOnboardingSurface|TestBuildReleaseProducesStableArchiveAndVersionedBinary|TestBuildReleaseHelpUsesStableExampleVersion|TestReleaseWorkflowWiresHomebrewTapPublishing|TestInstallDevHarness|TestInstallDevHarnessPrefersStablePathBinary|TestInstallDevHarnessPrefersStablePathBinaryWhenRepoBinaryMissing|TestInstallDevHarnessRepairsManagedWrapper|TestInstallDevHarnessRepairsLegacyManagedWrapper|TestInstallDevHarnessLeavesForeignHarnessAlone' -count=1`.

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Validate the release bump and prepare merge-ready handoff

- Done: [ ]

#### Objective

Prove the `0.2.1` bump resolves through the existing tooling and leave the
result packaged as the normal dedicated release change for merge.

#### Details

Use focused validation rather than a broad unrelated sweep. At minimum,
confirm the version helper maps `VERSION=0.2.1` to `v0.2.1`, run targeted
tests for any changed release/version fixtures, and confirm the final diff
stays narrowly scoped to the real release bump plus the one-time future-proof
example cleanup. If local GitHub publication is available, this step may
include the dedicated release branch/PR handoff; otherwise, leave a precise
note about the remaining publish action.

#### Expected Files

- changed files from Step 1
- optional git metadata only if the branch/PR handoff is completed

#### Validation

- Run `scripts/read-release-version --tag`.
- Run focused tests for changed release/version surfaces.
- Review the final diff to confirm it contains only the dedicated release bump
  and the deliberate example de-coupling needed to avoid recurring churn.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Lint the tracked plan before approval.
- After approval, use focused release/version validation instead of unrelated
  repository-wide changes.
- Re-check the final diff cold to make sure the bump remains a dedicated
  release slice.

## Risks

- Risk: A broad version search-and-replace could accidentally rewrite
  historical references or template/schema version markers that should remain
  tied to earlier work.
  - Mitigation: only touch live release-facing surfaces and leave archived or
    schema-version references alone.
- Risk: Placeholder examples might become too abstract or ambiguous for release
  maintainers if they stop resembling real semver/tag input.
  - Mitigation: use semver-shaped pseudo values like `0.0.0` / `v0.0.0` and
    keep the real `VERSION` file as the concrete source of truth.
- Risk: The version bump may look complete in docs while drifting from the
  actual helper/tooling path that resolves tags from `VERSION`.
  - Mitigation: validate with `scripts/read-release-version --tag` and any
    focused tests tied to the changed surfaces.

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
