---
template_version: 0.2.0
created_at: "2026-06-28T23:27:48+08:00"
approved_at: "2026-06-28T23:28:57+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/247
size: S
---

# Add repo config refresh diff preview

## Goal

Add a focused preview mode for `harness repo config refresh` so users can
inspect the canonical rewrite before applying it. The preview command should
show the same file change ordinary refresh would make, but it must not write
`.harness/config.yaml`.

Keep the command shape aligned with the existing repo config surface:
ordinary `harness repo config refresh` remains the JSON-emitting mutation
command, while `harness repo config refresh --diff` is a plain-text preview
mode that prints a unified diff to stdout and never embeds diff text inside the
bootstrap JSON result envelope.

## Scope

### In Scope

- Add `harness repo config refresh --diff`.
- Make `--diff` compute the canonical refresh result for missing, stale, or
  non-canonical valid config files without applying the planned write.
- Print a readable unified diff to stdout when refresh would change
  `.harness/config.yaml`.
- Print empty stdout and exit successfully when the file already matches the
  current canonical render.
- Reject invalid existing config through the same validation rules ordinary
  refresh uses, without overwriting the file.
- Preserve ordinary `harness repo config refresh` JSON output and write
  behavior.
- Document the canonical text contract for comments, field ordering, and
  configured value preservation.
- Add focused CLI, service or helper, and smoke coverage for the diff mode and
  canonical text behavior.

### Out of Scope

- Adding `harness repo config refresh --dry-run`.
- Adding a new `harness repo config diff` subcommand.
- Preserving user-authored YAML comments or original field ordering during
  refresh.
- Introducing YAML AST round-trip editing.
- Changing repo config schema, path-root semantics, or unsupported-field
  validation.
- Changing the bootstrap JSON result schema to carry diff content.

## Acceptance Criteria

- [x] `harness repo config refresh --diff` leaves `.harness/config.yaml`
      untouched for missing, stale, non-canonical valid, already canonical, and
      invalid existing config cases.
- [x] `--diff` prints a unified diff to stdout when refresh would create or
      update `.harness/config.yaml`.
- [x] `--diff` prints empty stdout and exits successfully when the current file
      is already canonical.
- [x] `--diff` rejects invalid existing config with the same failure semantics
      as ordinary refresh and does not print a misleading diff.
- [x] Ordinary `harness repo config refresh` still rewrites valid config to the
      canonical render, preserves configured values, emits the existing JSON
      result envelope, and does not include diff text in JSON.
- [x] Tests cover a valid config with user-authored comments and non-canonical
      field order so the expected diff demonstrates comment removal/order
      normalization while preserving configured values.
- [x] Docs/spec/help text explain that refresh owns canonical file shape:
      valid comments are accepted on load but not preserved by refresh, field
      ordering is normalized by the renderer, configured values are preserved,
      and unsupported fields remain invalid.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Define the refresh diff contract

- Done: [x]

#### Objective

Update the repo config command contract and user-facing help to describe
`harness repo config refresh --diff` and the canonical text behavior.

#### Details

The contract should present `--diff` as a preview mode for the existing
refresh operation, not as a generic dry-run facility and not as a new bootstrap
JSON envelope variant. Keep `repo config get/list` as existing plain-text
exceptions, and document this new plain-text preview mode narrowly so future
agents do not route diff text into `contracts.BootstrapResult`.

The canonical text documentation should say:

- refresh owns the canonical rendered file shape
- valid YAML comments are accepted by load when the values remain valid
- refresh does not preserve user-authored comments
- refresh normalizes field ordering through the canonical renderer
- configured values are preserved
- unsupported fields remain invalid

#### Expected Files

- `docs/specs/bootstrap-install.md`
- `docs/specs/repo-config.md`
- `docs/specs/cli-contract.md`
- `assets/help/repo/config.md`
- `internal/cli/app.go`
- `internal/cli/app_test.go`

#### Validation

- `go test ./internal/cli -run 'TestRepoConfigRefreshHelp|TestHelpRepoConfig'`
- Confirm help text mentions `--diff` without implying `--dry-run` support.
- Confirm the written contract makes ordinary refresh JSON output and diff
  preview stdout mutually clear.

