Repo Config Customization

Use this topic when a human asks to customize where easyharness stores plans
or disposable local runtime state.

The smallest valid `.harness/config.yaml` is:

```yaml
version: 1
```

Version 1 supports optional path roots:

```yaml
version: 1
paths:
  plans:
    active: docs/plans/active
    archived: docs/plans/archived
  local_runtime: .local/harness
```

Omitted fields use built-in defaults. Path values must be repo-relative,
slash-separated, non-overlapping, and must not resolve to the repository root.

Verify effective values with:

```bash
harness repo config list
harness repo config get paths.plans.active
harness repo config get paths.plans.archived
harness repo config get paths.local_runtime
```

Preview canonical refresh with `harness repo config refresh --diff`, then apply
it with `harness repo config refresh`. Invalid existing config is ignored with
a warning by ordinary consumers, but refresh fails rather than overwriting it.
