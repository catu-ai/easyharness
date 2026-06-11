---
template_version: 0.2.0
created_at: "2026-06-11T00:16:34+08:00"
approved_at: "2026-06-11T00:17:52+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/230
size: M
---

# Add review dimension catalog

## Goal

Add a small, agent-facing review dimension catalog so controller agents can
discover recommended review dimensions without carrying long reviewer
instructions in their own context, and reviewer agents can fetch the full
instruction for their assigned dimension when they need it.

The design should treat review dimensions as skill-like repo resources:
dimensions have stable names, concise descriptions for controller selection,
and Markdown bodies for reviewer instructions. Built-in dimensions and
repo-defined dimensions should appear through one catalog surface, while review
rounds still use explicit dimensions and do not gain automatic ref parsing or
CLI-enforced slot selection.

## Scope

### In Scope

- Define the review dimension file format: Markdown files with YAML
  frontmatter containing exactly `name` and `description`, followed by the full
  reviewer instruction body.
- Add built-in review dimensions with complete descriptions and instruction
  bodies.
- Discover repo-defined dimensions from a configured dimensions root,
  defaulting to `.harness/review/dimensions`.
- Extend repo config path roots with `paths.review.dimensions`.
- Add `harness review dimensions list`, returning compact JSON metadata for
  both built-in and repo-defined dimensions.
- Add `harness review dimensions instructions <name>`, returning the raw
  Markdown instruction body for a built-in or repo-defined dimension.
- Make dimension names stable skill-like identifiers made from lowercase
  alphanumeric segments separated by single hyphens. Treat the dimension name
  as the review slot identifier for catalog-managed dimensions.
- Let repo-defined dimensions override built-in dimensions with the same name.
- Update controller and reviewer guidance so controllers use `list` to choose
  dimensions and reviewers use `instructions <name>` to load full instructions.
- Update specs, command help, generated schemas or command references, tests,
  and bootstrap assets/materialized skills as needed.

### Out of Scope

- Adding `{ "ref": "dimension-name" }` or any other ref parsing to
  `harness review start`.
- Automatically injecting repo-recommended dimensions into every review round.
- Forcing controller agents to use every recommended dimension.
- Adding source types beyond `builtin` and `repo`.
- Supporting embedded instructions in `.harness/config.yaml`.
- Supporting display names separate from stable dimension names.
- Changing review result semantics, submission schemas, aggregate semantics, or
  reviewer slot lifecycle beyond the new catalog guidance.

## Acceptance Criteria

- [x] `harness review dimensions list` returns JSON by default with built-in
      and repo-defined dimensions, including `name`, `source`, and
      `description`, and excluding full instruction bodies.
- [x] `source` is limited to `builtin` and `repo`.
- [x] Built-in dimensions include `correctness`, `tests`,
      `docs-consistency`, `agent-ux`, and `risk-scan`, each with a useful
      description and full Markdown instruction body.
- [x] Repo dimension files under the configured dimensions root are discovered
      automatically without listing them in `.harness/config.yaml`.
- [x] Repo dimension files require `name` and `description` frontmatter plus a
      non-empty instruction body.
- [x] Dimension names must be made from lowercase alphanumeric segments
      separated by single hyphens.
- [x] Repo dimensions override built-ins with the same `name`; duplicate repo
      dimension names are reported clearly.
- [x] `harness review dimensions instructions <name>` returns raw Markdown
      instructions for both built-in and repo dimensions.
- [x] `paths.review.dimensions` is supported by repo config loading,
      validation, rendering, query/list surfaces, and docs.
- [x] Controller guidance says to use `harness review dimensions list` before
      choosing review dimensions, then still pass explicit `dimensions` to
      `harness review start`.
- [x] Reviewer guidance says to use
      `harness review dimensions instructions <assigned-name>` to load the
      full instruction for the assigned dimension.
- [x] Existing `harness review start` input remains explicit
      `name`/`instructions`; no dimension ref parser is added.
- [x] Tests cover built-in catalog output, repo discovery, repo-overrides-
      builtin behavior, invalid dimension files, configured dimensions root,
      and raw instruction output.

## Deferred Items

- Recording instruction file hashes or frozen instruction snapshots in review
  round artifacts is deferred. This slice intentionally lets reviewers read the
  current catalog instruction when their slot runs.
- Additional dimension sources such as user-level, plugin-level, or
  organization-level catalogs are deferred until a real source exists.
- A richer dimension authoring command is deferred; this slice only defines
  discovery and read surfaces.

## Work Breakdown

