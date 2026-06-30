# CLI Contract

## Purpose

`easyharness` is a CLI for agents first. The command surface should help an
agent decide what to do next, not dump long raw logs and force the model to
reconstruct workflow state from scratch. The public project name is
`easyharness`, while the executable name remains `harness`.

This document defines the normative v0.2 CLI contract. The command surface and
JSON envelopes described here assume the canonical-node runtime model from
[State Model](./state-model.md) and the exact transition matrix from
[State Transitions](./state-transitions.md).

The prose in this spec remains normative for command purpose, workflow intent,
and compatibility boundaries. Field-level JSON structure for the current
command outputs and inputs lives in the checked-in schema registry at
[`schema/index.json`](../../schema/index.json), sourced from the Go-owned
contract module under `internal/contracts` and explained at
[Contract Registry](./contract.md).

Bootstrap resource semantics, ownership rules, and support boundaries live in
[Bootstrap Install](./bootstrap-install.md). This CLI contract references that
spec instead of duplicating the bootstrap details here.

## Command Surface

The current command surface is:

- `harness plan template`
- `harness plan lint`
- `harness plan approve`
- `harness repo init`
- `harness repo skills install`
- `harness repo skills uninstall`
- `harness repo instructions install`
- `harness repo instructions uninstall`
- `harness repo config init`
- `harness repo config refresh`
- `harness repo config get <key>`
- `harness repo config list [prefix]`
- `harness execute start`
- `harness evidence submit`
- `harness evidence refresh`
- `harness status`
- `harness help [topic ...]`
- `harness dashboard`
- `harness ui`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness review dimensions list`
- `harness review dimensions instructions <name>`
- `harness archive`
- `harness reopen --mode <finalize-fix|new-step>`
- `harness land --pr <url> [--commit <sha>]`
- `harness land complete`

The root CLI also exposes one debug-oriented flag outside that stateful
workflow surface:

- `harness --version`

## Design Principles

### Agent-Friendly by Default

Stateful commands must return:

- a concise summary
- the current durable state
- the current step when it can be inferred
- key artifact paths or identifiers when they are useful
- recommended `next_actions`

They must not default to dumping long raw logs to stdout.

### JSON-First for Stateful Commands

Commands that inspect or mutate workflow state should default to a stable JSON
envelope.

Raw command output, subprocess logs, and verbose diagnostics belong behind an
explicit verbose or debug mode.

Commands that primarily render content, such as `harness plan template`, may
default to markdown or plain text instead of the JSON envelope.

`harness --version` is a JSON-first exception: it returns machine-readable
binary identity data, but it does not use the shared workflow-state envelope
because it is a binary probe rather than a workflow-state command.

`harness dashboard` and `harness ui` are plain-text exceptions because they
start the local read-only UI server rather than returning a workflow-state
JSON envelope.

`harness help [topic ...]` is a plain-text exception because it renders
agent-facing product guidance rather than workflow state. Its topic contract
lives in [Help Topics](./help.md).

The bootstrap commands described in [Bootstrap Install](./bootstrap-install.md)
are JSON-first, but they may omit workflow `state` because they manage
bootstrap assets rather than the tracked plan lifecycle.

The lifecycle commands `harness execute start`, `harness archive`,
`harness reopen`, `harness land`, and `harness land complete` are not special
legacy exceptions. They should use the same v0.2 envelope vocabulary as
`harness status`, centered on `state.current_node` plus concise `facts`,
transition-relevant `artifacts`, and stable `next_actions`.

### Help Must Be Actionable

Every command must have complete `--help` text that explains:

- what the command is for
- required inputs
- key side effects
- common next steps

Skills may refer to `harness --help` or `harness <subcommand> --help`, but the
CLI should remain understandable without repository-specific prompt text.

`harness help` is separate from command syntax help. The detailed topic
contract, topic asset rules, and generated subtopic behavior live in
[Help Topics](./help.md).

### Crash-Safe Runstate Writes

Commands that rewrite CLI-owned JSON runstate must protect those files against
interrupted or overlapping writes.

- write the current-plan pointer and any plan-local `state.json` under the
  local runtime root resolved by `harness repo config get paths.local_runtime`
  through atomic replacement in the destination directory
- acquire a shared per-plan state-mutation lock before loading and rewriting
  the resolved local runtime root's `plans/<plan-stem>/state.json`
- fail with a clear contention error when that state lock is already held
  instead of waiting silently or risking a stale overwrite

### Snapshot Reads and Checkpoint Settle

Read-model services are side-effect-free snapshots as defined in
[State Model](./state-model.md). They must not create mutation locks, rewrite
workflow state, append timeline events, or refresh the watchlist merely because
a UI/API/dashboard resource was polled.

The CLI command boundary may add extra checkpoint behavior on top of a pure
snapshot. `harness status` is the primary agent-facing checkpoint, so it may
wait briefly for a currently held state mutation lock to release before
resolving and returning the snapshot. If the lock remains held after the
bounded wait, `harness status` should return a clear contention result instead
of reading a likely in-flight state.

The status settle check must be passive and non-destructive. It must not create
`.state-mutation.lock` when the file is absent, must not use the ordinary
mutation-lock acquisition helper as its probe, and must not hold any mutation
lock while resolving the status snapshot. After the bounded settle check
completes, `harness status` should call the same pure snapshot resolver used by
UI/API read surfaces.

This wait rule does not apply to ordinary mutation commands. Commands that
mutate workflow state should continue to fail fast on mutation-lock contention
unless their own command contract explicitly defines a different behavior.

## Shared Output Envelope

Stateful commands share a common JSON envelope vocabulary, but not every
stateful command returns every field. Commands that report workflow position
should return an envelope shaped like:

```json
{
  "ok": true,
  "command": "status",
  "summary": "Plan is executing Step 3 and nothing is currently blocking continued work.",
  "state": {
    "current_node": "execution/step-3/implement"
  },
  "facts": {
    "current_step": "Step 3: Implement local state and harness status"
  },
  "artifacts": {
    "plan_path": "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md"
  },
  "next_actions": [
    {
      "command": null,
      "description": "Continue Step 3 implementation; no blocking review or CI artifact is currently active."
    },
    {
      "command": "harness review aggregate --round <round-id>",
      "description": "Run this once all reviewer submissions for the current round are present."
    }
  ],
  "warnings": []
}
```

### Required Fields

- `ok`
- `command`
- `summary`
- `next_actions`

`state` is required for commands that report workflow position, such as
`harness status`. Commands whose job is bootstrap, review-orchestration
artifacts, or append-only evidence recording may omit `state` when they do not
need to report a workflow-position payload.

### Common Optional Fields

- `artifacts`
- `blockers`
- `warnings`
- `errors`

When present, `state` should describe post-command state for mutating commands
and current state for read-only stateful commands.

`artifacts` is optional and command-specific. Omit it when there are no stable
artifact paths or IDs worth returning.

`plan_path` may point to a tracked active plan under the active plan root
resolved by `harness repo config get paths.plans.active`, a tracked standard
archive under the archived plan root resolved by
`harness repo config get paths.plans.archived`, or a lightweight local archive
under the local runtime root resolved by
`harness repo config get paths.local_runtime`.

When a matching `supplements/<plan-stem>/` directory exists for that markdown
plan, commands may also surface it through command-specific `artifacts`
without changing the markdown path's role as the primary plan handle.

`next_actions` should be short, concrete, non-empty, and ordered from the most
likely next step to less common alternatives.

For `harness status` specifically:

- use `next_actions` for ordinary workflow guidance such as continuing the
  current step, starting routine review, or aggregating the active round
- use `warnings` for recoverable ambiguity or workflow-discipline reminders
  that should not by themselves change `state.current_node`
- when the worktree is `idle`, `warnings` plus an optional `next_action` may
  also carry a non-blocking agent reminder that the default repo bootstrap
  assets are stale relative to the running binary; this reminder must not alter
  workflow state or block later execution
- avoid heuristic warnings for "the current slice may now be reviewable"; keep
  that kind of prompt in ordinary `next_actions`

## Status State Contract

The v0.2 CLI uses one canonical runtime state field:

- `state.current_node`
  - required for stateful commands that report workflow position
  - examples: `plan`, `execution/step-2/implement`,
    `execution/finalize/publish`, `land`, `idle`

`facts` is optional and should carry only selected, high-signal fields that
help explain the node:

- `current_step`
- `revision`
- `reopen_mode`
- `review_kind`
- `review_trigger`
  - optional derived label such as `step_closeout` or `pre_archive`
- `review_title`
  - optional derived human-readable review title
- `review_status`
- `archive_blocker_count`
- `evidence`
  - optional archived-candidate evidence group
  - `recorded`
    - durable local evidence that remains authoritative for workflow
      progression
    - `publish.status` and `publish.pr_url`
    - `ci.status`
    - `sync.status`
  - `remote`
    - compact, non-authoritative live observation of the recorded PR handoff
      target
    - `observation`: `complete`, `partial`, or `unavailable`
    - `assessment`: workflow meaning such as `matches_recorded`,
      `refresh_available`, `wait_for_remote`, `repair_remote`,
      `manual_evidence_required`, or `candidate_invalidated`
    - `message`: concise human-readable explanation
    - optional minimal `pr.state`/`pr.draft`, `ci.status`, `sync.status`, and
      degradation codes
- `land_pr_url`
- `land_commit`

`artifacts` may include stable pointers such as:

- `plan_path`
- `supplements_path`
- `review_round_id`
- `review_slots` for an in-flight active review round
- `project_root` only when a contention or error result needs an absolute
  workspace anchor
- `last_landed_at`

Legacy v0.1 fields are not part of the `harness status` contract and must not
be emitted by `harness status`:

- `plan_status`
- `lifecycle`
- `step`
- `step_state`
- `handoff_state`
- `worktree_state`

When a current step exists, `harness status` should infer it from the first
unfinished plan step and return it as `facts.current_step`.
When the current plan is `lightweight`, status should also surface the
repo-visible breadcrumb requirement through the summary or `next_actions`
before the candidate is treated as ready to wait for merge approval.

## Command Contracts

### Bootstrap Commands

Purpose:

- bootstrap or refresh repo/user instructions, skill packages, and repo config
  manifests without mutating tracked plan lifecycle state

Contract:

- `harness repo init` is the quick-start repo resource entrypoint
- `harness repo skills ...`, `harness repo instructions ...`, and
  `harness repo config ...` provide granular resource commands
- bootstrap resource semantics, ownership/version rules, and support boundaries
  are defined in [Bootstrap Install](./bootstrap-install.md)
- bootstrap commands share a JSON result envelope documented by the checked-in
  contract registry and may omit workflow `state`
- read-only repo config query commands are plain-text exceptions:
  `harness repo config get <key>` prints one resolved scalar value, while
  `harness repo config list [prefix]` prints resolved `key=value` leaf entries
  in deterministic order
- `harness repo config refresh --diff` is a narrow plain-text preview
  exception for the refresh mutation command: it prints the unified diff that
  refresh would apply, prints empty stdout when the file is already canonical,
  and does not write `.harness/config.yaml` or embed diff text in the bootstrap
  JSON result envelope

Recommended next action:

- for bootstrap commands that support `--dry-run`, rerun without `--dry-run`
  to apply a previewed bootstrap change
- after reviewing `harness repo config refresh --diff`, rerun
  `harness repo config refresh` to apply the previewed canonical config rewrite
- open the target instructions file or skills directory to review the installed
  contract

### `harness dashboard`

Purpose:

- start the local read-only harness dashboard home for the current machine

Contract:

- keep the UI read-only; it must not mutate harness state or trigger workflow
  commands directly in the first slice
- serve a self-contained local web application from the `harness` binary
  rather than requiring a separate frontend runtime at execution time
- support a small local-server flag surface:
  - `--host`
  - `--port`
  - `--no-open`
- default to binding a local interface and opening the browser automatically
  unless `--no-open` is set
- expose a resource-first read-only API surface for the embedded app instead
  of introducing a second hidden product-only state store
- open `/dashboard` as the canonical home route
- render the machine-local watchlist as one dashboard-owned surface and route
  workspace detail under `/workspace/<workspace_key>/...`

Recommended next action:

- open the local dashboard in a browser and inspect watched workspaces
- use `--no-open` or a fixed `--port` when debugging or automating the local
  server

### `harness ui`

Purpose:

- start the local read-only harness UI for the current repository through the
  dashboard-owned route family

Contract:

- keep the UI read-only; it must not mutate tracked workflow state or trigger
  workflow commands directly in the first slice
- support the same small local-server flag surface as `harness dashboard`:
  - `--host`
  - `--port`
  - `--no-open`
- default to binding a local interface and opening the browser automatically
  unless `--no-open` is set
- retain `harness ui` as a quiet compatibility entrypoint in the first
  dashboard slice instead of printing a deprecation warning immediately
- touch the current workdir into the machine-local watchlist before opening
  the matching `/workspace/<workspace_key>/status` route so the dashboard-owned
  workspace detail surface remains canonical
- render the current worktree's harness state through the same UI family as
  `harness dashboard`; it must not fork a competing interpretation of workflow
  state from the CLI
- expose `Status`, `Review`, and `Timeline` through real read-only resources
- keep timeline data grounded in command-owned runtime artifacts rather than
  reconstructing history from ad hoc client-side state

Recommended next action:

- open the local UI in a browser and inspect the current workspace `Status`
  view
- use `--no-open` or a fixed `--port` when debugging or automating the local
  server

### `harness plan template`

Purpose:

- render the canonical plan template with seeded metadata

Contract:

- use [the packaged template asset](../../assets/templates/plan-template.md) as
  the canonical template source
- print the rendered template to stdout by default
- optionally support writing directly to a target path
- support a lightweight authoring mode such as `--lightweight`
- support enough parameters to seed title, date, source metadata, and the
  required `size` field
- when only a date is provided, preserve the current local time-of-day on that
  date instead of silently forcing `created_at` to local midnight
- seed `template_version` from the packaged asset so generated plans record the
  schema/template version they started from
- avoid introducing a second handwritten template source of truth inside code
- when the caller does not provide a size, keep the required `size` field
  explicit in the rendered template rather than silently defaulting it behind
  the author's back
- in lightweight mode, seed `workflow_profile: lightweight`, a shorter
  single-step low-risk authoring shape, and guidance that the active plan
  still lives under the active plan root resolved by
  `harness repo config get paths.plans.active` while the archive goes to the
  local lightweight archive path
- lightweight authoring must only be available for `size: XXS`; command UX may
  enforce that either by requiring an explicit `XXS` size or by rendering the
  lightweight template with explicit `size: XXS`
- in standard mode, preserve current behavior when `workflow_profile` is
  omitted
- goal-oriented authoring support belongs to the goal-oriented template slice;
  when added, it should seed `workflow_profile: goal_oriented` and the
  required concepts from [Goal-Oriented Workflow](./goal-oriented-workflow.md)
  without changing ordinary standard or lightweight authoring; until that
  implementation lands, `workflow_profile: goal_oriented` is a reserved
  contract value rather than a lint-valid template output

The template asset belongs to the harness version, not to the user's tracked
plan history. Upgrading the harness may upgrade the generated template for new
plans without rewriting historical plans already checked into the repository.

For a Go implementation, the template should be embedded into the binary rather
than loaded from the user's current working directory at runtime. The source
file may live under `assets/`, `internal/templates/`, or a similar package-local
path in the harness source tree, but the built CLI should not depend on that
source path existing in the consumer repository.

One straightforward Go layout would be:

- `internal/templates/`
  - holds the canonical template source file
- `internal/templates/embed.go`
  - exposes an embedded `fs.FS` or string via `//go:embed`
