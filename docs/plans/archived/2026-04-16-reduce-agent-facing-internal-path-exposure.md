---
template_version: 0.2.0
created_at: "2026-04-16T09:14:22+08:00"
approved_at: "2026-04-16T09:16:22+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/169
size: L
---

# Reduce agent-facing exposure of internal harness paths

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Close issue 169 by shrinking low-value internal-path exposure across
`harness status`, review command outputs, UI read models, and timeline-facing
artifacts without breaking the command-owned files that agents actually use in
normal workflow.

The key contract for this slice is not "hide every `.local` path." Instead,
the system should keep agent-facing artifacts that are part of the intended
workflow surface, such as reviewer slot `submission.json`, while removing
internal control files, state files, log/index paths, and other storage-layout
details that agents do not need to see in order to decide or act through the
CLI.

## Scope

### In Scope

- Define and document one retained-path policy based on whether a path is
  genuinely agent-facing, not whether it happens to live under `.local/`.
- Remove low-value internal control paths from `harness status` and review
  command outputs while preserving stable non-path handles such as
  `round_id` and the slot-owned `submission.json` artifact.
- Normalize retained path values so agent-facing outputs use repo-facing
  relative paths instead of absolute storage paths; add `project_root` only
  where an explicit anchor is still needed by the contract.
- Apply the same policy to `/api/plan`, `/api/review`, `/api/timeline`, and
  timeline event artifact refs so UI-facing read models do not leak internal
  control files or index paths.
- Update schemas, specs, skills, and automated coverage so the new boundary is
  durable and reviewable.

### Out of Scope

- Reworking the underlying on-disk review/state/timeline storage layout.
- Hiding every `.local` path regardless of whether the agent must use it in
  ordinary workflow.
- Introducing a new debug or maintainer-only artifact browser in this slice.
- Changing review orchestration semantics beyond what is required to redefine
  which paths are agent-facing.
- Broad evidence/lifecycle output cleanup outside the low-value internal-path
  exposure covered by issue 169.

## Acceptance Criteria

- [x] The tracked contract explicitly defines agent-facing path retention in
      terms of workflow need: command-owned artifacts that agents use directly
      may remain surfaced, while internal control/state/log/index paths do
      not.
- [x] `harness status` no longer exposes low-value internal control paths such
      as `local_state_path` or absolute plan/storage paths; retained status
      paths are repo-facing and aligned with the approved contract.
- [x] Review command outputs keep the stable handles agents actually use
      (`round_id`, slot `submission.json` where appropriate) but stop
      surfacing `manifest.json`, `ledger.json`, `aggregate.json`, and similar
      internal control paths.
- [x] `/api/plan`, `/api/review`, `/api/timeline`, and timeline event refs no
      longer surface low-value internal paths such as `reviews_dir`,
      `event_index_path`, `local_state_path`, or review-control artifact
      paths, while preserving any intentionally agent-facing artifact handles.
- [x] Specs, schemas, skills, and regression tests all agree on the new
      boundary and fail if low-value internal paths leak back into
      agent-facing outputs.

## Deferred Items

- Reconsidering whether a future explicit debug/maintainer surface should
  expose internal control artifacts behind a non-default contract.
- Broader cleanup of non-review command outputs that may also benefit from the
  same policy but are not required to close issue 169.

## Work Breakdown

### Step 1: Narrow the retained path contract for status and review commands

- Done: [x]

#### Objective

Define which paths remain agent-facing in command outputs, then implement that
boundary in `harness status` and the review command result surfaces.

#### Details

Use the discovery decision as the governing rule: keep paths only when agents
are expected to use them directly in normal workflow. For status, that means
removing low-value internal control pointers such as `local_state_path` and
fixing any remaining absolute-path leakage. For review commands, preserve the
stable handles that matter to agents, especially `round_id` and the
slot-owned `submission.json`, while removing `manifest.json`, `ledger.json`,
`aggregate.json`, and other internal control-file paths from the surfaced
contract. If an output still needs a path anchor after the cleanup, prefer an
explicit repo-facing anchor such as `project_root` over leaking storage-layout
details.

#### Expected Files

