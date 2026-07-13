---
template_version: 0.2.0
created_at: "2026-07-13T17:06:00+08:00"
approved_at: "2026-07-13T17:16:34+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/263
size: L
---

# Review v2: finalize-first review with composable coverage

## Goal

Replace easyharness's step-review-heavy workflow with a finalize-first review
model suited to current strong coding agents. Tracked steps remain execution
and validation boundaries, but no longer create mandatory review debt. A final
full review uses one integrated reviewer by default and adds a small number of
risk-triggered specialists only when the completed candidate has a concrete
high-risk surface.

Make review outcomes git-native and composable. A full finalize review should
establish whole-candidate coverage at a captured git head, and a narrow repair
delta should extend that coverage and resolve findings without forcing another
full review. Archive must reject unreviewed candidate changes while allowing
the plan-only closeout updates required after review.

## Scope

### In Scope

- Remove mandatory step-closeout review and `NO_STEP_REVIEW_NEEDED` debt from
  ordinary workflow progression. Step-bound review remains available when the
  controller intentionally starts one at a real risk boundary.
- Make finalize full review the default formal review gate for plans of every
  size. Plan size alone must not cause per-step review.
- Replace the one-dimension/one-slot/one-agent review-spec assumption with
  explicit reviewer assignments:
  - one `integrated` reviewer covers the complete candidate and all assigned
    standard guidance;
  - zero or more `specialist` reviewers challenge concrete candidate risks;
  - ordinary review should default to zero specialists and normally cap
    specialist use at one unless multiple independent high-risk surfaces exist.
- Keep review dimensions as reusable guidance fragments, but allow one reviewer
  assignment to receive several dimensions. Dimension names no longer imply
  agent count or submission ownership.
- Add plan-scoped review guidance under the tracked plan package so approved
  invariants and specialist questions remain discoverable across finalize,
  repair, archive, and reopen. Adapt issue #263 to additive plan-local guidance;
  do not implement plan-level override semantics.
- Persist the git candidate boundary for every review round, including a
  command-captured `reviewed_head_sha`. Require a clean candidate worktree when
  a formal review starts and reject aggregation when candidate HEAD changes
  during the round.
- Model finalize review coverage as a chain:
  - `full` establishes whole-candidate coverage;
  - a repair `delta` references the prior finalize round and starts from the
    prior covered head;
  - a clean repair delta resolves the referenced findings and extends coverage
    to its captured head;
  - broad repair may intentionally reset coverage through a new full review.
- Make initial archive readiness accept a full review plus a continuous clean
  repair-delta chain. Do not require the latest round itself to be `full`.
- Preserve narrow reopened-revision behavior by allowing a later revision to
  extend the last archived coverage with a continuous delta; broad reopened
  work should establish a new full root.
- Treat `minor` findings as truly non-blocking. They remain visible but do not
  require repair or prevent archive. If the controller chooses to fix one, a
  narrow delta may extend coverage without invalidating the full root.
- Reject archive when product or source changes are not covered by the review
  chain. Explicitly allow only the current plan package's closeout summaries
  and archive move after final review; do not let that exemption cover product,
  specification, test, or unrelated documentation changes.
- Update status, lifecycle transitions, review UI resources, generated schemas,
  tests, and user-facing specifications for the new review topology and coverage
  resolver.
- Update the managed harness skills and AGENTS block for the current Codex
  collaboration surface:
  - applicable project/skill guidance may authorize bounded subagents without a
    separate per-run human authorization prompt;
  - clean-context spawn uses current `fork_turns` semantics;
  - same-agent follow-up uses `followup_task` rather than `resume_agent`;
  - waiting uses mailbox-oriented `wait_agent` plus `list_agents`, without an
    agent-ID list argument;
  - remove nonexistent `close_agent` requirements and describe interruption or
    completion using current capabilities.
- Dogfood Review v2 while executing this plan: no formal step review rounds;
  one final integrated full reviewer plus one
  `review-state-and-coverage` specialist; narrow review-driven repairs use
  repair delta rather than another full review unless the repair materially
  changes the candidate design.