- `internal/plan/`
  - owns rendering and linting logic against that embedded asset

Recommended next action:

- edit the generated plan content
- run `harness plan lint`

### `harness plan lint`

Purpose:

- validate a plan against the plan schema

Contract:

- stop with targeted structural errors instead of guessing or silently fixing
  invalid plan data
- report issues in a compact machine-readable form
- distinguish active-plan errors from archived-plan errors
- validate path/profile compatibility for tracked active plans, tracked
  standard archives, and lightweight local archived plans
- validate supported `template_version` values without invalidating older
  historical plans created by earlier harness versions
- reject malformed plan filenames and malformed `### Step N: ...` headings

Recommended next action:

- fix the listed fields or sections
- rerun lint

### `harness status`

Purpose:

- summarize the current plan and local execution state in the current worktree

This is the primary resume and handoff command. Another agent, a compacted
session, or a human should be able to run `harness status` and quickly
understand what is happening now and what to do next.

Contract:

- detect the current plan artifact, whether it is a tracked active plan, a
  tracked standard archive, or a lightweight local archive
- before resolving the status snapshot for a current plan, briefly wait for an
  actively held state mutation lock to settle; if the lock remains held beyond
  the bounded wait, return a clear local-mutation-in-progress result
- perform that settle wait with a passive, non-destructive advisory-lock check
  that does not create a missing lock file and does not hold any mutation lock
  while resolving the snapshot
