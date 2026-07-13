---
name: harness-reviewer
description: Use when acting as a dedicated reviewer subagent for one reviewer assignment in an existing harness review round and you need to inspect the change, write structured findings, and submit them through `harness review submit`. This skill is only for reviewer subagents, not for the controller agent.
---

# Harness Reviewer

## Purpose

Use this skill only in reviewer subagents, including an existing reviewer that
the controller later triggers with `followup_task` for a narrow repair delta in
materially the same assignment.

The reviewer agent owns exactly one assignment slot in an existing review
round. It does not start rounds, aggregate rounds, orchestrate other reviewers,
or infer workflow `current_node` on the controller's behalf.

This is a strong-reviewer role, not a passive checklist runner. Read the full
active plan, use the repository tools needed to inspect the change properly,
and use the round-local submission artifact to keep enough review state that
you do not stop after the first one or two findings.

## Submission Contract

Submit exactly one structured payload with:

```bash
harness review submit --round <round-id> --slot <slot> --by <reviewer-name> --input <path>
```

Use this payload shape:

```json
{
  "summary": "Short review summary.",
  "findings": [
    {
      "severity": "important",
      "area": "correctness",
      "title": "Short finding title",
      "details": "Concrete explanation of the issue and why it matters.",
      "locations": [
        "path/to/file.go",
        "path/to/file.go#L123",
        "path/to/file.go#L1-L3"
      ]
    }
  ],
  "worklog": {
    "full_plan_read": true,
    "checked_areas": [
      "docs/plans/active/2026-04-09-example.md",
      "internal/review/service.go"
    ],
    "open_questions": [],
    "candidate_findings": [
      "Verify whether the delta anchor guidance matches the implementation."
    ]
  }
}
```

Rules:

- `summary` is required
- `findings` may be empty when the slot finds no issues
- `resolutions` is used only in a linked repair round; record `resolved` or
  `unresolved` with concrete details for each targeted prior finding this
  assignment owns, and omit it for ordinary full or step review
- `--by` is required and should name the reviewer thread that owns the slot
  submission
- extra top-level fields such as `worklog` are allowed and remain in the stored
  submission artifact, but aggregate still only uses canonical `summary` and
  `findings`
- `locations` is optional on each finding
- `area` names the actionable concern, such as `correctness`, `tests`,
  `docs-consistency`, or the specialist's risk surface; it does not need to
  equal the assignment slot or one selected guidance dimension
- valid severities are `blocker`, `important`, and `minor`
- when present, `locations` should use repo-relative paths and only these
  lightweight forms:
  - `path/to/file.go`
  - `path/to/file.go#L123`
  - `path/to/file.go#L1-L3`
- do not invent a separate scratchpad format; use the slot's owned
  `submission.json` as the progressive working artifact for the round

For a linked repair round, add explicit resolution verdicts alongside any new
findings:

```json
{
  "summary": "The bounded repair preserves the covered behavior.",
  "resolutions": [
    {
      "finding_id": "review-001-full/integrated/001",
      "status": "resolved",
      "details": "The repair now preserves the required invariant."
    }
  ],
  "findings": []
}
```

## Severity Guidance

Use severities like this:

- `blocker`
  - correctness, safety, or workflow issue that must be fixed before the
    reviewed slice can proceed
- `important`
  - meaningful issue that still blocks approval for the current round
- `minor`
  - non-blocking improvement or observation

Prefer no finding over a vague finding. If the issue is real, say exactly what
is wrong and why it matters to your assigned slot.

Report only actionable defects that the reviewed change introduced, exposed, or
made relevant. Do not report pre-existing unrelated cleanup or style
preferences as findings.

For security findings, describe the concrete exploitable risk, removed safety
check, or missing validation at a trust boundary. Do not flag shell, filesystem,
network, authentication, or other sensitive surfaces just because the change
touches them.

Use the smallest useful location that demonstrates the issue. When a finding
points at code outside the directly changed files, explain why the reviewed
change makes that unchanged code part of the defect.

If the current plan explicitly defers a risk and the implementation still
matches that deferral, you do not need to raise it again as a finding. Raise it
only if the change contradicts the deferral, expands the risk, or makes the
deferral stale.

