---
template_version: 0.2.0
created_at: "2026-06-06T23:34:52+08:00"
approved_at: "2026-06-06T23:38:49+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/228
size: M
---

# Repo Resource Config Manifest

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Define the first repo-level customization manifest contract for easyharness
and make the repository resource commands read as one coherent command group.

After this slice, `.harness/config.yaml` is a tracked, visible manifest
entrypoint with a versioned v1 shape, `harness repo init` creates the minimal
config baseline alongside managed instructions and skills, and the former
top-level bootstrap resource commands move under `harness repo ...`.

## Scope

### In Scope

- Define `.harness/` as the tracked repo-level harness customization root and
  `.harness/config.yaml` as the manifest/index entrypoint.
- Define the v1 config contract with the minimal valid file:

  ```yaml
  version: 1
  ```

- Implement repo config loading and validation with this behavior:
  missing config is valid and uses built-in defaults; valid config is parsed
  and available to consumers; invalid config produces an agent-facing warning,
  is ignored as a whole, and falls back to built-in defaults.
- Introduce the `harness repo ...` command group as the clean target command
  shape:
  `harness repo init`, `harness repo skills install`,
  `harness repo skills uninstall`, `harness repo instructions install`,
  `harness repo instructions uninstall`, and `harness repo config init`.
- Update `harness repo init` to install or refresh managed instructions and
  skills and create `.harness/config.yaml` when it is missing, without
  overwriting an existing config.
- Add `harness repo config init` as the focused command for creating the
  minimal config file without installing other repo resources.
- Remove the old top-level `harness init`, `harness skills ...`, and
  `harness instructions ...` command shapes rather than preserving
  compatibility shims.
- Update CLI help, specs, docs, schemas or generated references as needed so a
  future agent can discover the repo resource contract without relying on this
  plan.
- Add focused automated coverage for the command routing, config creation,
  no-overwrite behavior, invalid-config warning/fallback behavior, and updated
  docs/schema expectations.

### Out of Scope

- Defining real business customization fields such as review defaults, remote
  provider mappings, plan paths, bootstrap target customization, or
  instruction-content customization.
- Wiring all workflow commands to read repo config. Only commands in the repo
  resource/config surface need to exercise the v1 loader in this slice.
- `harness repo config lint`.
- A broader `doctor` command.
- A repo config guide/skill for agents. Defer this until the first real
  customization field exists.
- Migration helpers or compatibility aliases for the old top-level
  `init`, `skills`, and `instructions` commands.

## Acceptance Criteria

- [x] `harness repo init` installs or refreshes managed skills and
      instructions, creates `.harness/config.yaml` with `version: 1` when it is
      missing, and never overwrites an existing config file.
- [x] `harness repo config init` creates the minimal config file, supports
      dry-run behavior, and reports clearly when the config already exists.
- [x] Old top-level repo-resource commands are removed from the public command
      tree and root usage points users to `harness repo ...`.
- [x] The v1 config contract is documented in a tracked spec/reference
      location, including versioning, missing/valid/invalid behavior,
      whole-config fallback on invalid config, and the rule that long-form
      customization text belongs in referenced `.harness/**/*.md` files rather
      than inline YAML.
- [x] Invalid `.harness/config.yaml` content, unsupported versions, and
      non-object YAML shapes are surfaced as warnings and ignored by the repo
      config loader instead of blocking repo-resource commands.
- [x] Tests cover config init, repo init, no-overwrite behavior, invalid
      config warnings/fallback, command help/routing, and any generated
      contract/docs updates.

## Deferred Items

- Add `harness repo config lint` once the config surface has enough behavior
  to justify an explicit validation command.
- Add a broader `doctor` command only after the project decides what health
  surfaces belong in a general diagnostic.
- Add an agent-facing repo config guide/skill after the first real
  customization field lands.
- Define and consume concrete business customization fields in follow-up
  issues, such as instruction path/content customization, review defaults, or
  remote mappings.

## Work Breakdown

### Step 1: Specify the repo config contract

- Done: [x]

#### Objective

Write the normative v1 `.harness/config.yaml` contract and update public
references for the new `harness repo ...` command group.

#### Details

The spec should describe `.harness/` as tracked repo customization space,
`.harness/config.yaml` as a manifest/index, and `version: 1` as the first
required version marker. It should explicitly say that missing config is okay,
invalid config warns and falls back to defaults, and invalid config is ignored
as a whole rather than partially consumed.

The spec should avoid inventing real customization fields in this slice. It may
describe the future extension rule that long-form customization text should be
referenced from `.harness/**/*.md` files instead of stored inline in YAML.

#### Expected Files

- `docs/specs/`
- `docs/specs/cli-contract.md`
- `docs/specs/index.md`
- `README.md`
- `docs/development.md`

#### Validation

- Documentation names the new command group and no longer presents the old
  top-level commands as the target shape.
- A cold reader can explain the v1 config behavior without reading chat
  history.

#### Execution Notes

