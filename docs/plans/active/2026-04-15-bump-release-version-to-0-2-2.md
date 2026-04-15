---
template_version: 0.2.0
created_at: "2026-04-15T09:31:33+08:00"
approved_at: "2026-04-15T09:32:38+08:00"
source_type: direct_request
source_refs: []
size: XXS
workflow_profile: lightweight
---

# Bump Release Version To 0.2.2

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Prepare the dedicated release PR input for `v0.2.2` by updating the root
`VERSION` file from `0.2.1` to `0.2.2`.

This plan is intentionally narrow. The motivating issue is that the current
public Homebrew release still lags behind the repository's newer release
contract, but this slice only bumps the release seed version and keeps the
installer smoke expectation aligned with that bumped version. It does not fix
the dev-only wrapper or change release automation behavior.

## Scope

### In Scope

- Update the root `VERSION` file to `0.2.2`.
- Keep the installer smoke expectation aligned with the bumped release seed so
  the dev wrapper version assertion still reflects the current repo version.
- Keep the execution slice limited to the version bump plus required plan
  lifecycle bookkeeping.
- Record that the human explicitly approved `workflow_profile: lightweight`
  for this `XXS` release-preparation slice.

### Out of Scope

- Any wrapper or CLI behavior changes.
- Any release workflow, Homebrew tap, or GitHub Actions changes.
- Any tag creation, release publication, or manual Homebrew operations.
- Any README, spec, or release-note edits beyond what the future release PR may
  choose to add separately.

## Acceptance Criteria

- [x] `VERSION` contains `0.2.2`.
- [x] Execution changes stay limited to `VERSION`, the smoke expectation repair,
      and plan lifecycle updates.
- [x] The resulting branch is suitable to use as the dedicated release PR input
      that later merge automation can turn into `v0.2.2`.
- [x] No wrapper fix or release-pipeline changes are included in this slice.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Bump the tracked release seed to 0.2.2

- Done: [x]

#### Objective

Change the root `VERSION` file from `0.2.1` to `0.2.2`, and keep the matching
installer smoke expectation aligned with that release seed, so the next
dedicated release PR advertises the intended stable patch release cleanly.

#### Details

The human explicitly approved `workflow_profile: lightweight` for this `XXS`
slice even though release-adjacent work is usually kept on the standard path.
That exception is acceptable here because execution is confined to one tracked
version file plus one narrow smoke repair, does not modify runtime behavior,
and leaves the existing tag/release/Homebrew automation untouched. The wrapper
mismatch discussed in discovery is motivation only and remains out of scope for
this plan. Revision `2` reopened the archived candidate in `finalize-fix` after
post-archive CI exposed a stale hardcoded smoke expectation for the dev version
string.

#### Expected Files

- `VERSION`
- `tests/smoke/install_dev_harness_test.go`

#### Validation

- Confirm `VERSION` reads `0.2.2`.
- Run `scripts/read-release-version --tag` and confirm it reports `v0.2.2`.
- Inspect the diff to confirm execution stayed within the planned narrow scope.
- Run `go test ./tests/smoke -run TestInstallDevHarnessVersionReportsDevModeAndPathInsideWorktree -count=1`.
- Run `go test ./tests/smoke -count=1`.
- Run `go test ./...`.

#### Execution Notes

Updated `VERSION` from `0.2.1` to `0.2.2` and confirmed the release helper now
reports `v0.2.2` through `scripts/read-release-version --tag`. Diff inspection
confirmed the execution slice stayed within the planned narrow scope: the
release seed plus tracked plan lifecycle notes only.

Revision `2` reopened the archived candidate after GitHub Actions failed on a
stale hardcoded expectation of `v0.2.1-dev` in
`TestInstallDevHarnessVersionReportsDevModeAndPathInsideWorktree`. Repaired the
smoke to derive its expected dev version from the fixture repo's `VERSION`
file, then reran the targeted test, the full `tests/smoke` package, and
`go test ./...` successfully.

When execution finishes, leave the required lightweight breadcrumb in the PR
body or another approved review surface before treating the candidate as ready
to wait for merge approval.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this slice is a single tracked version-field bump with
no code, contract, or automation logic changes.

## Validation Strategy

- Use direct file inspection plus `scripts/read-release-version --tag`.
- Run the installer smoke validations that cover the dev wrapper version output.
- Review the final diff to ensure no accidental scope expansion occurred.

## Risks

- Risk: A broader release-readiness concern could get conflated with this tiny
  version bump.
  - Mitigation: Keep the plan explicit that this slice only updates `VERSION`
    plus one directly related smoke expectation, and leaves all release
    mechanics and wrapper behavior unchanged.
- Risk: Merging the release PR before maintainers are ready would publish
  `v0.2.2` too early through the existing automation.
  - Mitigation: Human merge approval still gates release timing after this
    lightweight slice is prepared.

## Validation Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Confirmed `VERSION` now reads `0.2.2`.
- Ran `scripts/read-release-version --tag` and confirmed it reports `v0.2.2`.
- Inspected the diff and confirmed the execution slice stayed limited to the
  planned version bump and tracked plan lifecycle notes.

## Review Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Finalize review `review-001-full` passed on 2026-04-15 with zero blocking and
  zero non-blocking findings.
- Reviewer coverage confirmed the candidate stayed within the approved `XXS`
  lightweight scope and remained ready for merge handoff.

## Archive Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Archived At: 2026-04-15T09:36:02+08:00
- Revision: 1
- PR: NONE. The candidate has not been pushed or opened as a PR yet.
- Ready: Acceptance criteria are satisfied, Step 1 is complete, and finalize
  review `review-001-full` passed with no findings for revision `1`.
- Merge Handoff: Archive the lightweight plan snapshot, commit the tracked
  deletion from `docs/plans/active/`, push `codex/bump-release-version-to-0-2-2`,
  open the release PR, leave the lightweight breadcrumb in the PR body, and
  record publish, CI, and sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Bumped the tracked release seed from `0.2.1` to `0.2.2`.
- Recorded the approved lightweight execution history and clean finalize review
  for this release-preparation slice.

### Not Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- No wrapper, runtime, or release-workflow changes were made.
- No tag, GitHub Release, or Homebrew publication was triggered by this slice.

### Follow-Up Issues

UPDATE_REQUIRED_AFTER_REOPEN

NONE