## Role Contract

All reviewers use the common submission, severity, evidence, actionable-defect,
and no-tracked-edit rules in this skill. Then read and follow exactly one role
overlay named by the controller:

- `integrated`: read [integrated-role.md](references/integrated-role.md)
- `specialist`: read [specialist-role.md](references/specialist-role.md)

Do not infer a role from the slot name. If the controller omits or contradicts
the role, report the missing input instead of choosing one.

## Workflow

1. Read the controller's round ID, review kind, active-plan context, repo-facing
   `plan_path`, review title, revision context when present, reviewed HEAD SHA,
   slot, role, assigned dimensions, explicit assignment instructions, risk
   brief when present, reviewer-owned `submission_path`, anchor SHA and repaired
   round when present, and change summary.
2. If the controller did not give enough information to submit cleanly, report
   the missing input back to the controller instead of improvising.
3. Read the role overlay selected by the controller. Follow the explicit
   assignment instructions and every selected guidance handoff. If a handoff
   tells you to fetch catalog instructions, run each requested command, such
   as:

   ```bash
   harness review dimensions instructions <dimension-name>
   ```

   Treat the returned Markdown as authoritative additive guidance for this
   assignment. It augments rather than replaces the base and role contracts. If
   the handoff directly provides reviewer instructions instead of a catalog
   command, follow those instructions directly. If the handoff is unclear, or
   a requested command fails and leaves no usable instructions, report that
   back to the controller instead of guessing from the dimension name.
4. Open the controller-provided repo-facing `plan_path` and read the full plan
   before reviewing.
5. Locate the slot-owned progressive submission artifact using the
   controller-provided `submission_path`. That path is the reviewer-owned
   working artifact for the round.
6. Start updating that `submission.json` progressively while you review. Keep
   checked areas, open questions, candidate findings, or similar review
   progress in top-level worklog-style fields instead of a separate scratchpad.
7. Verify that the current candidate HEAD matches `Reviewed HEAD SHA` before
   relying on the review boundary. If it moved, report that to the controller
   rather than reviewing a different candidate silently.
8. For `delta` review, start from the anchored change since `Anchor SHA` and
   use the repaired-round context to verify the bounded findings and coverage
   extension. For each targeted prior finding this assignment owns, include
   exactly one explicit `resolved` or `unresolved` verdict with details.
   Treat that diff as the default starting lens, not a hard boundary. Begin
   with directly changed paths, then follow related logic, contracts, and
   runtime behavior when needed to decide whether the change is sound.
9. Continue inspection when related logic, plan intent, or contract meaning
   warrants it. If that deeper read uncovers additional real issues, report
   them in the same round with normal severities.
10. Do not early-stop just because you already found one or two issues. Use the
   progressive submission artifact to keep coverage and hypotheses visible
   while you continue checking the slot.
11. Submit the same `submission.json` with `harness review submit`.
   Include `--by <reviewer-name>` using a short stable name for your reviewer
   thread, such as `reviewer-integrated` or another clear slot-owned label.
12. Report the submission receipt back to the controller agent.
13. Stop once the receipt is reported. Completed reviewer agents may remain
    idle; there is no reviewer close operation.
14. If the controller later triggers you with `followup_task` for materially
    the same assignment, treat the newest round ID, review kind, review title,
    revision context, reviewed head, slot, role, selected dimensions,
    instruction handoff, anchor, repair link, and change summary as
    authoritative. Reuse prior context only to understand the bounded repair
    the controller asked you to verify.

## Do Not

- Do not call any harness command other than
  `harness review dimensions instructions <dimension-name>` and
  `harness review submit`.
- Do not spawn or orchestrate additional reviewers.
- Do not edit tracked files.
- Do not skip reading the full active plan, even for `delta` review.
- Do not keep exploring after a successful submission.
- Do not assume an older round ID, review kind, reviewed head, anchor SHA,
  repair link, revision context, role, or instructions still apply after a
  follow-up.
- Do not assume same-agent continuity carries across assignments, materially
  changed risk briefs, revisions, or a new full review.
