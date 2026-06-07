# Repo Config

## Purpose

`.harness/config.yaml` is the tracked repo-level customization manifest for
easyharness. It is an index and contract entrypoint, not a dumping ground for
long prompts or hidden workflow behavior.

The config root is:

```text
.harness/
```

The manifest file is:

```text
.harness/config.yaml
```

## Version 1

The smallest valid config is:

```yaml
version: 1
```

`version` is required when the file exists. The value must be integer `1`.

Version 1 also supports the first concrete customization field, `paths`, for
repo-configured harness path roots:

```yaml
version: 1
paths:
  plans:
    active: docs/plans/active
    archived: docs/plans/archived
  local_runtime: .local/harness
```

All path fields are optional. Omitted fields use the defaults shown above.

Path roots mean:

- `paths.plans.active`: the tracked active plan root
- `paths.plans.archived`: the tracked standard archived plan root
- `paths.local_runtime`: the command-owned disposable local runtime root

Supplements are not independently configured. They are derived from the
matching plan root as `supplements/<plan-stem>`.

Lightweight archived plan snapshots, current-plan pointer, plan-local state,
reviews, evidence, timeline events, locks, and other command-owned local
runtime files are derived from `paths.local_runtime`.

## Loading Behavior

- missing `.harness/config.yaml` is valid and uses built-in defaults
- valid v1 config is parsed and available to repo-resource consumers
- invalid config produces an agent-facing warning, is ignored as a whole, and
  falls back to built-in defaults

Invalid config includes malformed YAML, non-object YAML, missing `version`,
unsupported versions, fields not defined by the current config version, unsafe
path values, and ambiguous path roots.

Path values must be slash-separated repo-relative paths. They must not be
empty, absolute, home-relative, backslash-separated, outside the repository, or
the repository root itself. Configured path roots must not overlap with each
other.

Consumers must not partially consume invalid config. Whole-config fallback
keeps precedence and debugging simple until concrete customization fields
exist.

## Repo Resource Initialization

`harness repo init` creates `.harness/config.yaml` with the minimal v1 content
when the file is missing. It must not overwrite an existing config file, even
when that file is invalid.

`harness repo config init` is the focused command for creating the same minimal
config file without installing or refreshing other repo resources.

## Future Fields

Future customization fields should keep `.harness/config.yaml` as a manifest
and index. Long-form customization text belongs in referenced
`.harness/**/*.md` files rather than inline YAML.

This v1 contract does not define review defaults, remote mappings, instruction
content customization, executable hooks, plugins, or provider mapping.
