---
template_version: 0.2.0
created_at: "2026-04-09T22:19:30+08:00"
source_type: direct_request
source_refs:
    - '#89'
size: XL
---

# Delta review anchor and agent protocol

## Goal

Clarify the harness review protocol for strong controller and reviewer agents
while also giving reviewers durable round-local work artifacts that reduce
early-stop behavior. The updated contract should make `delta` review start from
a durable git anchor, keep `AGENTS.md` principle-level, create reviewer-local
artifacts at `review start`, and move the operational rules into the
`harness-execute` and `harness-reviewer` skills so reviewers know what to read,
where to start, how to progressively write their submission, and when
continuity via `resume_agent` is the default.

This slice should codify the decisions from discovery: both `full` and `delta`
reviewers read the full active plan; `delta` review starts from a real commit
anchor but may dig beyond the initial diff when related logic or contract risk
deserves it; newly discovered findings are reported in the same round with
normal severity handling; narrow same-slot `delta` follow-up should prefer
resume over fresh reviewer spawn; and every reviewer round should start with a
precreated local `submission.json` skeleton that the reviewer updates
progressively instead of writing only at the end.

## Scope

### In Scope

- Add one minimal repo-level review invariant to the harness-managed
  `AGENTS.md` block: `delta` review must anchor to a real git commit.
- Update bootstrap review orchestration guidance so controller dispatch
  explicitly includes review kind, active-plan context, `delta` anchor SHA,
  slot assignment, and bounded change summary.
- Extend review-round local artifacts so `review start` creates one
  reviewer-owned folder per slot with a `submission.json` skeleton that can be
  progressively updated during review.
- Allow `harness review submit` to retain reviewer-progress fields in the
  stored submission artifact while keeping aggregate logic focused on the
  canonical `summary` and `findings`.
- Update controller guidance so the latest passed review anchor becomes the
  default starting point for later `delta` review, with heuristics for when a
  narrow repair should escalate to `full`.
- Update reviewer guidance so both `full` and `delta` reviewers read the full
  active plan, begin from the anchored diff for `delta`, and may expand
  inspection when related logic or contract meaning warrants it.
- Make same-slot narrow `delta` follow-up default to `resume_agent` when the
  earlier reviewer submission was valid and continuity is still helpful.
- Sync bootstrap assets into the materialized repo-local skill tree and managed
  root `AGENTS.md`, then validate the synced outputs.

### Out of Scope

- Changing status/archive gating logic or adding new hard review-state gates.
- Adding hard workflow gates such as forbidding a new review round before the
  previous round aggregates.
- Encoding rigid `delta` versus `full` trigger rules beyond lightweight
  heuristics for controller judgment.
