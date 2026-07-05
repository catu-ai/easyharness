---
template_version: 0.2.0
created_at: "2026-07-05T12:31:03+08:00"
approved_at: "2026-07-05T12:38:10+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/271
    - https://github.com/catu-ai/easyharness/issues/262
    - https://github.com/catu-ai/easyharness/pull/279
size: S
---

# Define Goal-Oriented Checkpoint Reports

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Define the checkpoint report convention for `workflow_profile:
goal_oriented` so adaptive work has a durable intermediate record between
Codex goal turns and harness steps.

The finished contract should make checkpoint reports structured enough for
future shallow lint/status consumption while preserving enough writing room for
complex exploration. A checkpoint report should be a structured narrative
digest, not a fixed bullet-only form, raw transcript, standalone evidence
report, or formal review gate.

## Scope

### In Scope

- Define what a checkpoint report is and is not.
- Define the checkpoint report relationship to a harness step, Codex goal turn,
  local checkpoint draft, step `Execution Notes`, step `Review Notes`,
  advisory challenge, final synthesis, evidence, and archive summaries.
- Define the durable storage convention: inline plan-body checkpoint reports as
  the default reading and review entrypoint, with supplements only for approved
  curated support material that would bloat the plan body.
- Define stable shallow structure for future tooling, including a checkpoint
  section anchor, stable checkpoint IDs, and required labels.
- Explicitly allow checkpoint label content to use prose, bullets, tables, or
  short lists as appropriate instead of requiring bullet-only formatting.
- Define required and optional checkpoint report fields.
- Define when agents should write a checkpoint report and when they should
  consider optional challenge.
- Keep #271 bounded against the neighboring v0.6.0 issues: #270, #272, #273,
  #274, and #275.

### Out of Scope

- Implementing CLI status, next-action, lint, template, archive, or reopen
  behavior.
- Adding automatic checkpoint generation.
- Adding UI timeline rendering or dashboard behavior.
- Adding a new hypothesis state machine.
- Changing review aggregation semantics or creating a formal review gate for
  every checkpoint.
- Adding the goal-oriented plan template or user-facing example plan.
- Requiring every raw experiment log, failed attempt, transcript, or command
  trace to enter git.

## Acceptance Criteria

- [x] The goal-oriented spec defines checkpoint reports as structured narrative
  digests for meaningful adaptive work boundaries, not raw logs, transcripts,
  evidence reports by default, or formal review gates.
- [x] The spec distinguishes checkpoint reports from harness steps, Codex goal
  turns, local checkpoint drafts, `Execution Notes`, `Review Notes`, challenge
  rounds, final synthesis, and archive summaries.
- [x] The durable storage convention says inline plan-body checkpoint reports
  are the canonical entrypoint, while supplements are allowed only for approved
  curated support artifacts that are indexed from the inline report.
- [x] The checkpoint structure uses stable anchors, IDs, and labels that future
  tooling may consume shallowly without requiring bullet-only content or
  judging hypothesis/evidence quality.
- [x] Required checkpoint labels include trigger, hypotheses or candidate
  directions, probe or experiment, observed result, scorecard movement,
  decision or next mutation, residual uncertainty, and evidence pointers.
- [x] Optional fields cover challenge notes, rejected alternatives, human
  decision needs, follow-up/deferred items, curated supplement paths, and
  validation or reproduction notes.
- [x] The spec describes checkpoint triggers such as plateau, meaningful result,
  hypothesis pivot, challenge request, before synthesis, or human scope input.
- [x] The plan-schema or adjacent normative prose is updated only as needed to
  make the convention discoverable without absorbing #270, #273, #274, or #275
  implementation work.
- [x] The slice leaves lint/status/template/docs/examples work explicitly
  deferred to the existing follow-up issues.

## Deferred Items

- #270 owns goal-oriented template and workflow guidance, including the
  generated authoring shape.
