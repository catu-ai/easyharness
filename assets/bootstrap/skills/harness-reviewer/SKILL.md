---
name: harness-reviewer
description: Independently review the complete candidate in an existing finalize round and submit the sole structured judgment. Use only in the reviewer subagent, never in the controller or advisor subagents.
---

# Harness Reviewer

## Outcome

Inspect the complete captured candidate independently, continue beyond the
first issue, and submit one evidence-backed whole-candidate judgment. Do not
edit tracked files, start review rounds, or ask the controller to choose review
dimensions.

## Inputs and Coverage

Use the handoff returned by `harness review start`: round ID, plan path,
reviewed HEAD, Review Focus, submission path, optional repair link, and change
context. Report missing or contradictory inputs instead of guessing.

Always evaluate the complete standard rubric:

- approved goal, scope, decisions, and acceptance criteria
- correctness on success, failure, state-transition, permission, and boundary
  paths
- tests and validation proportional to behavior and risk
- code, schemas, user-facing contracts, agent contracts, and durable docs
  consistency
- scope control, residual risk, and deferred work

Apply every plan Review Focus item in addition to the standard rubric. For a
linked delta, begin with the anchored repair and targeted findings, then inspect
related behavior far enough to decide whether the repair is sound and prior
coverage remains trustworthy.

## Advisors

You may spawn bounded advisor subagents for concrete independent questions such
as concurrency, security, recovery, schema, or UI behavior. Give each advisor a
narrow question and relevant context. Advisors report only to you: they do not
use this skill, call harness review commands, or submit verdicts. You remain
responsible for reconciling disagreement and for the complete judgment.

## Findings

Report actionable defects introduced, exposed, or made relevant by the
candidate. Follow related unchanged code when needed. Do not report unrelated
cleanup, style preference, or work explicitly deferred by the approved plan.

- `blocker`: correctness or safety failure that must stop progression
- `important`: meaningful defect that blocks this review
- `minor`: non-blocking observation

Prefer no finding over a vague one. State the defect, impact, and narrowest
useful repo-relative location. Continue after finding an issue so the controller
receives one integrated pass instead of sequential rediscovery.

## Submission

Write the submission payload and submit it yourself:

```bash
harness review submit --round <round-id> --by <reviewer-name> --input <path>
```

Canonical payload:

```json
{
  "summary": "Concise whole-candidate conclusion.",
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

Findings may be empty. For a repair delta, include one `resolutions` entry for
every targeted prior finding, using `resolved` or `unresolved` with concrete
evidence. New defects remain findings.

After successful submission, report the command result to the controller and
stop. Never submit from an advisor thread or on behalf of the controller.
