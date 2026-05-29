---
template_version: 0.2.0
created_at: "2026-05-29T23:58:05+08:00"
approved_at: "2026-05-30T00:07:14+08:00"
source_type: direct_request
source_refs:
    - https://github.com/obra/superpowers/blob/main/skills/brainstorming/SKILL.md
    - https://github.com/mattpocock/skills/blob/main/skills/productivity/grill-me/SKILL.md
size: S
---

# Tighten Harness Subagent and Discovery Prompts

## Goal

Update the harness-managed prompts so agents treat bounded subagents as a
normal, actively used part of harness work once the human authorizes them for a
harness run.

At the same time, tighten `harness-discovery` so it is used for unclear
workflow direction, boundaries, size, tradeoffs, or success criteria, not for
ordinary repo facts, code lookup, status checks, or simple explanations. The
discovery guidance should stay direct and low-jargon: agents should own repo
fact-finding, make clear recommendations, and avoid pushing low-level
implementation choices back to the human as false binary decisions.
When discovery asks the human a question, the preferred shape is: state the
agent's current read, give a direct recommendation when there is enough signal,
then ask one plain boundary or approval question. Avoid jargon-heavy labels,
hedging, and loaded binary choices such as "big breaking rewrite or minimal
tweak?"

The discovery update should deliberately learn from the current
`superpowers/brainstorming` and `grill-me` skills without copying either one
wholesale. Keep the harness skill closer to boundary-setting than grilling:
use the brainstorming-style context read, one-question rhythm, option framing,
and recommendation discipline; borrow the grill-me habit of making the agent
answer repo-factual questions itself before asking the human; do not turn
harness discovery into adversarial interrogation.

## Scope

### In Scope

- Update the harness-managed `AGENTS.md` block source to request one harness-run
  subagent authorization at the first harness workflow boundary.
- Clarify that authorization covers explorer, worker, and reviewer subagents.
- Strengthen the managed block so subagents are used proactively for bounded,
  independent work after authorization instead of treated as an unusual
  fallback.
- Align adjacent harness skill wording that currently talks about reviewer
  subagent authorization later in the workflow, so the new first harness
  workflow boundary authorization rule is not contradicted by plan or execute
  guidance.
- Update `harness-discovery` frontmatter so discovery is size-independent and
  keyed to unclear direction, boundaries, tradeoffs, success criteria, or
  workflow path.
- Clarify in `harness-discovery` that simple repo orientation, factual
  explanation, status checks, and implementation-ready clear-scope work should
  not enter discovery.
- Tighten discovery questioning guidance so agents answer repository-factual
  parts themselves, use explorer subagents for bounded repo questions when
  useful, keep synthesis in the controller, recommend a direction when
  appropriate, and ask direct human-facing boundary questions.
- Define the preferred question format for discovery: current read,
  recommendation when available, then one plain boundary or approval question.
  Explicitly discourage loaded either/or implementation choices and
  jargon-heavy wording.
- Preserve and strengthen the end-of-discovery summary handoff before planning:
  summarize what was discovered, what direction the human accepted, rejected
  alternatives, draft acceptance criteria, and the rough plan shape before
  switching to `harness-plan`.
- Fold in the relevant lessons from `superpowers/brainstorming` and
  `grill-me`: read context first, ask one useful question at a time, frame
  choices plainly, recommend a path when the agent has enough signal, and ask
  the human for goals, priorities, boundaries, and approval rather than repo
  facts.
- Sync bootstrap assets so the materialized `.agents/skills/` and root
  `AGENTS.md` managed block match the bootstrap source.

### Out of Scope

- Changing CLI behavior, command schemas, or state transitions.
- Changing the subagent tool itself or runtime permission rules.
- Reworking reviewer orchestration beyond aligning it with the shared
  subagent authorization rule.
- Adding compatibility shims for old prompt wording.

## Acceptance Criteria

- [x] The managed `AGENTS.md` block says that, once a harness workflow skill is
  triggered, the controller should ask once for this harness run whether bounded
  subagents may be used.
- [x] The managed block says the authorization covers explorer, worker, and
  reviewer subagents, and that subagents should be actively used for bounded,
  independent work after authorization.
- [x] The managed block still preserves human steering, controller ownership of
  shared context, bounded subagent tasks, and prompt/tool rules that require
  explicit authorization before spawning.
- [x] Adjacent `harness-plan` and `harness-execute` authorization wording is
  aligned with the shared first-boundary harness-run authorization model rather
  than preserving a reviewer-only or late-workflow permission model.
- [x] `harness-discovery` frontmatter no longer limits discovery to
  medium/large work and clearly excludes casual Q&A, simple repo orientation,
  simple status checks, already-approved execution, and implementation-ready
  clear-scope changes.
