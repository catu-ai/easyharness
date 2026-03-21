# State Model

## Purpose

This document is a descriptive map of the current v0.1 state model used by
`superharness`.

The repository already defines parts of this behavior across:

- [Plan Schema](./plan-schema.md)
- [CLI Contract](./cli-contract.md)
- lifecycle, status, and review command implementations
- repo-local execution skills

This document consolidates those surfaces into one place so humans and agents
can answer four questions quickly:

- what state layers exist
- how those layers relate to each other
- what conditions move work from one state to another
- what `.local` artifacts are expected to exist as inputs or outputs

## Non-Goals

This document does not propose new states, rename existing states, or simplify
the model yet.

Future discussion may revise the model, but this document should first capture
the current behavior faithfully enough that optimization work starts from a
shared baseline instead of memory.

## Design Split

The current model is intentionally split between durable tracked state and
derived local state:

- tracked plan markdown is the durable source of truth for scope, lifecycle,
  step breakdown, and archive-time summaries
- `.local/harness/` is disposable execution support for review, CI, sync,
  publish, and worktree handoff details
- `harness status` combines the tracked plan and `.local` evidence into a
  small output vocabulary for humans and agents

This means the repository does not have one single flat state machine. It has
multiple layers that are read together.

## State Layers

### 1. Workflow Narrative Layer

The repository-level workflow currently reads as:

1. Discovery
2. Plan
3. Execute
4. Archive / publish handoff / await merge approval
5. Land

This is the broad human workflow. It is not stored verbatim in one field.

Important mapping notes:

- `Discovery` mostly happens outside the tracked plan state.
- `Plan` ends at `lifecycle: awaiting_plan_approval`.
- `Execute` maps to `lifecycle: executing` or `lifecycle: blocked`.
- `Archive / publish handoff / await merge approval` maps to
  `status: archived` plus `lifecycle: awaiting_merge_approval`, with an
  additional derived `handoff_state`.
- `Land` is a local cleanup step after merge, not a tracked plan lifecycle.

### 2. Tracked Plan Layer

The tracked plan carries the durable coarse state.

Primary carriers:

- plan path
  - `docs/plans/active/`
  - `docs/plans/archived/`
- frontmatter
  - `status`
  - `lifecycle`
  - `revision`

Current tracked vocabulary:

- `status`
  - `active`
  - `archived`
- `lifecycle`
  - `awaiting_plan_approval`
  - `executing`
  - `blocked`
  - `awaiting_merge_approval`

The tracked plan does not store:

- `step_state`
- `handoff_state`
- `worktree_state`

Those are always derived later.

### 3. Step Layer

The tracked plan also carries step-local progress inside `## Work Breakdown`.

Each step has its own `Status:` line:

- `pending`
- `in_progress`
- `completed`
- `blocked`

Current-step inference is intentionally narrow:

- prefer the first `in_progress` step
- otherwise use the first `pending` step
- otherwise omit `step`

Current implementation does not infer the current step from a `blocked` step.
In practice, the top-level `lifecycle: blocked` is expected to carry the main
workflow pause, while a step-level `blocked` value provides durable step
context inside the plan body.

### 4. Derived Execution Layer

When a plan is executing, `harness status` derives a smaller local hint called
`step_state`.

Current `step_state` vocabulary:

- `implementing`
- `reviewing`
- `fix_required`
- `waiting_ci`
- `resolving_conflicts`
- `ready_for_archive`

This is not a strict linear state machine. It is a prioritized answer to the
question: "What is the agent mainly doing around the current step right now?"

### 5. Derived Archived Handoff Layer

When a plan is archived, `harness status` derives a local `handoff_state`.

Current `handoff_state` vocabulary:

- `pending_publish`
- `waiting_post_archive_ci`
- `followup_required`
- `ready_for_merge_approval`

This means `lifecycle: awaiting_merge_approval` is still too coarse by itself.
An archived plan may still need commit/push/PR work, post-archive CI, or sync
repair before it is truly ready to wait for merge approval.

### 6. Worktree Layer

When no current plan is active, the worktree may still have local handoff
context.

Current `worktree_state` vocabulary:

- `idle_after_land`

This is produced only after merge cleanup has been recorded locally. It is not
stored in a tracked plan.

