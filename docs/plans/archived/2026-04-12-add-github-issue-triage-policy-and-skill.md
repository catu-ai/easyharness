---
template_version: 0.2.0
created_at: "2026-04-12T18:43:00+08:00"
approved_at: "2026-04-12T18:33:19+08:00"
source_type: issue
source_refs:
    - '#157'
    - '#87'
size: M
---

# Add GitHub issue triage policy and a reusable backlog-triage skill

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Turn the current undifferentiated GitHub issue backlog into a lightweight,
explainable steering surface. The repository should define one small triage
system that makes it obvious whether an issue was reviewed, what kind of
follow-up it represents, whether it is missing information, whether it is
deferred, and whether it has been accepted into a concrete release milestone.

This slice should not introduce a heavy project-management layer. Instead, it
should standardize a narrow set of GitHub labels plus concrete version
milestones, add one repo-local skill used only in this repository to tell
future agents how to triage issues consistently, and require a short issue
comment whenever triage changes so future passes over `state/deferred` and
`state/needs-info` can understand the earlier judgment and decide whether the
issue is ready to move.

## Scope

### In Scope

- Define the lightweight GitHub issue triage policy for this repository.
- Use existing default type labels as the primary issue-kind surface rather
  than inventing a parallel type taxonomy.
- Add the accepted triage-state labels for reviewed-but-unscheduled issues:
  `state/accepted`, `state/needs-info`, and `state/deferred`.
- Use concrete GitHub milestones such as `v0.2.2` to express release intent
  once an issue is actually accepted into a specific version scope.
- Require a short triage rationale comment whenever an issue is newly triaged
  or its triage state materially changes.
- Create one repo-local user-owned skill used only in this repository to guide
  agents through issue triage, including label selection, milestone rules, and
  the rationale-comment requirement.
- Backfill the current open GitHub issues using the new labels, milestones
  where appropriate, and rationale comments.

### Out of Scope

- Introducing GitHub Projects, custom fields, or a broader project-management
  workflow.
- A multi-level `priority/*` system unless the implementation work shows a
  concrete repository need that the agreed minimal system cannot handle.
- Rewriting the semantic-versioning or release-cadence policy itself beyond
  clarifying how concrete milestones interact with the separate release-policy
  follow-up in `#87`.
- Automating issue triage through bots or GitHub Actions in this slice.
- Backfilling closed historical issues.

## Acceptance Criteria

- [x] The repository documents one lightweight triage policy that a cold reader
      can follow without discovery chat, including:
      default type-label guidance, the three `state/*` labels, when to use a
      concrete version milestone, and when to close an issue instead of
      keeping it open.
- [x] The policy states that applying any triage label or concrete milestone
      counts as triaged, and that each triage decision or state change must
      leave a short rationale comment on the issue.
- [x] A new repo-local user-owned skill exists only for this repository,
      teaching future agents how to triage `easyharness` issues and what
      rationale comment shape to leave, without being folded into the
      easyharness-managed bootstrap pack.
- [x] The current open GitHub issues are backfilled to the new system, with no
      ambiguous open issue left in a reviewed state without either a triage
      label, a concrete milestone, or an intentional close decision.
- [x] The resulting system stays lightweight enough that maintainers can keep
      using it issue-by-issue without needing a separate project board.

## Deferred Items

- Any later expansion to `priority/*` labels if the backlog grows enough that
  milestone and triage-state signals no longer provide enough ordering.
- Project-board or custom-field workflows if the repository later needs a more
  operational planning surface than labels, milestones, and comments.
- Release-cadence decisions such as when to cut `v0.2.2`, what validation bar
  it needs, and how urgent patch releases interact with normal release timing;
  those remain the focus of `#87`.

## Work Breakdown

### Step 1: Define the repository triage policy and rationale-comment contract

- Done: [x]

#### Objective

Write the durable repository-facing policy for how GitHub issue triage works in
`easyharness`.

#### Details