- resolve the canonical `state.current_node` from the current plan,
  execute-start milestones, review artifacts, append-only evidence, reopen
  milestones, archive state, and land milestones
- return pure v0.2 JSON centered on `state.current_node`, selected `facts`,
  `artifacts`, `summary`, and `next_actions`
- keep the underlying status snapshot read-only: the snapshot resolver must
  not acquire mutation locks, rewrite `state.json`, append timeline events, or
  touch watchlist recency
- never emit legacy v0.1 fields such as `lifecycle`, `step_state`, or
  `handoff_state`
- surface aggregated review failures as a concrete repair signal rather than
  falling back to a generic step summary
- if review metadata cannot be recovered safely, degrade conservatively rather
  than writing a fallback cache answer into local state
- when an active review round is in flight, surface the reviewer-owned slot
  handles another controller would need to resume reviewer orchestration
  without reopening internal control files
- once all steps and acceptance criteria are complete, surface archive blockers
  early through a structured `blockers` list plus repair-first next actions
  instead of making the controller learn them only from `harness archive`
- surface stale or unknown remote freshness as warnings and next actions rather
  than as a derived state layer
- treat recorded publish evidence as the authoritative remote handoff identity:
  when the latest publish evidence has a PR URL, later read-only remote
  observation should anchor on that PR URL; when it does not, status should
  steer the controller to open or update the PR and record publish evidence
  rather than guessing a PR from the current branch
