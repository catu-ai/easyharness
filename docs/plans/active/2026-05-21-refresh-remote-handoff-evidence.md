---
template_version: 0.2.0
created_at: "2026-05-21T23:53:12+08:00"
approved_at: "2026-05-21T23:54:41+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/199
size: M
---

# Refresh Remote Handoff Evidence

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #199 by adding a read-only remote PR and CI read model, then
using it to refresh local `ci` and `sync` evidence for an archived candidate
with a recorded PR URL.

The end state should let a controller run a single harness command after
publish evidence exists and have harness read GitHub through `gh`, classify PR
checks and merge freshness, and append the corresponding local evidence. The
existing `harness evidence submit --kind publish|ci|sync` commands remain the
manual fallback when `gh`, auth, network access, or provider output is
unavailable.

## Scope

### In Scope

- Extend the recorded-PR remote read model from issue #203 so it can observe
  PR state, check status, and merge-readiness signals for a recorded GitHub PR.
- Add `harness evidence refresh` as an explicit mutating command that reads the
  recorded PR URL from latest publish evidence and appends `ci` and `sync`
  evidence when the remote facts are clear enough.
- Keep `gh` optional: missing `gh`, missing auth, network/API failures,
  unreadable PRs, invalid JSON, or unsupported provider output must return
  degraded command results without blocking manual evidence fallback.
- Preserve the no-guess identity rule: if publish evidence does not already
  contain a PR URL, refresh must not discover or infer a PR from the branch.
- Update CLI contracts, state-model contracts, generated schemas or command
  registry entries as needed for the new command surface.
- Add unit and focused CLI tests covering successful refresh and degraded read
  paths without requiring a real `gh` binary, auth, or network access.

### Out of Scope

- Surfacing live remote handoff facts and richer next actions through
  `harness status`; that belongs to issue #200.
- Opening, updating, commenting on, labeling, reviewing, rerunning checks for,
  or merging PRs.
- Guessing a PR from the current branch when publish evidence lacks a PR URL.
- Replacing the manual `harness evidence submit` fallback.
- Fetching full CI logs or deciding repository-specific merge policy beyond
  the compact evidence states already understood by harness.
- Introducing `.harness` customization or non-GitHub provider support.

## Acceptance Criteria

- [ ] A remote read model can classify recorded GitHub PR observation,
      aggregate check status, and merge freshness/conflict state using mockable
      command execution.
- [ ] `harness evidence refresh` reads the latest publish evidence PR URL and
      appends both `ci` and `sync` evidence when remote facts are clear.
- [ ] Refresh maps passing checks to `ci: success`, pending checks to
      `ci: pending`, failing/cancelled checks to `ci: failed`, clean/current
      merge state to `sync: fresh`, stale/behind/unknown-but-readable merge
      state to `sync: stale`, and conflict state to `sync: conflicted`.
- [ ] Missing recorded PR URL, unsupported PR URL, missing `gh`, missing auth,
      unreadable PRs, network/API failures, and invalid provider output degrade
      without writing misleading success evidence.
- [ ] Existing manual `harness evidence submit --kind publish|ci|sync` flows
      continue to work unchanged and remain the documented fallback.
- [ ] Tests cover success and degraded refresh paths, including the no-guess
      missing-PR case and partial evidence behavior when only one domain can be
      refreshed confidently.

## Deferred Items

