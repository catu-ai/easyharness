---
name: harness-execute
description: Drive an approved tracked plan through implementation, validation, independent review, archive, publish, CI/sync evidence, and readiness to wait for merge approval.
---

# Harness Execute

## Outcome

Deliver the approved scope as a validated, independently reviewed, archived,
and published candidate whose durable state reaches
`execution/finalize/await_merge`.

Use this skill whenever `harness status` resolves an approved current plan that
has not landed. The controller remains in this skill during review;
`harness-reviewer` is only for reviewer subagents.

## Start

1. Run `harness status` and open its `plan_path`.
2. Confirm the plan is approved. If it is not, stop at the approval boundary.
3. Follow the current node, plan outcome, acceptance criteria, and returned next
   actions. Read only the reference needed for the current phase.

If the installed `harness` is unavailable or stale, use the repository's
documented setup path before relying on it.

## Execution Contract

- Implement the approved outcome and validate behavior in proportion to its
  risk. Tests are evidence, not a narrated ceremony.
- Keep durable plan notes concise and sufficient for another agent to resume;
  do not preserve a tool-by-tool transcript.
- Delegate bounded exploration, non-overlapping implementation, validation, or
  advisory work when useful. The controller integrates shared state and owns
  final judgment.
- A completed step creates no review debt. Start a step review only for a real
  intermediate risk boundary that should be frozen before later work.
- Run the mandatory independent finalize review after the complete candidate is
  committed. Follow [review-orchestration.md](references/review-orchestration.md).
- Fix blocking findings and extend coverage with a linked repair delta unless
  design, scope, or risk changed materially enough to require a new full root.
- Archive only after acceptance criteria, validation, durable summaries, and
  review coverage are complete. Follow
  [closeout-and-archive.md](references/closeout-and-archive.md).
- Publish the archived candidate and record current PR, CI, and sync evidence.
  Follow [publish-ci-sync.md](references/publish-ci-sync.md).
- Stop at `execution/finalize/await_merge` until the human explicitly approves
  merge.

## Node Guide

- `plan`: wait for or record explicit approval, then start execution.
- `execution/step-<n>/implement`: complete the current plan outcome and its
  validation; an intentionally started step review may temporarily bind this
  step.
- `execution/step-<n>/review`: wait for or aggregate the active review.
- `execution/finalize/review|fix|archive`: establish finalize coverage, repair
  findings, then close out and archive.
- `execution/finalize/publish`: publish and record fresh remote evidence.
- `execution/finalize/await_merge`: report readiness and wait for merge
  approval; repair remote drift before claiming the candidate is still ready.

## References

- [step-inner-loop.md](references/step-inner-loop.md): step outcomes and notes
- [review-orchestration.md](references/review-orchestration.md): mandatory
  finalize review and optional specialists
- [closeout-and-archive.md](references/closeout-and-archive.md): durable closeout
- [publish-ci-sync.md](references/publish-ci-sync.md): PR, CI, and sync evidence

Do not start from an unapproved plan, impersonate a reviewer, bypass a real
workflow blocker, or claim merge readiness from memory instead of current
repository and remote evidence.
