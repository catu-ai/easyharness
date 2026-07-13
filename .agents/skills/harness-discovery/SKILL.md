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

Investigate repository facts yourself before asking the human. Keep shared
context with the controller; delegate only bounded factual questions whose
answers can be integrated cleanly.

Ask a concise question only when a real human decision remains. Frame realistic
options when that makes the tradeoff clearer, recommend a direction when the
evidence supports one, and allow the human to revise the frame. Do not force
options, questions, or extra discovery after the direction is clear.

Keep the conversation focused on:

- desired outcome and success criteria
- authority, constraints, and non-goals
- material design or workflow choices
- rejected alternatives that a future agent might otherwise reopen

## Handoff

When direction is clear, summarize the accepted approach, repository facts that
shaped it, draft acceptance criteria, rough plan shape, and deferred scope.
Hand off to `harness-plan` only after the human confirms the direction.

Answer factual questions directly. If the request is already clear and
approved, use the appropriate execution skill instead of manufacturing a
discovery phase.
