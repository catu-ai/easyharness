# Review Orchestration

Every candidate receives an independent finalize review. The controller stays
in `harness-execute`; each assignment belongs to a reviewer subagent using
`harness-reviewer`.

## Coverage

A finalize `full` round has exactly one `integrated` assignment covering the
complete plan and candidate. Always assign `correctness`, `tests`, `risk-scan`,
and `docs-consistency`; add other relevant guidance without splitting those
dimensions into separate reviewers. Dimensions are reusable guidance fragments,
not reviewer topology.

Add a `specialist` only when the completed candidate has a concrete high-risk
surface that benefits from an independent adversarial pass, such as security,
concurrency, recovery, migration, persistence, deployment, irreversible side
effects, or acceptance-critical performance. Give it a bounded risk brief with
non-empty risk surfaces and invariants. Size and file count are not triggers.

Repository dimensions may refine built-ins. Plan-scoped guidance is additive:
it may extend a selected dimension or add a plan-local dimension, but it never
overrides the base reviewer contract or creates an assignment automatically.
Inspect available guidance with:

```bash
harness review dimensions list
```

Step review is optional. Starting one intentionally binds that step until its
blocking findings are resolved; routine completed steps create no review debt.

## Full and Repair Delta

Use `full` to establish whole-candidate coverage. After narrow review-driven
repairs, use a linked `delta` anchored at the prior round's captured
`reviewed_head_sha`. Target each prior finding exactly once for an explicit
resolution verdict.

Start a new full root only when a repair materially changes design, scope, or
risk enough to invalidate the earlier judgment. Non-blocking `minor` findings
do not require repair.

## Start a Round

Commit intended candidate changes and keep the worktree clean. Create a review
spec and run:

```bash
harness review start --spec <path>
```

Minimal finalize spec:

```json
{
  "kind": "full",
  "assignments": [
    {
      "slot": "integrated",
      "role": "integrated",
      "dimensions": ["correctness", "tests", "risk-scan", "docs-consistency"],
      "instructions": "Review the complete candidate for archive readiness."
    }
  ]
}
```

A specialist assignment additionally requires `risk_brief.risk_surfaces` and
`risk_brief.invariants`; `failure_modes` is optional.

A finalize repair delta requires:

```json
{
  "kind": "delta",
  "anchor_sha": "<prior-reviewed-head-sha>",
  "repair": {
    "round_id": "<direct-coverage-tip-round>",
    "finding_ids": ["<finding-id>"]
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

Use stable unique slots and non-empty instructions. `step` is optional and only
for an intentionally chosen step review. Do not invent workflow metadata; the
CLI owns round allocation, binding, candidate-head capture, artifacts, and
normalized assignments.

## Dispatch and Aggregate

For every assignment returned by `review start`:

1. Spawn a clean reviewer subagent.
2. Tell it to use `harness-reviewer` exactly.
3. Pass the returned round ID, plan path, reviewed HEAD, slot, role, dimensions
   and resolved guidance handoff, assignment instructions, risk brief,
   submission path, anchor and repair link, plus a concise change summary.
4. Wait for the reviewer-owned submission. Agent completion is not submission;
   verify the slot artifact and replace a reviewer that finished without one.

Before aggregation, ensure every expected assignment submitted and candidate
HEAD still equals the round's captured head. Then run:

```bash
harness review aggregate --round <round-id>
harness status
```

The controller must not submit or rewrite reviewer results. If blocking
findings remain, repair and validate them, then dispatch the relevant linked
delta assignments. Return specialist findings to the same risk role; include
integrated coverage when the repair affects broad control flow.

Reuse an earlier reviewer for a narrow same-role repair only when continuity is
more useful than a fresh view. Otherwise dispatch a clean reviewer. The newest
round handles and guidance are always authoritative.
