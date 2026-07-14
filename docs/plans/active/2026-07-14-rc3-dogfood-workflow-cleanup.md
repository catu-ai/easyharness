---
template_version: 0.3.0
created_at: "2026-07-14T15:17:47+08:00"
approved_at: "2026-07-14T16:50:00+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/293
    - https://github.com/yzhang1918/grove/pull/294
    - https://github.com/yzhang1918/grove/pull/295
    - https://github.com/yzhang1918/grove/pull/297
size: M
---

# RC3 Dogfood Workflow Cleanup

## Goal

Remove the remaining workflow friction exposed by three Grove RC2 runs while
preserving the lean design: one mandatory independent whole-candidate review,
explicit plan and merge approval, durable evidence, and recoverable land.

Make candidate identity base-aware, separate branch freshness from CI state,
simplify archive closeout, and eliminate redundant discovery and land
ceremony. Package the corrected managed resources and CLI behavior as the
`v0.6.0-rc.3` candidate for another short dogfood cycle before stable v0.6.0.

### Decisions and Constraints

- Keep one mandatory integrated finalize reviewer and linked delta repairs;
  do not restore review dimensions, step reviews, or specialist orchestration.
- Treat a base update as non-candidate work when the candidate-owned delta is
  unchanged and the sync introduces no candidate conflict or overlap requiring
  judgment. Candidate-affecting sync changes still require review.
- Model CI state, branch freshness, merge conflicts, and merge-policy blocking
  as distinct facts; a pending or unstable check must not itself make sync
  stale.
- Keep tracked plan closeout focused on durable candidate outcomes. Remote PR,
  CI, sync, merge, and land facts remain in runtime evidence and forge history.
- Rely on the forge merge record by default. Add a land comment only when an
  unresolved issue, deployment result, or material post-merge fact needs a
  durable handoff.
- Preserve the final-review requirement even for narrow or surprising changes;
  this slice removes false invalidation and redundant ceremony, not review.

## Scope

### In Scope

- Preserve valid finalize coverage across conflict-free upstream-only base
  synchronization when the candidate-owned delta is unchanged, while detecting
  candidate-affecting sync changes.
- Classify remote sync independently from pending/unstable CI and merge-policy
  states, with actionable status output and regression coverage.
- Support indented Closeout field continuations and remove archive-time fields
  whose final facts cannot exist before publish or land.
- Require concrete follow-up issue references for durable Deferred Items, and
  guide authors to use Out of Scope plus `Deferred Items: None` when no future
  commitment exists.
- Remove the extra discovery confirmation when no human steering decision
  remains, while retaining plan approval as the execution boundary.
- Clarify squash/rebase land ordering and recovery, and make final PR comments
  explicitly conditional rather than routine.
- Reproduce and fix any command-owned archive trailing-blank-line behavior that
  causes `git diff --check` noise.
- Update CLI contracts, managed bootstrap assets, tests, release metadata, and
  versioning for the `v0.6.0-rc.3` candidate.

### Out of Scope

- Changing the mandatory integrated reviewer, its fixed rubric, advisor model,
  or linked-delta design except for the deferred-follow-up invariant it must
  observe.
- Goal-oriented workflow, reviewer model configuration, or new workflow
  profiles.
- General GitHub provider abstraction or support for a new forge.
- Automatic merge, removal of human plan approval, or removal of human merge
  approval.
- Publishing or landing the RC3 release before explicit merge approval.

## Acceptance Criteria

- [x] A branch synchronized with unrelated upstream commits retains finalize
      coverage when its candidate delta is unchanged, while conflicts,
      overlapping candidate changes, or changed candidate deltas still require
      appropriate review.
- [x] Remote observation distinguishes base freshness, conflicts, CI
      pending/failure, and merge-policy blocking; GitHub `UNSTABLE` alone does
      not produce stale sync evidence.
- [x] Compact Closeout fields accept safe indented prose continuations, omit
      PR/Ready/Merge-Handoff lifecycle facts, and archive without trailing
      whitespace noise.
- [x] Archived plans with real Deferred Items require recognizable concrete
      issue references, while plans with no committed follow-up use Out of
      Scope and `Deferred Items: None` without inventing an issue.
- [x] Managed discovery guidance proceeds directly to planning when no
      steering choice remains, and managed land guidance makes comments
      conditional while supporting repository-selected squash/rebase flows.
