---
name: harness-discovery
description: Clarify a harness task before planning when the objective, boundaries, tradeoffs, success criteria, or workflow direction are genuinely unclear. Do not use for factual repo questions, status checks, or approved execution.
metadata:
    easyharness-managed: "true"
    easyharness-version: dev
---

# Harness Discovery

## Outcome

Converge on a plan-ready direction: the goal, constraints, non-goals,
acceptance criteria, important tradeoffs, and next workflow step should be clear
without relying on hidden chat context.

Discovery is conversation-only. Do not modify repository files or begin
implementation.

## Approach

Investigate repository and environment facts yourself. Facts are agent-owned;
goals, preferences, boundaries, and material tradeoffs are human-owned. Keep
shared context with the controller; delegate only bounded factual questions
whose answers can be integrated cleanly.

Actively identify the decision tree behind an unclear request. When two or more
reasonable plan-ready directions remain, or a plan would encode an unconfirmed
human preference that changes the goal, scope, acceptance criteria,
constraints, or consequential design, do not choose silently. Resolve the
highest-leverage parent decision before its dependent branches:

1. state the current read and the decision it exposes;
2. frame realistic options when they make the tradeoff clearer;
3. give the recommended answer and a concise reason;
4. ask one question, then wait for the human to choose, revise, or reject it.

Continue one decision at a time until shared understanding is sufficient for a
plan. Do not ask the human for inspectable facts, bundle several decisions into
one turn, or grill reversible implementation details. Do not manufacture
questions for factual requests or work whose outcome and boundaries are
already clear.

Keep the conversation focused on:

- desired outcome and success criteria
- authority, constraints, and non-goals
- material design or workflow choices
- rejected alternatives that a future agent might otherwise reopen

## Handoff

When direction is clear, summarize the accepted approach, repository facts that
shaped it, draft acceptance criteria, rough plan shape, and deferred scope.
Proceed directly to `harness-plan` only when that handoff no longer depends on
an unconfirmed human preference; do not add a filler discovery confirmation.
Plan approval remains the execution boundary, not a substitute for unresolved
discovery decisions.

Answer factual questions directly. If the request is already clear and
approved, use the appropriate execution skill instead of manufacturing a
discovery phase.