### Out of Scope

- Compatibility reads, migrations, or dual-write support for review artifacts,
  review specs, local state, or managed prompts created before this cutover.
- Automatic risk scoring, file-pattern-based specialist injection, mandatory
  specialist catalogs, or CLI-owned decisions about how many agents to spawn.
- Automatically injecting every plan-scoped guidance file into every review.
- Plan-scoped override of built-in or repo-level review guidance.
- Concurrent active review rounds or multiple controllers mutating one round.
- Changing GitHub human-review policy, branch protection, CI requirements,
  publish evidence, or merge approval semantics except where archive readiness
  consumes the new local review coverage result.
- Selecting or pinning Codex models for integrated or specialist reviewers.
- Preserving the exact current review UI layout when a simpler reviewer- and
  coverage-oriented presentation is clearer.

## Acceptance Criteria

- [x] A tracked step can become complete and execution can advance without a
      step review round or `NO_STEP_REVIEW_NEEDED`; no earlier-step review debt
      blocks finalize or archive when no step review was intentionally started.
- [x] An intentionally started step review remains binding until aggregated;
      its blocking findings must be resolved before that step advances.
- [x] Finalize review defaults to one integrated reviewer assignment that may
      carry several review dimensions, and a specialist assignment requires a
      recorded concrete risk surface and invariants rather than plan size alone.
- [x] Integrated and specialist reviewers share the common submission,
      severity, evidence, and no-edit contract while receiving distinct role
      instructions; integrated review remains whole-candidate and specialist
      review remains a bounded adversarial challenge.
- [x] A plan package may carry additive plan-scoped review guidance that is
      discoverable when the plan is active, archived, or reopened and can be
      assigned to either an integrated reviewer or a specialist.
- [x] Plan-scoped guidance does not override the base reviewer contract or
      automatically create reviewer assignments.
- [x] `review start` captures the current candidate head in the round manifest,
      rejects a dirty candidate worktree, and a later aggregate rejects a round
      whose candidate HEAD moved after start.
- [x] A finalize repair delta records which prior finalize round it repairs,
      requires its git base to equal the prior covered head, captures its new
      reviewed head, and reports whether referenced blocking findings were
      resolved.
- [x] Archive accepts a continuous chain rooted in a full finalize review and
      extended by clean repair deltas, including a full round that originally
      requested changes whose findings were later resolved by delta.
- [x] Archive rejects missing full roots, broken or ambiguous delta links,
      unresolved blocking findings, failed or unaggregated rounds, and product
      changes after the latest covered head.
- [x] Plan-only closeout summaries and the command-owned active-to-archived plan
      move can occur after review without requiring a metadata-only review;
      the exemption is narrow and has regression coverage.
- [x] Non-blocking findings remain visible in aggregate, status, UI, and archive
      summaries but do not force repair or another review round.
- [x] Reopened narrow revisions can extend prior archived coverage with delta,
      while a controller can establish a new full root for materially broader
      changes.
- [x] Review specs, manifests, submissions, aggregates, status payloads, UI
      resources, schemas, CLI help, and normative docs consistently describe
      reviewer assignments and composable coverage rather than one agent per
      dimension or latest-round-only archive readiness.
- [x] Managed Codex instructions use only the current collaboration concepts and
      tool names/parameters described in scope, and no longer require a separate
      harness-run subagent authorization prompt.
- [x] Bootstrap assets are synchronized into `.agents/skills` and the managed
      root `AGENTS.md` block.
- [x] Focused package tests, lifecycle/review E2E coverage, schema and docs
      validation, UI tests/build, `git diff --check`, and `scripts/validate`
      pass.

## Deferred Items

- Consider closing or rewriting GitHub issue #263 after this plan lands so its
  title and body describe additive plan-scoped review guidance rather than
  append/override dimension semantics.
- Collect post-cutover dogfood metrics for rounds per plan, reviewer submissions,
  review wall time, specialist frequency, and findings discovered only in later
  unchanged-code reviews before deciding whether further reviewer reduction is
  warranted.