### Step 1: Define dimension catalog contracts

- Done: [x]

#### Objective

Document the dimension catalog model and update the contract/spec surface
before implementation.

#### Details

Define dimension files as Markdown resources with YAML frontmatter containing
only `name` and `description`; the body is the full reviewer instruction.
Names are stable skill-like IDs made from lowercase alphanumeric segments
separated by single hyphens. The dimensions root defaults to
`.harness/review/dimensions` and is configurable through
`paths.review.dimensions`.

Specify that `list` returns compact JSON metadata for controller selection,
while `instructions <name>` returns raw Markdown for reviewer use. Clarify that
repo dimensions override built-ins by name, and that review start still
requires explicit dimensions rather than refs.

#### Expected Files

- `docs/specs/repo-config.md`
- `docs/specs/cli-contract.md`
- `docs/specs/index.md`
- `schema/commands/*` as needed for command outputs

#### Validation

- Specs describe one coherent catalog model without adding hidden auto-injection
  or ref parsing.
- Any schema/reference updates are regenerated or kept in sync with the
  repository's existing contract tooling.

#### Execution Notes

Updated `docs/specs/repo-config.md` and `docs/specs/cli-contract.md` to define
the review dimension catalog, stable hyphenated dimension names, metadata-only
`list`, raw Markdown `instructions`, repo-overrides-builtin behavior, and the
explicit `review start` boundary. Added the command result contract and
regenerated schema/index artifacts with `go run ./cmd/contract-sync`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Contract docs and generated schema are covered by the
integrated catalog tests and final review for the cohesive slice. Finalize
review `review-003-full` found stale command-surface inventory and loose
dimension-name wording; repaired the docs, contract comments, generated
schemas, and parser error text to match the segmented slug rule.

### Step 2: Extend repo config paths

- Done: [x]

#### Objective

Teach repo config to resolve and validate the review dimensions root.

#### Details

Add `paths.review.dimensions` with default
`.harness/review/dimensions`. Reuse the existing repo-relative path safety
model: paths must be repo-relative, slash-separated, non-empty, not absolute,
not home-relative, not outside the repository, and not the repository root.

Include the new path in rendered canonical config comments and in config query
or listing surfaces that exist in the current command shape. The current
worktree command surface may differ from older `harness repo config ...`
documentation; execution should align the implementation and docs to the
current supported CLI surface rather than preserving obsolete command shapes.

#### Expected Files

- `internal/repoconfig/config.go`
- `internal/repoconfig/config_test.go`
- `internal/install/service.go`
- `internal/install/service_test.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `.harness/config.yaml`

#### Validation

- Focused repo config tests cover defaults, custom dimensions root rendering,
  invalid values, and coexistence with existing plan/local runtime paths.
- Command-level tests cover any exposed config query/list/update behavior that
  now includes `paths.review.dimensions`.

#### Execution Notes

Added `paths.review.dimensions` to repo config defaults, parsing, rendering,
validation, scalar get/list output, canonical config comments, install tests,
and this repository's `.harness/config.yaml` through
`go run ./cmd/harness repo config refresh`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The config path change is covered by focused
`repoconfig`, `install`, `cli`, and smoke tests plus final review.

### Step 3: Implement dimension discovery and commands

- Done: [x]

#### Objective

Provide the built-in and repo-defined dimension catalog behind
`harness review dimensions list` and
`harness review dimensions instructions <name>`.

#### Details

Add a review dimension catalog package or service that owns:

- built-in dimensions
- repo dimension discovery from the resolved root
- frontmatter parsing and validation
- name validation
- repo-overrides-builtin merge behavior
- duplicate repo name diagnostics
- compact metadata output for list
- raw Markdown instruction output for instructions

The built-in dimensions are:

- `correctness`
- `tests`
- `docs-consistency`
- `agent-ux`
- `risk-scan`

`list` should return JSON by default, following current harness command output
style. `instructions <name>` should print only the raw Markdown instruction
body on success so reviewer agents can read it without JSON escaping noise.
Errors should remain clear and agent-facing.

#### Expected Files

- `internal/reviewdimensions/` or another appropriately named package
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `schema/commands/review.dimensions.*.schema.json` as needed
- `internal/contracts/` as needed

#### Validation

- Unit tests cover built-in list and instruction output.
- Unit tests cover repo discovery, invalid frontmatter, missing fields,
  invalid names, empty bodies, duplicates, and repo overrides.
- CLI tests cover successful `list`, successful `instructions`, unknown
  dimension names, and configured dimensions roots.

#### Execution Notes

Added `internal/reviewdimensions` for built-in dimensions, repo markdown
discovery, strict frontmatter/name/body validation, duplicate detection,
repo-overrides-builtin merging, compact list metadata, and raw instruction
lookup. Added `harness review dimensions list` and
`harness review dimensions instructions <name>` CLI routes and tests.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Behavior is covered by focused catalog/CLI tests and
the integrated final review.

### Step 4: Update agent guidance and bootstrap assets

- Done: [x]

#### Objective

Update managed review orchestration guidance so controllers and reviewers use
the new catalog surfaces correctly.

#### Details

Controller guidance should say:

- run `harness review dimensions list`
- select only dimensions that fit the current change
- still create explicit `harness review start` specs
- do not assume CLI auto-injects recommended dimensions

Reviewer guidance should say:

- use the assigned dimension name
- run `harness review dimensions instructions <assigned-name>`
- follow the returned Markdown instruction
- submit through the existing `harness review submit` flow

Update `assets/bootstrap/` first, then run the repository bootstrap sync script
so `.agents/skills/` and the managed root `AGENTS.md` block stay materialized
from source assets.

#### Expected Files

- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`
- `.agents/skills/harness-reviewer/SKILL.md`
- `AGENTS.md`

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Confirm materialized skill files match bootstrap sources.
- Guidance does not instruct reviewer agents to read controller-only catalog
  metadata as their full review instruction.

