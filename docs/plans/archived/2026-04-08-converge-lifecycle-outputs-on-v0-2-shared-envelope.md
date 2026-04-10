---
template_version: 0.2.0
created_at: "2026-04-08T10:00:00+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/110
size: L
---

# Converge lifecycle outputs on the v0.2 shared envelope

## Goal

Remove the remaining lifecycle-oriented command-result shape that still emits
`state.plan_status` and `state.lifecycle`, and instead make the lifecycle
commands speak the same v0.2 output language already used by `harness status`:
`state.current_node`, selected `facts`, stable `artifacts`, and ordered
`next_actions`.

This slice should leave the underlying workflow and state-transition model
alone. The change is an output-contract cleanup: make
`harness execute start`, `harness archive`, `harness reopen`,
`harness land`, and `harness land complete` report their post-command state in
the same envelope vocabulary as the rest of the CLI, without preserving a
second legacy lifecycle-output contract.

## Scope

### In Scope

- Replace the shared lifecycle output contract under `internal/contracts/`
  so lifecycle commands return a v0.2-style envelope centered on
  `state.current_node`.
- Decide which lifecycle-specific details still belong in `facts`, such as
  `revision`, `reopen_mode`, `land_pr_url`, and `land_commit`.
- Keep stable transition pointers in `artifacts`, including plan move paths and
  relevant local-state pointers, while aligning the field set with the shared
  output vocabulary.
- Update command implementations, checked-in schemas, CLI contract prose, and
  tests together so the public command outputs are internally consistent.
- Make the intentional breaking change explicit in the plan summaries, issue
  context, and validation coverage.

### Out of Scope

- Changing the canonical node model, plan lifecycle rules, or state transition
  matrix itself.
- Adding compatibility shims that preserve `plan_status` or `lifecycle`
  alongside the new envelope.
- Reworking evidence, review, or status command outputs beyond any shared
  helper adjustments required by this cleanup.
- UI-specific read-model changes unless a compile/test break forces a narrow
  adaptation to the new command-result structs.

## Acceptance Criteria

- [x] `harness execute start`, `harness archive`, `harness reopen`,
      `harness land`, and `harness land complete` all return envelopes whose
      `state` is expressed through `current_node`, not `plan_status` or
      `lifecycle`.
- [x] Lifecycle-specific details that still matter after the cleanup move into
      `facts` and remain concise, stable, and command-relevant.
- [x] Any plan-path and local-state transition pointers remain available in
      `artifacts` where they are still useful after the contract convergence.
- [x] The lifecycle contract schema, checked-in generated artifacts, and
      `docs/specs/cli-contract.md` all describe the same final output shape.
- [x] Focused automated tests cover the updated lifecycle command outputs and
      prove the old `plan_status/lifecycle` fields are gone from the public
      payloads.
- [x] No compatibility bridge or dual-output path remains in the final
      implementation.

## Deferred Items

- Any lifecycle workflow redesign beyond output-contract cleanup.
- Any extension of the shared envelope cleanup to non-lifecycle commands that
  already match the v0.2 vocabulary.
- UI/read-model refactors that are not directly required by this contract
  convergence.

## Work Breakdown

### Step 1: Define the target lifecycle envelope and update the shared contract surface

- Done: [x]

#### Objective

Replace the legacy lifecycle output struct with a v0.2-aligned command result
shape and make the checked-in contract artifacts reflect that new shape.

#### Details

This step should inventory the fields currently emitted by lifecycle commands
and choose their final homes in the shared envelope. `state.current_node`
should become the only state field. `revision`, `reopen_mode`, `land_pr_url`,
and `land_commit` are the most likely `facts` candidates, but only if each one
is actually meaningful for the specific command. Any transition-specific plan
paths should stay in `artifacts` rather than leaking back into `facts`. Once
the field map is settled, update `internal/contracts`, schema generation, and
the CLI contract prose together so the repository has one documented end-state.

#### Expected Files

