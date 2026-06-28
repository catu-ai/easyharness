---
template_version: 0.2.0
created_at: "2026-06-28T11:29:21+08:00"
approved_at: "2026-06-28T11:30:45+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/260
size: XS
---

# Harden Contract Sync Comment Scope

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Make `scripts/sync-contract-artifacts --check` deterministic when frontend
dependency or build directories exist under the repository. Contract schema
generation should load Go comments only from the Go source locations relevant
to public contract types, so transient paths such as `web/node_modules` cannot
break CI with filesystem races.

This plan addresses issue #260, which captured an intermittent CI failure from
`jsonschema` comment loading traversing a temporary Vite/pnpm path under
`web/node_modules`. The fix should be narrow and aimed at the contract sync
comment-loading boundary, not at pnpm, Vite, or broader validation behavior.

## Scope

### In Scope

- Restrict `internal/contractsync`'s `jsonschema.Reflector.AddGoComments`
  usage so it does not traverse the whole repository root.
- Preserve generated schema descriptions sourced from `internal/contracts`
  comments.
- Add regression coverage that fails when a bad transient path under
  `web/node_modules` is traversed during `scripts/sync-contract-artifacts
  --check`.
- Run focused contract sync validation after the change.

### Out of Scope

- Changing frontend dependency installation, Vite configuration, pnpm build
  approvals, or `web/node_modules` layout.
- Reworking the contract schema registry or changing generated schema meaning
  beyond preserving existing comments.
- Adding compatibility shims for older contract sync behavior.
- Broad `scripts/validate` performance or CI profile changes.

## Acceptance Criteria

- [x] `scripts/sync-contract-artifacts --check` ignores unreadable or missing
      transient paths under `web/node_modules`.
- [x] Generated contract schemas still include existing type and field
      descriptions from `internal/contracts` comments.
- [x] A focused regression test creates a repository copy with a problematic
      frontend dependency path and proves contract sync still passes.
- [x] The contract sync smoke tests and `scripts/sync-contract-artifacts
      --check` pass after the fix.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Scope Reflector Comment Loading

- Done: [x]

#### Objective

Change contract schema generation so reflector Go comment loading is rooted at
the relevant Go source subtree instead of the full workdir.

#### Details

`internal/contractsync.expectedFiles` already calls `loadContractComments`,
which parses only `internal/contracts` with Go's parser. The wider traversal
comes from `reflector.AddGoComments(moduleRootID, workdir,
jsonschema.WithFullComment())`, where `workdir` is the repository root.

Prefer the clean target shape: make `AddGoComments` point at the package or
source root that contains the contract types needed by the schema registry. If
the `jsonschema` API requires a module path plus a filesystem root, keep the
module path stable while narrowing the filesystem root to avoid dependency,
build, and local runtime directories. Do not introduce an ignore list for every
possible generated directory unless the API cannot support a narrower root.

The implementation should preserve the existing `applyComments` behavior and
the generated schema output. If the narrowed reflector comments would remove a
description that is already supplied by `loadContractComments`, keep the
explicit `internal/contracts` parser as the source of truth rather than
expanding traversal again.

#### Expected Files

- `internal/contractsync/sync.go`
- generated schema files only if the stable expected output intentionally
  changes

#### Validation

- `scripts/sync-contract-artifacts --check` passes.
- Any generated schema diff is explained as intentional; the preferred result
  is no generated schema diff.

#### Execution Notes

Removed the redundant `jsonschema.Reflector.AddGoComments` call that walked
the repository root during contract schema generation. Contract descriptions
now continue to come from the existing `loadContractComments` parser scoped to
`internal/contracts`, and `scripts/sync-contract-artifacts --check` passed with
no generated schema diff.

#### Review Notes

Step-closeout delta review `review-001-delta` passed with `correctness` and
`tests` reviewer slots, 0 blocking findings, and 0 non-blocking findings.

### Step 2: Add Transient Dependency Regression Coverage

- Done: [x]

#### Objective

Cover the CI flake by proving contract sync does not inspect a problematic
frontend dependency path.

#### Details

Extend focused contract sync smoke coverage in `tests/smoke/contract_sync_test.go`
or add an equivalent narrowly scoped test. The test should copy the current
repository, create a `web/node_modules` structure that would fail if walked
directly, and then run the copied repository's
`scripts/sync-contract-artifacts --check`.

Model the failure in a portable way. A missing symlink target, unreadable path,
or dangling temporary directory entry is acceptable as long as the test fails
against the old whole-workdir traversal and passes once comment loading is
scoped. Keep the fixture under the copied repo so the real developer checkout
does not gain untracked dependency artifacts.

#### Expected Files

- `tests/smoke/contract_sync_test.go`
- test helper changes only if they make the fixture clearer

#### Validation

- The new regression test fails on the old traversal and passes with the
  scoped comment loading.
- `go test ./tests/smoke -run TestSyncContractArtifacts -count=1` passes.
- `scripts/sync-contract-artifacts --check` passes.

#### Execution Notes

Added a smoke regression that copies the repository, creates an unreadable
`web/node_modules/.pnpm/vite_tmp_missing` directory, and verifies the copied
repository's `scripts/sync-contract-artifacts --check` still passes. The test
failed against the previous whole-workdir traversal and passes after the
comment-loading scope fix.

#### Review Notes

Step-closeout delta review `review-002-delta` passed with the `tests`
reviewer slot, 0 blocking findings, and 0 non-blocking findings.

## Validation Strategy

- Run `scripts/sync-contract-artifacts --check` to verify generated contract
  artifacts are still in sync.
- Run the focused smoke package or test filter covering contract sync:
  `go test ./tests/smoke -run TestSyncContractArtifacts -count=1`.
- If execution changes generated schema output, run `scripts/sync-contract-artifacts`
  first, inspect the generated diff, and then rerun the check command.
- Optionally run `scripts/validate` after the focused fix if local Node/pnpm
  state allows it; this plan does not require solving unrelated frontend
  dependency installation policy issues.

## Risks

- Risk: Narrowing `AddGoComments` too far could silently remove descriptions
  that are currently populated by reflector comments.
  - Mitigation: Compare generated schema output before and after the change and
    keep the explicit `loadContractComments` metadata as the authoritative
    source for `internal/contracts` descriptions.
- Risk: A regression fixture based on permissions or symlinks may behave
  differently across local macOS and Linux CI.
  - Mitigation: Prefer a portable dangling or missing path shape, and validate
    it through the copied repository's script rather than relying on host-only
    filesystem semantics.

## Validation Summary

- `go test ./tests/smoke -run TestSyncContractArtifactsCheckIgnoresTransientFrontendDependencyDirs -count=1`
- `go test ./tests/smoke -run TestSyncContractArtifacts -count=1`
- `scripts/sync-contract-artifacts --check`

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
