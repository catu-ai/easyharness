Repo Config Customization

Use this topic when a human asks to customize where easyharness stores plans,
local runtime state, or repo-defined review dimensions. The expected
interaction is that humans describe the outcome they want, agents read this
help, edit `.harness/config.yaml` when needed, verify the effective values,
and report the result back to the human.

`.harness/config.yaml` is a tracked repo-level manifest and index. It should
stay small. Long-form customization text belongs in referenced `.harness/**/*.md`
files, not inline YAML.

The smallest valid config is:

```yaml
version: 1
```

Version 1 supports optional `paths` fields:

```yaml
version: 1
paths:
  plans:
    active: docs/plans/active
    archived: docs/plans/archived
  local_runtime: .local/harness
  review:
    dimensions: .harness/review/dimensions
```

All path fields are optional. Omitted fields use built-in defaults.

Path meanings:

- `paths.plans.active`: tracked active plan root
- `paths.plans.archived`: tracked standard archived plan root
- `paths.local_runtime`: disposable command-owned runtime root
- `paths.review.dimensions`: repo-defined review dimensions root

Path values must be repo-relative slash-separated paths. Do not use empty
paths, absolute paths, `~`, backslashes, paths outside the repository, the
repository root itself, or overlapping configured roots.

After editing config, verify effective values with:

```bash
harness repo config list
harness repo config get paths.plans.active
harness repo config get paths.plans.archived
harness repo config get paths.local_runtime
harness repo config get paths.review.dimensions
```

If `.harness/config.yaml` is missing, easyharness uses built-in defaults. If
the file exists but is invalid, commands warn the agent, ignore the whole
config, and use built-in defaults instead of partially consuming it.

Repo-defined review dimensions are Markdown files directly under the resolved
`paths.review.dimensions` root, which defaults to:

```text
.harness/review/dimensions
```

Each review dimension file has YAML frontmatter with exactly `name` and
`description`, followed by the reviewer instruction body. Use
`harness review dimensions list` to see available dimensions and
`harness review dimensions instructions <name>` to read the full instruction.
