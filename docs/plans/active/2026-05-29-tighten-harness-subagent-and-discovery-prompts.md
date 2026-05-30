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

Update the harness-managed prompts so agents treat specific, well-scoped
subagents as a normal, actively used part of harness work once the human
authorizes them for a harness run.

At the same time, tighten `harness-discovery` so it is used for unclear
workflow direction, boundaries, size, tradeoffs, or success criteria, not for
ordinary repo facts, code lookup, status checks, or simple explanations. The
discovery guidance should stay direct and low-jargon: agents should own repo
fact-finding, frame choices as options, make clear recommendations under the
preferred option, and avoid pushing low-level implementation choices back to the
human as false binary decisions.
When discovery asks the human a question, the preferred shape is: state the
agent's current read, present 2-4 realistic options, put the recommendation
under the preferred option, and ask the human to choose, edit, or reject the
options. Avoid jargon-heavy labels, hedging, and loaded binaries such as "big
breaking rewrite or minimal tweak?"

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
  explanation, status checks, and already-approved execution should not expand
  into discovery.
- Tighten discovery questioning guidance so agents answer repository-factual
  parts themselves, use explorer subagents for bounded repo questions when
  useful, keep synthesis in the controller, frame choices as options, put the
  recommendation under the preferred option, and ask direct human-facing
  boundary questions.
- Define the preferred question format for discovery: current read, 2-4
  realistic options, recommendation under the preferred option, then a request
  for the human to choose, edit, or reject the options.
  Explicitly discourage loaded either/or implementation choices and
  jargon-heavy wording.
- Preserve and strengthen the end-of-discovery summary handoff before planning:
  summarize what was discovered, what direction the human accepted, rejected
  alternatives, draft acceptance criteria, and the rough plan shape before
  switching to `harness-plan`.
- Fold in the relevant lessons from `superpowers/brainstorming` and
  `grill-me`: read context first, ask one useful question at a time, frame
  choices plainly as options, recommend a path when the agent has enough
  signal, and treat the human as having final say over goals, priorities,
  boundaries, and approval when repo context does not already settle them.
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
  triggered, the controller should ask once for this harness run whether
  specific, well-scoped subagents may be used.
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
  simple status checks, and already-approved execution.
- [x] `harness-discovery` guidance distinguishes agent-investigated repo facts
  and documented intent from human final say over goals, priorities,
  boundaries, approvals, and workflow direction when those are ambiguous,
  contested, or not already settled in the repository.
- [x] `harness-discovery` discourages low-level binary implementation-choice
  questions and prefers neutral option framing with a recommendation under the
  preferred option.
- [x] `harness-discovery` gives agents a concrete question format: current
  read, 2-4 realistic options, recommendation under the preferred option, then
  a request for the human to choose, edit, or reject the options, without
  jargon-heavy labels or loaded either/or wording.
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
subagent authorization and then actively use authorized, specific, well-scoped
subagents.

#### Details

The wording should keep the human in control without making subagents feel
exceptional. It should trigger only when a harness workflow skill is actually
being used, not for casual chat or simple repo questions. The authorization
should cover explorer, worker, and reviewer subagents for the current harness
run, using plain wording such as "specific, well-scoped subagents" where that
is clearer than "bounded subagents."

Do not duplicate the authorization prompt in every harness skill. The shared
managed `AGENTS.md` block owns that run-level rule. Nearby managed harness
skills should avoid reviewer-only or late-workflow authorization wording that
would contradict the shared model.

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
harness workflow tool after explicit run-level authorization. The revised
wording asks once at the first harness workflow boundary, uses "specific,
well-scoped subagents" for the authorization prompt, covers explorer, worker,
and reviewer subagents, and keeps the controller responsible for shared context
and final workflow judgment. Removed duplicate authorization prompting from
`harness-plan` and `harness-execute` while preserving the shared managed-block
rule.

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
The skill should give agents a compact option-shaped question pattern instead
of abstract style advice:

1. Say the current read in one or two sentences.
2. Present 2-4 realistic options, even for small decisions.
3. Put the recommendation under the option the agent prefers, with a short
   reason.
4. Ask the human to choose, edit, or reject the options.

Bad discovery question shape:

