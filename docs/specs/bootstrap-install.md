# Bootstrap Install

## Purpose

Define the normative resource model for bootstrap installation and refresh.
This spec owns:

- `harness repo init`
- `harness repo skills install|uninstall`
- `harness repo instructions install|uninstall`
- `harness repo config init`
- `harness repo config refresh`
- scope semantics for repository and user targets
- managed ownership and version markers for bootstrap skills and instructions
- repo config manifest creation and no-overwrite behavior
- current agent-profile support boundaries

The CLI contract should reference this spec for bootstrap semantics instead of
restating the same rules in multiple places.

## External References

- Agent Skills format specification:
  `https://agentskills.io/specification.md`
- Codex skills guide and optional `agents/openai.yaml` metadata:
  `https://developers.openai.com/api/docs/guides/tools-skills.md`

easyharness-managed skills must remain valid Agent Skills packages. Codex-only
metadata remains optional and should be treated as an extension on top of the
baseline Agent Skills format rather than a replacement for it.

## Resource Model

Repo resources split into three managed surfaces:

- `instructions`
  - a target instruction file such as `AGENTS.md`
  - a single easyharness-managed block inside that file, or the whole file when
    the file contains only bootstrap content
- `skills`
  - a target skill directory containing zero or more easyharness-managed skill
    packages
- `config`
  - the tracked repo customization manifest at `.harness/config.yaml`
  - a canonical v1 file that is default-equivalent when it contains only
    commented default path roots

`harness repo init` is the quick-start repo resource entrypoint. It installs
or refreshes instructions and skills, and creates `.harness/config.yaml` when
it is missing, in one idempotent action.

## Commands

### `harness repo init`

Purpose:

- install or refresh the default repo instructions and skills for the current
  repository
- create the canonical `.harness/config.yaml` v1 manifest when it is missing

Contract:

- default to the `codex` agent profile
- be safe to rerun idempotently
- refresh managed bootstrap assets after an easyharness version upgrade
- use the same underlying install logic as the resource commands
- never overwrite an existing `.harness/config.yaml`
- warn and continue with built-in defaults when an existing repo config is
  invalid
- support `--dry-run`
- support `--agent`, `--skills-dir`, and `--instructions-file` overrides for
  non-default layouts
- allow other commands such as `harness status` to reuse the same managed-asset
  comparison logic for non-blocking reminders about stale default repo
  bootstrap assets; inspection is read-only and applies during idle, active,
  archived, and land states without changing the lifecycle node

### `harness repo skills install|uninstall`

Purpose:

- manage easyharness-managed skill packages independently from instructions

Contract:

- support `--scope <repo|user>`, defaulting to `repo`
- support `--agent <name>`
- support `--dir <path>` as an explicit target override
- support `--dry-run`
- install only valid skill packages
- uninstall only easyharness-managed skill packages
- never silently overwrite unrelated user-owned skills

### `harness repo instructions install|uninstall`

Purpose:

- manage the easyharness bootstrap instructions independently from skills

Contract:

- support `--scope <repo|user>`, defaulting to `repo`
- support `--agent <name>`
- support `--file <path>` as an explicit target override
- support `--dir <path>` when rendering a managed block that needs to mention
  the paired skills directory
- support `--dry-run`
- update or remove only the easyharness-managed block unless the whole target
  file is bootstrap-owned
- never silently overwrite unrelated user-owned instruction content

### `harness repo config init`

Purpose:

- create `.harness/config.yaml` without installing or refreshing other repo
  resources

Contract:

- create `.harness/config.yaml` with the current canonical default-equivalent
  content when it is missing
- support `--dry-run`
- never overwrite an existing config file
- warn and continue with built-in defaults when an existing repo config is
  invalid
- leave explicit validation commands such as `harness repo config lint` to
  future work

### `harness repo config refresh`

Purpose:

- create or refresh `.harness/config.yaml` without installing or refreshing
  other repo resources

