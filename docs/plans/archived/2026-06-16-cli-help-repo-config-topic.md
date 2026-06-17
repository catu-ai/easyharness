---
template_version: 0.2.0
created_at: "2026-06-16T23:12:24+08:00"
approved_at: "2026-06-16T23:14:36+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/237
size: S
---

# Add CLI Help Topic for Repo Config Customization

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add a minimal `harness help` topic surface so agents in customer repositories
can discover repo config customization guidance from the installed
`easyharness` binary alone.

This slice is not a broad documentation system. It should provide just enough
plain-text, CLI-discoverable help for an agent to understand how to customize
`.harness/config.yaml`, verify effective values, and explain the result back to
the human who asked for the customization.

## Scope

### In Scope

- Add a root `harness help` command that prints available help topics.
- Add `harness help repo` as a short parent topic for repo-related help.
- Add `harness help repo config` as the agent-facing repo config
  customization guide.
- Print help topic content as plain text, not JSON.
- Store long-form help topic bodies in packaged assets, not Go string
  literals.
- Maintain the topic tree, topic summaries, and generated "Available
  subtopics" sections in program code.
- Ensure every non-leaf help topic prints its immediate available subtopics
  from the program-maintained registry instead of duplicating those lists in
  Markdown assets.
- Keep the normative help topic contract in an independent help spec, with
  the CLI and repo-config specs linking to that contract instead of duplicating
  it.
- Keep `--help` as command syntax help, with small see-also pointers from the
  relevant repo command help to `harness help repo config`.
- Add a small managed `AGENTS.md` cue telling agents to use `harness help`
  when easyharness product behavior or customization syntax is unclear.
- Cover the minimal command behavior and asset-backed rendering with focused
  tests.
- Create or update a follow-up GitHub issue before archive for the broader
  help topic system and coverage policy.

### Out of Scope

- Building a complete help taxonomy for all harness commands.
- Requiring every command to have a matching `harness help ...` topic.
- Making `harness repo --help` or other command `--help` output equivalent to
  `harness help repo`.
- JSON output, schemas, or stable machine-readable contracts for help topics.
- Putting repo config customization details into the managed `AGENTS.md` block
  or managed skill pack; the managed block only gets a discovery cue.
- Teaching human maintainers to hand-write `.harness/config.yaml` directly as
  the primary interaction model.
- Config mutation commands, config lint, or new repo config fields.

## Acceptance Criteria

- [x] `harness help` exits successfully and prints a plain-text topic index
      that includes `repo`.
- [x] `harness help repo` exits successfully, prints any repo parent topic
      body, and includes a generated "Available subtopics" section containing
      `config`.
- [x] `harness help repo config` exits successfully and prints a plain-text
      agent guide for repo config customization.
- [x] Help topic output is not JSON and does not expose Markdown examples
      through escaped JSON strings.
- [x] Long-form help bodies are sourced from packaged assets under a dedicated
      help asset tree, separate from `assets/bootstrap/`.
- [x] The program-owned help registry defines topic IDs, summaries, asset
      bindings, and parent/child relationships.
- [x] Every non-leaf help topic prints immediate subtopics from the registry;
      Markdown help assets do not manually list available subtopics.
- [x] Unknown help topics fail clearly with a non-zero exit code and show the
      nearest useful available topics.
- [x] `harness repo --help` and `harness repo config --help` remain command
      syntax help and include a concise see-also pointer to
      `harness help repo config`.
- [x] The repo config help topic tells agents that humans express intent,
      agents read help/config guidance, agents edit `.harness/config.yaml`,
      and agents report the effective result back to the human.
- [x] The repo config help topic covers supported v1 fields,
      `harness repo config get/list` verification, path safety constraints,
      invalid-config fallback/warning behavior, and repo review dimension file
      placement at a useful high level.
- [x] A standalone help spec owns the normative `harness help` topic contract,
      while the CLI and repo-config specs link to it.
- [x] The managed `AGENTS.md` block tells agents to use `harness help` for
      unclear product behavior or customization syntax without copying help
      topic content into the always-loaded agreement.
- [x] Tests cover root help, parent help with generated subtopics, leaf topic
      rendering, unknown topic errors, asset-backed body loading, and see-also
      text in relevant command help.
- [x] A follow-up GitHub issue exists for the broader `harness help` topic
      system, including topic coverage policy and deeper `--help` integration,
      and is recorded in this plan before archive.

## Deferred Items

- A broader `harness help` topic system and coverage policy is deferred to a
  follow-up issue. That work should decide how many product concepts deserve
  topic docs, how command syntax help should point into topic help, and whether
  any content should be generated from specs.