- #272 owns evidence-validity review and hypothesis-challenge guidance.
- #273 owns status and next-action behavior for goal-oriented plans.
- #274 owns lint enforcement for goal-oriented structure.
- #275 owns user-facing docs, help text, and examples.
- #276 owns any UI checkpoint timeline support.

## Work Breakdown

### Step 1: Define the checkpoint report contract

- Done: [x]

#### Objective

Update the goal-oriented workflow spec with a richer checkpoint report
contract that can stand alone from discovery chat.

#### Details

The contract should preserve the core #269 decisions while sharpening #271's
artifact shape:

- a checkpoint report is a concise durable digest of a meaningful checkpoint
  round inside an approved goal-oriented step;
- it records decision movement, not every action;
- it is distinct from local checkpoint drafts, which remain disposable runtime
  working memory unless promoted;
- it is distinct from step `Execution Notes`, which summarize completed step
  work after the fact;
- it is distinct from step `Review Notes`, which record formal review closeout
  or why step review was not needed;
- it can summarize optional challenge input, but challenge remains advisory
  unless an approved plan makes it a gate;
- it feeds final synthesis and archive summaries, but should not duplicate
  them once the decision trail has been absorbed.

Specify that checkpoint reports should use stable markdown anchors and labels,
but the content under each label may be prose, bullets, tables, or short lists.
The intended phrase is "structured narrative digest": structured enough for
future shallow tooling, expressive enough for real adaptive exploration.

Required labels should cover:

- `Trigger`
- `Hypotheses` or `Candidate Directions`
- `Probe` or `Experiment`
- `Observed Result`
- `Scorecard Movement`
- `Decision / Next Mutation`
- `Residuals`
- `Evidence`

Optional labels should cover:

- `Challenge`
- `Rejected Alternatives`
- `Human Decision Needed`
- `Follow-Up / Deferred`
- `Supplement`
- `Validation / Reproduction`

#### Expected Files

- `docs/specs/goal-oriented-workflow.md`

#### Validation

- Read the updated spec as a cold future agent and confirm it explains the
  checkpoint report contract without relying on issue or chat history.
- Targeted text checks should find the structured narrative digest wording,
  required labels, optional labels, and relationship boundaries.

#### Execution Notes

Updated `docs/specs/goal-oriented-workflow.md` to define tracked checkpoint
reports as concise, git-tracked structured narrative digests. The spec now
uses checkpoint report terminology consistently, names required and optional
labels, permits prose/bullets/tables/lists inside label content, and clarifies
that future tooling may consume headings, IDs, and labels shallowly without
judging hypothesis or evidence quality. TDD was not applicable because this
step changed normative documentation only.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 connects the same checkpoint-report convention
into the plan schema, so review is more useful after both documentation
surfaces are aligned.

### Step 2: Connect the convention to plan schema boundaries

- Done: [x]

#### Objective

Update the plan schema or adjacent normative prose only enough to make the
checkpoint report convention discoverable from the tracked plan contract.

#### Details

The plan schema should remain the general markdown-led plan package contract.
Do not turn this slice into the full goal-oriented template, lint, or status
implementation.

The connective wording should clarify:

- inline plan-body checkpoint reports are the default entrypoint for future
  resume, review, and archive;
- checkpoint reports may live under a dedicated step-local section such as
  `#### Checkpoint Reports`, with stable checkpoint IDs such as `CP1` or
  `S2-CP1`;
- supplements are acceptable only for approved curated support material, and
  the inline checkpoint report must carry the actual conclusion plus an index
  to that support;
- future lint/status may consume headings, IDs, and labels shallowly, but must
  not require bullet-only content or evaluate hypothesis/evidence quality.

If another spec needs a narrow cross-reference to avoid ambiguity, keep it as
a pointer. Leave concrete generated templates, lint fixtures, status outputs,
help text, and examples to the sibling issues.

#### Expected Files

- `docs/specs/plan-schema.md`
- `docs/specs/goal-oriented-workflow.md`
- `docs/specs/cli-contract.md` only if a narrow cross-reference is needed
- `docs/specs/state-model.md` only if a narrow cross-reference is needed

