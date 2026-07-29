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
- At `execution/coordinate`, treat the root subplans as one flat, partially
  ordered execution set. Use `harness status --plan <subplan-id-or-path>` for a
  focused child view, delegate runnable non-overlapping children when useful,
  and keep shared Git integration under the controller.
- A coordinated child advances through edits to its own ordered steps and
  concise Result. Do not run child-specific approve, execute-start, formal
  review, archive, publish, or land ceremonies.
- A completed step advances progress without a review boundary. Use ordinary
  bounded validation or advisory subagents for intermediate uncertainty.
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
- `execution/coordinate`: progress runnable subplans, resolve sibling
  dependencies, and integrate the shared candidate until every child is
  complete.
- `execution/step-<n>/implement`: complete the current plan outcome and its
  validation.
- `execution/finalize/review|fix|archive`: establish finalize coverage, repair
  findings, then close out and archive.
- `execution/finalize/publish`: publish and record fresh remote evidence.
- `execution/finalize/await_merge`: report readiness and wait for merge
  approval; repair remote drift before claiming the candidate is still ready.

## References

- [step-inner-loop.md](references/step-inner-loop.md): step outcomes and notes
- [review-orchestration.md](references/review-orchestration.md): mandatory
  integrated finalize review and repair deltas
- [closeout-and-archive.md](references/closeout-and-archive.md): durable closeout
- [publish-ci-sync.md](references/publish-ci-sync.md): PR, CI, and sync evidence

Do not start from an unapproved plan, impersonate a reviewer, bypass a real
workflow blocker, or claim merge readiness from memory instead of current
repository and remote evidence.