- Consider custom Codex reviewer agent configuration only after the workflow
  contract is stable; this slice keeps model and reasoning selection outside
  repository review state.

## Work Breakdown

### Step 1: Define the finalize-first review and reviewer-assignment contract

- Done: [x]

#### Objective

Write the normative Review v2 contract before changing runtime behavior: steps
are execution boundaries rather than mandatory review gates, finalize is the
default formal review, reviewer assignments group guidance dimensions, and
specialists are selected from actual candidate risk at pre-review time.

#### Details

Define the base reviewer contract, integrated role, specialist trigger rules,
risk-brief shape, assignment limits, and repair ownership. Specify that known
risks and invariants may be tracked in the plan while the actual reviewer
topology is chosen immediately before finalize review. Define additive
plan-scoped guidance and explicitly reject plan-local override semantics.

The contract must also define the dogfood boundary for this plan: do not start
formal step reviews. Until the runtime cutover removes the old step-closeout
gate, execution may use the old marker solely as a mechanical compatibility
note; it must not spawn step reviewers.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/state-transitions.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`
- `docs/specs/repo-config.md`
- `docs/specs/goal-oriented-workflow.md`

#### Validation

- Specs agree that plan size does not trigger step review.
- Reviewer assignments, integrated coverage, specialist triggers, common
  guidelines, plan-scoped guidance, and final-only dogfood rules are explicit.
- Documentation validation and `git diff --check` pass.

#### Execution Notes

Updated the normative state, transition, CLI, plan-schema, repo-config, and
goal-oriented workflow contracts for Review v2. Steps are now specified as
execution and validation boundaries rather than mandatory review gates;
finalize full review is the default formal gate at every plan size. The review
input is specified in terms of explicit integrated and specialist assignments,
where one assignment may compose several reusable guidance dimensions and a
specialist requires concrete risk surfaces plus invariants. The specs also
define additive plan-scoped guidance, narrow repair deltas, non-blocking minor
findings, and this plan's final-only dogfood topology.

Focused plan, repo-config, contracts, and contract-sync tests passed, as did
plan lint and `git diff --check`. The existing transition-catalog E2E still
expects the old normative table wording and is intentionally deferred to the
runtime behavior update in Step 3.

#### Review Notes

No formal step review ran. This approved Review v2 dogfood plan defers formal
review to the final candidate.

### Step 2: Cut review specs and plan-scoped guidance over to assignments

- Done: [x]

#### Objective

Replace dimension-owned reviewer slots with explicit reviewer assignments that
may consume several built-in, repo, or plan-scoped guidance dimensions.

#### Details

Change the review input, manifest, ledger, submission, aggregate, UI resource,
and generated-schema contracts together. Each assignment has a stable slot,
role (`integrated` or `specialist`), assigned guidance dimensions, and a
reviewer handoff. Findings retain their actionable area without assuming that
the submission's slot corresponds to one dimension.

Add conventional plan-scoped review-guidance discovery inside the plan's
supplements package. A same-name plan fragment may append plan-specific checks
to resolved guidance; new names act as plan-local guidance. Do not support
override. Listing/instruction commands must make source and plan scope visible,
but controllers still choose assignments explicitly.

#### Expected Files

- `internal/contracts/review.go`
- `internal/contracts/review_dimensions.go`
- `internal/contracts/review_ui.go`
- `internal/review/service.go`
- `internal/reviewdimensions/service.go`
- `internal/reviewui/service.go`
- `internal/repoconfig/config.go`
- `internal/plan/`
- `schema/`
- `web/`

#### Validation

- One integrated assignment can receive correctness, tests, risk, and docs
  guidance while producing one reviewer-owned submission.
- Specialist assignments require a non-empty risk/invariant handoff.
- Plan-scoped guidance moves with plan packages and resolves additively without
  automatic assignment or override.
- Old dimension-per-slot review specs are rejected rather than migrated.
- Focused contract, review, guidance, plan-package, schema, and UI tests pass.

#### Execution Notes

Cut the review contract over from one dimension-owned slot per reviewer to
explicit `integrated` and `specialist` assignments. One assignment now owns one
submission while snapshotting several resolved guidance fragments. Specialist
assignments require concrete risk surfaces and invariants; findings carry an
actionable area and harness-assigned stable ID; repair inputs and submissions
carry explicit finding references and resolution verdicts. Manifest, ledger,
submission, aggregate, status, review UI, web UI, timeline, input validation,
and generated schemas use the new shape with no legacy compatibility adapter.

Added strict additive plan-scoped guidance under
`supplements/<plan-stem>/review-guidance/`, including exact-plan resolution,
ordered source provenance, archive/reopen discovery, linting, and this plan's
`review-state-and-coverage` specialist guidance. Same-name plan guidance
appends after the built-in/repo result; new names remain plan-local; no guidance
creates an assignment automatically. While dogfooding the package, fixed a
relative-path lint defect that incorrectly treated a matching supplements
directory as an alternate-root conflict.

Focused review, guidance, plan, CLI, input-schema, contract-sync, status,
review-UI, and UI-server tests passed. All 34 web tests and TypeScript checking
passed, along with plan lint and `git diff --check`.

#### Review Notes

No formal step review ran. This approved Review v2 dogfood plan defers formal
review to the final candidate.

### Step 3: Implement git-bound composable finalize coverage

- Done: [x]

#### Objective

Bind formal review to immutable git candidate heads and derive archive
readiness from a continuous full-plus-repair-delta coverage chain.

#### Details

Capture `reviewed_head_sha` at review start, require the candidate worktree to
be clean, and reject aggregation if HEAD moved. Add explicit finalize repair
links and resolve coverage from review history rather than treating
`active_review_round` as the complete archive verdict.

The resolver must preserve broad coverage from a full round even when that
round requested changes, then allow a clean repair delta to resolve its
findings and extend the covered head. It must distinguish candidate-affecting
changes from the narrowly allowed plan-package closeout summaries and archive
move. Reopened narrow revisions may extend prior archived coverage; broad work
may start a new full root.

#### Expected Files

- `internal/contracts/runstate.go`
- `internal/runstate/`
- `internal/review/`
- `internal/lifecycle/`
- `internal/status/`
- `internal/stepcloseout/`
- `internal/timeline/`
- `tests/e2e/`
- `schema/`

#### Validation

- Unit and E2E tests cover clean full pass, full findings plus clean repair
  delta, multiple continuous deltas, optional minor repair, broken anchors,
  moved HEAD during review, unreviewed product changes, plan-only closeout,
  reopened narrow delta, and broad new-full reset.
- Steps without review advance normally, while an intentionally active or
  failed step review still blocks its own boundary.
- Status and archive errors identify the exact missing coverage or unresolved
  finding instead of asking for a generic latest full review.
- Focused review, runstate, lifecycle, status, archive/reopen, and E2E suites
  pass.

#### Execution Notes

Added command-captured `reviewed_head_sha`, clean committed-candidate checks at
review start and immediately before aggregate persistence, explicit finalize
repair links, cumulative blocking-finding resolution, and a durable
full-plus-delta coverage resolver. Local state caches the validated finalize
root/tip/head without replacing the manifest and aggregate source of truth.
Archive now accepts a continuous passing repair chain, preserves coverage
across narrow reopen revisions, rejects unreviewed product or supplement
changes, and permits only the four current-plan closeout bodies after review.

Removed routine step-closeout debt from review start, status, and archive.
Steps advance without review artifacts or magic markers; an intentionally
started step review remains binding until its blocking findings are resolved.
Removed the now-unused `internal/stepcloseout` implementation. Added stable
finding IDs, explicit repair resolutions, cumulative blocker display, coverage
status in the review UI, Git-aware unit fixtures, and adversarial resolver
checks for malformed anchors, targets, duplicate repair IDs, inherited-ID
collisions, and altered finding data.

Focused review, coverage, lifecycle, status, CLI, review UI, UI server, and
resilience suites pass. The full E2E suite passed with dirty start/aggregate,
moved HEAD, full-to-delta archive, optional minor repair, closeout exemption,
and narrow reopen coverage.

#### Review Notes

No formal step review ran. Independent controller audit found and prompted
repairs for unknown or duplicate repair targets and a parent-finding ID
collision that could rewrite inherited finding data; regressions now reject
each malformed durable chain. Formal review remains deferred to finalize.

### Step 4: Update managed agent workflow for integrated review and current Codex

- Done: [x]

#### Objective

Teach controllers and reviewers to execute Review v2 with current Codex
collaboration capabilities and without per-run subagent authorization ceremony.

#### Details

Update bootstrap-source assets rather than hand-editing materialized harness
skills. The controller should select actual specialist assignments during the
pre-finalize scan, keep the integrated reviewer whole-candidate, give each
specialist a concrete risk brief, and use repair deltas after narrow findings.

Replace stale `fork_context`, `resume_agent`, `close_agent`, and
`wait_agent(ids=...)` instructions with the current spawn, follow-up, wait,
list, send, and interrupt semantics. Keep runtime-specific details centralized
in one Codex adapter reference where practical; product specs should remain
runtime-neutral.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `assets/bootstrap/skills/harness-execute/references/step-inner-loop.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `assets/bootstrap/skills/harness-reviewer/`
- `AGENTS.md`
- `.agents/skills/harness-execute/`
- `.agents/skills/harness-reviewer/`