- `internal/contracts/status.go`
- `internal/contracts/review.go`
- `internal/status/service.go`
- `internal/review/service.go`
- `schema/commands/status.result.schema.json`
- `schema/commands/review.start.result.schema.json`
- `schema/commands/review.submit.result.schema.json`
- `schema/commands/review.aggregate.result.schema.json`
- `docs/specs/cli-contract.md`
- `internal/status/service_test.go`
- `internal/review/service_test.go`
- `internal/cli/app_test.go`
- `tests/e2e/helpers_test.go`

#### Validation

- Status output for active and idle paths no longer returns low-value internal
  control paths or absolute storage paths.
- Review start/submit/aggregate outputs preserve agent-usable handles but no
  longer emit manifest/ledger/aggregate control-file paths.
- Unit and command-level tests are updated to lock the retained-path contract
  and reject regressions in the removed fields.

#### Execution Notes

Updated the status and review command contracts so agent-facing outputs keep
repo-usable anchors (`project_root`, repo-facing `plan_path`, `round_id`, and
slot-owned `submission.json`) while dropping low-value internal control-file
paths such as `local_state_path`, `manifest.json`, `ledger.json`, and
`aggregate.json`. Adjusted the CLI timeline hooks, schemas, specs, and unit +
e2e coverage so tests now reconstruct internal review files from known storage
layout instead of depending on those paths being surfaced by command output.

#### Review Notes

Review should confirm that command-owned reviewer workflow artifacts remain
usable after the cleanup, especially that reviewers still receive the handles
needed to update and submit their slot-owned `submission.json`.

### Step 2: Apply the same path policy to UI read models and timeline artifacts

- Done: [x]

#### Objective

Make the read-only UI and timeline surfaces follow the same agent-facing path
boundary as the command layer.

#### Details

The UI workbench should orient agents without teaching them the internal
storage layout. Remove top-level read-model artifacts such as
`local_state_path`, `reviews_dir`, and `event_index_path`, and strip timeline
artifact refs that point at internal control files or path-only bookkeeping.
Preserve stable non-path handles and any intentionally agent-facing artifact
reference that is still part of the workflow, but do not keep control-file
paths merely because the UI can technically read them. Review-specific tabs or
artifact panels should continue to show meaningful review content without
depending on the control-path exposure being removed.

#### Expected Files

- `internal/contracts/plan_ui.go`
- `internal/contracts/review_ui.go`
- `internal/contracts/timeline.go`
- `internal/planui/service.go`
- `internal/reviewui/service.go`
- `internal/timeline/service.go`
- `internal/cli/timeline_events.go`
- `schema/ui-resources/plan.schema.json`
- `schema/ui-resources/review.schema.json`
- `schema/ui-resources/timeline.schema.json`
- `web/src/helpers.ts`
- `internal/planui/service_test.go`
- `internal/reviewui/service_test.go`
- `internal/timeline/service_test.go`
- `internal/ui/server_test.go`

#### Validation

- `/api/plan`, `/api/review`, and `/api/timeline` no longer surface removed
  internal path fields in their top-level artifacts.
- Timeline events no longer carry low-value internal path refs such as
  `local_state_path`, review-control artifact paths, or index bookkeeping
  paths.
- UI/read-model tests fail if removed internal paths reappear while still
  allowing intentionally retained agent-facing handles.

#### Execution Notes

Updated `/api/plan`, `/api/review`, and `/api/timeline` so their top-level
artifacts no longer expose `local_state_path`, `reviews_dir`, or
`event_index_path`. Review UI responses now keep reviewer-owned submission
artifacts surfaced with repo-facing paths while hiding manifest/ledger/aggregate
control artifacts from the read model. Timeline refs were narrowed so review
events keep reviewer-owned `submission_path` handles and lifecycle/evidence
events stop advertising internal control-file paths in their surfaced artifact
refs. Synced the UI schemas, frontend types/helpers, and rebuilt the embedded UI
bundle so the workbench reflects the same boundary.

#### Review Notes

Review should compare the read-model cleanup against the command-layer contract
so the UI does not silently preserve internal-path exposure that the CLI has
already removed.

### Step 3: Sync specs, skills, and regression coverage with the new boundary

- Done: [x]

#### Objective

Make the retained-path policy durable by aligning documentation, skill
guidance, and regression coverage with the implemented contract.