#### Execution Notes

Updated bootstrap source guidance for controller review orchestration and
reviewer behavior, then ran `scripts/sync-bootstrap-assets` to refresh the
materialized `.agents/skills` copies.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Finalize review `review-001-full` covered this guidance
slice and found repair work. Repaired the guidance so catalog-managed
dimensions use an instruction command, one-off slots may carry explicit
reviewer instructions, reviewers can fall back to explicit slot instructions
when lookup is not available, and the resume prompt now mirrors the first-pass
`Instruction command` / `Instruction handoff` fields. Finalize review
`review-003-full` found that the fixed prompt template still assumed every
dimension had a catalog command; repaired the template so one-off slots can
explicitly use `none` for the command and direct guidance as the handoff.

### Step 5: Validate the integrated review workflow

- Done: [x]

#### Objective

Exercise the new catalog surfaces together with the existing explicit review
start/submit flow.

#### Details

Add or update smoke/e2e coverage so a fixture repository can define a repo
dimension file, list dimensions, fetch instructions, start a review with an
explicit dimension using the catalog name, and submit a reviewer result through
the existing slot flow.

Because this slice does not add ref parsing, tests should prove that catalog
commands are the discovery/read surface and `review start` remains explicit.

#### Expected Files

- `tests/smoke/*`
- `tests/e2e/*`
- `internal/inputschema/*` if generated schema output changes
- generated contract references as needed

#### Validation

- Run focused package tests for repo config, dimension catalog, CLI, and review
  service changes.
- Run relevant smoke/e2e tests that cover review dimension catalog behavior.
- Run the repository's standard formatting/generation checks required by any
  touched Go, schema, or bootstrap files.

#### Execution Notes

Added focused service and CLI coverage for catalog list output, configured repo
dimensions root discovery, repo dimension instructions, unknown dimension
errors, invalid repo dimensions, duplicate names, and repo-overrides-builtin
behavior. Verified the repo-local binary with `.local/bin/harness review
dimensions list`, `.local/bin/harness review dimensions instructions
correctness`, and `.local/bin/harness repo config get paths.review.dimensions`.
After finalize review, expanded invalid repo dimension file coverage to include
malformed frontmatter, missing metadata, unsupported frontmatter fields, and
empty instruction bodies. Expanded smoke coverage so a fixture repo defines a
repo dimension, lists it, fetches instructions, starts a review using the
catalog dimension name, and submits a reviewer result through the existing slot
flow.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Finalize review `review-001-full` covered this
validation slice and found missing integrated review start/submit smoke
coverage plus several invalid dimension-file cases. Repaired with the expanded
service and smoke tests described above. Finalize review `review-003-full`
found the durable validation note overstated smoke coverage; repaired the note
to distinguish focused service/CLI coverage from the integrated smoke path.

## Validation Strategy

- Start with focused unit tests for repo config path resolution and dimension
  catalog parsing/merging.
- Add CLI tests for both new commands and their error cases.
- Add integration or smoke coverage for configured repo dimensions root and
  repo-overrides-builtin behavior.
- Run `scripts/sync-bootstrap-assets` after bootstrap edits.
- Run `harness plan lint` before approval and the relevant Go/smoke/e2e test
  suites during execution.

