---
template_version: 0.2.0
created_at: "2026-06-08T22:40:18+08:00"
approved_at: "2026-06-08T22:43:17+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/241
size: S
---

# Add Repo Config Query Commands

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add a small read-only CLI surface for agents and scripts to query resolved
repo config values without inferring path roots from docs or built-in defaults.

The command shape should follow the useful part of `git config`: exact scalar
lookups belong to `get`, while prefix enumeration belongs to `list`. This keeps
the behavior stable as `.harness/config.yaml` grows deeper over time.

## Scope

### In Scope

- Add `harness repo config get <key>` for resolved scalar config values.
- Add `harness repo config list [prefix]` for resolved leaf-key enumeration.
- Support the currently defined path keys:
  - `paths.plans.active`
  - `paths.plans.archived`
  - `paths.local_runtime`
- Preserve existing repo config loading behavior for missing config, partial
  config, and invalid-config whole-config fallback.
- Update CLI help, specs, docs, and tests for the new command behavior.
- Keep stdout script-friendly and deterministic.

### Out of Scope

- JSON output or a `--json` flag.
- Mutating config values.
- Adding new `.harness/config.yaml` fields.
- Repo config lint behavior beyond the existing load/fallback model.
- Alternate root-level command aliases such as `harness config get`.

## Acceptance Criteria

- [ ] `harness repo config get paths.local_runtime` prints only the resolved
      value plus a trailing newline.
- [ ] `harness repo config get paths.plans.active` and
      `harness repo config get paths.plans.archived` print their resolved
      values plus trailing newlines.
- [ ] Missing `.harness/config.yaml` returns built-in defaults.
- [ ] Partially specified path config returns explicit values for configured
      keys and built-in defaults for omitted keys.
- [ ] Invalid repo config follows the existing whole-config fallback model and
      returns default resolved values while preserving agent-facing warnings.
- [ ] Unknown keys fail clearly with a non-zero exit code.
- [ ] `get` rejects non-leaf config objects such as `paths` with a clear
      message that points users to `harness repo config list paths`.
- [ ] `harness repo config list` prints all supported resolved leaf entries as
      `key=value` lines in deterministic order.
- [ ] `harness repo config list paths` prints resolved leaf entries under that
      prefix in deterministic order.
- [ ] Docs and help describe `get` as exact scalar lookup and `list` as prefix
      enumeration.

## Deferred Items

- JSON output for config queries remains deferred until an agent or script
  workflow needs structured source metadata.
- Query commands for future config fields should be added when those fields
  become part of the repo config contract.

## Work Breakdown

### Step 1: Define resolved config query behavior

- Done: [x]

#### Objective

Document the command semantics for scalar lookup, prefix listing, invalid
config fallback, and non-leaf rejection.

#### Details

Use the Git-inspired split discussed during discovery:

- `get <key>` is an exact scalar lookup and should never return an object blob.
- `list [prefix]` enumerates resolved leaf keys and values.
- non-leaf `get` requests fail with a suggestion to use `list <prefix>`.
- output is plain text only in this slice.

The command remains under `harness repo config` because the values are
repo-level config, not global/user harness config.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/repo-config.md`

#### Validation

- Specs name the supported command shape and the out-of-scope JSON behavior is
  not implied.

#### Execution Notes

Defined the query-command contract in `docs/specs/cli-contract.md` and
`docs/specs/repo-config.md`: `get` is exact scalar lookup, `list` is
deterministic prefix enumeration, non-leaf `get` fails clearly, and this slice
does not define JSON output.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 only records the approved CLI semantics in
normative docs; behavior is covered by later implementation and validation
steps.

### Step 2: Implement query commands

- Done: [ ]

#### Objective

Wire `harness repo config get` and `harness repo config list` into the CLI
using the existing resolved `repoconfig.Load` behavior.

#### Details

Keep implementation small and data-driven enough that future config fields can
add supported leaves without duplicating switch-heavy command code everywhere.

Default stdout should be:

```text
.local/harness
```

for `get paths.local_runtime`, and:

```text
paths.plans.active=docs/plans/active
paths.plans.archived=docs/plans/archived
paths.local_runtime=.local/harness
```

for `list` under the default config.

Warnings from invalid config should remain agent-facing diagnostics rather than
being silently lost.

#### Expected Files

- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/repoconfig/config.go`
- `internal/repoconfig/config_test.go`

#### Validation

- Unit tests cover successful `get`, missing config defaults, partial config
  defaults, invalid config fallback, unknown keys, non-leaf rejection, full
  list, and prefixed list.

#### Execution Notes

Added resolved query helpers to `internal/repoconfig` and wired
`harness repo config get|list` through the CLI. Followed Red/Green/Refactor:
new unit tests first failed for missing helpers/subcommands, then passed after
implementation. Focused validation passed with `go test ./internal/repoconfig
./internal/cli`, and manual probes after `scripts/install-dev-harness` showed
plain-text `get`, `list`, prefixed `list`, and non-leaf `get` behavior.

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Prove script-facing behavior

- Done: [ ]

#### Objective

Add or update smoke/end-to-end coverage that exercises the command through the
real harness binary shape.

#### Details

Prefer focused smoke coverage unless existing e2e helpers make custom
workspaces simpler. The test should verify plain-text stdout rather than JSON.

#### Expected Files

- `tests/smoke/smoke_test.go`
- `tests/e2e/custom_path_roots_test.go`

#### Validation

- Relevant Go test packages pass locally.
- Direct manual probes for `get` and `list` show the expected plain-text
  output.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run focused unit tests for `internal/repoconfig` and `internal/cli`.
- Run the smoke or e2e tests that cover the new query commands.
- Reinstall the dev harness if Go CLI code changes before relying on direct
  `harness` probes.
- Manually verify the final command outputs for default and custom config.

## Risks

- Risk: `get` behavior could become ambiguous if it returns object-like
  content for non-leaf keys.
  - Mitigation: Make non-leaf `get` a clear error and reserve enumeration for
    `list`.
- Risk: warnings from invalid config could corrupt script-friendly stdout.
  - Mitigation: Keep resolved values on stdout and diagnostics on stderr.
- Risk: future config fields could require a different query shape.
  - Mitigation: Keep the initial registry of queryable leaves simple and make
    `list` prefix-based so deeper future fields fit the same model.

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
