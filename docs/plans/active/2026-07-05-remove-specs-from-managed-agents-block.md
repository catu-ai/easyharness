---
template_version: 0.2.0
created_at: "2026-07-05T11:01:01+08:00"
approved_at: "2026-07-05T11:02:17+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/280
size: XXS
workflow_profile: lightweight
---

# Remove Specs From Managed AGENTS Block

## Goal

Clarify the easyharness-managed `AGENTS.md` block by removing the generic
`docs/specs/` source-of-truth entry and proofreading the rest of the managed
block for small downstream-facing wording issues. The managed block should
describe only the repository surfaces easyharness can safely assume for
downstream users.

Downstream repositories may keep private operational specs or other
repo-specific guidance wherever they choose, but that belongs in user-owned
`AGENTS.md` content outside the easyharness-managed block. The generated block
must not imply that downstream repositories have a `docs/specs/` folder or
that easyharness owns the meaning of such repo-local specs.

## Scope

### In Scope

- Remove the `docs/specs/` source-of-truth bullet from the packaged managed
  block.
- Proofread the rest of the managed block and fix narrow editorial issues that
  are clearly owned by the generated easyharness text.
- Refresh this repository's materialized root `AGENTS.md` managed block from
  `assets/bootstrap/`.
- Keep the change to documentation/bootstrap wording only.

### Out of Scope

- Adding a repo config field, template variable, or customization point.
- Rewording the managed block to explain repo-local specs versus
  easyharness product contracts.
- Moving or changing easyharness dogfood specs in this repository.
- Editing downstream/user-owned `AGENTS.md` guidance outside the managed block,
  except for the automatic dogfood refresh of this repository's managed block.

## Acceptance Criteria

- [x] The generated managed block no longer mentions `docs/specs/` as a
      default harness source of truth.
- [x] The wording does not introduce a new claim about downstream repo-local
      spec folders or easyharness ownership of private specs.
- [x] `AGENTS.md` is refreshed from `assets/bootstrap/agents-managed-block.md`
      through `scripts/sync-bootstrap-assets`.
- [x] Bootstrap sync validation passes.
- [x] The managed workflow list uses correct sequential numbering.
- [x] The subagent guidance spells out "one subagent" instead of using the
      terse `1` shorthand.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Remove the assumed specs source from the managed block

- Done: [x]

#### Objective

Delete the `docs/specs/` bullet from the managed source-of-truth list and sync
the packaged bootstrap output into this repository's dogfood `AGENTS.md`.

#### Details

This qualifies for the lightweight path because it is an `XXS` documentation
and generated-bootstrap wording change with no runtime behavior, schema,
release-safety, security, or state-machine impact. The key constraint is to
remove the assumption entirely, not replace it with a broader explanation of
private repo specs.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Run `scripts/sync-bootstrap-assets --check`.
- Run `git diff --check -- assets/bootstrap/agents-managed-block.md AGENTS.md`.

#### Execution Notes

Removed the `docs/specs/` source-of-truth bullet from
`assets/bootstrap/agents-managed-block.md` and ran
`scripts/sync-bootstrap-assets` to refresh the root `AGENTS.md` managed block.
TDD was not applicable because this is a documentation/bootstrap wording
change with no behavior change. Validation passed with
`scripts/sync-bootstrap-assets --check` and
`git diff --check -- assets/bootstrap/agents-managed-block.md AGENTS.md`.

#### Review Notes

Step closeout delta review `review-001-delta` passed with 0 findings. The
`docs-consistency` reviewer confirmed that the managed block removes the
`docs/specs/` assumption from both packaged and materialized bootstrap text
without adding a replacement claim about downstream private specs.

## Validation Strategy

- Use the bootstrap sync script to materialize the source change and prove the
  dogfood output is in sync.
- Use whitespace diff checking for the touched markdown files.

## Risks

- Risk: The execution accidentally keeps the `docs/specs/` concept in the
  managed block under different wording.
  - Mitigation: Treat the accepted direction as deletion, not clarification;
    downstream repo-specific spec guidance remains outside the managed block.

## Validation Summary

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `git diff --check -- assets/bootstrap/agents-managed-block.md AGENTS.md`
- `git diff --check`
- `harness plan lint docs/plans/active/2026-07-05-remove-specs-from-managed-agents-block.md`
- After reopen revision 2, reran `scripts/sync-bootstrap-assets --check`,
  `git diff --check -- AGENTS.md assets/bootstrap/agents-managed-block.md`,
  and a targeted `rg` scan confirming the old `use \`1\`` wording and removed
  `docs/specs/` source-of-truth wording are absent from the managed block.

## Review Summary

- Step closeout delta review `review-001-delta` passed with 0 findings in the
  `docs-consistency` dimension.
- Finalize full review `review-002-full` passed with 0 findings across
  `docs-consistency` and `risk-scan`.
- Revision 2 finalize full review `review-003-full` passed with 0 findings
  across `docs-consistency` and `risk-scan`.

## Archive Summary

- Archived At: 2026-07-05T11:10:22+08:00
- Revision: 1
- Reopened: Revision 2 finalize-fix proofread updates corrected workflow
  numbering and clarified one-subagent wording in the managed block.
- PR: Pending publish after archive.
- Ready: The candidate is archive-ready after passing bootstrap sync,
  whitespace validation, plan lint, step review, the original finalize review,
  and the revision 2 finalize review.
- Merge Handoff: Publish should open a PR with a lightweight handoff memo
  before the candidate waits for merge approval.

## Outcome Summary

### Delivered

- Removed the generated managed-block `docs/specs/` source-of-truth entry from
  `assets/bootstrap/agents-managed-block.md`.
- Corrected the managed workflow numbering from duplicate `4` entries to
  sequential `1` through `6`.
- Reworded the subagent guidance from `use \`1\`` to `use one subagent`.
- Refreshed the root `AGENTS.md` managed block with
  `scripts/sync-bootstrap-assets`.
- Preserved downstream repo-specific spec guidance as out of scope for
  user-owned `AGENTS.md` content outside the easyharness-managed block.

### Not Delivered

- No repo config field, template variable, customization point, or new
  downstream-spec explanation was added.

### Follow-Up Issues

NONE
