---
template_version: 0.1.0
created_at: "2026-03-20T00:00:00+08:00"
source_type: issue
source_refs:
    - '#13'
size: XL
---

# Surface archive blockers, archive handoff, landed status, and execute TDD discipline

## Goal

Expose archive-readiness problems before the final `harness archive` write so
the controller can fix closeout gaps from `harness status` instead of only at
the freeze step. `harness archive` should run the same readiness evaluation as
a dry run before it mutates tracked files or local pointers.

This slice also tightens three workflow edges discovered during discovery:
after archive, `harness status` should stop presenting a merely local archived
candidate as if it were already waiting for merge approval; after land,
`harness status` should stop presenting the archived candidate as still
current; and the execute skill should explicitly require Red/Green/Refactor
TDD discipline for behavior changes.

## Scope

### In Scope

- Define which archive blockers should be surfaced before archive and expose
  them through `harness status`.
- Reuse the same readiness evaluation inside `harness archive` before any
  tracked file or local-state write happens.
- Keep the deferred-follow-up contract simple: when `## Deferred Items`
  contains real items at archive time, `## Outcome Summary > Follow-Up Issues`
  must not remain `NONE`.
- Clarify the time-based split between archive-known deferred work and
  retrospective follow-up discovered only after land.
- Distinguish coarse tracked lifecycle from local post-archive handoff state so
  `harness status` can tell whether the archived candidate still needs publish
  work, is waiting on post-archive CI, or is truly ready to wait for merge
  approval.
- Add disposable landed-worktree state so `harness status` can report an idle
  post-land worktree instead of an outdated `awaiting_merge_approval`
  candidate.
- Update execute guidance so behavior-changing work follows
  Red/Green/Refactor TDD by default, with documented exceptions only when TDD
  is genuinely impractical.

### Out of Scope

- First-class remote PR, branch publication, mergeability, or CI modeling from
  issue `#12`.
- Adding a standalone `harness preflight` command.
- Redesigning the tracked plan lifecycle beyond the local `harness land record`
  marker, such as adding a new persisted `landed` frontmatter state.
- Building first-class publish/PR/CI capture commands in the CLI before issue
  `#12`; this slice only consumes existing local publish and CI state when it
  is present.
- Verifying specific follow-up reference formats beyond the rule that
  `Follow-Up Issues` must not stay `NONE` when deferred items remain.

## Acceptance Criteria

- [x] `harness status` reports concrete archive blockers before archive,
      including deferred-item follow-up gaps and missing archive-summary
      content, with fix-oriented next actions.
- [x] `harness archive` runs the shared archive-readiness evaluation before any
      tracked or local-state writes, and a failing preflight leaves the current
      plan and `.local` pointers untouched.
- [x] After archive, `harness status` distinguishes a local archived candidate
      that still needs commit/push/PR handoff from one that is truly ready to
      wait for merge approval, without introducing new tracked lifecycle
      values.
- [x] After post-merge land cleanup records a landed local marker, `harness
      status` no longer reports the archived candidate as waiting for merge
      approval and instead surfaces an idle/no-current-plan worktree with
      last-landed context.
- [x] `harness-execute` and its step-inner-loop reference explicitly require
      Red/Green/Refactor TDD for behavior changes, and the docs say skipped
      TDD must be justified.

## Deferred Items

- First-class remote PR/CI/publish state remains deferred to `#12`.

## Work Breakdown

### Step 1: Align archive, land, and execute contracts

- Done: [x]

#### Objective

Update the durable specs and repo-local skills so archive blockers, landed
cleanup, and execute-time TDD rules all describe the same workflow.

#### Details

