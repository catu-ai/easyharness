# State Model v0.2 Proposal

## Status

Proposal only. This document does not describe current behavior. It proposes a
replacement for the layered v0.1 state model.

## Purpose

v0.1 currently spreads workflow meaning across multiple overlapping surfaces:

- tracked plan frontmatter
- step status lines
- derived `step_state`
- derived `handoff_state`
- worktree-local idle markers

That model is expressive, but it is harder than necessary for both humans and
agents to answer a simple question:

"Where am I, and what should happen next?"

This proposal replaces the layered state vocabulary with:

- one canonical runtime state value: `current_node`
- one explicit state tree
- a clear split between:
  - durable plan content
  - runtime evidence
  - summary rendering
  - next-action generation

## Goals

- make the state model legible at a glance
- keep agent-aware navigation
- preserve command recommendations in `harness status`
- stop treating review, CI, publish, and sync as parallel state machines
- keep archive placeholders and step-local notes useful
- reduce the amount of execution state stored in tracked plans

## Non-Goals

- preserve v0.1 compatibility
- keep `status`, `lifecycle`, `step_state`, and `handoff_state`
- make every execution move command-driven
- remove archive-time summaries from the plan
- remove step-local `Execution Notes` or `Review Notes`

## Core Proposal

v0.2 uses one canonical runtime state:

```json
{
  "current_node": "execution/step-2/review"
}
```

But the key change is ownership:

- agents do not write `current_node` directly
- `harness status` computes `current_node` from tracked plan content plus
  `.local` evidence
- if the CLI keeps any internal cache or pointer files, those are CLI-owned and
  are not part of the agent authoring contract

Everything else is support data:

- plan content
- approval timestamps
- branch name
- latest review facts
- latest CI facts
- latest publish facts
- latest sync facts

The runtime does not store parallel lifecycle layers such as:

- top-level lifecycle
- step substate
- archived handoff substate

Instead, it derives one path-like node string and derives `summary` and
`next_actions` from:

- `.local` facts
- tracked plan content

## Canonical State Tree

The proposal state tree is:

```text
root
├── idle
├── plan
├── execution
    ├── step-<n>
    │   ├── implement
    │   ├── review
    │   └── fix
    └── finalize
        ├── review
        ├── fix
        ├── archive
        ├── publish
        └── await_merge
└── land
```

Examples:

- `idle`
- `plan`
- `execution/step-1/implement`
- `execution/step-2/review`
- `execution/step-3/fix`
- `execution/finalize/review`
- `execution/finalize/archive`
- `execution/finalize/publish`
- `execution/finalize/await_merge`
- `land`

## Why This Tree

This tree keeps the useful part of the current model:

- the agent knows where it is
- status can still recommend commands
- final review is distinguished from step-local review

But it removes the confusing part:

- no separate `status` vs `lifecycle`
- no separate `step_state`
- no separate `handoff_state`
- no `blocked` branch in the formal state machine
- no worktree-level special state beyond `idle`

## Node Semantics

### `idle`

No active execution is currently being driven.

This is a normal state, not an error.

Typical meanings:

- clean repository with no current work
- previous work landed and cleanup is complete
- no plan has been selected yet

### `plan`

A plan exists, but execution has not started yet.

Typical work here:

- refine scope
- refine steps
- refine acceptance criteria
- update risks and validation strategy
- ask the human for a go-ahead when the plan looks ready

This proposal intentionally does not split `plan` into separate formal
substates such as `drafting` and `awaiting_approval`.

That distinction is still visible in:

- the plan content itself
- the summary text
- the recommended next actions

But it does not need a separate canonical node.

### `execution/step-<n>/implement`

The agent is implementing the current step.

Typical work here:

- write code or docs
- run focused validation
- update step-local `Execution Notes`

### `execution/step-<n>/review`

The current step is in a delta-review loop.

This node implies a real review round exists. It is not just a prose label.

### `execution/step-<n>/fix`

The current step has review findings or equivalent execution feedback that
needs a repair pass before step completion.

### `execution/finalize/review`

The branch candidate is in full-review mode.

This is not the same thing as "the last step's review". It is a whole-branch
review gate.

### `execution/finalize/fix`

The branch candidate needs repair after:

- full review findings
- publish/PR feedback
- CI failures
- sync/conflict repair

### `execution/finalize/archive`

The candidate is in archive-closeout mode.

Typical work here:

- fill plan summaries
- replace archive placeholders
- verify deferred follow-up notes
- run the archive command

