---
template_version: 0.2.0
created_at: "2026-06-11T00:10:48+08:00"
approved_at: "2026-06-11T00:15:51+08:00"
source_type: direct_request
source_refs:
    - 'PR #252 release GitHub CLI auth follow-up'
size: S
---

# Stabilize release GitHub CLI auth

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Stabilize the release workflow's GitHub CLI authentication and produce a
merge-ready candidate so the `v0.4.3` release Homebrew verification can be
rerun after the release-auth fix lands on `main`.

The immediate candidate is PR #252,
`https://github.com/catu-ai/easyharness/pull/252`, which was opened after
the `v0.4.3` release published assets successfully but the release workflow's
`Verify Homebrew Install` job repeatedly failed with GitHub API 401 errors
while the Homebrew smoke test inspected the previous Homebrew-capable release
(`v0.4.2`) before upgrading to `v0.4.3`.

## Scope

### In Scope

- Make GitHub CLI token handling explicit for release workflow steps that use
  `gh`.
- Keep the Homebrew smoke test's upgrade-path semantics: it may inspect the
  previous Homebrew-capable release before testing upgrade to the target
  release.
- Add or preserve regression coverage showing that release verification passes
  a usable `GH_TOKEN` to `gh` when only `GITHUB_TOKEN` is available.
- Validate the candidate locally and through PR CI.
- Publish and archive the candidate with PR, CI, and sync evidence until it is
  waiting for human merge approval.
- Record the required post-merge handoff: after PR #252 lands, rerun
  `release.yml` from fixed `main` with `version=v0.4.3` and verify
  `Verify Homebrew Install`.

### Out of Scope

- Changing release artifact contents for `v0.4.3`.
- Changing the `v0.4.3` tag target or rebuilding the already-published assets
  from a different commit.
- Removing the Homebrew smoke test's previous-version install and upgrade
  coverage.
- Broad release workflow redesign beyond the token handling needed to unblock
  this release verification.
- Changing repository secrets or replacing GitHub Actions authentication with a
  new credential system.

## Acceptance Criteria

- [x] Release workflow steps that invoke `gh` provide `GH_TOKEN` explicitly, or
  the release verifier maps `GITHUB_TOKEN` to `GH_TOKEN` before invoking `gh`.
- [x] Tests cover the token fallback behavior and the release workflow wiring.
- [x] `go test ./...` passes locally for the candidate.
- [x] PR #252 CI passes.
- [x] PR #252 is published, in sync with its branch, and CI passes.
- [x] The archived plan records PR, CI, sync, and merge-readiness evidence.
- [x] The candidate reaches `execution/finalize/await_merge` and waits for
  explicit human merge approval.
- [x] The archived handoff clearly states that after PR #252 lands,
  `release.yml` must be rerun from `main` with `version=v0.4.3`, and that the
  rerun must verify `Verify Homebrew Install`.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Confirm and Validate the Auth Fix Candidate

- Done: [x]

#### Objective

Confirm that the existing candidate PR #252 cleanly fixes the GitHub CLI token
handling problem without changing release or Homebrew upgrade semantics.

#### Details

The candidate should preserve the reason `v0.4.2` is inspected: the live
Homebrew smoke test validates upgrade from the previous Homebrew-capable
release to the requested release. The fix should instead make authentication
unambiguous for `gh` by passing `GH_TOKEN` in release workflow environments and
by ensuring `scripts/releaseverify` maps `GITHUB_TOKEN` to `GH_TOKEN` when
needed.

As of plan creation, PR #252 contains commit
`af9e02b41e7e2dfb427883b63d1709e9341cf72d`.

#### Expected Files

- `.github/workflows/release.yml`
- `scripts/releaseverify/main.go`
- `tests/smoke/homebrew_formula_test.go`
- `tests/smoke/verify_release_namespace_test.go`

#### Validation

