---
template_version: 0.3.0
created_at: "2026-07-28T23:14:13+08:00"
approved_at: "2026-07-28T23:16:45+08:00"
source_type: direct_request
source_refs: []
size: XL
---

# Support coordinated multi-plan execution

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb repository-facing normative content into formal tracked locations before
archive, and record supplement absorption in Closeout. Lightweight plans should
normally avoid supplements. -->

## Goal

Add an explicit `workflow_profile: coordinated` mode in which one
human-approved root plan owns a shared worktree candidate while controller
agents execute multiple flat subplans in parallel. Keep ordinary standard plans
as the default, preserve one whole-candidate review/archive/publish/land
lifecycle, and make the coordinated workflow easier for agents to operate than
manually switching a worktree-global current plan.

### Decisions and Constraints

- Omitted `workflow_profile` continues to mean the existing standard workflow;
  coordinated behavior is enabled only by
  `workflow_profile: coordinated`.
- One worktree may have one candidate-owning coordinated root with multiple
  flat subplans. Independent roots that need separate candidates, reviews, or
  land operations remain separate-worktree concerns.
- Human approval applies only to the coordinated root's durable goal, scope,
  acceptance criteria, decisions, review focus, and authority boundaries.
  Agents may create, revise, split, combine, or remove subplans within that
  approved boundary without separate approval.
- Trust agents and the existing human steering contract. Do not add approval
  epochs, digests, provenance, generation counters, finalization seals, or
  similar verification ceremony.
- Subplans are flat siblings under the root plan package; subagent nesting does
  not create nested subplans. Optional dependencies form an unordered or
  partially ordered execution graph across those siblings.
- Each subplan uses the existing compact ordered-step vocabulary internally.
  Simple subplans may contain one step. A subplan does not own independent
  approval, execution-start, formal review, archive, publish, or land
  milestones.
- The controller owns integration and serializes Git staging and commits when
  needed. Harness does not claim that parallel agents receive filesystem,
  index, HEAD, branch, or candidate isolation inside one worktree.
- The root current-plan pointer remains the default worktree context. Explicit
  subplan reads must not change that pointer or make the most recently touched
  child the implicit current plan.
- Formal review remains a single whole-candidate root review after coordinated
  implementation has settled. Existing clean committed HEAD and review
  coverage checks remain the final candidate boundary.
- Coordinated/multi-plan UI support is explicitly deferred. Existing UI
  behavior only needs to remain correct for the supported single-plan standard
  workflow.
- Codex Goal integration is not part of this design. A Goal may use a
  coordinated plan as an external controller choice, but the harness contract
  must not contain Goal-specific state, identifiers, budgets, or lifecycle
  behavior.

## Scope

### In Scope

- Define the coordinated root and subplan contracts, including their tracked
  package layout, approval ownership, execution semantics, completion rules,
  and archive representation.
- Add explicit authoring and lint support for
  `workflow_profile: coordinated` while leaving the standard plan shape
  unchanged.
- Support flat subplans under the coordinated root's matching
  `supplements/<root-stem>/subplans/` directory, with compact ordered steps,
  root-acceptance coverage, optional sibling dependencies, and concise final
  results.
- Resolve one coordinated root as the default worktree plan and support
  `harness status --plan <subplan-id-or-path>` for an explicitly selected
  subplan without mutating root current-plan context.
- Make root status aggregate coordinated progress and guidance rather than
  pretending that one child is the root's sequential current step.
- Make coordinated root lifecycle commands enforce root approval, root-level
  execution start, completed required subplans, root acceptance, one final
  review, whole-package archive/reopen, and the existing publish/land flow.
- Detect the minimal dependency errors needed for safe scheduling: duplicate
  identifiers, missing references, self-dependencies, and cycles.
- Keep subplan and aggregate read behavior understandable during parallel
  updates through conservative errors or warnings rather than invented
  transactional certainty.
- Update controller-facing guidance so agents can decompose approved work,
  delegate non-overlapping subplans, avoid concurrent Git mutation, settle
  child agents before final review, and keep user-facing steering at the root.
- Preserve existing single-plan standard CLI and UI behavior with focused
  regression coverage.

### Out of Scope

- Coordinated-plan UI browsing, plan switching, dependency visualization,
  aggregate dashboards, or any other multi-plan UI behavior.
