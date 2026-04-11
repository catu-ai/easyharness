---
template_version: 0.2.0
created_at: "2026-04-10T23:19:20+08:00"
source_type: direct_request
source_refs: []
size: L
---

# Add plan size frontmatter and relax lightweight eligibility

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add a required `size` field to tracked plan frontmatter so every plan carries a
durable t-shirt estimate that future agents and humans can read without
recovering intent from chat history. The first version should keep the size
model simple, repository-backed, and backfilled across the tracked plan
history that already lives in git.

Relax `lightweight` eligibility so the profile is no longer limited to docs or
copy-only work. `size` must still describe work magnitude rather than risk,
but `XXS` plans should become eligible to use `lightweight` when the slice is
truly tiny and bounded. At the same time, very large work should be called out
explicitly so `XXL` becomes a signal to split scope or defer follow-up issues
rather than a routine planning default.

## Scope

### In Scope

- Define a required top-level `size` frontmatter field for tracked plans.
- Define the supported size ladder for the initial rollout:
  `XXS`, `XS`, `S`, `M`, `L`, `XL`, `XXL`.
- Document a durable meaning for each size so future agents can size new plans
  and backfill historical plans consistently.
- Update `lightweight` guidance so `XXS` plans may use the lightweight profile
  without making size alone the only risk gate.
- Add guidance that `XXL` work is discouraged and should normally justify why
  it was not split further, with explicit human confirmation at plan creation
  time and deferred follow-up captured explicitly when appropriate.
- Update plan template, parsing, linting, and related tests so the new field is
  first-class in the CLI contract.
- Backfill `size` across tracked plans in `docs/plans/`, including the current
  active plan created for this slice.
- Update the harness-managed planning guidance in bootstrap assets and sync the
  generated `.agents/` and managed `AGENTS.md` materialized output.

### Out of Scope

- Changing workflow choice to be determined by `size` alone.
- Introducing `XXXL` or a larger ladder unless implementation uncovers a
  repository-backed need that contradicts current discovery evidence.
- Automatically splitting existing large historical plans into new plan files.
- Adding a new archive format or replacing `workflow_profile: lightweight`
  with a second workflow object.
- Backfilling disposable local archives under `.local/`.

## Acceptance Criteria

- [x] `docs/specs/plan-schema.md` defines required frontmatter `size`, accepts
      exactly `XXS`, `XS`, `S`, `M`, `L`, `XL`, and `XXL`, and explains the
      meaning of each size clearly enough for future agents to classify plans
      without chat-only context.
- [x] `harness plan template` and `harness plan lint` treat `size` as a
      first-class field, and focused automated tests cover valid, invalid, and
      omitted-size cases.
- [x] The `lightweight` eligibility docs and planning skill guidance explicitly
      allow `XXS` plans to use `lightweight` while preserving the rule that
      small size does not automatically make a change low-risk.
- [x] Planning guidance documents `XXL` as discouraged by default and explains
      that large plans should justify why they were not split further, require
      explicit human confirmation when first sized as `XXL`, and move
      obviously deferrable scope into `Deferred Items` or follow-up issues.
- [x] Every tracked plan under `docs/plans/active/` and `docs/plans/archived/`
      in this repository carries an explicitly chosen `size` value, and the
      resulting set passes `harness plan lint`.

## Deferred Items

- Revisit whether `lightweight` remains a distinct field once the repository
  has real history using `size` plus the relaxed `XXS` eligibility rule.
- Consider whether a future planner-facing command should suggest a candidate
  `size` automatically from plan structure once the backfilled corpus exists.

## Work Breakdown

### Step 1: Define the size and workflow contract

- Done: [x]

#### Objective

Write the durable repository contract for plan size, the initial size ladder,
the relaxed `XXS` lightweight rule, and the discouraged-by-default `XXL`
guidance.

#### Details