Capture the intended split explicitly: deferred items known before archive must
be closed out through the archived plan, while retrospective follow-up
discovered after land belongs in issues or PR comments instead of retroactive
plan edits. Define landed-worktree behavior as disposable local state rather
than a new tracked-plan lifecycle. Borrow the Red/Green/Refactor wording from
`missless` for behavior-changing work and document when skipping TDD is
allowed.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`
- `README.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/closeout-and-archive.md`
- `.agents/skills/harness-execute/references/step-inner-loop.md`
- `.agents/skills/harness-land/SKILL.md`

#### Validation

- The specs, README, and skill docs agree on archive-readiness, landed cleanup,
  and TDD expectations without contradicting each other.
- `harness plan lint` still passes for the tracked plan after any scope notes
  or examples are updated.

#### Execution Notes

Updated the durable docs and repo-local skills to agree on the new behavior:
`harness status` may now report structured archive blockers, `Follow-Up
Issues` only needs to stop being `NONE` when deferred items remain, land
cleanup should leave the worktree in an idle-after-land local state, and
behavior-changing execute work now defaults to Red/Green/Refactor TDD. The
closeout reference now tells agents to fix `status.blockers` before trying
`harness archive`.

#### Review Notes

Doc and skill changes were checked against the new CLI behavior implemented in
this slice, then validated by `harness plan lint`, `go test ./internal/plan
./internal/status ./internal/lifecycle`, `go test ./...`, and a direct
`harness status` run after reinstalling the dev binary.

### Step 2: Share archive-readiness evaluation between status and archive

- Done: [x]

#### Objective

Implement a single archive-readiness evaluation that `harness status` can
surface early and `harness archive` can enforce before any write happens.

#### Details

Factor the current late archive checks into reusable logic that covers
acceptance completion, step completion, archive placeholders, completed-step
placeholders, required archive-summary lines, deferred-item follow-up
non-`NONE`, and archive-sensitive local state such as review, CI, and sync.
`harness status` should report the blockers with repair-first guidance, while
`harness archive` should reuse the same evaluation before writing the archived
plan or updating `.local` pointers.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/plan/document.go`

#### Validation

- Targeted tests prove `harness status` surfaces concrete blocker messages for
  incomplete closeout state instead of generic archive guidance.
- Targeted tests prove `harness archive` returns preflight failures without
  mutating tracked files or local-state pointers.

#### Execution Notes

Extracted plan-local archive checks into `Document.ArchiveReadinessIssues()` and
added `lifecycle.EvaluateArchiveReadiness()` so both `harness status` and
`harness archive` read from the same blocker list. `harness archive` now
preflights before any tracked-file or `.local` write, while `harness status`
surfaces the blockers through a dedicated `blockers` field, blocker-aware
summary text, and fix-first next actions once the work breakdown is complete.

#### Review Notes

Added regression tests that prove status reports missing archive-summary fields
and deferred-item follow-up gaps before archive, and that a failing archive
preflight leaves the active plan and current pointers untouched. `go test
./internal/plan ./internal/status ./internal/lifecycle` and `go test ./...`
both passed.

### Step 3: Represent landed worktrees as idle local state

- Done: [x]

#### Objective

Teach status to represent a post-land worktree as idle state with last-landed
context instead of reusing the archived candidate as if merge approval were
still pending.

#### Details

Keep landed information in `.local` and avoid editing archived plans after
merge. Remove ambiguous archived-plan fallback from current-plan detection,
persist enough disposable local state to remember the most recently landed
plan, and make status prefer active plans when they exist while surfacing an
idle-after-land summary when only landed context remains. Update the land skill
so its cleanup step writes the expected local marker and clears the old current
candidate pointer.

#### Expected Files