This step should capture the decisions from discovery in tracked docs: default
issue kinds keep using the GitHub defaults already present in the repository,
reviewed issue state is expressed through `state/accepted`,
`state/needs-info`, and `state/deferred`, concrete release intent is expressed
through milestones such as `v0.2.2`, and an issue that is not planned should
usually be closed instead of staying in the open backlog.

The policy should also define the durable comment contract. Every time an
issue is first triaged or later moves between accepted, needs-info, deferred,
or a concrete milestone, the agent should leave a short comment explaining the
judgment basis. Those comments are the historical breadcrumb that later agents
should use when re-sweeping deferred or information-blocked issues instead of
guessing why an older decision was made.

The documentation should explain how this slice interacts with `#87`: this
triage system can assign a concrete version milestone once maintainers know the
intended target, but it does not by itself define the release cadence or
quality bar for cutting that version.

#### Expected Files

- `README.md` and/or a durable docs location such as `docs/releasing.md`
- any nearby tracked docs that should point maintainers toward the new policy

#### Validation

- The policy is specific enough that a future agent can decide among accepted,
  needs-info, deferred, milestone, or close without discovery chat.
- The policy explicitly states when a rationale comment is required and what it
  needs to capture.

#### Execution Notes

Added a dedicated tracked policy at `docs/issue-triage.md` that defines the
lightweight triage system for this repository: keep the default GitHub type
labels, use only `state/accepted`, `state/needs-info`, and `state/deferred`
for reviewed unscheduled issues, use concrete milestones such as `v0.2.2` only
for truthful version intent, and close not-planned issues instead of leaving
them open with a negative label.

Updated `AGENTS.md` and `docs/development.md` to point future agents at the new
policy and to carve out the `issue-triage` repo-only skill as an intentional
exception to the default "bootstrap-managed skills live under `.agents/skills/`
as generated output" rule. Also added a short `docs/releasing.md` note that
issue milestones express version scope but do not replace the separate
release-policy work in `#87`.

Revision 2 reopen follow-up tightened that ownership wording after PR review:
the docs now state the general rule that `harness-*` skills are the
easyharness-managed distributed pack, while other names are repo-owned local
development skills. `docs/issue-triage.md` also now says explicitly that the
current label system is a tracked convention plus live GitHub metadata, not an
already-automated `.github` label-sync setup.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The policy doc, repo-only skill, and live GitHub
backfill are one tightly coupled backlog-governance slice, so reviewing them
separately at step closeout would add churn without improving judgment.

### Step 2: Add a repo-only issue triage skill for this repository

- Done: [x]

#### Objective

Create the repo-local skill that teaches future agents how to apply the policy
consistently without turning it into a shipped easyharness bootstrap asset.

#### Details

This step should create a new repo-only skill dedicated to GitHub issue
triage for `easyharness`. It should be a user-owned skill package for this
repository rather than an easyharness-managed bootstrap asset, so it should
not live under `assets/bootstrap/` and should not use a `harness-*` name or
easyharness-managed metadata. Put it where this repository can use it locally
without implying that `harness init` should distribute it elsewhere.

The skill should stay concise and procedural. It should tell agents how to:
inspect the issue, choose the default type label when needed, decide among
`state/accepted`, `state/needs-info`, `state/deferred`, or a concrete version
milestone, leave a rationale comment, and avoid over-classifying the backlog
with heavier systems such as `priority/*` or project boards. If a short
comment template or reference file improves consistency without bloating the
skill body, add that as part of the skill package. If UI metadata is expected
for the new skill, generate and validate it rather than hand-waving the file.

Because this is a repo-only skill rather than a bootstrap-managed one, this
step should avoid `scripts/sync-bootstrap-assets` and any edits that would
imply the skill ships as part of the managed pack.

#### Expected Files

- `.agents/skills/<new-triage-skill>/SKILL.md`
- any minimal supporting `agents/openai.yaml`, `references/`, or `assets/`
  files needed by that skill
- any durable repository docs that should point agents toward the repo-only
  skill without claiming it is bootstrap-managed

#### Validation

- The new skill is concise, repo-specific, and gives a future agent a clear
  triage workflow plus rationale-comment rule.
- The skill is clearly user-owned and repo-local rather than an
  easyharness-managed distributed asset.