- [x] Existing full-review and linked-delta correctness gates continue to pass,
      managed bootstrap outputs are synchronized, and release validation passes
      for version `0.6.0-rc.3`.

## Review Focus

- Verify base-sync preservation cannot bless a modified candidate patch,
  conflict resolution, overlapping upstream edit, or unreviewed local change.
- Verify sync evidence does not infer ancestry from GitHub check or policy
  states and still detects a genuinely behind or conflicted branch.
- Verify plan simplification leaves durable validation, review, delivered,
  not-delivered, and follow-up outcomes without duplicating runtime evidence.
- Verify follow-up validation cannot be satisfied by prose saying no issue was
  opened when Deferred Items contain real future work.
- Verify squash/rebase recovery no longer depends on temporarily restoring the
  feature branch, and ordinary successful merges do not generate empty-value
  PR comments.
- Verify the managed source assets, materialized skills, contracts, schemas,
  version, and release namespace remain synchronized.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Make remote and candidate identity base-aware

- Done: [x]
- Outcome: Review coverage, sync evidence, and land recovery distinguish the
  reviewed candidate delta from upstream-only history and independent remote
  CI or merge-policy states.
- Covers: Candidate-delta preservation criterion; remote-state classification
  criterion; squash/rebase recovery portion of the managed-guidance criterion.
- Check: Focused unit and end-to-end cases cover unchanged base sync, overlap,
  conflict, changed candidate delta, pending CI, and post-squash land.

### Step 2: Remove redundant plan and handoff ceremony

- Done: [x]
- Outcome: Compact plan closeout, deferred follow-up validation, discovery
  handoff, archive rendering, and land guidance express only durable decisions
  and necessary human steering.
- Covers: Closeout criterion; deferred-follow-up criterion; discovery and land
  guidance criterion.
- Check: Template, lint, archive, bootstrap, and representative workflow tests
  exercise readable continuations, concrete issue references, direct planning
  handoff, clean archive output, and conditional land comments.

### Step 3: Prepare the RC3 candidate

- Done: [x]
- Outcome: CLI behavior, formal contracts, managed resources, schemas, and
  release metadata consistently describe and validate the corrected RC3
  workflow.
- Covers: Existing-review regression and RC3 release-validation criterion.
- Check: Bootstrap and contract synchronization, full Go/E2E validation,
  release validation, and diff checks pass from a clean candidate.

## Validation Strategy

- Add graph-based Git fixtures for unchanged upstream merges, rebases or
  equivalent candidate deltas, overlapping edits, conflict repairs, and real
  post-review candidate changes.
- Test GitHub remote observations with CI pending/success/failure independently
  from CLEAN, BEHIND, conflicted, blocked, and merged PR states.
- Round-trip multiline Closeout values through lint, archive, reopen, and
  dashboard/status parsing, and assert command-rendered files pass
  `git diff --check`.
- Exercise Deferred Items with valid issue URLs/references, missing references,
  misleading no-issue prose, and `None`.
- Synchronize bootstrap assets and run the repository's complete test,
  validation, schema, and release checks before integrated finalize review.

## Closeout

- Validation: UPDATE_REQUIRED_AFTER_REOPEN — Full `go test ./... -count=1`, `scripts/validate-release`, contract and bootstrap synchronization checks, `git diff --check`, and repeated detached-worktree fixture checks passed for the final candidate.
- Review: UPDATE_REQUIRED_AFTER_REOPEN — `review-001-full` requested immutable current-PR sync identity; `review-002-delta` resolved it and found the optional publish-base edge; `review-003-delta` closed both remaining findings and passed with no new findings.
- Delivered: UPDATE_REQUIRED_AFTER_REOPEN — RC3 preserves review coverage across unchanged upstream-only syncs, binds fresh sync evidence to immutable PR base/head identity, separates remote states, simplifies durable plan/land ceremony, stabilizes archive output, and packages synchronized `v0.6.0-rc.3` CLI, contracts, schemas, and managed resources.
- Not Delivered: UPDATE_REQUIRED_AFTER_REOPEN — No stable `v0.6.0` release, GitHub release, automatic merge, new forge integration, or change to explicit human plan and merge approval.
- Follow-Up Issues: UPDATE_REQUIRED_AFTER_REOPEN — NONE
