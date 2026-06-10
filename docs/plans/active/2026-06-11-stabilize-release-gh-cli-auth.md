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

- [ ] Release workflow steps that invoke `gh` provide `GH_TOKEN` explicitly, or
  the release verifier maps `GITHUB_TOKEN` to `GH_TOKEN` before invoking `gh`.
- [ ] Tests cover the token fallback behavior and the release workflow wiring.
- [ ] `go test ./...` passes locally for the candidate.
- [ ] PR #252 CI passes.
- [ ] PR #252 is published, in sync with its branch, and CI passes.
- [ ] The archived plan records PR, CI, sync, and merge-readiness evidence.
- [ ] The candidate reaches `execution/finalize/await_merge` and waits for
  explicit human merge approval.
- [ ] The archived handoff clearly states that after PR #252 lands,
  `release.yml` must be rerun from `main` with `version=v0.4.3`, and that the
  rerun must verify `Verify Homebrew Install`.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Confirm and Validate the Auth Fix Candidate

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Publish and Archive the Merge-Ready Candidate

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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
