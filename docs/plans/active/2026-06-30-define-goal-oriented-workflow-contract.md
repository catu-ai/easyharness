---
template_version: 0.2.0
created_at: "2026-06-30T22:20:58+08:00"
approved_at: "2026-06-30T22:23:25+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/269
    - https://github.com/catu-ai/easyharness/issues/262
size: M
---

# Define Goal-Oriented Workflow Contract

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Define the normative `workflow_profile: goal_oriented` contract before
template, lint, status, review guidance, UI, or example work depends on it.

The finished contract should describe how goal-oriented work keeps harness's
existing approval, step, review, archive, and evidence boundaries while giving
agents a disciplined way to run adaptive work through hypotheses, probes,
checkpoint rounds, optional challenge, and final synthesis.

## Scope

### In Scope

- Add a normative goal-oriented workflow spec under `docs/specs/`.
- Define when to use `workflow_profile: goal_oriented` and when to keep the
  ordinary standard workflow.
- Define the vocabulary and boundaries for stable plan steps, local checkpoint
  drafts, tracked checkpoint digests, optional challenge rounds, formal
  reviews, evidence, and final synthesis.
- Specify that checkpoint drafts may live under the local runtime root as
  disposable working memory, while tracked checkpoint digests are concise
  decision-quality summaries in the tracked plan package.
- Define that tracked checkpoint digests should be self-contained enough for
  resume and review, but must not become raw command logs or duplicate a final
  report.
- Define checkpoint cadence and status guidance at the contract level:
  goal-oriented plans should declare a checkpoint cadence or budget, and
  future status guidance may present a short list of general next actions such
  as continue another bounded checkpoint round, synthesize, request challenge,
  or close the step.
- Clarify that `harness status` may later do advisory structural checks for
  goal-oriented plans, but checkpoint markdown must not derive or mutate
  `current_node`.
- Clarify that `harness plan lint` remains the explicit validation gate; status
  may surface guidance but must not silently become full lint.
- Clarify that standalone evidence reports or decision reports are deliverables
  only when the user objective and approved plan scope call for them.
- Update the spec index and plan-schema prose so the new contract is
  discoverable and does not conflict with existing `workflow_profile`
  semantics.

### Out of Scope

- Implementing CLI behavior, `harness status` next-action logic, or
  `harness plan lint` enforcement.
- Adding the goal-oriented plan template or generated authoring shape.
- Defining the full checkpoint report storage convention beyond the tracked
  digest contract needed for this slice.
- Adding new review aggregation behavior or making challenge a formal review
  gate.
- Adding UI timeline support.
- Requiring every goal-oriented plan to produce a standalone evidence report.
- Requiring every goal-oriented plan to run at least one challenge round.
- Requiring every checkpoint draft, raw transcript, failed attempt, or command
  trace to enter git or archive.

## Acceptance Criteria

- [x] A normative spec defines `workflow_profile: goal_oriented` as a profile
  on top of the existing harness workflow, not a separate workflow engine.
- [x] The contract explains appropriate and inappropriate use cases for
  `goal_oriented`.
- [x] The contract distinguishes `step`, checkpoint draft, tracked checkpoint
  digest, `challenge`, formal `review`, evidence, and final synthesis.
- [x] The contract defines required plan concepts: objective, success
  scorecard, hypotheses or candidate directions, probe/checkpoint loop,
  checkpoint cadence, challenge triggers, evidence requirements, stopping
  conditions, and final synthesis.
- [x] The contract states that local checkpoint drafts are disposable working
  memory under the local runtime root unless their content is promoted into a
  tracked digest or approved deliverable.
- [x] The contract states that tracked checkpoint digests live in the tracked
  plan package, preferably in the plan body when concise, and record
  decision-quality summaries rather than raw logs.
- [x] The contract explains that status guidance for goal-oriented plans should
  be advisory and general, and must not derive `current_node` from checkpoint
  markdown.
