---
name: harness-plan
description: Create or update the tracked root plan and, for coordinated work, its agent-owned subplans.
---

# Harness Plan

Use this skill once direction is clear enough to propose an executable plan.
The plan must let a future agent continue without chat history while leaving
implementation choices to that agent.

## Workflow

1. Start from `harness plan template` and choose an honest size. Use
   `--coordinated` only when one approved candidate needs multiple concurrent
   workstreams.
2. Preserve the decisions and constraints settled during discovery.
3. State in-scope and out-of-scope boundaries plus observable acceptance
   criteria.
4. Record candidate-specific invariants or questions in `Review Focus`; the
   mandatory final reviewer receives them automatically.
5. For standard or lightweight work, use the fewest meaningful outcome steps.
   A step contains only its title, `Done`, `Outcome`, `Covers`, and an optional
   concise `Check`.
   - steps and acceptance criteria describe candidate outcomes that can be
     complete before finalize review
   - do not turn review, archive, publish, merge, or land milestones into plan
     steps or acceptance criteria; the harness lifecycle already owns them
   For coordinated work, the root omits `Work Breakdown`. Create flat subplans
   with `harness plan template --subplan` under the root's matching
   `supplements/<root-stem>/subplans/` directory. Each subplan uses the same
   compact ordered step shape and may name sibling dependencies.
6. Name deferred work explicitly and describe whole-plan validation.
7. Use supplements only for bulky durable approved context. Move normative
   repository behavior into formal code, tests, or specs before archive.
8. Reread the plan without chat context, then run
   `harness plan lint <plan-path>`.
9. Present the root plan for explicit human approval. After approval, record it
   with `harness plan approve --by human` before execution starts. Coordinated
   subplans are agent-owned decomposition within that approved root boundary;
   adding or refining them does not require separate approval.

## Boundaries

- Standard plans omit `workflow_profile`.
- Use `--lightweight` only when the human explicitly approves one bounded,
  low-risk `XXS` change. It still uses the tracked plan and steering gates.
- Use `--coordinated` only for one candidate-owning root whose flat subplans can
  progress concurrently. It does not allow multiple independent candidates in
  one worktree.
- Coordinated subplans do not have their own approve, execute-start, formal
  review, archive, publish, or land lifecycle. Do not nest subplans.
- If coordinated execution reveals a material change to the root goal, scope,
  acceptance criteria, constraints, or authority boundary, update the root and
  return to human steering just as for an ordinary plan.
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
