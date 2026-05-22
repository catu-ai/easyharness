---
template_version: 0.2.0
created_at: "2026-05-22T23:58:56+08:00"
approved_at: "2026-05-23T00:00:36+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/213
size: S
---

# Harden Evidence Refresh Workflow Integration

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #213 by hardening `harness evidence refresh` so controllers can
prefer it for ordinary archived-candidate publish/CI/sync handoff after
publish evidence records a PR URL.

The command should bound its read-only GitHub CLI calls, degrade without
writing misleading evidence when remote reads hang or fail, and return enough
machine-readable refresh detail for an agent to understand what was just
recorded before it runs `harness status`.

## Scope

### In Scope

- Add bounded timeout behavior around the `gh pr view` and `gh pr checks`
  reads used by `harness evidence refresh`.
- Classify timeout results as degraded remote reads and avoid writing evidence
  for domains whose facts were not confidently refreshed.
- Extend the `harness evidence refresh` result contract with a
  machine-readable refreshed/observed status surface, including CI and sync
  statuses when those domains are written.
- Update generated command schemas and contract references for the changed
  refresh result payload.
- Update repo-local `harness-execute` publish/CI/sync guidance so future
  controllers prefer:
  1. `harness evidence refresh` when publish evidence has a recorded PR URL.
  2. `harness status` after refresh.
  3. manual `harness evidence submit --kind publish|ci|sync` fallback when
     refresh degrades, is unavailable, or lacks a recorded PR URL.
- Refresh materialized bootstrap assets after editing
  `assets/bootstrap/skills/harness-execute/...`.

### Out of Scope

- Richer live remote handoff facts in `harness status`; that remains issue
  #200.
- Broad controller workflow redesign beyond the publish/CI/sync refresh
  guidance needed for this issue.
- Configurable timeout settings or repository-specific timeout policy.
- Opening, updating, commenting on, labeling, reviewing, rerunning checks for,
  or merging PRs.
- Guessing a PR from the current branch when publish evidence lacks a recorded
  PR URL.
- Replacing manual `harness evidence submit` fallback paths.

## Acceptance Criteria

- [ ] `gh pr view` and `gh pr checks` calls used by refresh have bounded
      runtime in the default command runner.
- [ ] Timeout and other remote read failures degrade cleanly and do not write
      misleading CI or sync evidence.
- [ ] Successful and partial refresh results expose machine-readable status
      details for the domains actually written, without requiring agents to
      parse summary text.
- [ ] The public refresh result schema and contract artifacts match the new
      output shape.
- [ ] `harness-execute` publish/CI/sync guidance names refresh as the ordinary
      first path after publish evidence records a PR URL, with `harness status`
      next and manual submit as fallback.
- [ ] Focused tests cover success, partial refresh, timeout degradation, and
      no-evidence timeout/failure behavior.

## Deferred Items

- Issue #200 remains responsible for richer remote handoff facts and status
  surfaces.
- Issue #202 remains responsible for broader controller documentation and
  skill updates after the handoff workflow settles.

## Work Breakdown

### Step 1: Bound remote refresh reads

- Done: [x]

#### Objective

Ensure refresh cannot hang indefinitely while reading the recorded PR or PR
checks through `gh`.

#### Details

Use a conservative fixed timeout in the default remote command runner rather
than adding a user-facing setting. The timeout should apply to read-only `gh`
operations used by `harness evidence refresh`, while keeping the existing
injectable command runner simple enough for tests to simulate success,
failure, and timeout outcomes.

Timeouts should classify as degraded remote reads. If PR observation times out,
refresh should write no CI or sync evidence. If checks observation times out
after PR observation succeeds, refresh may still write sync evidence when the
merge state is clear, but it must not write CI evidence.

#### Expected Files

- `internal/remote/identity.go`
- `internal/remote/gh_test.go`
- `internal/evidence/service_test.go`

#### Validation

- Unit tests prove the default runner returns a timeout failure instead of
  waiting forever.
- Remote and evidence tests prove timeout degradation preserves the existing
  no-misleading-evidence behavior.

#### Execution Notes

Added a fixed 30-second timeout to the default remote command runner used for
read-only `gh` calls, plus `gh_timeout` degradation classification. Added
focused tests proving the default runner returns `context.DeadlineExceeded`
for slow commands, PR observation timeout writes no evidence, and checks
timeout can still write sync evidence when merge state is clear. Validation:
`go test ./internal/remote ./internal/evidence`.

After `review-001-delta`, tightened timeout handling so `gh pr checks`
timeouts degrade even when stdout is non-empty, added a finite `WaitDelay` for
held-pipe cases, and added focused regression coverage. Validation:
`go test ./internal/remote ./internal/evidence -count=1`.

#### Review Notes

`review-001-delta` found timeout reads with non-empty stdout could still write
CI evidence, the default runner lacked `WaitDelay`, and tests missed the
non-empty stdout path. The repair addressed all three findings, and
`review-002-delta` passed clean.

### Step 2: Return refreshed statuses from the command result

- Done: [x]

#### Objective

Make `harness evidence refresh` output directly useful to controllers by
returning the machine-readable CI and sync statuses it just recorded.

#### Details

