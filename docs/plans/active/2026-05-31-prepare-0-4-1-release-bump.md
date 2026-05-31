---
template_version: 0.2.0
created_at: "2026-05-31T14:06:12+08:00"
approved_at: "2026-05-31T14:07:21+08:00"
source_type: direct_request
source_refs:
    - GitHub release v0.4.0
    - 'GitHub PR #220'
    - 'GitHub PR #223'
    - 'GitHub PR #224'
    - 'GitHub PR #225'
    - 'GitHub PR #226'
size: XS
---

# Prepare 0.4.1 release bump

## Goal

Prepare the dedicated release PR for `easyharness` version `0.4.1` by updating
the repository's tracked release entry point and leaving tag creation, GitHub
Release publication, and Homebrew publishing to the existing VERSION-driven
CI/CD automation after merge.

The current release source of truth on `main` is the root `VERSION` file,
which is `0.4.0` before this work. The latest published release is `v0.4.0`.
Since that release, `main` contains small release-policy, reviewer-prompt,
discovery-prompt, status-evidence handoff, and release-triage follow-up work in
PRs #220, #223, #224, #225, and #226. Release triage classifies this as a
patch candidate because the changes are low-risk fast-follows and policy/docs
repairs worth shipping before the next selected minor milestone.

After the release PR merges to `main`, automation is expected to create the
matching `v0.4.1` tag, dispatch the `Release` workflow, publish GitHub Release
assets, and update the Homebrew formula when configured.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.4.0` to `0.4.1`.
- Confirm `scripts/read-release-version` returns `0.4.1`.
- Confirm `scripts/read-release-version --tag` returns `v0.4.1`.
- Keep the resulting diff scoped to the dedicated release PR path.
- Publish a dedicated release PR whose body records the automation handoff.

### Out of Scope

- Changing release workflow, tag automation, build packaging, or Homebrew
  publishing behavior.
- Writing release notes, changelog entries, announcements, or broader release
  policy guidance.
- Creating or pushing a release tag manually.
- Running post-merge release verification before this release PR lands; CI/CD
  owns publication after the release PR merges.
- Closing, creating, or retitling GitHub milestones.
- Adding more feature, UI, backlog, or v0.5.0 work before this release.
- Editing archived plans or historical release references.

## Acceptance Criteria

- [x] `VERSION` contains exactly `0.4.1`.
- [x] `scripts/read-release-version` returns `0.4.1`, and
      `scripts/read-release-version --tag` returns `v0.4.1`.
- [x] The final source diff contains only the release bump plus tracked harness
      lifecycle updates needed to drive this work.
- [x] The candidate is ready for a dedicated release PR that can merge to
      `main` and let CI/CD perform tagging, release publication, and Homebrew
      update work.
- [x] The PR handoff says not to create the tag manually and to let automation
      publish `v0.4.1`.

## Deferred Items

- Post-merge verification of the published `v0.4.1` GitHub Release and
  Homebrew formula.
- Any release notes, changelog, or announcement packaging for `0.4.1`.
- Splitting or implementing the `v0.5.0` `.harness` customization work tracked
  by #71.
- Reconsidering deferred UI or workbench issues such as #206 and #155.

## Work Breakdown

### Step 1: Bump the release source of truth

- Done: [x]

#### Objective

Update the root `VERSION` file to `0.4.1` and prove the existing helper maps
that value to the expected release tag.

#### Details

Keep this as a dedicated release PR with no opportunistic cleanup. This plan
uses the standard workflow despite the small file change because the `VERSION`
bump touches release-safety behavior.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-05-31-prepare-0-4-1-release-bump.md`

#### Validation

- `scripts/read-release-version`
- `scripts/read-release-version --tag`
- Review `git diff --stat` and the final diff to confirm the release PR stays
  narrowly scoped.

#### Execution Notes

Updated `VERSION` from `0.4.0` to `0.4.1`. TDD was not practical for this
slice because it is a release-entry metadata bump rather than a behavior
change. Focused validation passed with `scripts/read-release-version`
returning `0.4.1`, `scripts/read-release-version --tag` returning `v0.4.1`,
and a diff check showing the source release change is limited to `VERSION`
plus tracked harness plan lifecycle updates.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The implementation slice is a one-line `VERSION` bump
validated through the existing release-version helper; the whole candidate will
still receive finalize review before archive.

### Step 2: Confirm release handoff bookkeeping

- Done: [x]

#### Objective

Confirm the release bump candidate has no additional pre-release issue work and
that post-merge release verification belongs to the normal finalize/publish
handoff rather than pre-archive implementation.

#### Details

Do not create tags manually. The release PR should merge to `main` and let the
existing automation create `v0.4.1`, publish release assets, and update the
Homebrew formula when configured. Use `harness evidence refresh` and
`harness status` for ordinary post-archive remote handoff evidence. Direct
`gh` inspection is acceptable as diagnostic confirmation when the local harness
surface cannot answer a release-publication question.

Patch releases do not require a milestone under `docs/releasing.md`. The
current open milestone is `v0.5.0`; it remains out of scope for this patch.

#### Expected Files

- `docs/plans/active/2026-05-31-prepare-0-4-1-release-bump.md`

#### Validation

- Confirm recent `main` CI runs are green before release handoff.
- Confirm no open GitHub issue should be added to this patch release before
  the release bump.

#### Execution Notes

Confirmed recent `main` CI runs for PRs #220, #223, #224, #225, and #226 are
green. Reviewed the open issue list and found no release blocker that needs to
join this patch: #71 remains selected for the later `v0.5.0` milestone, and
the remaining accepted or deferred backlog stays outside this patch release.
Patch releases do not require a milestone under `docs/releasing.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only records release-handoff scope discipline
and remote issue/CI facts; the source change remains the one-line `VERSION`
bump, and finalize review will cover the candidate.

## Validation Strategy

- Lint this tracked plan before approval.
- During execution, use the existing release-version helper as the focused
  validation boundary instead of broader local release builds.
- Before archive, confirm the final diff contains only the planned release bump
  and harness lifecycle updates.
- After archive and PR publication, rely on the existing release automation for
  tag creation, asset publication, and Homebrew update.

## Risks

- Risk: The release PR could grow beyond the requested version bump.
  - Mitigation: keep release notes, workflow edits, post-merge verification,
    v0.5.0 work, and unrelated cleanup out of scope.
- Risk: The root version and derived tag could drift.
  - Mitigation: validate both raw and tag helper output after editing
    `VERSION`.
- Risk: Treating this as a tiny edit could skip the repository's release-safety
  workflow boundary.
  - Mitigation: use the standard tracked-plan flow even though the file change
    itself is small.
- Risk: Shipping without a patch milestone could look underspecified.
  - Mitigation: record that routine patch releases do not need milestones under
    `docs/releasing.md`, and keep the release PR body explicit about the
    automation handoff.

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