#### Execution Notes

Added CLI help coverage for `harness repo config refresh --diff`, updated
refresh command help to advertise `--diff` without `--dry-run`, and documented
the plain-text preview/canonical text contract in bootstrap, CLI, repo config,
and help-topic docs. Validation: `go test ./internal/cli -run
'TestRepoConfigRefreshHelp|TestHelpRepoConfig'`; reran
`scripts/install-dev-harness` after changing Go CLI code.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 only establishes the documented command
contract and help text; Step 2 will receive implementation-focused review
after the behavior and tests exist.

### Step 2: Implement preview output and regression coverage

- Done: [x]

#### Objective

Implement `harness repo config refresh --diff` using the same canonical render
and validation rules as ordinary refresh, while leaving ordinary refresh
behavior unchanged.

#### Details

The implementation should keep the existing `install.Service.RefreshConfig`
JSON path stable for ordinary refresh. A focused helper may expose the planned
repo config refresh content to the CLI or service layer, but the public
bootstrap result contract should not gain a diff field.

Unified diff output should compare the current target path with the canonical
refresh content. For a missing config, the diff should make clear that
`.harness/config.yaml` would be created with canonical content. For an already
canonical config, stdout should be empty. For invalid existing config, return
the ordinary refresh-style failure and leave the file untouched.

#### Expected Files

- `internal/cli/app.go`
- `internal/install/service.go`
- `internal/install/service_test.go`
- `internal/cli/app_test.go`
- `tests/smoke/smoke_test.go`
- `internal/repoconfig/config_test.go`

#### Validation

- `go test ./internal/repoconfig`
- `go test ./internal/install`
- `go test ./internal/cli -run 'TestRepoConfigRefresh'`
- `go test ./tests/smoke -run TestRepoConfigRefresh`
- `git diff --check`

#### Execution Notes

Added `install.Service.PlanConfigRefreshDiff` plus internal unified-file diff
rendering for repo config refresh previews. Wired `harness repo config refresh
--diff` to print plain diff text without writing files, while ordinary refresh
continues to emit the existing bootstrap JSON result and apply changes. Added
install, CLI, and smoke tests for missing config creation preview,
commented/out-of-order valid config normalization, canonical no-op empty
stdout, invalid config failure/no-write behavior, and ordinary refresh JSON
preservation. Validation: `go test ./internal/repoconfig`; `go test
./internal/install`; `go test ./internal/cli -run 'TestRepoConfigRefresh'`;
`go test ./tests/smoke -run TestRepoConfigRefresh`; `git diff --check`;
`scripts/install-dev-harness`; manual repo-local binary preview confirmed diff
output and no-write behavior.

#### Review Notes

Step-closeout review `review-001-delta` passed with 0 findings. Reviewer
slots: `correctness` and `tests`; both submitted clean results and the
aggregate decision was `pass`.

## Validation Strategy

- Use focused package tests for canonical rendering, refresh planning, CLI
  stdout/stderr/exit behavior, and write/no-write guarantees.
- Use smoke coverage to verify the built CLI behavior matches the direct CLI
  tests for `refresh --diff`.
- Use `git diff --check` for whitespace and patch hygiene.
- After changing Go CLI code, rerun `scripts/install-dev-harness` before
  relying on the direct `harness` binary for any post-change harness command.

## Risks

- Risk: The diff mode could accidentally change the bootstrap JSON contract by
  adding diff text to `BootstrapResult` or routing preview output through the
  JSON writer.
  - Mitigation: Keep `--diff` as an explicit CLI output branch and test that
    ordinary refresh still emits JSON without diff content.
- Risk: The diff preview could drift from the actual refresh write behavior.
  - Mitigation: Compute preview content from the same canonical render and
    validation path used by refresh, then test preview-vs-refresh equivalence
    on representative configs.
- Risk: Missing-file diff output may be hard to read or inconsistent with
  users' `git diff` expectations.
  - Mitigation: Use conventional unified diff headers and add a smoke test for
    creation preview output.

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