- `internal/plan/current.go`
- `internal/plan/current_test.go`
- `internal/runstate/state.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `.agents/skills/harness-land/SKILL.md`

#### Validation

- Tests cover archive-created archived pointers, land cleanup that clears the
  current candidate, and status output for both active-plan and idle-after-land
  worktrees.
- Manual or scripted dogfood proves a landed worktree no longer reports the
  old archived candidate as awaiting merge approval.

#### Execution Notes

Removed the implicit archived-plan fallback from current-plan detection,
extended `.local/harness/current-plan.json` to store `last_landed_plan_path`
plus `last_landed_at`, and taught `harness status` to return
`worktree_state: idle_after_land` when that marker exists without a current
plan. The land skill now explicitly tells the controller to rewrite the local
marker after merge so status stops claiming the archived candidate is still
awaiting merge approval.

#### Review Notes

Added regression coverage for no-pointer archived plans and idle-after-land
status output, then validated the end-to-end status behavior with `go test
./internal/plan ./internal/status ./internal/lifecycle`, `go test ./...`, and
the direct post-install `harness status` smoke check.

### Step 4: Differentiate archived handoff from true merge-waiting state

- Done: [x]

#### Objective

Teach `harness status` to treat `awaiting_merge_approval` as a coarse tracked
phase and surface a separate local handoff state that says whether the archived
candidate still needs publish work, is waiting on post-archive CI, or is ready
for actual merge approval.

#### Details

Reuse the existing `.local` publish and CI fields instead of inventing a new
tracked lifecycle. Update the CLI contract, README, and execute guidance so
archive is no longer described as the end of execution by itself. Status
output should become conservative by default: without local publish evidence,
an archived plan should be described as locally archived and still pending
publish handoff rather than already waiting for merge approval.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`
- `README.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/closeout-and-archive.md`
- `.agents/skills/harness-execute/references/publish-ci-sync.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`

#### Validation

- Tests cover archived status output for at least pending-publish and
  ready-for-merge-approval local handoff states.
- Durable docs and skills no longer imply that local archive alone means the
  controller can stop before commit/push/PR handoff.

#### Execution Notes

Added archived-plan `handoff_state` inference to `harness status` so the CLI
now distinguishes `pending_publish`, `waiting_post_archive_ci`, and
`ready_for_merge_approval` instead of treating every archived candidate as
already waiting for merge approval. Reused the existing local publish and CI
fields, surfaced PR URLs in status artifacts, and updated specs, README, and
execute guidance so archive is no longer described as the natural end of
execution by itself. After review follow-up, tightened the handoff logic so
fresh sync evidence is also required before merge-ready status, added missing
CI/sync regressions, and documented that reviewer subagents should use clean
Codex contexts instead of inheriting the controller transcript. Revision 3 then
tightened the execute-skill wording so resumed agents are told to ALWAYS use
`harness-execute` when `harness status` resolves an approved current plan,
clarified that post-archive `harness status` is rerun to confirm the local
handoff state for the same worktree, and made the reviewer-spawn guidance stick
to the fixed clean prompt only.

#### Review Notes

`review-004-delta` caught two legitimate gaps in the first handoff-state pass:
published candidates without CI evidence were still marked merge-ready, and
the `followup_required` branch lacked a regression test. `review-005-delta`
then caught the remaining correctness gap that missing sync evidence also had
to block merge-ready status. After tightening the state machine and coverage,
`review-006-delta` passed cleanly across correctness, tests, and
docs-consistency. The final reviewer rerun also used clean Codex reviewer
subagents without inherited controller context. The revision-3 docs follow-up
for resume guidance and fixed clean reviewer prompts then passed in
`review-007-delta` across docs-consistency and agent-ux.

## Validation Strategy

- Run `harness plan lint` while evolving the tracked plan and doc wording.
- Run targeted package tests for `./internal/status`, `./internal/lifecycle`,
  `./internal/plan`, and `./internal/runstate` while implementing the shared
  readiness, archived-handoff, and landed-state behavior.
- Run `go test ./...` before closeout.
- Dogfood the workflow with local-state fixtures or a temp worktree to verify
  archive preflight guidance and landed-worktree status behavior end to end.

## Risks