- for archived publish and await-merge handoff states, default to live
  read-only observation of the recorded PR URL when that URL is available and
  supported
- surface live remote observation under `facts.evidence.remote` as a compact,
  non-authoritative hint containing observation completeness, workflow
  assessment, a concise message, minimal PR state, CI/sync statuses that
  refresh would record, and compact degradation facts
- keep live remote observation read-only and non-authoritative: `gh` may be
  used to observe a recorded PR URL when available, but missing `gh`, missing
  auth, network/API failure, an unreadable PR, or mismatched local git context
  should degrade to warnings or manual evidence guidance instead of failing
  local workflow state
- never advance `state.current_node` based on live remote observation alone;
  publish and await-merge progression remains driven by durable local publish,
  CI, and sync evidence
- never append evidence from `harness status`; automatic remote-to-evidence
  updates belong to the explicit `harness evidence refresh` command
- when recorded publish evidence has a PR URL and remote observation says the
  recorded PR facts are refreshable, include `harness evidence refresh` in
  status next actions while preserving manual `harness evidence submit`
  fallback commands
- when remote observation says to wait, repair, use manual evidence, or treat
  the candidate as invalidated, do not also include an immediate
  `harness evidence refresh` command in status next actions
- if no current plan is active but the current-plan pointer under the local
  runtime root resolved by `harness repo config get paths.local_runtime`
  records a landed candidate, return
  `state.current_node: idle` with landed context in `artifacts`
- when the current plan uses the lightweight profile, remind the controller to
  leave the agreed repo-visible breadcrumb, such as a readable PR body merge
  memo explaining what changed, why the branch is mergeable, and why the
  lightweight path was used
- when the current plan uses the goal-oriented profile, future status guidance
  may surface advisory checkpoint-round next actions from
  [Goal-Oriented Workflow](./goal-oriented-workflow.md), but status must not
  infer or mutate `state.current_node` from checkpoint markdown and must not
  silently replace `harness plan lint`; concrete status support belongs to the
  goal-oriented status implementation slice
- return recommended next actions for both "continue work" and "wait/observe"
  situations
- if an already completed earlier step is missing review-complete closeout,
  keep the current node stable, surface a warning, and put the earliest repair
  guidance first in `next_actions`
- if unreadable historical review metadata cannot be mapped back to a tracked
  step, keep the current node stable, preserve a conservative warning, and
  steer the controller toward repairing artifacts or rerunning the relevant
  step-closeout review instead of silently trusting older clean passes
- treat `state.json` as a control-plane artifact for cross-command runtime
  state, not as a cache of the latest resolved node or evidence pointers

Recommended next action examples:

- continue the current step
- start step-closeout review before marking a completed step done
- update step-local `Execution Notes` or `Review Notes` after a review or
  step closeout
- update the plan if scope changed
- run review aggregation
- refresh remote state if the latest sync evidence is stale or missing
- wait for CI
- archive the plan
- commit, push, and update the PR after archive before waiting for merge
  approval

### `harness review start`

Purpose:

- begin a deterministic review round without embedding runtime-specific agent
  spawning in the CLI

Contract:

- accept an agent-supplied review spec instead of inventing one inside the CLI
  itself
- create a `round_id`
- require a `kind` of either `delta` or `full`
- accept the review spec via a structured input such as `--spec <path>` or
  stdin
- validate and persist the supplied review spec as CLI-owned round metadata
- normalize each review dimension into a deterministic reviewer slot
- reserve reviewer output paths
- precreate one reviewer-owned folder per slot with a `submission.json`
  skeleton that the reviewer can progressively update during the round
- initialize round dispatch bookkeeping
- when `step` is omitted and the inferred binding would be finalize review,
  reject the request if earlier completed steps still lack review-complete
  closeout and direct the controller toward explicit `step=<i>` repair instead
- when mutating both review artifacts and local state, acquire the review
  mutation lock before the state mutation lock instead of inventing a separate
  acquisition order for this command
- update local `state.json` so `harness status` can surface the active round
- return round metadata plus next actions for the controller agent