## Current State Carriers

### Tracked Carriers

- `docs/plans/active/<plan>.md`
  - active plan content
- `docs/plans/archived/<plan>.md`
  - frozen archived plan content

Tracked plan data owns:

- durable scope
- acceptance criteria
- step list and step statuses
- top-level lifecycle
- archive-time summaries
- revision

### Local Worktree Carrier

- `.local/harness/current-plan.json`

Current shape:

```json
{
  "plan_path": "docs/plans/active/2026-03-21-example.md",
  "last_landed_plan_path": "docs/plans/archived/2026-03-20-example.md",
  "last_landed_at": "2026-03-21T09:30:00+08:00"
}
```

Current semantics:

- `plan_path`
  - points to the current tracked plan when one is selected
- `last_landed_plan_path`
  - records the last landed archived plan after merge cleanup
- `last_landed_at`
  - timestamp for that landed handoff marker

### Plan-Local Execution Carrier

- `.local/harness/plans/<plan-stem>/state.json`

Current shape:

```json
{
  "plan_path": "docs/plans/active/2026-03-21-example.md",
  "plan_stem": "2026-03-21-example",
  "active_review_round": {
    "round_id": "review-001-delta",
    "kind": "delta",
    "aggregated": false,
    "decision": ""
  },
  "latest_ci": {
    "snapshot_id": "ci-001",
    "status": "pending"
  },
  "sync": {
    "freshness": "fresh",
    "conflicts": false
  },
  "latest_publish": {
    "attempt_id": "publish-001",
    "pr_url": "https://example.invalid/pull/123"
  }
}
```

`state.json` is the CLI-owned local snapshot for the current plan. It points to
the latest review, CI, sync, and publish evidence rather than duplicating the
full history inline.

### Plan-Local History Carriers

- `.local/harness/plans/<plan-stem>/events.jsonl`
  - append-only local trajectory
- `.local/harness/plans/<plan-stem>/reviews/<round-id>/`
  - review round directory
  - commonly contains `manifest.json`, `ledger.json`,
    `submissions/<slot>.json`, and `aggregate.json`
- `.local/harness/plans/<plan-stem>/ci/<snapshot-id>.json`
  - CI evidence for one candidate state
- `.local/harness/plans/<plan-stem>/sync/<snapshot-id>.json`
  - remote freshness and conflict evidence for one candidate state
- `.local/harness/plans/<plan-stem>/publish/<attempt-id>.json`
  - PR or publish metadata for one push/update attempt

## How Status Is Read

The current status model is easiest to read in this order:

1. determine whether a current tracked plan can be resolved
2. if yes, read `plan_status` and `lifecycle`
3. infer the current `step`
4. if `lifecycle: executing`, infer `step_state`
5. if `lifecycle: awaiting_merge_approval`, infer `handoff_state`
6. if no current plan is resolved, check whether `idle_after_land` can be
   reported from `.local`

This layered read is important because the same plan can be:

- `active + executing + Step 2 + implementing`
- `active + executing + Step 2 + reviewing`
- `archived + awaiting_merge_approval + pending_publish`
- `archived + awaiting_merge_approval + ready_for_merge_approval`

Those are not four different top-level lifecycles. They are different
combinations of coarse tracked state and local derived hints.

## Current Plan Resolution

The current plan is resolved using a mix of tracked files and local pointer
state:

- if exactly one active plan exists under `docs/plans/active/`, it can be used
  even without `.local/harness/current-plan.json`
- if multiple active plans exist, `current-plan.json` is required to disambiguate
- if `current-plan.json` still points at archived work but exactly one active
  plan exists, the active plan wins
- if no current plan can be resolved and a last-landed marker exists,
  `harness status` reports `worktree_state: idle_after_land`
- if no current plan can be resolved and no last-landed marker exists,
  `harness status` currently returns an error rather than an explicit idle state

## Step-State Inference

Current implementation uses this precedence order while
`lifecycle: executing`:

1. `resolving_conflicts`
2. `reviewing`
3. `fix_required`
4. `waiting_ci`
5. `ready_for_archive`
6. `implementing`

The order matters:

- conflict work overrides review and CI hints
- an unaggregated review round overrides CI waiting
- an aggregated non-passing review overrides closeout guidance
- archive-ready only appears after all stronger execution hints are absent

