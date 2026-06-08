---
template_version: 0.2.0
created_at: "2026-06-08T22:20:00+08:00"
approved_at: "2026-06-08T22:34:02+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/pull/224
size: S
---

# Tighten Discovery Turn Exit Discipline

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Tighten the managed `harness-discovery` skill so discovery agents reliably end
each turn in the right mode: factual answer, genuine decision question, or
plan-ready handoff. The fix should make the skill general enough to guide any
unclear harness work, not just one recent conversation pattern.

After this slice, discovery should avoid two common failures: asking a question
when repository facts already make the next workflow step clear, and presenting
multiple options before collapsing the final ask into a narrower or different
choice.

## Scope

### In Scope

- Update the easyharness-managed `harness-discovery` skill source under
  `assets/bootstrap/`.
- Sync the materialized `.agents/skills/harness-discovery/` output from the
  bootstrap source.
- Clarify the allowed discovery turn exits:
  - direct factual answer when no workflow decision is pending
  - one focused question when a real human decision is needed
  - plan-ready summary and rough plan outline when discovery has converged
- Clarify that agents should not invent a question merely to keep discovery
  going.
- Clarify option integrity rules so an option-shaped prompt keeps the same
  options through the final question.
- Add concise positive guidance and at least one anti-pattern that is generic
  enough to prevent the failure mode without binding the skill to a specific
  issue, feature, or repository topic.
- Add an execution-time validation plan that uses spawned agents for
  interactive black-box checks of the revised skill behavior.
- Validate bootstrap source/output sync and the active plan.

### Out of Scope

- Changing CLI behavior, state transitions, review orchestration, or subagent
  runtime behavior.
- Reworking all harness-managed skills beyond the discovery skill unless sync
  naturally updates generated managed output.
- Encoding a case-specific rule for any one GitHub issue, PR, feature area, or
  conversation transcript.
- Adding a separate automated prompt-evaluation harness.
- Treating spawned-agent validation as a replacement for source review,
  bootstrap sync checks, or tracked plan lint.

## Acceptance Criteria

- [x] `harness-discovery` explicitly tells agents that if no real question
      remains, they should move to a plan-ready summary and rough plan outline
      rather than asking a filler question.
- [x] `harness-discovery` defines a small set of valid turn exits that future
      agents can follow without relying on hidden chat context.
- [x] Option-framed discovery prompts must preserve their presented options in
      the final ask; they must not collapse, rename, or replace the option set
      at the end of the turn.
- [x] The wording stays topic-general and reusable across discovery for bugs,
      enhancements, workflow design, triage, and reopening decisions.
- [x] Execution validation includes `spawn_agent`-based interactive probes that
      exercise at least these discovery shapes: factual answer with no pending
      decision, unresolved human decision with one focused question, plan-ready
      handoff with rough plan outline, and option-framed prompt with preserved
      option set.
- [x] `assets/bootstrap/skills/harness-discovery/SKILL.md` and
      `.agents/skills/harness-discovery/SKILL.md` are synchronized.
- [x] Validation covers plan lint, bootstrap sync check, spawned-agent probe
      notes, and diff hygiene.

## Deferred Items

- A broader prompt-evaluation or transcript-regression framework is deferred.
- Additional tuning of `harness-plan`, `harness-execute`, or reviewer prompt
  behavior is deferred unless a separate issue or discovery slice accepts it.

## Work Breakdown

### Step 1: Tighten discovery turn exits and option integrity

- Done: [x]

#### Objective

Update the managed discovery skill source with concise, general rules for turn
exit selection and option-framed questions.

#### Details

Revise `assets/bootstrap/skills/harness-discovery/SKILL.md` so the skill gives
agents an explicit decision loop for ending a discovery turn. The guidance
should make clear that discovery does not always end in a question: once repo
facts and human intent are clear enough, the correct next move is to summarize
the accepted direction, draft acceptance criteria, sketch the rough plan shape,
and ask for approval to hand off to `harness-plan`.

