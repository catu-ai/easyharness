# State Model

## Purpose

This document defines the normative v0.2 state model for `easyharness`.

The v0.2 model replaces the v0.1 layered vocabulary of tracked lifecycle,
derived step state, derived handoff state, and worktree hints with one
canonical runtime node:

```json
{
  "current_node": "execution/step-2/implement"
}
```

Exact transition enumeration lives in
[State Transitions](./state-transitions.md). Exact tracked-plan and CLI schema
details live in [Plan Schema](./plan-schema.md) and
[CLI Contract](./cli-contract.md).

Field-level JSON structure for the CLI-owned local artifacts described here
lives in the checked-in schema registry at
[`schema/index.json`](../../schema/index.json). See
[Contract Registry](./contract.md) for the ownership model and discovery
entrypoints.

## Non-Goals

The v0.2 state model does not:

- preserve v0.1 compatibility layers such as `lifecycle`, `step_state`,
  `handoff_state`, or `worktree_state`
- keep top-level execution state in tracked plan frontmatter
- support multiple simultaneous active plans in one repository
- make harness a wrapper around routine `git` or `gh` operations

## Core Principles

### One Canonical Node

Every workflow question should reduce to one node string. Summary text,
selected facts, and recommended next actions are all derived from that node
plus the latest relevant artifacts.

### CLI-Owned Resolution

Agents do not set `current_node` directly. The CLI resolves it from the
current plan artifact plus command-owned artifacts. The plan-local
`state.json` remains a CLI-owned control artifact for runtime facts that must
persist across commands, but it is not the storage location for a cached
latest `current_node`.

### Safe Local Persistence

CLI-owned runstate files must stay parseable even when commands run close
together or a process exits during persistence.

- the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` must be written with atomic
  replacement rather than in-place overwrite writes
- each plan-local `state.json` under the local runtime root resolved by
  `harness repo config get paths.local_runtime` must also use atomic
  replacement
- any command that mutates a plan-local `state.json` must acquire a shared
  per-plan state-mutation lock before it loads and rewrites that file
- if the per-plan state lock is already held, the command should fail with a
  clear error rather than waiting silently or risking a stale overwrite

### Read-Model Purity

Read-model services resolve snapshots from the current tracked plan,
CLI-owned runtime artifacts, and append-only evidence or review records. They
must not acquire mutation locks, create lock files, rewrite workflow state,
append timeline events, or refresh machine-local watchlist recency.

This rule applies to status resolution as a library/read-model operation and
to UI/API/dashboard resource reads. A read-model caller may observe a momentary
multi-file transition while another command is mutating local artifacts. The
reader should surface a conservative degraded result, warning, or error from
the files it can read rather than hiding the possibility behind a read lock.

Agent-facing CLI commands may add command-level coordination around a
read-model snapshot when their public contract calls for it. In particular,
`harness status` may briefly wait for a state mutation lock to settle before
resolving a snapshot, as defined in [CLI Contract](./cli-contract.md). That
settle behavior belongs to the CLI checkpoint command, not to the underlying
status read model. The settle check must be passive and non-destructive: it
must not create a missing lock file, must not use the ordinary mutation-lock
acquisition helper as its probe, and must not hold any mutation lock while
resolving the snapshot.

### Durable Plan, Disposable Runtime

Tracked active plans remain the durable source of decisions, scope, acceptance,
Review Focus, step progress, and Closeout for all workflow profiles.
Lightweight work uses the same
schema and the same active-plan root resolved by
`harness repo config get paths.plans.active`, but its archived snapshot moves
under the local runtime root resolved by
`harness repo config get paths.local_runtime` so the workflow can stay
lightweight for narrow low-risk changes. Goal-oriented work is deferred to
v0.7.0 and is not a current workflow profile. Runtime trajectory,
milestone timestamps, and external-fact capture also belong in the resolved
local runtime root. There is no separate local active lightweight plan path in
this model.

### Explicit Command Boundaries

Commands own milestones and append-only trajectory where consistency and
timestamps matter. Agents still own plan edits, reviewable code and docs, and
all direct `git` or GitHub actions.

## Canonical Node Tree

```text
root
├── idle
├── plan
├── execution
│   ├── step-<n>
│   │   └── implement
│   └── finalize
│       ├── review
│       ├── fix
│       ├── archive
│       ├── publish
│       └── await_merge
└── land
```

## Ownership Split

### Plan Artifact Owns

- durable scope and non-goals
- durable decisions and Review Focus
- acceptance criteria
- outcome-based step list and step `Done` markers
- archive-time `Closeout`

For active work in the currently implemented workflow profiles, this plan
artifact is a tracked file under the active plan root resolved by
`harness repo config get paths.plans.active`, which defaults to
`docs/plans/active/`. Standard archives stay tracked under the archived plan
root resolved by `harness repo config get paths.plans.archived`, which
defaults to `docs/plans/archived/`. Lightweight archived snapshots move under
the local runtime root resolved by
`harness repo config get paths.local_runtime`, defaulting to
`.local/harness/plans/archived/` for the snapshot directory.
Agents and scripts should resolve these roots with
`harness repo config get paths.plans.active`,
`harness repo config get paths.plans.archived`, and
`harness repo config get paths.local_runtime` instead of inferring defaults.

### Command-Owned Runtime Artifacts Own

- worktree-level current-plan and last-landed context
- execute-start milestones
- review round metadata, including the command-captured `reviewed_head_sha`,
  submission-tracking data, reviewer submissions, persisted review decisions,
  and continuous full-plus-repair-delta finalize coverage; optional
  reviewer-provided finding locations remain preserved in submission and
  decision artifacts
- append-only timeline event indexes under the local runtime root resolved by
  `harness repo config get paths.local_runtime`, defaulting to
  `.local/harness/plans/<plan-stem>/events.jsonl`
- append-only `ci`, `publish`, and `sync` evidence records
- archive milestones
- reopen milestones, including the explicit reopen mode
- land entry and land completion milestones
- the plan-local `state.json` control artifact containing only cross-command
  runtime facts that are not otherwise reconstructed directly from plans or
  append-only artifacts

These CLI-owned JSON artifacts are disposable runtime state, but they still
need crash-safe persistence so the controller can trust `harness status` after
any interrupted or overlapping command.

### Agents Must Not Directly Edit

- `current_node`
- CLI-owned pointer files
- CLI-owned `state.json`
- hidden review-control artifacts or persisted review decision data
- command-owned evidence records that should have been created through
  `harness evidence submit`

## Current Plan Selection

v0.2 assumes one active plan artifact per repository.

Resolution rules:

- if more than one active tracked plan exists under the active plan root
  resolved by `harness repo config get paths.plans.active`, state resolution
  is invalid and should fail rather than guess
- lightweight archived snapshots under the local runtime root resolved by
  `harness repo config get paths.local_runtime` do not count as active-plan
  candidates
- if the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` points to the sole active plan
  path and that path still exists, that plan is current
