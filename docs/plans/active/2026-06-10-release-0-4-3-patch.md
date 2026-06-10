---
template_version: 0.2.0
created_at: "2026-06-10T23:01:58+08:00"
approved_at: "2026-06-10T23:05:46+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/pull/245
size: S
---

# Release 0.4.3 Patch

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare a narrow patch release PR for `easyharness` `0.4.3` after PR #245
landed the repo config canonical refresh improvement for issue #240.

The release PR should stay intentionally small: bump `VERSION`, run the
release-PR validation expected by `docs/releasing.md`, open the PR, and leave
actual tag/release publication to the existing VERSION-driven release
automation after the PR merges.

## Scope

### In Scope

- Bump root `VERSION` from `0.4.2` to `0.4.3`.
- Run the release-PR validation from `docs/releasing.md` that is feasible
  before merge:
  - `scripts/build-embedded-ui`
  - `go test ./...`
- Optionally run `scripts/build-release --version "v$(cat VERSION)"` as an
  extra packaging check if local prerequisites allow it.
- Open a patch release PR that explains the scope and links PR #245 / issue
  #240.
- Archive the plan and publish the PR handoff through harness evidence.

### Out of Scope

- Publishing the GitHub Release.
- Creating tags manually.
- Updating the Homebrew tap manually.
- Adding new release process machinery.
- Broad release notes or additional feature work beyond this patch bump.

## Acceptance Criteria

- [x] `VERSION` contains `0.4.3`.
- [x] `scripts/build-embedded-ui` passes.
- [x] `go test ./...` passes.
- [x] Any optional packaging check result is recorded in the plan.
- [ ] A PR for the `0.4.3` patch release is opened against `main`.
- [ ] The PR body states that the release is for the merged repo config
      canonical refresh improvement from PR #245 and issue #240.

## Deferred Items

- Release publication, tag creation, and Homebrew tap verification remain
  deferred to the existing release automation after the release PR merges.

## Work Breakdown

### Step 1: Prepare patch release PR

- Done: [ ]

#### Objective

Bump `VERSION` to `0.4.3`, validate the release candidate, and open the patch
release PR.

#### Details

This is a standard harness workflow, not lightweight, because it touches
release safety. Keep the change narrow and avoid adding release-process
machinery.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-06-10-release-0-4-3-patch.md`

#### Validation

- `scripts/build-embedded-ui`
- `go test ./...`
- Optional: `scripts/build-release --version "v$(cat VERSION)"`

#### Execution Notes

Updated `VERSION` from `0.4.2` to `0.4.3`. Release-PR validation passed:
`scripts/build-embedded-ui`, `go test ./...`, and the optional
`scripts/build-release --version "v$(cat VERSION)"` packaging check. The
packaging check produced local ignored artifacts under `dist/release`:
`easyharness_v0.4.3_darwin_amd64.zip`,
`easyharness_v0.4.3_darwin_arm64.zip`,
`easyharness_v0.4.3_linux_amd64.zip`,
`easyharness_v0.4.3_linux_arm64.zip`, and `SHA256SUMS`.

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Follow the release checklist in `docs/releasing.md` for pre-merge release PR
  validation.
- Use review for the version bump, PR body, and recorded validation.

## Risks

- Risk: The patch release could imply more scope than PR #245 delivered.
  - Mitigation: Keep the PR body explicit that this is a patch release for the
    repo config canonical refresh improvement.
- Risk: Local release validation could miss an automation-only release failure.
  - Mitigation: Run the documented local checks and leave publication to the
    existing tag/release workflow after PR merge.
- Risk: Treating a release PR as lightweight could skip useful review.
  - Mitigation: Use the standard harness workflow despite the small file
    change because release safety is in scope.

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