- [x] `harness-discovery` guidance distinguishes agent-owned repo facts from
  human-owned goals, priorities, boundaries, approvals, and workflow direction.
- [x] `harness-discovery` discourages low-level binary implementation-choice
  questions and prefers direct recommendations plus one human-facing boundary
  question.
- [x] `harness-discovery` gives agents a concrete question format: current
  read, recommendation when there is enough signal, then one plain boundary or
  approval question, without jargon-heavy labels or loaded either/or wording.
- [x] `harness-discovery` requires a concise discovery summary before handoff
  to `harness-plan`, including discovered facts, confirmed direction, rejected
  alternatives, draft acceptance criteria, rough plan content, and the next
  workflow step.
- [x] `harness-discovery` explicitly captures the useful parts of
  `superpowers/brainstorming` and `grill-me` while preserving harness discovery
  as collaborative boundary-setting, not a user-grilling mode.
- [x] `scripts/sync-bootstrap-assets --check` passes after syncing managed
  assets.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Update the managed subagent contract

- Done: [x]

#### Objective

Revise the managed `AGENTS.md` source so harness runs ask once for broad
bounded-subagent authorization and then actively use authorized subagents.

#### Details

The wording should keep the human in control without making subagents feel
exceptional. It should trigger only when a harness workflow skill is actually
being used, not for casual chat or simple repo questions. The authorization
should cover explorer, worker, and reviewer subagents for the current harness
run.

Also update any nearby managed harness skill wording that still treats subagent
authorization as reviewer-only or something to request late in plan approval or
execution. Those sections should point back to the shared first-boundary
authorization model while still respecting explicit human approval before
subagent spawning.

#### Expected Files

- `assets/bootstrap/agents-managed-block.md`
- `assets/bootstrap/skills/harness-plan/SKILL.md`
- `assets/bootstrap/skills/harness-execute/SKILL.md`

#### Validation

- Reread the managed block for consistency with the existing "Humans steer.
  Agents execute." rule.
- Confirm the text does not imply the controller may spawn subagents without
  explicit user authorization.

#### Execution Notes

Updated the managed `AGENTS.md` source so subagents are described as a normal
harness workflow tool after explicit run-level authorization. The new wording
asks once at the first harness workflow boundary, covers explorer, worker, and
reviewer subagents, and keeps the controller responsible for shared context and
final workflow judgment. Also aligned `harness-plan` and `harness-execute` so
they no longer preserve reviewer-only or late-workflow authorization language.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This tightly-coupled prompt-only slice will receive one
final full review after bootstrap sync and validation.

### Step 2: Tighten the discovery prompt contract

- Done: [x]

#### Objective

Revise `harness-discovery` so its use-when, repo fact-finding, explorer use,
and question style match the agreed discovery boundary.

#### Details

Discovery should not be tied to work size because discovery may be needed to
learn the size. It should be tied to unclear direction, scope, boundary, size,
tradeoff, success criteria, or workflow path. Simple factual repo explanation
should stay outside discovery unless the user is deciding what work to do next.

Questioning guidance should stay plain and direct. Agents should not ask the
human to answer repository facts or choose between low-level implementation
mechanics when the controller can recommend the better path from repo context.
The skill should give agents a compact question pattern instead of abstract
style advice:

1. Say the current read in one or two sentences.
2. Recommend a direction when the evidence is strong enough.
3. Ask one plain question about the human-owned boundary, priority, or
   approval.

Bad discovery question shape:

> Do you want direct breaking schema convergence, or the minimal change of
> reordering fields and folding `remote_handoff`?

Better shape:

> I recommend making default `harness status` a short control-panel view and
> keeping full remote details for diagnostics. Are you comfortable changing the
> default output shape to get that clarity?

The existing end-of-discovery `Output` section should be kept and made harder
to skip. Before handing off to `harness-plan`, the controller should give a
concise discovery summary with:

- what was discovered
- the accepted direction
- rejected alternatives and why
- draft acceptance criteria
- the rough plan shape
- the next workflow step

Use the two reference skills as design input:

- From `superpowers/brainstorming`, keep the flow of reading context first,
  asking one focused question per turn, using concise option framing when it
  helps, and giving a recommendation rather than staying neutral when tradeoffs
  are asymmetric.
- From `grill-me`, keep the discipline that the agent should investigate
  factual repo questions itself before asking the human.
- Do not import the whole grill-me posture. Harness discovery should help the
  human clarify boundaries and workflow direction, not pressure-test them with
  a long interrogation.

#### Expected Files

- `assets/bootstrap/skills/harness-discovery/SKILL.md`

#### Validation

- Reread the skill as if a future agent has no chat context.
- Confirm it gives enough guidance to avoid the observed "breaking schema or
  minimal tweak?" style of implementation-level binary question.

#### Execution Notes