### `execution/finalize/publish`

The candidate is already archived locally or otherwise in handoff preparation,
and the agent is performing external publication work.

This node intentionally absorbs several old sub-states:

- PR creation or update
- push
- CI waiting
- remote freshness checks
- conflict repair before merge readiness

They remain visible as facts and next actions, but not as separate nodes.

### `execution/finalize/await_merge`

The candidate is ready and the next major transition depends on a human merge
decision or new feedback that would trigger reopen.

### `land`

Merge has been confirmed and post-merge cleanup is in progress before the
repository returns to `idle`.

Typical work here:

- update linked issues when appropriate
- leave durable PR or issue comments when appropriate
- clear or record CLI-owned pointer files

This node exists because land is a real workflow stage in the repository's
skill system, even though many land actions happen in external systems.

## Plan Contract Changes

v0.2 proposes that tracked plans stop storing top-level execution state.

### Remove From Plan Frontmatter

- `status`
- `lifecycle`
- `revision`
- `updated_at`

The plan remains durable scope and summary, not a runtime state carrier.

### Keep In Plan Frontmatter

- `template_version`
- `created_at`
- `source_type`
- `source_refs`

Additional metadata may be added later, but top-level execution state should
not return to the tracked plan.

### Step Shape

Each step keeps:

- title
- objective
- details
- expected files
- validation
- execution notes
- review notes

Replace:

- `- Status: pending|in_progress|completed|blocked`

With a single durable completion marker, for example:

```md
### Step 1: Replace with first step title

- Done: [ ]
```

The checkbox answers only one question:

"Is this step durably complete?"

It does not answer:

- is the step currently active
- is the agent blocked
- is review in progress
- is CI pending

Those are runtime concerns and belong in `.local`.

### Step Completion Expectations

A step should not be marked done until:

- implementation work for the step is complete
- the step's `Execution Notes` are updated
- the step's `Review Notes` are updated
- the relevant review loop has passed or the step explains why no review was
  needed

### Archive Placeholders

Keep archive placeholders in active plans.

Examples:

- `PENDING_UNTIL_ARCHIVE`
- `PENDING_STEP_EXECUTION`
- `PENDING_STEP_REVIEW`

These are useful authoring reminders and should remain in the plan contract.
They are not themselves lifecycle state.

### Archive-Time Summaries

Keep archive-time summaries in the same plan file.

The same plan remains:

- the planning document before execution
- the durable outcome summary after archive

## `.local` Runtime Contract

v0.2 replaces the v0.1 layered runtime state with an evidence-first model.

The important shift is this:

- agents write plan content and selected evidence files
- harness commands write command-owned artifacts
- `harness status` resolves and may refresh `current_node`
- agents do not directly edit `state.json`, `plan_path`, or `current_node`

### Agent Contract

The agent should author:

- the tracked plan file
- step completion checkboxes
- step-local `Execution Notes`
- step-local `Review Notes`
- lightweight evidence files for external facts that harness does not own

Examples of external evidence that may be agent-authored:

- finalize-readiness checklist evidence
- CI evidence
- publish or PR evidence

The preferred shape is a small checklist-style evidence file that records what
the agent observed in external systems, rather than a file that tries to force
a final node directly.

One simple proposed layout is:

- `.local/harness/plans/<plan-stem>/evidence/finalize-checklist.json`

Example finalize checklist evidence:

```json
{
  "checked_at": "2026-03-21T11:45:00+08:00",
  "freshness": "fresh",
  "conflicts": false,
  "ci": "pending",
  "pr_url": "https://github.com/org/repo/pull/123"
}
```

Additional evidence files may still exist, but the important contract is:

- the agent records observed external facts
- `harness status` interprets those facts and may refresh `current_node`
- the evidence file is not itself a node declaration

### Command-Owned Artifacts

Harness commands should continue to own artifacts that need formal allocation,
validation, or mechanical repository mutation.

Examples:

- execute-start records
- CLI-owned resolved-state cache in `state.json`
- review manifests and ledgers
- review aggregate artifacts
- archive-time move metadata
- reopen metadata
- land metadata

### Forbidden Direct Writes

The agent should not directly write or mutate:

- `current_node`
- `plan_path` pointer files
- the CLI-owned `state.json`
- review manifests or aggregate artifacts by hand
- archive or reopen pointer files by hand

The machine-oriented `state.json` in v0.2 is a CLI-owned cache, not an agent
authoring surface.

## Current-Node Resolution

`harness` should resolve `current_node` from:

