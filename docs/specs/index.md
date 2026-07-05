# Specifications

- [State Model](./state-model.md): normative v0.2 canonical-node model for
  `current_node`, node semantics, ownership split, and status-rendering
  principles.
- [State Transitions](./state-transitions.md): exhaustive enumeration of every
  allowed v0.2 `current_node` transition, including command-driven milestones
  and derived progression rules.
- [Plan Schema](./plan-schema.md): shared plan contract for tracked `standard`,
  `lightweight`, and goal-oriented authoring-preview plans, their
  markdown-led package layout, and local state expectations.
- [Goal-Oriented Workflow](./goal-oriented-workflow.md): normative
  `workflow_profile: goal_oriented` authoring-preview contract for adaptive
  work with explicit objectives, scorecards, checkpoint reports, optional
  challenge, evidence, and final synthesis.
- [Bootstrap Install](./bootstrap-install.md): normative bootstrap resource
  model for `harness repo init`, `harness repo skills ...`,
  `harness repo instructions ...`, and `harness repo config ...`, including
  ownership, version markers, and support boundaries.
- [Repo Config](./repo-config.md): normative `.harness/config.yaml` v1
  manifest contract, including missing/valid/invalid behavior and future
  field-extension rules.
- [CLI Contract](./cli-contract.md): agent-facing command surface and JSON
  contract, including how the bootstrap commands fit into the overall CLI.
- [Help Topics](./help.md): normative contract for plain-text
  `harness help` topic guidance, including the command-help boundary,
  asset-backed topic bodies, and generated subtopic discovery.
- [Contract Registry](./contract.md): normative guide to the checked-in JSON
  Schema registry, its ownership model, the public-vs-runtime boundary, and
  what is intentionally not rendered as duplicated markdown.
- [Watchlist Contract](./watchlist-contract.md): normative machine-local
  storage contract for watched git-backed workspaces, including the minimal
  persisted schema, path-based local identity, and the boundary between
  persisted workspace records and derived repository-family grouping.
- [Schema Registry](../../schema/index.json): checked-in JSON Schema index for
  command outputs, command inputs, shared shapes, and CLI-owned local JSON
  artifacts, with each entry labeled by surface.

## Proposals

- [Harness UI Steering Surface Proposal](./proposals/harness-ui-steering-surface.md):
  non-normative recommendation for the long-term `harness ui` workbench shape,
  navigation model, and information architecture.
- [Project Naming Proposal: `easyharness`](./proposals/project-name-easyharness.md):
  non-normative recommendation that favors `easyharness` over
  `microharness` and `superharness` from a user-mental-model perspective.
- [Testing Structure Proposal](./proposals/testing-structure.md): non-normative
  proposal for how `easyharness` should organize smoke, end-to-end, and
  resilience tests alongside the existing package-level Go test suite.