- `go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyReleaseNamespace|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
- `go test ./scripts/releaseverify -count=1`
- `go test ./...`
- PR #252 `Go Test` check succeeds.

#### Execution Notes

Implemented before formal plan approval in PR #252, then brought under the
tracked plan before closeout. The candidate explicitly wires `GH_TOKEN` for
release workflow steps that call `gh`, maps `GITHUB_TOKEN` to `GH_TOKEN` in
`scripts/releaseverify` when needed, and adds fake-`gh` smoke coverage for the
fallback.

After approval, the branch was synced with latest `origin/main` via merge
commit `2689b84ac3821c2923ca2caa71a9d1849c50f0fe`.

Validation:

- `harness plan lint docs/plans/active/2026-06-11-stabilize-release-gh-cli-auth.md`
- `go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyReleaseNamespace|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
- PR #252 CI `Go Test` succeeded for head
  `2689b84ac3821c2923ca2caa71a9d1849c50f0fe` in run
  `https://github.com/catu-ai/easyharness/actions/runs/27289996283`.

#### Review Notes

Delta review `review-001-delta` anchored at
`f1cf3cdd7038452b137eac6633166f51b7cb6e3d` passed with 0 findings.

- `correctness`: no findings; reviewer verified token precedence, fallback
  behavior, workflow coverage, and fake-`gh` regression coverage.
- `release_safety`: no findings; reviewer verified v0.4.3 provenance remains
  unchanged, Homebrew previous-version upgrade semantics remain intact, and
  release rerun work is deferred until after merge approval.

### Step 2: Publish and Archive the Merge-Ready Candidate

- Done: [x]

#### Objective

Publish the scoped fix as PR #252, record CI and sync evidence, and archive the
candidate at `wait_for_merge_approval` without merging it.

#### Details

This step should not merge PR #252. The controller should verify PR #252 is up
to date, CI is green, and the diff still matches the scoped token-handling fix.
The archive handoff must say that merge approval is still required and that
the `v0.4.3` release workflow rerun is post-merge work.

#### Expected Files

- No additional code files are expected unless PR #252 needs a small rebase or
  review fix.

#### Validation

- PR #252 is open, in sync, and has passing CI.
- Harness publish/sync evidence records the PR URL, head commit, and CI result.
- The plan is archived and `harness status` reaches
  `execution/finalize/await_merge`.

#### Execution Notes

PR #252 is published at
`https://github.com/catu-ai/easyharness/pull/252`.

The branch was synced with latest `origin/main`
(`f1cf3cdd7038452b137eac6633166f51b7cb6e3d`) by merge commit
`2689b84ac3821c2923ca2caa71a9d1849c50f0fe`.

PR CI was observed passing during closeout, including the repair run for the
committed Step 2 notes:

- run: `https://github.com/catu-ai/easyharness/actions/runs/27291218500`
- job: `Go Test`
- result: success in 4m12s

Final PR head, CI, and sync facts must be recorded through harness
publish/CI/sync evidence after archive, so this tracked note does not pretend
to be the final remote evidence after later archive commits.

Post-merge handoff remains required: after PR #252 is merged, rerun
`release.yml` from `main` with `version=v0.4.3` and verify
`Verify Homebrew Install`.

Reopened after merge approval because `origin/main` advanced with PR #251
before GitHub accepted the merge. The branch was refreshed by merging
`origin/main` at `df56b0ffc3e7d5bc3ca06121221d564a6a8f46f6` into PR #252.
This was a sync-only finalize repair; the release-auth code and post-merge
handoff did not change.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only records publish/sync/CI readiness and the
post-merge release rerun handoff. The behavior-changing release-auth fix
already passed step-bound delta review `review-001-delta`; finalize review
will cover the full archived candidate.

## Validation Strategy

Use layered validation:

- local Go/smoke tests for workflow wiring and release verifier behavior
- PR #252 CI for repository-wide regression coverage
- harness publish/sync evidence to prove the PR is merge-ready
- post-merge handoff evidence describing the required `v0.4.3` release rerun

## Risks

- Risk: The fix is validated in PR CI but not in the actual release workflow.
  - Mitigation: Treat the `v0.4.3` release workflow rerun as required
    post-merge handoff work and record it in the archive before waiting for
    merge approval.
- Risk: A repeated 401 is misread as a broken release artifact or formula.
  - Mitigation: Preserve exact job logs and compare against release asset and
    tap formula state before changing release artifacts.
- Risk: Rerunning release workflow accidentally changes release provenance.
  - Mitigation: The handoff should instruct the landing agent to run with
    `--ref main -f version=v0.4.3`; the workflow verifies the release-source
    checkout matches the `v0.4.3` tag before building or uploading.

