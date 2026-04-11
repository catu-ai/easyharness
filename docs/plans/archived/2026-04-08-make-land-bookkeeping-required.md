---
template_version: 0.2.0
created_at: "2026-04-08T20:43:52+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/16
size: S
---

# Make Harness-Land Post-Merge Bookkeeping Required

## Goal

Clarify the `harness-land` workflow so merge-time remote bookkeeping is treated
as required closeout work rather than optional cleanup. The result should make
future agents consistently leave the permanent PR record in good shape and
close or update linked issues before they treat land cleanup as complete.

This is a workflow-contract change, not a new remote-state enforcement system.
The repository should communicate the requirement clearly through the packaged
skill, synced repo-local outputs, and CLI next-action guidance.

## Scope

### In Scope

- Tighten the bootstrap `harness-land` skill wording so post-merge bookkeeping
  is explicitly required work.
- Define when a final PR comment must be added and the minimum content it
  should include.
- Define when linked issues should be closed versus updated with a follow-up
  reference.
- Update `land` and `status` next-action guidance so the CLI reinforces the
  same completion expectations.
- Add or update tests that protect the contract wording from regressing.

### Out of Scope

- Adding new remote-state modeling or GitHub API validation to prove PR
  comments or issue updates happened.
- Changing lifecycle node semantics or adding new land-state persistence.
- Expanding the work into publish, CI, or sync evidence behavior tracked by
  other issues.

## Acceptance Criteria

- [x] The bootstrap `harness-land` skill states that merge-time PR and issue
      bookkeeping is required before land cleanup is considered complete.
- [x] The skill defines when a final PR comment is required and names the
      minimum content that comment must contain.
- [x] The skill defines when linked issues should be closed versus updated with
      a follow-up reference.
- [x] CLI next-action guidance for `harness land` and `harness status` reflects
      the same required bookkeeping expectations.
- [x] Automated tests cover the tightened wording or output so future changes
      do not silently soften the contract again.

## Deferred Items

- System-enforced confirmation that the required PR comment or linked-issue
  updates actually exist on the remote.
- Richer `harness status` modeling for post-merge remote bookkeeping progress.

## Work Breakdown

### Step 1: Define the required land-bookkeeping contract

- Done: [x]

#### Objective

Rewrite the packaged `harness-land` guidance so remote bookkeeping is a clear
required part of post-merge closeout, with concrete rules for PR comments and
linked-issue handling.

#### Details

Edit the bootstrap skill source rather than hand-editing materialized repo
outputs. Make the wording specific enough that a cold agent can tell when a
final PR comment is required, what minimum facts it should capture, when an
issue is closed directly, and when a follow-up update is the right outcome.

#### Expected Files

- `assets/bootstrap/skills/harness-land/SKILL.md`

#### Validation

- The bootstrap skill text clearly describes required remote bookkeeping and
  completion gating without introducing new remote-state mechanisms.
- The wording remains aligned with issue `#16` and does not expand into a
  stronger enforcement design.

#### Execution Notes

Updated `assets/bootstrap/skills/harness-land/SKILL.md` to move post-merge
bookkeeping into the required `land` cleanup path, define when a final PR
comment is required, list the minimum closeout content, and distinguish issue
closure from follow-up updates for unresolved work.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step changed only workflow guidance text and was
reviewed together with the follow-on CLI wording and tests in the same bounded
slice.

### Step 2: Sync repo outputs and CLI guidance to the contract

- Done: [x]

#### Objective

Propagate the clarified contract into synced repo-local skill outputs and the
CLI next-action messages that guide land cleanup.

#### Details

Run the bootstrap sync after editing the packaged skill. Update `status` and
`land` next-action text so they describe post-merge bookkeeping as required
closeout rather than generic cleanup. Keep this to message-level changes; do
not add new state validation.

#### Expected Files

- `.agents/skills/harness-land/SKILL.md`
- `AGENTS.md`
- `internal/lifecycle/service.go`
- `internal/status/service.go`

#### Validation

- `scripts/sync-bootstrap-assets` completes cleanly.
- CLI next-action strings consistently mention the required PR/issue
  bookkeeping expectations.
- No lifecycle or persistence behavior changes are introduced.

#### Execution Notes

Ran `scripts/sync-bootstrap-assets` to refresh the repo-local `harness-land`
skill output, then tightened `internal/lifecycle/service.go` and
`internal/status/service.go` so `land` guidance now calls out required PR and
linked-issue bookkeeping before `harness land complete`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This was a message-only alignment change and the updated
guidance is covered by focused tests in the next step.

