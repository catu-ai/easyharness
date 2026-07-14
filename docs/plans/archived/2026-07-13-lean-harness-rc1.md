---
template_version: 0.3.0
created_at: "2026-07-13T23:30:01+08:00"
approved_at: "2026-07-13T23:37:38+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/293
    - https://developers.openai.com/api/docs/guides/prompt-guidance-gpt-5p6
size: XL
---

# Lean easyharness workflow and v0.6.0-rc.1 release

## Goal

Make easyharness materially lighter for current Codex while retaining the
boundaries that give humans real steering and candidates real reliability.
Plans should preserve decisions, scope, acceptance, and visible progress
without prescribing implementation. Execution should culminate in one
mandatory independent whole-candidate review rather than controller-managed
dimension, specialist, step-review, and aggregate ceremony.

Ship the result as `v0.6.0-rc.1` through the existing GitHub prerelease and
Homebrew channels in the same change, then use real feedback before adding
more workflow machinery.

### Decisions and Constraints

- Final independent review is mandatory for every standard plan. Change size,
  apparent narrowness, or controller confidence cannot waive it.
- Steps remain human-visible execution boundaries. They are not review units or
  implementation recipes; a plan with no natural intermediate boundary may use
  one step.
- A step carries only a title, `Done`, outcome, covered acceptance criteria,
  and an optional concise check. Discovery decisions and rejected alternatives
  belong in a plan-level decisions section rather than step details.
- Finalize has one formal integrated reviewer with a complete fixed rubric.
  Plan-specific review focus is attached automatically, not selected through
  dimension orchestration.
- The integrated reviewer may spawn bounded advisor subagents and remains the
  sole owner of the review judgment and harness submission. Advisors are not
  harness assignments, slots, or aggregate participants.
- Bounded exploration, implementation, validation, review, and nested
  delegation are preauthorized by the managed repository agreement. Humans may
  still restrict delegation explicitly, and the owning agent remains
  responsible for scope, integration, and shared-worktree coordination.
- A narrow review-driven repair closes through a linked delta. Another full
  review is required only when the repair materially changes design, scope, or
  risk.
- Preserve the current-step lifecycle and dashboard progress concept. Do not
  rewrite the entire state model merely to remove step-review ceremony.
- This RC intentionally allows breaking changes and removes obsolete workflow
  surfaces without compatibility shims.
- `goal_oriented` remains deferred to `v0.7.0`.

## Scope

### In Scope

- Compress the harness-managed AGENTS block, skills, and execution references
  around outcomes, authority boundaries, stopping conditions, evidence, and
  concise state-driven cues. Remove repeated tool narration, approval prompts,
  copied command schemas, defensive ceremony, and goal-oriented preview text.
- Replace the plan template and schema with a compact structure that preserves
  goal, decisions and constraints, scope in/out, acceptance criteria, review
  focus, outcome-based steps, validation, deferred work, and closeout.
- Keep `Step N / M` as the primary human-visible progress signal and expose
  acceptance completion as a derived `X / Y` signal where status or dashboard
  progress is shown. Update durable plan state only at meaningful step or
  evidence boundaries, not after routine tool calls.
- Remove formal step-review nodes and debt from the standard lifecycle while
  retaining current-step execution, pause/resume, archive, publish,
  invalidation, merge approval, and land boundaries.
- Replace controller-authored review specs and dimension/specialist selection
  with one mandatory integrated finalize reviewer using a fixed complete
  rubric plus automatically included plan review focus.
- Make reviewer-owned advisors ordinary nested subagents. Remove formal
  specialist assignments, advisor aggregation, and reviewer progressive
  worklog ceremony.
- Make a valid integrated reviewer submission complete the round and update
  review coverage without a separate controller aggregate action. Preserve the
  reviewed git boundary, actionable findings, full-root plus linked-delta
  coverage, archive rejection for uncovered product changes, and narrow
  plan-only closeout allowance.
- Remove obsolete dimension catalog/configuration/CLI/UI surfaces rather than
  retaining compatibility reads or hidden controller choices. Ensure the fixed
  integrated rubric covers correctness, acceptance, success/failure/state and
  permission behavior, tests and evidence, code/schema/docs/agent contracts,
  scope, residual risk, and deferred work.
- Synchronize bootstrap assets and update normative specs, help, examples,
  generated contracts, focused tests, lifecycle/review E2E coverage, and UI
  presentation for the simplified model.
- Consolidate the `v0.6.0` issue set into #293: absorb the useful evidence/help
  expectations from #288, #291, and #292; record custom reviewer configuration
  #289 as not planned; close superseded issues with rationale; keep
  goal-oriented work under `v0.7.0`.