## Validation Summary

- `harness plan lint docs/plans/active/2026-06-11-stabilize-release-gh-cli-auth.md`
  passed during planning and closeout.
- Focused release-auth smoke validation passed:
  `go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyReleaseNamespace|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`.
- `go test ./scripts/releaseverify -count=1` passed; that package has no
  standalone test files, but it still compiles.
- Full repository validation passed locally during candidate review:
  `go test ./...`.
- PR #252 CI passed repeatedly during revision 1 closeout. The historical
  pre-reopen run for head `7c42f320e124857e419717592aba4cbd22c4cc02` was
  `https://github.com/catu-ai/easyharness/actions/runs/27291661103`, with
  `Go Test` succeeding in 5m32s.
- Final PR head, CI, and sync evidence are recorded through harness publish,
  CI, and sync evidence after archive.
- Reopen validation after syncing PR #252 with `origin/main` at
  `df56b0ffc3e7d5bc3ca06121221d564a6a8f46f6`:
  - focused release-auth smoke test passed:
    `go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyReleaseNamespace|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
  - release verifier package compile passed:
    `go test ./scripts/releaseverify -count=1`
  - affected package sweep passed:
    `go test ./internal/... ./cmd/... ./scripts/releaseverify ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyReleaseNamespace|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
  - full local `go test ./...` was attempted after the sync merge, but the
    local installer smoke `TestInstallDevHarnessDefaultsToUserLocalBin`
    exceeded a 3 minute focused rerun timeout in this worktree. Final merge
    readiness for the refreshed branch depends on the post-push PR CI `Go Test`
    result.

## Review Summary

- Step delta review `review-001-delta` passed with 0 findings. Reviewers
  checked correctness and release safety.
- Finalize review `review-002-full` found that Step 2 closeout was still local
  and that the PR body did not explicitly name `Verify Homebrew Install` as
  the post-merge verification target. Both findings were fixed by committing
  Step 2 notes and updating the PR body.
- Finalize review `review-003-full` found stale wording that treated a
  superseded PR head/run as final evidence. The plan now states that final PR
  head, CI, and sync facts belong to harness evidence after archive.
- Finalize repair review `review-004-full` passed with 0 findings.
- Reopened finalize repair after PR #251 advanced `main`; finalize review
  `review-005-full` found two sync-readiness wording findings in the plan:
  the wrong `origin/main` SHA and stale "current PR head" wording for
  revision-1 CI evidence.
- Follow-up finalize review `review-006-full` passed with 0 findings after
  those sync-readiness findings were fixed.

## Archive Summary

- Archived At: 2026-06-11T09:56:10+08:00
- Revision: 2
- PR: PR #252 is published at
  `https://github.com/catu-ai/easyharness/pull/252`.
- Ready: The candidate is merge-ready after local validation, passing PR CI,
  and finalize review; it intentionally stops before merge and waits for
  explicit human merge approval.
- Merge Handoff: After PR #252 lands, rerun `release.yml` from `main` with
  `version=v0.4.3` and verify that the rerun succeeds, including the
  `Verify Homebrew Install` job. This rerun should validate the release smoke
  fix without moving the `v0.4.3` tag or changing release artifact provenance.
- Reopen Repair: Revision 2 refreshed PR #252 against `origin/main` after PR
  #251 landed and invalidated the previous sync evidence. No release-auth scope
  changed; final PR head, CI, sync, and merge-ready evidence must be recorded
  again after re-archive.

## Outcome Summary

### Delivered

- Release workflow `gh` calls now receive explicit `GH_TOKEN` alongside
  `GITHUB_TOKEN`.
- `scripts/releaseverify` maps a non-empty `GITHUB_TOKEN` into missing or empty
  `GH_TOKEN` for child `gh` invocations while preserving an existing non-empty
  `GH_TOKEN`.
- Smoke tests cover workflow wiring and fake-`gh` fallback behavior.
- PR #252 body includes explicit post-merge release verification instructions
  for `release.yml`, `version=v0.4.3`, and `Verify Homebrew Install`.

### Not Delivered

The actual `v0.4.3` release workflow rerun is not performed before merge. It
must happen after PR #252 lands on `main`, per the post-merge handoff.

### Follow-Up Issues

NONE