- `internal/contracts/lifecycle.go`
- `internal/contracts/registry.go` if schema metadata needs adjustment
- generated schema artifacts under `schema/`
- `docs/specs/cli-contract.md`

#### Validation

- Contract generation artifacts are updated and in sync with the chosen
  lifecycle envelope.
- The prose CLI contract no longer describes lifecycle commands as a special
  legacy output shape.
- No legacy `plan_status` or `lifecycle` field remains in the public lifecycle
  output contract surface.

#### Execution Notes

Replaced the legacy lifecycle output contract with a v0.2-aligned envelope:
`state.current_node` is now the only state field, while `revision`,
`reopen_mode`, `land_pr_url`, and `land_commit` moved into `facts` as needed.
Regenerated `schema/commands/lifecycle.result.schema.json` and updated
`docs/specs/cli-contract.md` so the shared output-envelope prose now treats
`execute start`, `archive`, `reopen`, `land`, and `land complete` as ordinary
v0.2 envelope users rather than legacy lifecycle exceptions.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 is tightly coupled to the runtime return-shape
cutover in Step 2, so reviewing the contract-only slice in isolation would be
artificially narrow.

### Step 2: Move lifecycle command implementations and tests onto the shared envelope

- Done: [x]

#### Objective

Update lifecycle command handlers and their tests so runtime outputs match the
new shared contract exactly.

#### Details

