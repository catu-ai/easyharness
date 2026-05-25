---
template_version: 0.2.0
created_at: "2026-05-25T15:41:00+08:00"
approved_at: "2026-05-25T16:36:16+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Prepare 0.3.1 release bump

## Goal

Prepare the dedicated release PR for `easyharness` version `0.3.1` by updating
the repository's tracked release entry point and leaving the rest of the release
publication path to the existing VERSION-driven CI/CD automation.

The current release source of truth is the root `VERSION` file, which is
`0.3.0` before this work. After the release PR merges to `main`, automation is
expected to create the matching `v0.3.1` tag, dispatch the `Release` workflow,
publish GitHub Release assets, and update the Homebrew formula when configured.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.3.0` to `0.3.1`.
- Confirm `scripts/read-release-version` returns `0.3.1`.
- Confirm `scripts/read-release-version --tag` returns `v0.3.1`.
- Keep the resulting diff scoped to the dedicated release PR path.

### Out of Scope

- Changing release workflow, tag automation, build packaging, or Homebrew
  publishing behavior.
- Writing release notes, changelog entries, announcements, or broader release
  policy guidance.
- Creating or pushing a release tag manually.
- Running post-merge release verification before this release PR lands; CI/CD
  owns publication after the release PR merges.
- Editing archived plans or historical release references.

## Acceptance Criteria

- [x] `VERSION` contains exactly `0.3.1`.
- [x] `scripts/read-release-version` returns `0.3.1`, and
      `scripts/read-release-version --tag` returns `v0.3.1`.
- [x] The final code diff contains only the release bump and harness plan
      lifecycle updates needed to drive this work.
- [x] The candidate is ready for a dedicated release PR that can merge to
      `main` and let CI/CD perform tagging, release publication, and Homebrew
      update work.

## Deferred Items

- Post-merge verification of the published `v0.3.1` GitHub Release and
  Homebrew formula.
- Any release notes, changelog, or announcement packaging for `0.3.1`.
- Any automation that bumps `VERSION` again after the release ships.

## Work Breakdown

### Step 1: Bump the release source of truth

- Done: [x]

#### Objective

Update the root `VERSION` file to `0.3.1` and prove the existing helper maps
that value to the expected release tag.

#### Details

Keep this as a dedicated release PR with no opportunistic cleanup. The human
selected a patch release for `0.3.1`; CI/CD owns the downstream tag,
publication, and Homebrew work. Do not alter workflows, release docs, archived
plans, or generated assets unless the version bump unexpectedly exposes a
direct inconsistency in the live release entry point.

This plan intentionally uses the standard workflow despite the small file
change because the `VERSION` bump touches release-safety behavior.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-05-25-prepare-0-3-1-release-bump.md`

#### Validation

- `scripts/read-release-version`
- `scripts/read-release-version --tag`
- Review `git diff --stat` and the final diff to confirm the release PR stays
  narrowly scoped.

#### Execution Notes

Updated `VERSION` from `0.3.0` to `0.3.1`. TDD was not practical for this
slice because it is a release-entry metadata bump rather than a behavior
change. Focused validation passed with `scripts/read-release-version`
returning `0.3.1`, `scripts/read-release-version --tag` returning `v0.3.1`,
and a diff check showing the source change is limited to `VERSION` plus
harness plan lifecycle updates.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The implementation slice is a one-line `VERSION` bump
validated through the existing release-version helper; the whole candidate
will still receive finalize review before archive.

## Validation Strategy

- Lint this tracked plan before approval.
- During execution, use the existing release-version helper as the focused
  validation boundary instead of broader local release builds.
- Before archive, confirm the final diff contains only the planned release bump
  and harness lifecycle updates.

## Risks

- Risk: The release PR could grow beyond the requested version bump.
  - Mitigation: keep release notes, workflow edits, post-merge verification,
    and unrelated cleanup out of scope.
- Risk: The root version and derived tag could drift.
  - Mitigation: validate both raw and tag helper output after editing
    `VERSION`.
- Risk: Treating this as a tiny edit could skip the repository's release-safety
  workflow boundary.
  - Mitigation: use the standard tracked-plan flow even though the file change
    itself is small.

## Validation Summary

- `scripts/read-release-version` returned `0.3.1`.
- `scripts/read-release-version --tag` returned `v0.3.1`.
- `harness plan lint docs/plans/active/2026-05-25-prepare-0-3-1-release-bump.md`
  passed before execution and again before archive.
- Final diff review confirmed the source release change is limited to the
  `VERSION` bump plus tracked harness lifecycle updates.

## Review Summary

- Step review was skipped with `NO_STEP_REVIEW_NEEDED` because the
  implementation was a one-line `VERSION` bump validated through the existing
  release-version helper.
- Finalize review `review-001-full` passed on 2026-05-25 with 0 blocking and 0
  non-blocking findings.
- Reviewer slots covered `correctness` and `release_scope`; both reported no
  findings.

## Archive Summary

- Archived At: 2026-05-25T16:39:02+08:00
- Revision: 1
- PR: pending archive publication. After archive, push branch
  `codex/release-0-3-1` and open the dedicated release PR for `0.3.1`.
- Ready: the candidate is ready for PR publication; `VERSION` is `0.3.1`, the
  helper resolves `v0.3.1`, and finalize review passed with no findings.
- Merge Handoff: after the PR exists and CI/CD readiness is recorded, stop at
  `execution/finalize/await_merge` for explicit human merge approval.
- The release publication path itself remains CI/CD-owned after merge to
  `main`; no manual tag or release publish work was performed in this slice.

## Outcome Summary

### Delivered

- Root `VERSION` bumped from `0.3.0` to `0.3.1`.
- Focused helper validation confirmed the raw version and matching release tag
  resolve to `0.3.1` and `v0.3.1`.
- The tracked plan records the approved release-PR-only scope, validation, and
  finalize review outcome.

### Not Delivered

- No release workflow, tag automation, package build, release notes, changelog,
  announcement, or Homebrew behavior was changed.
- No manual `v0.3.1` tag or GitHub Release was created; CI/CD owns that after
  the release PR merges.

### Follow-Up Issues

- No GitHub issues were created. Deferred items are operational release
  follow-ups rather than repository backlog: verify the published `v0.3.1`
  GitHub Release and Homebrew formula after CI/CD publishes them, and handle
  any future changelog or announcement packaging outside this release-bump PR.