Contract:

- create `.harness/config.yaml` with the current canonical default-equivalent
  content when it is missing
- refresh an existing valid config into the current canonical file shape
- preserve valid custom path-root values while refreshing canonical comments,
  ordering, and default-equivalent fields
- report `noop` when the existing config already matches the current canonical
  file shape
- fail without overwriting when the existing config is invalid
- return the existing repo resource JSON result shape
- support `--diff` as a plain-text preview mode that prints the unified diff
  refresh would apply without writing `.harness/config.yaml`
- print empty stdout and exit successfully for `--diff` when the existing
  config already matches the current canonical file shape
- do not embed `--diff` output in the repo resource JSON result shape
- do not support `--dry-run`
- leave explicit validation commands such as `harness repo config lint` to
  future work

## Scopes

- `repo`
  - target the current repository worktree
- `user`
  - target a user-level location outside the current repository

Scope names are part of the public contract. Do not reintroduce the older
bootstrap-specific `agents|skills|all` scope model.

## Agent Support Matrix

Current support is intentionally narrow:

- `codex`
  - repo instructions default: `AGENTS.md`
  - repo skills default: `.agents/skills`
  - user instructions default: `${CODEX_HOME:-$HOME/.codex}/AGENTS.md`
  - user skills default: `${CODEX_HOME:-$HOME/.codex}/skills`
- any other `--agent` value
  - no built-in defaults yet
  - explicit `--file` and/or `--dir` overrides are required

This keeps the bootstrap surface extensible without pretending that easyharness
already ships first-class profiles for every coding agent.

## Managed Ownership

### Instructions

easyharness owns one stable managed block delimited by explicit markers:

- begin marker:
  `<!-- easyharness:begin version="<version>" -->`
- end marker:
  `<!-- easyharness:end -->`

Contract:

- insert the managed block when the target file exists without it
- replace exactly one valid managed block on rerun
- fail when the marker layout is duplicated or otherwise ambiguous
- preserve user-owned content outside the managed block
- remove the whole file on uninstall only when the file contains no meaningful
  user-owned content outside the managed block

### Skills

easyharness-managed skills must remain valid Agent Skills packages and must
carry in-band ownership/version metadata in `SKILL.md` frontmatter:

```yaml
metadata:
  easyharness-managed: "true"
  easyharness-version: "<version>"
```

Contract:

- install or refresh only easyharness-managed skill packages
- treat the whole managed skill directory as owned by easyharness once the
  ownership marker is present
- remove stale easyharness-managed skill packages that are no longer in the
  packaged bootstrap set
- leave unrelated user-owned skills untouched
- fail on same-path collisions when the existing skill is not recognized as
  easyharness-managed

## Version Markers

Managed bootstrap assets must carry an easyharness version marker:

- instructions: begin marker version attribute
- skills: `metadata.easyharness-version`

When the running binary has a release version, use that exact version string.
When the running binary is a development build without a release version, use a
stable development marker instead of a commit-specific value so dogfood outputs
do not churn on every commit.

## Packaging and Source of Truth

- packaged bootstrap assets are shipped with the easyharness binary so bootstrap
  commands work without network access
- in this repository, `assets/bootstrap/` is the canonical hand-edited source
- this repository's `.agents/skills/` tree and managed `AGENTS.md` block are
  dogfood materialized outputs derived from `assets/bootstrap/`

## Output Contract

Bootstrap resource commands are JSON-first and may omit workflow `state`
because they manage bootstrap assets rather than tracked plan lifecycle state.

The shared bootstrap result should report:

- `ok`
- `command`
- `summary`
- `mode`
- `resource`
- `operation`
- `scope`
- `agent`
- `actions`
- `next_actions`

## Non-Goals

- remote skill registries or marketplace installs
- hidden global install registries outside managed files
- first-class native profiles for every non-Codex agent in the first slice