- Bump `VERSION` from `0.5.2` to `0.6.0-rc.1` in this implementation PR, run
  release validation, and after explicit merge approval verify the GitHub
  prerelease, release assets, Homebrew formula update, and Homebrew install.

### Out of Scope

- Goal-oriented execution, checkpoints, scoring, or UI; that work remains in
  `v0.7.0`.
- Model selection, custom reviewer model configuration, automatic risk scoring,
  or a permanent telemetry subsystem.
- Replacing current-step state with acceptance-only state or redesigning the
  whole dashboard/workbench information architecture.
- Preserving old review specs, dimension catalogs, formal specialist artifacts,
  step-review rounds, plan templates, or their command shapes.
- Weakening plan approval, candidate git-boundary checks, CI/sync evidence,
  merge approval, reopen invalidation, or post-merge release verification.
- Adding migration, deprecation, dual-read, fallback, or dual-write behavior for
  removed pre-RC contracts.

## Acceptance Criteria

- [x] The distributed managed instruction pack is substantially smaller, with
      a recorded before/after byte or token comparison targeting at least a 50%
      reduction across the managed AGENTS block and harness-managed skills and
      references, without removing authority, evidence, safety, or completion
      boundaries.
- [x] Managed subagent guidance preauthorizes bounded exploration,
      implementation, validation, review, advisor use, and nested delegation
      without per-run human approval while preserving owner responsibility and
      explicit human restriction.
- [x] The canonical standard plan structure contains the agreed durable
      decisions, scope, acceptance, review focus, progress, validation,
      deferred, and closeout content without required file predictions,
      implementation details, execution notes, or review notes.
- [x] Every step is an outcome-based progress boundary with `Done`, covered
      acceptance criteria, and optional concise validation; current status and
      dashboard surfaces present `Step N / M` and derived acceptance progress
      without requiring frequent plan writes.
- [x] Standard lifecycle and UI progression no longer create or require formal
      step-review nodes, while current-step execution and later finalize,
      archive, publish, await-merge, reopen, and land boundaries remain sound.
- [x] Finalize cannot complete without exactly one independent integrated
      reviewer submission for the complete candidate, and the reviewer always
      receives the fixed complete rubric plus the plan's review focus without
      controller dimension selection.
- [x] Reviewer subagents can and are instructed to create bounded advisor
      subagents when useful; advisors report only to the reviewer, create no
      harness slots or submissions, and do not reduce the reviewer's complete
      judgment responsibility.
- [x] Controller-facing review dimensions, formal specialists, review-spec
      assignment orchestration, progressive reviewer worklogs, and separate
      aggregate ceremony are removed from commands, prompts, schemas, status,
      UI, and normative contracts.
- [x] Full review coverage remains tied to the captured candidate git head;
      narrow linked repair deltas resolve findings and extend coverage without
      another full review, while materially broad repairs require a new full
      root and uncovered product changes still block archive.
- [x] Representative before/after dogfood traces show the ordinary plan,
      execute, review, repair, archive, publish, and land path using fewer
      controller decisions and review rounds without weakening the mandatory
      gates.
- [x] Bootstrap synchronization, focused Go tests, lifecycle/review E2E tests,
      schema and documentation validation, UI tests/build, `git diff --check`,
      `scripts/validate`, and `scripts/validate-release` pass.
- [x] Issue #293 is the sole open `v0.6.0` delivery issue for this slice; #288,
      #289, #291, and #292 carry clear consolidation/not-planned rationale and
      no goal-oriented issue is pulled out of `v0.7.0`.
- [x] `VERSION` is `0.6.0-rc.1`, the merge handoff requires explicit human
      approval, and land verification covers tag `v0.6.0-rc.1`, GitHub
      prerelease assets, the updated default Homebrew formula, and an installed
      Homebrew binary reporting the RC version.

## Review Focus

- Look for any path that can archive or publish without the mandatory complete
  integrated review and continuous reviewed-head coverage.
- Verify that removed prompts or lifecycle nodes do not erase human approval,
  permission, external evidence, invalidation, or merge boundaries.
- Challenge whether the fixed rubric and automatic plan focus actually cover
  concerns previously supplied by dimensions, including plan-specific ones.
- Check that advisor delegation is technically usable and that responsibility
  remains unambiguous when advisors are absent, fail, or disagree.
- Verify progress is useful without creating frequent write or commentary
  requirements, and that release/Homebrew behavior matches RC semantics.

## Deferred Items

- Use RC dogfood and user feedback to decide whether any remaining internal
  review artifact structure should be removed before stable `v0.6.0`.
- Goal-oriented workflow development remains grouped under `v0.7.0`.
- Add model-specific prompting or reviewer configuration only in response to a
  measured regression that the fixed integrated rubric cannot address.

