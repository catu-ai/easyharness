---
template_version: 0.2.0
created_at: "2026-05-29T00:04:04+08:00"
approved_at: "2026-05-29T00:05:40+08:00"
source_type: issue
source_refs:
    - '#87'
size: S
---

# Document 0.x release policy and release-readiness triage

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #87 by documenting the repository's current 0.x release policy:
minor releases represent coherent user-facing promises, patch releases can be
lightweight fast-follows or repairs, and version bumps happen only in dedicated
release PRs after the intended scope is ready.

Add one repo-local `release-triage` skill that helps future agents decide,
after merged work or milestone closeout, whether to recommend no release, a
patch release PR, a minor release PR, or more milestone shaping. The skill is a
repository helper only; it should not become part of the distributed
easyharness-managed skill pack.

## Scope

### In Scope

- Update `docs/releasing.md` with a concise release policy for the public 0.x
  line.
- Clarify that ordinary implementation PRs do not bump `VERSION`; only a
  dedicated release PR bumps the tracked release source of truth.
- Define minor releases as coherent user promises, usually represented by a
  concrete GitHub milestone whose release-critical issues are complete.
- Define patch releases as small repairs, fast-follows, or same-theme
  incremental improvements that do not require a patch milestone by default.
- State that milestones are release promises, not feature-family backlogs, and
  that issue bodies may carry lightweight context for must-deliver,
  fast-follow, and later/not-now distinctions when useful.
- Clarify the release bar as required CI plus maintainer judgment about scope,
  blockers, and publication readiness, rather than duplicating CI command lists
  as manual ceremony.
- Update repo-local issue triage guidance so future triage points to the new
  release policy instead of treating #87 as unresolved.
- Update issue #87's live GitHub title/body or closing handoff so the issue no
  longer frames the work as alpha-only.
- Add a repo-local `.agents/skills/release-triage/SKILL.md` for release
  readiness recommendation after meaningful land, milestone closeout, or
  release-scope review.

### Out of Scope

- Changing release automation, `VERSION` semantics, tag creation, GitHub
  Actions workflows, release asset packaging, or Homebrew publishing behavior.
- Adding priority labels, GitHub Projects, custom fields, a long-running
  release tracker issue, or a new roadmap document for customization
  follow-ups.
- Requiring patch releases to use GitHub milestones.
- Closing #87 before the tracked policy change has landed.
- Adding `release-triage` to `assets/bootstrap/` or the distributed
  easyharness-managed skill pack.
- Reworking current `v0.5.0` scope or splitting customization issues as part
  of this slice.

## Acceptance Criteria

- [x] `docs/releasing.md` explains how `easyharness` chooses minor versus
      patch releases during the public 0.x line.
- [x] The policy states that release PRs, not ordinary feature/fix PRs, bump
      `VERSION`.
- [x] The policy states that minor milestones hold the current release promise
      and should contain only release-critical issues, while related
      follow-ups can remain ordinary triaged issues until selected for a later
      release.
- [x] The policy allows routine patch releases without patch milestones while
      preserving the option to create a patch milestone when a patch needs an
      explicit bucket.
- [x] The policy describes release readiness as required CI plus maintainer
      judgment about scope completeness, known blockers, and post-publication
      verification.
- [x] Repo-local issue triage guidance no longer says release-cadence policy is
      still tracked by #87 after this work lands.
- [x] A repo-local `release-triage` skill exists, is not easyharness-managed,
      and tells future agents how to recommend release readiness without
      automatically bumping versions, creating milestones, or performing
      release work.
- [x] Issue #87 is updated or the archive/publish handoff clearly instructs
      the PR to close it after merge.

## Deferred Items

- Any future release automation that detects closed milestones or opens
  release PRs automatically.
- Any future priority label system if the GitHub backlog later becomes hard to
  steer with milestones, issue bodies, and maintainer judgment alone.
- Any customization-specific issue splitting or `v0.5.0` milestone refinement
  beyond using the new policy as context.

## Work Breakdown

### Step 1: Document the 0.x release policy

- Done: [x]

#### Objective

Make the current release-selection rules durable in `docs/releasing.md` and
align repo-local triage wording with the new policy.

#### Details

The policy should reflect the discovery decisions:

- `easyharness` no longer ships alpha-only releases; the policy is for the
  public 0.x line.
- Minor releases should deliver a coherent user-facing promise, not a complete
  feature universe. A minor can be built from several implementation PRs and
  should usually be represented by a GitHub milestone.
- Patch releases can include bugfixes, release repairs, documentation or CI
  corrections, and small same-theme follow-ups. They do not need patch
  milestones by default.
- GitHub milestones should contain the issues that are critical for that
  release promise. They should not become backlogs for every related idea.
- Issue bodies may carry lightweight context such as "must deliver",
  "likely fast-follow", and "later/not now" when that helps maintainers choose
  future issues, but the repository should not add priority labels or a
  separate tracker workflow in this slice.
