---
template_version: 0.2.0
created_at: "2026-06-09T22:57:53+08:00"
approved_at: "2026-06-09T23:00:14+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/pull/243
    - https://github.com/catu-ai/easyharness/issues/241
size: S
---

# Release 0.4.2 Patch

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare a narrow patch release PR for `easyharness` `0.4.2` after PR #243
landed the agent-facing repo config query command improvement and follow-up
path-root guidance.

The release PR should stay intentionally small: bump `VERSION`, run the
release-PR validation expected by `docs/releasing.md`, open the PR, and leave
actual tag/release publication to the existing VERSION-driven release
automation after the PR merges.

## Scope

### In Scope

- Bump root `VERSION` from `0.4.1` to `0.4.2`.
- Run the release-PR validation from `docs/releasing.md` that is feasible
  before merge:
  - `scripts/build-embedded-ui`
  - `go test ./...`
- Optionally run `scripts/build-release --version "v$(cat VERSION)"` as an
  extra packaging check if local prerequisites allow it.
- Open a patch release PR that explains the scope and links PR #243 / issue
  #241.
- Archive the plan and publish the PR handoff through harness evidence.

### Out of Scope

- Publishing the GitHub Release.
- Creating tags manually.
- Updating the Homebrew tap manually.
- Adding new release process machinery.
- Broad release notes or additional feature work beyond this patch bump.

## Acceptance Criteria

- [x] `VERSION` contains `0.4.2`.
- [x] `scripts/build-embedded-ui` passes.
- [x] `go test ./...` passes.
- [x] Any optional packaging check result is recorded in the plan.
- [x] A PR for the `0.4.2` patch release is opened against `main`.
- [x] The PR body states that the release is for the merged repo config query
      command / configured-root guidance improvement from PR #243 and issue
      #241.

## Deferred Items

- Release publication, tag creation, and Homebrew tap verification remain
  deferred to the existing release automation after the release PR merges.

## Work Breakdown

### Step 1: Prepare patch release PR

- Done: [x]

#### Objective

Bump `VERSION` to `0.4.2`, validate the release candidate, and open the patch
release PR.

#### Details

This is a standard harness workflow, not lightweight, because it touches
release safety. Keep the change narrow and avoid adding release-process
machinery.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-06-09-release-0-4-2-patch.md`

#### Validation

- `scripts/build-embedded-ui`
- `go test ./...`
- Optional: `scripts/build-release --version "v$(cat VERSION)"`

#### Execution Notes

Updated `VERSION` from `0.4.1` to `0.4.2`. Release-PR validation passed:
`scripts/build-embedded-ui`, `go test ./...`, and the optional
`scripts/build-release --version "v$(cat VERSION)"` packaging check. The
packaging check produced local ignored artifacts under `dist/release`:
`easyharness_v0.4.2_darwin_amd64.zip`,
`easyharness_v0.4.2_darwin_arm64.zip`,
`easyharness_v0.4.2_linux_amd64.zip`,
`easyharness_v0.4.2_linux_arm64.zip`, and `SHA256SUMS`. Opened PR #246:
https://github.com/catu-ai/easyharness/pull/246.

#### Review Notes

Step delta review `review-001-delta` passed with no blocking or non-blocking
findings. Reviewers checked release correctness and validation: the candidate
is scoped to `VERSION` plus the tracked plan, PR #246 accurately ties the patch
release to PR #243 / issue #241, and the release validation plus ignored
artifact names are recorded.

## Validation Strategy

- Follow the release checklist in `docs/releasing.md` for pre-merge release PR
  validation.
- Use focused review for the version bump, PR body, and recorded validation.

## Risks

- Risk: The patch release could imply more scope than PR #243 delivered.
  - Mitigation: Keep the PR body explicit that this is a patch release for the
    repo config query command and configured-root guidance improvement.
- Risk: Local release validation could miss an automation-only release failure.
  - Mitigation: Run the documented local checks and leave publication to the
    existing tag/release workflow after PR merge.
- Risk: Treating a release PR as lightweight could skip useful review.
  - Mitigation: Use the standard harness workflow despite the small file
    change because release safety is in scope.

## Validation Summary

- `scripts/build-embedded-ui`
- `go test ./...`
- `scripts/build-release --version "v$(cat VERSION)"`
- `harness plan lint docs/plans/active/2026-06-09-release-0-4-2-patch.md`
- `git diff --check`

## Review Summary

- Step delta review `review-001-delta` passed with no blocking or non-blocking
  findings. Reviewers checked release correctness and validation.
- Finalize full review `review-002-full` passed with no blocking or
  non-blocking findings. Reviewers confirmed the release scope, PR #246 body,
  validation record, ignored release artifacts, and post-merge automation
  handoff.

## Archive Summary

- Archived At: 2026-06-09T23:33:07+08:00
- Revision: 1
- PR: https://github.com/catu-ai/easyharness/pull/246
- Ready: Acceptance criteria are satisfied. `VERSION` is `0.4.2`, release
  validation and optional packaging check passed, PR #246 is open with the
  expected patch-release scope, and finalize review `review-002-full` passed.
- Merge Handoff: Push the archived plan move, refresh publish/CI/sync evidence
  for PR #246, wait for GitHub checks to pass and sync to remain fresh, then
  wait for explicit human merge approval. After the release PR merges, existing
  VERSION-driven automation should create tag `v0.4.2`, publish GitHub Release
  artifacts, and run the Homebrew tap update path when configured.

## Outcome Summary

### Delivered

- Bumped `VERSION` from `0.4.1` to `0.4.2`.
- Opened PR #246 for the patch release.
- Recorded release PR validation and packaging evidence in this plan and the
  PR body.

### Not Delivered

- Publishing the GitHub Release, creating tag `v0.4.2`, and Homebrew tap
  verification remain deferred to existing automation after the release PR
  merges.

### Follow-Up Issues

- No new GitHub issue was created. Release publication, tag creation, and
  Homebrew tap verification are expected follow-up actions of the existing
  VERSION-driven release automation after PR #246 merges, as documented in
  `docs/releasing.md` and the PR body.