- [x] The contract explains that lint remains an explicit validation surface,
  while status may later surface lightweight structural guidance.
- [x] The contract preserves existing human approval, formal review, archive,
  and evidence boundaries.
- [x] The contract explicitly rejects one harness step per model turn as the
  default shape.
- [x] Follow-up implementation surfaces are clearly left to the existing
  v0.6.0 issues: #270, #271, #272, #273, #274, and #275.

## Deferred Items

- #270 will add the goal-oriented plan template and workflow guidance.
- #271 will define richer checkpoint report conventions and storage details if
  more than tracked plan digests are needed.
- #272 will add evidence-validity and hypothesis-challenge guidance.
- #273 will implement status/next-action behavior for goal-oriented plans.
- #274 will add lint coverage for goal-oriented plans.
- #275 will add user-facing docs, help, and examples.

## Work Breakdown

### Step 1: Write the normative contract

- Done: [x]

#### Objective

Add a self-contained `docs/specs/goal-oriented-workflow.md` document that
defines the profile's semantics and boundaries.

#### Details

The spec should make the following settled discovery decisions durable:

- `goal_oriented` is a workflow profile layered on the existing harness
  workflow, not a new engine.
- Steps stay stable approved phase boundaries; checkpoint rounds live inside
  those steps and do not map one-to-one with model turns.
- Agents may keep messy checkpoint drafts under the local runtime root as
  disposable working memory.
- A tracked checkpoint digest is a concise, git-tracked summary of a meaningful
  checkpoint round. It records the hypotheses or directions considered,
  observed signal, scorecard movement, decision, residual uncertainty, and
  evidence pointers needed for future resume, review, and archive. It is not a
  raw log, not a formal review gate, and not the canonical evidence report
  unless the plan explicitly designates it as such.
- Tracked checkpoint digests should normally live in the plan body under a
  `Checkpoint Digests` section when concise enough to keep the plan readable.
  Supplements are acceptable only when the digest or related durable material
  would bloat the plan.
- Each goal-oriented plan should declare checkpoint cadence or budget instead
  of relying on a global hard minimum or maximum. Before an adaptive step
  closes, at least one tracked checkpoint digest or equivalent final synthesis
  must explain the adaptive work.
- Challenge is optional unless the approved plan declares a required challenge
  boundary. It should be triggered by agent judgment, human direction, or plan
  criteria such as high uncertainty, competing hypotheses, weak evidence, or a
  risky synthesis.
- Evidence/report artifacts are created according to the user objective and
  approved scope. The profile requires conclusions to be supported by evidence;
  it does not require every plan to create a standalone evidence report.
- Future status guidance may surface a general action list at checkpoint-round
  boundaries, such as continue another bounded checkpoint round, synthesize if
  the scorecard is decisive, request challenge if uncertainty remains, or close
  the step if synthesis and evidence are ready.
- Checkpoint digests may inform status guidance, but must not derive, mutate,
  or override `current_node`.
- Lint remains the explicit validation gate. Status may later surface advisory
  structural checks, but must not silently run or replace full plan lint.

#### Expected Files

- `docs/specs/goal-oriented-workflow.md`

#### Validation

- The new spec can be read without issue or chat history and answers the
  vocabulary, checkpoint, challenge, evidence, lint, status, and archive
  boundary questions above.
- No CLI behavior, tests, templates, or schema files are changed in this step
  unless needed only for documentation linkage.

#### Execution Notes

Added `docs/specs/goal-oriented-workflow.md` as the normative profile
contract. The spec defines use cases, profile semantics, required plan
concepts, stable steps versus checkpoint rounds, local checkpoint drafts,
tracked checkpoint digests, optional challenge, evidence/report ownership,
status guidance, lint boundaries, review/archive behavior, and follow-up issue
boundaries. This is documentation-only, so TDD was not applicable.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 will immediately connect the new contract into
the existing specs, so review is more useful after the normative document and
cross-references are both in place.

