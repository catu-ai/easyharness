---
template_version: 0.3.0
created_at: "2026-07-16T13:16:07+08:00"
approved_at: "2026-07-16T13:18:22+08:00"
source_type: direct_request
source_refs:
    - User discovery feedback, 2026-07-16
    - https://github.com/mattpocock/skills/tree/main/skills/productivity/grilling
size: XS
---

# Restore Bounded Grilling in Harness Discovery

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb repository-facing normative content into formal tracked locations before
archive, and record supplement absorption in Closeout. Lightweight plans should
normally avoid supplements. -->

## Goal

Restore active, bounded interviewing to the managed `harness-discovery` skill.
Discovery should investigate repository facts itself, surface material choices
that cannot be inferred from those facts, and resolve those choices with the
human one at a time before producing a plan-ready handoff.

Keep the RC4 skill compact. The target is not the former exhaustive prompt or
an unbounded interrogation; it is a small, explicit decision-ownership and
questioning loop inspired by Matt Pocock's `grilling` discipline and adapted to
easyharness's tracked-plan approval boundary.

### Decisions and Constraints

- Repository and environment facts are agent-owned to investigate; goals,
  preferences, boundaries, and material tradeoffs are human-owned decisions.
- Discovery actively identifies unresolved decision-tree branches instead of
  asking only questions that are already obvious blockers.
- Ask one highest-leverage decision question at a time, wait for the answer,
  and include the agent's recommended answer with concise reasoning.
- Stop when shared understanding is sufficient to state the goal, scope,
  non-goals, acceptance criteria, and consequential choices without silently
  assuming a human preference.
- Preserve direct answers for factual or already-clear requests and avoid
  questions about facts the agent can inspect itself, reversible implementation
  details, and filler confirmation.
- Discovery remains conversation-only. Durable decisions enter the tracked
  plan at handoff; plan approval remains the authorization to execute, not a
  substitute for unresolved discovery decisions.
- Treat `mattpocock/skills` as design input, not a runtime dependency or text to
  copy wholesale.

## Scope

### In Scope

- Tighten the canonical managed `harness-discovery` skill contract around
  active decision discovery, one-question pacing, recommendations, waiting for
  human answers, and bounded stopping conditions.
- Refresh the repository's materialized managed skill output from the bootstrap
  source.
- Validate both sides of the behavior boundary with representative ambiguous
  and already-clear/factual scenarios, plus repository bootstrap checks.

### Out of Scope

- Restoring the former long-form Socratic prompt or requiring every discovery
  session to exhaust every conceivable design branch.
- Adding separate `grill-me`, `grill-with-docs`, or domain-modeling skills.
- Allowing discovery to edit glossaries, ADRs, plans, code, or other repository
  files before handoff.
- Changing plan approval, execution, review, publish, merge, or land lifecycle
  semantics.
- Adding CLI state or commands for discovery conversations.

## Acceptance Criteria

- [x] The managed discovery contract explicitly distinguishes agent-owned facts
      from human-owned decisions and requires active surfacing of material
      choices rather than a blocking-question-only posture.
- [x] When unresolved choices would change the goal, scope, acceptance criteria,
      constraints, or consequential design, discovery asks one highest-leverage
      question at a time, gives a recommended answer, and waits for the human
      before proceeding down dependent branches.
- [x] The contract has a bounded stop rule: already-clear or factual requests
      are answered without manufactured questions, while a plan-ready handoff
      is allowed only when it no longer depends on an unconfirmed material
      preference.
- [x] Discovery remains conversation-only and hands the shared understanding to
      a tracked plan; bootstrap source/output synchronization and representative
      behavioral probes pass together with repository validation.

## Review Focus

- Check that the concise wording reliably changes behavior from passive
  clarification to active decision elicitation without recreating an
  indiscriminate or exhausting interview loop.
- Check the fact/decision boundary for both failure modes: asking the human for
  inspectable facts and silently deciding human-owned tradeoffs.
- Check that direct plan handoff cannot bypass an unresolved material decision,
  while clear implementation requests and factual questions remain fast.
- Check that the canonical bootstrap source and materialized dogfood skill are
  identical after synchronization.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Restore the bounded discovery interview loop

- Done: [x]
- Outcome: The compact managed discovery skill actively resolves human-owned
  decision branches one at a time, recommends answers, stops at shared
  understanding, and preserves the fast path for inspectable facts and clear
  work; canonical and materialized skill packages stay synchronized.
- Covers: All acceptance criteria.
- Check: Exercise fresh-agent ambiguous, factual, and implementation-ready
  scenarios; run bootstrap synchronization checks and repository validation.

## Validation Strategy

- Inspect the canonical/materialized diff for compactness and exact sync.
- Use fresh-agent behavioral probes that cover at least: multiple plausible
  product directions, an inspectable repository fact, and a request whose goal
  and boundaries are already explicit. Confirm that only the first case asks a
  decision-shaped question and that it includes a recommendation.
- Run `scripts/sync-bootstrap-assets --check`, focused bootstrap/smoke tests,
  plan lint, diff checks, and the repository's ordinary validation profile.

## Closeout

- Validation: Fresh-agent probes covered ambiguous, factual, and
  implementation-ready requests with the expected question/no-question split;
  bootstrap sync checks, smoke tests, plan lint, diff checks, and full
  `scripts/validate` passed.
- Review: `review-001-full` passed the complete candidate at `02ec34c` with no
  findings; the reviewer independently confirmed the behavioral probes and
  repository validation.
- Delivered: Restored a compact bounded-grilling loop in the managed discovery
  contract, including fact/decision ownership, active decision-tree discovery,
  one-question pacing, recommended answers, waiting for human choices, and a
  shared-understanding stop rule; refreshed the materialized dogfood skill.
- Not Delivered: No separate grilling/domain-modeling skill, repository writes
  during discovery, CLI discovery state, or lifecycle changes were added.
- Follow-Up Issues: None.
