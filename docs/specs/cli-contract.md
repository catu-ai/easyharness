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
- `harness review start [--full]`
- `harness review submit --round <round-id> --by <reviewer> [--input <path>]`
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
      "command": "harness review submit --round <round-id> --by integrated --input <path>",
      "description": "Have the independent integrated reviewer submit its complete judgment for the active finalize round."
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
  current step, starting mandatory finalize review, or handing the active
  round to its integrated reviewer
- use `warnings` for recoverable ambiguity or workflow-discipline reminders
  that should not by themselves change `state.current_node`
- in active, archived, and idle workflow states, `facts.managed_resources`,
  `warnings`, and an optional `next_action` may carry a non-blocking agent
  reminder that the default repo bootstrap assets are stale relative to the
  running binary; inspection must be read-only and must not alter workflow
  state, displace the active workflow action, or block later execution
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
- `review_status`
- `reviewed_head_sha`
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
      `manual_evidence_required`, `candidate_invalidated`, or
      `merged_pending_land`
    - `message`: concise human-readable explanation
    - optional minimal `pr.state`/`pr.draft`, `ci.status`, `sync.status`, and
      degradation codes
- `managed_resources`
  - present only when default repo managed assets are stale relative to the
    running binary
  - reports the `codex` agent, stale instructions, and stale, missing, or extra
    managed skill package names without changing lifecycle state
- `land_pr_url`
- `land_commit`

`artifacts` may include stable pointers such as:

- `plan_path`
- `supplements_path`
- `review_round_id`
- `review_submission_path` for an in-flight active review round
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
- goal-oriented authoring is not part of this command contract; its future
  direction is deferred to `v0.7.0` and described only as a non-normative note
  in [Goal-Oriented Workflow](./goal-oriented-workflow.md)

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
- surface review findings as a concrete finalize-repair signal rather than
  falling back to a generic step summary
- if review metadata cannot be recovered safely, degrade conservatively rather
  than writing a fallback cache answer into local state
- when an active review round is in flight, surface the round ID, reviewed
  commit, integrated reviewer handoff, and submission path another controller
  needs without reopening internal control files
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
- return recommended next actions for both "continue work" and "wait/observe"
  situations
- if an active finalize round or unresolved blocking finding exists, put its
  reviewer or repair guidance first in `next_actions`
- if finalize review metadata cannot be recovered safely, keep the current
  node conservative and steer the controller toward repairing the artifacts
  or establishing a new full review root
- treat `state.json` as a control-plane artifact for cross-command runtime
  state, not as a cache of the latest resolved node or evidence pointers

Recommended next action examples:

- continue the current step
- complete the current outcome and its concise check before marking a step done
- update the plan if scope changed
- start the mandatory finalize review after the complete candidate is committed
- have the integrated reviewer submit its complete judgment
- refresh remote state if the latest sync evidence is stale or missing
- wait for CI
- archive the plan
- commit, push, and update the PR after archive before waiting for merge
  approval

### `harness review start [--full]`

Purpose:

- start the mandatory independent review of the complete committed candidate

Contract:

- run only in finalize review or finalize repair; steps are execution progress
  boundaries and cannot bind formal review rounds
- require every tracked step and acceptance criterion to be complete before
  creating review state
- require a Git-backed candidate with a clean worktree and committed `HEAD`,
  ignoring only command-owned runtime artifacts, and capture that immutable
  commit as `reviewed_head_sha`
- create exactly one fixed integrated reviewer assignment; the command accepts
  no review spec, assignment, slot, dimension, specialist, step, or worklog
  configuration
- automatically attach the fixed complete rubric and every item from the
  tracked plan's `Review Focus` section to the reviewer handoff
- make the fixed rubric cover correctness, acceptance criteria, success and
  failure paths, state and permission behavior, tests and evidence,
  code/schema/documentation/agent contracts, scope, deferred work, and residual
  risk
- require the reviewer to address every fixed rubric area and every plan review
  focus item with concise evidence or an explicit not-applicable rationale;
  removing dimension orchestration must not remove review coverage
