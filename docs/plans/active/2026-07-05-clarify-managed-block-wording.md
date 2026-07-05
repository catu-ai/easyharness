---
template_version: 0.2.0
created_at: "2026-07-05T08:22:30+08:00"
approved_at: "2026-07-05T08:23:10+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/280
size: XS
---

# Clarify Managed Block Wording

## Goal

Clarify the easyharness-managed `AGENTS.md` block so downstream repositories
can distinguish harness workflow guidance from repository-owned product or
operational documentation. The generated block should still stay compact and
agent-facing, but it should avoid implying that every downstream `docs/specs/`
tree is owned by easyharness or that easyharness controls all tracked
repository documentation and code language.

## Scope

### In Scope

- Update the managed block source in `assets/bootstrap/agents-managed-block.md`.
- Sync the materialized managed block in the root `AGENTS.md`.
- Keep the workflow list numbering coherent in the raw managed prompt.
- Clarify that the harness workflow applies to work coordinated through
  harness, not only work on easyharness itself.
- Clarify the relationship between easyharness-managed skills and repo-owned
  local skills.
- Triage GitHub issue #280 with the appropriate type/state labels and
  rationale comment.

### Out of Scope

- Adding new `.harness/config.yaml` fields.
- Creating a new long-form help topic for managed instruction customization.
- Changing harness command behavior, state transitions, review semantics, or
  bootstrap install mechanics.
- Rewriting the managed skill pack beyond sync output caused by the managed
  block source change.

## Acceptance Criteria

- [ ] The managed block no longer presents `docs/specs/` as an unconditional
      downstream easyharness product-contract location.
- [ ] The managed block no longer broadly requires all tracked docs and code in
      downstream repositories to be English unless repo-owned instructions add
      that rule outside the managed block.
- [ ] The managed workflow list has coherent numbering in raw markdown.
- [ ] The managed block wording distinguishes easyharness-managed workflow
      skills from repo-owned local skills.
- [ ] `scripts/sync-bootstrap-assets` has refreshed materialized output after
      the bootstrap source edit.
- [ ] Focused validation passes for bootstrap sync/install/status behavior.
- [ ] Issue #280 is triaged with a short rationale comment.

## Deferred Items

- Consider a future `harness help repo instructions` topic only if more
  downstream customization examples show that a compact managed block pointer
  is not enough.

## Work Breakdown

### Step 1: Tighten the managed block source

- Done: [ ]

#### Objective

Revise the managed block source to remove downstream ambiguity while keeping
the always-loaded prompt compact.

#### Details

Keep the change targeted to wording. Do not introduce a compatibility layer,
new config shape, or additional bootstrap command behavior. Preserve the
existing source-of-truth model: edit `assets/bootstrap/`, then sync generated
outputs.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Inspect the diff for source/materialized consistency and no unrelated managed
  skill churn.

#### Execution Notes

Updated `assets/bootstrap/agents-managed-block.md` to narrow the language
policy to harness workflow artifacts, clarify the default `docs/specs/`
meaning, distinguish easyharness-managed `harness-*` skills from repo-owned
skills, rephrase the workflow as work coordinated through harness, fix the raw
workflow numbering, and narrow resumed-work guidance to approved or in-progress
harness execution. Ran `scripts/sync-bootstrap-assets` so the root `AGENTS.md`
managed block matches the bootstrap source. TDD was not applicable because this
step is a documentation/prompt wording change.

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Validate and triage the issue

- Done: [ ]

#### Objective

Validate the managed block change and record the issue triage decision on
GitHub.

#### Details

Use the repo-local issue triage rules. The issue should receive the appropriate
GitHub type label and open backlog state or milestone; leave a concise
rationale comment explaining the decision.

#### Expected Files

- `docs/plans/active/2026-07-05-clarify-managed-block-wording.md`

#### Validation

- Run focused Go tests for bootstrap sync/install/status behavior.
- Run `harness repo init --dry-run` to confirm managed outputs are current.
- Run `harness plan lint` on this plan.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Use bootstrap sync as the source/materialized-output check.
- Use focused Go tests covering bootstrap sync, install, and status drift
  behavior.
- Use `harness repo init --dry-run` as an end-to-end check that the running
  binary sees managed assets as current.
- Use harness review orchestration for step closeout and finalize review.

## Risks

- Risk: The managed block becomes too vague and stops giving downstream agents
  enough useful defaults.
  - Mitigation: Keep resolved harness roots and command pointers explicit while
    narrowing only the misleading ownership language.
- Risk: Sync output accidentally changes unrelated managed skills.
  - Mitigation: Review the diff after `scripts/sync-bootstrap-assets` and keep
    the source edit limited to `agents-managed-block.md`.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
