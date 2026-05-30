---
template_version: 0.2.0
created_at: "2026-05-30T23:22:40+08:00"
approved_at: "2026-05-30T23:29:26+08:00"
source_type: direct_request
source_refs: []
size: M
---

# Simplify status evidence and remote handoff output

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Make `harness status` easier for a cold controller agent to read by reshaping
the evidence and remote-handoff output into a compact control surface. The
default status result should keep `state.current_node`, `summary`,
`next_actions`, warnings, blockers, and only the high-signal facts needed to
decide the next workflow move.

The end state should preserve the read-only boundary of `harness status`:
recorded local evidence remains authoritative for workflow progression, live
remote observation is only explanatory, and mutating evidence updates still
belong to `harness evidence refresh` or manual `harness evidence submit`.

## Scope

### In Scope

- Update the status contract and generated schemas so post-archive evidence
  facts are grouped under `facts.evidence.recorded`.
- Replace the default `facts.remote_handoff` provider-shaped tree with a
  compact `facts.evidence.remote` projection that includes observation
  completeness, workflow assessment, a short message, minimal PR state, and
  CI/sync statuses that `harness evidence refresh` would record.
- Keep `next_actions` as the only command recommendation surface; facts may
  explain state but must not include command fields such as
  `recommended_command`.
- Reduce default `artifacts` to workflow handles a controller needs to open a
  plan or continue review/land handoff; move evidence record IDs and raw
  provider diagnostics out of the default status payload.
- Update status implementation, tests, specs, generated schema artifacts, and
  repo-local skill guidance that names the old status fields.
- Preserve the read-only/mutating split between `harness status` and
  `harness evidence refresh`.

### Out of Scope

- Adding a normal controller requirement to run `harness status --details`,
  `-v`, or any second status command before acting.
- Making `harness status` append evidence or otherwise mutate workflow state.
- Removing `harness evidence refresh` or manual evidence submit fallbacks.
- Building a new remote provider integration or expanding beyond the existing
  recorded GitHub PR observation behavior.
- Redesigning unrelated status nodes, review orchestration, archive readiness,
  or land bookkeeping semantics.
- Preserving backward-compatible duplicate legacy status fields unless a
  specific test or consumer requires a short-lived compatibility decision to be
  explicitly approved.

## Acceptance Criteria

- [ ] Default `harness status` remains the single agent-facing status entrypoint
      and does not require agents to know when to request a verbose/details
      variant.
- [ ] Archived-candidate status groups local durable evidence under
      `facts.evidence.recorded`, with publish, CI, and sync statuses clearly
      separated from live remote observation.
- [ ] Live remote observation is exposed through a compact
      `facts.evidence.remote` object rather than the current provider-shaped
      `facts.remote_handoff` dump.
- [ ] `facts.evidence.remote.observation` uses a small clear vocabulary such as
      `complete`, `partial`, and `unavailable` to describe observation
      completeness.
- [ ] `facts.evidence.remote.assessment` uses an agent-facing workflow
      vocabulary such as `matches_recorded`, `refresh_available`,
      `wait_for_remote`, `repair_remote`, `manual_evidence_required`, and
      `candidate_invalidated`.
- [ ] Default remote facts include only high-signal fields: a human-readable
      message, minimal PR state/draft information when relevant, and the remote
      CI/sync evidence statuses that refresh would record.
- [ ] Default status output does not include raw provider check rows, provider
      merge-state internals, head commit OIDs, duplicate PR URLs under remote
      facts, or evidence record IDs unless they are needed for the current
      workflow node.
- [ ] `next_actions` remains the sole command recommendation surface; no fact
      field duplicates commands or tells the agent what to run.
- [ ] Status remains read-only and cannot advance to
      `execution/finalize/await_merge` from live remote facts alone; durable
      recorded publish, CI, and sync evidence still determine that node.
- [ ] `harness evidence refresh` continues to be the mutating bridge that
      records observed CI and sync facts, and its output remains consistent with
      the compact status remote assessment.
- [ ] Specs, generated schemas, implementation tests, CLI/e2e tests, and
      bootstrap skill guidance agree on the new status field names and
      semantics.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Define the compact status evidence contract

- Done: [x]

#### Objective

Update the normative contract so the new status shape is clear before runtime
code changes.

#### Details

The contract should state that `harness status` is still the only normal
agent-facing status command. Do not introduce a routine `--details` workflow
for controllers. If a diagnostic variant is mentioned, make it explicitly
optional for humans/debugging and outside the normal agent loop.

Define `facts.evidence.recorded` as durable local evidence and
`facts.evidence.remote` as a compact read-only projection of live remote
observation. Recorded evidence decides workflow progression. Remote evidence
only explains drift, refresh opportunities, or manual fallback needs.

