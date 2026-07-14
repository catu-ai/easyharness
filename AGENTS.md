# AGENTS.md

This document defines repo-specific guidance for how humans and Codex
collaborate in `easyharness`.

## Mission

Build `easyharness` as a thin, git-native, agent-first harness system that is
easier to understand and maintain than a scripts-heavy workflow. The project
name is `easyharness`; the CLI executable remains `harness`.

## Fast-Development Bias

`easyharness` is currently in a rapid development phase. When agents make
plans or implement changes, do not preserve compatibility for older command
shapes, file layouts, or intermediate contracts unless a human explicitly asks
for that work.

- prefer the clean target design over compatibility shims
- do not add migration paths, fallback reads, dual-write logic, or deprecation
  layers by default
- allow breaking changes when they simplify the system and remove obsolete
  behavior
- do not leave avoidable technical debt behind in the name of incremental
  compatibility

If a change would normally invite a compatibility bridge, stop and choose the
cleaner end-state unless the requested scope explicitly includes compatibility
or migration work.

## Trust-Based Core Development

When working on `easyharness` itself, treat the agent as a trusted collaborator
rather than an adversary to police.

- trust agents by default
- reduce agent cognitive load when designing prompts, commands, plans, and
  workflow surfaces
- prefer clearer workflow cues over pseudo-verification or anti-fraud ceremony
- do not propose heavier provenance, receipt, or identity mechanics for core
  workflow when they mainly create the appearance of verification without a
  real external trust surface
- if stronger trust guarantees are ever needed, treat that as explicit
  human-directed integration work rather than silently hardening the core
  harness by default

## Development Prerequisite

Before using repo-local skills that call `harness`, make sure the command is
available:

```bash
command -v harness
```

If not, bootstrap it from this repository:

```bash
scripts/install-dev-harness
```

If you change Go CLI code, rerun the installer before relying on the direct
`harness` command again.

## Bootstrap Asset Editing

This repository dogfoods the same repo resource assets that
`harness repo init` and the granular repo resource commands package for other
repositories.

- Edit `assets/bootstrap/` when changing the harness-managed skill pack or the
  managed `AGENTS.md` block content.
- Treat `.agents/skills/` in this repository as tracked materialized output from
  `assets/bootstrap/`, not as a hand-edited source tree.
- In this repository, `harness-*` skills are the easyharness-managed skill
  pack and belong in `assets/bootstrap/`.
- Other skill names are repo-owned local development skills. Keep them
  user-owned under `.agents/skills/` unless the repository explicitly decides
  to promote them into the distributed easyharness-managed pack later.
- After editing `assets/bootstrap/`, run `scripts/sync-bootstrap-assets` to
  refresh `.agents/skills/` and the managed block in this root `AGENTS.md`.
- Keep easyharness-specific guidance in this root `AGENTS.md` outside the
  managed markers below.

When triaging this repository's GitHub issues, use the repo-local
`issue-triage` skill. `docs/issue-triage.md` is only a thin discovery note;
the self-contained triage contract lives in the skill package.

The block below is the same harness-managed repository contract that
`harness repo instructions install` would install into another repository.
Keep easyharness-specific guidance outside the managed markers.

<!-- easyharness:begin version="dev" -->
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
`.agents/skills`; repository configuration may override these paths.

Start or resume with `harness status` and follow the current plan and returned
next actions:

- use `harness-discovery` when direction is unclear
- use `harness-plan` to create or revise the tracked plan
- use `harness-execute` after plan approval through `await_merge`
- use `harness-reviewer` only in reviewer subagents
- use `harness-land` only after explicit merge approval
<!-- easyharness:end -->

## Git and PR Rules

- main branch: `main`
- working branches: `codex/<topic>`
- commits: small and reviewable
- append `Co-authored-by: Codex <codex@openai.com>` unless the human requests
  otherwise
- when writing multi-line git or gh bodies, prefer heredocs so shell quoting
  does not eat backticks or other structured text
- when using the lightweight workflow, leave the agreed repo-visible breadcrumb
  in the PR body or other approved review surface before treating the candidate
  as ready to wait for merge approval
- default merge strategy: `Merge commit`
- do not rewrite shared history without explicit approval

If work creates durable deferred scope, create or update GitHub issues before
archive and record them in the plan.
