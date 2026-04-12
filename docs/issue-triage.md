# GitHub Issue Triage

This repository keeps GitHub issue triage intentionally lightweight. The goal
is to make backlog decisions legible without introducing a heavy project board
or a large label taxonomy.

## Core Rules

- Keep the default GitHub type labels as the main issue-kind surface:
  `bug`, `enhancement`, `documentation`, and `question` when it fits.
- Use at most one `state/*` label on an open issue.
- Use concrete version milestones such as `v0.2.2` only when the issue is
  truly accepted into that release scope.
- Treat any applied `state/*` label or concrete milestone as "triaged".
- Close issues that are not planned instead of leaving them open with a
  negative state label.

## Tracked Convention vs GitHub Metadata

The labels and milestones themselves live as GitHub repository metadata, not as
files that GitHub automatically reads from `.github/`.

For this slice, the tracked source of truth for the label system is this
document plus the repo-local `issue-triage` skill. The repository does not yet
ship a `.github`-backed label-sync workflow or a file such as
`.github/labels.yml` that automatically manages label state. If the team later
wants file-backed label automation, treat that as a follow-up workflow change
rather than assuming `.github/` already enforces the current labels.

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

- Good: `v0.2.2`, `v0.2.3`, `v0.3.0`
- Avoid using milestones as a generic backlog bin.

A milestone means "this issue belongs to the intended scope of that version"
and should be more specific than `state/accepted`. It does not by itself mean
the release is ready to cut, nor does it define the release cadence or quality
bar. Those release-policy questions remain separate work, currently tracked by
issue `#87`.

## Required Rationale Comments

Whenever an issue is first triaged or later changes triage state, leave a
short comment that records:

- the judgment
- the main reason for that judgment
- what would cause the issue to be revisited, when that is useful

This comment is required when:

- adding a `state/*` label for the first time
- changing from one `state/*` label to another
- moving an issue from a `state/*` label into a concrete milestone
- removing a concrete milestone in favor of another open-backlog state
- closing an issue as not planned after a real triage decision

Short is fine. The point is to leave a durable breadcrumb for later backlog
sweeps, especially for `state/deferred` and `state/needs-info`.

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
Triaged into milestone v0.2.2. This belongs in the next patch release because
it improves an already-shipped maintainer workflow without widening scope into
larger release-policy work.
```

## Sweep Guidance

When revisiting backlog issues:

- read the prior rationale comment before changing labels
- prefer updating the existing issue over creating duplicate follow-ups
- move `state/deferred` or `state/needs-info` to `state/accepted` or a
  milestone only when the earlier blocking reason no longer applies

If the earlier rationale still stands, leave the issue in place and update the
comment only when a new fact materially changes the story.