The proposed default shape is:

```json
{
  "facts": {
    "evidence": {
      "recorded": {
        "publish": {
          "status": "recorded",
          "pr_url": "https://github.com/example/repo/pull/123"
        },
        "ci": {
          "status": "pending"
        },
        "sync": {
          "status": "stale"
        }
      },
      "remote": {
        "observation": "complete",
        "assessment": "refresh_available",
        "message": "Remote PR checks are passing and merge state is fresh; recorded evidence has not been refreshed yet.",
        "pr": {
          "state": "open",
          "draft": false
        },
        "ci": {
          "status": "success"
        },
        "sync": {
          "status": "fresh"
        }
      }
    }
  },
  "next_actions": [
    {
      "command": "harness evidence refresh",
      "description": "Record the current remote CI and sync facts as durable evidence."
    }
  ]
}
```

Use `recorded`, not `local`, because these fields are durable evidence records,
not an arbitrary local git snapshot. Use `message`, not `summary`, inside the
remote object so it does not compete with the top-level status summary.

Suggested `remote.observation` values:

- `complete`: recorded PR, CI, and sync observations were all readable and
  classifiable
- `partial`: at least one useful remote domain was readable, but another
  domain degraded or could not be classified
- `unavailable`: no useful remote facts could be observed

Suggested `remote.assessment` values:

- `matches_recorded`: remote facts align with the durable evidence that already
  drives the node
- `refresh_available`: remote facts can update missing, pending, stale, failed,
  or conflicted recorded evidence
- `wait_for_remote`: remote checks, draft state, or other remote conditions are
  not ready yet
- `repair_remote`: remote facts show failing CI, stale sync, conflicts, or other
  repair-sensitive conditions before merge-ready handoff
- `manual_evidence_required`: remote observation is unavailable or incomplete
  enough that the agent needs manual evidence fallback
- `candidate_invalidated`: recorded evidence currently says merge-ready, but
  live remote facts show the candidate should no longer be treated as
  merge-ready

