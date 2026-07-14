---
name: harness-plan
description: Create or update the tracked plan that records an approved outcome, boundaries, acceptance criteria, review focus, and meaningful progress steps.
---

# Harness Plan

Use this skill once direction is clear enough to propose an executable plan.
The plan must let a future agent continue without chat history while leaving
implementation choices to that agent.

## Workflow

1. Start from `harness plan template` and choose an honest size.
2. Preserve the decisions and constraints settled during discovery.
3. State in-scope and out-of-scope boundaries plus observable acceptance
   criteria.
4. Record candidate-specific invariants or questions in `Review Focus`; the
   mandatory final reviewer receives them automatically.
5. Use the fewest meaningful outcome steps. A step contains only its title,
   `Done`, `Outcome`, `Covers`, and an optional concise `Check`.
6. Name deferred work explicitly and describe whole-plan validation.
7. Use supplements only for bulky durable approved context. Move normative
   repository behavior into formal code, tests, or specs before archive.
8. Reread the plan without chat context, then run
   `harness plan lint <plan-path>`.
9. Present the plan for explicit human approval. After approval, record it with
   `harness plan approve --by human` before execution starts.

## Boundaries

- Standard plans omit `workflow_profile`.
- Use `--lightweight` only when the human explicitly approves one bounded,
  low-risk `XXS` change. It still uses the tracked plan and steering gates.
- Goal-oriented authoring is deferred to `v0.7.0`; do not use or invent a
  `goal_oriented` profile in the current contract.
- If the initial estimate is `XXL`, ask whether to split the work before
  approval. If it remains coherent and approved, move spillover into deferred
  items rather than quietly expanding the slice.
- Do not predict files or prescribe implementation details merely to make the
  plan look complete.
- Do not add execution diaries, review notes, step-local acceptance sections,
  or routine tool narration to steps.
- Do not treat the original task request as approval of the written plan.

## Commands

```bash
harness plan template --help
harness plan lint --help
```

The plan is ready when lint passes, the human can approve or challenge it from
the document alone, and another agent can execute the intended outcome without
hidden context.