- allocate a short plan-local `review-<NNN>-<kind>` round ID and one
  reviewer-owned submission path
- default to `full` when establishing finalize coverage; during finalize
  repair, infer a linked `delta` only when the current coverage tip, repair
  revision, reviewed-head ancestry, and targeted finding IDs identify one
  unambiguous narrow repair
- fail safely instead of inventing a delta when that link cannot be inferred;
  `--full` explicitly establishes a replacement full root after a material
  design, scope, or risk change
- preserve the active-round mutation lock and, when both review artifacts and
  workflow state change, acquire the review lock before the state lock
- persist round kind, captured head, plan identity and revision, automatic
  reviewer handoff, and for a linked delta its parent round, anchor, and
  targeted finding IDs
- return the round ID, reviewed head, reviewer handoff, submission path, and a
  next action to launch the one independent integrated reviewer

The CLI owns deterministic review metadata and coverage bookkeeping, but it
does not spawn the reviewer or choose model/runtime topology.

### `harness review submit --round <round-id> --by <reviewer> [--input <path>]`

Purpose:

- record the integrated reviewer's complete judgment and finish the active
  review round atomically

This command belongs to the independent reviewer subagent, not the controller.

Contract:

- require the active round ID and a non-empty reviewer name
- create the returned submission path as a directly editable input skeleton
  containing only reviewer-owned input fields; round, slot, role, reviewer, and
  submission-time metadata are command-owned and must not appear in that
  pre-submit skeleton
- accept the edited generated skeleton directly through `--input <path>`, or
  accept an equivalent structured submission through another path or stdin;
  validate the reviewer input before adding command-owned artifact metadata
- accept one complete integrated judgment containing a concise summary,
  findings, fixed-rubric and plan-focus coverage, and linked-finding
  resolutions when the round is a repair delta
- reject slots, dimensions, specialist submissions, progressive worklogs, and
  partial submissions
- preserve finding severity, evidence, and lightweight repo-relative locations;
  `blocker` and `important` findings block progression, while `minor`
  findings remain visible without forcing repair
- verify under the review mutation lock that the round is active, the payload
  is valid, and current clean committed `HEAD` still equals
  `reviewed_head_sha`
- reject any second or concurrent-losing submission after the round has a
  completed decision; completed rounds and their stored artifacts are
  immutable, and later candidate work requires a new review round
- for a linked delta, require explicit resolution of every targeted parent
  finding, preserve any unresolved parent findings, add new findings, and
  reject inconsistent anchors, ancestry, revisions, or finding references
- in one transaction, persist the reviewer submission, derive the round
  decision, update the continuous full-plus-delta coverage chain, and update
  workflow state; a failure must leave no partial submission, decision, or
  coverage update
- expose a concise receipt and next action: repair linked blocking findings,
  start a new full review after a material change, or continue toward archive
  after a passing covered candidate

There is no separate aggregate command. A valid integrated submission is the
single authoritative review decision for the round and immediately completes
its decision and coverage update.

## Review Sequencing

Every standard candidate requires an independent whole-candidate finalize
review, regardless of apparent size or narrowness. Ordinary execution steps do
not create review rounds or review debt.

The first passing coverage root is a `full` review of the complete candidate.
A narrow review-driven, CI, or conflict repair extends that root through an
automatically linked `delta` that verifies the targeted findings and related
regressions. A repair that materially changes candidate design, scope, or risk
uses `review start --full` and establishes a replacement root.

Archive readiness requires a continuous coverage chain rooted in a full round,
no unresolved blocking findings, no active round, and coverage of the current
candidate revision and every candidate-affecting change.

Post-archive merge readiness additionally requires:

