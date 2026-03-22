# Review Discipline Postmortem

## Scope

This note analyzes why the controller agent skipped step-level review discipline
for the `2026-03-22-repo-level-smoke-and-review-workflow-tests` plan and then
stopped after implementation instead of autonomously continuing through
finalize review and archive closeout.

The question is whether the failure was primarily:

- instruction ambiguity
- skill gap
- tooling or permission mismatch
- controller judgment

## Summary

The primary cause was controller judgment. The controller had enough
information to continue the execution loop, but it chose to stop and ask the
human whether it should run review orchestration instead of proceeding
autonomously.

There was also some instruction ambiguity, but it was secondary. The repo docs
and harness skills describe review orchestration clearly enough to proceed once
the work becomes reviewable, yet they do not spell out a single hard trigger for
when step-level review must start versus when finalize review must start. That
left a seam the controller used as a reason to pause.

This was not a tooling or permission problem. `harness` was available, the plan
was active, `harness status` worked, and there was no command failure that
blocked review or archive progression.

## Evidence

The repo agreement already says agents should execute approved scope and avoid
making humans micromanage routine execution. See [AGENTS.md](/Users/yaozhang/.codex/worktrees/f978/superharness/AGENTS.md#L12-L18)
and [AGENTS.md](/Users/yaozhang/.codex/worktrees/f978/superharness/AGENTS.md#L52-L71).

The harness execute skill says the controller should stay in `harness-execute`
through the whole execution loop, keep going until the plan is archived and the
worktree reaches `execution/finalize/await_merge`, and not skip routine
execution by asking the human to micromanage. See [harness-execute/SKILL.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/SKILL.md#L13-L30)
and [harness-execute/SKILL.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/SKILL.md#L80-L99).

The review orchestration guide says that once review is in flight, the controller
should create the round, spawn reviewer subagents, wait for them, verify their
submissions, and only then aggregate. See [review-orchestration.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/references/review-orchestration.md#L3-L18)
and [review-orchestration.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/references/review-orchestration.md#L87-L113).

The step inner loop also says a finished slice that is ready for review should
run a delta review, and that the step should only be marked complete once the
objective and validation are genuinely satisfied. See [step-inner-loop.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/references/step-inner-loop.md#L7-L22).

The tracked plan for this slice had already been advanced into
`execution/finalize/review` when the controller paused. From that node, the
correct next action was still to continue into finalize review instead of
asking the human to choose whether routine review orchestration should happen.
That evidence is about the controller's stop-and-wait mistake only; it does not
endorse skipping step-level review discipline earlier in the branch. See
[2026-03-22-repo-level-smoke-and-review-workflow-tests.md](/Users/yaozhang/.codex/worktrees/f978/superharness/docs/plans/active/2026-03-22-repo-level-smoke-and-review-workflow-tests.md#L239-L246).

## Root Cause Analysis

### Primary cause: controller judgment

The controller had enough evidence to continue but chose to pause and ask the
human whether it should run finalize review. That is a workflow judgment error,
not a missing capability.

The controller should have treated these as automatic continuation signals:

- the tracked plan had already been approved
- the tracked plan had already been advanced into finalize review
- `harness status` had already moved the worktree into `execution/finalize/review`
- the skills explicitly say the controller should stay in `harness-execute`
  through review orchestration and archive closeout

Instead, the controller treated the final review step like a human-choice gate.
That contradicts the working agreement and the execute skill.

Separately, this postmortem should not be read as approving the earlier lack of
step-scoped delta review. That is a different workflow defect, and the tracked
tests/docs need to model the state-model review rules more faithfully.

### Secondary cause: instruction ambiguity

The docs describe the review flow well, but they do not state one explicit
sentence that says, "when the last tracked step is complete, automatically start
finalize review before asking for anything else." The controller could therefore
misread the system as allowing a pause after implementation.

The seam is narrow, but it is real:

- `step-inner-loop.md` says to run delta review "if the slice is ready for review"
- `review-orchestration.md` explains how to run review once it is active
- `harness-execute/SKILL.md` says to keep moving until archive and await merge

What is missing is a crisp rule tying those together into a single automatic
transition.

### Not a tooling or permission mismatch

There was no evidence of a blocked command path or a missing binary. The
controller had working `harness` access, could inspect status, and could have
continued into review orchestration and archive work.

## What Should Change

The repo instructions should make two things explicit:

1. Routine review and archive progression is controller-owned and automatic once
   the plan is approved.
2. Human confirmation is only needed for scope changes, blockers, or actual
   merge approval, not for routine step closeout.

Recommended edits:

- Tighten [AGENTS.md](/Users/yaozhang/.codex/worktrees/f978/superharness/AGENTS.md) so the review section says the controller must autonomously start step or finalize review when a slice becomes reviewable.
- Tighten [harness-execute/SKILL.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/SKILL.md) so `execution/finalize/review` is treated as a mandatory continuation state, not just a descriptive hint.
- Tighten [step-inner-loop.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/references/step-inner-loop.md) so "ready for review" explicitly means "run the review round now, before closing the step."
- Add one short rule to [review-orchestration.md](/Users/yaozhang/.codex/worktrees/f978/superharness/.agents/skills/harness-execute/references/review-orchestration.md) that the controller should not ask the human whether to start routine finalize review once the plan reaches closeout.

## Follow-Up Issue

See [GitHub issue #22](https://github.com/yzhang1918/superharness/issues/22)
for the concrete instruction changes to make this failure less likely in future
runs.
