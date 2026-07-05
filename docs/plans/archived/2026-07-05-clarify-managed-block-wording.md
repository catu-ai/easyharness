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

- [x] The managed block no longer presents `docs/specs/` as an unconditional
      downstream easyharness product-contract location.
- [x] The managed block no longer broadly requires all tracked docs and code in
      downstream repositories to be English unless repo-owned instructions add
      that rule outside the managed block.
- [x] The managed workflow list has coherent numbering in raw markdown.
- [x] The managed block wording distinguishes easyharness-managed workflow
      skills from repo-owned local skills.
- [x] `scripts/sync-bootstrap-assets` has refreshed materialized output after
      the bootstrap source edit.
- [x] Focused validation passes for bootstrap sync/install/status behavior.
- [x] Issue #280 is triaged with a short rationale comment.

## Deferred Items

- Consider a future `harness help repo instructions` topic only if more
  downstream customization examples show that a compact managed block pointer
  is not enough.

## Work Breakdown

### Step 1: Tighten the managed block source

- Done: [x]

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

Revision 2 addressed PR feedback that the managed block must not assume user
repositories have `docs/specs/`, `.agents/skills/`, or any other non-configured
repo path. The source-of-truth section now lists only roots resolved through
`harness repo config get ...`; durable repository contracts are described as
living wherever repo-owned instructions, plans, docs, or code say they live,
and bootstrap-installed skills are described conditionally. Ran
`scripts/sync-bootstrap-assets` again so the root `AGENTS.md` managed block
matches the repaired bootstrap source.

Revision 3 addressed follow-up PR feedback that omitting the conventional
locations entirely left agents without a discovery cue. The managed block now
names `docs/specs/` as one possible repo-owned contract location while still
saying not to assume it exists, and names `.agents/skills/` as the default
Codex bootstrap skills target unless a different target was used. Updated the
install regression test to require that conditional/default guidance while
still rejecting the old unconditional path bullets.

`review-004-delta` found one blocking tests finding: the regression test kept
the new default `.agents/skills` cue but no longer explicitly protected the
`When bootstrap-installed skills are present` conditional. Added that assertion
so the test covers both the default discovery cue and the conditional ownership
guard.

Revision 4 addressed follow-up PR feedback that the revision 3 wording sounded
too defensive. The managed block now uses positive discovery language:
`docs/specs/` is named as a common repo-owned specs/workflow-contract location
when present, `.agents/skills/` is named as the default Codex bootstrap skills
target unless customized, and the old "Do not assume" wording is removed.
Updated the install regression test to protect that tone while still rejecting
the old unconditional path bullets.

#### Review Notes

`review-001-delta` passed with 0 blocking and 0 non-blocking findings across
`docs-consistency` and `agent-ux`. Reviewer submissions confirmed the source
and materialized managed block align with plan intent, the wording reduces the
downstream ownership ambiguity, and the `.agents/skills/` source versus
rendered `.agents/skills` display difference is expected rendering behavior.

### Step 2: Validate and triage the issue

- Done: [x]

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

Ran focused validation:
`go test ./internal/bootstrapsync ./internal/install ./internal/status`,
`harness repo init --dry-run`, and
`harness plan lint docs/plans/active/2026-07-05-clarify-managed-block-wording.md`.
Reinstalled the dev harness binary with `scripts/install-dev-harness` after the
embedded bootstrap asset changed, then reran `harness repo init --dry-run` to
confirm the installed binary reports managed assets as current. Triaged
GitHub issue #280 with `documentation` and `state/accepted`, and left a
rationale comment explaining the documentation scope and why no new config
field or concrete milestone is needed yet.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only recorded validation evidence and GitHub issue
triage after the managed block wording change had already passed step-closeout
review.

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
  - Mitigation: Kept resolved harness roots and command pointers explicit while
    narrowing only the misleading ownership language.
- Risk: Sync output accidentally changes unrelated managed skills.
  - Mitigation: Reviewed the diff after `scripts/sync-bootstrap-assets` and kept
    the source edit limited to `agents-managed-block.md`.

## Validation Summary

- `scripts/sync-bootstrap-assets` refreshed the materialized root `AGENTS.md`
  managed block from `assets/bootstrap/agents-managed-block.md`, including the
  revision 2, revision 3, and revision 4 repairs.
- `go test ./internal/bootstrapsync ./internal/install ./internal/status`
  passed before finalize, during archive prep, after the revision 2 repair, and
  after the revision 3 and revision 4 repair work.
