---
name: issue-triage
description: Triage GitHub issues for the easyharness repository using the local backlog policy, concrete version milestones, and required rationale comments. Use when reviewing a new issue, backfilling labels on existing issues, revisiting deferred or needs-info backlog items, or deciding whether an issue should stay open, move into a concrete version milestone, or close as not planned.
---

# Issue Triage

## Overview

Use this skill only for `easyharness` repository backlog work. This skill
package is the self-contained operational contract for how this repository
triages GitHub issues. Do not rely on external docs to recover the rules.

Keep the backlog legible without adding a heavy project board or a large label
taxonomy.

## Core Rules

- Keep the default GitHub type labels as the main issue-kind surface:
  `bug`, `enhancement`, `documentation`, and `question` when it fits.
- Use at most one `state/*` label on an open issue.
- Use concrete version milestones such as `v0.x.y` only when the issue is
  truly accepted into that release scope.
- Treat any applied `state/*` label or concrete milestone as "triaged".
- Close issues that are not planned instead of leaving them open with a
  negative state label.

## Triage States

Use these labels for reviewed issues that are still open but not yet tied to a
concrete version milestone:

- `state/accepted`
  - The issue is worth doing, but it is not yet assigned to a concrete release.
- `state/needs-info`
  - The issue cannot be judged or advanced yet because key information is
    missing.
- `state/deferred`
  - The issue remains worth keeping open, but it should not move now.
    This label intentionally covers both "later" and "not yet mature enough".

If an issue moves into a concrete milestone, remove the `state/*` label rather
than keeping both.

## Milestones

Use milestones for real version intent, not vague release buckets.

- Good: `v0.x.y`, `v0.y.z`, `v1.x.y`
- Avoid using milestones as a generic backlog bin.

A milestone means "this issue belongs to the intended scope of that version"
and should be more specific than `state/accepted`. It does not by itself mean
the release is ready to cut, nor does it define the release cadence or quality
bar. Those release-policy questions remain separate work, currently tracked by
issue `#87`.

## Workflow

1. Read the issue body, relevant comments, and any linked plan or release
   context before choosing labels.
2. Apply or correct the default GitHub type label when the issue clearly fits
   `bug`, `enhancement`, `documentation`, or `question`.
3. Decide whether the issue belongs in a `state/*` label, a concrete version
   milestone such as `v0.x.y`, or a close-as-not-planned outcome.
4. Close the issue as not planned when that is the honest outcome instead of
   inventing another open-state label.
5. Leave a short rationale comment that records the judgment, the main reason,
   and what would cause a revisit when that matters.
6. When revisiting `state/deferred` or `state/needs-info`, read the earlier
   rationale comment first and update the state only when the earlier reason no
   longer holds.

## Rationale Comments

Whenever an issue is first triaged or later changes triage state, leave a short
comment that records:

- the judgment
- the main reason for that judgment
- what would cause the issue to be revisited, when that is useful

This comment is required when:

- adding a `state/*` label for the first time
- changing from one `state/*` label to another
- moving an issue from a `state/*` label into a concrete milestone
- removing a concrete milestone in favor of another open-backlog state
- closing an issue as not planned after a real triage decision

Keep the comment short and specific. Future backlog sweeps should be able to
answer two questions from it:

- why was this state chosen then?
- what would make it reasonable to revisit now?

Example shapes:

```text
Triaged as state/deferred. The idea still looks worthwhile, but the current
repository surface is still moving and I do not want to lock in the wrong
shape yet. Revisit after more dogfooding or when adjacent workflow surfaces
settle.
```

```text
Triaged as state/needs-info. I do not yet have enough evidence about the user
impact and the preferred UX shape. Revisit once there is a concrete workflow
example or a sharper acceptance target.
```

```text
Triaged into milestone v0.x.y. This belongs in the next concrete release scope
because it improves an already-shipped maintainer workflow without widening
scope into broader release-policy work.
```

## Sweep Guidance

When revisiting backlog issues:

- read the prior rationale comment before changing labels
- prefer updating the existing issue over creating duplicate follow-ups
- move `state/deferred` or `state/needs-info` to `state/accepted` or a
  milestone only when the earlier blocking reason no longer applies

If the earlier rationale still stands, leave the issue in place and update the
comment only when a new fact materially changes the story.

## Guardrails

- Keep this skill repo-local. Do not add `easyharness-managed` metadata or move
  it into `assets/bootstrap/` unless the repository explicitly decides to ship
  it later.
- Keep this skill self-contained. If the triage rules change, update this skill
  package first instead of pushing the real policy into an external doc.
- Do not rely on labels alone when a short rationale comment would prevent
  future confusion.