- otherwise, if exactly one active tracked plan exists under the active plan
  root resolved by `harness repo config get paths.plans.active`, that plan is
  current for `plan` and `execution/...` nodes
- if no active plan exists, CLI-owned archived or landed context may still
  identify the current archived candidate or the most recent landed candidate

## Runtime Inputs

`harness status` resolves `current_node` from:

- the current plan content
- the plan path and any optional `workflow_profile`
- whether execution-start has been recorded
- the first unfinished step from the current plan
- review artifacts for the current step or the finalize gate
- append-only `ci`, `publish`, and `sync` evidence
- archive, reopen, and land milestones
- worktree-level last-landed context when no current work remains

The plan-local `state.json` carries only the control-plane subset of those
inputs that do not already live in a more specific artifact:

- `execution_started_at`
- `revision`
- `active_review_round`
- `finalize_coverage`
- `reopen`
- `land`

`active_review_round` is the mutable pointer for the round that is currently in
flight or most recently completed. An explicit `review abort` clears an
unfinished pointer while preserving that round's artifacts as aborted history.
It is not, by itself, the
archive verdict. `finalize_coverage` is a compact command-validated cache of a
durable coverage chain: the full root round, current tip round, covered Git
head, revision, and unresolved blocking count. Round manifests, the sole
reviewer submission, and the command-owned decision artifact remain the source
of truth, so archive must validate that chain rather than trusting the cache
alone. Reopen preserves the prior coverage tip
so a narrow later revision can extend it with a linked delta; a new full review
may replace the root when broader work invalidates the earlier judgment.
Rewritten ancestry is detected before round creation. When a clean existing
coverage tip is no longer an ancestor of the candidate, automatic inference
starts a new full root rather than creating a delta that cannot aggregate. If
the old tip has unresolved findings, inference fails before creating a round;
only restored ancestry or an explicit full replacement may proceed.

The mutation surfaces around those runtime artifacts stay split on purpose:

- `.state-mutation.lock` serializes rewrites of plan-local `state.json`
- `.review-mutation.lock` serializes review round creation and the atomic
  submission/decision/coverage transaction
- `.timeline-mutation.lock` serializes appends to the plan-local
  `events.jsonl` index

When a review command mutates both review artifacts and `state.json`, it should
acquire the review mutation lock before the state mutation lock. Commands that
only submit reviewer output should stay on the review-artifact path and should
not acquire the state mutation lock just because the round also has local
state.

Mutation commands should remain fail-fast on mutation-lock contention unless a
more specific command contract says otherwise. Waiting for a mutation lock is
reserved for checkpoint reads such as `harness status`, where a short wait can
avoid reporting an in-flight snapshot without letting two mutation commands
silently queue behind each other.