- `harness repo init --dry-run` reports bootstrap assets are already up to
  date after reinstalling the dev harness binary with the updated embedded
  bootstrap asset for the original wording change and later repairs.
- `harness plan lint docs/plans/active/2026-07-05-clarify-managed-block-wording.md`
  passed after the plan was written, after step notes were updated, and during
  the repair rounds.
- `git diff --check` passed after the repair rounds.
- Revision 3 reran the same focused validation after restoring conditional
  discovery cues for `docs/specs/` and `.agents/skills/`.
- After the `review-004-delta` tests finding, focused validation was rerun with
  the restored bootstrap-installed-skills conditional assertion.
- Revision 4 reran focused validation after replacing defensive wording with
  positive discovery cues.

## Review Summary

- Step closeout review `review-001-delta` passed with 0 blocking and 0
  non-blocking findings across `docs-consistency` and `agent-ux`.
- Finalize review `review-002-full` passed with 0 blocking and 0 non-blocking
  findings across `docs-consistency`, `agent-ux`, and `tests`.
- Reviewers confirmed the managed block source, materialized root `AGENTS.md`,
  issue triage record, validation coverage, and plan notes align with the
  approved wording cleanup.
- Revision 2 repaired PR feedback that the managed block still assumed
  non-configured repo paths. Finalize repair review `review-003-delta` passed
  with 0 blocking and 0 non-blocking findings across `docs-consistency`,
  `agent-ux`, and `tests`.
- Revision 3 repaired follow-up PR feedback that the managed block should still
  mention conventional/default surfaces so agents can discover them.
- `review-004-delta` requested one blocking tests fix for the install
  regression coverage. The assertion was restored.
- `review-005-delta` passed with 0 blocking and 0 non-blocking findings for the
  restored test assertion.
- Revision 4 repaired follow-up PR feedback that the managed block sounded too
  defensive. Finalize repair review `review-006-delta` passed with 0 blocking
  and 0 non-blocking findings across `docs-consistency`, `agent-ux`, and
  `tests`.

## Archive Summary

- Archived At: 2026-07-05T09:41:51+08:00
- Revision: 4
- Reopened At: revision 2, after PR feedback on non-configured repo path
  assumptions.
- Reopened At: revision 3, after follow-up PR feedback that the conditional
  guidance still needs to mention the conventional/default locations.
- Reopened At: revision 4, after follow-up PR feedback that the discovery
  wording should be positive rather than defensive.
- PR: https://github.com/catu-ai/easyharness/pull/281
- Ready: Yes. Revision 4 is implemented, validated, reviewed, and ready for
  publish/sync evidence refresh.
- Merge Handoff: Commit and push the archive move, update PR #281, refresh
  CI/sync evidence, and wait for explicit human merge approval.

## Outcome Summary

### Delivered

- Clarified the managed block source so downstream repositories do not read
  `docs/specs/` as an unconditional upstream easyharness product-contract
  location.
- Repaired revision 2 feedback by removing `docs/specs/`, `.agents/skills/`,
  and other non-configured repo paths from the managed block's default split.
  The block now lists only harness roots resolved through repo config, and
  describes repo-owned contracts and bootstrap-installed skills conditionally.
- Repaired revision 3 feedback by restoring explicit conditional discovery cues:
  `docs/specs/` is named as one possible repo-owned contract location, and
  `.agents/skills/` is named as the default Codex bootstrap skills target unless
  a different target was used.
- Repaired revision 4 feedback by replacing defensive "do not assume" wording
  with positive discovery guidance for common repo-owned specs and Codex
  bootstrap skill locations.
- Narrowed the managed block language policy from all tracked docs and code to
  harness workflow artifacts unless repo-owned instructions say otherwise.
- Fixed the raw workflow numbering and rephrased the workflow as work
  coordinated through harness.
- Clarified that easyharness-managed `harness-*` skills are bootstrap-owned
  while other repo-owned skills stay outside easyharness ownership.
- Synced the root `AGENTS.md` managed block from the bootstrap source.
- Triaged issue #280 with `documentation` and `state/accepted`, plus a
  rationale comment.

### Not Delivered

- No new repo config field or managed-instructions help topic was added.
- No command behavior, state transition, review, archive, evidence, or
  bootstrap install mechanics changed.

### Follow-Up Issues

- No GitHub follow-up issue created. The deferred help-topic idea remains
  conditional: consider a future `harness help repo instructions` topic only if
  more downstream customization examples show that compact managed-block
  wording is not enough.
