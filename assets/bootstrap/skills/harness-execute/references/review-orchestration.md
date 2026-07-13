# Review Orchestration

The controller stays in `harness-execute`. Each reviewer assignment belongs to
a bounded subagent using `harness-reviewer`.

Finalize review is the ordinary formal gate. Steps are implementation,
validation, commit, and durable-note boundaries; completing one does not create
review debt. Start a step-bound review only when an intermediate artifact
crosses a concrete risk boundary that should be frozen before later work. Once
started, that optional review remains binding until its blocking findings are
resolved.

Starting review is controller-owned. Do not ask the human to micromanage an
ordinary finalize review after plan approval.

## Choose Reviewers at Pre-Review

Choose the actual reviewer topology from the completed candidate, acceptance
criteria, validation evidence, and residual risks immediately before review.
Plan-time risks and invariants are inputs, not a frozen agent count.

Every finalize `full` round has exactly one `integrated` assignment. The
integrated reviewer reads the complete plan and candidate, composes all assigned
standard guidance, and remains responsible for correctness, tests, risk, and
documentation even when a specialist is present.

Ordinary candidates add no specialist. Add a `specialist` only when the
candidate has a concrete high-risk surface that benefits from an independent
adversarial challenge, for example:

- process, concurrency, or lifecycle coordination
- failure, retry, idempotency, or recovery behavior
- security or another trust boundary
- schema, migration, persistence, or data-integrity behavior
- release, deployment, or version-skew risk
- irreversible or external side effects
- acceptance-critical performance or resource behavior

A specialist needs a bounded risk brief with non-empty risk surfaces and
invariants, plus relevant failure modes. Plan size, step count, file count, or
the mere availability of specialist guidance is not a trigger. Normally use at
most one specialist; use more only for multiple independent high-risk surfaces.
A specialist does not replace or shrink integrated coverage.

Both roles follow the base `harness-reviewer` submission, severity, evidence,
and no-edit contract. The reviewer skill adds the role-specific overlay after
the base contract:

- integrated: whole-candidate synthesis across every assigned guidance fragment
- specialist: bounded adversarial challenge of the concrete risk brief

## Full, Repair Delta, and Anchors

Use `full` to establish whole-candidate coverage before archive. Use a linked
repair `delta` after narrow review-driven changes. A repair delta starts from
the prior covered git head, references the prior finalize round, and extends
that coverage when it passes.

Do not rerun `full` merely because a full round found blocking issues or because
a later delta became the latest round. Promote a repair to a new full round only
when the repair materially changes candidate design, scope, or risk enough that
the prior whole-candidate judgment is no longer trustworthy. Record that reason.

`minor` findings are non-blocking. They remain visible and do not require a
repair round. If the controller chooses to fix one narrowly, a linked repair
delta is sufficient.

Every `delta` uses a real git commit anchor. For a repair delta, use the prior
round's captured `reviewed_head_sha`; do not invent a moving branch name or a
worktree-only anchor.

After reopen, narrow work may extend prior archived coverage with a repair
delta when the runtime reports a continuous base. Broad reopened work starts a
new full root.

## Pre-Review Scan

Before `harness review start`:

1. Run `harness status` and read the complete tracked plan.
2. Run the `Pre-Review` checklist in
   [controller-truth-surfaces.md](controller-truth-surfaces.md).
3. Ensure intended candidate changes are committed and the worktree is clean.
4. Decide `full` versus linked repair `delta` from the coverage boundary above.
5. Create one integrated assignment for a full finalize review.
6. Add only concrete-risk specialists, normally zero and normally at most one.
7. Inspect reusable guidance with:

   ```bash
   harness review dimensions list
   ```

8. Assign only guidance that fits the role. Built-in, repository, and additive
   plan-scoped dimensions are guidance fragments, not agents. A single
   assignment may consume several dimensions.

