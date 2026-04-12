---
template_version: 0.2.0
created_at: "2026-04-12T11:18:00+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Prepare the v0.2.1 release candidate

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare the dedicated `0.2.1` release PR so the repository's documented
release flow can publish `v0.2.1` cleanly after merge. This slice should bump
the tracked release version, align the maintainer-facing release examples that
intentionally point at the current stable line, and leave the branch validated
and published for review.

The actual public tag creation, GitHub Release asset publication, and Homebrew
tap update remain automation-owned post-merge steps rather than work performed
directly in this slice.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.2.0` to `0.2.1`.
- Update release-facing docs, workflow help text, and smoke checks that treat
  the current stable release as an example value so they now point at
  `0.2.1` / `v0.2.1`.
- Run targeted release-path validation for the changed surfaces.
- Publish a dedicated release branch and PR that explains the merge-triggered
  automation flow for `v0.2.1`.

### Out of Scope

- Merging the PR to `main`.
- Manually creating or moving release tags.
- Manually running the GitHub Release workflow or pushing Homebrew formula
  updates outside the repository's normal merge automation.
- Broad historical cleanup of archived plans, prerelease fixtures, or other
  sample versions that are not meant to reflect the current stable release.

## Acceptance Criteria

- [ ] `VERSION` is `0.2.1`, and the main release-facing examples that point at
      the current stable line now reference `0.2.1` / `v0.2.1`.
- [ ] Targeted smoke validation passes for the updated release docs and
      release-build help surfaces.
- [ ] The dedicated `0.2.1` release branch is pushed and a PR is opened with
      the expected post-merge automation path called out.

## Deferred Items

- Merge approval and post-merge release verification.
- Any broader release-note writing or changelog policy work beyond the
  repository's current generated-notes release flow.

## Work Breakdown

### Step 1: Align the tracked release version and stable release examples

- Done: [x]

#### Objective

Update the repository-owned release version and the release-facing example
surfaces that intentionally describe the current stable line so they all agree
on `0.2.1`.

#### Details

Keep the slice narrow and release-oriented. Update only the root version file,
maintainer docs, workflow/help text, and smoke checks whose purpose is to show
or enforce the current stable release example. Do not churn prerelease
fixtures, parser tests, or other examples that exist to cover formatting
behavior rather than the live stable line.

#### Expected Files

- `VERSION`
- `README.md`
- `docs/releasing.md`
- `.github/workflows/release.yml`
- `scripts/build-release`
- `tests/smoke/release_docs_test.go`
- `tests/smoke/release_build_test.go`
- `tests/smoke/homebrew_formula_test.go`

#### Validation

- The changed files consistently reference `0.2.1` / `v0.2.1` where they are
  meant to track the current stable release example.
- The release PR remains a focused version-and-docs slice without unrelated
  fixture churn.

#### Execution Notes

Updated the root `VERSION` file to `0.2.1` and aligned every
release-facing current-stable example that intentionally tracks the live stable
line: the README release-PR paragraph, `docs/releasing.md`, the workflow
dispatch input description, the `scripts/build-release --help` example, and
the matching smoke assertions. Focused validation passed with:
`go test ./tests/smoke -run 'TestReleaseDocsPresentStableOnboardingSurface|TestBuildReleaseProducesStableArchiveAndVersionedBinary|TestBuildReleaseHelpUsesStableExampleVersion|TestReleaseWorkflowWiresHomebrewTapPublishing'`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step only aligns the tracked stable release version
and maintainer-facing example text, with no runtime or contract-semantic
change, and the focused smoke suite already covers the affected behavior.

### Step 2: Validate and publish the release candidate branch

- Done: [ ]

#### Objective

Prove the updated release surfaces still pass targeted validation, then publish
the dedicated release candidate branch and PR.

#### Details

Use targeted smoke coverage rather than a broad speculative sweep unless a
changed surface fails and forces expansion. The PR body should make the
repository contract explicit: merging the release PR to `main` is what causes
the automatic tag creation and release workflow dispatch for `v0.2.1`.

#### Expected Files

- `docs/plans/active/2026-04-12-prepare-v0-2-1-release-candidate.md`

#### Validation

- `go test ./tests/smoke -run 'TestReleaseDocsPresentStableOnboardingSurface|TestBuildReleaseProducesStableArchiveAndVersionedBinary|TestBuildReleaseHelpUsesStableExampleVersion|TestReleaseWorkflowWiresHomebrewTapPublishing'`
- `git status --short` is clean except for the intended release candidate
  changes before commit.
- The branch is pushed and the resulting PR URL is recorded in the plan before
  archive/readiness handoff.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Use focused smoke tests for release docs, release-build packaging/help, and
  workflow wiring because those are the only behavior-bearing surfaces changed
  by this release slice.
- Inspect the final diff to ensure the branch remains a dedicated release PR
  rather than absorbing unrelated cleanup.

## Risks

- Risk: Missing one current-stable example leaves the release PR internally
  inconsistent and weakens confidence in the merge-triggered release path.
  - Mitigation: Search the repo for `0.2.0` / `v0.2.0`, update only the
    release-facing stable examples, and rerun the matching smoke coverage.
- Risk: The release PR could imply that the public release is already live
  before merge.
  - Mitigation: Keep the plan and PR wording explicit that tag creation and
    publication happen after merge through automation.

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