#### Validation

- Targeted searches for `Checkpoint Reports`, `structured narrative digest`,
  `goal_oriented`, `supplements`, `lint`, and `status` should show no conflict
  with the existing #269 support boundary.
- Confirm #270, #273, #274, and #275 implementation behavior remains deferred.

#### Execution Notes

Updated `docs/specs/plan-schema.md` so the reserved goal-oriented profile
points to tracked checkpoint reports as the durable intermediate plan-body
record, normally under `#### Checkpoint Reports` with stable IDs such as `CP1`
or `S2-CP1`. Also updated the spec index and state model to use the same
checkpoint report terminology. Kept CLI/status/lint/template behavior
deferred; TDD was not applicable because this step changed normative
documentation only.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 performs integrated validation for the complete
documentation contract before finalize review.

### Step 3: Validate issue boundaries and archive readiness

- Done: [x]

#### Objective

Confirm the documentation-only checkpoint report slice is internally
consistent, lintable, and cleanly bounded against the rest of v0.6.0.

#### Details

Validate that #271 does not accidentally absorb neighboring implementation
work. The final plan and changed specs should say enough for a future
goal-oriented executor to know:

- when a checkpoint report should be written;
- how many small attempts should be summarized inside one report instead of
  split into tiny checkpoints;
- when a pivot, plateau, meaningful result, challenge request, or pre-synthesis
  boundary deserves a new checkpoint;
- when bulky evidence belongs in supplements or another approved artifact;
- what future tooling may inspect structurally and what remains human/reviewer
  judgment.

#### Expected Files

- `docs/plans/active/2026-07-05-define-goal-oriented-checkpoint-reports.md`
- Any docs changed in Steps 1 and 2

#### Validation

- Run `harness plan lint docs/plans/active/2026-07-05-define-goal-oriented-checkpoint-reports.md`.
- Run `git diff --check`.
- Run targeted `rg` checks for the accepted vocabulary, labels, storage
  convention, issue boundaries, and non-goals.
- Run broader docs or repository validation only if the execution changes
  warrant it; otherwise record why targeted docs validation is sufficient.

#### Execution Notes

Validated the documentation slice with `harness plan lint`, `git diff
--check`, targeted `rg` checks for checkpoint-report terminology and issue
boundaries, and `go test` for all packages except environment-blocked
`internal/ui` and `tests/release`. Attempted `scripts/validate`, but pnpm
blocked the embedded UI build because `esbuild` build scripts require local
approval; the generated approval stub was removed. `go test ./internal/ui`
fails with the same unbuilt-UI-asset symptom, and `go test ./tests/release`
fails because `corepack` is unavailable in this environment. TDD was not
applicable because this step performed validation and plan closeout only.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Finalize review will cover the integrated documentation
candidate.

## Validation Strategy

- Lint the active plan before approval.
- During execution, validate documentation consistency with targeted searches
  for checkpoint reports, structured narrative digest, required labels,
  optional labels, supplements, challenge, final synthesis, status, and lint.
- Run `git diff --check` before archive.
- Run broader repository validation only if the documentation changes affect
  generated assets or behavior; otherwise record that this is a documentation
  contract slice and why targeted validation is enough.

## Risks

- Risk: The convention becomes a rigid form that cannot handle complex
  exploration.
  - Mitigation: Require stable anchors and labels, but explicitly allow prose,
    bullets, tables, and short lists inside label content.
- Risk: The convention is too loose for future lint/status support.
  - Mitigation: Define shallow structural anchors, checkpoint IDs, and labels
    that tooling can detect without judging natural-language quality.
- Risk: Checkpoint reports turn into raw process logs.
  - Mitigation: Define reports as decision-movement digests and leave raw logs
    local, regenerable, external, or summarized into approved artifacts.
- Risk: #271 absorbs neighboring v0.6.0 work.
  - Mitigation: Keep template, challenge/review guidance, status, lint, docs,
    examples, and UI behavior explicitly deferred to their existing issues.

## Validation Summary

