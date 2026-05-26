---
template_version: 0.2.0
created_at: "2026-05-26T22:14:39+08:00"
approved_at: "2026-05-26T22:16:05+08:00"
source_type: direct_request
source_refs:
    - GitHub milestone v0.4.0
    - GitHub release v0.3.1
    - 'GitHub PR #218'
size: XS
---

# Prepare 0.4.0 release bump

## Goal

Prepare the dedicated release PR for `easyharness` version `0.4.0` by updating
the repository's tracked release entry point and leaving the rest of the release
publication path to the existing VERSION-driven CI/CD automation.

The current release source of truth on `main` is the root `VERSION` file, which
is `0.3.1` before this work. The `v0.4.0` milestone is complete with 0 open
issues and 6 closed issues: #203, #199, #213, #200, #202, and #12. PR #218
merged after the `0.3.1` release and is accepted as part of the current
`main`-based `0.4.0` release candidate because it is adjacent controller
handoff guidance rather than a reason to cut a special historical base.

After the release PR merges to `main`, automation is expected to create the
matching `v0.4.0` tag, dispatch the `Release` workflow, publish GitHub Release
assets, and update the Homebrew formula when configured.

## Scope

### In Scope

- Bump the root `VERSION` file from `0.3.1` to `0.4.0`.
- Confirm `scripts/read-release-version` returns `0.4.0`.
- Confirm `scripts/read-release-version --tag` returns `v0.4.0`.
- Keep the resulting diff scoped to the dedicated release PR path.

### Out of Scope

- Changing release workflow, tag automation, build packaging, or Homebrew
  publishing behavior.
- Writing release notes, changelog entries, announcements, or broader release
  policy guidance.
- Creating or pushing a release tag manually.
- Running post-merge release verification before this release PR lands; CI/CD
  owns publication after the release PR merges.
- Closing the `v0.4.0` milestone before the release PR merges and the release
  publication path has completed.
- Adding more feature, UI, backlog, or v0.5.0 work before this release.
- Editing archived plans or historical release references.

## Acceptance Criteria

- [x] `VERSION` contains exactly `0.4.0`.
- [x] `scripts/read-release-version` returns `0.4.0`, and
      `scripts/read-release-version --tag` returns `v0.4.0`.
- [x] The final code diff contains only the release bump and harness plan
      lifecycle updates needed to drive this work.
- [x] The candidate is ready for a dedicated release PR that can merge to
      `main` and let CI/CD perform tagging, release publication, and Homebrew
      update work.
- [x] The `v0.4.0` milestone remains complete with 0 open issues, and milestone
      closure is deferred until after release publication.

## Deferred Items

- Post-merge verification of the published `v0.4.0` GitHub Release and
  Homebrew formula.
- Closing the already-complete `v0.4.0` milestone after the release
  publication path completes.
- Any release notes, changelog, or announcement packaging for `0.4.0`.
- Splitting or implementing the `v0.5.0` `.harness` customization work tracked
  by #71.
- Reconsidering deferred UI or workbench issues such as #206 and #155.

## Work Breakdown

### Step 1: Bump the release source of truth

- Done: [x]

#### Objective

Update the root `VERSION` file to `0.4.0` and prove the existing helper maps
that value to the expected release tag.

#### Details

Keep this as a dedicated release PR with no opportunistic cleanup. The human
selected the current `main` release base after confirming there are no new
issues or release blockers that should land before `0.4.0`. PR #218 is allowed
to ride along because it is already on `main` and is adjacent handoff guidance.