- Required CI is the mechanical release bar. Maintainer judgment still decides
  whether the release scope is complete, whether blockers remain, and whether
  publication verification needs repair.

Update `.agents/skills/issue-triage/SKILL.md` so it points to the documented
release policy once this work lands. Because this skill is repo-owned local
material, edit `.agents/skills/` directly and do not sync bootstrap assets.

#### Expected Files

- `docs/releasing.md`
- `.agents/skills/issue-triage/SKILL.md`

#### Validation

- Reread `docs/releasing.md` against issue #87's requested topics:
  release timing shape, validation/release bar, urgent fixes, stable/public
  line language, and milestone/patch interaction.
- Confirm the issue-triage skill no longer describes #87 as the active source
  of release-policy truth after the policy has been documented.

#### Execution Notes

Updated `docs/releasing.md` with a new `Release Policy` section for the public
0.x line. It now distinguishes change-driven releases from calendar-driven
cadence, keeps `VERSION` bumps in dedicated release PRs, defines minor releases
as coherent user promises, allows routine patch releases without patch
milestones, treats milestones as release promises rather than feature-family
backlogs, and frames release readiness as required CI plus maintainer judgment.

Updated `.agents/skills/issue-triage/SKILL.md` so milestone triage points to
`docs/releasing.md` for release policy instead of describing #87 as unresolved.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 is text-only policy documentation and repo-local
skill wording; finalize review will cover the combined docs/skill change.

### Step 2: Add repo-local release readiness triage skill

- Done: [x]

#### Objective

Create a small repo-local skill that future agents can use to recommend
whether maintainers should consider a release PR after land, milestone
closeout, or release-scope review.

#### Details

The skill should be advisory and lightweight. It should tell agents to inspect
the relevant GitHub milestone, recently merged work, closed issues, release
policy, and known blockers, then report one of a small set of recommendations:

- no release recommended yet
- consider a patch release PR
- consider a minor release PR
- shape or adjust a milestone before deciding

It must not instruct agents to automatically bump `VERSION`, open release PRs,
create milestones, close issues, or perform release publication. It should also
avoid adding priority labels or tracker issues. The skill belongs only in this
repository's local skill set and must not include easyharness-managed metadata.

#### Expected Files

- `.agents/skills/release-triage/SKILL.md`

#### Validation

- Confirm the skill references `docs/releasing.md` as the policy source.
- Confirm the skill is repo-local only and does not appear under
  `assets/bootstrap/`.
- Reread the skill as a cold agent and verify it can produce an advisory
  release-readiness recommendation without hidden chat context.

#### Execution Notes

Added `.agents/skills/release-triage/SKILL.md` as a repo-local advisory skill.
It tells future agents to read `docs/releasing.md`, inspect relevant merged
work, milestones, issues, CI/blocker context, and then recommend one of four
outcomes: no release yet, consider a patch release PR, consider a minor release
PR, or shape/adjust the milestone first. The guardrails explicitly forbid
editing `VERSION`, opening release PRs, publishing releases, creating required
patch milestones, adding priority labels, or treating the skill as distributed
easyharness-managed material.

Revision 2 finalize-fix updated the skill frontmatter `description` to carry
the proactive trigger condition directly in discoverable metadata: use after
every `easyharness` PR lands, and when checking milestone closeout or release
scope, to decide whether to recommend that the human open a patch or minor
release PR. The body overview was aligned with the same proactive trigger.

Revision 3 finalize-fix converted the frontmatter `description` to folded YAML
style after CI showed the unquoted colon in the proactive advisory sentence
broke skill YAML parsing during bootstrap dogfood checks.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 adds one repo-local text skill with no runtime
behavior; finalize review will cover whether the advisory boundaries are clear.

### Step 3: Update #87 handoff and validate the plan package

- Done: [x]

#### Objective

Make the live issue and tracked plan package consistent enough for execution,
review, and eventual PR closeout.

#### Details

During execution, update issue #87's live title/body or add a clear rationale
comment so the issue no longer frames the missing policy as "beyond alpha" and
so maintainers can see that the intended resolution is the tracked
`docs/releasing.md` policy plus the repo-local release-triage skill.

If the implementation reaches archive before merge, the archive/publish
handoff should make the closeout explicit: the PR should close #87 after the
tracked policy lands. Do not close #87 while the policy is still only a local
candidate.

#### Expected Files

- `docs/plans/active/2026-05-29-document-0-x-release-policy.md`
- GitHub issue #87

#### Validation

- Run `harness plan lint` on the active plan after edits.
- Run documentation or formatting checks that are locally relevant to changed
  Markdown/skill files.
- Confirm `git status --short` shows only intentional plan/docs/skill changes.

#### Execution Notes

