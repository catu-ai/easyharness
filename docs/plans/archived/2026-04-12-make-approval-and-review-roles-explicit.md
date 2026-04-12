---
template_version: 0.2.0
created_at: "2026-04-12T12:07:00+08:00"
source_type: direct_request
source_refs: []
size: M
---

# Make approval and review roles explicit in the harness workflow

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Reduce agent workflow slips around plan approval and reviewer boundaries by
making those moments explicit in the CLI and bootstrap guidance instead of
leaving them mostly implicit in prompt text. The new workflow should still be
agent-operated and trust-based: agents execute the harness commands, and the
system assumes agents will not fabricate human approval, but the command
surface should make the required steps much harder to miss.

The main product change is a new explicit approval step in the plan lifecycle:
`harness plan approve --by human`. Approval becomes a durable tracked plan
fact through `approved_at` frontmatter, while `harness execute start` remains a
separate execution milestone and refuses to proceed until approval is recorded.
Review submission should gain a similar lightweight role cue through a `--by`
flag so reviewer work is easier for agents to keep distinct from controller
work without turning harness into an identity-verification system.

## Scope

### In Scope

- Add a new `harness plan approve --by human` command to make human approval an
  explicit workflow transition before execution starts.
- Extend the tracked plan contract to carry a durable `approved_at` frontmatter
  field for plans approved through the new workflow.
- Keep `harness execute start` as a separate execution-start milestone, but
  require recorded approval before it can succeed.
- Update `harness status` and related lifecycle guidance so approval-seeking is
  an explicit next action rather than an implied prerequisite.
- Add `--by` to `harness review submit` as a lightweight reviewer-role cue and
  persist that role hint in the stored review submission artifact.
- Update bootstrap prompts, CLI/spec docs, and focused tests so agents learn
  and the repository enforces the new command shape consistently.
- Preserve the trust-based operating model: no strong identity verification or
  external approval service is required for this slice.

### Out of Scope

- Strong authentication or cryptographic proof that approval came from a human
  or that review came from a distinct runtime actor.
- A separate external approval queue, web UI, or out-of-band human signoff
  service.
- Broad redesign of the full harness lifecycle beyond approval and reviewer
  role cues.
- Backfilling historical archived plans with new approval metadata.

## Acceptance Criteria

- [x] `harness plan approve --by human` exists, records approval on the tracked
      active plan, and updates workflow/status output so approval is no longer
      an implicit step.
- [x] New tracked plans may carry `approved_at` in frontmatter after approval,
      and plan/schema/lint behavior clearly documents and tolerates missing
      values for historical plans that predate this workflow.
- [x] `harness execute start` fails with a clear message when approval has not
      been recorded, and succeeds normally once approval exists.
- [x] `harness status` and related lifecycle guidance explicitly tell agents to
      seek Human approval and run `harness plan approve --by human` before
      execution when the plan is ready but not yet approved.
- [x] `harness review submit` accepts a `--by` role cue, persists it, and the
      reviewer skill plus review docs teach reviewer subagents to use it.
- [x] Bootstrap prompts, CLI/spec docs, and focused automated tests cover the
      approval boundary, reviewer-role cue, and the fact that `size` and
      `workflow_profile` remain independent concepts.

## Deferred Items

- Strong actor identity enforcement for controller versus reviewer submissions.
- Any future richer approval provenance beyond the tracked `approved_at`
  timestamp and the trust-based `--by human` command shape.
- Non-Codex-specific integration hooks for external approval tools.

## Work Breakdown

### Step 1: Define the approval and reviewer-role contract

- Done: [x]

#### Objective

Make the intended workflow shape explicit in the tracked specs and bootstrap
guidance before changing runtime behavior.

#### Details

This step should update the contract surfaces that teach both humans and agents
how the harness works: the managed `AGENTS.md` block, `harness-plan`,
`harness-execute`, `harness-reviewer`, and the CLI/spec docs. The contract
should say plainly that a direct request to do work does not by itself approve
the newly written plan; approval becomes explicit through `harness plan approve
--by human`. It should also say that reviewer-role separation is prompted by
`harness review submit --by <reviewer-name>`, but harness still relies on trust rather than
hard identity verification in this slice.

