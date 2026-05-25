---
template_version: 0.2.0
created_at: "2026-05-25T22:23:40+08:00"
approved_at: "2026-05-25T22:25:15+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/pull/217
    - https://github.com/catu-ai/easyharness/pull/215
    - https://github.com/yzhang1918/HEJI/pull/107
    - https://github.com/yzhang1918/HEJI/pull/106
size: S
---

# Standardize PR Body Handoff Guidance

## Goal

Update the distributed easyharness controller guidance so PR bodies become
readable merge memos instead of duplicated execution logs.

The end state should make the PR body answer three human-facing questions:
what changed, why the branch is mergeable without a human rereviewing the diff,
and what handoff notes still matter. Detailed commands, review trajectory,
acceptance criteria, repair history, and full validation records should stay in
the tracked plan and harness evidence artifacts.

## Scope

### In Scope

- Define the preferred PR body structure as:
  - `What Changed`
  - `Confidence`
  - `Handoff`
- Describe `What Changed` as outcome-first and story-shaped rather than a file
  list, command log, commit-log rewrite, or copied plan summary.
- Describe `Confidence` as a compact merge of self-review and validation:
  three to five high-signal bullets that pair each checked risk surface with
  its result.
- Clarify that raw validation command lists, full review-round logs, and
  step-by-step execution history belong in the tracked plan or evidence
  artifacts, not in the PR body.
- Clarify the boundary between PR body, tracked plan, and harness evidence for
  standard and lightweight workflows.
- Update the managed bootstrap skill guidance under `assets/bootstrap/` and
  refresh the materialized `.agents/skills/` output.
- Update any normative spec wording that currently mentions PR-body
  breadcrumbs if it needs to point at the new merge-memo shape.

### Out of Scope

- Adding a `harness publish` command or automating PR body creation.
- Changing publish, CI, sync, review, archive, land, or state-transition
  semantics.
- Rewriting historical PR bodies.
- Rewriting archived plans except for this plan's normal execution closeout.
- Creating repository-specific PR templates for HEJI, missless, or any
  downstream repository.
- Requiring humans to review code diffs after harness finalize review.

## Acceptance Criteria

- [x] Managed controller guidance tells agents to use a `What Changed`,
      `Confidence`, and optional `Handoff` PR body shape during publish
      handoff.
- [x] `What Changed` guidance explains that the section should lead with the
      outcome, use readable short sentences, and avoid file-shaped or
      command-shaped summaries.
- [x] `Confidence` guidance explains that self-review and validation are
      combined and condensed into risk-surface-plus-result bullets, with raw
      command lists left to the tracked plan or evidence artifacts.
- [x] Guidance clearly distinguishes what belongs in the PR body from what
      belongs in the plan and what belongs in harness evidence.
- [x] Lightweight repo-visible breadcrumb guidance remains intact but points at
      the same readable merge-memo principles.
- [x] Bootstrap assets and materialized skill outputs are synchronized.
- [x] Validation confirms plan lint, bootstrap sync check, and targeted wording
      checks pass.

## Deferred Items

- Automated PR body generation or linting.
- Downstream repository-specific templates.

## Work Breakdown

### Step 1: Define the PR body guidance contract

- Done: [x]

#### Objective

Write the reusable guidance for what easyharness expects a PR body to contain
and what it should leave to the tracked plan or evidence artifacts.

#### Details

The guidance should encode the discovered direction:

- PR body is a human merge memo, not a review request or execution log.
- `What Changed` explains what is now true after the PR. It should be
  outcome-first, story-shaped, and readable; length is secondary to clarity.
- `Confidence` combines self-review and validation into concise bullets. Each
  bullet should name a risk surface and the evidence/result that supports
  merge confidence.
- `Handoff` is optional and only carries merge-time, release-time, follow-up,
  non-goal, known-gap, or deferred-work notes that matter after reading the PR.
- The tracked plan remains the full audit trail for scope, acceptance criteria,
  commands, review rounds, repairs, validation details, and outcome history.
- Harness evidence remains the durable record for publish, CI, and sync facts.

The guidance may include a small illustrative example if that makes the
standard easier for future agents to apply, but it should avoid a long template
that agents copy mechanically.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/references/closeout-and-archive.md`
- `assets/bootstrap/skills/harness-execute/references/publish-ci-sync.md`
- `assets/bootstrap/agents-managed-block.md`
- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`

#### Validation

- The edited source guidance consistently uses the `What Changed`,
  `Confidence`, and `Handoff` terms.
- Targeted search confirms the older generic "PR body note" or
  "repo-visible breadcrumb" wording is either still correct or explicitly tied
  to the new merge-memo standard.

#### Execution Notes

Added the reusable PR body handoff guidance to the managed
`harness-execute` publish/CI/sync reference. The new guidance defines the PR
body as a readable merge memo with `What Changed`, `Confidence`, and optional
`Handoff`, and it draws the boundary between PR body, tracked plan, and
harness evidence. Updated lightweight breadcrumb wording in the managed block
and normative specs so PR body breadcrumbs point at the same merge-memo shape.
Validation used targeted wording search, `git diff --check`, and plan lint.

#### Review Notes