Added `docs/specs/repo-config.md` for the v1 `.harness/config.yaml` manifest
contract and updated the spec index, CLI contract, bootstrap resource spec,
README, and development docs to present `harness repo ...` as the target repo
resource surface.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step was implemented as part of one integrated
CLI/docs/schema slice and will receive a full finalize review before archive.

### Step 2: Add config loader and init behavior

- Done: [x]

#### Objective

Implement the minimal v1 config loader plus creation behavior for
`.harness/config.yaml`.

#### Details

Add a small internal package or service for repo config loading and validation.
The public behavior is intentionally narrow: no file means defaults, a valid
file returns parsed config, and an invalid file returns warnings plus defaults.

Config init should create the `.harness/` directory and write a minimal
`config.yaml` containing `version: 1`. Existing config must be preserved.
Dry-run should show the planned creation without writing files.

#### Expected Files

- `internal/`
- `internal/cli/`
- `tests/`

#### Validation

- Unit tests cover missing, valid, malformed YAML, unsupported version,
  non-object YAML, config creation, dry-run, and existing-file preservation.
- Loader tests verify invalid config produces warnings and falls back to
  defaults without returning partially parsed config.

#### Execution Notes

Added `internal/repoconfig` with v1 loading and whole-config fallback on
invalid config. Updated repo resource installation so `harness repo init` and
`harness repo config init` create the minimal config when missing, preserve
existing config files, and surface invalid config as warnings.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Covered by focused loader/install/CLI tests and the
planned full finalize review for the integrated branch.

### Step 3: Move repo resource commands under `harness repo`

- Done: [x]

#### Objective

Replace top-level repo resource commands with the nested `harness repo ...`
command group.

#### Details

The target command tree is:

```text
harness repo init
harness repo skills install
harness repo skills uninstall
harness repo instructions install
harness repo instructions uninstall
harness repo config init
```

Keep command depth at this level. Do not introduce deeper config subcommands
such as `harness repo config schema validate` in this slice.

Because the repository is still in fast development, remove the old top-level
`harness init`, `harness skills ...`, and `harness instructions ...` shapes
instead of adding compatibility aliases. Update tests and docs to the clean
target shape.

#### Expected Files

- `internal/cli/`
- `internal/install/`
- `assets/bootstrap/`
- `docs/`
- `tests/`

#### Validation

- CLI routing tests cover the new command tree and the removed old command
  shapes.
- Help output shows the `repo` group clearly and keeps the root command list
  legible.

#### Execution Notes

Moved public repo resource routing under `harness repo ...`, removed top-level
`init`, `skills`, and `instructions` command routing, updated root/subcommand
help, status idle guidance, command-result identifiers, and black-box smoke
coverage.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Command regrouping is validated by CLI and smoke tests
and will receive a full finalize review before archive.

### Step 4: Refresh generated assets and end-to-end evidence

- Done: [x]

#### Objective

Bring generated bootstrap/contract artifacts and smoke coverage in line with
the new repo resource command contract.

#### Details

If bootstrap-managed instructions or skill text changes, edit
`assets/bootstrap/` first and run `scripts/sync-bootstrap-assets` to refresh
materialized outputs. If schemas or contract references change, run the
repository's generation/sync command for those artifacts.

Before archive, create or update GitHub follow-up issues for durable deferred
items that do not already have suitable tracking issues, then record them in
the plan outcome.

#### Expected Files

- `assets/bootstrap/`
- `.agents/skills/`
- `AGENTS.md`
- `schema/`
- `docs/specs/`
- `tests/smoke/`

#### Validation

- `scripts/sync-bootstrap-assets` if bootstrap assets changed.
- Contract/schema generation or sync command if schema references changed.
- Relevant Go unit tests.
- Relevant smoke tests for repo resource initialization.
- `harness status` before marking the final step done.

#### Execution Notes

Regenerated contract schemas after adding bootstrap-result warnings and repo
command descriptions, ran `scripts/sync-bootstrap-assets` to materialize
`.harness/config.yaml` for this dogfood repository, reinstalled the dev
harness, and validated the integrated change.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Generated artifacts and smoke coverage were validated
directly; full candidate review remains required before archive.

## Validation Strategy

- Run focused unit tests for the new config loader, config init behavior, and
  CLI command routing.
- Run smoke coverage that exercises `harness repo init` in a disposable
  workspace and verifies `.harness/config.yaml` creation and no-overwrite
  behavior.
- Run generated artifact sync/check commands for any touched bootstrap or
  contract artifacts.
- Run `harness status` at routine checkpoints and before archive.

## Risks

- Risk: The command regrouping touches user-facing CLI entrypoints and can
  leave docs, tests, or managed bootstrap text stale.
  - Mitigation: Update command routing, help, docs, and smoke tests together;
    prefer the clean target design over compatibility shims.
- Risk: The v1 config contract could accidentally become a broad customization
  system.
  - Mitigation: Keep the only required field as `version: 1`, avoid business
    fields, and record concrete customization surfaces as follow-up issues.
- Risk: Invalid config handling could be inconsistent across commands.
  - Mitigation: Centralize loading/validation and make invalid config degrade
    to warnings plus built-in defaults as a whole.

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
