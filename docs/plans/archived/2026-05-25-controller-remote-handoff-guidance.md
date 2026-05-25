---
template_version: 0.2.0
created_at: "2026-05-25T09:45:05+08:00"
approved_at: "2026-05-25T09:46:40+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/202
    - https://github.com/catu-ai/easyharness/issues/12
    - https://github.com/catu-ai/easyharness/issues/200
    - https://github.com/catu-ai/easyharness/issues/213
size: XS
---

# Controller remote handoff guidance

## Goal

Resolve issue #202 by updating the controller-facing docs and managed
`harness-execute` skill guidance for the read-only remote handoff workflow
delivered across #199, #200, and #213.

The end state should let a future controller use `harness status` as the first
orientation surface during publish, CI, sync, and merge handoff. The guidance
should explain how to interpret durable local evidence together with
non-authoritative `facts.remote_handoff`, while preserving the boundary that
`harness status` is read-only, `harness evidence refresh` is the explicit
remote-to-evidence write path, and git/GitHub write operations stay outside
harness core commands.

## Scope

### In Scope

- Update the distributed managed skill pack under `assets/bootstrap/` so
  controller agents get complete remote handoff guidance from the packaged
  `harness-execute` materials.
- Cover the common `harness status` remote handoff cases a controller must act
  on: missing recorded PR evidence, degraded remote reads, pending checks,
  failing checks, passing or fresh remote facts that still need local evidence,
  stale or conflicted sync, and merge-ready / merged handoff boundaries.
- Clarify that `facts.remote_handoff` is an advisory live observation surface;
  `state.current_node` remains driven by durable local publish, CI, and sync
  evidence.
- Preserve the explicit manual fallback path through
  `harness evidence submit --kind publish|ci|sync`.
- Run bootstrap sync so `.agents/skills/` and the managed `AGENTS.md` block, if
  touched by generated output, match `assets/bootstrap/`.
- Record the milestone closeout implication: v0.4.0 currently has #202 and the
  parent umbrella #12 open; #202 appears to be the final concrete child item
  before #12 can be closed or marked complete.

### Out of Scope

- Implementing new remote observation, evidence refresh, status, or schema
  behavior.
- Changing merge policy, review policy, state transitions, or evidence
  semantics.
- Adding a `harness publish` command or any git/GitHub write command to harness
  core.
- Performing PR creation, issue closure, or milestone bookkeeping before the
  archived candidate reaches publish handoff.

## Acceptance Criteria

- [x] `harness-execute` publish/CI/sync guidance tells controllers to consult
      `harness status` remote facts before ad hoc remote inspection.
- [x] The guidance explains the expected controller response for missing PR
      evidence, degraded observations, pending checks, failing checks,
      passing/fresh-but-unrecorded facts, stale/conflicted sync, and
      merge-ready handoff states.
- [x] The docs clearly distinguish read-only status facts from evidence-writing
      commands and from human/agent-executed git or GitHub write actions.
- [x] Managed bootstrap outputs are synchronized after editing
      `assets/bootstrap/`.
- [x] Validation confirms the plan lint, bootstrap sync check, and relevant
      docs checks pass.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Update managed controller guidance

- Done: [x]

#### Objective

Make the packaged `harness-execute` remote handoff guidance complete enough for
a future controller to follow without reconstructing #200/#213 from issue
history.

#### Details

Focus on the distributed source materials under `assets/bootstrap/`, especially
`harness-execute` references that govern publish, CI, sync, closeout, and
controller truth-surface use. Keep the guidance concise and decision-shaped:
the controller should know what `harness status` is saying, which command or
manual action comes next, and when direct `gh` inspection is only diagnostic.

Do not duplicate full JSON schemas or implementation internals. Point to the
existing command contracts where appropriate, but make the controller workflow
clear enough that an agent can handle normal handoff cases from the skill text.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/publish-ci-sync.md`
- `assets/bootstrap/skills/harness-execute/references/closeout-and-archive.md`
- `assets/bootstrap/skills/harness-execute/references/controller-truth-surfaces.md`

#### Validation

- Reread the changed guidance against issue #202 acceptance criteria.
- Confirm the wording preserves the read-only boundary for `harness status` and
  the explicit write boundary for `harness evidence refresh`.

#### Execution Notes

Updated the managed `harness-execute` source under `assets/bootstrap/` to make
remote handoff status-first. The main addition is a decision guide in
`publish-ci-sync.md` covering missing recorded PR evidence, unsupported
handoff targets, degraded observations, pending checks, failed checks,
passing/fresh remote facts that still need evidence refresh, stale or
conflicted sync, live drift after local merge-ready evidence, and already
merged PRs.

Also tightened the main execute skill's node hints, closeout/archive handoff
steps, and Pre-Land truth-surface checklist so controllers look at
`harness status` first, treat `facts.remote_handoff` as read-only and
non-authoritative, use `harness evidence refresh` to record remote facts, keep
manual `harness evidence submit` as fallback, and reserve direct `gh`
inspection for diagnostics.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 is documentation and managed-skill guidance only;
Step 2 will run bootstrap sync, lint, and final validation before finalize
review.

### Step 2: Sync bootstrap outputs and verify closeout readiness

- Done: [x]

#### Objective

Refresh materialized bootstrap outputs, validate the plan and docs, and record
whether #202 completes the v0.4.0 parent issue split.

#### Details

Run the bootstrap sync script after editing `assets/bootstrap/` so repo-local
materialized skills stay in sync. Recheck the v0.4.0 milestone before archive:
as of planning, GitHub reports open issues #202 and parent #12 only, and #12's
split comment identifies #202 as the remaining controller-doc child item. The
archive or PR handoff should state whether landing this plan is expected to
close #202 and make #12 closeable.

#### Expected Files

- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/publish-ci-sync.md`
- `.agents/skills/harness-execute/references/closeout-and-archive.md`
- `.agents/skills/harness-execute/references/controller-truth-surfaces.md`
- `AGENTS.md`
- `docs/plans/active/2026-05-25-controller-remote-handoff-guidance.md`

