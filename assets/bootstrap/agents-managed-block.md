## Harness Working Agreement

Humans steer goals, scope, and approval boundaries. Agents execute the approved
work and choose the implementation path.

- Keep approved scope in a git-tracked plan.
- Keep durable outcomes and behavior changes in tracked code or docs.
- Keep disposable trajectory, review artifacts, and evidence artifacts under
  the local runtime root reported by `harness repo config get
  paths.local_runtime`.
- Prefer repository state and external evidence over chat memory.
- Keep tracked docs and code in English.

The tracked plan wins if it conflicts with a workflow skill. Use `harness help`
or command `--help` when product behavior or syntax is unclear.

## Workflow and Authority

The ordinary workflow is discovery, plan, explicit plan approval, execution,
archive and publish, explicit merge approval, then land.

- A task request or newly written plan is not execution approval. Record human
  approval with `harness plan approve --by human` before execution.
- Once a plan is approved, the controller may implement, validate, review,
  archive, publish, and follow CI/sync evidence without routine confirmation.
- Stop for real blockers, material scope changes, authority expansion, and
  merge approval.
- Use `harness reopen --mode finalize-fix|new-step` when an archived candidate
  is invalidated.

Use `workflow_profile: lightweight` only when the human explicitly approves it
for one bounded, low-risk `XXS` change. It does not remove plan approval,
review, evidence, or merge approval.

Use `workflow_profile: coordinated` when one approved candidate needs multiple
flat agent-owned subplans. The root is the only approval and candidate
lifecycle owner; subplans may progress concurrently but do not approve, start,
review, archive, publish, or land independently. Use
`harness status --plan <subplan-id-or-path>` for a focused child view.

## Delegation

Repository and harness skill instructions authorize bounded subagents without a
separate per-run prompt. This includes explorers, parallel non-overlapping
implementation, validation, reviewers, advisors, and nested delegation by those
agents. A human may narrow or prohibit delegation explicitly.

Delegate concrete outcomes with clear ownership. Keep shared-context work local,
and parallelize only independent or non-overlapping work. The controller owns
integration and final workflow judgment.

## Review

Every candidate requires an independent finalize review before archive. Use
one integrated reviewer for whole-candidate coverage. The reviewer receives the
fixed standard rubric plus the plan's Review Focus automatically and may spawn
bounded advisor subagents for deeper investigation. Advisors report to the
reviewer and do not own harness submissions.

Reviewer subagents own their submissions through `harness review submit`; the
controller must not submit on their behalf. A narrow review repair normally
closes with a linked delta. Run a new full review only when the repair changes
design, scope, or risk enough to invalidate prior coverage.

## Source of Truth and Start Points

Active plans default to `docs/plans/active/`, archived plans to
`docs/plans/archived/`, runtime state to `.local/harness/`, and local skills to
`.agents/skills/`; repository configuration may override these paths.

Start or resume with `harness status` and follow the current plan and returned
next actions:

- use `harness-discovery` when direction is unclear
- use `harness-plan` to create or revise the tracked plan
- use `harness-execute` after plan approval through `await_merge`
- use `harness-reviewer` only in reviewer subagents
- use `harness-land` only after explicit merge approval