## High-Level Resolution Order

1. If merge has been confirmed and the required post-merge bookkeeping is still
   incomplete, resolve `land`.
2. Otherwise, if no current work exists, resolve `idle`.
3. Otherwise, if the current active plan exists but execution-start has not
   been recorded, resolve `plan`.
4. Otherwise, if an unfinished step exists, resolve the appropriate
   `execution/step-<n>/...` node.
5. Otherwise, resolve the appropriate `execution/finalize/...` node.

The exact transition matrix is normative in
[State Transitions](./state-transitions.md).

Workflow profiles do not add a second node tree. Lightweight reuses the same
canonical nodes while changing where the archived snapshot lives and what
closeout guidance `harness status` should emphasize. Goal-oriented execution is
deferred to v0.7.0 and is not a current profile.

## Node Semantics

### `idle`

No current work is in flight. This is the normal post-land resting state.

### `plan`

A current tracked active plan exists, but execution has not started. Plan
edits, approval, and step refinement happen here for all implemented workflow
profiles.

### `execution/step-<n>/implement`

Execution has started, step `<n>` is the first unfinished step, and no active
review round is relevant yet. Steps are human-visible outcome and validation
boundaries; formal review belongs to finalize.

### `execution/finalize/review`

All intended steps are complete, and the whole-branch candidate still needs the
mandatory formal review gate.

### `execution/finalize/fix`

The whole-branch candidate needs repair because of finalize review findings,
reopened work that did not justify a new step, a `new-step` reopen that is
still waiting for the first new unfinished step to be added, or archived
candidate invalidation that must be repaired before archive or merge readiness
can be claimed again.

### `execution/finalize/archive`

Finalize review is satisfied and the remaining work is archive closeout:
replacing the plan's `Closeout` placeholders and preparing the archive move or
snapshot update.

### `execution/finalize/publish`

The plan is already archived, but merge readiness still depends on external
handoff facts recorded through `publish`, `ci`, and `sync` evidence. For
lightweight work, this phase is also where status should remind the controller
to leave the agreed repo-visible breadcrumb. A PR body can satisfy that
breadcrumb when it is a readable merge memo that explains what changed, why the
branch is mergeable, and why the lightweight path was appropriate. Passing
evidence cannot advance an archived branch whose candidate-owned delta differs
from review beyond the mechanical plan/supplement archive move and allowed
`Closeout` updates. Unrelated upstream base advancement may preserve coverage
when fresh sync evidence and the remote-tracking base prove that delta stayed
identical; overlap, conflict resolution, or candidate drift must reopen and
review again.

### `execution/finalize/await_merge`

The archived candidate is ready for human merge approval. PR existence, CI,
and sync freshness or conflict checks are already satisfied or explicitly
marked `not_applied`, and the archived branch still matches the reviewed
candidate boundary.

### `land`

Merge is confirmed and required post-merge bookkeeping is in progress. This
work remains in `land` until `harness land complete` intentionally restores
`idle`.

## Step and Review Rules

- The first unfinished step determines the current execution step.
- Plan steps are execution and validation boundaries, not review boundaries.
  Each carries a title, `Done`, outcome, covered acceptance criteria, and an
  optional concise check.
- Step and acceptance progress are derived from the tracked plan. The agent
  updates them only when evidence establishes a meaningful boundary; no
  per-tool status writes are required.
- Intermediate uncertainty may use focused validation or bounded advisor
  subagents without creating review state.
- Finalize review always creates one integrated reviewer for the complete
  candidate. The fixed standard rubric and plan `Review Focus` are automatic;
  the controller cannot omit or select them.
- The reviewer may create bounded advisor subagents but remains the sole
  reviewer and submission owner. Advisors have no harness slot, role, finding
  provenance, or aggregate participation.
- Reviewer submission atomically rechecks captured HEAD, stores the judgment,
  resolves inherited findings, derives the decision, and updates coverage.
- A completed review round is immutable. A second or concurrent-losing submit
  cannot replace its submission, findings, decision, or coverage.
- Blocking findings enter `execution/finalize/fix`; non-blocking findings stay
  visible without forcing repair.
- After `execution/finalize/fix`, a narrow repair should be reviewed by a linked
  repair delta that extends the existing full coverage. A new full review is
  required only when the repair materially changes the design, scope, or risk
  model enough to invalidate the earlier whole-candidate judgment.

## Publish, CI, and Sync Evidence Rules

The repository standardizes three command-owned evidence domains:

- `publish`
  - records PR or handoff facts for the archived candidate
- `ci`
  - records the latest relevant CI or required-check result
- `sync`
  - records remote freshness and conflict facts relevant to merge readiness

Rules:

- all three domains may be recorded manually through
  `harness evidence submit`
