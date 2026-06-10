---
template_version: 0.2.0
created_at: 2026-06-11T00:10:48+08:00
source_type: direct_request
source_refs: ["PR #252 release GitHub CLI auth follow-up"]
size: S
---

# Stabilize release GitHub CLI auth

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Stabilize the release workflow's GitHub CLI authentication so the
`v0.4.3` release Homebrew verification can complete reliably after the
release-auth fix lands on `main`.

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
- After approval and merge, rerun `release.yml` from fixed `main` with
  `version=v0.4.3` and verify the release workflow, especially
  `Verify Homebrew Install`, completes successfully.
- Archive the plan with the final PR, merge, release rerun, and Homebrew
  verification evidence.

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
- [ ] After PR #252 lands, `release.yml` is rerun from `main` with
  `version=v0.4.3`.
- [ ] The rerun release workflow succeeds, including `Verify Homebrew Install`.
- [ ] The archived plan records release URL, workflow run URL, and Homebrew tap
  verification evidence.

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

### Step 2: Land PR #252

- Done: [ ]

#### Objective

After human approval, merge PR #252 and record normal harness land evidence.

#### Details

This step should not start until the plan is approved and execution is
recorded. The controller should verify PR #252 is up to date, CI is green, and
the diff still matches the scoped token-handling fix before merge.

#### Expected Files

- No additional code files are expected unless PR #252 needs a small rebase or
  review fix.

#### Validation

- PR #252 is merged to `main`.
- Local primary checkout is fast-forwarded to the merge commit.
- `harness land complete` returns the repository to `idle`.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Rerun and Verify the v0.4.3 Release Workflow

- Done: [ ]

#### Objective

Rerun `release.yml` from fixed `main` for `version=v0.4.3` and verify the
previously failing Homebrew verification job now succeeds.

#### Details

The release workflow checks out the requested release tag for release-source
artifact work while running smoke code from the root checkout. This is why
rerunning `release.yml` from the fixed `main` branch can validate the auth fix
without moving the `v0.4.3` tag or changing release artifact provenance.

If GitHub CLI or GitHub API calls fail again with 401, collect the exact job
log and distinguish auth flake from release asset, tag, or Homebrew formula
problems before retrying.

#### Expected Files

- No repository file changes are expected in this step.

#### Validation

- `gh workflow run release.yml --ref main -f version=v0.4.3` starts a release
  workflow run.
- The release run succeeds.
- `Verify Homebrew Install` succeeds.
- `gh release view v0.4.3 --repo catu-ai/easyharness --json tagName,url,assets`
  confirms the release remains published with the expected assets.
- The Homebrew tap formula remains at `version "0.4.3"` and points to
  `v0.4.3` assets.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

Use layered validation:

- local Go/smoke tests for workflow wiring and release verifier behavior
- PR #252 CI for repository-wide regression coverage
- post-merge release workflow rerun for the real `v0.4.3` release path
- release and Homebrew tap inspection to confirm published state remains
  coherent after rerun

## Risks

- Risk: The fix is validated in PR CI but not in the actual release workflow.
  - Mitigation: Treat the `v0.4.3` release workflow rerun as required
    acceptance evidence before archive.
- Risk: A repeated 401 is misread as a broken release artifact or formula.
  - Mitigation: Preserve exact job logs and compare against release asset and
    tap formula state before changing release artifacts.
- Risk: Rerunning release workflow accidentally changes release provenance.
  - Mitigation: Run with `--ref main -f version=v0.4.3`; the workflow verifies
    the release-source checkout matches the `v0.4.3` tag before building or
    uploading.

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