- Any required skill validation passes.

#### Execution Notes

Created the repo-only skill package at `.agents/skills/issue-triage/` using the
skill-creator initializer, then replaced the template content with a concise
triage workflow tailored to this repository. The skill now points agents at the
tracked policy, explains how to choose among accepted / needs-info / deferred /
milestone / close, and makes rationale comments part of the workflow instead of
an optional extra.

Generated `agents/openai.yaml` for the new skill, added a small
`references/rationale-comments.md` helper with comment shapes, and validated the
skill with the skill-creator `quick_validate.py` script. Kept the skill
explicitly repo-owned: no `harness-*` naming, no `easyharness-managed`
metadata, and no bootstrap sync.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The skill only makes sense together with the policy and
the live GitHub backfill, so this step is reviewed as part of the same full
finalize pass.

### Step 3: Create GitHub metadata and backfill the open issue backlog

- Done: [x]

#### Objective

Apply the new system to the live GitHub backlog so the policy is real rather
than only documented.

#### Details

This step should create any missing GitHub labels for `state/accepted`,
`state/needs-info`, and `state/deferred`, create concrete version milestones
only where the current backlog really justifies them, and backfill each open
issue accordingly. Every reviewed issue should end this step in one of these
states: explicitly accepted but unscheduled, explicitly waiting on missing
information, explicitly deferred, accepted into a concrete milestone, or
closed as not planned.

Each triaged issue should receive a short comment capturing the reasoning and,
when useful, the condition that would cause the issue to be revisited. That is
especially important for `state/needs-info` and `state/deferred`, because the
future sweep needs a durable explanation rather than a bare label.

This step should leave the backlog readable without creating a fake sense of
precision. Do not force every issue into a milestone if the repository does
not yet have a truthful concrete version target for it.

#### Expected Files

- no tracked repository files beyond any docs/skill adjustments needed while
  applying the policy

#### Validation

- GitHub now contains the agreed triage labels.
- Every current open issue is backfilled consistently with the documented
  policy.
- Each triaged issue has a rationale comment that matches its current state.

#### Execution Notes

Created the new live GitHub labels `state/accepted`, `state/needs-info`, and
`state/deferred`, plus milestone `v0.2.2` for concrete next-patch scope. Then
backfilled every open issue so none remained in an ambiguous reviewed state:
`#157` now targets `v0.2.2`; exploratory architectural work such as `#156`
lands in `state/needs-info`; intentionally later or not-yet items such as
`#146`, `#136`, `#133`, and `#128` land in `state/deferred`; and the remaining
reviewed backlog is marked `state/accepted`.

Left a short rationale comment on each triaged issue explaining why that state
was chosen and, where useful, what should trigger a later revisit. One
duplicate triage comment landed on `#133` during a retry and was removed so the
backlog breadcrumbs stayed clean.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The GitHub metadata backfill should be reviewed
together with the new policy and skill that justify those decisions, so the
controller will use one full finalize review instead of isolated step review.

## Validation Strategy

- Run `harness plan lint` on the tracked plan before approval.
- Validate any new or updated skill with the repository's expected bootstrap
  sync and skill-validation workflow.
- Before archive, do a direct GitHub audit of labels, milestones, and issue
  comments to confirm the live backlog matches the documented policy.

## Risks

- Risk: The triage system could still be too abstract, leaving future agents to
  interpret the state labels differently.
  - Mitigation: Keep the labels minimal, define them in durable docs, and make
    rationale comments mandatory so ambiguous edge cases remain legible later.
- Risk: The repo-only skill could accidentally be mistaken for part of the
  distributed easyharness bootstrap pack.
  - Mitigation: Keep it outside `assets/bootstrap/`, avoid `harness-*`
    naming and easyharness-managed metadata, and describe it explicitly as a
    repository-owned local skill.