- publish evidence with a PR URL
- CI good enough or explicit `not_applied`
- sync freshness or explicit `not_applied`
- the same reviewed candidate-owned delta, plus only command-owned
  active-to-archived plan and supplement moves and allowed `Closeout` updates.
  Unrelated base advancement may preserve coverage when a fresh sync record and
  the remote-tracking base prove the candidate delta is byte-for-byte and
  mode-for-mode unchanged. Upstream overlap, conflict resolution, candidate
  delta drift, or uncommitted work still requires reopen and review before
  `await_merge` or land.

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

- continue the current tracked outcome and record progress only at meaningful
  step or evidence boundaries

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
- assume the plan's durable `Closeout` has already been written from the
  current plan plus local artifacts, not reconstructed from agent memory
- require finalize review to be satisfied before archive succeeds
- if the plan still contains real deferred items, require the `Closeout`
  `Follow-Up Issues` value to contain a concrete issue URL or `#number`
  before allowing archive to succeed
- reject archive unless finalize coverage resolves to a continuous chain rooted
  in a full round, followed only by directly linked repair deltas whose anchors,
  reviewed heads, revisions, findings, and ancestry are consistent
- reject archive when the coverage tip has unresolved blocking findings, does
  not cover the current revision, or does not cover every candidate-affecting
  change
- allow post-review edits only inside the current plan's `Closeout` body; reject
  plan structure changes, supplements, product code, specifications, tests,
  unrelated documentation, and non-ignored untracked files after the covered
  head; only actual top-level Markdown sections qualify, never heading-like
  text inside fenced examples
- require `Closeout` to include structured `Validation`, `Review`, `Delivered`,
  `Not Delivered`, and `Follow-Up Issues` lines; their ordinary prose may wrap
  onto immediately following lines indented by at least two spaces
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
- after archive, keep the candidate bound to the reviewed head: only the
  mechanical plan/supplement archive move and allowed `Closeout` update may
  differ; any other tracked or untracked change requires reopen and review
- after archive, correctness should not depend on archived supplements still
  being present; anything the repository must keep relying on should already be
  absorbed into formal tracked locations
- PR checks may rerun on that archive commit; if new feedback or check failures
  appear, use `harness reopen --mode <finalize-fix|new-step>`
- merge actor, merge timestamp, and other land-only notes belong to forge
  history or conditional post-merge handoff rather than the archived plan
- if deferred items exist, the controller agent should replace `NONE` in
  `Follow-Up Issues` with a concrete issue URL or `#number` before archive
  completes

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
- classify branch freshness from the PR base/head commit comparison, independent
  of checks and provider policy: zero commits behind becomes `fresh`, a
  genuinely behind head becomes `stale`, and a merge conflict becomes
  `conflicted`
- record the compared immutable `base_commit` and `head_commit` with refreshed
  sync evidence so later base-aware coverage and post-merge land validation do
  not depend on mutable branch names
- surface provider approvals or merge-rule blocking separately from `sync`;
  `UNSTABLE` with pending checks does not make a current branch stale, while
  `BLOCKED` may leave sync fresh and still prevent merge approval
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
- when the current checkout is already the squash- or rebase-landed result,
  validate the immutable commit recorded by publish evidence instead of
  requiring the landed commit to descend from the feature head
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

The controller agent launches exactly one independent integrated reviewer. In
Codex, that means using `spawn_agent` rather than trying to do reviewer work in
the controller thread. The reviewer skill owns the complete fixed rubric,
automatic plan review focus, optional nested advisors, and the final call to
`harness review submit`.

In current Codex collaboration, a completed reviewer remains available but
needs no close operation. A later narrow repair with materially the same
candidate responsibility may trigger that idle reviewer with `followup_task`;
broader or changed scope should use a fresh `spawn_agent`.
Controllers wait for mailbox updates with `wait_agent` and inspect agent state
with `list_agents`; neither review artifacts nor this CLI contract invent a
runtime-specific resume/close protocol.

The CLI only owns deterministic local contracts:

- round-metadata persistence
- one reviewer handoff and submission path
- submission validation
- atomic decision and coverage update
- audit trail

## Deferred Commands

No additional user-facing command is committed in this spec yet beyond the
surface listed above.
