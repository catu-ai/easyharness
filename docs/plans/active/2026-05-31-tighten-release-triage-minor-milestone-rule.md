---
template_version: 0.2.0
created_at: "2026-05-31T13:32:04+08:00"
approved_at: "2026-05-31T13:33:08+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Tighten Release Triage Minor Milestone Rule

## Goal

Clarify the release-triage contract so agents do not recommend a minor release
only because a landed change touches a public or agent-facing workflow surface.
Minor release recommendations should require a preselected concrete milestone
whose release-critical issues are complete and whose coherent user promise is
ready to ship.

Keep the rule general. The durable guidance should describe the decision order
for future release triage, not encode a one-off example from the conversation.

## Scope

### In Scope

- Strengthen the repo-local `release-triage` skill so minor recommendations
  must be tied to an existing concrete milestone and completed release promise.
- Clarify `docs/releasing.md` so minor releases are not treated as the default
  response to any public contract change.
- Preserve patch-release flexibility for low-risk fixes, repairs,
  documentation or CI corrections, and same-theme follow-ups worth shipping
  before the next minor.
- Validate that the updated guidance is internally consistent and does not
  mention a specific PR as policy.

### Out of Scope

- Creating, retitling, closing, or reshaping GitHub milestones.
- Creating a release PR or changing `VERSION`.
- Changing issue triage rules beyond any release-policy wording needed for
  this slice.
- Adding automation, hard gates, labels, project fields, or release machinery.

## Acceptance Criteria

- [ ] `release-triage` requires `Consider a minor release PR` to be grounded in
      an existing concrete milestone whose release-critical issues are complete.
- [ ] `release-triage` directs single-landed-PR follow-up toward
      `No release recommended yet`, `Consider a patch release PR`, or
      `Shape or adjust the milestone first` unless that PR is already part of
      the completed minor milestone.
- [ ] `docs/releasing.md` says minor releases are selected release promises,
      not automatic version bumps for any public or agent-facing contract
      change.
- [ ] The final wording is general and does not depend on any one PR number,
      title, or recent incident.
- [ ] No release publication, `VERSION` change, or milestone mutation is made.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Tighten the release-triage skill

- Done: [x]

#### Objective

Update `.agents/skills/release-triage/SKILL.md` so the skill's decision order
prevents agents from jumping from "public surface changed" directly to a minor
release recommendation.

#### Details

The skill should make the strict milestone rule explicit:

- minor candidates require an existing concrete version milestone
- the milestone must represent the selected coherent release promise
- release-critical issues in that milestone must be complete
- if the milestone is missing, incomplete, or not clearly connected to the
  landed work, the recommendation should not be `Consider a minor release PR`
- ordinary post-PR triage should decide among no release, patch, or milestone
  shaping unless the landed work completes the selected minor promise

Keep examples and wording generic.

#### Expected Files

- `.agents/skills/release-triage/SKILL.md`

#### Validation

- Read the updated skill for contradiction with `docs/releasing.md`.
- Confirm the skill does not mention a specific PR, incident, or release as the
  reason for the new rule.

#### Execution Notes

Updated the repo-local `release-triage` skill so minor release triage starts
from an existing concrete version milestone and only recommends a minor release
when that selected milestone's release-critical issues are complete. TDD was
not practical because this step changes policy guidance text rather than
runtime behavior; validation was direct wording inspection against the approved
scope.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step is a narrow wording update to a repo-local
skill, and the broader consistency check belongs with the final combined review
after the release policy wording is updated.

### Step 2: Clarify the release policy

- Done: [x]

#### Objective

Update `docs/releasing.md` with concise general language that matches the
strict minor-milestone rule while preserving patch-release flexibility.

#### Details

The policy should explain that a public or agent-facing contract change can be
release-worthy without automatically defining the next minor release. Minor
releases should follow the maintainer-selected release promise represented by a
concrete milestone; changes outside that promise can wait or ship as a patch
when they fit the patch criteria.

#### Expected Files

- `docs/releasing.md`

#### Validation

- Read the release policy and skill together to ensure the normative policy and
  agent workflow guidance agree.
- Confirm no release checklist or publication mechanics are changed.

#### Execution Notes

Clarified `docs/releasing.md` so minor releases are selected release promises,
not automatic version bumps for every public, user-facing, or agent-facing
contract change. Preserved patch flexibility by saying such changes can wait
for a later selected promise or ship as a patch when they fit the patch
criteria. TDD was not practical because this step changes policy guidance text
rather than runtime behavior.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this is a narrow release-policy wording update and will
be covered by the final consistency review for the combined policy and skill
change.

## Validation Strategy

- Run `harness plan lint` before approval.
- After implementation, inspect the changed text directly and run any existing
  lightweight documentation or formatting checks that apply.
- Confirm `git diff` contains only release-policy and repo-local skill wording,
  with no `VERSION`, milestone, release workflow, or unrelated churn.

## Risks

- Risk: The new wording could make patch releases sound too restrictive.
  - Mitigation: Explicitly preserve patch releases for small repairs,
    same-theme fast-follows, and low-risk improvements worth shipping before a
    later minor.
- Risk: The skill and release policy could drift in authority.
  - Mitigation: Keep `docs/releasing.md` as the policy source and make the
    skill point agents at the same strict milestone interpretation.

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