- Risk: Removing implicit archived-plan fallback could make some cold-start
  resume flows less convenient.
  - Mitigation: Preserve explicit archived pointers after `harness archive`,
    add regression tests for active-plan preference and idle-after-land flows,
    and document land cleanup clearly in the skill.
- Risk: A non-`NONE` follow-up rule may allow vague handoff text.
  - Mitigation: Keep the docs explicit that the section must contain actionable
    follow-up information for archive-known deferred items, and make status
    prompt the controller to fill that section before archive.
- Risk: TDD guidance could be over-applied to pure docs or mechanical
  refactors.
  - Mitigation: Limit the requirement to behavior-changing work and require a
    brief documented reason when TDD is skipped.
- Risk: Archived handoff state could look more authoritative than the local
  publish and CI evidence actually available in v0.1.
  - Mitigation: Keep the new handoff state conservative by default and fall
    back to pending-publish guidance when local publish evidence is absent.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-20-archive-readiness-landed-status-and-execute-tdd.md`
  passed after the revision-3 execute-skill wording updates.
- `go test ./...` passed after the comment-driven docs follow-up.
- `harness status` was rerun after reopen to confirm the revision-3 candidate
  was back in active execution and again before archive to confirm closeout was
  complete.

## Review Summary

- `review-001-full` requested changes for archive/reopen rollback safety,
  landed-state coverage, stale issue-ref wording in the plan schema, and the
  missing land-cleanup CLI path; those issues were fixed before the next round.
- `review-002-full` requested changes for the missing negative-path
  `land record` regression and stale active-plan workflow wording; both were
  fixed before rerunning review.
- `review-003-full` passed with zero blocking or non-blocking findings for the
  initial archive-readiness slice.
- After reopen, `review-004-delta` and `review-005-delta` caught archived
  handoff-state correctness and coverage gaps around missing CI and sync
  evidence before merge approval.
- `review-006-delta` passed with zero blocking or non-blocking findings after
  those archived-handoff fixes.
- `review-007-delta` passed with zero blocking or non-blocking findings after
  the execute-skill wording follow-up for ALWAYS resume guidance and fixed
  clean reviewer prompts.

## Archive Summary

- Archived At: 2026-03-20T23:30:31+08:00
- Revision: 3
- PR: NONE
- Ready: Acceptance criteria are satisfied, archive blockers still surface
  early, archived candidates distinguish local publish handoff from true
  merge-ready waiting, and the latest comment-driven execute-skill follow-up
  closed cleanly in `review-007-delta`.
- Merge Handoff: Commit the archive move, push the branch, open or update the
  PR, and only treat the candidate as truly waiting for merge approval once
  post-archive handoff evidence is complete.

## Outcome Summary

### Delivered

- Added shared archive-readiness evaluation so `harness status` surfaces
  closeout blockers early and `harness archive` refuses to write partial
  archive state.
- Added landed local-state recording with `harness land record`, explicit
  `idle_after_land` status output, and rollback-safe archive/reopen state
  transitions.
- Added archived-plan `handoff_state` output so `harness status` distinguishes
  `pending_publish`, `waiting_post_archive_ci`, `followup_required`, and true
  merge-ready waiting from the same coarse archived lifecycle.
- Updated the durable specs, README, and repo-local execute skills so archive
  is no longer described as the end of execution by itself, `harness-execute`
  now explicitly says to ALWAYS resume there when an approved current plan is
  present, and reviewer orchestration now documents fixed clean Codex reviewer
  prompts with `fork_context=false`.
- Updated execute guidance to require Red/Green/Refactor TDD by default for
  behavior-changing work, with documented exceptions only when TDD is
  genuinely impractical.

### Not Delivered

- First-class remote PR, publish, and CI capture commands from `#12`.
- A richer tracked `landed` lifecycle or stricter follow-up reference parsing
  beyond the local marker plus non-`NONE` contract shipped here.

### Follow-Up Issues

- `#12` tracks first-class remote PR, publish, and CI modeling.