#### Validation

- Prompt simulations show that an ordinary candidate starts one integrated
  reviewer, a concrete process-lifecycle risk adds one specialist, and plan
  size alone adds none.
- Repair simulations select delta by default and promote to full only after a
  material scope/risk change.
- Codex simulations use `fork_turns`, `followup_task`, mailbox-oriented
  `wait_agent`, and `list_agents` correctly and contain no stale tool calls.
- No managed prompt asks for separate per-run subagent authorization when
  applicable project or skill guidance already authorizes bounded work.
- `scripts/sync-bootstrap-assets`, bootstrap drift checks, prompt validation,
  and `git diff --check` pass.

#### Execution Notes

Updated bootstrap-source managed instructions for finalize-first orchestration,
one integrated reviewer by default, concrete-risk specialists, shared base plus
role overlays, and linked repair delta closeout. The Codex adapter now describes
`spawn_agent` with `fork_turns`, `followup_task`, non-triggering
`send_message`, mailbox-oriented `wait_agent`, `list_agents`, and
`interrupt_agent`; it removes per-run authorization prompts and nonexistent
resume/close calls when applicable repository or skill guidance already
authorizes bounded delegation.

Ran `scripts/sync-bootstrap-assets` to refresh root `AGENTS.md` and the
materialized `.agents/skills` pack. Sync/check, install, bootstrap smoke,
stale-tool scans, prompt assertions, and focused review package tests pass.