Add a semantic refreshed or observed result object rather than making agents
infer status from prose or record IDs. The result should expose status fields
only for domains whose evidence was actually written, and warnings/next
actions should continue to guide manual fallback for degraded domains.

The exact field names should be chosen to read clearly in the JSON contract,
for example a `refreshed` object with `ci_status` and `sync_status` fields.
Keep the payload compact; this slice does not need to embed full check rows or
PR metadata.

#### Expected Files

- `internal/contracts/evidence.go`
- `internal/evidence/service.go`
- `internal/evidence/service_test.go`
- `schema/commands/evidence.refresh.result.schema.json`
- `schema/index.json`
- any generated contract reference artifacts touched by the repository's
  contract sync workflow

#### Validation

- Service and CLI tests assert that full refresh returns both statuses, partial
  refresh returns only the written domain's status, and all-failed refresh
  omits misleading success detail.
- Contract/schema sync checks pass after regenerating artifacts.

#### Execution Notes

Added `refreshed` to the `harness evidence refresh` result contract with
compact `ci_status` and `sync_status` fields for evidence domains written by
the refresh. Wired service and CLI outputs to populate the statuses for full
and partial refreshes while omitting the object on failed refreshes. Refreshed
the generated schema artifacts. Validation:
`go test ./internal/evidence ./internal/cli -run 'TestRefresh|TestEvidenceRefresh'`;
`scripts/sync-contract-artifacts --check`;
`go test ./internal/evidence ./internal/cli ./internal/contractsync -count=1`.

After `review-003-delta`, added CLI coverage proving partial refresh JSON
omits the unwritten status key. Validation:
`go test ./internal/cli -run 'TestEvidenceRefreshCommand(PartialOutputOmitsUnwrittenStatus|WritesEvidenceAndUpdatesStatus|MapsNonSuccessRemoteStates)' -count=1`;
`scripts/sync-contract-artifacts --check`;
`go test ./internal/evidence ./internal/cli ./internal/contractsync -count=1`.

#### Review Notes

`review-003-delta` found a missing CLI assertion for partial refresh JSON
omitting unwritten status keys. The repair added that coverage; follow-up
`review-004-delta` passed clean.

### Step 3: Refresh controller guidance

- Done: [x]

#### Objective

Teach future controller agents to use refresh as the ordinary publish/CI/sync
handoff path once publish evidence contains a recorded PR URL.

#### Details

Update the managed bootstrap `harness-execute` reference material in
`assets/bootstrap/` first, then run the bootstrap sync script so
`.agents/skills/` and the managed instructions block stay materialized from
the source assets. The guidance should make direct `gh` inspection diagnostic,
not the first path for routine evidence refresh.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/references/publish-ci-sync.md`
- `.agents/skills/harness-execute/references/publish-ci-sync.md`
- `AGENTS.md`, only if the bootstrap sync script refreshes the managed block

#### Validation

- `scripts/sync-bootstrap-assets --check` passes after the asset refresh.
- The updated guidance clearly lists refresh, status, and manual submit
  fallback in that order.

#### Execution Notes

Updated the managed `harness-execute` publish/CI/sync guidance so archived
candidate handoff records publish evidence first, then prefers
`harness evidence refresh`, then runs `harness status`, with manual
`harness evidence submit --kind publish|ci|sync` as fallback for degraded or
unavailable refresh paths. Clarified direct `gh` inspection as a diagnostic
fallback rather than the routine refresh path. Refreshed materialized
`.agents/skills/` output from `assets/bootstrap/`. Validation:
`scripts/sync-bootstrap-assets --check`.

After `review-005-delta`, also updated the managed closeout/archive handoff
checklist so it no longer presents manual CI/sync submit as the ordinary path.
The checklist now records publish evidence, runs `harness evidence refresh`,
runs `harness status`, and keeps manual submit as fallback for degraded or
unavailable refresh paths. Validation: `scripts/sync-bootstrap-assets --check`.

#### Review Notes

`review-005-delta` found the managed closeout/archive checklist still taught
manual evidence submit as the ordinary handoff path. The repair aligned that
checklist with refresh-first guidance, and `review-006-delta` passed clean.

## Validation Strategy

- Run focused Go tests for remote observation, evidence refresh, and CLI
  output behavior.
- Run the contract artifact sync check after updating the refresh result
  schema.
- Run the bootstrap asset sync check after updating managed skill guidance.
- Run `harness plan lint` before approval and again after any plan edits.

## Risks

- Risk: A timeout could be classified too broadly and hide an actionable auth
  or provider error.
  - Mitigation: Preserve existing `gh` failure classifications where possible
    and add timeout-specific coverage for only elapsed reads.
- Risk: The result contract could grow into the broader status surface tracked
  by issue #200.
  - Mitigation: Return only the statuses written by this refresh command and
    keep full remote handoff surfacing out of scope.
- Risk: Updating managed bootstrap guidance by hand could drift from
  materialized `.agents/skills/`.
  - Mitigation: Edit `assets/bootstrap/` first and use
    `scripts/sync-bootstrap-assets` to refresh generated outputs.

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

- #200 remains the follow-up for richer remote handoff facts in status.
- #202 remains the follow-up for broader controller documentation and skills.