The controller agent should only need to know the round ID, repo-facing
`plan_path`, review kind, dimension definitions, any reviewer-owned
`submission.json` paths it must hand to reviewer subagents, and how to invoke
the reviewer skill. It should not need to know the paths or storage names of
CLI-owned internal review-control artifacts.

`harness review start` is still useful even when the agent provides the review
spec because the CLI owns:

- round ID allocation
- review-spec validation and persistence
- deterministic artifact locations
- dispatch and review bookkeeping
- local-state updates for `harness status`

In this contract, the review spec is the command input. The CLI also persists
its own internal review-control artifacts derived from that input plus
CLI-owned fields such as `round_id`, timestamps, and internal bookkeeping
state. Agent-facing command output should expose only the stable handles and
workflow-owned artifact paths the agent actually needs, not the internal
control-artifact locations.

Canonical input shape:

```json
{
  "kind": "delta",
  "anchor_sha": "<base-commit-sha>",
  "review_title": "Check the completed step for state-machine mistakes and handoff clarity.",
  "dimensions": [
    {
      "name": "correctness",
      "instructions": "Run `harness review dimensions instructions correctness` and follow the returned Markdown instruction."
    },
    {
      "name": "agent-ux",
      "instructions": "Run `harness review dimensions instructions agent-ux` and follow the returned Markdown instruction."
    },
    {
      "name": "docs-consistency",
      "instructions": "Run `harness review dimensions instructions docs-consistency` and follow the returned Markdown instruction."
    },
    {
      "name": "migration-compat",
      "instructions": "Review migration compatibility and old-path fallback behavior for this change."
    }
  ]
}
```

Example invocation:

```bash
harness review start --spec /tmp/review-spec.json
```

The command returns JSON describing the created round, the reviewer-owned slot
artifacts, and next actions for the controller agent.

Review-spec semantics:

- `kind`
  - required
  - enum: `delta` or `full`
- `dimensions`
  - required
  - one reviewer slot per normalized dimension
  - catalog-managed dimensions use stable names made from lowercase
    alphanumeric segments separated by single hyphens; for those dimensions the
    name and slot identifier are the same
  - `instructions` is still explicit review-start handoff text; for
    catalog-managed dimensions it should usually be a command to fetch the full
    instruction, while one-off dimensions are first-class review slots and may
    carry direct reviewer guidance
- `anchor_sha`
  - optional for `full`
  - required for `delta`
  - for `delta`, carries the controller-chosen git commit anchor so the
    persisted round metadata records the review starting point durably
- `review_title`
  - optional
  - human-readable review title shown back to the controller and reviewers
- `step`
  - optional 1-based tracked step number
  - usually omitted
  - when present, explicitly binds the round to that tracked step's
    step-closeout review, even if the current execution frontier is already on
    a later step or in finalize repair
  - use this path for earlier-step closeout repair when missing or failed
    closeout evidence needs to be repaired intentionally rather than only
    surfaced as passive warning debt
  - when omitted, `harness review start` infers the binding from workflow state:
    - during `execution/step-<n>/implement`, the round binds to the current step
    - during `execution/finalize/review` or `execution/finalize/fix`, the round binds to finalize review

Agents should not supply structural workflow tags such as `step_closeout` or
`pre_archive`. The CLI owns that inference and persists the bound step or
finalize scope in stored round metadata and local state.

`harness review start` does not resolve dimension references, does not
automatically inject built-in or repo-defined dimensions, and does not force a
controller to use every recommended dimension. Controllers remain responsible
for choosing dimensions and passing an explicit review spec. The catalog is a
source of reusable recommended dimensions, not a closed enum; controllers may
create one-off dimensions with explicit instructions for a particular round.

Explicit `step` binding intentionally re-enters the targeted step's review
loop. If the controller is already executing `step-k` or finalize work and
starts a repair review for earlier `step-i`, status may report
`execution/step-<i>/review` while the round is in flight and
`execution/step-<i>/implement` after a non-pass aggregate. This is distinct
from passive warning debt, where status may keep the later frontier stable
until the controller explicitly starts a repair review for the earlier step.

Round identifiers should be short and plan-local:

- use `review-<NNN>-<kind>`
- examples: `review-001-delta`, `review-002-full`
- keep precise timestamps in stored review metadata rather than
  embedding them in the round ID

If `review_title` is omitted, the CLI fills one in:

- step-bound review defaults to the tracked step title
- finalize `full` review defaults to `Full branch candidate before archive`
- finalize `delta` review defaults to `Branch candidate before archive`

If an explicit earlier-step repair review aggregates with findings, the
follow-up work still belongs to the current candidate, but status should pin
the repaired step as current until that step-closeout debt is resolved by a
later clean repair review or explicit `NO_STEP_REVIEW_NEEDED` closeout.

Dimension-specific reviewer instructions belong in the input review spec.
For catalog-managed dimensions, the spec should usually hand reviewers a
`harness review dimensions instructions <name>` command instead of embedding the
full instruction body. For one-off review slots that are not in the catalog,
the spec may carry explicit reviewer guidance directly. Generic reviewer
behavior, such as "inspect the current diff and submit results through the
harness contract," belongs in the reviewer skill or in command output helpers,
not duplicated in every review spec.

Recommended next action:

- launch reviewer subagents using the runtime's native delegation mechanism
  and have each subagent use the reviewer skill or reviewer prompt that owns
  submission details.

### `harness review dimensions list`

Purpose:

- show controller agents the recommended review dimensions available for the
  current repository without loading long reviewer instructions into the
  controller context

Contract:

- discover built-in dimensions and repo-defined dimensions
- discover repo dimensions from the root resolved by
  `harness repo config get paths.review.dimensions`, defaulting to
  `.harness/review/dimensions`
- repo dimensions are Markdown files with YAML frontmatter containing exactly
  `name` and `description`, followed by the full reviewer instruction body;
  additional frontmatter fields are invalid