#### Review Notes

No formal step review ran. Managed-source/materialized drift and current-tool
assertions are covered mechanically; formal integrated and specialist review
remains deferred to the complete candidate.

### Step 5: Complete cross-surface validation and dogfood final-only review

- Done: [x]

#### Objective

Validate the breaking Review v2 cutover across CLI, lifecycle, UI, schemas,
managed assets, and real harness execution, then review the complete candidate
once under the new topology.

#### Details

Remove obsolete step-review fixtures and expectations rather than retaining
compatibility paths. Refresh generated schemas and any UI/build artifacts
through repository commands. Exercise a real active plan through step
completion without formal step review, one full finalize round with a single
integrated reviewer plus one `review-state-and-coverage` specialist, a narrow
repair delta if findings require it, archive, and reopen-sensitive coverage.

The specialist risk brief for this plan must challenge candidate SHA capture,
HEAD mutation detection, full-to-delta coverage continuity, finding resolution,
plan-only closeout exemptions, and archive refusal for unreviewed product
changes. The integrated reviewer remains responsible for the entire candidate.

#### Expected Files

- `tests/e2e/`
- `internal/*_test.go`
- `scripts/`
- `schema/`
- `web/`
- `README.md`
- `docs/`

#### Validation

- The repository's focused Go, E2E, schema, documentation, bootstrap, UI, and
  build checks pass.