Updated `harness-discovery` source frontmatter and body so discovery is
size-independent, excludes simple repo orientation/status/code lookup and clear
implementation-ready work, assigns repo facts to the agent and goals/boundaries
to the human, defines the current-read/recommendation/plain-question format,
discourages loaded implementation binaries, and strengthens the required
end-of-discovery summary before handoff to `harness-plan`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This tightly-coupled prompt-only slice will receive one
final full review after bootstrap sync and validation.

### Step 3: Sync and validate bootstrap output

- Done: [x]

#### Objective

Materialize the bootstrap source changes into repo-local generated prompt
assets and run the relevant drift checks.

#### Details

Use the repository's bootstrap sync script rather than hand-editing generated
`.agents/skills/` files or the managed root `AGENTS.md` block.

#### Expected Files

- `AGENTS.md`
- `.agents/skills/harness-discovery/SKILL.md`
- `.agents/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Run `scripts/sync-bootstrap-assets --check`.
- Run `harness plan lint docs/plans/active/2026-05-29-tighten-harness-subagent-and-discovery-prompts.md`.

#### Execution Notes

Ran `scripts/sync-bootstrap-assets` to update the materialized root
`AGENTS.md` block and `.agents/skills/` copies from `assets/bootstrap/`.
Validation passed with `scripts/sync-bootstrap-assets --check`, `harness plan
lint docs/plans/active/2026-05-29-tighten-harness-subagent-and-discovery-prompts.md`,
and a stale-wording search for the old medium/large, Socratic, and
reviewer-only authorization phrases across the edited managed prompt files.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step validation is mechanical bootstrap sync and lint;
the final full review will cover the prompt wording as one coherent change.

## Validation Strategy

- Lint the tracked plan before approval.
- During execution, use bootstrap sync and its `--check` mode as the primary
  prompt drift validation.
- Review the updated managed instructions and discovery skill manually against
  the discovery decisions captured in this plan.

## Risks

- Risk: The managed block could make subagent use sound automatic even though
  Codex requires explicit user authorization.
  - Mitigation: Require one harness-run authorization at the first harness
    workflow boundary, then allow proactive bounded use only after that.
- Risk: Discovery guidance could become too long or jargon-heavy and increase
  agent/user cognitive load.
  - Mitigation: Prefer direct rules and short examples over abstract labels.
- Risk: Discovery exclusions could prevent useful discovery for small work.
  - Mitigation: Make discovery size-independent and exclude only clear factual
    or implementation-ready cases.

## Validation Summary

Validated the prompt-only candidate with:

- `git diff --check`
- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-29-tighten-harness-subagent-and-discovery-prompts.md`
- stale-wording search across the edited managed prompt surfaces for old
  medium/large discovery, Socratic discovery, reviewer-only authorization, and
  late reviewer-subagent authorization language

## Review Summary

`review-001-full` passed cleanly with 0 findings. Reviewer slots:

- `docs_consistency`: confirmed the active plan, managed `AGENTS.md` source and
  output, bootstrap skill sources, and materialized skill copies consistently
  describe run-level bounded-subagent authorization, active post-authorization
  subagent use, and the tightened discovery boundary.
- `agent_ux`: confirmed the updated prompts are direct and usable for future
  agents, including size-independent discovery, agent-owned repo facts, the
  current-read/recommendation/plain-boundary question format, no loaded
  implementation binaries, and explicit discovery-summary handoff.

## Archive Summary

- Archived At: 2026-05-30T00:16:03+08:00
- Revision: 1
- PR: pending publish after archive; no PR URL exists yet.
- Ready: The managed bootstrap source, root managed block, materialized
  harness skills, and active plan now satisfy the approved prompt-only scope.
  Validation passed and `review-001-full` found no issues.
- Merge Handoff: Archive the plan, commit the tracked plan move and closeout
  summary, push branch `codex/tighten-harness-subagent-discovery-prompts`, open
  a PR, record publish/CI/sync evidence, and wait for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Added first-boundary harness-run subagent authorization guidance to the
  managed `AGENTS.md` contract, covering explorer, worker, and reviewer
  subagents.
- Reframed subagents as a normal authorized harness workflow tool for bounded,
  independent work while preserving controller ownership and human steering.
- Aligned `harness-plan` and `harness-execute` with the shared run-level
  authorization model instead of reviewer-only late authorization wording.
- Updated `harness-discovery` so discovery is size-independent, excludes simple
  repo Q&A/status/code lookup and clear implementation-ready work, assigns repo
  facts to the agent, gives a direct question format, discourages loaded
  implementation binaries, and requires a concise discovery summary before
  handoff to `harness-plan`.
- Synced bootstrap outputs into root `AGENTS.md` and `.agents/skills/`.

### Not Delivered

No CLI behavior, command schema, state transition, or subagent runtime changes
were made.

### Follow-Up Issues

NONE