- More help topics beyond `repo` and `repo config` are deferred until each
  topic has a real agent-facing need.
- Aliases that make command `--help` output equivalent to topic help are
  deferred. This slice keeps command syntax help and topic docs distinct.

## Work Breakdown

### Step 1: Define the help topic contract

- Done: [x]

#### Objective

Document the minimal `harness help` behavior and the boundary between command
syntax help and agent-facing topic docs.

#### Details

The contract should say that `harness help` topics are plain-text product and
workflow guidance for agents, not script-facing JSON APIs and not a mirror of
every command. Command `--help` remains short syntax help.

Record the generated-subtopic rule: every non-leaf help topic prints immediate
available subtopics from the program-maintained registry, while Markdown assets
own only body content.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/help.md`
- `docs/specs/index.md`
- `docs/specs/repo-config.md`
- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`
- `README.md`

#### Validation

- Specs and README explain how an agent discovers repo config customization
  guidance without teaching humans to hand-author config as the primary flow.
- The documented scope does not imply that all commands need matching help
  topics in this slice.

#### Execution Notes

Documented `harness help [topic ...]` as a plain-text, agent-facing topic
surface distinct from command syntax `--help`. Updated the CLI contract,
repo-config spec, and README to record the asset-backed body rule,
program-owned topic registry, generated non-leaf subtopic sections, and the
initial `repo` / `repo config` topic scope.

Revision 2 extracted the topic contract into standalone
`docs/specs/help.md`, linked the CLI and repo-config specs to it, and added a
small bootstrap-managed `AGENTS.md` cue telling agents to use `harness help`
when product behavior, command concepts, or repo customization syntax is
unclear. The managed cue deliberately avoids copying low-frequency help topic
content into the always-loaded agreement.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 records the approved help-topic contract in
docs. The implemented command behavior and repo-config guide content are
covered by later step validation and review.

### Step 2: Implement asset-backed help topics

- Done: [x]

#### Objective

Add the minimal help topic registry, asset loading, and CLI routing for
`harness help`, `harness help repo`, and `harness help repo config`.

#### Details

Use a dedicated packaged help asset tree such as `assets/help/`. Keep it
separate from `assets/bootstrap/` because these docs are read by the installed
binary on demand; they are not repo resources installed into customer
repositories.

The registry should own:

- topic path, such as `repo` or `repo config`
- one-line summary for generated indexes
- optional body asset path
- parent/child relationships

Unknown topics should fail clearly and show the nearest useful available
topics so agents can recover without guessing.

#### Expected Files

- `assets/help/`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- a small internal help package if useful

#### Validation

- Focused tests cover successful root, parent, and leaf topic rendering.
- Tests prove non-leaf subtopics are generated from the registry, not copied
  from Markdown assets.
- Tests cover unknown topic behavior and plain-text output.

#### Execution Notes

Added a dedicated `assets/help/` package for packaged help assets and a small
`internal/helptopics` renderer with a program-owned registry for topic paths,
summaries, asset bindings, and generated child-topic listings. Wired
`harness help`, `harness help repo`, and `harness help repo config` through the
CLI as stdout plain-text topic output while keeping `harness --help` on the
existing command-usage surface.

Validation passed with `go test ./internal/cli ./internal/helptopics
./internal/repoconfig`, `scripts/install-dev-harness`, and manual probes for
`harness help`, `harness help repo`, `harness help repo config`,
`harness help repo missing`, `harness repo --help`, and
`harness repo config --help`. A broader `go test ./...` run passed through
core packages, e2e, and resilience output but was interrupted after
`tests/smoke` stayed quiet for several minutes; no assertion failure was
observed before the interrupt.

#### Review Notes

Delta review `review-001-delta` found one duplicated important issue across
`correctness` and `docs-consistency`: `harness help --help` and
`harness help -h` were routed as unknown help topics instead of command syntax
help. Added regression coverage for both forms, implemented `printHelpUsage`,
and verified the repaired behavior with `go test -count=1 ./internal/cli
./internal/helptopics ./internal/repoconfig`, `scripts/install-dev-harness`,
and manual probes for `harness help --help`, `harness help -h`, and
`harness help repo config`. `go test ./tests/smoke -run
TestHelpShowsTopLevelUsage -count=1` also passed, but only covers the existing
root help smoke path rather than the repaired help-subcommand flags.
Follow-up delta review `review-002-delta` passed with one non-blocking tests
finding about this smoke-evidence wording, which this note corrects.
Finalize review `review-003-full` then found one important docs-consistency
issue: unknown help topics below a leaf, such as
`harness help repo config missing`, did not print useful available topics even
though the contract promised nearest-topic recovery. It also found one minor
tests issue: root-help tests did not pin the new top-level `help` command line.
Added regression coverage for both cases, changed leaf-child unknown topics to
fall back to the root topic list, and verified with `go test -count=1
./internal/cli ./internal/helptopics ./internal/repoconfig`, `go test -count=1
./tests/smoke -run TestHelpShowsTopLevelUsage`, `scripts/install-dev-harness`,
and manual probes for `harness help repo config missing`, `harness --help`, and
`harness help --help`. Follow-up finalize review is pending.