### `implementing`

Default executing mode when nothing stronger applies.

Typical evidence:

- current plan is `active + executing`
- no active unaggregated review round
- no known review-fix requirement
- no pending CI snapshot
- no active conflict signal
- archive-readiness conditions are not all satisfied yet

### `reviewing`

Produced when:

- `state.json.active_review_round` exists
- `active_review_round.aggregated == false`

Expected local evidence:

- review round directory exists
- round manifest exists
- reviewer submissions are in flight or still pending

### `fix_required`

Produced when:

- the active review round is aggregated
- and the effective decision is not `pass`
- or the aggregated decision cannot be recovered safely from local state or
  aggregate artifacts

This is a conservative state. Current behavior prefers "repair or rerun review"
over closeout or archive guidance whenever the latest review decision is not
cleanly known.

### `waiting_ci`

Produced when:

- `state.json.latest_ci.status == "pending"`

### `resolving_conflicts`

Produced when:

- `state.json.sync.conflicts == true`

### `ready_for_archive`

Produced only when all of these hold:

- every plan step is `completed`
- every acceptance criterion is checked
- archive-readiness evaluation returns no blockers

This is still an executing-state hint. It means "closeout is complete and the
plan is ready to archive now", not "the plan is already archived".

## Archived Handoff-State Inference

Current implementation uses this logic while
`lifecycle: awaiting_merge_approval`:

### `pending_publish`

Produced when:

- `latest_publish` is absent
- or `latest_publish.pr_url` is empty

Interpretation:

- the plan is archived locally
- but the archive move has not yet been fully published or connected to the PR

### `followup_required`

Produced when publish exists but local handoff evidence is not clean enough.

Current triggers include:

- sync state is absent
- sync says conflicts are present
- sync freshness is not `fresh`
- CI exists with a non-success, non-pending terminal state

Interpretation:

- the plan is archived
- but the archived candidate still needs publish, CI, or sync repair before
  merge approval

### `waiting_post_archive_ci`

Produced when:

- publish evidence exists
- sync evidence exists and is fresh
- and CI is either pending or absent

Interpretation:

- the archive move has been published
- but the post-archive candidate is still waiting on CI evidence

### `ready_for_merge_approval`

Produced when:

- publish evidence exists
- sync evidence exists and is fresh
- sync reports no conflicts
- CI evidence exists and is successful

Interpretation:

- the archived candidate is now ready to wait for merge approval

## Workflow and Transition Model

### Discovery to Plan Approval

The broad workflow starts outside the tracked lifecycle.

Typical transition:

- discovery clarifies direction
- a tracked plan is created under `docs/plans/active/`
- frontmatter starts with:
  - `status: active`
  - `lifecycle: awaiting_plan_approval`
  - `revision: 1`

Typical `.local` expectations:

- `current-plan.json` may be absent or may point at the new plan
- `state.json` may be absent

### Plan Approval to Execution

Transition:

- human approves the tracked plan
- plan lifecycle moves from `awaiting_plan_approval` to `executing`

Tracked changes:

- frontmatter `lifecycle` is updated
- one or more step statuses usually move from `pending` to `in_progress`

Local expectations:

- `.local` may still be absent at the beginning of execution
- `.local` becomes more important once review, CI, sync, or publish work starts

### Execution Inner Loop

The current step inner loop is:

1. infer or choose the current step
2. implement the slice
3. run focused validation
4. update `Execution Notes`
5. if ready, start a delta review
6. aggregate the round
7. fix findings if needed
8. update `Review Notes`
9. mark the step complete when the objective is genuinely satisfied

The key point is that step completion and execution substate are separate:

- a step can be `in_progress` while `step_state` is `reviewing`
- all steps can be `completed` while `step_state` is still `fix_required`
- all steps can be `completed` and `step_state` can become `ready_for_archive`

### Starting a Review Round

Command:

```bash
harness review start --spec <path>
```

Transition effects:

- allocates a plan-local round ID such as `review-001-delta`
- creates a round directory under `.local/harness/plans/<plan-stem>/reviews/`
- writes the round manifest and ledger
- updates `state.json.active_review_round`
  - `round_id`
  - `kind`
  - `aggregated: false`
  - `decision: ""`

Status effect:

- execution typically presents `step_state: reviewing`

### Submitting Review Work

Command:

```bash
harness review submit --round <round-id> --slot <slot>
```

Transition effects:

- writes `submissions/<slot>.json`
- updates the round ledger

Status effect:

- no coarse state changes yet
- the round remains `reviewing` until aggregation

### Aggregating a Review Round

Command:

```bash
harness review aggregate --round <round-id>
```

Transition effects:

- writes `aggregate.json`
- updates `state.json.active_review_round`
  - `aggregated: true`
  - `decision: <pass|changes_requested|...>`

Status effects:

- if the effective decision is `pass`, execution falls through to the next
  strongest state such as `implementing`, `waiting_ci`, or `ready_for_archive`
- if the effective decision is non-passing, execution moves to `fix_required`
- if the aggregate cannot be recovered safely later, execution also stays in
  the conservative `fix_required` path

### Entering and Leaving Blocked

Blocked work is carried at the plan lifecycle layer.

Transition:

- when work cannot proceed without human input or an external dependency,
  frontmatter moves to `lifecycle: blocked`
- after the blocker is resolved, lifecycle returns to `executing`

Expected evidence:

- the tracked plan should describe the blocker durably
- `.local` may still retain review/CI/sync evidence from before the blockage

### Reaching Archive Closeout

The plan becomes archive-closeout eligible only after:

- all steps are completed
- all acceptance criteria are checked
- step-local execution and review placeholders are replaced
- archive summary placeholders are replaced
- archive summary contains `PR`, `Ready`, and `Merge Handoff`
- if deferred items remain, follow-up issues are not left as `NONE`
- review gating is satisfied
- any present CI or sync evidence is archive-clean

Current implementation details:

- a passing aggregated review is required before archive
- revision `1` specifically requires that passing review to be `full`
- if CI evidence exists and is not successful, archive is blocked
- if sync evidence exists and is stale or conflicted, archive is blocked
- absence of CI or sync evidence does not currently block archive by itself

When all blockers are cleared, execution can surface `step_state: ready_for_archive`.

### Archiving

Command:

```bash
harness archive
```

Preconditions:

- current plan resolves successfully
- tracked plan is `status: active`
- tracked plan is `lifecycle: executing`
- archive-readiness evaluation returns no blockers

Tracked changes:

- file moves from `docs/plans/active/` to `docs/plans/archived/`
- frontmatter `status` becomes `archived`
- frontmatter `lifecycle` becomes `awaiting_merge_approval`
- frontmatter `updated_at` is refreshed
- `Archive Summary` is stamped with:
  - `Archived At`
  - `Revision`

Local changes:

- `current-plan.json.plan_path` is updated to the archived path
- if `state.json` already exists, its `plan_path` is updated to the archived
  path
- existing plan-local review, CI, sync, and publish state is retained

Status effect after archive:

- coarse state becomes `archived + awaiting_merge_approval`
- a separate `handoff_state` still decides whether publish and CI handoff are
  complete

### Archived Handoff Progression

Typical archived progression:

1. `pending_publish`
2. `waiting_post_archive_ci`
3. `ready_for_merge_approval`

Possible interruption:

- `followup_required` if sync evidence is missing, stale, conflicted, or if CI
  finishes in a non-success state

Important note:

- an archived plan is not automatically merge-ready just because archive
  succeeded
- post-archive publish and CI work still belongs to the execution story

### Reopening

Command:

```bash
harness reopen
```

Preconditions:

- current plan is `status: archived`
- current plan is `lifecycle: awaiting_merge_approval`

Tracked changes:

- file moves back to `docs/plans/active/`
- `status` becomes `active`
- `lifecycle` becomes `executing`
- `revision` increments by one
- `updated_at` is refreshed
- `Validation Summary` resets to `PENDING_UNTIL_ARCHIVE`
- `Review Summary` resets to `PENDING_UNTIL_ARCHIVE`
- `Archive Summary` resets to `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary` resets its archive-time content placeholders

Local changes:

- `current-plan.json.plan_path` points back to the active path
- if `state.json` exists:
  - `plan_path` is updated back to the active path
  - `active_review_round` is cleared
  - `latest_ci` is cleared
  - `sync` is cleared
  - `latest_publish` is currently retained