Also strengthen the option-framing section. When the agent presents options,
the final question should reference the same option set, such as asking the
human to choose, edit, or reject one of the listed options. The skill should
warn against presenting three options and then ending with a narrowed two-way
choice, renamed options, or a new choice that was not part of the framed set.

Keep the text compact and topic-neutral. Use examples only to illustrate the
interaction pattern, not any specific issue or feature.

#### Expected Files

- `assets/bootstrap/skills/harness-discovery/SKILL.md`

#### Validation

- Reread the edited skill and confirm the new guidance is general rather than
  tied to a specific feature, issue, or chat transcript.
- Run `git diff --check`.

#### Execution Notes

Updated `assets/bootstrap/skills/harness-discovery/SKILL.md` with a
topic-neutral turn exit discipline covering factual answers, decision
questions, and plan-ready handoff. Tightened the execution contract and option
framing guidance so agents preserve the presented option set in the final ask
and avoid filler questions when discovery has already converged. Reread the
edited guidance for issue-specific overfitting and ran `git diff --check`
successfully.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Prompt-only source guidance will be validated through
managed-output sync, spawned-agent probes in Step 3, and finalize review.

### Step 2: Sync managed outputs and validate the plan package

- Done: [x]

#### Objective

Refresh generated managed skill output and verify the tracked plan remains
valid and executable.

#### Details

Run `scripts/sync-bootstrap-assets` after editing the bootstrap source so the
materialized `.agents/skills/harness-discovery/SKILL.md` stays in sync. Inspect
the resulting diff to ensure sync did not introduce unrelated managed prompt
changes.

Then lint this active plan and run the repository's bootstrap sync check. If
the sync script has a check mode, use it after the write mode to prove the
source and materialized output are aligned.

#### Expected Files

- `.agents/skills/harness-discovery/SKILL.md`
- `docs/plans/active/2026-06-08-tighten-discovery-turn-exit-discipline.md`