### Step 3: Write repo config agent guidance and see-also pointers

- Done: [x]

#### Objective

Create the `repo config` help body and make relevant command syntax help point
agents toward it without expanding managed prompts or skills.

#### Details

The guide should assume the human asks for an outcome, not for YAML lessons.
It should instruct agents to inspect the topic help, edit `.harness/config.yaml`
when needed, verify effective values with `harness repo config get/list`, and
report the resulting configuration back to the human.

The guide should include useful examples while staying concise enough for
terminal reading. It should cover:

- `.harness/config.yaml` as tracked manifest/index
- `version: 1`
- optional `paths` fields and their meanings
- safe repo-relative path rules
- whole-config fallback on invalid config, with agent-facing warnings
- effective value verification through `harness repo config get/list`
- review dimension root and Markdown file format at a high level

Relevant command syntax help should get short see-also lines, not long topic
content.

#### Expected Files

- `assets/help/repo.md`
- `assets/help/repo/config.md`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `docs/specs/repo-config.md`

#### Validation

- `harness help repo config` is readable as plain terminal text.
- The guide does not duplicate generated available-subtopic lists in Markdown.
- `harness repo --help` and `harness repo config --help` still read as command
  help and point to `harness help repo config`.

#### Execution Notes

Wrote the `repo` parent topic and `repo config` agent guide as packaged
plain-text Markdown-ish assets. The guide frames the intended human-agent
interaction, documents the v1 config shape and path constraints, points agents
to `harness repo config get/list` for verification, and summarizes
repo-defined review dimension placement. Added concise see-also pointers from
`harness repo --help` and `harness repo config --help` to
`harness help repo config`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 supplies topic text and see-also pointers that
are exercised by the Step 2 CLI/help tests and manual probes; the behavior
change remains pending Step 2 closeout review.

### Step 4: Track the broader help-system follow-up

- Done: [x]

#### Objective

Create or update a GitHub issue for the larger `harness help` system so this
slice can stay intentionally narrow.

#### Details

The follow-up should cover the broader questions that discovery intentionally
left out:

- topic taxonomy beyond repo config customization
- when a concept deserves `harness help ...`
- how command `--help` should point to topic help
- whether command aliases or equivalence rules are desirable
- whether any topic content should be generated from specs

Record the issue URL in the plan before archive.

#### Expected Files

- this plan file
- GitHub issue tracker

#### Validation

- A follow-up issue exists and is listed in `Outcome Summary / Follow-Up
  Issues` before archive.
- The current implementation remains scoped to `repo` and `repo config` help
  topics.

#### Execution Notes

Created follow-up issue https://github.com/catu-ai/easyharness/issues/254
for the broader `harness help` topic system, including topic coverage policy,
command `--help` integration, alias/equivalence decisions, and whether any
topic content should be generated from specs. Labeled it `enhancement` and
`state/accepted`, then added the required triage rationale comment explaining
why it stays outside this minimal repo-config help slice.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 4 only creates and records the deferred follow-up
issue; no repository behavior changes are introduced by this step.

## Validation Strategy

- Run focused Go tests for the CLI and any new help package.
- Run `harness plan lint` before approval and repeat it after material plan
  updates.
- After implementation, reinstall the dev harness with
  `scripts/install-dev-harness` before manual CLI probes.
- Manually probe `harness help`, `harness help repo`,
  `harness help repo config`, an unknown topic, `harness repo --help`, and
  `harness repo config --help`.
- Use text search to confirm available-subtopic lists are not duplicated in
  Markdown help assets.

## Risks

- Risk: `harness help` could accidentally become a promise to mirror every CLI
  command.
  - Mitigation: Define topic help as concept-oriented, keep this slice scoped
    to repo config customization, and defer coverage policy to a follow-up
    issue.
- Risk: Help output could become hard for agents to consume if represented as
  JSON.
  - Mitigation: Make `harness help` plain text only in this slice and keep
    script-facing behavior with existing command surfaces.
- Risk: Markdown assets could drift from the topic tree.
  - Mitigation: Keep subtopic discovery in the program registry and test that
    non-leaf output appends generated subtopics.

## Validation Summary

Passed:

- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-06-16-cli-help-repo-config-topic.md`
- `go test -count=1 ./internal/cli ./internal/helptopics ./internal/repoconfig ./internal/bootstrapsync ./internal/install`
- `go test -count=1 ./tests/smoke -run TestSyncBootstrapAssetsCheckPassesForCurrentRepo`
- `go test -count=1 ./internal/cli ./internal/helptopics ./internal/repoconfig`
- `go test -count=1 ./internal/cli ./internal/helptopics ./internal/repoconfig ./internal/reviewdimensions`
- `go test -count=1 ./tests/smoke -run TestHelpShowsTopLevelUsage`
- `scripts/install-dev-harness`
- manual probes for `harness help`, `harness help repo`,
  `harness help repo config`, `harness help --help`, `harness help -h`,
  `harness help repo missing`, `harness help repo config missing`,
  `harness repo --help`, and `harness repo config --help`

A broader `go test ./...` run was attempted during execution. It passed
through core packages, e2e, and resilience output, then was interrupted after
the smoke package stayed quiet for several minutes. No assertion failure was
observed before the interrupt, but this candidate does not claim a clean full
suite pass.

## Review Summary

Step-closeout review `review-001-delta` found one duplicated important issue:
`harness help --help` and `harness help -h` were treated as unknown topics
instead of command syntax help. The repair added syntax-help handling and
regression coverage. Follow-up `review-002-delta` passed with one minor
evidence wording note, which was corrected in the plan.

Finalize review `review-003-full` found one important issue: unknown help
topics below a leaf did not show the documented recovery topics. It also found
one minor root-help coverage gap. The repair added leaf-child recovery fallback
to root topics plus root-help discoverability assertions. Follow-up
`review-004-full` passed with no findings across correctness, tests,
docs-consistency, agent-UX, and risk-scan.

Revision 2 was reopened from `await_merge` after human feedback requested an
independent help spec and a managed `AGENTS.md` discovery cue for `harness
help`. Delta finalize review `review-005-delta` passed with no findings across
docs-consistency, agent-UX, tests, and risk-scan.

## Archive Summary

- Archived At: 2026-06-17T10:03:25+08:00
- Revision: 2

Revision 1 was archived at 2026-06-16T23:51:59+08:00 and published through PR
https://github.com/catu-ai/easyharness/pull/255, then reopened in
`finalize-fix` mode after human feedback clarified that help needs its own
spec and a managed `AGENTS.md` discoverability cue.

- PR: https://github.com/catu-ai/easyharness/pull/255
- Ready: yes; all tracked steps remain complete, revision 2 focused
  validation passed, and delta finalize review `review-005-delta` passed with
  no findings.
- Merge Handoff: after archive, commit the archive move and summary updates,
  push the branch, update the PR memo if needed, refresh publish/CI/sync
  evidence, and wait for `harness status` to reach
  `execution/finalize/await_merge` before asking for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Added a minimal plain-text `harness help` topic surface with root topic
  index, `repo` parent topic, and `repo config` customization guide.
- Added packaged help assets under `assets/help/`, separate from bootstrap
  install assets.
- Added standalone `docs/specs/help.md` as the normative help topic contract,
  with CLI and repo-config specs linking to it instead of duplicating the same
  rules.
- Added a program-owned help topic registry and renderer that owns topic paths,
  summaries, asset bindings, parent/child relationships, and generated
  subtopic listings for non-leaf topics.
- Added CLI routing and syntax help behavior for `harness help`, including
  `harness help --help` and `harness help -h`.
- Added concise see-also pointers from `harness repo --help` and
  `harness repo config --help` to `harness help repo config`.
- Updated README, CLI contract, and repo config spec to document the
  concept-oriented help topic boundary without implying that every command
  needs a matching long-form topic.
- Added a bootstrap-managed `AGENTS.md` product-help cue telling agents to use
  `harness help` when easyharness behavior or repo customization syntax is
  unclear, while leaving detailed topic content in binary-shipped help.
- Added focused tests for topic rendering, asset-backed bodies, generated
  subtopic lists, unknown-topic recovery, command syntax help, root-help
  discoverability, and repo/config see-also pointers.
- Created follow-up issue https://github.com/catu-ai/easyharness/issues/254
  for the broader help topic system and coverage policy.

### Not Delivered

- A broad `harness help` taxonomy for every command was intentionally not
  delivered.
- Command `--help` output was not made equivalent to topic help.
- Detailed repo config customization guidance was not copied into the managed
  `AGENTS.md` block or managed skills.
- No JSON or schema contract was added for help topic output.
- No repo config mutation command, config lint command, or new repo config
  field was added.

### Follow-Up Issues

- https://github.com/catu-ai/easyharness/issues/254