- Issue #200: surface remote handoff facts and next actions in `harness
  status` without mutating workflow state.
- Issue #202: update controller docs and skills after the remote handoff
  command and status surfaces settle.
- Future customization work may define repository-specific remote mappings,
  but this slice keeps the core model evidence-first and GitHub-only.

## Work Breakdown

### Step 1: Specify refresh semantics and extend the remote read model

- Done: [x]

#### Objective

Define and implement the read-only provider boundary that turns a recorded PR
URL into normalized PR, checks, and merge-state observations.

#### Details

The read model should build on `internal/remote` from issue #203. It must keep
recorded publish `pr_url` as the only PR identity source and use `gh` only for
read operations against that recorded URL. The command execution boundary must
stay injectable so tests can simulate `gh pr view`, `gh pr checks`, missing
auth, invalid JSON, pending checks, failing checks, and conflict states.

The model should expose normalized outcomes instead of leaking raw `gh`
shapes throughout the codebase. If GitHub exposes both PR-level fields and
check rows, the implementation may use the smallest stable `gh` surface that
can distinguish passing, pending, failing, cancelled, unavailable, fresh,
stale, and conflicted outcomes.

#### Expected Files

- `internal/remote/...`
- `internal/remote/..._test.go`
- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- Contract schema or registry files if the read model affects public command
  output types

#### Validation

- Unit tests cover normalized PR/check/sync observation success and degraded
  paths using fake command execution.
- Tests assert that no branch-based PR discovery runs when recorded PR URL is
  missing.
- Specs explain that refresh is the mutating evidence command, while status
  remains read-only.

#### Execution Notes

Extended `internal/remote` with a normalized handoff observation that reads a
recorded PR through `gh pr view`, reads checks through `gh pr checks`, and
maps clear provider facts to local evidence statuses for later refresh:
`ci: success|pending|failed` and `sync: fresh|stale|conflicted`. The read
model keeps command execution injectable for tests, does not discover PRs from
branches, and degrades per domain when checks or merge state are unavailable.
Updated `docs/specs/state-model.md` and `docs/specs/cli-contract.md` to define
`harness evidence refresh` as the explicit mutating bridge while keeping
`harness status` read-only. Validation: `go test ./internal/remote`;
`harness plan lint
docs/plans/active/2026-05-21-refresh-remote-handoff-evidence.md`; `git diff
--check -- internal/remote docs/specs
docs/plans/active/2026-05-21-refresh-remote-handoff-evidence.md`.

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Add `harness evidence refresh`

- Done: [ ]

#### Objective

Add the explicit command that observes the recorded PR and writes append-only
`ci` and `sync` evidence for the current archived candidate.

#### Details

`harness evidence refresh` should only operate when the current plan is an
archived candidate in the publish/await-merge phase. It should load latest
publish evidence for the current revision, parse the recorded PR URL, observe
remote PR/check state, and append evidence records for domains whose facts are
clear enough.

The command should be all-or-clear per evidence domain rather than inventing
success when data is unavailable. For example, if checks are readable but merge
state is unavailable, refresh may write `ci` evidence and report degraded
`sync` refresh; if neither domain is clear, it should write no evidence and
steer the controller to manual `harness evidence submit`. The command must not
write publish evidence, mutate GitHub, update PRs, rerun checks, or alter local
workflow state outside the normal evidence append and timeline mechanics.

#### Expected Files

- `internal/evidence/...`
- `internal/cli/...`
- `internal/contracts/...`
- `internal/inputschema/...` or generated schema files if needed
- `docs/specs/cli-contract.md`

#### Validation

- CLI tests exercise successful refresh that writes both `ci` and `sync`
  evidence and causes existing status resolution to reach
  `execution/finalize/await_merge` when publish evidence is already recorded.
- CLI tests cover missing publish PR URL, unsupported PR URL, `gh` unavailable,
  auth unavailable, unreadable PR, invalid JSON, pending checks, failing
  checks, stale merge state, and conflicted merge state.
- Tests verify manual `evidence submit` commands still pass unchanged.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Close contracts, generated surfaces, and regression coverage

- Done: [ ]

#### Objective

Make the new refresh behavior durable across docs, schemas, help text, and
end-to-end workflow expectations.

#### Details

Refresh should appear in the public CLI contract as the automatic evidence
path for recorded PR handoff, with manual submit preserved as the fallback.
Any generated contract snapshots, command schemas, help text, or docs that
enumerate evidence commands must be updated together so future agents do not
learn conflicting command shapes.

Keep issue #200 out of scope: this step may add minimal next-action text that
points from publish-phase guidance toward `harness evidence refresh`, but it
should not build the richer status remote-facts surface.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`
- Contract registry/schema snapshots if generated by the repository workflow
- CLI/help tests and targeted workflow tests

#### Validation

- `go test ./internal/remote ./internal/evidence ./internal/cli`
- Broader `go test ./...` unless a narrower failure is justified in execution
  notes.
- `harness plan lint docs/plans/active/2026-05-21-refresh-remote-handoff-evidence.md`
- `git diff --check`

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

Validation should start with focused unit tests for the normalized remote read
model, then exercise the command through CLI-level tests that verify evidence
records are appended exactly when facts are clear. The full candidate should
run the relevant package tests plus `go test ./...` before archive because
evidence refresh touches command routing, public contracts, and status
progression indirectly through existing evidence reads.

## Risks

- Risk: Refresh could accidentally make `gh` a hard dependency for normal
  harness status or merge-readiness work.
  - Mitigation: Keep refresh as an explicit command, preserve submit fallback,
    and test missing `gh`/auth degraded results.
- Risk: Mapping GitHub merge/check states too aggressively could write
  misleading evidence.
  - Mitigation: Normalize only clear outcomes, degrade instead of guessing, and
    cover ambiguous provider output with negative tests.
- Risk: The new command could blur the #199/#200 boundary by adding a large
  status surface.
  - Mitigation: Limit status changes to existing evidence-driven behavior and
    defer richer remote facts/next actions to issue #200.

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

- https://github.com/catu-ai/easyharness/issues/200
- https://github.com/catu-ai/easyharness/issues/202
