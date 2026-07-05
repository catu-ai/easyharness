---
template_version: 0.2.0
created_at: "2026-07-05T18:15:56+08:00"
approved_at: "2026-07-05T18:26:33+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/270
    - https://github.com/catu-ai/easyharness/pull/284
size: S
---

# Add Goal-Oriented Plan Authoring Preview

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add a first-class authoring preview for `workflow_profile: goal_oriented` so
controllers can draft credible goal-oriented tracked plans from repository
guidance alone, without implying the full v0.6.0 execution surface has shipped.

The finished slice should make `goal_oriented` recognizable as a defined
preview workflow profile for v0.6.0 authoring. It should seed or document a
stable plan shape with objective, scorecard, hypotheses, checkpoint cadence,
tracked checkpoint reports, evidence requirements, stopping conditions, and
final synthesis, while leaving full execution support, challenge/review
guidance, status next actions, structural lint coverage, and public examples to
the sibling v0.6.0 issues.

## Scope

### In Scope

- Add goal-oriented authoring guidance and, if the existing CLI shape supports
  it cleanly, a dedicated `harness plan template` variant for the preview
  profile.
- Show the stable goal-oriented step shape: setup, adaptive exploration,
  synthesis, and closeout.
- Make the generated or documented shape include objective, success scorecard,
  hypotheses or candidate directions, checkpoint cadence, checkpoint report
  anchors, evidence requirements, challenge triggers, stopping conditions, and
  final synthesis.
- Keep checkpoint reports under goal-oriented steps rather than turning each
  checkpoint into a separate harness workflow step.
- Clearly label `workflow_profile: goal_oriented` as a recognized preview
  workflow profile defined for v0.6.0 authoring while full execution support is
  still being completed.
- Add only the narrow CLI or lint recognition needed to avoid misleading users
  about the authoring preview, without adding full goal-oriented lint
  enforcement.
- Update bootstrap-managed agent guidance from `assets/bootstrap/` if the
  harness-managed skills or managed `AGENTS.md` block need new authoring cues,
  then sync materialized assets.
- Add focused tests or validation for the authoring/template behavior and the
  chosen preview boundary.

### Out of Scope

- Implementing a hypothesis state machine or separate workflow engine.
- Changing default `standard` or `lightweight` behavior.
- Adding full goal-oriented execution guidance, status next actions, archive
  and reopen support, challenge/review protocol, or UI rendering.
- Adding broad structural lint coverage for goal-oriented plans; #274 owns
  full lint enforcement.
- Adding user-facing docs, examples, or tutorials beyond the repo guidance
  needed for this authoring preview; #275 owns broader docs/examples.
- Changing review aggregation semantics.

## Acceptance Criteria

- [ ] A controller can draft a credible goal-oriented tracked plan from the
      template and repository guidance without relying on hidden issue or chat
      context.
- [ ] The authoring shape includes setup, adaptive exploration, synthesis, and
      closeout phases and names where tracked checkpoint reports live.
- [ ] Goal-oriented plans require objective, success scorecard, hypotheses or
      candidate directions, checkpoint cadence, evidence requirements,
      challenge triggers, stopping conditions, and final synthesis.
- [ ] The guidance says checkpoint reports live inside goal-oriented steps and
      do not automatically become separate harness workflow steps.
- [ ] The CLI, docs, or both describe `workflow_profile: goal_oriented` as a
      recognized preview workflow profile defined for v0.6.0 authoring, with
      full execution support still being completed.
- [ ] Existing standard and lightweight template behavior remains unchanged
      unless the caller explicitly requests the goal-oriented preview.
- [ ] Any lint or CLI guard added in this slice is explicitly narrow preview
      recognition, not full #274 structural lint enforcement.
- [ ] Deferred sibling issues #272, #273, #274, and #275 remain named as the
      owners for challenge/review guidance, next actions, lint coverage, and
      docs/examples.

## Deferred Items

- #272 owns evidence-validity review and hypothesis-challenge guidance.
- #273 owns status and next-action behavior for goal-oriented plans.
- #274 owns full lint coverage for goal-oriented structure.
- #275 owns user-facing docs, help text, and examples.
- #276 owns any UI checkpoint timeline support.

## Work Breakdown

### Step 1: Align the authoring contract

- Done: [x]

#### Objective

Decide the narrow implementation shape for the goal-oriented authoring preview
and align the normative wording before touching behavior.

#### Details

Read the current goal-oriented workflow spec, plan schema, state model, plan
template code, template CLI help, lint path/profile checks, and the archived
#271 checkpoint-report plan together.

Use that context to choose the smallest durable change that makes
`goal_oriented` authorable without pretending it is fully deliverable. The
likely target is a preview authoring template plus narrow profile recognition.
If inspection shows that making generated goal-oriented plans lint-valid would
absorb #274 or imply full archive/reopen support, stop and keep the generated
shape documented as preview-only instead.

#### Expected Files

- `docs/specs/goal-oriented-workflow.md`
- `docs/specs/plan-schema.md`
- `internal/plan/template.go`
- `internal/plan/lint.go`
- `internal/cli/app.go`
- `assets/templates/plan-template.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`

#### Validation

- The selected implementation path is written into this plan's execution notes
  with an explicit boundary for #272, #273, #274, and #275.
