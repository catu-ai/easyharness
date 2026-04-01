---
template_version: 0.2.0
created_at: 2026-04-01T20:44:49+08:00
source_type: direct_request
source_refs: []
workflow_profile: lightweight
---

# Cut Release VERSION to 0.1.0-alpha.6

## Goal

Ship the next alpha release by advancing the root `VERSION` file from
`0.1.0-alpha.5` to `0.1.0-alpha.6` in a dedicated release PR. This intentionally
avoids changing the bootstrap workflow logic; instead, it uses the newly
VERSION-driven release path the way maintainers expect to use it going
forward.

The outcome should be a merge-ready release PR that bumps the version, updates
any release-facing docs that still mention the old example version, and leaves
the repository ready for automation to create the new `v0.1.0-alpha.6` tag and
dispatch the existing `Release` workflow after merge.

## Scope

### In Scope

- Update the root `VERSION` file to `0.1.0-alpha.6`.
- Refresh maintainer-facing release docs where the example next version should
  now point to `0.1.0-alpha.6`.
- Run the relevant local validation for the release bump PR.
- Carry the change through review, archive, publish, and wait-for-merge
  readiness.

### Out of Scope

- Fixing the one-time bootstrap workflow behavior for already-published tags.
- Changing release cadence, stable-version policy, or workflow semantics.
- Any non-release product or harness behavior changes.

## Acceptance Criteria

- [ ] The root `VERSION` file is updated to `0.1.0-alpha.6`.
- [ ] Maintainer docs accurately describe `0.1.0-alpha.6` as the next release
      example where the old example would now be stale or confusing.
- [ ] Relevant local validation passes for the release bump candidate.
- [ ] The candidate reaches archived, published, merge-ready state for a
      dedicated alpha.6 release PR.

## Deferred Items

- Consider a separate follow-up only if the one-time bootstrap workflow failure
  needs to be cleaned up later for repository hygiene.

## Work Breakdown

### Step 1: Prepare and publish the alpha.6 release bump

- Done: [x]

#### Objective

Update `VERSION`, refresh any adjacent release guidance, validate the release
bump, and carry the dedicated alpha.6 candidate through the lightweight
publish path.

#### Details

Keep the diff intentionally narrow and release-focused. The release PR should
remain separate from feature work and should not change release automation
logic. This plan uses `workflow_profile: lightweight` as a user-approved
exception to the repository's default lightweight eligibility guidance because
the actual code change is intentionally minimal: a one-line release version
bump plus matching release-doc examples.

#### Expected Files

- `VERSION`
- `docs/releasing.md`
- `README.md`
- `docs/plans/active/2026-04-01-version-tag-bootstrap-skip.md`

#### Validation

- `VERSION` resolves to `v0.1.0-alpha.6`.
- Docs remain internally consistent about the release flow and examples.
- Relevant local tests or release checks pass.
- Review and archive artifacts accurately reflect that this is a dedicated
  alpha.6 release bump candidate.

#### Execution Notes

Updated `VERSION` from `0.1.0-alpha.5` to `0.1.0-alpha.6` and refreshed the
README release-process example to match. `docs/releasing.md` already used the
alpha.6 examples, so no change was needed there. Validated the bump with
`scripts/read-release-version --tag` and `go test ./tests/smoke -run
ReleaseVersion -count=1`.

#### Review Notes

`review-001-delta` passed clean for `docs_consistency` and `risk_scan` with no
findings. The release bump stayed narrow to `VERSION` plus the matching README
example, and the lightweight exception remains explicitly documented in this
plan.

## Validation Strategy

- Run the narrowest relevant local validation for the release bump candidate.
- Re-read the docs and version file together to confirm the examples and tag
  mapping remain correct.

## Risks

- Risk: The version bump could still leave stale release examples or summaries
  that mention alpha.5.
  - Mitigation: Update the few maintainer-facing docs that are part of the
    release path and verify the rendered version examples match `0.1.0-alpha.6`.

## Validation Summary

Validated the alpha.6 release bump by resolving the release tag directly from
the repo-owned `VERSION` file and by rerunning the focused smoke coverage for
release-version handling:

- `scripts/read-release-version --tag`
- `go test ./tests/smoke -run ReleaseVersion -count=1`

The candidate only changes `VERSION`, the matching README release example, and
this tracked lightweight plan.

## Review Summary

Step-closeout delta review `review-001-delta` passed clean for
`docs_consistency` and `risk_scan`. Finalize full review `review-002-full`
initially found one blocking `agent_ux` issue: the tracked plan lacked a
durable PR/merge breadcrumb for the lightweight handoff. The current candidate
repairs that gap by recording explicit `PR` and `Merge Handoff` entries in the
plan so archive/publish work no longer depends on chat history.

## Archive Summary

The candidate is a user-approved lightweight exception for a dedicated
`0.1.0-alpha.6` release bump. It keeps the tracked diff intentionally narrow:
update `VERSION`, align the README maintainer example, and preserve the
existing release automation unchanged. The remaining work is archive, open or
refresh the release PR, leave the lightweight breadcrumb in the PR body, and
record publish/CI/sync evidence until merge-ready.

- PR: NONE. The alpha.6 release PR has not been opened yet.
- Ready: The version bump, focused validation, and tracked step review are
  complete; only finalize review closeout, archive, and publish handoff remain.
- Merge Handoff: After archive, commit the tracked archive update, push branch
  `codex/version-alpha-6-release`, open or refresh the dedicated alpha.6
  release PR, add a PR-body breadcrumb explaining that this lightweight plan is
  a user-approved exception for a one-line release bump, and record
  publish/CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Root `VERSION` bump from `0.1.0-alpha.5` to `0.1.0-alpha.6`.
- Matching README maintainer guidance that now uses alpha.6 as the example
  release version.
- A tracked lightweight plan that records the validation, review closeout, and
  merge-handoff expectations for this dedicated release PR.

### Not Delivered

- No bootstrap-workflow fix for the one-time historical alpha.5 tag conflict.
- No broader release-policy or versioning-flow changes beyond cutting alpha.6.

### Follow-Up Issues

NONE