### Step 2: Connect the contract to existing specs

- Done: [x]

#### Objective

Update existing normative indexes and workflow-profile prose so the new
goal-oriented contract is discoverable and does not contradict the existing
`lightweight` profile contract.

#### Details

The plan-schema prose currently documents `workflow_profile` as supporting only
`lightweight`. Update that surface carefully so:

- omitted `workflow_profile` still means ordinary standard workflow;
- `lightweight` keeps its existing low-risk `XXS` meaning and archive behavior;
- `goal_oriented` is described as a standard tracked-plan profile for adaptive
  work, not a lightweight shortcut;
- active tracked plans may use `workflow_profile: goal_oriented` once the
  implementation slices land;
- archived tracked plans should preserve the durable goal-oriented digests and
  synthesis in the plan package, while local drafts remain disposable.

Avoid over-updating CLI, lint, or status contracts in this slice. If a document
needs a cross-reference to prevent ambiguity, keep it as a contract pointer and
leave behavior implementation to the follow-up issues.

#### Expected Files

- `docs/specs/index.md`
- `docs/specs/plan-schema.md`
- `docs/specs/cli-contract.md` only if a narrow cross-reference is needed to
  keep status/lint boundaries understandable

#### Validation

- `rg "workflow_profile|goal_oriented|lightweight"` shows no obvious conflict
  where docs say only `lightweight` can ever be supported.
- The updated docs still make clear that this issue does not implement CLI
  status, lint, template, or UI behavior.

#### Execution Notes

Updated `docs/specs/index.md`, `docs/specs/plan-schema.md`,
`docs/specs/state-model.md`, and `docs/specs/cli-contract.md` so the new
contract is discoverable and does not conflict with existing profile, archive,
status, lint, or `current_node` boundaries. The changes keep implementation
details for template, status, lint, and review guidance in the existing
follow-up issues. This is documentation-only, so TDD was not applicable.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 performs the integrated boundary and validation
pass for the complete contract/documentation slice before final review.

### Step 3: Validate issue boundaries and archive readiness

- Done: [x]

#### Objective

Confirm the plan's delivered contract is internally consistent, lintable, and
clearly scoped against the existing v0.6.0 follow-up issues.

#### Details

Check that #269 remains a contract-first slice. The final plan and spec should
not accidentally absorb #270-#275 implementation details. Any unavoidable
spillover or open question should be recorded as a deferred item tied to the
existing issue that owns it.

The validation should also confirm the spec says enough for a future
goal-oriented plan author to know:

- when to use the profile;
- how checkpoint drafts differ from tracked checkpoint digests;
- where challenge fits;
- what status may advise without becoming a state machine;
- when lint remains the explicit validation surface;
- when evidence/report artifacts are deliverables versus support material.

#### Expected Files

- `docs/plans/active/2026-06-30-define-goal-oriented-workflow-contract.md`
- Any files changed in Steps 1 and 2

#### Validation

- Run `harness plan lint docs/plans/active/2026-06-30-define-goal-oriented-workflow-contract.md`.
- Run targeted text checks for the settled vocabulary and follow-up issue
  boundaries.
- Run the repository's relevant documentation or test validation only if the
  touched docs have an existing automated check.

#### Execution Notes