- `harness plan lint docs/plans/active/2026-07-05-define-goal-oriented-checkpoint-reports.md`
  passed after plan creation, step closeout, review repair, and archive
  closeout.
- `git diff --check` passed.
- Targeted `rg` checks passed for checkpoint report terminology, stable
  required labels, optional labels, storage convention, issue boundaries, and
  absence of stale `checkpoint digest` wording.
- `go test $(go list ./... | grep -v '/internal/ui$' | grep -v '/tests/release$')`
  passed.
- `scripts/validate` was attempted with bundled Node and pnpm on `PATH`, but
  pnpm blocked the embedded UI build because `esbuild` build scripts require
  local approval in this environment.
- `go test ./internal/ui` reproduced the same unbuilt embedded-UI-asset
  symptom with `TestNewHandlerFallsBackToIndexForSPAPath` returning 500
  instead of 200.
- `go test ./tests/release` was blocked because `corepack` is unavailable in
  this environment.

## Review Summary

- Finalize review `review-001-full` requested changes across docs-consistency,
  agent-ux, and risk-scan. The blocking findings were stale follow-up-boundary
  wording that still described #271 as future digest work, plus ambiguous
  required-label aliases.
- Repaired the findings by making required checkpoint-report label names stable
  except for the explicitly listed pairs, and by stating that this contract now
  owns the #271 checkpoint report convention while leaving #270, #272, #273,
  #274, and #275 as follow-up implementation/documentation slices.
- Repair review `review-002-delta` passed across docs-consistency, agent-ux,
  and risk-scan with no findings.
- Finalize review `review-003-full` passed across docs-consistency, agent-ux,
  risk-scan, and tests with no blocking or non-blocking findings.

## Archive Summary

- Archived At: 2026-07-05T12:56:27+08:00
- Revision: 1
- PR: NONE
- Ready: Yes. Acceptance criteria are checked, tracked steps are complete,
  targeted validation passed, the only broader validation gaps are documented
  environment blockers, repair review passed, and final full review passed.
- Merge Handoff: After archive, commit the archive move, push
  `codex/define-goal-oriented-checkpoint-reports`, open a PR for #271, record
  publish/CI/sync evidence through harness, and wait for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Defined tracked checkpoint reports in `docs/specs/goal-oriented-workflow.md`
  as concise, git-tracked structured narrative digests for meaningful adaptive
  work boundaries.
- Replaced stale checkpoint digest terminology in the touched normative specs
  with checkpoint report terminology.
- Defined what checkpoint reports are not: raw logs, transcripts, every-command
  lists, every-small-attempt records, bullet-only forms, formal review gates,
  or evidence reports by default.
- Defined the durable storage convention: inline plan-body checkpoint reports
  are the canonical review/resume entrypoint, with supplements only for
  approved curated support artifacts indexed from the inline report.
- Defined stable shallow structure for future tooling: checkpoint report
  headings, stable IDs such as `CP1` or `S2-CP1`, and required labels.
- Clarified that label content may use prose, bullets, tables, or short lists,
  and that future tooling must not require bullet-only content or judge
  hypothesis/evidence quality.
- Connected the convention into `docs/specs/plan-schema.md`,
  `docs/specs/state-model.md`, and `docs/specs/index.md` without implementing
  template, status, lint, archive, reopen, or UI behavior.

### Not Delivered

- No CLI status, next-action, lint, template, archive, or reopen behavior was
  implemented.
- No automatic checkpoint generation was added.
- No UI timeline or dashboard behavior was added.
- No new hypothesis state machine or review aggregation behavior was added.
- No user-facing help text or example plan was added.

### Follow-Up Issues

- #270: Add the goal-oriented plan template and workflow guidance.
- #272: Add evidence-validity and hypothesis-challenge guidance.
- #273: Teach next actions for goal-oriented plans.
- #274: Add lint coverage for goal-oriented plans.
- #275: Add docs and examples for goal-oriented workflow.
- #276: Add UI support for goal-oriented checkpoint timelines.