- Targeted searches show no wording that claims full goal-oriented execution,
  status, archive, reopen, challenge/review, or lint support has shipped.

#### Execution Notes

Selected a narrow #270 implementation path: add `harness plan template
--goal-oriented` as a recognized preview authoring variant, seed
`workflow_profile: goal_oriented` plus the required goal-oriented plan
concepts, and add shallow active-plan lint recognition so generated preview
plans can lint from a valid active-plan path. Full structural lint coverage,
status next actions, challenge/review guidance, archive/reopen profile
preservation, docs/examples, and UI behavior remain deferred to #272, #273,
#274, #275, and #276. Targeted searches found no remaining stale wording that
describes `goal_oriented` as not lint-valid or not yet templated.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step recorded the contract decision and boundary;
the behavior and documentation changes are implemented and reviewed under Step
2 and final validation.

### Step 2: Implement the preview authoring shape

- Done: [ ]

#### Objective

Add the goal-oriented preview template or equivalent authoring guidance, plus
focused recognition tests, while preserving existing standard and lightweight
behavior.

#### Details

The goal-oriented shape should seed stable phase boundaries rather than a
linear list of every future probe:

- setup and scorecard framing;
- adaptive exploration with checkpoint reports under the step body;
- synthesis of accepted conclusions, rejected hypotheses, residuals, evidence,
  and follow-up;
- closeout/validation/archive readiness.

The preview should include anchors or headings that future lint/status work can
consume shallowly, but this slice must not implement full structural linting.
If the implementation touches bootstrap-managed skills or the managed root
`AGENTS.md` block, edit `assets/bootstrap/` first and run
`scripts/sync-bootstrap-assets`.

#### Expected Files

- `internal/plan/template.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `assets/templates/plan-template.md`
- `docs/specs/goal-oriented-workflow.md`
- `docs/specs/plan-schema.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-plan/SKILL.md`

#### Validation

- Add or update focused tests proving standard and lightweight templates keep
  their existing shape.
- Add focused tests for any new goal-oriented template flag, rendering path, or
  narrow lint/profile guard.
- Run `scripts/sync-bootstrap-assets` if bootstrap assets change.

#### Execution Notes

Implemented the goal-oriented authoring preview. Added
`WorkflowProfileGoalOriented`, `harness plan template --goal-oriented`, a
three-step preview template shape, a `#### Checkpoint Reports` step subsection
anchor, and shallow active-plan lint recognition for
`workflow_profile: goal_oriented`. Archived goal-oriented plans still fail lint
with preview-boundary wording because archive/reopen profile preservation is
out of scope. Updated the goal-oriented workflow, plan schema, state model,
CLI contract, spec index, and harness-plan skill guidance from
`assets/bootstrap/`, then synced materialized `.agents/skills/` output.
Validation so far: `go test ./internal/plan ./internal/cli`, generated
template text check for preview/checkpoint anchors, and generated active-plan
lint smoke test all passed.

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Validate and prepare for review

- Done: [ ]

#### Objective

Confirm the #270 slice is internally consistent, tested, and PR-ready without
expanding into the rest of the v0.6.0 goal-oriented capability.

#### Details

Validate that the candidate can generate or document a credible goal-oriented
plan, that it uses checkpoint report terminology from #271/PR #284, and that
it clearly describes the preview boundary. Check that no default standard or
lightweight behavior changed.

#### Expected Files

- `docs/plans/active/2026-07-05-add-goal-oriented-plan-authoring-preview.md`
- Any files changed in Steps 1 and 2

#### Validation

- Run `harness plan lint docs/plans/active/2026-07-05-add-goal-oriented-plan-authoring-preview.md`.
- Run focused Go tests for plan template and CLI behavior.
- Run `git diff --check`.
- Run targeted `rg` checks for `goal_oriented`, `recognized preview workflow
  profile`, `checkpoint reports`, `full execution support`, and sibling issue
  boundaries.
- Run broader validation only if the final change touches shared behavior
  beyond the template/profile-recognition surface; record any environment
  blockers exactly.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Lint this active plan before approval and again before archive.
- Use focused unit tests for CLI/template/profile behavior because this slice
  is primarily an authoring/template change.
- Use targeted text checks to confirm the preview boundary and sibling issue
  ownership remain explicit.
- Run `git diff --check`.
- If bootstrap assets change, run `scripts/sync-bootstrap-assets` and inspect
  the materialized diff.
- Run broader Go or repository validation if implementation affects shared
  plan loading, lint, status, archive, or reopen behavior; otherwise keep
  validation focused and explain why.

## Risks

- Risk: The template makes users think the full goal-oriented workflow has
  shipped.
  - Mitigation: Label the profile as a recognized preview workflow profile and
    explicitly state that full execution support is still being completed.
- Risk: A narrow lint or CLI change accidentally absorbs #274.
  - Mitigation: Limit this slice to profile/template recognition and avoid
    structural quality checks for scorecards, hypotheses, checkpoints, or
    synthesis.
- Risk: The new authoring path regresses standard or lightweight defaults.
  - Mitigation: Add focused regression tests for the existing template modes.
- Risk: The plan shape encourages one harness step per checkpoint.
  - Mitigation: Seed checkpoint reports inside adaptive steps and repeat that
    checkpoint reports do not automatically become separate workflow steps.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