Step-closeout delta review `review-001-delta` passed with no findings. The
`docs_consistency` slot confirmed that managed bootstrap source and normative
specs consistently frame PR bodies as readable merge memos while preserving
lightweight breadcrumb and publish/CI/sync evidence semantics. The `agent_ux`
slot confirmed that future controllers can distinguish `What Changed`,
`Confidence`, and optional `Handoff` without being steered into rigid templates
or command dumps.

### Step 2: Sync outputs and validate the guidance

- Done: [x]

#### Objective

Refresh generated bootstrap outputs, verify the managed guidance is coherent in
both source and materialized locations, and leave the plan ready for normal
harness review and archive.

#### Details

Run the repository's bootstrap sync after editing `assets/bootstrap/`. Then
verify that `.agents/skills/` and the managed block are synchronized and that
the new PR-body guidance does not conflict with publish/CI/sync evidence
semantics or lightweight breadcrumb requirements.

If execution discovers that the normative spec wording does not need changes,
record that explicitly in execution notes rather than forcing a spec edit.

#### Expected Files

- `.agents/skills/harness-execute/references/closeout-and-archive.md`
- `.agents/skills/harness-execute/references/publish-ci-sync.md`
- `AGENTS.md`
- `docs/plans/active/2026-05-25-standardize-pr-body-handoff-guidance.md`

#### Validation

- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-25-standardize-pr-body-handoff-guidance.md`
- targeted `rg` checks for `What Changed`, `Confidence`, `Handoff`,
  `PR body`, and `repo-visible breadcrumb` across managed source and
  materialized outputs
- `git diff --check`

#### Execution Notes

Ran `scripts/sync-bootstrap-assets`, which refreshed the materialized root
`AGENTS.md` managed block and `.agents/skills/harness-execute` reference files
from `assets/bootstrap/`. Validation confirmed bootstrap outputs are in sync,
the tracked plan lints, targeted wording checks find the new merge-memo
guidance in both source and materialized outputs, and diff hygiene is clean.
No normative spec edit was skipped; the lightweight breadcrumb references in
`cli-contract`, `state-model`, and `plan-schema` were updated in Step 1.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only synchronized generated bootstrap outputs
from the reviewed source guidance and reran the planned consistency checks.
The guidance contract itself passed `review-001-delta` in Step 1.

## Validation Strategy

This is a guidance-only change, so validation should focus on plan integrity,
bootstrap synchronization, wording consistency, and diff hygiene. No Go,
frontend, or end-to-end runtime tests are expected unless execution changes
code or generated contracts.

## Risks

- Risk: The PR body standard could become a rigid template that produces
  formulaic text instead of clearer handoff notes.
  - Mitigation: Phrase `What Changed` as a readability principle and permit
    paragraphs or bullets according to the shape of the change.
- Risk: Condensing validation could hide important merge risks.
  - Mitigation: Require `Confidence` bullets to pair each checked risk surface
    with a result, and keep full validation details in the tracked plan and
    evidence artifacts.
- Risk: Updating only the skill source could leave materialized bootstrap
  output stale.
  - Mitigation: Run bootstrap sync and verify with
    `scripts/sync-bootstrap-assets --check`.

## Validation Summary

- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-25-standardize-pr-body-handoff-guidance.md`
- Targeted wording checks for `What Changed`, `Confidence`, `Handoff`,
  `PR body`, `repo-visible breadcrumb`, `merge memo`, and `validation command`
  across managed source, materialized outputs, root guidance, and specs.
- `git diff --check`
- Repair validation after finalize review: plan lint, bootstrap sync check,
  diff hygiene, and deferred/follow-up scan all passed.

## Review Summary

- Step-closeout delta review `review-001-delta` passed with no findings across
  `docs_consistency` and `agent_ux`.
- Finalize review `review-002-full` found one archive-readiness blocker: real
  deferred items still had `Follow-Up Issues: NONE`.
- The blocker was repaired by adding explicit follow-up handoff bullets for
  possible future PR body generation/linting and downstream repository
  templates.
- Finalize repair review `review-003-full` passed with no findings across
  `archive_readiness` and `docs_consistency`.

## Archive Summary

- Archived At: 2026-05-25T22:36:14+08:00
- Revision: 1
- PR: pending publish after archive; no PR URL exists yet.
- Ready: Managed source guidance, materialized bootstrap output, root managed
  block, and normative specs agree on the readable PR body merge-memo
  standard, and the final repair review passed cleanly.
- Merge Handoff: Commit the archive move, push branch
  `codex/pr-body-handoff-guidance`, open or update the PR with a `What
  Changed` / `Confidence` / optional `Handoff` body, then record publish, CI,
  and sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

- Added PR body handoff guidance to the managed `harness-execute`
  publish/CI/sync reference.
- Defined the preferred PR body shape as `What Changed`, `Confidence`, and
  optional `Handoff`.
- Clarified that PR bodies are readable merge memos for human merge approval,
  while tracked plans and harness evidence hold the full audit trail.
- Updated lightweight breadcrumb wording in managed guidance and normative
  specs to point at readable PR body merge memos.
- Synchronized materialized `.agents/skills` output and the root managed
  `AGENTS.md` block from `assets/bootstrap`.

### Not Delivered

- No automated PR body generation or linting was added.
- No downstream repository-specific PR template was created.

### Follow-Up Issues

- Future work may add automated PR body generation or linting if repeated
  manual application of the merge-memo guidance proves inconsistent.
- Downstream repositories may add local PR templates when their domain needs
  more specific handoff fields than the generic easyharness guidance provides.