- dimension names are stable skill-like identifiers made from lowercase
  alphanumeric segments separated by single hyphens
- repo dimensions override built-in dimensions with the same name
- duplicate repo dimension names are invalid
- return compact JSON metadata only; do not include full instruction bodies

The built-in dimensions are:

- `correctness`
- `tests`
- `docs-consistency`
- `agent-ux`
- `risk-scan`

Canonical output shape:

```json
{
  "ok": true,
  "command": "review dimensions list",
  "summary": "Found 5 review dimensions.",
  "dimensions": [
    {
      "name": "correctness",
      "source": "builtin",
      "description": "Use when reviewing implementation logic, workflow state transitions, command contracts, or negative-path behavior."
    }
  ]
}
```

`source` is limited to:

- `builtin`: packaged easyharness dimension
- `repo`: dimension defined by the current repository

### `harness review dimensions instructions <name>`

Purpose:

- let reviewer agents fetch the full Markdown instruction for their assigned
  dimension without JSON escaping noise

Contract:

- accept one stable dimension name
- resolve the same built-in plus repo-overridden catalog used by
  `harness review dimensions list`
- print only the raw Markdown instruction body on success
- print clear errors and exit non-zero when the name is invalid, unknown, or
  the repo dimension catalog is invalid

This command intentionally does not return JSON on success. Reviewers should
read the raw Markdown and then submit through the existing
`harness review submit` flow.

### `harness review submit`

Purpose:

- record one reviewer result for a specific review round and reviewer slot

This command is primarily for reviewer subagents rather than the main
controller agent.

Contract:

- require `--by <reviewer-name>` as a lightweight reviewer-role cue
- accept the reviewer payload via `--input <path>` or stdin
- validate that the submission matches an expected slot
- treat top-level `summary` and `findings` as the canonical aggregate fields
- allow extra top-level reviewer worklog fields in the submission payload and
  preserve them in the stored submission artifact
- preserve the `--by` value in the stored submission artifact as review
  provenance
- allow each finding to omit `locations` or provide `locations: []string`
  using lightweight repo-relative anchors in one of these forms:
  - `path/to/file.go`
  - `path/to/file.go#L123`
  - `path/to/file.go#L1-L3`
- store the structured reviewer artifact in the round's owned location
- update round dispatch bookkeeping
- stay on the review-artifact mutation path only; reviewer submission should
  not acquire the plan-local state mutation lock
- return a submission receipt plus clear next actions without surfacing
  internal review-control artifact paths

Recommended next action:

- on success, report the receipt to the controller agent and end the reviewer
  thread; a runtime may later reopen the same reviewer for a narrow same-slot
  follow-up for the same tracked step or the same finalize review scope in
  the same revision, but only after the earlier submission was verified and
  only when the slot instructions still materially match; immediate closeout
  is the safe default
- on validation failure, fix the reviewer artifact and resubmit

The main controller agent should not use this command to stand in as its own
reviewer. In Codex, reviewer submission still belongs in a bounded reviewer
subagent thread, with `--by` acting as a role cue rather than a strong
identity assertion.

### `harness review aggregate`

Purpose:

- aggregate a review round into a concise decision surface for the controller
  agent

Contract:

- require `--round <round-id>` to select the round
- reject the request unless `--round` matches the current active review round
  for the executing plan; in the v0.1 single-active-round model, `review
  aggregate` is not a historical backfill or repair command for older rounds
- collect reviewer artifacts
- compute blocking and non-blocking findings
- stop with an error when expected reviewer slots are missing or invalid
- ignore preserved extra top-level reviewer worklog fields when computing the
  decision surface
- write persisted review decision data that captures the review decision surface and
  preserves any finding `locations` verbatim
- when mutating both review artifacts and local state, acquire the review
  mutation lock before the state mutation lock instead of inventing a separate
  acquisition order for this command
- update local `state.json` with the aggregate result, including whether the
  round passed or requested changes
- allow later commands to recover that decision from the persisted review
  decision data when older local state predates the stored `decision` field
- return next actions that depend on the review kind without surfacing
  internal review-control artifact paths or local state files

Recommended next action:

- for a passing `delta` review, continue the current step or mark the step
  complete, then update the step's `Execution Notes` and `Review Notes`
- for a failing `delta` review, fix the current slice and rerun a delta review
- for a passing `full` review, move toward final CI and archive readiness
- for a failing `full` review, fix findings before archive

## Review Sequencing

The CLI contract should assume this review cadence:

- use `delta` review after a completed plan step or after a narrow follow-up fix
- anchor `delta` review to a real git commit chosen by the controller; the
  latest passed-review anchor should be the default start point for a later
  `delta` review
- allow a `full` review to satisfy step closeout when a narrower review would
  be misleading for that completed step
- use `full` review once all planned work appears complete and the branch looks
  like an archive candidate
- if CI failure or conflict resolution creates a narrow, well-bounded change,
  run a `delta` review on that change
- if CI or conflict repair is broad or invalidates the prior full-review scope,
  rerun `full` review before archive

Archive readiness requires:

- a clean `full` review for the initial archive candidate (`revision: 1`)
- a clean review result for later reopened revisions, where a narrow fix may
  use `delta` review instead of forcing another `full` review
- no unresolved active review round
- no unresolved finalize repair work

Post-archive merge readiness additionally requires:

- publish evidence with a PR URL
- CI good enough or explicit `not_applied`
- sync freshness or explicit `not_applied`

The PR URL recorded in publish evidence is the remote handoff anchor for later
read-only PR and CI observation. Local branch, commit, upstream, and remote
repository facts are context only; they may explain why a controller needs to
repair or refresh evidence, but they must not replace the explicit PR URL or
trigger branch-based PR discovery. If no PR URL has been recorded, the
controller should continue the existing manual publish path and record evidence
once the PR or handoff target exists.