For goal-oriented work, consider `evidence-validity` when conclusions depend on
adaptive evidence, synthesis, rejected alternatives, residual uncertainty, or
follow-up handling. Do not add it mechanically. `hypothesis-challenge` remains
a checkpoint advisory action, not a formal review dimension.

Plan-scoped guidance augments built-in or repository guidance and may introduce
a plan-local name. It never overrides the base reviewer contract, never creates
an assignment automatically, and must still be selected explicitly.

## Review Spec

Create a spec and pass it to:

```bash
harness review start --spec <path>
```

Ordinary finalize example:

```json
{
  "kind": "full",
  "review_title": "Review the complete branch candidate before archive",
  "assignments": [
    {
      "slot": "integrated",
      "role": "integrated",
      "dimensions": ["correctness", "tests", "risk-scan", "docs-consistency"],
      "instructions": "Inspect the complete candidate under the common reviewer contract and all resolved guidance."
    }
  ]
}
```

Concrete-risk specialist example:

```json
{
  "kind": "full",
  "review_title": "Review the complete branch candidate before archive",
  "assignments": [
    {
      "slot": "integrated",
      "role": "integrated",
      "dimensions": ["correctness", "tests", "risk-scan", "docs-consistency"],
      "instructions": "Inspect the complete candidate under the common reviewer contract and all resolved guidance."
    },
    {
      "slot": "process-lifecycle",
      "role": "specialist",
      "dimensions": ["risk-scan"],
      "instructions": "Adversarially challenge the process-lifecycle risk brief.",
      "risk_brief": {
        "risk_surfaces": ["Helper lifecycle and status propagation"],
        "invariants": [
          "Known capture failure prevents workload side effects",
          "Runtime capture failure makes the run fail"
        ],
        "failure_modes": ["stale binary", "missing binary", "late helper exit"]
      }
    }
  ]
}
```

Field rules:

- `kind`: `full` or `delta`
- `assignments`: one or more unique stable slots; each declares `role`,
  non-empty `instructions`, and one or more selected `dimensions`
- `role`: `integrated` or `specialist`
- `risk_brief`: required for specialists, with non-empty `risk_surfaces` and
  `invariants`; `failure_modes` is optional
- `anchor_sha`: required for `delta`; use the prior covered head for a repair
- `review_title`: optional human-readable title
- `step`: optional 1-based binding for an intentionally chosen step review;
  omit it for ordinary finalize review
- `repair`: required for a finalize delta and omitted for full or step review;
  provide the direct coverage-tip `round_id` plus the `finding_ids` this round
  intends to resolve

Do not invent structural metadata such as `trigger` or `target`. The CLI owns
round allocation, binding, candidate-head capture, normalized assignments, and
review artifact locations.

Linked repair example:

```json
{
  "kind": "delta",
  "anchor_sha": "<prior-reviewed-head-sha>",
  "repair": {
    "round_id": "review-001-full",
    "finding_ids": ["review-001-full/integrated/001"]
  },
  "assignments": [
    {
      "slot": "integrated",
      "role": "integrated",
      "dimensions": ["correctness", "tests"],
      "instructions": "Verify the bounded repair and its interaction with the covered candidate."
    }
  ]
}
```

## Controller Flow

1. Start the round with `harness review start --spec <path>`.
2. Read the returned round ID, captured `reviewed_head_sha`, plan path,
   assignments, submission paths, and next actions.
3. Spawn one reviewer subagent per returned assignment with the fixed prompt
   below. Use a clean context for every new assignment.
4. Wait for mailbox updates and verify every expected slot has a valid
   reviewer-owned submission. If a reviewer finishes without submitting,
   dispatch a clean replacement for that slot.
5. Before aggregate, run the `Pre-Aggregate` checklist and confirm candidate
   HEAD still equals the captured review head.
6. Aggregate only when every assignment has submitted:

   ```bash
   harness review aggregate --round <round-id>
   ```

