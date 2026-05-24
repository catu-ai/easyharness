---
template_version: 0.2.0
created_at: "2026-05-24T23:35:00+08:00"
approved_at: "2026-05-24T23:34:00+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/200
    - https://github.com/catu-ai/easyharness/issues/12
size: M
---

# Surface Remote Handoff Status

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #200 by making `harness status` surface read-only remote
handoff facts for an archived candidate with recorded PR evidence.

The end state should let a controller treat `harness status` as the primary
orientation command during publish and merge handoff: local durable evidence
continues to decide `current_node`, while optional live remote observation
explains what the recorded PR currently says about PR state, checks, and merge
freshness. `harness status` must remain non-mutating; `harness evidence
refresh` remains the explicit command that records remote CI and sync facts as
local evidence.

## Scope

### In Scope

- Extend the `harness status` contract with a compact remote observation
  surface anchored only on the PR URL recorded in latest publish evidence.
- Default to read-only observation of the recorded PR when status is in the
  archived publish or await-merge handoff path and a supported PR URL exists.
- Keep live remote facts non-authoritative: they may inform summaries,
  warnings, and `next_actions`, but must not advance `state.current_node` or
  replace local durable evidence.
- Degrade cleanly when `gh`, auth, network/API access, provider output, or the
  PR itself is unavailable; local status should still return `ok: true` when
  local harness state is readable.
- Preserve the no-guess identity rule: status must not discover a PR from the
  current branch when publish evidence lacks a PR URL.
- Add next actions for common handoff cases such as missing PR evidence,
  pending checks, failed checks, stale or conflicted sync, unreadable remote
  state, passing remote facts that still need `harness evidence refresh`, and
  already-recorded merge-ready local evidence.
- Update CLI contract text, generated status schema artifacts, and tests for
  the new status surface.

### Out of Scope

- Writing CI or sync evidence from `harness status`.
- Removing or weakening `harness evidence refresh`; it remains the explicit
  command for recording remote facts as durable local evidence.
- Opening, updating, commenting on, labeling, reviewing, rerunning checks for,
  or merging PRs.
- Inferring a PR from branch name, upstream remote, local git config, or GitHub
  PR lists.
- Provider support beyond the existing recorded GitHub PR URL model.
- Broader controller skill guidance for remote handoff beyond status-specific
  references; issue #202 remains the larger skill/docs integration bucket.

## Acceptance Criteria

- [ ] `harness status` includes remote handoff observation for archived
      candidates with recorded GitHub PR URLs when live observation succeeds.
- [ ] Remote observation failures appear as degraded remote facts, warnings, or
      manual fallback next actions without making local status fail.
- [ ] `state.current_node` remains evidence-driven: live remote passing checks
      and clean merge state do not move status to `execution/finalize/await_merge`
      until local CI and sync evidence are recorded.
- [ ] When live remote facts are passing or fresh but local evidence is missing
      or stale, `next_actions` steer the controller to `harness evidence
      refresh`.
- [ ] When live remote checks fail, are pending, or remote sync is stale or
      conflicted, `next_actions` explain the repair or wait path without
      writing evidence.
- [ ] Missing recorded PR evidence still steers the controller to publish and
      record publish evidence instead of attempting branch-based PR discovery.
- [ ] Tests cover successful live observation, degraded remote observation,
      non-authoritative remote facts, no-guess missing PR behavior, and
      preservation of existing evidence-driven publish and await-merge
      behavior.

## Deferred Items

- Issue #202: update broader controller docs and harness-execute skill
  guidance after the status surface lands.
- A future issue may add explicit flags or output filters if default live
  observation proves too noisy in dogfooding.

## Work Breakdown

### Step 1: Define the status contract for remote observation

- Done: [x]

#### Objective

Make the non-authoritative remote observation boundary explicit in normative
docs and generated contract types before changing status behavior.

#### Details

The contract should distinguish durable local evidence from live remote
observation. The remote surface should be compact enough for agents to use in
handoff decisions without dumping raw provider output. It should include a
clear status or degraded result, PR identity/state when available, CI/check
summary, sync or merge-state summary, and enough degradation information to
explain why live observation is unavailable.

This step should also state the key invariant: `harness status` may observe
the recorded PR URL, but it must not write evidence or advance
`state.current_node` based on live facts alone.

#### Expected Files

- `docs/specs/cli-contract.md`
- `internal/contracts/status.go`
- `internal/contracts/registry.go`
- `schema/commands/status.result.schema.json`
- `schema/index.json`

#### Validation

- Run contract generation or sync commands required by the repository.
- Run focused contract or schema tests if the changed files have package-local
  coverage.

#### Execution Notes

Defined `facts.remote_handoff` as the non-authoritative live remote
observation surface in `internal/contracts/status.go`, updated the CLI
contract to keep status read-only and evidence-driven, and regenerated the
status schema artifacts. Validation: `scripts/sync-contract-artifacts`,
`scripts/sync-contract-artifacts --check`, and
`go test ./internal/contracts ./internal/contractsync -count=1`.

#### Review Notes

`review-001-delta` found one important contract bug and one minor docs
inventory gap: value-typed sub-observation `degraded` fields would serialize
empty objects, and the top-level status facts inventory omitted
`remote_handoff`. Fixed both by switching sub-observation degraded fields to
`*StatusRemoteDegradation` and adding `remote_handoff` to the facts inventory.
Follow-up `review-002-delta` passed with zero findings.

### Step 2: Wire read-only remote observation into status

- Done: [x]

#### Objective

Teach `internal/status` to observe the recorded PR URL during archived handoff
status resolution while keeping local evidence as the only progression source.

