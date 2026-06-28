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
- Reopen and slim the existing #265 candidate so the release PR only contains
  the `VERSION` bump plus tracked harness plan/archive material.
- Validate the release PR with the repository release validation profile.
- Keep the PR body focused on the patch release purpose and the shipped #261
  fix.
- Archive and publish the release candidate through harness evidence so it can
  wait for merge approval.

### Out of Scope

- Additional product, workflow, or release automation changes.
- Node version tooling such as `.nvmrc`, `.node-version`, Volta, mise, or
  documentation updates. Discovery found that this is useful follow-up, but the
  human chose the smallest release PR for this slice.
- Vite, pnpm, installer-smoke, or release-validation stability changes.
- Milestone creation or broader minor-release scope shaping.
- Publishing the GitHub release or Homebrew artifacts directly from this
  branch; those are expected to run after the release PR merges.

## Acceptance Criteria

- [x] `VERSION` contains `0.5.1`.
- [x] The branch diff against `origin/main` contains no Vite, pnpm,
      installer-smoke, or validation-stability code churn.
- [x] `scripts/validate-release` passes for the release PR under the CI-aligned
      local environment: Node 22 via `nvm` and pnpm 10.32.1 via Corepack.
- [x] The release PR explains that `0.5.1` ships the Plan reader refresh-scroll
      bug fix from #264/#261.

## Deferred Items

- Add a repo-local Node version hint such as `.nvmrc` or another tool pin. This
  was intentionally deferred so the patch release PR can stay minimal.

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
release-readiness repair is required.

#### Expected Files

- `VERSION`
- this tracked plan, including its archived form after closeout

#### Validation

- `scripts/validate-release` passes under Node 22 and pnpm 10.32.1.
- `git diff` shows no unrelated file changes.

#### Execution Notes

Updated `VERSION` from `0.5.0` to `0.5.1`.

The first #265 candidate grew broader after local Node 24 validation failures.
Discovery on 2026-06-28 showed `origin/main` passes `scripts/validate-release`
under CI-aligned Node 22 and pnpm 10.32.1, so the human chose to slim #265 back
to the minimal patch release instead of carrying validation-stability changes.

#### Review Notes

Earlier review rounds belong to the superseded broad candidate. After reopen,
review should focus on whether the branch was slimmed back to a minimal patch
release, whether Node 22 / pnpm 10.32.1 validation passes, and whether the PR
handoff clearly explains the #264/#261 fix.

### Step 2: Slim Reopened Release Candidate

- Done: [x]

#### Objective

Remove the validation-stability churn from the reopened #265 branch while
keeping the `0.5.1` patch release bump and harness plan history.

#### Details

Revert the candidate back to the smallest release PR:

- keep `VERSION` at `0.5.1`
- keep this plan as the tracked harness artifact
- remove changes to Vite config, pnpm approval files, installer smoke helpers,
  build scripts, validation scripts, and related tests unless they are already
  present on `origin/main`
- update the PR body after push so it describes only the patch release and the
  validation evidence

#### Expected Files

- `VERSION`
- `docs/plans/active/2026-06-26-patch-release-0-5-1.md`

#### Validation

- `git diff --name-status origin/main...HEAD` shows only the version bump and
  tracked plan/archive material.
- `scripts/validate-release` passes with Node 22 selected by `nvm` and pnpm
  10.32.1 selected by Corepack.
- Finalize review passes after the slim repair.

#### Execution Notes

Restored the reopened candidate to the smallest patch-release shape. The
staged diff against `origin/main` now contains only `VERSION` and this active
plan; the earlier Vite, pnpm, installer-smoke, script, and test changes were
removed from the effective PR diff.

Validation:

- `git diff --cached --name-status origin/main` showed only `VERSION` and
  `docs/plans/active/2026-06-26-patch-release-0-5-1.md`.
- `scripts/validate-release` passed on retry with `nvm` Node `v22.23.0` and
  Corepack pnpm `10.32.1`.

The first full validation attempt hit the existing installer smoke test's
5-second internal command timeout under local load. The specific timed-out test
passed when rerun alone, and the subsequent complete `scripts/validate-release`
run passed end-to-end without code changes.

#### Review Notes

`review-008-delta` passed on 2026-06-28 with no blocking or non-blocking
findings across `correctness`, `tests`, and `risk-scan`.

Reviewer conclusions:

- the effective branch diff is limited to `VERSION=0.5.1` plus the active plan
- the Node `v22.23.0` / pnpm `10.32.1` `scripts/validate-release` pass is
  sufficient for this VERSION-only patch release candidate
- the local installer-smoke timeout was adequately handled as a documented
  retry without broadening the PR
- Node-version hint work remains correctly deferred outside this release slice

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

Validated the slimmed release candidate with CI-aligned local tooling:

- Node `v22.23.0` selected through `nvm`
- pnpm `10.32.1` selected through Corepack
- `scripts/validate-release` passed end-to-end after one local installer-smoke
  timeout was retried successfully

The effective branch diff against `origin/main` contains only the `VERSION`
bump to `0.5.1` and tracked harness plan/archive material.

## Review Summary

The superseded broad #265 candidate review history is obsolete after reopen.
The relevant reopened-candidate closeout is `review-008-delta`, which passed on
2026-06-28 with no findings across `correctness`, `tests`, and `risk-scan`.

The first finalize review after reopen, `review-009-full`, found one blocking
handoff issue: local `HEAD` was slimmed, but PR #265 had not yet been pushed
and still showed the superseded broad diff. The local branch was pushed to
`origin/codex/patch-release-0.5.1`, and `gh pr diff 265 --name-only` then
showed only `VERSION` and
`docs/plans/active/2026-06-26-patch-release-0-5-1.md`.

Finalize review needs to be rerun after that handoff repair before archive.

## Archive Summary

- PR: https://github.com/catu-ai/easyharness/pull/265
- Ready: Pending finalize review, archive, push, and post-archive evidence
  refresh.
- Merge Handoff: Keep the PR body focused on the `0.5.1` patch release for the
  #264/#261 Plan reader refresh-scroll fix.

## Outcome Summary

### Delivered

- `VERSION` is bumped from `0.5.0` to `0.5.1`.
- The reopened #265 candidate is slimmed back to the minimal patch release
  scope.
- Local release validation passes with Node `v22.23.0` and pnpm `10.32.1`.
- PR #265 describes the patch release as shipping the already-landed #264/#261
  Plan reader refresh-scroll fix.

### Not Delivered

- No additional product fixes beyond the already landed #264/#261 Plan reader
  fix.
- No manual GitHub release, tag publication, or Homebrew publishing from this
  branch.
- No Node version hint in this slice.

### Follow-Up Issues

- https://github.com/catu-ai/easyharness/issues/266 tracks adding a repo-local
  Node version hint such as `.nvmrc`, `.node-version`, mise/asdf config, or
  equivalent contributor guidance.