## Risks

- Risk: The new catalog could accidentally look like automatic review slot
  selection.
  - Mitigation: Keep `review start` explicit, avoid ref parsing, and make
    controller guidance say that `list` is for selection only.
- Risk: Long instructions could leak back into controller context.
  - Mitigation: Keep `list` metadata-only and make `instructions <name>` the
    reviewer-facing raw Markdown command.
- Risk: `name` and `slot` terminology could stay confusing.
  - Mitigation: Define catalog dimension `name` as the stable skill-like ID and
    keep review artifact `slot` equal to that name for catalog-managed
    dimensions.
- Risk: Repo config command docs may be stale relative to the current CLI.
  - Mitigation: During execution, reconcile the plan's repo config path work
    with the current supported command surface and update specs/tests together.

## Validation Summary

- `go test ./internal/reviewdimensions ./internal/repoconfig ./internal/cli ./internal/install ./internal/contractsync`
- `go test ./internal/reviewdimensions ./internal/cli ./internal/repoconfig ./internal/contractsync`
- `go test ./tests/smoke -run TestReviewDimensionsCatalogViaCLI -count=1`
- `go run ./cmd/contract-sync`
- `go run ./cmd/contract-sync --check`
- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `git diff --check`
- `.local/bin/harness plan lint docs/plans/active/2026-06-11-review-dimension-catalog.md`
- `go test ./...`
- Revision 2 reopen validation after merging `origin/main`:
  `go test ./internal/reviewdimensions ./internal/cli ./internal/repoconfig ./internal/contractsync ./tests/smoke -run TestReviewDimensionsCatalogViaCLI`
  passed.
- Full `go test ./...` passed after merging `origin/main`.
- Revision 3 finalize-fix repair updated controller/reviewer guidance and CLI
  contract docs for catalog source-of-truth and first-class one-off dimensions,
  then refreshed materialized bootstrap assets with
  `scripts/sync-bootstrap-assets` and contract artifacts with
  `go run ./cmd/contract-sync`.
- Revision 3 validation passed:
  `scripts/sync-bootstrap-assets --check`,
  `go run ./cmd/contract-sync --check`,
  `.local/bin/harness plan lint docs/plans/active/2026-06-11-review-dimension-catalog.md`,
  `git diff --check`,
  `go test ./internal/reviewdimensions ./internal/cli ./internal/repoconfig ./internal/contractsync`,
  `go test ./tests/smoke -run TestReviewDimensionsCatalogViaCLI -count=1`,
  and full
  `COREPACK_HOME="$HOME/.cache/node/corepack" NPM_CONFIG_STORE_DIR="$HOME/Library/pnpm/store/v10" GOPROXY="file://$(go env GOMODCACHE)/cache/download,https://proxy.golang.org,direct" go test ./...`.
  Earlier unqualified full smoke runs failed only while isolated installer
  tests repeatedly fetched `pnpm` packages or Go modules from external
  registries; the final cached full run passed.
- Revision 4 reopened after publish sync evidence reported the branch stale
  against `origin/main`; merged `origin/main` cleanly to refresh the candidate.
- Revision 4 validation passed:
  `scripts/sync-bootstrap-assets --check`,
  `go run ./cmd/contract-sync --check`,
  `.local/bin/harness plan lint docs/plans/active/2026-06-11-review-dimension-catalog.md`,
  `git diff --check`,
  `go test ./internal/reviewdimensions ./internal/cli ./internal/repoconfig ./internal/contractsync`,
  `go test ./tests/smoke -run TestReviewDimensionsCatalogViaCLI -count=1`,
  and full
  `COREPACK_HOME="$HOME/.cache/node/corepack" NPM_CONFIG_STORE_DIR="$HOME/Library/pnpm/store/v10" GOPROXY="file://$(go env GOMODCACHE)/cache/download,https://proxy.golang.org,direct" go test ./...`.

## Review Summary

- `review-001-full` found three blocking issues: missing integrated
  review-start/submit smoke coverage, missing invalid repo dimension file
  cases, and guidance that made catalog lookup too mandatory while the review
  spec still supports explicit one-off instructions. All were repaired.
- `review-002-delta` passed with no findings after the first repair.
- `review-003-full` found two blocking documentation consistency issues and
  four non-blocking clarity issues around command-surface docs, segmented slug
  naming, prompt templates, and validation wording. All were repaired.
- `review-004-full` passed with two duplicate non-blocking README command-list
  findings. README was updated to include the new review dimension commands.
