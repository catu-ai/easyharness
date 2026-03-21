# Specifications

- [State Model](./state-model.md): consolidated map of the current layered
  workflow state, derived substate, transition triggers, and `.local` inputs.
- [Plan Schema](./plan-schema.md): durable tracked-plan contract plus local
  state expectations for archive and reopen flows.
- [CLI Contract](./cli-contract.md): v0.1 command surface and output envelope for
  agent-facing CLI behavior.

## Proposals

- [State Model v0.2 Proposal](./proposals/state-model-v0.2.md): proposed
  simplified canonical-node state model that replaces v0.1 layered lifecycle,
  step-state, and handoff-state behavior.
- [Testing Structure Proposal](./proposals/testing-structure.md): non-normative
  proposal for how `superharness` should organize smoke, end-to-end, and
  resilience tests alongside the existing package-level Go test suite.
