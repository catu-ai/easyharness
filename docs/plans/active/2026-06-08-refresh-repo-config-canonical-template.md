---
template_version: 0.2.0
created_at: "2026-06-08T23:01:54+08:00"
approved_at: "2026-06-08T23:04:27+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/240
size: S
---

# Refresh repo config canonical template

## Goal

Make `.harness/config.yaml` a useful repo-visible customization entrypoint
after path roots became configurable in PR #239. Newly created config files
should show the supported path fields through commented defaults, and existing
valid config files should have a command-supported way to refresh to the
current canonical file shape while preserving user customizations.

The slice should keep `harness repo config init` as the safe missing-file
creator and add a focused refresh command for making the config file
up-to-date. It should not broaden into a general redesign of repo resource JSON
result envelopes.

## Scope

### In Scope

- Update the canonical default repo config template so it shows commented
  default path roots while remaining default-equivalent.
- Keep `harness repo config init` no-op behavior for existing config files,
  but make missing-file creation use the new canonical default template.
- Add `harness repo config refresh` as the command that creates a missing
  config or rewrites a valid existing config into the current canonical shape
  while preserving configured path values.
- Preserve existing valid custom path roots when refreshing; reject or warn
  without overwriting invalid config because user intent cannot be recovered
  safely.
- Update command help, specs, tests, and the dogfood `.harness/config.yaml`
  through the supported command path.

### Out of Scope

- Broad repo resource result-envelope redesign.
- Adding `harness repo config lint`.
- Adding refresh preview or diff output.
- Changing path-root config semantics beyond canonical rendering.

## Acceptance Criteria

- [ ] `repoconfig.DefaultContent` includes commented default path roots and is
      used by `harness repo init` and `harness repo config init` when they
      create a missing config file.
- [ ] `harness repo config init` continues to preserve any existing
      `.harness/config.yaml` without rewriting it.
- [ ] `harness repo config refresh` exists, returns the existing repo resource
      JSON result shape, supports create/update/noop action reporting, and does
      not introduce `--dry-run` in this slice.
- [ ] Refreshing a valid config preserves custom path-root values while
      rendering the current canonical file shape.
- [ ] Refreshing an invalid config does not overwrite the file and reports the
      invalid-config reason.
- [ ] Tests and specs no longer describe generated config content as exactly
      `version: 1` only.
- [ ] This repository's `.harness/config.yaml` is refreshed with the supported
      command path.

## Deferred Items

- Repo config command result-envelope cleanup remains deferred. Before archive,
  create or update a follow-up issue for deciding whether
  `harness repo config init|refresh|lint` should eventually move from the
  bootstrap result envelope into a config-specific command result shape.

## Work Breakdown

### Step 1: Define canonical config rendering

- Done: [x]

#### Objective

Teach the repo config package to render default and custom configs into the
current canonical file shape.

#### Details

Use the existing default path constants as the single source of truth for the
commented defaults. The default-equivalent rendered file should keep `paths`
commented so the semantics remain identical to omitted defaults. A config with
custom values should render an explicit `paths:` block with those values
preserved.

#### Expected Files

- `internal/repoconfig/config.go`
- `internal/repoconfig/config_test.go`

#### Validation

- Focused tests cover default canonical content, custom path rendering, and
  default-equivalent behavior.

#### Execution Notes

Added canonical repo config rendering in `internal/repoconfig`: default
content now shows commented path defaults, custom configs render an explicit
`paths:` block, and invalid loads expose the invalid reason for refresh
errors. Focused `go test ./internal/repoconfig` passed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only establishes local rendering helpers and
focused tests; the behavior-facing command changes and final candidate review
will cover the combined config-refresh surface.

### Step 2: Add the refresh command

- Done: [ ]

#### Objective

Expose `harness repo config refresh` as the supported way to make
`.harness/config.yaml` match the current canonical shape.

#### Details

Keep `init` as the missing-file creator that no-ops when the file already
exists. Add refresh behavior through the install service and CLI routing:
missing config creates canonical defaults, valid existing config rewrites only
when canonical content differs, valid custom values are preserved, and invalid
config is not overwritten. Reuse the existing repo resource JSON result shape
for this slice even though config-specific result-envelope cleanup is deferred.

#### Expected Files

- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/install/service.go`
- `internal/install/service_test.go`
- `tests/smoke/smoke_test.go`

#### Validation

- CLI and install-service tests cover create, update, noop, custom-value
  preservation, invalid-config rejection or warning behavior, and command help.
- Smoke coverage exercises `harness repo config refresh`.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Update contracts and dogfood config

- Done: [ ]

#### Objective

Align repository-facing docs and this repository's checked-in config with the
new canonical config behavior.

#### Details

Update specs and command-contract text so they distinguish `init` from
`refresh`, remove the stale "minimal `version: 1` only" expectation, and keep
the deferred result-envelope cleanup explicit. Refresh the dogfood
`.harness/config.yaml` through the newly supported command rather than manual
editing.

#### Expected Files

- `.harness/config.yaml`
- `docs/specs/bootstrap-install.md`
- `docs/specs/cli-contract.md`
- `docs/specs/repo-config.md`

#### Validation

- `harness repo config refresh` updates the dogfood config as expected.
- Relevant smoke, install, CLI, and repoconfig tests pass.
- `harness plan lint docs/plans/active/2026-06-08-refresh-repo-config-canonical-template.md`
  passes after any plan updates.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run focused Go tests for repo config rendering and install/CLI behavior.
- Run smoke coverage for repo init/config init/config refresh behavior.
- Run broader `go test ./...` if the focused suite does not already cover all
  touched shared command surfaces.
- Run `harness plan lint` before execution starts and again before archive if
  the tracked plan changes.

## Risks

- Risk: Refresh could accidentally erase meaningful user customization.
  - Mitigation: Parse valid config first, render from the parsed config, and
    add regression coverage for custom path values.
- Risk: Invalid config handling could silently replace a file whose intent is
  unclear.
  - Mitigation: Treat invalid existing config as non-refreshable and leave it
    untouched.
- Risk: The command result-envelope concern could expand the issue beyond the
  config template problem.
  - Mitigation: Keep existing repo resource result shape for this slice and
    track result-envelope cleanup as a separate follow-up issue.

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