#### Details

The final state should be understandable without this discovery chat. Update
the CLI contract and review workflow guidance so future agents know that the
decision boundary is "agent-facing versus internal control path," not
"tracked versus `.local`." Skills should keep steering reviewers toward the
slot-owned `submission.json` and away from command-owned control files. Add
or tighten regression coverage anywhere the old path surface was implicitly
accepted so the issue does not reopen through schema drift, helper fixtures,
or timeline-hook behavior.

#### Expected Files

- `docs/specs/cli-contract.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `assets/bootstrap/skills/harness-execute/references/controller-truth-surfaces.md`
- `schema/index.json`
- `internal/contractsync/sync_test.go`
- `tests/e2e/review_workflow_test.go`
- `tests/e2e/helpers_test.go`
- `tests/resilience/helpers_test.go`

#### Validation

- The written docs and skills match the implemented retention boundary and do
  not instruct agents to inspect removed internal control files.
- Contract-sync and regression tests catch reintroduction of removed internal
  path fields.
- The issue can be closed from repository state alone without relying on chat
  context to explain why some `.local` paths remain and others do not.

#### Execution Notes

Aligned the written contract and bootstrap skills with the new
agent-facing-versus-internal-control rule. The CLI spec now explains that
review start returns reviewer-owned slot artifacts instead of control-file
paths, reviewer guidance now treats the controller-provided `submission_path`
as the working artifact, and controller review-orchestration guidance now
passes `submission_path` explicitly instead of teaching reviewers about
manifest/ledger/aggregate files. Synced bootstrap materializations and
regenerated contract artifacts so schemas, schema index, generated schema
embeds, and regression coverage all agree with the implementation.

#### Review Notes

Review should verify that documentation and skills preserve the intended
reviewer workflow surface while removing references to internal control files
that agents no longer need to know.

## Validation Strategy

- Run `harness plan lint` on this tracked plan before approval.
- Run targeted Go tests for the touched command, contract, status, review, UI,
  and timeline packages while implementing each step.
- Run the relevant command-level and end-to-end review workflow tests that
  exercise `status`, `review start`, `review submit`, `review aggregate`, and
  the read-only UI/timeline surfaces.
- Run contract/schema sync coverage so the checked-in schemas match the Go
  contracts after the path-surface changes.
- Run `git diff --check`.
- Run `go test ./...` if the targeted suites pass and the total runtime is
  still practical for the slice.

## Risks

- Risk: Removing too much path data could strand reviewer workflows by hiding a
  command-owned artifact that agents actually need.
  - Mitigation: Keep the boundary centered on direct workflow use, preserve the
    slot-owned `submission.json` and stable IDs, and add focused review/tests
    around the retained handles.
- Risk: Partial cleanup could leave the CLI, UI, schemas, and skills
  disagreeing about which paths are agent-facing.
  - Mitigation: Treat docs/schema/test alignment as a first-class work step and
    validate the same retention rules across command outputs, read models, and
    skill guidance.
- Risk: Existing tests or helpers may silently encode the old internal-path
  surface and mask regressions until later.
  - Mitigation: Update fixture helpers and regression assertions early so the
    new boundary is enforced during the same slice.

## Validation Summary

- `scripts/sync-bootstrap-assets` refreshed the managed reviewer/controller
  skill materializations after the handoff-contract updates.
- `pnpm --dir web build` rebuilt the embedded workbench after the timeline
  helper changes that hide legacy evidence labels in artifact tabs.
- `go test ./internal/timeline ./internal/reviewui ./internal/ui ./internal/cli ./internal/contractsync ./tests/e2e/... ./tests/smoke`
  passed after the reviewer-handoff and timeline-payload fixes.
- `go test ./internal/timeline ./internal/ui ./tests/e2e ./tests/smoke`
  passed after the legacy evidence-label cleanup for `publish_record`,
  `ci_record`, and `sync_record`.
- Finalize-review follow-up verification also passed in reviewer-owned runs
  covering the command, UI, schema, evidence, and end-to-end surfaces touched
  by this slice.
- Revision 2 reopen validation merged `origin/main` into the candidate,
  resolved the generated UI artifact conflicts by rebuilding the embedded UI
  bundle, and passed `go test ./...` against the merged worktree.

## Review Summary

- Step 1 full review `review-001-full` requested changes; after narrowing the
  status/review command outputs and updating the command-level contract,
  `review-002-full` passed.
- Step 2 full review `review-003-full` requested changes; after aligning the
  read-only UI/timeline surfaces and their tests, `review-004-full` passed.
- Step 3 full review needed multiple repair rounds (`review-005-full` through
  `review-008-full`) before the docs, skills, schema index, and generated
  artifacts fully matched the new agent-facing boundary.
- Finalize full review then exercised the branch candidate through
  `review-009-full` through `review-017-full`, surfacing the remaining legacy
  leaks and reviewer-handoff gaps in timeline payloads, degraded errors,
  evidence outputs, raw read-model artifacts, and reviewer plan discovery.
- Finalize full review `review-018-full` passed with no findings; both
  `correctness` and `agent_ux` confirmed that the candidate now keeps only
  repo-facing workflow handles surfaced while hiding internal control files and
  legacy evidence-path refs.
- After the archived candidate became unmergeable against updated `origin/main`,
  revision 2 reopened in `finalize-fix`, merged the new mainline UI changes,
  rebuilt the embedded UI bundle, and reran full finalize review
  `review-019-full`.
- Finalize full review `review-019-full` passed with no findings; both
  `correctness` and `agent_ux` confirmed that the merge-conflict repair did
  not reintroduce hidden-path leaks or break repo-facing handoff surfaces.

## Archive Summary

- Archived At: 2026-04-16T21:27:09+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/178
- Ready: The candidate now narrows agent-facing path exposure across
  `harness status`, review command outputs, plan/review/timeline read models,
  timeline artifact refs and raw payloads, evidence-submit output, the schema
  index, and the managed reviewer/controller guidance. The final repair loop
  also closed the remaining legacy leaks in raw timeline JSON, reviewer plan
  discovery, and historical evidence labels such as `publish_record`,
  `ci_record`, and `sync_record`. Finalize full review `review-018-full`
  passed cleanly. Revision 2 then reopened solely because the archived
  candidate conflicted with updated `origin/main`; the repair merged the new
  mainline UI changes, rebuilt the embedded UI bundle, passed `go test ./...`,
  and then passed full finalize review `review-019-full` with no findings.
- Merge Handoff: Push the refreshed candidate on PR #178, refresh
  publish/CI/sync evidence for the new head, and then wait for explicit merge
  approval once `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Narrowed the retained-path contract to the agent-facing workflow handles that
  matter in normal use: repo-facing `plan_path`, `project_root`, `round_id`,
  reviewer-owned `submission.json`, and similar stable handles.
