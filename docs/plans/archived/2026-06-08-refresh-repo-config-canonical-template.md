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

- [x] `repoconfig.DefaultContent` includes commented default path roots and is
      used by `harness repo init` and `harness repo config init` when they
      create a missing config file.
- [x] `harness repo config init` continues to preserve any existing
      `.harness/config.yaml` without rewriting it.
- [x] `harness repo config refresh` exists, returns the existing repo resource
      JSON result shape, supports create/update/noop action reporting, and does
      not introduce `--dry-run` in this slice.
- [x] Refreshing a valid config preserves custom path-root values while
      rendering the current canonical file shape.
- [x] Refreshing an invalid config does not overwrite the file and reports the
      invalid-config reason.
- [x] Tests and specs no longer describe generated config content as exactly
      `version: 1` only.
- [x] This repository's `.harness/config.yaml` is refreshed with the supported
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

- Done: [x]

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

Added `harness repo config refresh` through the install service and CLI. The
command creates missing config, refreshes old/default-equivalent config,
preserves valid custom paths, noops when already canonical, and rejects invalid
config without overwriting it. Validation passed with
`go test ./internal/install ./internal/cli ./tests/smoke`; smoke took the
expected longer release-build path. Reinstalled the dev harness with
`scripts/install-dev-harness` after changing Go CLI code.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The step has focused install, CLI, and smoke coverage;
the final review inspected the integrated command/docs/dogfood behavior.
Finalize review `review-001-full` found one minor automated-help coverage gap;
added CLI help tests for `repo config --help` and
`repo config refresh --help`.

### Step 3: Update contracts and dogfood config

- Done: [x]

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

Refreshed this repository's `.harness/config.yaml` with
`harness repo config refresh`, then reran the command to verify it reports a
noop once canonical. Updated README and the normative config/bootstrap/CLI
specs to describe canonical default-equivalent config creation and the new
refresh command. `harness repo config --help` lists `refresh`, and
`harness plan lint docs/plans/active/2026-06-08-refresh-repo-config-canonical-template.md`
passed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is docs plus dogfood config alignment; final
review inspected the complete code/docs/test/config candidate.
Finalize review `review-001-full` found that the deferred result-envelope
cleanup had not yet been recorded in Follow-Up Issues. Created #244 and
recorded it in this plan.

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

- `go test ./internal/repoconfig` passed after adding canonical rendering and
  ambiguous YAML scalar round-trip coverage.
- `go test ./internal/install ./internal/cli ./tests/smoke` passed after
  adding `harness repo config refresh`, service/CLI/smoke coverage, and
  canonical config assertions.
- `go test ./internal/repoconfig ./internal/cli ./internal/install` passed
  after finalize review repairs.
- `go test ./...` passed after the review repairs; the smoke package took the
  expected release-build path.
- Manual command checks passed: `harness repo config refresh` updated the
  dogfood config, a second refresh reported `noop`, `harness repo config
  --help` listed `refresh`, and `harness plan lint` passed for this plan.
- Revision 2 reopened for remote-sync repair after `origin/main` advanced via
  PR #242. The branch merged `origin/main` cleanly, `go test ./tests/smoke`
  passed on retry after a transient Corepack network timeout, and a subsequent
  full `go test ./...` passed.
- Revision 3 reopened for remote-sync repair after `origin/main` advanced via
  the 0.4.2 release. The branch merged `origin/main`, conflicts in README,
  CLI contract, CLI config routing, and repo config helpers were resolved by
  keeping both `refresh` and the newer `get`/`list` query commands, and
  `scripts/install-dev-harness` passed after the Go CLI merge.
- `go test ./internal/repoconfig ./internal/cli ./internal/install ./tests/smoke`
  passed after the conflict repair; smoke took the expected release-build path.
- Full `go test ./...` passed after the conflict repair.

## Review Summary

- Step-local reviews were skipped with recorded `NO_STEP_REVIEW_NEEDED`
  markers because focused tests covered the local slices and the integrated
  behavior was reviewed at finalize.
- Finalize review `review-001-full` requested changes:
  - fixed custom path rendering so YAML-ambiguous string values such as
    `"true"` and `"2026"` round-trip as strings after refresh
  - added automated CLI help coverage for `repo config --help` and
    `repo config refresh --help`
  - created and recorded follow-up issue #244 for repo config result-envelope
    cleanup
- Finalize review `review-002-full` passed with no findings after the repair.
- Finalize review `review-003-full` passed with no findings after the revision
  2 remote-sync merge from `origin/main`.
- Finalize review `review-004-full` passed with no findings after the revision
  3 remote-sync merge from `origin/main`.

## Archive Summary

- Archived At: 2026-06-10T00:02:12+08:00
- Revision: 3
- PR: https://github.com/catu-ai/easyharness/pull/245
- Ready: Acceptance criteria are satisfied, the branch has merged
  `origin/main` with conflicts resolved against the 0.4.2 release state, and
  full `go test ./...` passed after the remote-sync repair. Finalize review
  `review-004-full` passed with no findings.
- Merge Handoff: Re-archive the plan, commit the tracked archive move and
  closeout summaries, push branch
  `codex/refresh-repo-config-canonical-template` to PR #245, refresh
  publish/CI/sync evidence, and stop once `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- `repoconfig.DefaultContent` now shows commented default path roots while
  remaining default-equivalent.
- `repoconfig.Render` renders default config as the commented canonical
  template and renders custom path roots through the YAML encoder so valid
  string values are preserved safely.
- `harness repo init` and `harness repo config init` create the canonical
  default-equivalent config when the file is missing and continue preserving
  existing config files.
- `harness repo config refresh` creates missing config, updates old/default or
  custom valid config to the canonical shape, reports noop when already
  canonical, and refuses invalid existing config without overwriting it.
- README, bootstrap/config/CLI specs, tests, and this repository's
  `.harness/config.yaml` now match the canonical refresh behavior.
- Revision 2 also includes the clean merge from `origin/main` after PR #242
  landed.
- Revision 3 includes the conflict-resolved merge from `origin/main` after the
  0.4.2 release landed, preserving both this refresh command and the newer
  repo config query commands.

### Not Delivered

- Repo config command result-envelope cleanup was intentionally deferred to
  issue #244.
- Repo config refresh diff preview and explicit canonical text/comment/order
  tests were intentionally deferred to issue #247.

### Follow-Up Issues

- [#244 Review repo config command result envelope](https://github.com/catu-ai/easyharness/issues/244)
- [#247 Add repo config refresh diff preview and canonical text tests](https://github.com/catu-ai/easyharness/issues/247)