- tracked plan content
- command-owned artifacts
- agent-authored evidence files
- merge or land records

The important nuance is:

- the CLI should persist the latest resolved `current_node` in a CLI-owned
  `state.json`
- explicit harness commands may update that file when the transition is
  explicit
- `harness status` may also recompute and refresh that file when implicit facts
  change, such as CI, freshness, conflict, or publish evidence
- agents still do not edit that file directly

The node should not be treated as a second human-authored source of truth. It
is the CLI's resolved view of the current situation.

### High-Level Resolution Order

One workable resolution order is:

1. if land is in progress, resolve `land`
2. otherwise if no current work exists, resolve `idle`
3. otherwise if a plan exists but execution has not started, resolve `plan`
4. otherwise if execution is active, resolve the relevant step or finalize node

### `plan`

Return `plan` when:

- a current plan exists
- `harness execute start` has not yet been recorded for it

`harness status` may still distinguish in prose between:

- "continue editing the plan"
- "ask the human for approval"

But those are summary and next-action concerns, not separate canonical nodes.

### `execution/step-<n>/implement`

Return this node when:

- execution has started
- step `<n>` is the first unfinished step
- there is no active review node for that step
- there is no review result or other evidence forcing a fix node

### `execution/step-<n>/review`

Return this node when:

- step `<n>` is the first unfinished step
- and a review round exists for that step that has not yet been aggregated

Review kind is evidence, not node identity.

That means a step review may still be:

- `delta`
- `full`

Without changing the node name away from `execution/step-<n>/review`.

What matters is where the review sits in the workflow, not which review recipe
was used.

### `execution/step-<n>/fix`

Return this node by default when the latest relevant step review aggregate has
findings that are still actionable.

Proposed rule:

- blockers always resolve to `fix`
- non-blocking findings also resolve to `fix` by default

Why default non-blocking findings to `fix`:

- it keeps the audit trail conservative
- it makes review feedback visible
- the agent can still decide not to change code and advance later

If the agent decides not to address non-blocking findings, the audit trail is
still visible:

- the review found issues
- the plan later advanced without an intervening repair review

The agent should record that judgment in step-local or finalize-local review
notes before the runtime advances.

### `execution/finalize/review`

Return this node when:

- all step checkboxes are complete
- and the branch candidate still needs a whole-branch review gate

This node is not "the last step's review".

It is the review stage for the branch candidate as a whole.

This remains true even if an earlier step-level review happened to use a
full-review method.

Transition note:

- if the last step's review is still active, stay on that step's `review` node
- if the last step's review has actionable findings, go to that step's `fix`
  node
- only after the last step is durably complete should the runtime advance to
  `execution/finalize/review`

### `execution/finalize/fix`

Return this node when:

- the latest relevant finalize review has actionable findings
- or reopened work did not justify adding a new step and should be treated as
  finalize-scope repair
- or publish/CI/sync feedback requires repair on the candidate

### `execution/finalize/archive`

Return this node when:

- finalize review is satisfied
- archive placeholders and summary work remain
- the plan has not yet been archived

### `execution/finalize/publish`

Return this node when:

- the plan has already been archived
- but merge readiness still depends on external evidence such as:
  - PR existence
  - CI
  - freshness
  - conflicts

This is where agent-authored checklist evidence is especially important.

### `execution/finalize/await_merge`

Return this node when:

- archive is complete
- publish evidence exists
- finalize checklist evidence says freshness and conflicts are acceptable
- CI evidence is good enough for merge readiness
- no unresolved finalize repair condition remains

### `land`

Return this node when:

- merge has been confirmed
- but post-merge cleanup is not yet fully recorded

Typical next actions from here may include:

- close linked issues when appropriate
- add PR or issue comments when appropriate
- record land cleanup completion

### Reopen Resolution

After `harness reopen`, the next node should be resolved by the CLI rather than
written by the agent.

Proposed rule:

- if reopened feedback is large enough to justify adding a new step, add the
  new step to the plan and resolve to that new unfinished step's `implement`
  node
- otherwise, resolve to `execution/finalize/fix`

Important audit rule:

- if a new step is added after reopen, do not rewrite prior completed step
  intent just to smuggle the new work into older steps

Reopen placeholder rule:

- reopen should not blank archive-time or finalize-time fields back to empty
- instead, it should replace reopen-sensitive fields with explicit placeholders
  that tell the agent an update is required
- the placeholder should make it obvious that the previous archived wording is
  now stale and must be refreshed before archive or merge readiness is claimed