- Revision 2 was reopened after publish evidence showed the branch was stale
  against `origin/main`; the branch merged `origin/main` cleanly.
- `review-005-full` found one blocking docs issue: specs did not explicitly say
  repo dimension frontmatter rejects fields beyond `name` and `description`.
  It also found a non-blocking README command inventory drift for
  `harness evidence refresh`. Both were repaired.
- `review-006-full` found one blocking command-inventory drift for
  `harness plan approve`; README and CLI contract command inventories were
  updated.
- `review-007-full` passed with no findings after the final command-inventory
  repair.
- Revision 3 was reopened from `await_merge` after human diff review pointed
  out that guidance was duplicating built-in dimensions now exposed by the CLI
  and treating non-catalog dimensions like fallback behavior. Repaired guidance
  so catalog-managed dimensions come from
  `harness review dimensions list`, while one-off dimensions remain explicit,
  first-class review slots with controller-written instructions.
- `review-008-full` passed with no findings after the revision 3 guidance and
  contract repair.
- Revision 4 was reopened only to refresh against `origin/main` after remote
  sync evidence became stale again. The merge completed without conflicts.
- `review-009-full` passed with no findings after the revision 4 main refresh.

## Archive Summary

- Archived At: 2026-06-11T11:03:51+08:00
- Revision: 4
- Reopen History: Revision 1 was archived at `2026-06-11T01:15:42+08:00`,
  published to PR #253, then reopened because `origin/main` advanced and sync
  evidence reported the PR branch as stale. Revision 2 was archived at
  `2026-06-11T01:43:52+08:00`, published to PR #253, then reopened from
  `await_merge` for human diff feedback about catalog source-of-truth and
  one-off dimension semantics. Revision 3 was archived at
  `2026-06-11T10:50:39+08:00`, published to PR #253, then reopened because
  `origin/main` advanced again and sync evidence reported the PR branch as
  stale.
- PR: [#253](https://github.com/catu-ai/easyharness/pull/253)
- Ready: Revision 4 validation passed and `review-009-full` passed with no
  findings after merging latest `origin/main`.
- Merge Handoff: After re-archive, commit the revision 4 merge-base refresh
  plus archive move, push the branch to PR #253, refresh publish/CI/sync
  evidence, and wait for `harness status` to reach
  `execution/finalize/await_merge` before asking for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Added the built-in review dimension catalog with `correctness`, `tests`,
  `docs-consistency`, `agent-ux`, and `risk-scan` dimensions.
- Added repo-defined Markdown dimension discovery under configurable
  `paths.review.dimensions`, defaulting to `.harness/review/dimensions`.
- Added metadata-only `harness review dimensions list` and raw Markdown
  `harness review dimensions instructions <name>` command surfaces.
- Preserved explicit `harness review start` `name`/`instructions` semantics;
  catalog-managed dimensions use stable names as slots, while one-off slots can
  still carry direct instructions.
- Updated repo config docs, CLI contract docs, README command inventory,
  contracts, generated schemas, controller/reviewer skill guidance, and
  materialized `.agents` outputs.
- Added focused service/CLI coverage and integrated smoke coverage for repo
  dimension discovery, instruction loading, explicit review start, and reviewer
  submission.
- Reopened revision 1 after stale publish sync evidence and merged
  `origin/main` so the final candidate is based on the current mainline.
- Reopened revision 2 after human diff feedback, removed the static built-in
  dimension list from controller guidance, made
  `harness review dimensions list` the durable source of truth for
  catalog-managed dimensions, and documented one-off dimensions as first-class
  explicit review slots rather than fallback behavior.
- Simplified reviewer guidance so reviewer agents follow the controller's
  instruction handoff: run the catalog instruction command when requested, or
  follow direct slot instructions for one-off dimensions.
- Reopened revision 3 after stale publish sync evidence and merged the latest
  `origin/main` so the final candidate can be republished from a fresh base.

### Not Delivered

- Instruction file hashing or frozen instruction snapshots in review round
  artifacts.
- Additional user-level, plugin-level, or organization-level dimension sources.
- A richer dimension authoring command.
- Migrating built-in dimension metadata and instruction bodies from Go literals
  into bundled Markdown/resource files. That storage refactor is intentionally
  left as follow-up work.

### Follow-Up Issues

- No GitHub follow-up issues were opened in this slice. Deferred items remain
  intentionally out of scope until there is concrete demand for instruction
  snapshots, additional catalog sources, an authoring command, or moving
  built-in dimensions into bundled Markdown resources.
