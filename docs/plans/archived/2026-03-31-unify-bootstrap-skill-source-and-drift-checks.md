---
template_version: 0.2.0
created_at: "2026-03-31T13:25:46+08:00"
source_type: direct_request
source_refs: []
---

# Unify bootstrap skill source and drift checks

## Goal

Make `assets/bootstrap/` the only hand-edited source for the harness-managed
bootstrap contract that this repository dogsfoods and that release builds embed
for `harness install`. The dogfood copies under `.agents/skills/` and the
managed block content mirrored in this repository's `AGENTS.md` should become
materialized outputs derived from those packaged assets instead of a second set
of independently edited files.

The resulting workflow should stay self-contained for release users: `harness
install` must continue to install ordinary repository files with no network or
symlink dependency. In this repository, humans and agents should have one clear
editing surface, one explicit refresh path for dogfood outputs, and automated
drift checks that fail when the tracked materialized files diverge from
`assets/bootstrap/`.

## Scope

### In Scope

- Define and document `assets/bootstrap/` as the canonical editing surface for
  bootstrap skills and the harness-managed `AGENTS.md` block.
- Add a repo-local sync path that refreshes tracked dogfood outputs from
  `assets/bootstrap/` into `.agents/skills/` and the root `AGENTS.md` managed
  block while preserving repo-specific `AGENTS.md` guidance outside the managed
  markers.
- Update repo-visible guidance for both humans and agents so future edits land
  in the canonical source instead of the materialized outputs.
- Add automated drift validation so test or CI fails when tracked materialized
  outputs no longer match the canonical bootstrap assets.
- Preserve the current release contract: embedded bootstrap assets still ship in
  the binary and `harness install` still writes normal files into target repos.

### Out of Scope

- Switching dogfood skills to symlinks.
- Introducing a separate `bootstrap-src/` source tree in this slice.
- Changing `harness install` to fetch remote assets or install anything other
  than normal repository files.
- Broad redesign of the harness install scope model beyond the canonical-source
  and drift-guard changes above.

## Acceptance Criteria

- [x] Repository guidance states that bootstrap skill and managed-block edits
      belong in `assets/bootstrap/`, and that `.agents/skills/` is a
      materialized output that should not be edited directly.
- [x] A deterministic repo-local sync path can regenerate the tracked
      `.agents/skills/**` files and this repository's managed `AGENTS.md` block
      from `assets/bootstrap/` without disturbing repo-specific `AGENTS.md`
      content outside the markers.
- [x] Automated validation detects drift between `assets/bootstrap/` and the
      tracked dogfood outputs, so forgetting to resync fails local tests and
      CI paths that already run the relevant suite.
- [x] Existing bootstrap behavior remains intact for release users: embedded
      assets still back `harness install`, and install/smoke coverage continues
      to pass with the new canonical-source workflow.

## Deferred Items

- A later split between human-authored bootstrap source and generated
  embed-ready assets if `assets/bootstrap/` becomes too overloaded.
- Optional developer ergonomics such as a dedicated CLI subcommand for
  bootstrap syncing instead of a repo-local script.

## Work Breakdown

### Step 1: Define the canonical bootstrap-editing contract

- Done: [x]

#### Objective

Make the source-of-truth boundary explicit in tracked docs so a future agent can
tell, without chat history, which files are hand-edited and which are generated
dogfood outputs.

#### Details

Document that `assets/bootstrap/` is the only hand-edited source for packaged
bootstrap content in this repository. Clarify that `.agents/skills/` is a
tracked materialization of those packaged assets, and that the harness-managed
block inside this repository's `AGENTS.md` is likewise refreshed from
`assets/bootstrap/agents-managed-block.md` while the easyharness-specific
guidance outside the markers remains hand-owned repo documentation.

#### Expected Files

- `AGENTS.md`
- `README.md`
- `docs/specs/cli-contract.md`

#### Validation

- A cold reader can identify the canonical editing surface and the materialized
  outputs directly from tracked docs.
- The documented workflow still matches the existing install contract in
  `docs/specs/cli-contract.md`.

#### Execution Notes

Clarified the canonical editing contract in `AGENTS.md`, `README.md`, and
`docs/specs/cli-contract.md`: bootstrap-skill and managed-block edits now point
to `assets/bootstrap/`, while `.agents/skills/` and the root `AGENTS.md`
managed block are documented as tracked materialized outputs. Validation:
`scripts/sync-bootstrap-assets --check`, `go test ./internal/install`, and
`go test ./tests/smoke -run 'TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestInstall'`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is documentation-only and was reviewed
together with the coupled sync implementation in the later full finalize
review.

### Step 2: Add deterministic dogfood materialization from bootstrap assets

- Done: [x]

#### Objective

Introduce one repo-local sync path that refreshes this repository's tracked
dogfood bootstrap outputs from `assets/bootstrap/` instead of maintaining those
outputs by hand.

#### Details

The implementation should choose one clear maintenance entrypoint, such as a
repo-local script, and use it to refresh `.agents/skills/**` plus the managed
block inside the root `AGENTS.md`. The sync path must preserve repo-specific
guidance outside the managed markers, keep line-ending handling deterministic,
and avoid turning `harness install` itself into the repository-maintenance
mechanism. `assets/bootstrap/embed.go` should remain the source of packaged
release assets, not become another editable copy.

#### Expected Files