## Command-Owned Milestones and Agent-Authored Evidence

This boundary must be explicit. It is the highest-leverage part of the v0.2
proposal.

### Command-Owned Rule

A command should own a milestone when it:

- allocates or validates structured artifacts
- performs a tracked-file move or other mechanical repository mutation
- records a durable milestone that should be timestamped uniformly
- must stay independent from ad hoc agent edits

### Agent-Authored Evidence Rule

The agent should author evidence when it:

- is recording external facts that harness does not directly control
- is updating the tracked plan
- is writing step-local closeout notes
- is updating evidence that `harness status` will later interpret

### Commands Must Not Proxy Git or GitHub

This proposal explicitly avoids making harness a wrapper around `git` or `gh`
for normal branch, push, PR, or comment work.

Harness may record milestones about those actions, but it should not perform
those remote or VCS operations on the agent's behalf.

### Command-Owned Milestones

#### `plan -> execution/step-<first-unfinished>/implement`

Required driver:

- `harness execute start`

Why command-owned:

- records approval time
- records execution start time
- establishes the execution milestone

What it should not do:

- run `git checkout`
- run `git switch`
- open a PR
- proxy any `gh` command

Recommended command behavior:

- record approval and start timestamps
- inspect the current branch
- warn or fail if execution is being started from an inappropriate branch
- let the agent perform branch creation or switching directly with git

#### `execution/... -> .../review`

Required driver:

- `harness review start ...`

Why command-owned:

- review nodes depend on real review artifacts
- the CLI owns round IDs, manifests, ledgers, and artifact paths

Node effect:

- once the review round exists, `harness status` should resolve the relevant
  `.../review` node

#### `.../review -> .../fix` or later advancement

Required driver:

- `harness review aggregate ...`

Why command-owned:

- aggregate results are formal evidence
- the CLI owns validation of reviewer submissions and the aggregate artifact

How node resolution should work after aggregate:

- if the aggregate contains blockers, `harness status` should resolve to `fix`
- if the aggregate contains only non-blocking findings, `harness status` should
  still resolve to `fix` by default until later evidence supersedes it
- if the aggregate is clean, `harness status` should keep the node in review
  until the agent completes local closeout, then advance based on plan and
  finalize rules

This means `review aggregate` records the evidence, and `status` interprets the
next node.

In other words:

- aggregate owns the formal review result
- `status` may immediately show `fix`
- the agent still decides whether the repair is code, docs, notes-only, or an
  explicit no-change judgment

#### `execution/finalize/archive -> execution/finalize/publish`

Required driver:

- `harness archive`

Why command-owned:

- replaces placeholders
- validates archive readiness
- performs the tracked-file move
- records archive metadata

Node effect:

- once archive succeeds, `harness status` should resolve to
  `execution/finalize/publish`

#### `execution/finalize/await_merge -> execution/...`

Required driver:

- `harness reopen`

Why command-owned:

- reopen is a mechanical reversal of archive-time assumptions
- the CLI should reset archive-only surfaces and record the reopen milestone

Reopen reset behavior:

- do not wipe finalize or archive summaries back to blank
- replace reopen-sensitive fields with explicit update-required placeholders
- preserve a clear audit trail that the candidate was once archived and then
  invalidated by reopen

Node effect:

- after reopen, `status` should resolve either:
  - a new unfinished step's `implement` node
  - or `execution/finalize/fix`

#### `execution/finalize/await_merge -> land`

Required driver:

- `harness land`

Why command-owned:

- land is now a formal workflow stage
- merge confirmation should be recorded in a CLI-owned way

What it should not do:

- merge the PR itself
- close issues itself
- post comments itself

Those actions still belong to the agent using external tools.

#### `land -> idle`

Required driver:

- `harness land`

Why command-owned:

- idle should only be restored after land cleanup is intentionally recorded
- pointer cleanup is CLI-owned

Recommended minimal command surface:

- v0.2 should prefer a single land command, `harness land`, rather than split
  `land start` and `land record` commands
- the CLI may use the same command to record land entry, refresh land progress,
  and finally restore `idle` once cleanup is complete

Suggested behavior:

- if merge is confirmed but cleanup is still pending, `harness land` should
  record or refresh the `land` milestone and leave the node at `land`
- once cleanup is complete, running `harness land` again should restore `idle`

### Agent-Authored Evidence and Plan Changes

The agent should directly author:

- plan scope edits
- step `Done` checkboxes
- step `Execution Notes`
- step `Review Notes`
- finalize checklist evidence files
- git and GitHub actions themselves

### What the Agent Should Never Do

The agent should never:

- directly set `current_node`
- directly set `plan_path`
- hand-edit a CLI-owned cache file to force a node change
- fabricate review manifests or aggregates
- fabricate archive or reopen milestones

## Status Rendering

`harness status` in v0.2 should read:

- the current plan
- the plan's step completion markers
- the plan's notes and summaries
- `.local` runtime facts
- `current_node`

And then render:

- one concise summary
- the current node
- selected facts
- recommended commands and next actions

### Summary Principle

`summary` should describe:

- where the agent is now
- what kind of work that node implies
- whether any fact changes the obvious next move

Examples:

- `plan`
  - "A plan exists and execution has not started yet."
- `execution/step-2/implement`
  - "Executing Step 2 implementation."
- `execution/step-2/review`
  - "Step 2 is in review."
- `execution/finalize/publish`
  - "Archived candidate is in publish handoff; PR, freshness, conflicts, and CI evidence determine the next move."
- `land`
  - "Merge has been confirmed; post-merge cleanup is still in progress."

### Next-Action Principle

`next_actions` should be derived primarily from `current_node`.

Facts should refine, not replace, that node.

Examples:

- if `current_node = plan`
  - recommend continuing plan edits if needed
  - recommend asking the human for a go-ahead when the plan looks ready
  - recommend `harness execute start` once approval is given
- if `current_node = execution/step-2/implement`
  - recommend continuing the step
  - recommend updating step notes
  - recommend starting a step review once reviewable
- if `current_node = execution/step-2/review`
  - recommend waiting for reviewer submissions
  - recommend `harness review aggregate` once ready
- if `current_node = execution/finalize/archive`
  - recommend final summary closeout
  - recommend `harness archive`
- if `current_node = execution/finalize/publish` and CI is pending
  - recommend waiting for CI
- if `current_node = execution/finalize/publish` and the finalize checklist says
  freshness is stale or conflicts exist
  - recommend refreshing remote state and updating the checklist evidence
- if `current_node = execution/finalize/await_merge`
  - recommend waiting for merge approval
  - recommend `harness reopen` if new feedback arrives
- if `current_node = land`
  - recommend completing post-merge comments or issue updates if needed
  - recommend `harness land` once cleanup is finished

## Recommended `status` Checkpoints

The agent should get the most value from `harness status` at these moments:

1. on entering the repository or resuming work
2. after materially revising a plan before asking for approval
3. immediately after `harness execute start`
4. before starting any review round
5. after every review aggregate
6. after checking a step's `Done` box
7. before entering finalize review
8. before archive
9. immediately after archive
10. after PR creation or update
11. after CI changes state
12. after reopen
13. after `harness land`

## Review Scope in v0.2

v0.2 should keep a distinct whole-branch review concept.

Do not collapse review kind into node identity.

Proposed rule:

- step review nodes do not imply a fixed review kind
- a step review may use either a delta or full review method
- `execution/finalize/review` names the workflow stage, not merely the review
  method

Why keep `execution/finalize/review` as a distinct node:

- step review validates an in-progress slice
- finalize review validates the whole branch candidate after all intended steps
  are complete
- finalize review should cover cross-step consistency, archive closeout quality,
  handoff readiness, and anything else that only makes sense at candidate scope

Transition rule from the last step:

- if the last step review still has actionable findings, stay in the last step's
  repair loop
- once the last step is durably complete, the runtime should resolve
  `execution/finalize/review`
- a full review performed earlier during a step does not automatically clear the
  finalize review gate unless explicit evidence says it did

## Open Questions

This proposal intentionally leaves these questions open for the implementation
plan:

- exact evidence checklist file paths and schema details
- exact CLI-owned `state.json` schema and refresh policy for `current_node`
- exact command names and flags for:
  - `harness execute start`
  - `harness reopen`
  - `harness land`
- exact step checkbox syntax in the template
- how much of the node transition logic should be validated by the CLI versus
  trusted to the controller agent

## Acceptance Criteria

This proposal is ready to turn into an implementation plan when reviewers agree
that it clearly defines:

- one canonical state field: `current_node`
- the v0.2 node tree
- the new split between plan content and runtime facts
- the role of step completion markers and step notes
- the role of finalize review as the whole-branch review gate
- the command-driven milestone set
- the agent-authored evidence set
- the principle used to generate `summary` and `next_actions`