> Do you want the broad breaking rewrite, or the minimal low-risk patch?

Better shape:

1. `Option A`
   - upside: the main goal is addressed directly
   - downside: the change may be broader
   - best when: the clean target shape matters most
   - recommendation: I recommend this when repo context supports the broader
     change.
2. `Option B`
   - upside: the change is smaller
   - downside: the original confusion may only be reduced, not removed
   - best when: limiting scope matters most

Which direction should I plan around?

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
  helps, and putting a recommendation under the preferred option rather than
  staying neutral when tradeoffs are asymmetric.
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
interactive and collaborative, size-independent, excludes simple repo
orientation/status/code lookup and already-approved execution, treats repo
facts and documented intent as agent-investigated context while preserving
human final say over ambiguous goals/boundaries/approvals, defines the
current-read/options/recommendation-under-preferred-option format, discourages
loaded implementation binaries, and strengthens the required end-of-discovery
summary before handoff to `harness-plan`.

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
    workflow boundary, then allow proactive use of specific, well-scoped
    subagents only after that.
- Risk: Discovery guidance could become too long or jargon-heavy and increase
  agent/user cognitive load.
  - Mitigation: Prefer direct rules and short examples over abstract labels.
- Risk: Discovery exclusions could prevent useful discovery for small work.
  - Mitigation: Make discovery size-independent and exclude only simple factual
    orientation, status/code lookup, and already-approved execution cases.

## Validation Summary

Revision 2 validation covers the original prompt-only candidate plus the PR
feedback repair:

- `git diff --check`
- `scripts/sync-bootstrap-assets --check`
- `harness plan lint docs/plans/active/2026-05-29-tighten-harness-subagent-and-discovery-prompts.md`
- stale-wording search across the edited managed prompt surfaces for old
  medium/large discovery, Socratic discovery, reviewer-only authorization,
  late reviewer-subagent authorization, implementation-ready exclusion,
  obsolete direct-question format, and status-specific example wording
- no-context simulation subagents against the materialized prompts:
  - unclear `harness status` noise request correctly entered discovery and
    asked for run-level subagent authorization before repo exploration
  - dashboard watchlist code-lookup request correctly bypassed discovery and
    did not ask for subagent authorization
- after `review-002-full` found the reusable option pattern still placed the
  recommendation after the list, updated both bootstrap source and materialized
  output so the recommendation lives under the preferred option, then reran
  `scripts/sync-bootstrap-assets --check` and `git diff --check`

## Review Summary

Revision 1 `review-001-full` passed cleanly with 0 findings before PR feedback.
Revision 2 `review-002-full` found 1 blocking `agent_ux` issue: the reusable
`Option Framing Pattern` still told agents to add the recommendation after the
options. The source and materialized skill were repaired so the reusable
pattern now puts the recommendation under the preferred option.
Revision 1 reviewer slots:

- `docs_consistency`: confirmed the active plan, managed `AGENTS.md` source and
  output, bootstrap skill sources, and materialized skill copies consistently
  described the revision-1 run-level subagent authorization, active
  post-authorization subagent use, and the tightened discovery boundary.
- `agent_ux`: confirmed the updated prompts are direct and usable for future
  agents, including size-independent discovery, agent-owned repo facts, the
  earlier direct question format, no loaded implementation binaries, and
  explicit discovery-summary handoff.

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
- Reframed subagents as a normal authorized harness workflow tool for specific,
  well-scoped work while preserving controller ownership and human steering.
- Removed duplicated subagent authorization prompts from `harness-plan` and
  `harness-execute`; the managed `AGENTS.md` block owns the shared run-level
  authorization model.
- Updated `harness-discovery` so discovery is size-independent, excludes simple
  repo Q&A/status/code lookup and already-approved execution, treats repo facts
  and documented intent as agent-investigated context, gives an option-shaped
  question format with the recommendation under the preferred option,
  discourages loaded implementation binaries, and requires a concise discovery
  summary before handoff to `harness-plan`.
- Synced bootstrap outputs into root `AGENTS.md` and `.agents/skills/`.

### Not Delivered

No CLI behavior, command schema, state transition, or subagent runtime changes
were made.

### Follow-Up Issues

NONE