This plan intentionally uses the standard workflow despite the small file
change because the `VERSION` bump touches release-safety behavior.

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-05-26-prepare-0-4-0-release-bump.md`

#### Validation

- `scripts/read-release-version`
- `scripts/read-release-version --tag`
- Review `git diff --stat` and the final diff to confirm the release PR stays
  narrowly scoped.

#### Execution Notes

Updated `VERSION` from `0.3.1` to `0.4.0`. TDD was not practical for this
slice because it is a release-entry metadata bump rather than a behavior
change. Focused validation passed with `scripts/read-release-version`
returning `0.4.0`, `scripts/read-release-version --tag` returning `v0.4.0`,
and a diff check showing the source release change is limited to `VERSION`
plus harness plan lifecycle updates.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The implementation slice is a one-line `VERSION` bump
validated through the existing release-version helper; the whole candidate
will still receive finalize review before archive.

### Step 2: Confirm release handoff bookkeeping

- Done: [x]

#### Objective

Confirm the release bump candidate has no additional pre-release issue work and
that post-merge release and milestone closure work belongs to the normal
finalize/publish handoff rather than pre-archive implementation.

#### Details

Do not create tags manually. The release PR should merge to `main` and let the
existing automation create `v0.4.0`, publish release assets, and update the
Homebrew formula when configured. Use `harness evidence refresh` and
`harness status` for ordinary post-archive remote handoff evidence. Direct
`gh` inspection is acceptable as diagnostic confirmation when the local harness
surface cannot answer a release-publication question.

Closing the milestone is post-merge GitHub bookkeeping after the release path
is complete, so it must not block the release-bump candidate from reaching
`execution/finalize/await_merge`.

#### Expected Files

- `docs/plans/active/2026-05-26-prepare-0-4-0-release-bump.md`

#### Validation

- Confirm the `v0.4.0` milestone has 0 open issues before release handoff.
- Confirm no open GitHub issue should be added to the `v0.4.0` release before
  this release bump.

#### Execution Notes

Confirmed the `v0.4.0` milestone has 0 open issues and that the open backlog
does not contain a new release blocker. The actual PR publication, CI/sync
evidence, and merge-readiness handoff belong to finalize/publish after archive.
Milestone closure is deferred until the release publication path completes.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only records release-handoff scope discipline;
the source change remains the one-line `VERSION` bump, and finalize review will
cover the candidate.

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
- Risk: Closing the milestone before publication could make release state look
  more complete than it is.
  - Mitigation: defer milestone closure until the release publication path
    completes after the release PR merges.

## Validation Summary

- `scripts/read-release-version` returned `0.4.0`.
- `scripts/read-release-version --tag` returned `v0.4.0`.
- `harness plan lint docs/plans/active/2026-05-26-prepare-0-4-0-release-bump.md`
  passed before execution and again before archive.
- `gh issue list --state open --milestone v0.4.0 --json number,title,url`
  returned an empty list.
- Final diff review confirmed the source release change is limited to the
  `VERSION` bump plus tracked harness lifecycle updates.

## Review Summary

- Step review was skipped with `NO_STEP_REVIEW_NEEDED` because the
  implementation was a one-line `VERSION` bump plus release-handoff scope
  confirmation validated through existing helpers and issue queries.
- Finalize review `review-001-full` passed on 2026-05-26 with 0 blocking and 0
  non-blocking findings.
- Reviewer slots covered `correctness` and `release_scope`; both reported no
  findings.

## Archive Summary

- Archived At: 2026-05-26T22:20:38+08:00
- Revision: 1
- PR: pending archive publication. After archive, push branch
  `codex/release-0-4-0` and open the dedicated release PR for `0.4.0`.
- Ready: the candidate is ready for PR publication; `VERSION` is `0.4.0`, the
  helper resolves `v0.4.0`, the `v0.4.0` milestone has 0 open issues, and
  finalize review passed with no findings.
- Merge Handoff: after the PR exists and CI/CD readiness is recorded, stop at
  `execution/finalize/await_merge` for explicit human merge approval. The
  release publication path itself remains CI/CD-owned after merge to `main`;
  no manual tag or GitHub Release publication is performed in this slice.

## Outcome Summary

### Delivered

- Root `VERSION` bumped from `0.3.1` to `0.4.0`.
- Focused helper validation confirmed the raw version and matching release tag
  resolve to `0.4.0` and `v0.4.0`.
- Confirmed the `v0.4.0` milestone has no open issues and no additional issue
  should block the release bump.
- The tracked plan records the approved release-PR-only scope, validation, and
  finalize review outcome.

### Not Delivered

- No release workflow, tag automation, package build, release notes, changelog,
  announcement, Homebrew behavior, or feature code was changed.
- No manual `v0.4.0` tag or GitHub Release was created; CI/CD owns that after
  the release PR merges.
- The `v0.4.0` milestone was not closed before release publication.

### Follow-Up Issues

- No GitHub issues were created. Deferred items are operational release
  follow-ups rather than repository backlog: verify the published `v0.4.0`
  GitHub Release and Homebrew formula after CI/CD publishes them, close the
  already-complete `v0.4.0` milestone after publication, and handle any future
  changelog or announcement packaging outside this release-bump PR.