#### Validation

- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-25-controller-remote-handoff-guidance.md`
- `git diff --check`

#### Execution Notes

Ran `scripts/sync-bootstrap-assets`, which refreshed the materialized
`.agents/skills/harness-execute` files from `assets/bootstrap/`. Verified the
managed bootstrap output is synchronized, the tracked plan still lints, and the
diff has no whitespace errors.

Rechecked the v0.4.0 milestone through `gh api`: it still has open issues #202
and parent #12 only. #12's split comment lists #203, #199, #200, and #202 as
the v0.4.0 child scope, so landing this plan should close #202 and make #12
ready to close or explicitly retarget.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only synchronized generated bootstrap output,
validated the docs/plan state, and recorded milestone status. Finalize review
will cover the full docs and managed-skill change.

## Validation Strategy

This is docs and managed-skill integration work, so validation centers on
contract consistency rather than runtime behavior. The executor should lint the
tracked plan, verify bootstrap dogfood outputs are synchronized, run
whitespace/diff checks, and reread the changed guidance against the status and
evidence contracts in `docs/specs/cli-contract.md` and `docs/specs/state-model.md`.

No Go behavior tests are expected unless the implementation unexpectedly changes
runtime code.

## Risks

- Risk: The guidance could imply that live remote facts advance workflow state.
  - Mitigation: State repeatedly that `facts.remote_handoff` is
    non-authoritative and that durable publish, CI, and sync evidence still
    drive `state.current_node`.
- Risk: The skill text could drift from the distributed bootstrap source.
  - Mitigation: Edit `assets/bootstrap/` first and run
    `scripts/sync-bootstrap-assets` before validation.
- Risk: Closing #202 could leave parent #12 ambiguous.
  - Mitigation: Record in archive/PR handoff that #202 is the last concrete
    child item observed in the v0.4.0 milestone and that #12 should be closed or
    explicitly retargeted if maintainers want more parent-scope work.

## Validation Summary

- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-25-controller-remote-handoff-guidance.md`
- `git diff --check`
- `gh api repos/catu-ai/easyharness/milestones --jq '.[] | select(.title=="v0.4.0") | {number,title,open_issues,closed_issues,state}'`
- `gh api 'repos/catu-ai/easyharness/issues?milestone=6&state=open&per_page=100' --jq '.[] | {number,title,state,html_url}'`

## Review Summary

Finalize `review-001-full` passed with no blocking findings. The `agent_ux`
and `workflow_boundary` reviewer slots found no issues. The
`docs_consistency` slot reported one minor wording drift where the main
`harness-execute` startup checklist described live remote handoff observations
as local state; the wording was repaired to distinguish local review/evidence
state from live remote handoff observations, and bootstrap output was
resynchronized.

## Archive Summary

- Archived At: 2026-05-25T09:51:59+08:00
- Revision: 1
- PR: NONE. Create or update the PR after committing and pushing this archived
  candidate.
- Ready: The tracked steps are complete, acceptance criteria are checked,
  validation has passed, and full finalize review passed with the single
  non-blocking docs wording finding repaired.
- Merge Handoff: After archive, commit the archive move and summary updates,
  push the branch, open or update the PR for issue #202, record publish
  evidence with the PR URL, then use `harness evidence refresh` or manual
  CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`. Landing this candidate should also make
  parent issue #12 ready to close or explicitly retarget.

## Outcome Summary

### Delivered

- Added status-first remote handoff guidance to the managed
  `harness-execute` skill.
- Added a controller decision guide for missing recorded PR evidence,
  unsupported handoff targets, degraded observations, pending checks, failed
  checks, passing/fresh live facts that still need evidence refresh, stale or
  conflicted sync, live drift after local merge-ready evidence, and already
  merged PRs.
- Tightened closeout/archive and Pre-Land guidance so controllers use
  `harness status` first, record remote facts through `harness evidence
  refresh` or manual evidence submit, and keep git/GitHub write actions outside
  harness core commands.
- Synchronized the materialized `.agents/skills/harness-execute` output from
  `assets/bootstrap/`.
- Rechecked the v0.4.0 milestone: #202 and umbrella #12 are the only open
  issues, and #202 appears to be the last concrete child item in #12's v0.4.0
  split.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