This step should encode the discovery decisions explicitly:
- `size` measures work magnitude, not risk by itself
- the supported ladder is `XXS` through `XXL`
- `XXS` plans may use `lightweight`, but small size does not waive risk
  judgment
- `XXL` is allowed but should trigger an explicit explanation and serious
  consideration of deferred follow-up scope
- if a plan is sized `XXL` at creation time, the planning workflow should stop
  and confirm with the human whether the slice should be split, potentially
  returning to discovery to decide how to split it
- planning guidance should include practical sizing heuristics, not only enum
  names

Because the repository dogsfoods bootstrap assets, update the bootstrap source
files rather than hand-editing generated `.agents/skills/` guidance or the
managed `AGENTS.md` block directly. Keep the normative rules in specs and the
practical sizing heuristics in the planning skill/docs aligned.

#### Expected Files

- `docs/specs/plan-schema.md`
- `docs/specs/cli-contract.md`
- `README.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`
- `assets/bootstrap/agents-managed-block.md`
- `AGENTS.md`
- `.agents/skills/harness-plan/SKILL.md`

#### Validation

- The specs define the enum, meanings, `XXS` lightweight eligibility, and
  `XXL` guidance without relying on discovery chat.
- The planning skill tells a future agent how to choose a size in practice.
- The planning workflow makes it explicit that an initial `XXL` estimate
  requires human confirmation and may send the work back through discovery to
  settle a better split.
- `scripts/sync-bootstrap-assets` refreshes the generated skill and managed
  `AGENTS.md` output cleanly after the bootstrap source changes.

#### Execution Notes

Defined `size` as a required t-shirt field in `docs/specs/plan-schema.md`,
added the initial `XXS` through `XXL` ladder plus adjacent-size heuristics,
and updated the lightweight rules so only `XXS` plans may use
`workflow_profile: lightweight` while keeping explicit low-risk judgment.
Documented that an initial `XXL` estimate is a planning warning that requires
human confirmation and may send the work back through discovery to decide how
to split it. Propagated the same guidance into `docs/specs/cli-contract.md`,
`README.md`, `assets/bootstrap/skills/harness-plan/SKILL.md`, and
`assets/bootstrap/agents-managed-block.md`, then ran
`scripts/sync-bootstrap-assets` so `.agents/skills/harness-plan/SKILL.md` and
the managed block in `AGENTS.md` stayed in sync with the bootstrap sources.
Finalize review `review-002-full` later surfaced one remaining planner-facing
drift: the schema already said `XXL` work should push obvious spillover into
`Deferred Items` or follow-up issues, but the README and planning guidance
only mentioned confirming or rediscovering the split. Updated the README,
bootstrap guidance, and synced managed outputs so that `XXL` plans now carry
that deferred-scope handoff explicitly wherever planners are likely to look.
Finalize review `review-004-full` then caught one last operator-facing docs
drift: the schema and planning skill already required explicit human approval
for `workflow_profile: lightweight`, but the README and managed `AGENTS.md`
block only implied approval through `XXS` plus low-risk wording. Added the
explicit human-approval gate to the README and bootstrap-managed workflow
guidance, then reran `scripts/sync-bootstrap-assets` so `AGENTS.md` and
`.agents/skills/harness-plan/SKILL.md` matched the durable contract again.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 establishes the contract that Step 2 implements,
so review is more meaningful once the template/lint/CLI behavior lands against
the updated rules.

### Step 2: Implement required size support in plan tooling

- Done: [x]

#### Objective

Teach the plan template, parser, linting, and related tests to require and
validate `size` as part of the tracked-plan schema.

#### Details

The implementation should make `size` a normal frontmatter field rather than a
docs-only convention. Generated plans need an explicit way to carry `size`
from the start, historical files with missing `size` must fail lint after the
rollout, and invalid values must produce targeted lint errors. Keep the change
clean: do not add compatibility shims, dual paths, or a hidden default that
masks omitted sizing decisions.

