---
template_version: 0.2.0
created_at: "2026-05-01T22:02:37+08:00"
approved_at: "2026-05-01T22:04:01+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Prepare the 0.3.0 release bump

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare the dedicated release PR for `easyharness` version `0.3.0` by updating
the repository's tracked release entry point and leaving the rest of the
release publication path to the existing CI/CD automation.

This slice is intentionally narrow. The current release source of truth is the
root `VERSION` file, which is `0.2.6` before this work. After the release PR
merges to `main`, the documented VERSION-driven automation is expected to
create the matching `v0.3.0` tag, dispatch the Release workflow, publish
GitHub Release assets, and update the Homebrew formula when configured.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.2.6` to `0.3.0`.
- Confirm `scripts/read-release-version --tag` resolves the updated file to
  `v0.3.0`.
- Keep the resulting diff scoped to the dedicated release PR path.

### Out of Scope

- Changing release workflow, tag automation, build packaging, or Homebrew
  publishing behavior.
- Writing release notes, changelog entries, announcements, or broader release
  policy guidance.
- Creating or pushing a release tag manually.
- Running post-merge release verification; CI/CD owns publication after the
  release PR merges.
- Editing archived plans or historical release references.

## Acceptance Criteria

- [x] `VERSION` contains exactly `0.3.0`.
- [x] `scripts/read-release-version` returns `0.3.0`, and
      `scripts/read-release-version --tag` returns `v0.3.0`.
- [x] The final code diff contains only the release bump and harness plan
      lifecycle updates needed to drive this work.
- [x] The candidate is ready for a dedicated release PR that can merge to
      `main` and let CI/CD perform tagging, release publication, and Homebrew
      update work.

## Deferred Items

- Post-merge verification of the published `v0.3.0` GitHub Release and
  Homebrew formula.
- Any release notes, changelog, or announcement packaging for `0.3.0`.
- Any automation that bumps `VERSION` again after the release ships.

## Work Breakdown

### Step 1: Bump the release source of truth

- Done: [x]

#### Objective

Update the root `VERSION` file to `0.3.0` and prove the existing helper maps
that value to the expected release tag.

#### Details

Keep this as a dedicated release PR with no opportunistic cleanup. The human
selected the release-PR-only path because CI/CD owns the downstream tag,
publication, and Homebrew work. Do not alter workflows, release docs, archived
plans, or generated assets unless the version bump unexpectedly exposes a
direct inconsistency in the live release entry point.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-05-01-prepare-0-3-0-release-bump.md`

#### Validation

- Run `scripts/read-release-version`.
- Run `scripts/read-release-version --tag`.
- Review `git diff --stat` and the final diff to confirm the release PR stays
  narrowly scoped.

#### Execution Notes

Updated `VERSION` from `0.2.6` to `0.3.0`. TDD was not practical for this
slice because it is a release-entry metadata bump rather than a behavior
change. Focused validation passed with `scripts/read-release-version`
returning `0.3.0`, `scripts/read-release-version --tag` returning `v0.3.0`,
and a final diff check showing the release surface change is limited to
`VERSION` plus harness plan lifecycle updates.

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
