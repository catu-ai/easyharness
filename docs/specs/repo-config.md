---
title: Repo Config
created_at: "2026-03-17T14:00:00+08:00"
updated_at: "2026-07-14T00:05:00+08:00"
reviewed_at: "2026-07-14T00:05:00+08:00"
status: active
---

# Repo Config

## Purpose

`.harness/config.yaml` is the tracked repository customization manifest. It
selects plan and local-runtime roots; it does not contain prompts, reviewer
topology, or review guidance.

## Version 1

The smallest valid config is:

```yaml
version: 1
```

Optional path roots are:

```yaml
version: 1
paths:
  plans:
    active: docs/plans/active
    archived: docs/plans/archived
  local_runtime: .local/harness
```

Defaults are:

- `paths.plans.active`: `docs/plans/active`
- `paths.plans.archived`: `docs/plans/archived`
- `paths.local_runtime`: `.local/harness`

Supplements derive from the matching plan root as
`supplements/<plan-stem>/`. Lightweight archive snapshots and command-owned
state derive from `paths.local_runtime`.

Path values must be slash-separated, repository-relative, non-empty, inside
the repository, distinct, and non-overlapping. Unsupported fields are invalid;
there is no review-dimensions config surface.

## Loading and Rendering

A missing config is valid and uses defaults. A valid existing config applies
specified values and fills omitted fields from defaults. An invalid existing
config is ignored as a whole with an agent-facing warning; consumers never
partially apply it.

`harness repo init` and `harness repo config init` create the canonical
default-equivalent file when missing. They do not overwrite an existing file.

`harness repo config refresh --diff` previews canonical rendering.
`harness repo config refresh` creates a missing file or rewrites a valid file,
preserving supported values but not comments or original field order. Refresh
fails without writing when the existing config is invalid.

## Queries

Supported scalar keys are:

- `paths.plans.active`
- `paths.plans.archived`
- `paths.local_runtime`

Use `harness repo config get <key>` for one value and
`harness repo config list [prefix]` for deterministic `key=value` output.
Object keys such as `paths` and `paths.plans` are list prefixes, not scalar
values.

## Agent Help

`harness help repo config` exposes the same concise customization contract from
the installed binary.
