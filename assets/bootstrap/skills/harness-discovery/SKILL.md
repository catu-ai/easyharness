---
name: harness-discovery
description: Run interactive, Socratic discovery before planning or execution when the objective, boundaries, tradeoffs, success criteria, size, or workflow direction are unclear, or when archived work may need to reopen. Do not use for casual Q&A, simple repo orientation, status checks, or already-approved execution.
---

# Harness Discovery

## Overview

Run discovery before implementation when the task needs real clarification.
Discovery is conversation-only. It should reduce ambiguity, surface tradeoffs,
and end with a clear next workflow step.

Discovery is size-independent. Use it to learn whether work is tiny, large, or
not work at all. Do not use discovery for ordinary repository facts, code
lookup, status checks, or simple explanations unless the human is deciding what
work to do next.

## Inputs

- the human's objective or problem statement
- relevant plans, specs, or design context from the repository
- current `harness status` output when the repository already has an active
  plan and local state

## Explorer Subagent Decision

Use explorer subagents on demand, not by default.

- Keep user-supplied core context and other shared repository context with the
  controller whenever later questions may depend on the details.
- Stay local when the controller can answer the next question from the shared
  context it already needs to hold.
- Use one explorer subagent when one bounded repository question or hypothesis
  needs checking.
- Use multiple explorer subagents in parallel only when multiple bounded
  hypotheses or questions are genuinely independent.
- Do not split one shared context bundle across multiple explorer subagents
  just to get summaries back.
- Explorer subagents should return factual findings for the bounded question
  only. They do not choose the next user question, recommend the workflow
  direction, or replace controller judgment.

## Questioning Style

Repository facts and documented project intent are agent-owned to investigate.
Humans have final say over goals, priorities, boundaries, and approvals when
they are ambiguous, contested, or not already settled in the repository. Before
asking the human, read the relevant context and answer factual repository
questions yourself, using bounded explorers when that will sharpen the next
human question.

Use a Socratic posture to clarify the work: test assumptions, name tensions,
and ask focused questions when they expose a real decision. Challenge the
human's framing when repository evidence points another way, but keep the
challenge tied to a concrete choice instead of debate. Read context first, ask
one focused question at a time, frame real choices plainly, recommend a
direction when the evidence is strong enough, and stop when there is enough
clarity to summarize and hand off.

Prefer option-shaped questions:

1. State the current read in one or two sentences.
2. Present 2-4 realistic options, even when the decision is small.
3. Put the recommendation under the option the agent prefers, with a short
   reason.
4. Ask the human to choose, edit, or reject the options.

Do not force a new question into every discovery turn. If the human asks for
details about an option, asks a side question, or needs a factual
explanation before deciding, answer directly. When a new human decision is
needed, ask only one question, and make it the highest-leverage question for
the current moment.

## Turn Exit Discipline

End each discovery turn in exactly one of these shapes:

1. `Factual answer`
   - use when the human asked for repository facts, code orientation, status,
     or an explanation and no workflow decision is pending
   - answer directly and do not append a workflow question just to keep
     discovery moving
2. `Decision question`
   - use when a real human choice is still blocking the next workflow step
   - ask one focused question and make clear what decision it resolves
3. `Plan-ready handoff`
   - use when repository facts and human intent are clear enough to plan
   - summarize the accepted direction, draft acceptance criteria, sketch the
     rough plan shape, and ask whether to hand off to `harness-plan`

If none of these shapes fits, gather the next missing repository fact before
responding. Do not ask a filler question when the right next move is to state
that discovery has converged.

Avoid jargon-heavy labels, hedging, and loaded binary implementation choices.
Do not ask a loaded binary question like:

> Do you want the broad breaking rewrite, or the minimal low-risk patch?

Instead, use the option framing pattern below. Keep labels neutral, name the
real tradeoff, and put the recommendation under the option the agent prefers:

1. `Option A`
   - upside: the main goal is addressed directly
   - downside: the change may be broader
   - best when: the clean target shape matters most
   - recommendation: I recommend this when repo context supports the broader
     change.
2. `Option B`
   - upside: the change is smaller
   - downside: the original confusion may only be reduced, not removed
   - best when: limiting scope matters most

Which option should I plan around: A, B, or a revision of one of these?

## Execution Contract

1. If the request is simple repo orientation, factual explanation, code lookup,
   status checking, or already-approved execution, answer directly and do not
   expand the turn into discovery.
2. Read the most relevant repository context needed to ask sharper questions.
3. If the task is still fuzzy after repo-factual context is handled, ask one
   concise clarification question before doing broader discovery.
4. Use bounded repository exploration according to `Explorer Subagent
   Decision` above whenever local reading alone is not enough.
5. Discovery may alternate between human answers and further bounded
   exploration. Re-evaluate whether more exploration is needed after each
   clarification turn.
6. Choose the turn exit deliberately: factual answer, decision question, or
   plan-ready handoff. Do not ask a new question merely to satisfy the
   discovery rhythm.
7. Use Socratic, focused questioning to clarify:
   - purpose
   - constraints
   - non-goals
   - success criteria
   - workflow direction
8. When a decision benefits from framing, present 2-4 realistic options.
9. Preserve the same option set in the final ask; do not narrow, rename, or
   replace framed options after presenting them.
10. For each option, give:
   - a short label
   - one clear upside
   - one clear downside
   - when it fits
11. Recommend a direction when the tradeoffs are asymmetric.
12. When no real question remains, converge on a concrete approach, draft
    acceptance criteria, sketch the rough plan shape, and state the next
    workflow step explicitly.
13. Give a concise discovery summary before handoff.
14. Hand off to `harness-plan` only after the human confirms the direction.

## Option Framing Pattern

When you offer options, keep them concise and decision-shaped. A good pattern
is:

1. `Option A`
   - upside
   - downside
   - best when ...
   - recommendation: I recommend this when ...
2. `Option B`
   - upside
   - downside
   - best when ...
3. `Option C`
   - upside
   - downside
   - best when ...

Include the recommendation under the option the agent prefers. If no option is
clearly better, say that briefly after the list instead of forcing a weak
recommendation.

Keep option integrity through the final question. If you present options A, B,
and C, the final ask should still offer A, B, and C, plus room to edit or
reject them. Do not collapse a three-option frame into a new two-way question,
rename the options at the end, or introduce a new choice that was not part of
the framed set.

A good final ask is:

> Which option should I plan around: A, B, C, or a revision of one of these?

An invalid final ask after presenting A, B, and C is:

> Should I do the broad version or the small version?

That drops one option and changes the decision surface after the human has
already read the framed tradeoffs.

## Output

Discovery should end with a concise conversation summary containing:

- the problem statement
- discovered repository facts that shaped the decision
- key constraints and non-goals
- the accepted direction
- rejected alternatives with short rationale
- draft acceptance criteria
- the rough plan shape
- the next workflow step

## Guardrails

- Do not implement code in this skill.
- Do not write or modify repository files during discovery.
- Do not ask bundled multi-question prompts; keep one question per turn.
- Do not ask a question when discovery has already converged enough for a
  plan-ready summary and rough plan outline.
- Do not offer weak filler options just to reach four.
- Do not present one option set and end with a different or narrower one.
- Do not turn option framing into long compare tables or verbose essays.
- Do not treat explorer use as mandatory when local reading is enough.
- Do not let explorer subagents own the shared context the controller still
  needs for later questioning.
- Do not treat factual explorer output as permission to skip controller
  synthesis or user clarification.
- Do not proceed until the human has enough clarity to approve the next step.
- Do not turn discovery into a hidden plan that only exists in chat.
