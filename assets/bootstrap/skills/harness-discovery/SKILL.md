---
name: harness-discovery
description: Use before planning or execution when the objective, boundaries, tradeoffs, success criteria, size, or workflow direction are unclear, or when archived work may need to reopen. Do not use for casual Q&A, simple repo orientation, status checks, already-approved execution, or implementation-ready work with clear scope.
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

Repository facts are agent-owned. Product intent, priorities, boundaries, and
approval are human-owned. Before asking the human, read the relevant context
and answer factual repository questions yourself, using bounded explorers when
that will sharpen the next human question.

Keep the harness posture collaborative, not adversarial. Borrow the useful
parts of brainstorming and grill-style questioning: read context first, ask one
focused question at a time, frame real choices plainly, and recommend a
direction when the evidence is strong enough. Do not turn discovery into a long
interrogation.

Prefer this question shape:

1. State the current read in one or two sentences.
2. Recommend a direction when there is enough signal.
3. Ask one plain question about the human-owned boundary, priority, or
   approval.

Avoid jargon-heavy labels, hedging, and loaded binary implementation choices.
Do not ask a question like:

> Do you want direct breaking schema convergence, or the minimal change of
> reordering fields and folding `remote_handoff`?

Prefer:

> I recommend making default `harness status` a short control-panel view and
> keeping full remote details for diagnostics. Are you comfortable changing the
> default output shape to get that clarity?

## Execution Contract

1. If the request is simple repo orientation, factual explanation, code lookup,
   status checking, already-approved execution, or clear implementation-ready
   work, answer or route it directly instead of entering discovery.
2. Read the most relevant repository context needed to ask sharper questions.
3. If the task is still fuzzy after repo-factual context is handled, ask one
   concise clarification question before doing broader discovery.
4. Use bounded repository exploration according to `Explorer Subagent
   Decision` above whenever local reading alone is not enough.
5. Discovery may alternate between human answers and further bounded
   exploration. Re-evaluate whether more exploration is needed after each
   clarification turn.
6. Ask exactly one high-leverage question per turn.
7. Use focused questioning to clarify:
   - purpose
   - constraints
   - non-goals
   - success criteria
   - workflow direction
8. When a decision benefits from framing, present 2-4 realistic options.
9. For each option, give:
   - a short label
   - one clear upside
   - one clear downside
   - when it fits
10. Recommend a direction when the tradeoffs are asymmetric.
11. Converge on a concrete approach, draft acceptance criteria, and state the
   next workflow step explicitly.
12. Give a concise discovery summary before handoff.
13. Hand off to `harness-plan` only after the human confirms the direction.

## Option Framing Pattern

When you offer options, keep them concise and decision-shaped. A good pattern
is:

1. `Option A`
   - upside
   - downside
   - best when ...
2. `Option B`
   - upside
   - downside
   - best when ...
3. `Option C`
   - upside
   - downside
   - best when ...

Then add a short recommendation and why.

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
- Do not offer weak filler options just to reach four.
- Do not turn option framing into long compare tables or verbose essays.
- Do not treat explorer use as mandatory when local reading is enough.
- Do not let explorer subagents own the shared context the controller still
  needs for later questioning.
- Do not treat factual explorer output as permission to skip controller
  synthesis or user clarification.
- Do not proceed until the human has enough clarity to approve the next step.
- Do not turn discovery into a hidden plan that only exists in chat.
