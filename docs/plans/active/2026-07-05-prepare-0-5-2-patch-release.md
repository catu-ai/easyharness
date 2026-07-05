---
template_version: 0.2.0
created_at: "2026-07-05T12:27:48+08:00"
approved_at: "2026-07-05T12:29:05+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/pull/282
    - https://github.com/catu-ai/easyharness/issues/280
size: XS
---

# Prepare 0.5.2 Patch Release

## Goal

Prepare a dedicated patch release PR that bumps `easyharness` from `0.5.1` to
`0.5.2` after the managed `AGENTS.md` bootstrap wording fix landed in PR #282.

This is a release-safety workflow, so it uses the standard tracked-plan path
even though the expected code change is narrow. The release PR should stay
focused on the `VERSION` bump and release-readiness validation.

## Scope

### In Scope

- Update the root `VERSION` file from `0.5.1` to `0.5.2`.
- Run release-ready validation appropriate for a `VERSION` PR.
- Open a release PR against `main` for the patch release.
- Record publish, CI, and sync evidence until the candidate waits for merge
  approval.

### Out of Scope

- Publishing the GitHub Release or Homebrew tap update directly.
- Adding release-process machinery or changing release policy.
- Bundling unrelated fixes or documentation changes into the release PR.
- Merging the release PR without explicit human merge approval.

## Acceptance Criteria

- [x] `VERSION` contains `0.5.2`.
- [x] Release validation for the release PR passes or any failure is resolved
      before archive.
- [x] The archive handoff specifies that the release PR body should explain
      this patch ships the managed `AGENTS.md` bootstrap wording correction
      from PR #282.
- [x] The archive handoff requires publish, CI, and sync evidence before the
      candidate waits for merge approval.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Bump VERSION for 0.5.2

- Done: [x]

#### Objective

Update the release version file to `0.5.2` and validate the release candidate.

#### Details

The release policy says ordinary fixes leave `VERSION` alone and a dedicated
release PR carries the version bump once the intended scope is merged. The
patch scope here is the already-landed PR #282 managed bootstrap wording fix.

#### Expected Files

- `VERSION`

#### Validation

- Run `scripts/validate-release`.
- Run any focused checks needed if release validation reports a real issue.

#### Execution Notes

Updated `VERSION` from `0.5.1` to `0.5.2`. TDD was not applicable because
this is a release metadata bump rather than a behavior change. Release-ready
validation passed with `scripts/validate-release`.

#### Review Notes

Step closeout delta review `review-001-delta` passed with 0 findings across
`risk-scan` and `tests`. Review confirmed the delta is limited to `VERSION`
plus the active plan, matches the dedicated release PR policy, and uses the
documented release-ready validation profile for `VERSION` PRs.

## Validation Strategy

- Use `scripts/validate-release` as the release-ready validation profile for
  the `VERSION` PR.
- Use harness review before archive to check release scope, version bump
  correctness, and handoff readiness.

## Risks

- Risk: The release PR accidentally includes unrelated work.
  - Mitigation: Keep expected tracked changes to `VERSION` plus the active
    plan before archive, then archive before publish.
- Risk: The patch release is cut without release-ready validation.
  - Mitigation: Require `scripts/validate-release` before archive and record
    CI evidence after publish.

## Validation Summary

- `scripts/validate-release`
- `harness plan lint docs/plans/active/2026-07-05-prepare-0-5-2-patch-release.md`
- `git diff --check`

## Review Summary

- Step closeout delta review `review-001-delta` passed with 0 findings across
  `risk-scan` and `tests`.
- Finalize full review `review-002-full` passed with 0 findings across
  `risk-scan`, `tests`, and `docs-consistency`.

## Archive Summary

- PR: Pending publish after archive.
- Ready: The candidate is archive-ready after `VERSION` was bumped to `0.5.2`,
  release validation passed, and both step and finalize reviews passed with 0
  findings.
- Merge Handoff: Publish should open a release PR describing that this patch
  ships the managed `AGENTS.md` bootstrap wording correction from PR #282, then
  record publish, CI, and sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

- Updated `VERSION` from `0.5.1` to `0.5.2` for a dedicated patch release PR.
- Validated the release candidate with `scripts/validate-release`.

### Not Delivered

- No release was published.
- No release-process machinery, release policy, or unrelated product files were
  changed.

### Follow-Up Issues

NONE