- Multiple independent candidate-owning roots executing, reviewing, publishing,
  or landing concurrently in one worktree.
- Nested subplans or a plan topology that mirrors nested subagent delegation.
- Child-level human approval, `execute start`, formal review coverage, archive,
  publish, CI/sync evidence, merge approval, or land.
- Tracking live subagent identity, ownership, liveness, or assignment state in
  durable plan artifacts.
- Goal-specific plan binding, token-budget behavior, automatic continuation,
  blocked semantics, or completion semantics.
- Approval fingerprints, inherited approval records, content digests,
  provenance receipts, graph generations, finalization seals, or public
  package-lock ceremony.
- Automatic downstream invalidation when a dependency changes, subplan
  revision propagation, mandatory split/merge lineage, cancellation
  taxonomies, or build-system-style incremental correctness.
- Git worktree, branch, index, commit, merge, or file-ownership isolation for
  parallel agents.
- Making every temporary explorer, advisor, validation task, or subagent
  assignment into a tracked subplan.

## Acceptance Criteria

- [x] A coordinated profile is explicitly selectable with
      `workflow_profile: coordinated`, while plans without that field retain
      the existing standard contract and behavior.
- [x] A valid coordinated root can own multiple flat subplan documents under
      its matching supplements package without treating those documents as
      separately human-approved root plans.
- [x] Coordinated subplans use compact ordered steps and optional sibling
      dependencies, can represent related or unrelated work bundled into one
      candidate, and cannot form nested subplan trees.
- [x] Subplan completion is derived from its durable step and result content
      without requiring child-level execution-start, formal review, archive,
      publish, or land commands.
- [x] `harness status` resolves and summarizes the coordinated root by default,
      while `harness status --plan <subplan-id-or-path>` reports the selected
      child without changing the worktree's root current-plan pointer.
- [x] Coordinated root status reports aggregate child progress and dependency
      blockers without selecting one arbitrary child as the root's current
      sequential step.
- [x] Duplicate subplan identifiers, missing dependency references,
      self-dependencies, and dependency cycles produce clear failures or
      blockers instead of guessed scheduling.
- [x] Root execution and archive readiness require root approval, settled
      required subplans, satisfied root acceptance criteria, whole-candidate
      validation, and one clean root finalize review at the current committed
      candidate.
- [x] Archiving and reopening a coordinated root move its complete plan
      package, leave only the root Markdown at the top-level archived plan
      surface, and preserve subplans beneath the root's archived supplements.
- [x] Child selection and progress updates never replace the coordinated root
      as the worktree-global current plan or redirect unqualified lifecycle
      commands to the most recently touched child.
- [x] Agent-facing workflow guidance makes controller integration, bounded
      non-overlapping delegation, serialized Git mutation, and pre-review
      subagent settlement explicit without adding new verification ceremony.
- [x] Existing standard single-plan CLI behavior and supported single-plan UI
      behavior remain covered and pass, while coordinated-plan UI support is
      clearly documented as deferred and unsupported.

## Review Focus

- Confirm that coordinated execution adds only the concepts agents need:
  explicit root mode, flat subplans, optional dependencies, child selection,
  root aggregation, and one final candidate lifecycle.
- Check that no command or fallback can silently switch the root current-plan
  pointer because a child was read or updated.
- Check that root approval remains the only human approval boundary without
  introducing provenance or anti-agent verification mechanics.
- Verify that standard plans retain their current schema, state resolution,
  lifecycle, CLI output, and supported UI behavior.
- Verify that subplan dependency validation fails clearly for ambiguous or
  invalid graphs without growing into a general scheduler or build system.
- Verify that child completion cannot by itself claim root acceptance,
  whole-candidate review coverage, archive readiness, or merge readiness.
- Check archive/reopen rollback and failure paths for complete root packages
  containing subplans.
- Check parallel read and mutation paths for stale pointer use, basename
  collisions, accidental child formal-review state, and misleading aggregate
  snapshots.

## Deferred Items

- Multi-plan UI support, including coordinated root summaries, child browsing,
  child switching, dependency graphs, and parallel progress visualization.