The implementation should keep command behavior and transition side effects the
same while changing only the returned JSON envelope. Each lifecycle command
should report its post-command node in `state.current_node`, populate concise
`facts` only when helpful, and keep `artifacts` focused on stable transition
pointers. Tests should stop asserting `plan_status/lifecycle` and instead lock
the new node/facts/artifacts shape, including land and reopen-specific details.
Any helper or fixture code that decodes lifecycle results should be updated in
the same slice so there is no hidden dependency on the old contract.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/cli/app_test.go`
- focused lifecycle/status/timeline tests under `internal/`
- any repo-level E2E coverage that decodes lifecycle command results

#### Validation

- Lifecycle command tests assert `current_node`-based outputs and pass.
- Focused CLI or E2E coverage proves the lifecycle commands no longer emit
  `plan_status/lifecycle`.
- The command outputs remain actionable, including transition artifacts and
  next-action guidance.

#### Execution Notes

Updated lifecycle command implementations, CLI timeline detail generation, and
CLI/E2E payload assertions to use `current_node` plus lifecycle `facts`
instead of `plan_status/lifecycle`. Focused validation:
`go test ./internal/lifecycle ./internal/cli ./internal/status ./internal/timeline ./tests/e2e/...`
and `scripts/sync-contract-artifacts --check`. The CLI tests now assert that
public lifecycle payloads expose `current_node`/`facts` and explicitly omit the
old `plan_status` and `lifecycle` fields.

#### Review Notes

`review-001-delta` requested changes because built-binary lifecycle E2E decode
still ignored extra legacy keys, so the first pass did not truly prove
`plan_status/lifecycle` disappeared from public payloads. Follow-up repair
`review-002-delta` closed the archive/reopen/lightweight/land gap, and
`review-003-delta` then passed cleanly after extending the same raw absence
checks to built-binary `execute start`. Finalize review `review-004-full`
requested one more repair because built-binary `archive`, `land`, and
`land complete` still did not positively assert the new `current_node` and
`facts` fields. The follow-up fix added explicit shared-envelope assertions to
those E2E payloads and revalidated with
`go test ./tests/e2e/... ./internal/cli ./internal/lifecycle ./internal/status ./internal/timeline`.
Finalize follow-up `review-005-delta` passed after that repair, but the next
full finalize pass `review-006-full` still found one remaining proof gap: the
shared helpers did not explicitly reject legacy `state.revision`. The final
repair tightened both the built-binary helper and the CLI JSON assertions to
fail if `state.revision` is still emitted, and reran the same focused test
suite cleanly.

## Validation Strategy

- Run focused package tests for lifecycle command handlers, CLI output shaping,
  and any status/timeline code that depends on shared result structs.
- Re-run contract-sync checks so checked-in schemas stay aligned with the new
  lifecycle envelope.
- Run targeted CLI or E2E coverage for at least one full lifecycle path that
  exercises `execute start`, `archive`, `reopen`, `land`, and
  `land complete` payload assertions.

## Risks

- Risk: Moving lifecycle-specific fields into the shared envelope could make one
  or more command outputs lose useful transition context.
  - Mitigation: explicitly inventory each currently returned field and decide
    whether it belongs in `facts`, `artifacts`, or should be dropped.
- Risk: Tests or helper code may still decode the old lifecycle struct even if
  the command implementations are updated.
  - Mitigation: search for all lifecycle-result decoders and update assertions
    in the same slice instead of relying on compile errors alone.
- Risk: The cleanup could accidentally diverge from `status` node vocabulary
  and create two interpretations of post-command state.
  - Mitigation: reuse the canonical `current_node` vocabulary already emitted
    by `status` and assert concrete node values in tests.

## Validation Summary

Validated the lifecycle-envelope convergence with focused contract, unit, and
built-binary coverage. The final green pass was
`go test ./tests/e2e/... ./internal/cli ./internal/lifecycle ./internal/status ./internal/timeline`,
which now proves `execute start`, `archive`, `reopen`, `land`, and
`land complete` emit the shared v0.2 envelope and reject the legacy lifecycle
state fields in raw JSON assertions. Contract artifacts were also regenerated
earlier in the slice through `scripts/sync-contract-artifacts` and checked with
`scripts/sync-contract-artifacts --check`.

## Review Summary

Step 2 initially needed three delta passes before the built-binary proof was
complete: `review-001-delta` caught legacy-key gaps in lifecycle E2E decoding,
`review-002-delta` closed the archive/reopen/lightweight/land helper gap, and
`review-003-delta` closed the matching `execute start` gap. Finalize review
then needed two repair cycles: `review-004-full` required positive built-binary
assertions for `archive`, `land`, and `land complete`, `review-005-delta`
passed after that repair, `review-006-full` caught one last missing rejection
of legacy `state.revision`, and the final reread `review-007-full` passed
cleanly with no findings.

## Archive Summary

- Archived At: 2026-04-08T10:07:37+08:00
- Revision: 1
- PR: NONE. Open or refresh the PR after the archive move is committed, then
  record the PR URL through publish evidence.
- Ready: Acceptance criteria are satisfied, lifecycle command outputs now share
  the v0.2 envelope vocabulary, contract/spec/schema artifacts agree, and the
  final full finalize review `review-007-full` passed after repairing the
  built-binary proof gaps for legacy lifecycle fields.
- Merge Handoff: Run `harness plan lint`, archive the plan, commit the tracked
  archive move plus the lifecycle contract/test updates, push branch
  `codex/issue-110`, open or update the PR, and record publish/CI/sync
  evidence until `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Converged lifecycle command outputs on the shared v0.2 envelope by replacing
  legacy `state.plan_status` / `state.lifecycle` usage with
  `state.current_node`, concise lifecycle `facts`, stable `artifacts`, and the
  existing next-action guidance.
- Updated the lifecycle contract/spec/schema surface so `internal/contracts`,
  checked-in schemas, and `docs/specs/cli-contract.md` all describe the same
  post-command output language.
- Updated timeline and CLI assertions plus built-binary E2E helpers so the
  public payloads now positively prove the shared envelope and explicitly fail
  if legacy lifecycle state fields reappear.

### Not Delivered

- No workflow-state redesign or node-model changes were attempted beyond the
  output-contract cleanup.
- No UI/read-model cleanup beyond the lifecycle command-result convergence was
  attempted in this slice.

### Follow-Up Issues

- No new follow-up issue was created in this slice. The remaining deferred
  items stay as broader future-design work and should only be reopened when the
  repository intentionally revisits lifecycle workflow/state modeling beyond
  output-contract cleanup.