- `scripts/validate` and `git diff --check` pass from a clean worktree.
- No formal step review round exists for this plan.
- Finalize review uses exactly one integrated reviewer and one
  `review-state-and-coverage` specialist unless the approved scope materially
  changes.
- A narrow review repair is closed by a continuous repair delta; a new full
  round is used only when the controller records why earlier whole-candidate
  coverage is no longer trustworthy.

#### Execution Notes

Completed the breaking cross-surface cutover across CLI inputs/results,
lifecycle and status resolution, review UI resources, generated schemas,
managed assets, smoke fixtures, and real E2E execution. Removed legacy
dimension-per-slot fixtures and mandatory step-review-debt expectations instead
of adding compatibility adapters. Review UI and Playwright fixtures now expose
assignment roles, reviewed heads, repair parents, targeted/resolved/unresolved
finding IDs, and cumulative coverage state.

Validation passed with `scripts/validate` (embedded UI build plus all Go,
release, E2E, resilience, smoke, and support tests), 35 web tests, TypeScript
checking, bootstrap and contract drift checks, plan lint, Bash syntax checks for
the Playwright smoke fixtures, and `git diff --check`. The E2E suite also passed
uncached in 59 seconds. The repository has no `tools/check`; the plan was
corrected to name its actual full validation entrypoint, `scripts/validate`.

#### Review Notes

No formal step review ran. The complete committed candidate will now receive
the approved finalize topology: one integrated reviewer and one
`review-state-and-coverage` specialist.

## Validation Strategy

- Use Red/Green/Refactor for contract and lifecycle behavior changes.
- Treat generated schemas, CLI envelopes, review UI resources, managed
  bootstrap assets, and tracked specifications as one coordinated breaking
  cutover; do not leave dual shapes or fallback reads.
- Run focused tests after each implementation step, but do not start formal
  step review rounds. While the old runtime is still active, close step notes
  with the minimum mechanical `NO_STEP_REVIEW_NEEDED` explanation required to
  reach the final dogfood boundary.
- Before formal finalize review, commit all candidate-affecting changes and
  require a clean worktree so the new review round can capture its head.
- Run one full finalize round with:
  - one integrated reviewer assigned all relevant standard guidance;
  - one `review-state-and-coverage` specialist with the explicit invariants in
    Step 5.
- For narrow findings, repair, validate, commit, and run a linked repair delta
  with only the reviewer assignments needed to close the affected risk. Do not
  rerun full merely because a delta became the latest round.
- Run `scripts/validate` before finalize review and again after any behavior-changing
  repair.

## Risks

- Risk: Removing mandatory step review could let architectural mistakes
  accumulate in very large plans.
  - Mitigation: Keep step review available at explicit semantic risk boundaries,
    rely on focused validation and controller challenge during implementation,
    and split work whose final candidate is too broad for reliable integrated
    review.
- Risk: One integrated reviewer could trade breadth for shallower attention.
  - Mitigation: Give it explicit multi-guidance coverage and add at most a small
    number of independent specialists for concrete high-risk mechanisms rather
    than fragmenting every dimension into a separate agent.
- Risk: Coverage-chain logic could accept an unreviewed candidate through an
  incorrect anchor or closeout exemption.
  - Mitigation: Capture heads in command-owned artifacts, validate continuous
    ancestry, reject HEAD movement during rounds, keep plan-only exemptions
    path- and transition-specific, and exercise adversarial E2E cases plus the
    dedicated final specialist.
- Risk: Replacing review schemas and prompts together creates a large cutover
  surface.
  - Mitigation: Use the repository's fast-development policy: remove obsolete
    shapes, regenerate all contracts in one branch, and validate every first-
    party consumer without compatibility shims.
- Risk: Static Codex tool guidance may drift again.
  - Mitigation: Keep runtime-specific names in one adapter reference where
    possible and keep product contracts capability-oriented.
- Risk: Plan-scoped guidance could become another automatic ceremony layer.
  - Mitigation: Make it optional, additive, explicitly assigned, and incapable
    of creating specialists or blocking archive on its own.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
