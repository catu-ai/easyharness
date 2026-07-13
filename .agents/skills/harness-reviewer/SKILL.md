---
name: harness-reviewer
description: Independently review one assignment in an existing harness round and submit structured findings. Use only in reviewer subagents, never in the controller.
metadata:
    easyharness-managed: "true"
    easyharness-version: dev
---

# Harness Reviewer

## Outcome

Inspect the assigned candidate independently, continue beyond the first issue,
and submit one evidence-backed result for the owned slot. Do not edit tracked
files, start or aggregate rounds, or orchestrate other reviewers.

## Inputs

Use the controller-provided round ID, review kind, plan path, reviewed HEAD,
slot, role, selected dimensions and guidance handoff, assignment instructions,
risk brief, submission path, anchor, repair link, and change summary. Report
missing or contradictory inputs instead of guessing.

Read the complete tracked plan and exactly one role overlay:

- `integrated`: [integrated-role.md](references/integrated-role.md)
- `specialist`: [specialist-role.md](references/specialist-role.md)

Fetch each selected guidance body when instructed:

```bash
harness review dimensions instructions <dimension-name>
```

Resolved dimension guidance is additive to this skill and the role contract.

## Review

Verify current HEAD matches the reviewed SHA. For `full`, inspect the complete
candidate. For `delta`, begin with the anchored change and referenced findings,
then follow related behavior far enough to decide whether the repair is sound
and preserves prior coverage.

Report only actionable defects introduced, exposed, or made relevant by the
candidate. Follow related unchanged code when necessary. Do not report unrelated
cleanup, style preference, or a risk explicitly and consistently deferred by
the approved plan.

Severities:

- `blocker`: correctness or safety failure that must stop progression
- `important`: meaningful defect that blocks approval of this round
- `minor`: non-blocking observation

Prefer no finding over a vague one. State what is wrong, why it matters, and
the narrowest useful repo-relative location. Security findings require a
concrete trust-boundary failure, not merely sensitive-looking code.

## Submission

Write the reviewer-owned submission payload and submit it:

```bash
harness review submit --round <round-id> --slot <slot> --by <reviewer-name> --input <path>
```

Canonical payload:

```json
{
  "summary": "Concise whole-assignment conclusion.",
  "findings": [
    {
      "severity": "important",
      "area": "correctness",
      "title": "Short finding title",
      "details": "Concrete defect and impact.",
      "locations": ["path/to/file.go#L123"]
    }
  ]
}
```

`findings` may be empty. `locations` are optional and support a repo-relative
path, `#L123`, or `#L1-L3` suffix. `area` names the actionable concern and need
not equal the slot or dimension.

For a linked repair, also include one `resolutions` entry for every targeted
prior finding this assignment owns:

```json
{
  "finding_id": "review-001-full/integrated/001",
  "status": "resolved",
  "details": "Evidence for the verdict."
}
```

Use `resolved` or `unresolved`; new defects still belong in `findings`. After a
successful submission, report the receipt to the controller and stop. If a
later narrow follow-up reuses this reviewer, treat every handle in the newest
assignment as authoritative.

The only harness commands available to this role are dimension instruction
reads and its own `review submit`. Never submit for another slot or on behalf of
the controller.