When `harness status` observes a recorded PR through `gh`, it must treat `gh`
as an optional read-through provider rather than a hard workflow dependency.
Missing `gh`, missing auth, network or API errors, invalid provider output, and
unreadable PRs should produce compact `facts.evidence.remote` degradation
output or next-action guidance while preserving the local status result and the
manual `harness evidence submit --kind publish|ci|sync` fallback. Live remote
observation is useful for explaining what to do next, but it is not durable
evidence: status must not enter `execution/finalize/await_merge` from passing
remote checks or clean merge state until those facts have been recorded through
local CI and sync evidence, normally by `harness evidence refresh`.

### `harness execute start`

Purpose:

- record the execution-start milestone for the current active plan

Contract:

- require the current plan to be active before recording execution start
- require explicit plan approval to already be recorded before execution can
  start
- reject the command with a clear error when the current active plan still
  lacks recorded approval
- persist the execution-start milestone in plan-local runtime state
- update the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` to point at the active tracked
  plan
- return the shared v0.2 envelope with the post-command
  `state.current_node`, `facts.revision`, transition-relevant `artifacts`, and
  actionable `next_actions`
- avoid emitting a lifecycle-specific state sublanguage separate from
  `current_node`

Recommended next action:

- continue the current tracked step and keep step-local `Execution Notes` and
  `Review Notes` current as implementation proceeds

### `harness plan approve`

Purpose:

- record explicit human approval for the current active tracked plan before
  execution starts

Contract:

- require the current plan to be active
- require `--by human`
- treat this command as a trust-based workflow acknowledgment rather than a
  strong identity check; harness records the approval boundary but does not
  authenticate the actor
- persist approval durably in the tracked plan frontmatter as `approved_at`
- keep approval separate from `harness execute start`; approval records the
  human steering boundary, while execution start records the later execution
  milestone
- return a concise result that steers the agent toward `harness execute start`
  after approval is recorded

Recommended next action:

- start execution with `harness execute start` once the approved plan is ready
  for implementation

### `harness archive`

Purpose:

- freeze the current plan locally for merge handoff

Contract:

- validate that the plan is active and archive-ready
- run the shared archive-readiness evaluation before any tracked-file or local
  state write happens so a failing archive attempt leaves the current candidate
  untouched
- assume the plan's durable summary sections have already been written from the
  current plan plus local artifacts, not reconstructed from agent memory
- require finalize review to be satisfied before archive succeeds
- reject archive while earlier completed steps still lack review-complete
  closeout, even if the latest finalize review is clean
- if the plan still contains `## Deferred Items`, require
  `## Outcome Summary > Follow-Up Issues` to be something other than `NONE`
  before allowing archive to succeed
- reject archive when plan-local state still shows unresolved finalize review
  or archive-closeout blockers for the current candidate
- require plan-local review state to retain the latest review decision, or
  recover it from the latest persisted review decision data for older local
  state, so archive can distinguish a failed aggregated review from a passing one
- require the pre-archive `Archive Summary` to include structured `PR`,
  `Ready`, and `Merge Handoff` lines
- move the plan from its active path to its archived path:
  - active plan root resolved by `harness repo config get paths.plans.active`
    -> archived plan root resolved by
    `harness repo config get paths.plans.archived` for `standard`
  - active plan root resolved by `harness repo config get paths.plans.active`
    -> lightweight archived snapshot root under the local runtime root resolved
    by `harness repo config get paths.local_runtime` for `lightweight`
- when a matching `supplements/<plan-stem>/` directory exists, move it with
  the markdown plan to the corresponding archived root
- for `lightweight`, that archived root is the local snapshot path under the
  resolved local runtime root, not tracked git
- update the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` to the archived plan path
- keep publish, CI, and sync follow-up out of the archive gate; those belong to
  `execution/finalize/publish`
- return the shared v0.2 envelope with `state.current_node` set to the
  post-archive handoff node, concise `facts`, transition artifacts, and
  actionable `next_actions`
- return next actions that explicitly include the profile-appropriate handoff:
  commit and push the archive move for `standard`, or update the repo-visible
  breadcrumb for `lightweight`

Important note:

- `harness archive` changes tracked files locally for every profile because
  the active tracked plan is removed from the active plan root resolved by
  `harness repo config get paths.plans.active`
- the controller agent should commit and push the archive change before
  treating the candidate as truly waiting for merge approval
- the controller agent should also update the agreed repo-visible breadcrumb
  for `lightweight` before treating the candidate as truly waiting for merge
  approval; a PR body breadcrumb should read as a merge memo rather than a raw
  validation command log
- after archive, record publish, CI, and sync observations through
  `harness evidence submit` instead of treating missing evidence as success
- after archive, correctness should not depend on archived supplements still
  being present; anything the repository must keep relying on should already be
  absorbed into formal tracked locations
- PR checks may rerun on that archive commit; if new feedback or check failures
  appear, use `harness reopen --mode <finalize-fix|new-step>`
- merge actor, merge timestamp, and other land-only notes should go to PR
  comments or remote history rather than back into the archived plan
- if deferred items exist, the controller agent should replace `NONE` in
  `Follow-Up Issues` with durable handoff details before archive completes

Recommended next action:

- create or verify durable follow-up notes for deferred work
- commit and push the archived plan for `standard`, or update the repo-visible
  breadcrumb for `lightweight`
- wait for post-archive CI or human merge approval once publish, CI, and sync
  evidence move the candidate into `execution/finalize/await_merge`
- reopen with `harness reopen --mode finalize-fix` for narrow repair or
  `harness reopen --mode new-step` when the invalidation deserves a new step

### `harness reopen`

Purpose:

- restore an archived plan to active execution

Contract:

- move the plan from its archived path back to the matching active path
- when a matching `supplements/<plan-stem>/` directory exists, move it with
  the markdown plan back to the active root
- increment command-owned revision state
- require an explicit mode such as `finalize-fix` or `new-step`
- preserve archive audit history via explicit update-required placeholders
- clear stale review and land control-plane signals from the prior archived
  candidate
- update the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` back to the active path
- return the shared v0.2 envelope with the reopened post-command node,
  `facts.revision`, `facts.reopen_mode`, transition artifacts, and actionable
  `next_actions`