#### Details

Reuse the existing `internal/remote` service and command runner boundary
instead of creating another GitHub read path. Observation should run only when
status has latest publish evidence with a supported recorded PR URL and the
current status node is in the archived publish or await-merge handoff path.

When observation succeeds, status should expose the remote facts and refine
summary or next-action text. When observation degrades, status should retain
the local publish or await-merge node, include the degraded remote state, and
preserve manual evidence submit fallback guidance. If no recorded PR URL
exists, status should keep the existing publish guidance and must not call
`gh`.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`

#### Validation

- Add or update unit tests proving status observes recorded PRs through an
  injectable command runner.
- Add or update tests for missing PR URL, `gh` failure, auth failure, pending
  checks, failed checks, clean remote facts without local evidence, and
  existing local merge-ready evidence.
- Run focused Go tests for status, remote, evidence, and CLI packages.

#### Execution Notes

Wired read-only remote observation into status behind an explicit
`status.Service.ObserveRemote` switch, with `harness status` enabling it and
passing the CLI's injectable `RunCommand`. Status now maps the existing
`internal/remote` handoff observation into `facts.remote_handoff`, keeps
`current_node` driven by local evidence, avoids branch-based PR guessing when
publish evidence is missing, and adds remote-specific next-action guidance for
pending or failed checks and stale or conflicted sync. Validation:
`go test ./internal/status ./internal/cli -run 'TestStatusArchivedPlanSurfacesRemoteHandoffObservation|TestStatusRemoteHandoffObservationDegradesWithoutFailingLocalStatus|TestStatusRemoteHandoffNextActionsExplainNonReadyRemoteFacts|TestStatusRemoteHandoffDoesNotGuessPRWhenPublishEvidenceMissing|TestStatusCommandSurfacesRemoteHandoffObservation' -count=1`;
`go test ./internal/status ./internal/cli ./internal/remote ./internal/evidence -count=1`;
`git diff --check`.

#### Review Notes

`review-003-delta` found two important gaps: await-merge status observed live
remote facts but did not include remote-specific wait/repair guidance, and the
tests did not cover remote observation when durable local evidence already
placed status in `execution/finalize/await_merge`. Fixed both by prepending
remote handoff guidance in the await-merge action list and adding
`TestStatusAwaitMergeIncludesRemoteHandoffWarningsWithoutRegressingNode`.
Follow-up `review-004-delta` passed with zero findings.

### Step 3: Align handoff guidance and full validation

- Done: [ ]

#### Objective

Make the delivered behavior coherent across docs, schemas, and end-to-end
workflow expectations, then validate the complete slice.

#### Details

Review the status next actions in realistic handoff states. A passing live
remote observation with missing local evidence should point to `harness
evidence refresh`; pending or failed checks should point to waiting or repair;
degraded remote observation should point to retrying refresh or manual
evidence submit fallbacks. The final wording should make it obvious that
`status` explains and `evidence refresh` records.

If the implementation changes the status JSON schema or generated contract
reference files, regenerate them and keep drift checks passing.

#### Expected Files

- `docs/specs/cli-contract.md`
- `internal/status/service_test.go`
- `internal/cli/app_test.go`
- `tests/e2e/...`
- `schema/...`

#### Validation

- Run focused Go tests for changed packages.
- Run relevant e2e coverage for archived publish and await-merge handoff.
- Run schema or contract drift checks.
- Run `git diff --check`.
- Run `harness plan lint` on this plan before execution starts and again if
  the plan changes materially.

#### Execution Notes

Ran the final guidance and workflow validation pass after the contract and
status implementation steps. Confirmed the generated contract artifacts are in
sync, the status/CLI/remote/evidence packages pass together, and the
publish/await-merge/land e2e paths still progress through local durable
evidence. Validation: `scripts/sync-contract-artifacts --check`;
`go test ./internal/status ./internal/cli ./internal/remote ./internal/evidence ./internal/contractsync -count=1`;
`go test ./tests/e2e -run 'TestPublishHandoff|TestAwaitMerge|TestLandWorkflow|TestLightweightWorkflow' -count=1`;
`git diff --check`.

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Start with focused unit coverage in `internal/status` using fake remote
  command runners to avoid real `gh`, auth, or network dependencies.
- Keep existing `internal/remote` and `internal/evidence` tests green so status
  reuses the same classification rules as `harness evidence refresh`.
- Add CLI-level coverage for JSON output and degraded remote observation where
  the command runner injection matters.
- Use relevant e2e coverage to confirm the archived publish and await-merge
  workflow still advances only after durable local evidence is present.
- Run contract/schema sync checks whenever status contract types change.

## Risks

- Risk: Live remote observation could make `harness status` feel flaky or slow.
  - Mitigation: Treat remote failures as degraded observations, preserve
    local `ok: true` status when local state is readable, and reuse the bounded
    remote read behavior from the existing refresh path.
- Risk: Agents could confuse live remote observations with durable evidence.
  - Mitigation: Name and document the remote surface as observation, keep
    `current_node` evidence-driven, and make next actions tell agents to run
    `harness evidence refresh` when remote facts need to be recorded.
- Risk: The status contract could grow too provider-shaped.
  - Mitigation: Expose normalized PR, CI, sync, and degraded summaries instead
    of raw `gh` payloads, and keep GitHub-specific details behind
    `internal/remote`.
- Risk: Default live observation could accidentally infer PR identity from
  local branch state.
  - Mitigation: Test the no-guess missing-PR case and call remote observation
    only from recorded publish evidence.

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