The size versus workflow-profile clarification belongs here too: approval and
review-role fixes should not leave the recent `size` / `lightweight` confusion
unaddressed. The docs should make it explicit that `size` describes magnitude,
while `workflow_profile` separately decides whether a plan is `lightweight`.
`XS` remains a normal standard-plan size.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`
- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/controller-truth-surfaces.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- The docs and prompts consistently describe explicit plan approval, separate
  execute start semantics, and the reviewer-role cue.
- A cold reader can tell that the new approval step is explicit, trust-based,
  and separate from execution start.

#### Execution Notes

Updated the bootstrap-managed AGENTS block, `harness-plan`,
`harness-execute`, `harness-reviewer`, and the execute-controller references
to make the approval boundary explicit. The contract now states that writing a
plan or receiving the original task request does not approve execution, that
agents must record approval with `harness plan approve --by human` before
`harness execute start`, and that reviewer subagents should submit through
`harness review submit --by <reviewer-name>`.

This step also clarified the `size` versus `workflow_profile` distinction in
the managed docs and spec text so `lightweight` is no longer conflated with
small-but-standard slices such as `XS`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This contract-editing step stayed inside one coherent
controller-owned implementation slice and was reviewed at finalize time with
the full behavior change in view.

### Step 2: Add the CLI and runtime support for explicit approval

- Done: [x]

#### Objective

Implement the new approval command and make lifecycle/status behavior depend on
recorded approval before execution can start.

#### Details

This step should add `harness plan approve --by human` to the CLI, teach the
plan document/frontmatter model about `approved_at`, and update lifecycle
services so `execute start` refuses unapproved plans with a clear error. The
status and next-action surfaces should point agents toward approval when the
current plan is still waiting for it.

Historical plans should remain readable and lint-compatible without any bulk
backfill. The implementation should define one clean legacy rule for missing
`approved_at` instead of faking historical timestamps into old plans. Runtime
and status behavior should stay clear about the difference between “this plan
predates the explicit approval field” and “this active plan still needs human
approval now.”

#### Expected Files

- `internal/cli/app.go`
- `internal/lifecycle/service.go`
- `internal/plan/document.go`
- `internal/plan/lint.go`
- `internal/plan/template.go` only if template/frontmatter examples need
  structural changes
- supporting contract/schema files under `schema/` and `internal/contracts/`
  as needed

#### Validation

- The new command writes approval into the tracked plan and is reflected by
  status.
- `execute start` fails before approval and succeeds after approval.
- Legacy plans without `approved_at` still have a documented, tested behavior
  rather than forcing repository-wide backfill.

#### Execution Notes

Added `harness plan approve --by human` to the CLI and lifecycle service,
persisting approval as tracked-plan frontmatter `approved_at`. The lifecycle
service now renders and validates the updated plan file, keeps approval
idempotent, and rejects approval attempts after execution has already started.

`harness execute start` now refuses unapproved active plans while still
tolerating already-executing legacy state that predates explicit approval.
`harness status` now distinguishes between unapproved plans, which prompt the
controller to ask the human and run `harness plan approve --by human`, and
approved plans that are ready for `harness execute start`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The runtime and status gate changes were intentionally
reviewed together with the reviewer-role persistence and end-to-end tests
instead of as an isolated mid-slice checkpoint.

### Step 3: Add reviewer-role cues and focused regression coverage

- Done: [x]

#### Objective

Complete the workflow change by adding `review submit --by`, syncing the
materialized bootstrap outputs, and adding focused tests that keep the new
approval and reviewer-role contracts from drifting.

#### Details

This step should extend review submission inputs and artifacts with the `--by`
cue, update the reviewer skill to tell reviewer subagents to choose and submit
their reviewer name/role explicitly, and add targeted tests for the new
command shapes and next-action messaging. Because the repository dogsfoods the
bootstrap assets, any source-asset edits must be followed by
`scripts/sync-bootstrap-assets`.

The tests should not try to prove real human identity. Instead they should
assert the repository-owned contract: the approval step is explicit, the
execute gate is enforced, the reviewer cue is persisted, and the docs/prompts
continue to describe the intended workflow precisely.

#### Expected Files

- `assets/bootstrap/`
- `.agents/skills/` and root `AGENTS.md` after sync
- `internal/review/service.go`
- `schema/inputs/review.submission.schema.json`
- focused tests under `internal/..._test.go`, `tests/e2e/`, or `tests/smoke/`
  matching the changed behavior
- `scripts/sync-bootstrap-assets`

#### Validation

- `scripts/sync-bootstrap-assets` leaves the materialized bootstrap outputs in
  sync with the edited source assets.
- Focused tests cover approval gating, status guidance, and reviewer `--by`
  persistence.
- The repository's own prompts and generated outputs teach the same workflow.

#### Execution Notes

Extended `harness review submit` to require `--by`, persisted the reviewer
label in stored submission artifacts, and validated it before aggregation.
Updated the CLI help, contract schemas, generated schema index, bootstrap
outputs, and focused unit/e2e/smoke tests so the approval gate and reviewer
role cue are enforced consistently.

Follow-up PR review comments then tightened the published wording:

- command examples now consistently use `--by human` and
  `--by <reviewer-name>` rather than mixing flag styles
- the managed agreement now states only that `XXS` may still choose the
  ordinary standard workflow, without pulling `XS` into the same sentence
- the reviewer-orchestration reference no longer adds the extra “not a strong
  identity check” reminder inside the reviewer submission prompt
- the controller pre-review checklist no longer includes an out-of-place
  approval check

The repo-local dogfood material was resynced with `scripts/sync-bootstrap-assets`
and `scripts/sync-contract-artifacts`, and the full `go test ./...` suite
passed after the contract updates.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This regression-coverage step existed to lock down the
same end-to-end behavior that finalize review inspects, so a separate
step-closeout round would have duplicated the final reviewer pass.

## Validation Strategy

- Run `harness plan lint` on the new tracked plan before seeking approval.
- Use focused unit/e2e coverage for the approval command, execution gate,
  status guidance, and review submission cue instead of a broad unrelated test
  sweep.
- Re-sync bootstrap assets and verify both source assets and materialized
  outputs reflect the same explicit approval/reviewer-role contract.

## Risks

- Risk: approval semantics become muddy if `plan approve` and `execute start`
  are not kept clearly separate.
  - Mitigation: keep approval as a tracked-plan fact and preserve
    `execute start` as the distinct execution-start milestone in both docs and
    runtime behavior.
- Risk: introducing `approved_at` could accidentally force noisy history
  backfill or break older plans.
  - Mitigation: define a clean legacy-missing rule, tolerate historical
    absence, and test that old plans remain valid without backfill.
- Risk: `review submit --by` could look like strong identity enforcement even
  though this slice intentionally remains trust-based.
  - Mitigation: document the flag as a role cue and provenance hint, not an
    authenticated identity claim.

## Validation Summary

- Added focused regression coverage for:
  - `PlanApprove` recording and refreshing `approved_at`
  - `ExecuteStart` rejecting unapproved plans
  - `review submit --by` persisting reviewer provenance
  - aggregate tolerating legacy stored submissions without `by`
- Re-synced bootstrap and contract artifacts with:
  - `scripts/sync-bootstrap-assets`
  - `scripts/sync-contract-artifacts`
- Verified the repo-local binary after reinstalling with:
  - `scripts/install-dev-harness`
- Passed full repository validation with:
  - `go test ./...`
- After PR wording follow-up, reran targeted validation with:
  - `harness plan lint docs/plans/active/2026-04-12-make-approval-and-review-roles-explicit.md`
  - `go test ./internal/status ./tests/smoke`

## Review Summary

- `review-001-full` requested changes.
  - Findings: repeat `plan approve` did not refresh `approved_at`, and
    aggregate became too strict about legacy stored submissions missing `by`.
- Addressed the blocking findings by:
  - making repeat `PlanApprove` rewrite `approved_at`
  - keeping `review submit --by` required for new submissions while allowing
    aggregate to read legacy stored submissions without `by`
  - adding a focused assertion that persisted submissions store reviewer
    provenance in `submission.by`
- `review-002-full` passed cleanly with reviewer subagent follow-up confirming
  the fixes and contract alignment.
- PR review comments then requested wording cleanup around `--by` examples,
  `XXS` versus `lightweight`, the pre-review checklist, and reviewer prompt
  tone.
- `review-003-full` passed cleanly after the wording follow-up confirmed those
  contract surfaces stayed aligned.

## Archive Summary

- Archived At: 2026-04-12T14:02:34+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/150
- Ready: Candidate revalidated after PR comment wording follow-up and ready for
  archive plus fresh publish/CI/sync evidence before waiting for merge
  approval.
- Merge Handoff: After archive, commit and push the reopened finalize-fix
  updates, refresh publish/CI/sync evidence on PR #150 for revision 2, and
  then wait for explicit human merge approval.

## Outcome Summary

### Delivered

- Added explicit plan approval via `harness plan approve --by human`.
- Persisted approval in tracked plan frontmatter as `approved_at`.
- Gated `harness execute start` on recorded approval while preserving legacy
  already-executing tolerance.
- Added `--by` to `harness review submit` and stored reviewer provenance in
  submission artifacts.
- Updated status guidance, managed prompts, specs, generated schemas, and
  regression coverage to teach and enforce the new workflow.
- Followed up on PR comments by standardizing `--by` examples to space
  separation, tightening the `XXS` / `lightweight` wording, removing the
  pre-review approval checklist item, and simplifying the reviewer
  orchestration note.

### Not Delivered

- Strong actor identity enforcement or cryptographic proof of reviewer /
  approver identity.
- External approval-system integrations beyond the trust-based local command
  flow.

### Follow-Up Issues

- #149 Track stronger approval and reviewer provenance after explicit approval
  rollout