- return next actions that help the next agent resume work

Recommended next action:

- review the feedback or remote change that caused reopen
- update plan content if scope or acceptance criteria changed
- continue finalize repair for `finalize-fix`, or add a new unfinished step
  before resuming implementation for `new-step`

### `harness evidence submit`

Purpose:

- record append-only publish, CI, or sync evidence for the current archived
  candidate

Contract:

- require the current plan to already be archived before accepting evidence
- support `--kind <ci|publish|sync>` with JSON payloads documented in
  `--help`
- write a timestamped evidence artifact under the local runtime root resolved
  by `harness repo config get paths.local_runtime`, inside
  `plans/<plan-stem>/evidence/<kind>/`
- let later status and land-readiness checks discover the latest evidence
  directly from append-only evidence artifacts instead of storing a pointer in
  `state.json`
- preserve trajectory by never editing older evidence artifacts in place
- accept explicit `not_applied` payloads when a domain truly does not apply

### `harness evidence refresh`

Purpose:

- observe the recorded PR URL from latest publish evidence and append derived
  `ci` and `sync` evidence for the current archived candidate

Contract:

- require the current plan to already be archived before refreshing evidence
- require latest publish evidence to record a supported PR URL; if it does not,
  do not infer a PR from the current branch and steer the controller to
  publish or manual evidence submit
- use `gh` only for read operations against the recorded PR URL
- classify readable PR checks into `ci` evidence: passing checks become
  `success`, pending checks become `pending`, and failing or cancelled checks
  become `failed`
- classify readable PR merge state into `sync` evidence: clean/current state
  becomes `fresh`, stale/behind/blocked/unknown-but-readable state becomes
  `stale`, and conflict state becomes `conflicted`
- append evidence independently per domain: if CI facts are clear but sync
  facts degrade, write only CI evidence and return clear degraded sync
  guidance, and vice versa
- never write misleading success evidence when `gh`, auth, network/API access,
  provider output, checks, or merge state are unavailable
- preserve `harness evidence submit --kind publish|ci|sync` as the manual
  fallback for every degraded refresh path
- never create or update PRs, rerun checks, comment, label, review, merge, or
  perform GitHub writes

### `harness land --pr <url> [--commit <sha>]`

Purpose:

- record merge confirmation for the current archived candidate and enter land
  for the required post-merge bookkeeping

Contract:

- require the current plan artifact to still be the archived candidate
- require `--pr <url>` and optionally accept `--commit <sha>`
- validate that publish, CI, and sync evidence make the candidate merge-ready
- record merge confirmation in plan-local runtime state
- leave archived plan content untouched; this is a local-state milestone only
- return the shared v0.2 envelope with the `land` post-command node and any
  relevant `land_pr_url` / `land_commit` facts
- return next actions that guide the required post-merge bookkeeping

### `harness land complete`

Purpose:

- record required post-merge bookkeeping completion and restore idle worktree
  state

Contract:

- require prior `harness land --pr <url>` for the same archived candidate
- persist local completion metadata in plan-local runtime state
- rewrite the current-plan pointer under the local runtime root resolved by
  `harness repo config get paths.local_runtime` so `plan_path` is cleared
- record `last_landed_plan_path` and `last_landed_at` for worktree handoff
- leave archived plan content untouched; this is local-state cleanup only
- return the shared v0.2 envelope with `state.current_node: idle`,
  `facts.revision`, and the artifact pointers needed for handoff confirmation
- return next actions that guide the worktree back to idle or on to the next
  slice

### `harness --version`

Purpose:

- print JSON build information for the running harness binary

Contract:

- remain a root-level flag rather than a workflow subcommand
- return JSON rather than the shared workflow-state envelope
- report the running binary's execution mode and build commit when that commit
  is available from the binary metadata
- report the release-facing identity subset by default:
  `version`, `mode`, `commit`, with optional `go_version` and `build_time`
  when available from the binary metadata
- allow dev binaries to expose additional debug-oriented fields such as
  `modified` and the resolved executable `path`
- omit unavailable metadata rather than fabricating placeholders

## Review Runtime Boundary

The CLI does not own subagent spawning.

The controller agent decides how to launch reviewer subagents. In Codex, that
means using `spawn_agent` rather than trying to do reviewer work in the main
agent thread. The reviewer skill or reviewer prompt should own the details of
calling `harness review submit`.

Codex should still default to closing reviewer agents after each verified
submission. If a later narrow follow-up round keeps the same slot and
materially the same instructions for the same tracked step or the same
finalize review scope in the same revision, the controller may reopen that
previously closed reviewer with `resume_agent` instead of spawning fresh.
Moving to a different tracked step, moving from step review to finalize
review, changing the review scope because of reopen or a new revision, broad
follow-up, changed slot ownership, invalid earlier submissions, or any
situation where a clean reread is safer should stay on fresh `spawn_agent`
reviewer threads.

The CLI only owns deterministic local contracts:

- round-metadata persistence
- output paths
- submission validation
- aggregation
- audit trail

## Deferred Commands

No additional user-facing command is committed in this spec yet beyond the
surface listed above.