- Removed low-value internal control paths from `harness status`, review start,
  review submit, and review aggregate outputs while preserving the agent-usable
  review handles.
- Applied the same boundary to `/api/plan`, `/api/review`, `/api/timeline`,
  timeline artifact refs, and raw timeline payload sanitization so hidden
  control files, state pointers, and index directories no longer leak back into
  UI-facing reads.
- Narrowed evidence-submit output and legacy timeline/event payload handling so
  old `record_path`, `submission_path`, `submissions_dir`, `publish_record`,
  `ci_record`, and `sync_record` shapes are scrubbed from the surfaced
  contract.
- Updated the CLI spec, state-model wording, managed reviewer/controller
  skills, schema index, generated schemas, frontend helpers/types, and
  regression suites so the new agent-facing-versus-internal-control rule is
  durable.
- Reopened the archived candidate for revision 2 when the PR became
  unmergeable against `origin/main`, merged the new mainline UI changes into
  the branch, resolved the generated static bundle conflicts by rebuilding the
  embedded UI assets, and revalidated the merged candidate before re-review.

### Not Delivered

- This slice does not add a non-default maintainer/debug browser for hidden
  control artifacts.
- This slice does not extend the same cleanup pass to every other command
  output beyond the issue-169 surfaces.
- This slice does not redesign the underlying local review/state/timeline
  storage layout.

### Follow-Up Issues

- #176 Track a non-default maintainer/debug surface for hidden harness control
  artifacts
- #177 Extend agent-facing path cleanup beyond the issue-169 review and
  timeline surfaces