Updated live GitHub issue #87 to `Define release policy for the public 0.x
line` and replaced the alpha-only body with the current 0.x release-policy
scope. Validation so far:

- `harness plan lint docs/plans/active/2026-05-29-document-0-x-release-policy.md`
- `git diff --check`

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 is issue text and plan validation bookkeeping;
finalize review will cover the complete candidate before archive.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Reread the changed release policy and skills against #87 and this plan's
  acceptance criteria.
- Run any existing lightweight Markdown/docs checks if the repository exposes
  one; otherwise record that the validation is review-based because the slice
  is documentation and repo-local skill text only.

## Risks

- Risk: The policy becomes too bureaucratic and slows normal patch releases.
  - Mitigation: State that patch milestones are optional and that release PRs
    remain the only version-bump mechanism.
- Risk: The policy under-specifies release readiness and future agents
  recommend releases too eagerly.
  - Mitigation: Require advisory recommendations to check CI, scope
    completeness, known blockers, and maintainer judgment without
    auto-publishing.
- Risk: The new skill is mistaken for distributed easyharness behavior.
  - Mitigation: Keep it under `.agents/skills/`, omit managed metadata, and
    explicitly say it is repo-local.

## Validation Summary

Validated the documentation and repo-local skill candidate with:

- `harness plan lint docs/plans/active/2026-05-29-document-0-x-release-policy.md`
- `git diff --check`
- live #87 verification with `gh issue view 87 --json number,title,state,labels,body,url`
- revision 1 remote PR CI through GitHub Actions `Go Test`
- revision 1 local `go test ./...`
- revision 3 `scripts/sync-bootstrap-assets --check`

The slice is text-only. No Go, UI, release workflow, or packaging behavior
changed in revisions 2 or 3, so finalize-fix validation focused on plan lint,
whitespace validation, bootstrap skill parsing, and delta review of the skill
trigger metadata.

## Review Summary

Finalize full review `review-001-full` passed with 0 findings at revision 1.
Reviewer slots:

- `policy_consistency`: no findings; confirmed the release policy,
  issue-triage handoff, release-triage skill, active plan, and live issue #87
  framing agree without stale alpha assumptions or rejected process machinery.
- `agent_ux`: no findings; confirmed the repo-local release-triage skill is
  advisory, points to `docs/releasing.md`, and includes guardrails against
  version bumps, release PR creation, publication, milestone churn, priority
  labels, tracker issues, and distributed-skill confusion.

Finalize repair delta review `review-002-delta` passed with 0 findings at
revision 2. The reviewer confirmed the `release-triage` frontmatter now carries
the proactive discoverability trigger after every `easyharness` PR lands, while
preserving advisory-only guardrails and accurately recording the repair in the
active plan.

Revision 3 fixed YAML formatting for that same frontmatter description after
remote CI caught the parse failure. The follow-up review should confirm the
proactive trigger remains discoverable and the YAML shape is valid.

Finalize repair delta review `review-003-delta` passed with 0 findings at
revision 3. The reviewer confirmed the folded YAML frontmatter parses, the
proactive trigger remains discoverable in skill metadata, advisory-only
guardrails are preserved, and the active plan accurately records the CI parse
failure plus fix.

## Archive Summary

- Archived At: 2026-05-29T22:52:01+08:00
- Revision: 3
- PR: Open from branch `codex/document-0-x-release-policy` after archive.
- Ready: Documentation, repo-local skills, live issue #87 handoff, plan lint,
  whitespace validation, bootstrap dogfood check, finalize full review,
  revision 2 delta repair review, and revision 3 YAML-format delta review are
  complete.
- Merge Handoff: After PR publication, record publish evidence, refresh CI/sync
  evidence, and wait for human merge approval once `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added the public 0.x release policy to `docs/releasing.md`, including
  change-driven release timing, dedicated release PR version bumps, minor
  release promises, optional patch milestones, milestone scope boundaries, and
  release-readiness judgment.
- Updated the repo-local `issue-triage` skill so milestone triage points to
  `docs/releasing.md` for release policy instead of treating #87 as unresolved.
- Added the repo-local `release-triage` skill for advisory release-readiness
  recommendations after land, milestone closeout, or release-scope review.
- Updated the `release-triage` frontmatter description so skill discovery can
  proactively trigger it after every `easyharness` PR lands and when checking
  milestone closeout or release scope.
- Formatted the proactive frontmatter description as folded YAML so
  repo-local skill parsing stays valid during bootstrap dogfood checks.
- Updated live issue #87 to remove alpha-only framing and point at the tracked
  policy plus repo-local release-triage skill as the intended resolution.

### Not Delivered

- No release automation, `VERSION` behavior, GitHub Actions workflows,
  packaging, or Homebrew publishing behavior changed.
- No priority labels, GitHub Projects, tracker issues, or roadmap documents
  were added.
- No patch milestone requirement was introduced.
- No `v0.5.0` customization scope or issue splitting was performed.

### Follow-Up Issues

- No new GitHub follow-up issue was created. Deferred items are optional future
  expansions already named in this plan: release automation, possible future
  priority labels if the backlog outgrows the current system, and separate
  customization milestone refinement if maintainers choose to pursue it later.