- Independent multi-root candidate execution within one worktree.
- Rich scheduling, automatic downstream invalidation, subplan revision
  propagation, persistent agent assignment, and historical execution-graph
  analytics.

## Work Breakdown

### Step 1: Establish the coordinated plan contract

- Done: [x]
- Outcome: Formal plan, state, CLI, and package contracts define the explicit
  coordinated root, flat ordered-step subplans, root-only approval and final
  lifecycle, minimal dependency rules, and deliberate exclusions without
  changing the standard plan contract.
- Covers: Acceptance criteria 1 through 4, 7, and 12.
- Check: Contract, template, and lint-focused tests accept valid coordinated
  packages, reject invalid child shapes and dependency graphs, and retain
  standard-plan coverage.

### Step 2: Make resolution and status coordinated-aware

- Done: [x]
- Outcome: Root-default resolution, explicit child status selection, scoped
  child identity, aggregate coordinated status, and conservative parallel-read
  behavior work without child operations mutating the root current-plan
  context.
- Covers: Acceptance criteria 5 through 7 and 10.
- Check: Focused resolver and status tests cover root defaults, explicit child
  reads, ambiguity, pointer stability, aggregate progress, dependency blockers,
  and concurrent snapshot degradation.

### Step 3: Integrate coordinated packages with the root lifecycle

- Done: [x]
- Outcome: Approval, execution start, finalize readiness, archive, reopen,
  publish, and land continue to operate once per coordinated root candidate,
  with complete child-package preservation and no child-level lifecycle
  ceremony.
- Covers: Acceptance criteria 8 through 10.
- Check: Lifecycle and end-to-end tests exercise coordinated success, incomplete
  children, invalid dependencies, final-review gating, package archive/reopen,
  rollback, and existing candidate coverage checks.

### Step 4: Align agent guidance and protect the standard workflow

- Done: [x]
- Outcome: Controller guidance explains coordinated decomposition and safe
  parallel execution, while standard single-plan CLI and supported UI behavior
  remain stable and coordinated UI behavior is explicitly deferred.
- Covers: Acceptance criteria 11 and 12.
- Check: Bootstrap synchronization, documentation checks, standard workflow
  regression tests, and supported single-plan UI smoke coverage pass without
  adding coordinated UI requirements.

## Validation Strategy

- Run focused plan template, parser, lint, resolver, status, lifecycle,
  archive/reopen, review-coverage, timeline, and contract-schema tests for both
  coordinated and standard plans.
- Add end-to-end coverage for one coordinated root with parallel-ready,
  dependency-blocked, completed, and invalid subplans through root finalization
  and package archive/reopen.
- Exercise custom configured active, archived, and local-runtime roots so
  coordinated package and explicit child resolution do not rely on default
  paths.
- Run the complete Go test suite and repository smoke validation after focused
  coverage passes.
- Run the existing supported single-plan UI tests and smoke path only; do not
  add coordinated/multi-plan UI implementation or acceptance.
- Rerun bootstrap asset synchronization and drift checks when managed
  controller guidance changes.

## Closeout

- Archived At: 2026-07-29T00:17:32+08:00
- Revision: 1
- Validation: `scripts/validate`; focused coordinated built-binary E2E and
  transition-catalog tests; race tests for plan, status, lifecycle, and review;
  bootstrap/contract sync checks; `git diff --check`; and direct installed
  harness lint/status checks all passed.
- Review: Independent full review `review-001-full` and linked deltas
  `review-002-delta` and `review-003-delta` covered the complete candidate.
  The final delta passed after all basename-collision, path-confinement,
  concurrent-snapshot, and transition-contract findings were resolved.
- Delivered: Added the explicit coordinated root profile, flat ordered
  subplans with optional sibling dependencies, aggregate and selected-child
  status, one root-owned approval/review/archive/publish/land lifecycle,
  complete-package archive/reopen behavior, CLI/templates, durable specs and
  agent guidance, conservative package reads, and regression coverage for the
  existing standard/lightweight and single-plan UI paths.
- Not Delivered: Coordinated/multi-plan UI, independent multi-root candidates
  in one worktree, and advanced scheduling or execution-graph machinery remain
  deferred.
- Follow-Up Issues: https://github.com/catu-ai/easyharness/issues/133,
  https://github.com/catu-ai/easyharness/issues/303