### Step 3: Lock the contract with tests

- Done: [x]

#### Objective

Add regression coverage for the updated land-bookkeeping guidance in the CLI
and synced assets.

#### Details

Prefer focused tests or existing output assertions over broad snapshots. Cover
the user-visible wording surfaces most likely to regress: status/land next
actions and any bootstrap-sync-sensitive artifacts that should preserve the new
required contract language.

#### Expected Files

- `internal/lifecycle/service_test.go`
- `internal/status/service_test.go`
- optional existing bootstrap sync or asset drift tests if they are the right
  fit

#### Validation

- Relevant tests fail before the wording change and pass after it.
- Coverage protects the specific requirement that PR and linked-issue
  bookkeeping is required before `harness land complete`.

#### Execution Notes

Added focused regression coverage in `internal/lifecycle/service_test.go` and
`internal/status/service_test.go` for the new land-bookkeeping wording. Ran
`go test ./internal/lifecycle ./internal/status ./internal/bootstrapsync ./tests/smoke -run 'TestLandGuidanceRequiresPRAndIssueBookkeeping|TestStatusLandNode|TestSyncBootstrapAssetsCheckPassesForCurrentRepo'`
and all targeted checks passed. A finalize review then caught stale CLI help
wording in `internal/cli/app.go`; repaired that mismatch and added a smoke test
for `harness land --help`, validated with
`go test ./internal/cli ./tests/smoke -run 'TestHelpShowsTopLevelUsage|TestLandHelpShowsRequiredBookkeepingContract'`.
The next finalize review found two more stale user-facing phrases in
`runLandEntry`, `runLandComplete`, and await-merge status guidance; repaired
them and expanded smoke/status coverage with
`go test ./internal/status ./tests/smoke -run 'TestStatusArchivedPlanReadyForAwaitMerge|TestHelpShowsTopLevelUsage|TestLandHelpShowsRequiredBookkeepingContract|TestLandEntryUsageShowsRequiredBookkeepingContract|TestLandCompleteHelpShowsRequiredBookkeepingContract'`.
The following finalize review then surfaced the last outer-layer wording drift
in top-level help plus `land` / `land complete` summaries; repaired those and
validated with
`go test ./internal/lifecycle ./internal/status ./tests/smoke -run 'TestLandCompleteWritesIdleMarkerForStatus|TestStatusLandNode|TestStatusArchivedPlanReadyForAwaitMerge|TestHelpShowsTopLevelUsage|TestLandHelpShowsRequiredBookkeepingContract|TestLandEntryUsageShowsRequiredBookkeepingContract|TestLandCompleteHelpShowsRequiredBookkeepingContract'`.
One more finalize review then found stale `harness-land` skill metadata and the
land-entry sentence still using `post-merge cleanup`; repaired the bootstrap
source, synced repo-local outputs, and validated with
`go test ./internal/lifecycle ./internal/status ./internal/bootstrapsync ./tests/smoke -run 'TestLandCompleteWritesIdleMarkerForStatus|TestLandGuidanceRequiresPRAndIssueBookkeeping|TestStatusLandNode|TestStatusArchivedPlanReadyForAwaitMerge|TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestHelpShowsTopLevelUsage|TestLandHelpShowsRequiredBookkeepingContract|TestLandEntryUsageShowsRequiredBookkeepingContract|TestLandCompleteHelpShowsRequiredBookkeepingContract'`.
The next finalize review then found remaining lifecycle wording drift in
`reopen`, `land`, `land complete`, and land-readiness errors. Repaired those
user-facing strings, synced bootstrap outputs again, and reran the same focused
validation set.
The next finalize review then found two last consistency gaps: durable docs
still described land as generic cleanup, and runtime next-action text still
said `Record cleanup completion`. Repaired README/spec wording plus the
remaining runtime milestone labels, then reran the focused lifecycle/status and
smoke checks.
The next finalize review then found the final user-facing leftovers in the
evidence service, review UI summary, and e2e contract text. Repaired those
surfaces and reran focused evidence/review UI/E2E checks, then confirmed the
remaining `cleanup` mentions were limited to internal comments and test failure
messages rather than active user-facing contract surfaces.
One more finalize review found a final stale `land cleanup` phrase in the
normative CLI contract purpose text for `harness land --pr`; repaired it and
confirmed the user-facing repository surfaces were clear of the old wording.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 was exercised by repeated finalize reviews
(`review-001-full` through `review-008-full`) that flushed out wording drift
across CLI help, status/lifecycle summaries, durable specs, evidence/review UI
surfaces, E2E contract text, and the packaged `harness-land` skill metadata.
Those findings have been repaired; the remaining review work is the fresh
finalize pass needed before archive.