- `assets/bootstrap/embed.go`
- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/**`
- `scripts/sync-bootstrap-assets`
- `.agents/skills/**`
- `AGENTS.md`

#### Validation

- Running the chosen sync path after editing `assets/bootstrap/` deterministically
  updates the tracked dogfood outputs and produces no unrelated file churn.
- Existing install tests and smoke tests still pass after the sync mechanism is
  introduced or refactored.

#### Execution Notes

Added a repo-local sync entrypoint at `scripts/sync-bootstrap-assets`, backed by
`cmd/bootstrap-sync` and `internal/bootstrapsync`, so this repository can
refresh `.agents/skills/**` and the root `AGENTS.md` managed block from
`assets/bootstrap/` without using `harness install` as the human-facing
maintenance command. Finalize review then caught an orphan-file gap, so the
sync helper now also detects and removes stale materialized skill files that no
longer exist in `assets/bootstrap/`. Validation:
`scripts/sync-bootstrap-assets`, `scripts/sync-bootstrap-assets --check`,
`go test ./cmd/bootstrap-sync`, and `go test ./internal/bootstrapsync ./internal/install`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The sync helper and its enforcement tests landed as one
tightly coupled slice and are covered together by the branch-level full
finalize review.

### Step 3: Enforce drift checks in automated validation

- Done: [x]

#### Objective

Make drift between `assets/bootstrap/` and the tracked dogfood outputs fail
automatically instead of relying on contributor memory.

#### Details

Prefer enforcement that naturally rides on the repository's existing automated
test paths so CI picks it up without bespoke human steps. The checks should
verify both sides of the dogfood contract: `.agents/skills/**` matches the
packaged bootstrap skills, and the harness-managed block in the root
`AGENTS.md` matches `assets/bootstrap/agents-managed-block.md`. If the chosen
test suite is not already covered by CI, update the relevant workflow or test
entrypoint so the drift guard runs by default.

#### Expected Files

- `internal/install/service_test.go`
- `tests/smoke/smoke_test.go`
- additional focused test files under `internal/` or `tests/` if the execution
  path warrants them

#### Validation

- Intentionally introducing bootstrap drift causes the automated check to fail.
- After rerunning the sync path, the same automated check passes again.
- The repository's standard validation instructions make the drift guard easy to
  discover for both humans and future agents.

#### Execution Notes

Added drift enforcement in `internal/bootstrapsync` tests and a smoke test that
runs `scripts/sync-bootstrap-assets --check` against the current repository so
CI-covered validation fails when tracked dogfood outputs drift from
`assets/bootstrap/`, including orphaned files under `.agents/skills/`.
Validation: `go test ./internal/bootstrapsync` and `go test ./tests/smoke -run 'TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestInstall'`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step adds automated enforcement around the Step 2
sync path and is covered by the same full finalize review.

## Validation Strategy

- Lint the tracked plan before approval and keep it self-contained enough for a
  future agent to execute without discovery chat.
- Validate the implementation with focused unit or smoke coverage around the
  sync path and drift checks, then rerun the broader install-related tests that
  cover `harness install` behavior.
- Confirm the repository docs, dogfood outputs, and packaged bootstrap assets
  agree after the sync path runs with no residual drift.

## Risks

- Risk: Contributors may continue editing `.agents/skills/` directly because the
  repository currently treats those files as ordinary tracked content.
  - Mitigation: Put the canonical-source rule in repo-visible docs and back it
    with automated drift failures.
- Risk: Syncing the root `AGENTS.md` managed block could accidentally overwrite
  easyharness-specific guidance outside the managed markers.
  - Mitigation: Reuse the existing managed-block replacement behavior and add
    explicit validation for preserving surrounding content.
- Risk: A bespoke sync flow could diverge from the packaged assets used by
  `harness install`.
  - Mitigation: Treat `assets/bootstrap/` as the only editable source and keep
    install-path tests in the validation loop.

## Validation Summary

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `go test ./cmd/bootstrap-sync`
- `go test ./internal/bootstrapsync ./internal/install`
- `go test ./tests/smoke -run 'TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestInstall'`

## Review Summary

- `review-001-full`: changes requested after `correctness` found that
  orphaned `.agents/skills` files were invisible to the initial drift check
- `review-002-full`: full finalize review passed with no findings after the
  orphan-detection and orphan-removal repair landed

## Archive Summary

- Archived At: 2026-03-31T13:48:36+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate satisfies the acceptance criteria, the canonical-source
  docs now point edits at `assets/bootstrap/`, the repo-local sync path is
  deterministic, and the final finalize review passed after the orphan-file
  repair.
- Merge Handoff: Commit the candidate on `codex/bootstrap-sync-canonical-source`,
  push the branch, open a PR, then record publish/CI/sync evidence until
  `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Established `assets/bootstrap/` as the single hand-edited source for the
  bootstrap skill pack and the managed `AGENTS.md` block content in this repo.
- Added repo-visible guidance in `AGENTS.md`, `README.md`, and the CLI contract
  that tells humans and agents to edit `assets/bootstrap/` and treat
  `.agents/skills/` as materialized output.
- Added `scripts/sync-bootstrap-assets`, backed by `cmd/bootstrap-sync` and
  `internal/bootstrapsync`, so this repository can refresh dogfood outputs
  without using `harness install` as the human-facing maintenance command.
- Added drift detection and repair for orphaned `.agents/skills` files that no
  longer exist in `assets/bootstrap/`.
- Added focused unit coverage for sync/apply drift paths and a smoke test that
  enforces `scripts/sync-bootstrap-assets --check` against the current repo.

### Not Delivered

- A later split between human-authored bootstrap source and generated
  embed-ready assets remains deferred.
- A dedicated first-class `harness` subcommand for repo-local bootstrap syncing
  remains deferred.

### Follow-Up Issues

- [#82](https://github.com/catu-ai/easyharness/issues/82): Evaluate splitting
  `assets/bootstrap` into authored source and generated embed assets.
- [#83](https://github.com/catu-ai/easyharness/issues/83): Decide whether
  bootstrap sync should become a first-class `harness` subcommand.