- Risk: Backfilling the current backlog could tempt the agent to assign
  milestones that look organized but are not yet real commitments.
  - Mitigation: Use concrete milestones only when the version intent is
    truthful, and otherwise prefer accepted / needs-info / deferred plus a
    rationale comment.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-12-add-github-issue-triage-policy-and-skill.md`
  passed after the execution notes, summaries, and review-closeout text were
  filled in.
- `python3 /Users/yaozhang/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/issue-triage`
  passed for the new repo-only skill package.
- GitHub live-state audit confirmed the new `state/accepted`,
  `state/needs-info`, and `state/deferred` labels exist, milestone `v0.2.2`
  exists, every open issue now carries either a triage label or a milestone,
  and each triaged issue received a rationale comment.
- Revision 2 reopen validation confirmed the ownership wording now matches the
  intended rule: `harness-*` skills are the easyharness-managed distributed
  pack, while other names remain repo-owned local development skills.
- Revision 2 also confirmed the docs now state clearly that the current label
  system is tracked by repository docs plus live GitHub metadata rather than an
  existing `.github` label-sync automation path.

## Review Summary

- Finalize full review `review-001-full` passed with no findings.
- Reviewer slot `correctness` found no contradictions between the tracked
  policy, the repo-only skill, and the live GitHub backlog backfill.
- Reviewer slot `agent_ux` found the new skill discoverable from `AGENTS.md`
  and `docs/development.md`, with the rationale-comment rule and repo-only
  ownership boundary spelled out clearly.
- Reopen delta review `review-002-delta` also passed with no findings.
- Reviewer slot `docs-consistency` confirmed the revised ownership wording and
  the `.github` boundary text stay consistent across `AGENTS.md`,
  `docs/development.md`, and `docs/issue-triage.md`.

## Archive Summary

- Archived At: 2026-04-12T19:38:40+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/160
- Ready: The candidate adds a dedicated GitHub issue triage policy, introduces
  the repo-only `issue-triage` skill plus rationale-comment guidance, and
  backfills the live open-issue backlog with the new state labels and
  milestone semantics. Revision 2 tightened the skill-ownership wording and
  clarified that the current label system is a tracked convention plus live
  GitHub metadata rather than an existing `.github` automation layer. Finalize
  review `review-001-full` and reopen delta review `review-002-delta` both
  passed cleanly. After the refreshed candidate is pushed to PR #160 and the
  publish/CI/sync facts are refreshed for the new head, it is ready to wait for
  merge approval.
- Merge Handoff: Push the revision-2 repair to PR #160, refresh
  publish/CI/sync evidence for the latest head commit, and then wait for
  explicit merge approval once status returns to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added `docs/issue-triage.md` as the durable repository policy for GitHub
  backlog triage.
- Updated `AGENTS.md`, `docs/development.md`, and `docs/releasing.md` so
  future agents can discover the triage workflow, understand the repo-only
  ownership boundary of `issue-triage`, and treat milestones as version-scope
  signals rather than release-policy decisions.
- Added the repo-only skill package `.agents/skills/issue-triage/` with a
  validated `SKILL.md`, `agents/openai.yaml`, and rationale-comment reference.
- Created live GitHub labels `state/accepted`, `state/needs-info`, and
  `state/deferred`, created milestone `v0.2.2`, and backfilled every open
  issue with the new triage system.
- Left a short rationale comment on each triaged issue and cleaned up the one
  accidental duplicate comment created during the backfill retry.
- Followed up on PR feedback by rewriting the ownership rule in general form:
  `harness-*` names identify the easyharness-managed distributed pack, while
  other skill names stay repo-owned local development skills unless promoted
  later.
- Clarified in `docs/issue-triage.md` that the current label system is tracked
  in repository docs plus live GitHub metadata rather than an already-shipped
  `.github` label-sync contract.

### Not Delivered

- No `priority/*` label system or project-board/custom-field workflow was
  introduced in this slice.
- No release-cadence, release-bar, or urgent-fix policy was decided beyond the
  narrow rule that concrete milestones represent intended version scope.
- No GitHub automation or bot-driven triage was added.

### Follow-Up Issues

- #87 Define a release policy and cadence beyond the current alpha workflow
- #159 Revisit whether backlog triage needs priority labels or project fields
  after the lightweight rollout