Validated the contract-first boundary with targeted searches for
`goal_oriented`, `workflow_profile`, checkpoint, challenge, `current_node`,
status, lint, and #270-#275 references. Ran `harness plan lint`, `git diff
--check`, and `scripts/validate`; all passed. No separate docs-only checker was
found beyond the plan lint and repository validation script.

#### Review Notes

Review `review-001-delta` requested changes: the first draft made
`workflow_profile: goal_oriented` read like a currently lint-valid active-plan
value, and left archive/reopen profile identity handling implicit. Fixed by
framing `goal_oriented` as a reserved contract until the follow-up authoring,
lint, status, archive, and reopen slices land, and by requiring those slices to
define profile identity preservation explicitly. Follow-up review
`review-002-delta` passed with no blocking or non-blocking findings.

## Validation Strategy

- Lint the active plan before approval.
- During execution, validate documentation consistency with targeted searches
  for `goal_oriented`, `workflow_profile`, `checkpoint`, `challenge`,
  `current_node`, `status`, and `lint`.
- Run any existing docs/spec validation commands discovered during execution.
  If there is no dedicated docs check, record that explicitly in the validation
  summary.

## Risks

- Risk: The contract accidentally creates a second workflow engine.
  - Mitigation: State repeatedly that `goal_oriented` uses existing approval,
    step, formal review, archive, and evidence boundaries.
- Risk: Checkpoints become process logs that duplicate final evidence or report
  artifacts.
  - Mitigation: Define local drafts as disposable and tracked digests as
    concise decision-quality summaries; define evidence/report artifacts by
    user objective and approved scope.
- Risk: Status or lint semantics become over-specified before implementation.
  - Mitigation: Define only contract-level boundaries and leave concrete
    behavior to #273 and #274.
- Risk: Challenge becomes mandatory ceremony.
  - Mitigation: Define challenge as optional unless a plan declares it
    required, with clear triggers for when agents should consider it.

## Validation Summary

- `harness plan lint docs/plans/active/2026-06-30-define-goal-oriented-workflow-contract.md`
  passed.
- `git diff --check` passed.
- Targeted `rg` checks confirmed the specs describe `goal_oriented` as a
  reserved contract instead of a currently lint-valid profile value, and that
  #270-#275 remain the implementation follow-up surface.
- `scripts/validate` passed, including UI build, Go package tests, and e2e,
  release, resilience, smoke, and support test packages.

## Review Summary

- Step-closeout review `review-001-delta` requested changes for two related
  contract risks: the first draft made `goal_oriented` look currently
  lint-valid, and archive/reopen profile identity handling was implicit.
- Repair review `review-002-delta` passed with no findings after the contract
  was reframed as reserved until follow-up implementation slices add authoring,
  lint, status, archive, and reopen support.
- Finalize review `review-003-full` passed with no blocking or non-blocking
  findings.

## Archive Summary

- PR: Pending post-archive publish from branch
  `codex/define-goal-oriented-workflow-contract`.
- Ready: Yes. Acceptance criteria are checked, validation passed,
  step-closeout review passed after repair, and finalize review passed with no
  findings.
- Merge Handoff: After archive, push the branch, open a PR for issue #269,
  record publish/CI/sync evidence through harness, and wait for explicit human
  merge approval.

## Outcome Summary

### Delivered

- Added `docs/specs/goal-oriented-workflow.md` as the normative reserved
  `workflow_profile: goal_oriented` contract.
- Updated the spec index, plan schema, state model, and CLI contract to point
  to the goal-oriented contract while preserving current standard/lightweight
  implementation behavior.
- Defined checkpoint drafts, tracked checkpoint digests, checkpoint cadence,
  optional challenge, evidence/report ownership, advisory status guidance,
  lint boundaries, final synthesis, review, archive, and follow-up boundaries.
- Clarified that `goal_oriented` is reserved until follow-up implementation
  slices add authoring, lint, status, archive, and reopen support.

### Not Delivered

- CLI authoring, lint, status, archive, reopen, review-dimension, help, and
  example behavior for `goal_oriented`; those remain assigned to existing
  follow-up issues.

### Follow-Up Issues

- #270: Add the goal-oriented plan template and workflow guidance.
- #271: Define richer checkpoint report conventions and storage details if
  tracked plan digests are not enough.
- #272: Add evidence-validity and hypothesis-challenge guidance.
- #273: Teach status and next-action behavior for goal-oriented plans.
- #274: Add lint coverage for goal-oriented plans, including when
  `workflow_profile: goal_oriented` becomes lint-valid.
- #275: Add user-facing docs, help text, and examples.