- Turning the managed `AGENTS.md` block into a long operational playbook.
- Building the reviewer-worklog UI in this slice; that follow-up is tracked by
  [#125](https://github.com/catu-ai/easyharness/issues/125).

## Acceptance Criteria

- [x] The harness-managed `AGENTS.md` block adds only the minimal review
      invariant that `delta` review must anchor to a real git commit, without
      bloating the repo-level contract with reviewer operating detail.
- [x] `harness-execute` bootstrap guidance tells the controller to dispatch
      reviewer subagents with explicit `review kind`, active-plan step/finalize
      context, slot assignment, dimension instructions, bounded change summary,
      and `anchor sha` for `delta` review.
- [x] `harness review start` creates a reviewer-local artifact directory for
      each slot with a preseeded `submission.json` skeleton that the reviewer
      can progressively update during the round.
- [x] `harness review submit` accepts and persists the richer reviewer
      `submission.json` payload, including non-aggregate progress fields, while
      `harness review aggregate` ignores those extra fields when computing the
      round decision.
- [x] Controller guidance explains that later `delta` review should default to
      the latest passed-review anchor commit and keeps `delta -> full`
      escalation as heuristic examples rather than hard triggers.
- [x] `harness-reviewer` guidance tells both `full` and `delta` reviewers to
      read the full active plan before reviewing, start `delta` review from the
      anchored diff, and freely deepen inspection when related logic or
      contract risk is uncovered.
- [x] Reviewer guidance tells reviewer agents to use the round-local
      `submission.json` as a progressive working file during review instead of
      writing the final payload only at the end.
- [x] Reviewer guidance explicitly allows same-round reporting of newly found
      related issues, with severity deciding whether they block approval.
- [x] Review-orchestration guidance defaults narrow same-slot `delta`
      follow-up to `resume_agent` instead of fresh reviewer spawn, while still
      documenting the cases that require a fresh reviewer.
- [x] `scripts/sync-bootstrap-assets` refreshes the materialized `.agents`
      skills and managed root `AGENTS.md`, and `scripts/sync-bootstrap-assets
      --check` plus the relevant smoke/bootstrap tests pass afterward.

## Deferred Items

- [#125](https://github.com/catu-ai/easyharness/issues/125): Expose reviewer
  progressive review worklog detail in the harness UI once the local artifact
  contract lands.

## Work Breakdown

### Step 1: Add the minimal repo-level review invariant

- Done: [x]

#### Objective

Keep the managed `AGENTS.md` block lean while still making the `delta` review
anchor contract repo-visible.

#### Details

Update only the bootstrap-managed `AGENTS.md` source so the repo-level review
contract states that `delta` review must anchor to a real git commit. Do not
move detailed reviewer operating procedure into the managed block; those rules
belong in the skills. If needed, make the surrounding wording slightly crisper
so the new invariant fits naturally inside the existing review-execution
section.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`

#### Validation

- A cold reader can find the `delta` anchor invariant in the managed block
  without being forced to absorb controller/reviewer playbook detail there.
- `scripts/sync-bootstrap-assets --check` passes after the bootstrap sync.

#### Execution Notes

Added the minimal managed-block invariant in
`assets/bootstrap/agents-managed-block.md` so the repo-level contract now says
`delta` review must anchor to a real git commit. Synced the managed root
`AGENTS.md` afterward through the bootstrap asset refresh.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This invariant landed as part of one tightly coupled
protocol slice across runtime contracts and bootstrap assets, so a separate
step-local review would create an artificial boundary. The combined candidate
is reviewed in finalize full review.

### Step 2: Add reviewer-local progressive submission artifacts

- Done: [x]

#### Objective

Give each reviewer round a durable local working artifact that reduces
early-stop behavior without introducing a second scratchpad format.

#### Details

Update the review-round artifact layout so each reviewer slot gets its own
round-local folder under `.local/harness/plans/<plan-stem>/reviews/<round-id>/`
with a precreated `submission.json` skeleton. That JSON should preserve the
current `summary` and `findings` contract while permitting extra fields for
progressive reviewer work such as coverage, checked areas, open hypotheses, or
other worklog detail. `harness review submit` should validate and store the
canonical review fields while preserving the extra fields in the recorded
artifact, and `harness review aggregate` should ignore those extra fields
rather than rejecting the artifact.

#### Expected Files

- `internal/contracts/review.go`
- `internal/review/service.go`
- `internal/inputschema/generated_schemas.go`
- `docs/specs/cli-contract.md`
- `tests/e2e/review_workflow_test.go`
- `internal/review/service_test.go`

#### Validation

- A new review round creates reviewer-local directories and a progressive
  `submission.json` skeleton for each slot.
- `harness review submit` accepts the richer skeleton format without dropping
  or rejecting the extra progress fields.
- `harness review aggregate` still computes the same decision using only the
  canonical review payload fields.

#### Execution Notes

Changed `harness review start` to precreate
`.local/harness/plans/<plan-stem>/reviews/<round-id>/submissions/<slot>/submission.json`
for every slot, seeded with canonical slot identity plus reviewer-owned
worklog fields. `harness review submit` now preserves top-level extra fields
in stored submission artifacts, while `harness review aggregate` ignores those
extras and still requires canonical submitted fields through the ledger-driven
aggregation path. Added focused review-service, CLI, review UI, contract-sync,
and E2E coverage for the new artifact shape.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The progressive submission artifact contract is tightly
coupled to the bootstrap reviewer guidance and was implemented as one combined
slice. Finalize full review checks the end-to-end behavior instead of
pretending there was an isolated step boundary here.

### Step 3: Update controller and reviewer protocol guidance

- Done: [x]

#### Objective

Teach the bootstrap skills how strong agents should dispatch and consume
`full` versus `delta` review with the new progressive submission artifact.

#### Details

Update the controller-side guidance in `harness-execute` and its review
references so reviewer dispatch carries the round-specific fields that matter:
review kind, active-plan step/finalize context, slot, dimension instructions,
bounded change summary, and `anchor sha` for `delta`. The same guidance should
say that the latest passed review anchor becomes the default start point for
later `delta` review, and that choosing `full` remains a controller judgment
supported by heuristics, not hard triggers.

Update `harness-reviewer` so reviewer agents always read the full active plan,
start `delta` review from the anchored diff, progressively update the
round-local `submission.json`, and may extend inspection when the diff touches
related logic or contract meaning that deserves deeper review. The reviewer
should report such findings in the same round instead of deferring them purely
because they were discovered through expansion beyond the initial diff. Keep
the tone consistent with strong subagents that can use the full toolkit rather
than passive checklist runners.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `assets/bootstrap/skills/harness-execute/references/step-inner-loop.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`
- `.agents/skills/harness-execute/references/step-inner-loop.md`
- `.agents/skills/harness-reviewer/SKILL.md`

#### Validation

- The execute-side guidance is enough for a cold controller agent to dispatch a
  `delta` review with the required anchor and context fields.
- The reviewer-side guidance is enough for a cold reviewer agent to know it
  must read the full plan, where to start `delta` review, and that it should
  progressively maintain the round-local submission artifact.
- The updated resume guidance makes narrow same-slot `delta` follow-up default
  to continuity while still documenting when a fresh reviewer is safer.

#### Execution Notes

Updated the bootstrap protocol so controller guidance now treats real commit
anchors as the default start point for later `delta` review, dispatches review
kind plus active-plan context plus bounded change summary, and prefers
same-slot `resume_agent` for narrow `delta` follow-up. Updated
`harness-reviewer` so reviewer subagents read the full active plan, start from
the anchored diff for `delta`, progressively maintain the slot-owned
`submission.json`, and keep inspecting past the first few findings when
related logic or contract risk deserves it.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Controller and reviewer protocol updates only make
sense when reviewed together with the runtime artifact changes, so this step
defers to the combined finalize full review.

### Step 4: Sync bootstrap outputs and lock the protocol with validation

- Done: [x]

#### Objective

Materialize the bootstrap changes into the dogfood outputs and prove the repo
still packages and syncs the review protocol deterministically.

#### Details

Run `scripts/sync-bootstrap-assets` after the bootstrap source edits so the
managed root `AGENTS.md` and repo-local `.agents/skills/` outputs stay aligned.
Then validate the synced protocol with the existing bootstrap sync checks and
the most relevant smoke coverage. Update the tracked plan with the final
validation and review closeout notes so a future agent can trust the result
from the repo alone.

#### Expected Files

- `AGENTS.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`
- `.agents/skills/harness-execute/references/step-inner-loop.md`
- `.agents/skills/harness-reviewer/SKILL.md`
- `docs/specs/cli-contract.md`
- `internal/contracts/review.go`
- `internal/review/service.go`
- `tests/smoke/bootstrap_sync_test.go`
- `internal/bootstrapsync/sync_test.go`

#### Validation

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `go test ./internal/bootstrapsync ./tests/smoke`
- `harness plan lint docs/plans/active/2026-04-09-delta-review-anchor-and-agent-protocol.md`

#### Execution Notes

Ran `scripts/sync-bootstrap-assets`, `scripts/sync-bootstrap-assets --check`,
`scripts/sync-contract-artifacts`, and `scripts/sync-contract-artifacts
--check` so the managed root `AGENTS.md`, repo-local `.agents` skills, and
generated schema artifacts all match source. Validation covered:

- `go test ./internal/review ./internal/reviewui ./internal/contractsync ./internal/cli -count=1`
- `go test ./internal/bootstrapsync -count=1`
- `go test ./tests/e2e -run 'ReviewWorkflow|ReviewRepairLoop|ExplicitStepRepair' -count=1`
- `go test ./tests/e2e -run ReviewWorkflow -count=1`
- `go test ./tests/smoke -run 'SyncBootstrapAssetsCheckPassesForCurrentRepo|SyncContractArtifactsCheckPassesForCurrentRepo|InstallBootstrapsFreshRepository|InstallSkillsScopeBootstrapsOnlySkills' -count=1`

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is the validation and packaging closeout for
the same protocol slice and is covered by the candidate-level finalize full
review.

## Validation Strategy

- Lint the tracked plan before execution starts.
- Add focused review-service and E2E coverage for progressive reviewer
  `submission.json` skeleton creation, permissive submit-time retention of
  extra progress fields, and aggregate-time ignoring of those fields.
- Sync the bootstrap assets and rerun the drift check so packaged assets,
  materialized `.agents` skills, and the managed root `AGENTS.md` stay aligned.
- Run focused bootstrap sync and smoke tests after the protocol edits.
- During execution review, use reviewer subagents to check both sides of the
  protocol: controller dispatch guidance and reviewer behavior guidance.

## Risks

- Risk: The managed `AGENTS.md` block grows from a principle-level contract
  into a second copy of the operational review playbook.
  - Mitigation: Keep only the real-commit `delta` anchor invariant in the
    managed block and push all controller/reviewer mechanics into the skills.
- Risk: Execute and reviewer guidance can drift so controller dispatch fields
  and reviewer expectations no longer line up.
  - Mitigation: Treat the two skills plus the review-orchestration reference as
    one protocol change, review them together, and validate the synced outputs
    after bootstrap refresh.
- Risk: The richer reviewer submission artifact could accidentally change
  aggregate semantics or make submit validation too permissive.
  - Mitigation: Keep the canonical `summary` and `findings` contract intact,
    add focused tests that prove aggregate ignores the extra fields, and defer
    UI consumption to [#125](https://github.com/catu-ai/easyharness/issues/125)
    rather than coupling both changes in one slice.

## Validation Summary

- `scripts/sync-bootstrap-assets`
- `scripts/sync-contract-artifacts`
- `scripts/install-dev-harness`
- `go test ./internal/review ./internal/reviewui ./internal/cli ./tests/e2e -count=1`
- `go test ./tests/smoke -run 'SyncBootstrapAssetsCheckPassesForCurrentRepo|SyncContractArtifactsCheckPassesForCurrentRepo' -count=1`

## Review Summary

Finalize review rounds `review-001-full` through `review-004-full` surfaced
and drove the remaining contract gaps: controller-owned submission identity
fields overriding reviewer worklog state, missing durable manifest anchor
persistence, starter `submission.json` skeletons lacking canonical
`summary/findings`, degraded review UI treating untouched starter artifacts as
submitted, CLI/docs wording that still weakened the required delta anchor, and
runtime validation that only required a non-empty anchor instead of a real git
commit in git-backed repos. After those repairs plus the final focused test
updates landed, `review-005-full` passed clean with no remaining correctness,
docs-consistency, or agent-UX findings.

## Archive Summary

- Archived At: 2026-04-09T23:41:44+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate has a passing finalize review, synchronized bootstrap
  and contract artifacts, and focused validation coverage proving the delta
  anchor plus progressive reviewer-artifact protocol end to end.
- Merge Handoff: Archive this candidate, commit and push the archive move,
  open or update the PR, record publish/CI/sync evidence for the archived
  candidate, and then wait for explicit human merge approval before land.

## Outcome Summary

### Delivered

- Added the minimal managed `AGENTS.md` review invariant that `delta` review
  must anchor to a real git commit, while moving controller/reviewer operating
  detail into the bootstrap skills and review-orchestration guidance.
- Extended `harness review start`, `submit`, `aggregate`, contracts, schemas,
  and review UI handling so each slot gets a progressive `submission.json`
  skeleton, reviewer worklog fields persist without affecting aggregate
  semantics, and untouched starter artifacts stay conservative in degraded UI
  states.
- Tightened the delta-review runtime contract so git-backed repositories must
  resolve `anchor_sha` to a real commit, refreshed the materialized `.agents`
  outputs and managed root `AGENTS.md`, and locked the protocol with focused
  unit, CLI, E2E, and smoke coverage.

### Not Delivered

- [#125](https://github.com/catu-ai/easyharness/issues/125): Expose reviewer
  progressive review worklog detail in the harness UI.

### Follow-Up Issues

- [#125](https://github.com/catu-ai/easyharness/issues/125): Expose reviewer
  progressive review worklog detail in the harness UI.