7. Run `harness status` after aggregate.
8. If blocking findings exist, repair and validate. Use a linked delta by
   default for a narrow repair, returning the relevant assignment(s): the same
   specialist for a specialist finding, and integrated as well when the repair
   changes broad control flow or cross-cutting behavior. Assign each targeted
   finding to exactly one reviewer submission for an explicit `resolved` or
   `unresolved` verdict.
9. Establish a new full root only for a material design, scope, or risk change.

The controller must not submit or rewrite reviewer results on a reviewer's
behalf.

## Fixed Reviewer Prompt

Pass only stable repo-facing handles returned by `review start`, not internal
review-control artifact paths:

```text
You are the reviewer for one harness reviewer assignment.

Use the harness-reviewer skill and follow it exactly.

Round ID: <round-id>
Review kind: <delta-or-full>
Active plan context: <Step N: title | Finalize: title>
Plan Path: <repo-facing-plan-path>
Review title: <review-title>
Revision: <candidate-revision-or-none>
Reviewed HEAD SHA: <reviewed-head-sha>
Slot: <slot>
Role: <integrated-or-specialist>
Assigned dimensions: <dimension-names-or-none>
Resolved guidance handoff: <commands or direct guidance returned for assignment>
Assignment instructions: <explicit assignment instructions>
Risk brief: <specialist-risk-brief-or-none>
Submission Path: <repo-facing-submission-path>
Anchor SHA: <commit-sha-or-none>
Repairs round: <prior-round-id-or-none>
Change summary: <bounded-change-summary>
```

Reviewer subagents submit through:

```bash
harness review submit --round <round-id> --slot <slot> --by <reviewer-name> --input <path>
```

Findings may include optional `locations` arrays with repo-relative anchors
such as `path/to/file.go`, `path/to/file.go#L123`, or
`path/to/file.go#L1-L3`.

## Narrow Same-Agent Follow-Up

For a narrow repair delta in materially the same assignment, continuity can be
useful. Reuse the earlier reviewer only when its submission was valid, the slot
and role remain the same, and the controller can provide a bounded change
summary tied to the earlier findings. Use a fresh reviewer when scope broadened,
the role or instructions changed, a new full review is needed, the earlier
submission was invalid, or an unbiased second look is more valuable.

When continuing an existing reviewer, send the same fixed prompt with the new
round handles and change summary. The newest round ID, reviewed head, submission
path, anchor, repair link, and guidance are authoritative.

## Codex Collaboration Adapter

Current Codex collaboration calls are:

- `spawn_agent({task_name, message, fork_turns})` creates a bounded subagent.
  Use `fork_turns: "none"` for a clean reviewer context; use `"all"` or a
  positive integer string only when the task intentionally needs recent parent
  context.
- `followup_task({target, message})` sends a follow-up and triggers a new turn
  when that existing agent is idle. If it is running, the message is delivered
  at a tool or message boundary.
- `send_message({target, message})` delivers context but does not trigger a new
  turn. Do not use it when an idle reviewer must act.
- `wait_agent({timeout_ms})` waits for the next mailbox update from any live
  agent, or for user steering. It does not accept agent IDs and does not mean
  that every reviewer finished.
- `list_agents({path_prefix})` shows live agent state. Omit `path_prefix` to
  inspect the full current agent tree.
- `interrupt_agent({target})` interrupts an active agent turn. It does not
  dispose of the agent; use it only when work should stop or be redirected.

There are no separate resume or close operations. Completed agents remain
available for a later `followup_task`; otherwise leave them idle. Agent
completion and reviewer submission are separate facts, so always verify the
slot artifact before aggregation.

Use this mailbox-oriented pattern:

1. spawn one clean agent per assignment
2. keep a pending set of assignment slots and agent targets
3. call `wait_agent({timeout_ms: ...})`
4. inspect updates and `list_agents` as needed
5. remove a slot only after its submission is valid
6. replace any finished agent that did not submit
7. aggregate only after the pending slot set is empty

Do not append controller reasoning, artifact tours, or unrelated instructions
to the fixed reviewer prompt.