Status effect:

- coarse state becomes `active + executing`
- execution resumes from the inferred current step or from whatever plan update
  follows

### Merge and Land Recording

Merge itself happens outside the tracked plan contract.

After the archived candidate is actually merged, local cleanup is recorded with:

```bash
harness land record
```

Preconditions:

- current plan is still `status: archived`
- current plan is still `lifecycle: awaiting_merge_approval`

Tracked changes:

- none

Local changes:

- `current-plan.json.plan_path` is cleared
- `current-plan.json.last_landed_plan_path` is set
- `current-plan.json.last_landed_at` is set

Status effect:

- if no new current plan exists, `harness status` reports
  `worktree_state: idle_after_land`

## Expected `.local` Inputs and Outputs by Phase

### Awaiting Plan Approval

Minimum expected inputs:

- tracked plan under `docs/plans/active/`

Optional local inputs:

- `current-plan.json`

Usually absent:

- review, CI, sync, publish artifacts

### Early Execution

Minimum expected inputs:

- tracked active plan

Optional local inputs:

- `state.json`

Common outputs as work begins:

- step-local `Execution Notes` in the tracked plan

### Active Review

Expected local inputs and outputs:

- `state.json.active_review_round`
- `reviews/<round-id>/manifest.json`
- `reviews/<round-id>/ledger.json`
- `reviews/<round-id>/submissions/<slot>.json`
- eventually `reviews/<round-id>/aggregate.json`

### CI Waiting

Expected local inputs:

- `state.json.latest_ci`
- optionally `ci/<snapshot-id>.json`

### Conflict Resolution

Expected local inputs:

- `state.json.sync`
- optionally `sync/<snapshot-id>.json`

Required signal for `resolving_conflicts`:

- `sync.conflicts == true`

### Archive Closeout

Expected tracked inputs:

- completed steps
- checked acceptance criteria
- archive summaries filled in

Expected local inputs:

- effective latest aggregated review decision
- any CI evidence relevant to the candidate
- any sync evidence relevant to the candidate
- any publish evidence needed to complete archive summaries

### Archived Handoff

Expected tracked inputs:

- archived frozen plan

Expected local inputs:

- `current-plan.json.plan_path` pointing at the archived plan
- `state.json.plan_path` pointing at the archived plan, if `state.json` exists
- publish evidence for PR linkage
- post-archive CI evidence
- sync evidence for merge-sensitive freshness

### Post-Land Idle

Expected local inputs:

- `current-plan.json.last_landed_plan_path`
- `current-plan.json.last_landed_at`

Expected absence:

- no current `plan_path` in `current-plan.json`

## Current Command Ownership

The current model works because different surfaces are owned by different
actors:

- humans or controller agents update tracked plan content
  - lifecycle
  - step statuses
  - execution notes
  - review notes
  - archive summaries
- `harness review start`
  - creates review round artifacts
  - sets `active_review_round`
- `harness review submit`
  - writes reviewer submissions
  - updates the round ledger
- `harness review aggregate`
  - writes `aggregate.json`
  - records aggregated review decision in `state.json`
- `harness archive`
  - moves the plan to archived state
  - updates plan-path pointers
- `harness reopen`
  - moves the plan back to active execution
  - clears stale review, CI, and sync state
- `harness land record`
  - clears the current-plan pointer
  - records last-landed context
- `harness status`
  - reads everything above
  - infers the small state vocabulary presented to agents and humans

## Baseline Observations About Current Behavior

These observations are part of the current baseline and should be kept explicit
for future simplification work:

- the repository currently uses layered state, not one flat state machine
- `step_state` is a prioritized hint, not a durable or strictly sequential FSM
- `handoff_state` is required to understand archived work correctly
- `Land` is not a tracked lifecycle; it is a local cleanup transition
- `idle_after_land` exists, but a completely empty worktree without a current
  plan still returns a "no current plan found" error rather than a dedicated
  idle state
- step-level `blocked` exists in the plan schema, but current-step inference
  does not surface it as the current step
- archive readiness is stricter about review evidence than about absent CI or
  sync evidence
- post-archive handoff is stricter about requiring explicit publish, sync, and
  CI evidence
- `reopen` currently clears review, CI, and sync state but retains the latest
  publish metadata