## Validation Strategy

- Run `harness plan lint` on the tracked plan before execution.
- Run `scripts/sync-bootstrap-assets` after bootstrap skill edits.
- Run the focused Go tests covering lifecycle/status next-action output and any
  related asset-sync checks that protect the contract wording.

## Risks

- Risk: The wording could become more explicit in one surface but stay vague in
  another, leaving agents with mixed guidance.
  - Mitigation: Update the bootstrap source first, sync materialized outputs,
    and cover the CLI next-action messages with tests in the same slice.
- Risk: The change could drift into enforcement semantics that issue `#16`
  explicitly did not request.
  - Mitigation: Keep the work limited to skills, user-visible guidance, and
    regression tests for those messages.

## Validation Summary

Validated the contract tightening with focused lifecycle, status, smoke,
evidence, review UI, bootstrap-sync, and E2E checks while repeatedly sweeping
old `cleanup` wording out of user-facing surfaces. Representative green runs
included:

- `go test ./internal/lifecycle ./internal/status ./internal/bootstrapsync ./tests/smoke -run 'TestLandCompleteWritesIdleMarkerForStatus|TestLandGuidanceRequiresPRAndIssueBookkeeping|TestStatusLandNode|TestStatusArchivedPlanReadyForAwaitMerge|TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestHelpShowsTopLevelUsage|TestLandHelpShowsRequiredBookkeepingContract|TestLandEntryUsageShowsRequiredBookkeepingContract|TestLandCompleteHelpShowsRequiredBookkeepingContract'`
- `go test ./internal/evidence ./internal/reviewui ./tests/e2e -run 'TestServiceReadHidesArchivedRoundsDuringLand|TestServiceReadHidesArchivedRoundsDuringLegacyLandCleanup|TestTransitionFamiliesAreUniqueAndComplete|TestLandWorkflow'`

A final repository-wide grep over the active contract surfaces showed no
remaining user-facing `land cleanup`, `post-merge cleanup`, `merge cleanup`,
or `cleanup completion` wording outside internal comments or test failure
messages.

## Review Summary

Finalize review required several repair loops because the wording drift reached
more surfaces than the original issue description suggested. `review-001-full`
through `review-008-full` each found another remaining user-facing surface that
still softened the contract, spanning CLI help, lifecycle/status messages,
bootstrap skill metadata, durable specs, evidence/review UI summaries, and E2E
contract text. After those repairs, `review-009-full` passed cleanly with no
findings in either slot.

## Archive Summary

- Archived At: 2026-04-08T21:13:49+08:00
- Revision: 1
- PR: NONE. Create or refresh the PR after committing and pushing this branch,
  then record the PR URL through publish evidence.
- Ready: Acceptance criteria are satisfied, the final full finalize review
  `review-009-full` passed cleanly, and the user-facing contract surfaces now
  consistently describe `harness land complete` as the required post-merge
  bookkeeping milestone.
- Merge Handoff: Create or switch to a `codex/` branch for this slice, commit
  the tracked contract/test/doc updates, push the branch, open or update the
  PR, then record publish/CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Tightened the packaged `harness-land` contract so final PR closeout comments
  and linked-issue updates are explicit required work before
  `harness land complete`.
- Synced the repo-local `harness-land` skill output and updated lifecycle,
  status, CLI help, evidence, and review UI wording so the live agent guidance
  consistently uses the required post-merge bookkeeping language.
- Updated durable README/spec/E2E contract text and added focused regression
  assertions so future drift back to the old cleanup framing is more likely to
  be caught quickly.

### Not Delivered

- This slice did not add remote-state enforcement or GitHub API verification
  for whether the required PR/issue bookkeeping has actually happened.
- This slice did not add richer structured `land`-phase progress modeling
  beyond the wording-level contract updates.

### Follow-Up Issues

- `#115` Consider stronger land bookkeeping completion verification.
- `#116` Model post-merge bookkeeping progress in harness status.