Only `next_actions` may recommend commands. The assessment explains why a
command is present; it is not itself the command source.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`
- `docs/specs/index.md`
- `internal/contracts/status.go`
- `schema/commands/status.result.schema.json`
- generated contract/schema references touched by the repository's
  contract-sync tooling

#### Validation

- `harness plan lint docs/plans/active/2026-05-30-simplify-status-evidence-remote-handoff-output.md`
- Run the repository's contract/schema generation or sync check used for status
  result schema changes.
- Focused contract tests should prove the new status structs and generated
  schema expose `facts.evidence.recorded` and `facts.evidence.remote`, not the
  old default `facts.remote_handoff` tree.

#### Execution Notes

Updated the public status contract around grouped archived-candidate evidence:
`facts.evidence.recorded` now owns durable publish/CI/sync facts, while
`facts.evidence.remote` owns compact read-only remote observation with
`observation`, `assessment`, `message`, minimal PR state, remote CI/sync
statuses, and compact degradation codes. Regenerated checked-in JSON schemas
with `go run ./cmd/contract-sync` and verified them with
`go run ./cmd/contract-sync --check`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 was implemented together with the runtime and
guidance changes so the meaningful review boundary is the final full candidate
review; focused contract-sync and status tests cover the contract shape.

### Step 2: Implement status evidence and remote projection

- Done: [x]

#### Objective

Change `harness status` runtime output to emit grouped recorded evidence and
compact remote assessment while preserving workflow behavior.

#### Details

Refactor the current status facts assembly so archived candidates populate
`facts.evidence.recorded.publish`, `facts.evidence.recorded.ci`, and
`facts.evidence.recorded.sync`. These recorded values replace the flat
`publish_status`, `pr_url`, `ci_status`, and `sync_status` defaults for the
archived evidence handoff surface.

Map the existing remote observation into `facts.evidence.remote`:

- `observation` summarizes read completeness (`complete`, `partial`,
  `unavailable`)
- `assessment` summarizes workflow meaning without prescribing a command
- `message` explains the relationship in one sentence
- `pr.state` and `pr.draft` are included only when observed and useful
- `ci.status` and `sync.status` report the statuses that refresh would record
- degraded details are compact; retain stable degradation codes when they help
  choose manual fallback, but do not expose raw provider dumps by default

Preserve all node decisions. In particular, `execution/finalize/await_merge`
must still require durable recorded publish evidence with a PR URL plus
recorded CI success/not_applied and sync fresh/not_applied.

Keep remote observation read-only. `harness status` may still read the recorded
PR to guide `next_actions`, but it must not append evidence or mutate
workflow state.

#### Expected Files

- `internal/contracts/status.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/cli/app_test.go`
- `tests/e2e/*`
- `tests/smoke/*`
- generated schema files under `schema/` and `internal/inputschema/` if touched
  by contract generation

#### Validation

- Focused status tests for archived candidates with missing evidence, pending
  evidence, clean remote facts, failed remote facts, stale/conflicted sync,
  degraded CI-only or sync-only observation, unavailable remote observation,
  and locally merge-ready candidates whose live remote facts have drifted.
- CLI/e2e tests updated so consumers assert grouped evidence and compact remote
  assessment instead of `facts.remote_handoff` internals.
- `go test ./internal/status ./internal/cli ./internal/evidence ./internal/remote -count=1`

#### Execution Notes

Reworked status facts assembly to emit recorded evidence under
`facts.evidence.recorded` and compact remote facts under
`facts.evidence.remote`. Status remains read-only and evidence-driven:
`execution/finalize/await_merge` still depends on durable recorded publish,
CI, and sync evidence, while remote facts only affect assessment and
`next_actions`. Removed default status evidence record IDs and default
provider-shaped remote handoff details from normal status output.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 was validated as part of the same cohesive
contract/runtime slice and will receive full-candidate reviewer coverage before
archive. Focused status, CLI, e2e, UI, dashboard, and watchlist tests passed.

### Step 3: Align guidance, artifacts, and end-to-end behavior

- Done: [x]

#### Objective

Update agent-facing guidance and final validation so the new status surface is
the normal controller workflow.

#### Details

Refresh repo-local harness-execute references and bootstrap assets so agents
read `facts.evidence.recorded`, `facts.evidence.remote`, `state.current_node`,
warnings, blockers, and `next_actions` together. Remove guidance that tells
controllers to interpret raw `facts.remote_handoff` provider fields.

While touching status output, clean up default artifacts consistently:

- keep `plan_path`, `supplements_path`, active `review_round_id`, active
  `review_slots`, and landed handoff fields when they are needed for the
  current node
- omit default `project_root` when it only repeats the current workspace
- omit evidence record IDs from default status unless a current workflow action
  genuinely needs the exact record ID

If a diagnostic status surface is left for future work, document it only as
optional human/debugging support and not as part of the normal agent control
loop.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/publish-ci-sync.md`
- `assets/bootstrap/skills/harness-execute/references/controller-truth-surfaces.md`
- `.agents/skills/harness-execute/**` after bootstrap sync
- `AGENTS.md` if managed bootstrap sync changes it
- relevant status/dashboard/watchlist docs if they consume status artifacts

#### Validation

- Run `scripts/sync-bootstrap-assets` after editing bootstrap assets.
- Run focused tests that cover status output in CLI, dashboard/watchlist
  pass-throughs, and remote handoff guidance.
- Run `go test ./...` unless a narrower, justified validation set is recorded
  in Execution Notes.

#### Execution Notes

Updated harness-execute bootstrap guidance to point controllers at
`facts.evidence.recorded` and `facts.evidence.remote` instead of raw
`facts.remote_handoff`. Ran `scripts/sync-bootstrap-assets`, which refreshed
the materialized `.agents/skills/harness-execute` copies. End-to-end status
consumers now assert grouped evidence rather than flat evidence status fields
or evidence record IDs.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 is guidance and consumer alignment for the same
status contract change; final full review will check docs, agent UX, tests,
and runtime consistency across the whole candidate.

## Validation Strategy

- Start with contract/schema validation so the intended field shape is locked
  before runtime changes.
- Use focused unit tests for the status service's evidence and remote
  projection cases, because most ambiguity risk lives in classification and
  next-action guidance.
- Update CLI and e2e tests that consume status JSON so regressions in the
  public command payload are caught.
- Run bootstrap sync and relevant skill guidance checks after editing
  distributed agent instructions.
- Finish with `go test ./...` when feasible, plus `harness status` to confirm
  the current worktree remains in the expected workflow node.

## Risks

- Risk: The new compact fields could hide detail that a controller needs to
  repair a remote handoff problem.
  - Mitigation: Keep stable degradation codes and high-signal PR state in
    default output, and rely on `next_actions` for manual fallback when remote
    observation is incomplete.
- Risk: `remote.assessment` could become a second action source that conflicts
  with `next_actions`.
  - Mitigation: Document and test that assessment is explanatory only;
    commands appear only in `next_actions`.
- Risk: Grouping evidence fields may break status consumers that expect flat
  `ci_status` or `facts.remote_handoff`.
  - Mitigation: This repository is in a rapid development phase, so prefer the
    clean target schema. Update in-repo consumers and tests rather than adding
    compatibility shims by default.
- Risk: Status and evidence refresh both observe remote facts, which may look
  redundant.
  - Mitigation: Preserve the clear read-only versus mutating boundary. Status
    explains and guides; evidence refresh writes durable evidence.

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
