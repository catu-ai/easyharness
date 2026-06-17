# Help Topics

## Purpose

`harness help` exposes low-frequency, agent-facing product guidance from the
installed binary. It is for concepts and workflows an agent may need on demand
inside a customer repository, without loading that guidance into every
repository prompt, managed `AGENTS.md` block, or managed skill.

Human maintainers do not need to learn the underlying config or topic details
as their primary workflow. The intended interaction is:

1. The human asks for an outcome.
2. The agent runs `harness help` or `harness help <topic ...>` when the
   easyharness product detail is unclear.
3. The agent applies the change or uses the guidance.
4. The agent reports the result back to the human.

## Command Boundary

`harness help` is topic documentation, not command syntax help.

- `harness <command> --help` remains the surface for usage, flags, required
  inputs, side effects, and short see-also pointers.
- `harness help [topic ...]` prints plain-text product guidance for selected
  topics.
- `harness help --help` and `harness help -h` print syntax help for the help
  command itself.
- Help topics are concept-oriented. They are not required for every command
  and should not become a mirror of the CLI command tree.

## Output Contract

Help topic output is plain text for agents and humans reading a terminal. It is
not JSON and does not define a stable machine-readable schema.

Topic text may use Markdown-like examples and code fences, but callers should
treat the output as human-readable guidance rather than structured data.

Unknown topics fail with a non-zero exit code and print a useful recovery list:

- if the nearest valid parent has subtopics, print those subtopics
- if the unknown path extends below a leaf, print the root available topics

## Topic Assets and Registry

Long-form help topic bodies live in packaged help assets under `assets/help/`,
not in Go string literals and not under `assets/bootstrap/`.

The program-owned help registry owns:

- topic paths, such as `repo` or `repo config`
- one-line topic summaries
- asset bindings
- parent/child relationships

Every non-leaf topic prints an `Available subtopics` section generated from
the registry. Markdown assets must not manually list available subtopics.

## Managed Instructions Boundary

Managed `AGENTS.md` blocks may tell agents that `harness help` exists and when
to use it. They should not copy low-frequency topic content into the managed
agreement.

This keeps always-loaded instructions small while preserving a binary-shipped
way for agents to discover product details in repositories that only installed
`easyharness`.

## Initial Topics

The initial topic set is intentionally narrow:

- `harness help repo`
- `harness help repo config`

`harness help repo config` is the agent-facing entrypoint for repo config
customization. When a human asks for repo customization and the exact config
shape is unclear, the agent should read that topic, apply the requested
customization, verify effective values with `harness repo config get` or
`harness repo config list`, and summarize the result back to the human.