If command help or template seeding needs an explicit size flag, choose the
clearest end-state directly and keep tests focused on authoring plus linting
behavior.

#### Expected Files

- `assets/templates/plan-template.md`
- `assets/templates/embed.go`
- `cmd/harness/main.go`
- `internal/plan/document.go`
- `internal/plan/document_test.go`
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`
- `internal/plan/template.go`
- `internal/plan/template_test.go`

#### Validation

- `harness plan template` emits plans with explicit `size` support and clear
  authoring guidance.
- `harness plan lint` rejects omitted or unsupported sizes with targeted
  errors.
- Focused Go tests cover template rendering, document parsing, and linting for
  the new field.

#### Execution Notes

Added first-class size support in the plan tooling. `assets/templates/plan-template.md`
now carries an explicit `size` frontmatter slot, `internal/plan/size.go`
defines the supported ladder, `internal/plan/lint.go` requires `size` and
rejects unsupported values, and `internal/plan/template.go` now keeps missing
size explicit instead of silently defaulting it while making lightweight
templates render with `size: XXS` and rejecting non-`XXS` lightweight input.
`internal/cli/app.go` now exposes `--size` for `harness plan template`, and
the impacted tests/fixtures/scripts were updated so valid plan fixtures seed a
real size deliberately. Focused validation used `go test ./internal/plan
./internal/cli`, plus broader impacted-package reruns across `internal/evidence`,
`internal/lifecycle`, `internal/review`, `internal/status`, `internal/ui`, and
`tests/e2e`. A `go test ./...` pass printed successful results for every
package but did not return cleanly in this environment, so the impacted suites
were rerun explicitly and passed. Finalize review `review-001-full` then asked
for stronger negative coverage around missing/unsupported size values and the
`XXS` lightweight rule, plus a durable regression check for the migrated
archived corpus. Added template, lint, and CLI negative tests for those
failure modes and a repository-backed archived-plan corpus lint test before
rerunning the focused and broader suites successfully. Finalize review
`review-002-full` then caught one remaining correctness gap: size validation
was still normalizing case and surrounding whitespace instead of enforcing the
documented enum exactly. Tightened the validator to accept only canonical
spellings and expanded the repository-backed regression from archived plans to
the full tracked active-plus-archived plan corpus before rerunning the focused
and broader suites successfully again. Finalize review `review-003-full` then
found one remaining tests gap: the lightweight path still was not asserting
the raw emitted `size: XXS`, because the plan template unit helper could patch
in the placeholder replacement after rendering and the CLI test only checked
workflow profile plus step shape. Reworked the lightweight template unit test
to assert the direct render output and extended the CLI lightweight test to
assert `size: XXS` explicitly, then reran `go test ./internal/plan
./internal/cli` and `go test ./tests/e2e` successfully before starting a
fresh finalize review.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 and Step 3 materially depend on each other, so a
full-candidate review is more meaningful than isolating tooling changes before
the tracked corpus is backfilled and lint-clean.

### Step 3: Backfill tracked plans and prove the repository state

- Done: [x]

#### Objective

Assign sizes to the tracked plan corpus, update repository-visible examples,
and prove the backfilled repository passes plan validation end to end.

#### Details

Use the agreed ladder and heuristics from Step 1 to backfill every tracked plan
under `docs/plans/`. Preserve the existing historical content and only add the
minimal frontmatter needed for `size` unless a nearby wording update is
required to keep guidance consistent. The current active plan for this slice
must also remain compliant with the new schema.

This step should also backfill any spec or example snippets that show plan
frontmatter so the docs do not keep teaching the old schema. If any historical
plan truly looks like `XXL`, keep it valid but leave the durable guidance in
place that such size should normally push future work toward splitting and
deferred follow-up. The stronger human-confirmation rule applies to newly
planned work; historical `XXL` plans only need accurate backfilled sizing plus
the updated durable guidance.

#### Expected Files

- `docs/plans/active/2026-04-10-add-plan-size-frontmatter-and-relax-lightweight-eligibility.md`
- `docs/plans/archived/*.md`
- `docs/specs/plan-schema.md`
- `README.md`

#### Validation

- Every tracked plan in `docs/plans/active/` and `docs/plans/archived/` has a
  valid `size`.
- Repository examples and schema snippets match the new frontmatter shape.
- `harness plan lint` passes on the touched tracked plans, and focused checks
  make it hard to miss an un-backfilled file.

#### Execution Notes

Backfilled `size` across every tracked archived plan under `docs/plans/archived/`
and kept those edits minimal to frontmatter. The current active plan already
carried `size: L`. During full-corpus validation, four early archived v0.1
plans surfaced pre-existing lint debt because they still carried legacy
runtime frontmatter fields and `- Status:` step markers. Migrated those four
plans to the current lint-compatible archived shape by removing the obsolete
runtime frontmatter keys and converting completed step markers to `- Done: [x]`
without changing the historical execution notes or summaries. After that
cleanup, a repository-wide loop over `docs/plans/active/*.md` and
`docs/plans/archived/*.md` completed with `COUNT=0` lint failures.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The historical backfill is easiest to assess together
with the schema, template, and lint changes in the final candidate review.

## Validation Strategy

- Use focused Go tests for template, parsing, and lint behavior while the field
  is introduced.
- Rerun bootstrap sync after source-asset edits so generated guidance stays in
  lockstep with the tracked sources.
- Run repository-level checks over the tracked plan set so missing backfills or
  stale examples are caught before approval or archive.

## Risks

- Risk: `size` becomes a vague label rather than a durable planning aid.
  - Mitigation: keep the ladder small, document concrete adjacent-size
    heuristics, and backfill the historical corpus consistently in the same
    slice.
- Risk: relaxing `lightweight` makes the profile too broad and encourages
  risky small changes to bypass the standard path.
  - Mitigation: document that `XXS` may use `lightweight`, but size is not the
    only gate; keep explicit low-risk judgment and escalation guidance.
- Risk: the repository ends up partially backfilled, leaving historical plans
  inconsistent with the new schema.
  - Mitigation: treat backfill as a required acceptance criterion and lint the
    tracked plan corpus before closeout.

## Validation Summary

- Ran `scripts/sync-bootstrap-assets` after the bootstrap guidance edits so
  `.agents/skills/harness-plan/SKILL.md` and the managed block in `AGENTS.md`
  stayed aligned with `assets/bootstrap/`.
- Ran focused tooling tests with `go test ./internal/plan ./internal/cli`
  after the size-contract implementation and again after the finalize repairs
  for exact enum enforcement and raw lightweight `size: XXS` assertions.
- Ran broader impacted validation with
  `go test ./internal/evidence ./internal/lifecycle ./internal/review ./internal/status ./internal/ui ./tests/e2e`.
- Confirmed the lightweight E2E path still passed after the final test repair
  with `go test ./tests/e2e`.
- Linted the active tracked plan with
  `harness plan lint docs/plans/active/2026-04-10-add-plan-size-frontmatter-and-relax-lightweight-eligibility.md`.
- Ran a repository-wide tracked-plan lint sweep over
  `docs/plans/active/*.md` and `docs/plans/archived/*.md`; after backfilling
  sizes and migrating four legacy archived plans away from obsolete runtime
  frontmatter and `- Status:` markers, the final sweep completed with
  `COUNT=0` failures.
- After PR `#138` CI failed on revision 1, reinstalled the repo-local
  `harness` binary with `scripts/install-dev-harness` so later lifecycle
  commands used the current Go CLI instead of a stale pre-size build.
- For revision 2 finalize-fix repair, ran
  `go test ./internal/lifecycle ./internal/plan` to cover the archive/reopen
  lifecycle path plus tracked-plan lint expectations after restoring size to
  the reopened candidate.
- Ran
  `go test ./tests/smoke -run TestPlanTemplateAndLintRoundTrip -count=1` to
  prove the smoke round-trip now passes with explicit `--size M`.

## Review Summary

- Finalize review `review-001-full` found a blocking tests gap around missing
  negative coverage for invalid size handling and the lack of a durable
  repository-backed regression for the migrated archive corpus.
- Finalize review `review-002-full` found two blocking gaps: size validation
  still normalized case and whitespace instead of enforcing the documented
  enum exactly, and planner-facing docs outside the schema had not yet
  reflected the `XXL` deferred-scope handoff guidance.
- Finalize review `review-003-full` found one blocking tests gap: the
  lightweight authoring path was not directly asserting the raw emitted
  `size: XXS` because helper code could patch size into the rendered output
  after the fact.
- Finalize review `review-004-full` found one blocking docs-consistency gap:
  the schema and planning skill required explicit human approval for
  `workflow_profile: lightweight`, but README and the managed `AGENTS.md`
  block still only implied that gate.
- Finalize review `review-005-full` passed cleanly with no blocking or
  non-blocking findings after the last tests and docs repairs.
- Post-archive PR `#138` CI then failed on two missed test surfaces: the
  archived revision-1 candidate had been created with a stale pre-size
  `harness` binary and lost `size` in frontmatter, and
  `tests/smoke/TestPlanTemplateAndLintRoundTrip` still authored a plan without
  an explicit `--size` flag.
- Delta finalize review `review-006-delta`, anchored at commit
  `86bc0f273483d533748464622f2d7a08a6f2104e`, passed cleanly with no blocking
  or non-blocking findings after the revision-2 reopen repair added the smoke
  size fix and archive size-preservation regression coverage.

## Archive Summary

- Archived At: 2026-04-11T00:30:51+08:00
- Revision: 2
- PR: `#138` (`https://github.com/catu-ai/easyharness/pull/138`)
- Ready: The archived revision-2 candidate repairs the failed revision-1 CI
  run by restoring the missing archived-plan size frontmatter, fixing the
  smoke authoring path to pass an explicit size, and carrying a clean delta
  finalize review (`review-006-delta`) plus focused local validation.
- Merge Handoff: Push the refreshed branch to PR `#138`, refresh
  publish/CI/sync evidence for the new head commit, and keep the candidate in
  publish handoff until `harness status` advances to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added required tracked-plan frontmatter `size` support with the initial
  ladder `XXS`, `XS`, `S`, `M`, `L`, `XL`, and `XXL`.
- Updated plan-schema, CLI-contract, README, bootstrap planning guidance, and
  managed agent guidance so size means work magnitude, `XXS` may use
  `lightweight` only with explicit human approval, and initial `XXL` sizing
  now prompts split confirmation plus deferred-scope thinking.
- Implemented first-class size handling in plan templating and linting, added
  `harness plan template --size`, and enforced the exact documented enum with
  targeted negative coverage.
- Made lightweight template generation emit `size: XXS` directly and covered
  that behavior in both template and CLI tests.
- Backfilled `size` across the tracked plan corpus and migrated four legacy
  archived plans off obsolete runtime frontmatter so the whole tracked plan set
  is lint-clean under the new schema.
- Reopened the archived candidate after CI failure, restored `size: L` to the
  revision-2 candidate, fixed the smoke template round-trip to pass an
  explicit size, and added lifecycle regression coverage that archive
  preserves size frontmatter.

### Not Delivered

- This slice did not remove `workflow_profile: lightweight` as a separate
  planning surface.
- This slice did not add automatic size suggestion or inference to
  `harness plan`.

### Follow-Up Issues

- #136 Revisit whether lightweight should remain a distinct workflow profile
  after size backfill (`https://github.com/catu-ai/easyharness/issues/136`)
- #137 Consider suggesting candidate plan sizes from plan structure and the
  backfilled corpus (`https://github.com/catu-ai/easyharness/issues/137`)