#### Validation

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-06-08-tighten-discovery-turn-exit-discipline.md`
- `git diff --check`

#### Execution Notes

Ran `scripts/sync-bootstrap-assets`, which updated only the materialized
`.agents/skills/harness-discovery/SKILL.md` output. Verified the managed output
with `scripts/sync-bootstrap-assets --check`, linted this active plan with
`harness plan lint docs/plans/active/2026-06-08-tighten-discovery-turn-exit-discipline.md`,
and reran `git diff --check`. The prompt diff is limited to the bootstrap
discovery skill source and its materialized dogfood output.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is mechanical bootstrap synchronization and
validation; spawned-agent probes in Step 3 and finalize review cover behavior.

### Step 3: Probe the revised skill with spawned agents

- Done: [x]

#### Objective

Use spawned agents as interactive black-box validation for the revised
discovery guidance before archive.

#### Details

After the skill text has been edited and synced, spawn small, bounded
validation agents that are asked to use the updated `harness-discovery` skill
against synthetic discovery prompts. The probes should not be told the intended
fix in detail; they should receive the updated skill plus enough task context
to produce a discovery response.

Cover at least four prompt shapes:

- a factual repository-orientation request where discovery should answer
  directly and avoid adding a workflow question
- an unclear objective where one real human decision remains and the response
  should ask exactly one focused question
- a converged objective where no real question remains and the response should
  present a plan-ready summary plus rough plan outline
- an option-framed decision where the response presents multiple options and
  the final ask preserves that same option set

Treat the spawned agents' outputs as evidence, not authority. The controller
should inspect the responses, note whether the revised skill produced the
intended turn shape, and adjust the skill if a probe still shows filler
questions, premature plan handoff, or option-set collapse. Close spawned agents
after their bounded validation result is captured.

#### Expected Files

- `assets/bootstrap/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-discovery/SKILL.md`
- `docs/plans/active/2026-06-08-tighten-discovery-turn-exit-discipline.md`

#### Validation

- Spawn interactive validation agents with bounded prompts for the four
  discovery shapes above.
- Record concise probe outcomes in this plan's execution notes or validation
  summary before archive.
- Confirm no spawned-agent output requires topic-specific rules or additional
  hidden context to pass.

#### Execution Notes

Spawned four bounded validation agents against the synced
`.agents/skills/harness-discovery/SKILL.md` and closed all four after their
results were captured:

- factual-answer probe: answered idle harness status directly and did not add a
  workflow question
- unresolved-decision probe: framed three viable review-workflow surfaces and
  asked one focused choice question; minor observation that it listed options
  numerically but ended with A/B/C labels while still preserving the option set
- plan-ready probe: stated discovery had converged, summarized the accepted
  prompt-only direction, drafted acceptance criteria, sketched the plan shape,
  and asked to hand off to `harness-plan`
- option-integrity probe: presented Options A, B, and C and ended by asking
  which of A, B, C, or a revision should be planned around

The probes exercised the intended turn shapes without requiring topic-specific
rules or hidden controller context.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is the spawned-agent validation closeout
itself; finalize review will inspect the full prompt change and recorded probe
evidence before archive.

## Validation Strategy

Use plan lint for the tracked plan contract, bootstrap sync check for managed
asset consistency, and diff hygiene for whitespace and patch cleanliness.
Because this is a prompt-only change, the main behavioral validation is careful
review of the edited discovery instructions against generic discovery
scenarios plus spawned-agent probes that simulate discovery turns without
sharing the controller's private reasoning. The probes should cover factual
answer, unresolved decision, plan-ready handoff, and option-framed decision
integrity.

## Risks

- Risk: The skill becomes too procedural and makes discovery feel mechanical.
  - Mitigation: Keep the new rules short, describe exits rather than scripts,
    and preserve the existing Socratic style.
- Risk: The fix overfits one observed conversation.
  - Mitigation: Avoid issue-specific examples and phrase the rules around
    reusable turn-shape failures.
- Risk: Bootstrap source and generated skill output drift.
  - Mitigation: Edit only `assets/bootstrap/`, run the sync script, and verify
    with check mode.
- Risk: Spawned-agent probes overfit to the controller's expected answer.
  - Mitigation: Keep validation prompts synthetic, bounded, and minimally
    revealing; judge turn shape rather than exact wording.

## Validation Summary

Validated the prompt-only change with:

- `scripts/sync-bootstrap-assets`
- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-06-08-tighten-discovery-turn-exit-discipline.md`
- `git diff --check`
- four spawned-agent probes covering factual answer, unresolved decision,
  plan-ready handoff, and option-integrity scenarios

All validation passed. The only probe observation was cosmetic: one unresolved
decision response numbered the options but ended with A/B/C labels; the option
set itself stayed intact, so no skill change was needed.

## Review Summary

Finalize review `review-001-full` passed with 0 findings across
`docs_consistency` and `agent_ux`. Reviewers confirmed that the bootstrap
source, materialized managed output, and plan are consistent, and that the
revised discovery guidance clarifies turn exits and option-set preservation
without becoming overly rigid.

## Archive Summary

- Archived At: 2026-06-08T22:42:03+08:00
- Revision: 1
- PR: Pending post-archive publish handoff from branch
  `codex/tighten-discovery-turn-exit`.
- Ready: The prompt-only candidate is ready to archive after successful sync,
  lint, diff hygiene, spawned-agent probes, and clean finalize review.
- Merge Handoff: Commit the archived plan and prompt updates, push
  `codex/tighten-discovery-turn-exit`, open a PR, then record publish, CI, and
  sync evidence before waiting for merge approval.

## Outcome Summary

### Delivered

- Added generic turn-exit guidance to the managed `harness-discovery` skill.
- Tightened option-framing instructions so the final ask preserves the
  presented option set.
- Synced the materialized dogfood skill output from `assets/bootstrap/`.
- Validated the prompt behavior with bounded spawned-agent probes.

### Not Delivered

- No CLI behavior, state transition, subagent runtime, or broader prompt
  evaluation framework changes were included.

### Follow-Up Issues

- No new follow-up issues were created. Deferred scope remains advisory only:
  a broader prompt-evaluation framework and additional managed-skill tuning
  should each be handled by a separate accepted discovery slice if they become
  concrete work.