## Work Breakdown

### Step 1: Simplify plans and managed instructions

- Done: [x]
- Outcome: Deliver the compact plan contract and outcome-driven managed prompt pack while preserving the decisions and boundaries in this plan.
- Covers: Managed prompt reduction, delegation authorization, compact plan structure, and outcome-based step criteria.
- Check: Record prompt-size comparison and prove template, lint, status parsing, bootstrap sync, and representative plan authoring behavior.

### Step 2: Collapse review into one accountable reviewer

- Done: [x]
- Outcome: Deliver mandatory integrated finalize review, automatic review focus, reviewer-owned advisors, and direct submission closeout.
- Covers: Mandatory review, advisor ownership, orchestration removal, and continuous full-plus-delta coverage criteria.
- Check: Exercise clean review, findings, advisor handoff, narrow repair delta, broad repair reset, changed-HEAD rejection, and uncovered-change archive rejection.

### Step 3: Preserve simple human-visible progress

- Done: [x]
- Outcome: Deliver step and acceptance progress without formal step-review lifecycle or frequent agent-authored status churn.
- Covers: Lifecycle and UI progress criteria while retaining current state and all post-execution steering gates.
- Check: Exercise multi-step progress, acceptance counts, resume, finalize, reopen, and dashboard rendering with no step-review nodes.

### Step 4: Consolidate the milestone and prepare the RC release

- Done: [x]
- Outcome: Leave one delivery issue and a validated `v0.6.0-rc.1` candidate ready for explicit merge approval and automated GitHub/Homebrew publication.
- Covers: Dogfood, issue consolidation, validation, VERSION, and release handoff criteria; the human approved including VERSION in this PR.
- Check: Run representative traces and the release-ready validation profile, then record the exact post-merge GitHub and Homebrew checks.

## Validation Strategy

- Validate each outcome at its natural boundary, but run no formal step review.
- Reinstall the development harness after Go CLI changes before relying on the
  direct `harness` command.
- Use focused contract and behavior tests while changing each subsystem, then
  run the full repository and release validation profiles.
- Compare representative old and new controller traces for an ordinary clean
  candidate and a candidate with one narrow review repair.
- The Orchard timestamp dogfood that motivated this slice required 11 formal
  rounds: two step-review pairs, repeated finalize full reviews, and separate
  dimension submissions and aggregate actions. The new real-binary E2E trace
  reaches archive with one full round and one integrated submission when clean;
  a narrow finding requires exactly one additional linked delta and submission.
  Neither path asks the controller to select dimensions, start specialists, or
  aggregate reviewer output.
- Run one mandatory final integrated full reviewer. The reviewer may create
  bounded advisors and must submit the complete judgment itself.
- Close any narrow blocking repair with a linked delta; rerun full review only
  if the repair materially broadens candidate design, scope, or risk.

## Closeout

- Archived At: 2026-07-14T09:43:55+08:00
- Revision: 1
- Validation: Managed instruction sources fell from 77,314 to 21,159 bytes (72.6%); final `scripts/validate-release`, full Go, focused race, real-binary lifecycle E2E, UI build, and diff checks pass.
- Review: One integrated full review plus reviewer-owned linked deltas resolved round immutability, progress derivation, documentation consistency, and the complete post-archive reviewed-head boundary; the reviewer personally submitted every judgment, nested advisors proved usable, and `review-005-delta` passes with no unresolved blockers.
- Delivered: Compact plans and prompts, preauthorized bounded/nested delegation, outcome and acceptance progress, one mandatory integrated reviewer with automatic plan focus and optional advisors, direct submission, full-plus-delta coverage, simplified CLI/state/contracts/UI, issue consolidation, and `VERSION=0.6.0-rc.1`.
- Not Delivered: RC-feedback-driven private artifact cleanup, goal-oriented execution, model/reasoning configuration, risk scoring, and permanent telemetry remain outside this candidate.
- Follow-Up Issues: [#293](https://github.com/catu-ai/easyharness/issues/293) tracks RC publication, Homebrew verification, feedback, and the stable `v0.6.0` decision; goal-oriented work remains under `v0.7.0`.
- PR: Publish `codex/lean-harness-rc1` as the ready RC implementation PR after the archive commit and record its URL in harness publish evidence.
- Ready: All 13 acceptance criteria and four outcome steps are complete, final release validation passes, and integrated full-plus-delta coverage reaches the final candidate.
- Merge Handoff: Wait for explicit human merge approval after current PR CI and base sync pass; after merge, verify tag `v0.6.0-rc.1`, GitHub prerelease assets, the default Homebrew formula update, and an installed Homebrew binary reporting the RC version.

