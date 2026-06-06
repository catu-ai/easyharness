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

The first supported config shape is intentionally minimal:

```yaml
version: 1
```

`version` is required when the file exists. The value must be integer `1`.
No other fields are defined in v1.

## Loading Behavior

- missing `.harness/config.yaml` is valid and uses built-in defaults
- valid v1 config is parsed and available to repo-resource consumers
- invalid config produces an agent-facing warning, is ignored as a whole, and
  falls back to built-in defaults

Invalid config includes malformed YAML, non-object YAML, missing `version`,
unsupported versions, and fields not defined by the current config version.

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

This v1 contract does not define review defaults, remote mappings, plan paths,
instruction content customization, executable hooks, plugins, or provider
mapping.
