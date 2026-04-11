---
name: harness-reviewer
description: Use when acting as a dedicated reviewer subagent for one assigned harness review slot in an existing review round and you need to inspect the change, write structured findings, and submit them through `harness review submit`. This skill is only for reviewer subagents, not for the controller agent.
metadata:
    easyharness-managed: "true"
    easyharness-version: dev
---

# Harness Reviewer

## Purpose

Use this skill only in reviewer subagents, including a reviewer subagent that
the controller later resumes for the same slot within the same tracked step
review boundary or for the same finalize review title in the same revision.

The reviewer agent owns exactly one review slot in an existing review round. It
does not start rounds, aggregate rounds, orchestrate other reviewers, or infer
workflow `current_node` on the controller's behalf.

This is a strong-reviewer role, not a passive checklist runner. Read the full
active plan, use the repository tools needed to inspect the change properly,
and use the round-local submission artifact to keep enough review state that
you do not stop after the first one or two findings.

## Submission Contract

Submit exactly one structured payload with:

```bash
harness review submit --round <round-id> --slot <slot> --input <path>
```

Use this payload shape:

```json
{
  "summary": "Short review summary.",
  "findings": [
    {
      "severity": "important",
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
- extra top-level fields such as `worklog` are allowed and remain in the stored
  submission artifact, but aggregate still only uses canonical `summary` and
  `findings`
- `locations` is optional on each finding
- valid severities are `blocker`, `important`, and `minor`
- when present, `locations` should use repo-relative paths and only these
  lightweight forms:
  - `path/to/file.go`
  - `path/to/file.go#L123`
  - `path/to/file.go#L1-L3`
- do not invent a separate scratchpad format; use the slot's owned
  `submission.json` as the progressive working artifact for the round

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

If the current plan explicitly defers a risk and the implementation still
matches that deferral, you do not need to raise it again as a finding. Raise it
only if the change contradicts the deferral, expands the risk, or makes the
deferral stale.

## Workflow

1. Read the controller's round ID, review kind, active-plan context, review
   title, revision context when present, slot, assigned instructions, anchor
   SHA when present, and change summary.
2. If the controller did not give enough information to submit cleanly, report
   the missing input back to the controller instead of improvising.
3. Read `.local/harness/current-plan.json`, open the active tracked plan, and
   read the full plan before reviewing.
4. Locate the slot-owned progressive submission artifact at
   `.local/harness/plans/<plan-stem>/reviews/<round-id>/submissions/<slot>/submission.json`.
   If needed, inspect the round manifest to confirm the exact path.
5. Start updating that `submission.json` progressively while you review. Keep
   checked areas, open questions, candidate findings, or similar review
   progress in top-level worklog-style fields instead of a separate scratchpad.
6. For `delta` review, start from the anchored change since `Anchor SHA`.
   Treat that diff as the default starting lens, not a hard boundary.
7. Continue inspection when related logic, plan intent, or contract meaning
   warrants it. If that deeper read uncovers additional real issues, report
   them in the same round with normal severities.
8. Do not early-stop just because you already found one or two issues. Use the
   progressive submission artifact to keep coverage and hypotheses visible
   while you continue checking the slot.
9. Submit the same `submission.json` with `harness review submit`.
10. Report the submission receipt back to the controller agent.
11. Stop once the receipt is reported. The controller agent is responsible for
    closing reviewer subagents after verifying the successful submission.
12. If the controller later resumes you for the same slot within the same
    tracked step review boundary or for the same finalize review title in the
    same revision, treat the newest round ID, review kind, review title,
    revision context, slot, instructions, anchor SHA, and change summary as
    authoritative for that new assignment. Reuse your prior context only to
    understand the bounded follow-up the controller asked you to verify.

## Do Not

- Do not call any harness command other than `harness review submit`.
- Do not edit tracked files.
- Do not skip reading the full active plan, even for `delta` review.
- Do not keep exploring after a successful submission.
- Do not assume an older round ID, review kind, anchor SHA, revision context,
  or instructions still apply after a resume.
- Do not assume a resume carries across tracked steps or from step review into
  finalize review.