- when publish evidence already records a supported PR URL,
  `harness evidence refresh` may observe that PR through a read-only provider
  and record derived `ci` and `sync` evidence
- missing evidence never means success or not-applicable
- `not_applied` must be recorded explicitly when a domain truly does not apply
- freshness belongs to `execution/finalize/publish`, not to pre-archive
  readiness

### Remote Handoff Identity

The recorded `publish` evidence is the authoritative identity source for the
remote handoff candidate. When the latest applicable publish evidence records
a PR URL, that URL is the candidate's remote handoff anchor for later PR and
CI reads.

Local git and remote facts are contextual observations, not replacement
identity. The current worktree branch, HEAD commit, upstream, `origin`, and
derived GitHub owner/repo can help explain warnings or guide manual follow-up,
but they must not be used to guess a PR when publish evidence has not recorded
one.

If publish evidence has no PR URL, the workflow remains in manual handoff:
open or update the PR outside harness, then record publish evidence. Harness
should not infer a PR from the current branch as a substitute for that explicit
handoff record. Future repository customization may define more complex remote
mapping, but the core model stays evidence-first.

### Remote Evidence Refresh

`harness evidence refresh` is the explicit mutating bridge from remote
observation to local evidence. It may use `gh` to read the PR recorded in
publish evidence, classify PR checks as `ci` evidence, and compare the PR's
base and head commits for `sync` freshness or conflicts. Provider approvals and
merge-policy blocking remain separate live facts: pending checks or an
`UNSTABLE` provider state do not make a current branch stale. Refresh must not
create or update PRs,
rerun checks, comment, label, review, merge, or perform other GitHub writes.

Refresh writes evidence only for domains whose remote facts are clear enough.
Fresh sync evidence records the immutable compared base and head commit IDs as
well as their human-facing refs and the publish-recorded PR URL. The sync head
is the current remote candidate identity and may supersede an older
publish-time commit after branch synchronization. Those IDs bind base-aware
review preservation and post-merge recovery to the exact observed candidate
rather than a branch name that may move after squash or rebase land; mismatched
PR identity fails closed.
If checks are unreadable but merge state is clear, refresh may write `sync`
evidence while degrading `ci`; the reverse is also allowed. If a domain is
unavailable, ambiguous, or unsupported, refresh should leave that evidence
domain untouched and guide the controller back to manual
`harness evidence submit`.

`harness status` remains a read-only snapshot resolver. Status may recommend a
refresh or manual evidence submit, but it must not append evidence merely
because it can observe remote facts.

## Reopen Rules

`harness reopen` is the mechanical reversal of archive-time assumptions and
requires an explicit mode:

- `finalize-fix`
  - reopened work stays in finalize-scope repair
- `new-step`
  - reopened work must be represented by a new unfinished step rather than
    being smuggled into prior completed steps

Reopen must preserve audit history:

- do not blank archive-time wording back to empty
- replace reopen-sensitive summaries with explicit update-required placeholders
- keep it obvious that the plan was once archived and is no longer current

When reopen mode is `new-step`, the controller should add the new step after
reopen and continue execution at that new step's `implement` node. Once that
first reopened step has been added, the `new-step` requirement is considered
consumed: later finalize-time findings should repair the latest reopened work
or resume finalize-scope repair instead of forcing another new unfinished step
by default. Until that first new unfinished step exists, status remains in
`execution/finalize/fix` and should keep prompting for the new step rather than
pretending implementation has already resumed.

## Commits and Nodes

Git commits are workflow guidance, not state transitions.

- `delta` review must anchor to a real git commit.
- A small reviewable commit should exist before any intentionally started
  step-bound `delta` review so that review has a durable starting point.
- A meaningful review-driven repair that may need later `delta` review should
  establish another small anchor commit before the fresh review round starts.
- Archive readiness still requires the archived tracked move to be committed
  and pushed before publish, CI, and merge-approval work can be treated as
  complete.

Commit boundaries can support reviewability and handoff, but they do not
change `current_node` by themselves.

## Land Rules

Land is explicit and two-stage:

- `harness land --pr <url> [--commit <sha>]`
  - records merge confirmation and enters `land`
- `harness land complete`
  - records required post-merge bookkeeping completion and restores `idle`

The PR URL is required for land entry in v0.2. Commit SHA is optional because
merge strategy may produce a different landed commit shape across merge-commit,
squash, or rebase flows. Publish evidence should retain the immutable feature
commit so land can validate that reviewed candidate directly after squash or
rebase changes ancestry. Merge confirmation is recorded before local checkout
or branch cleanup.

## Status Rendering

`harness status` should render:

- the resolved `current_node`
- one concise summary
- selected supporting facts
- concrete next actions ordered by likely workflow need

Status output should explain where the controller is now and what kind of work
that node implies. It should not recreate v0.1 by reintroducing parallel
top-level lifecycle fields.
