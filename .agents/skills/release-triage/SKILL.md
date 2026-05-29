---
name: release-triage
description: >-
  Proactively use after every easyharness PR lands, and when checking milestone
  closeout or release scope, to decide whether to recommend that the human open
  a patch or minor release PR. This repo-local skill is advisory only: do not
  bump VERSION, create release PRs, publish releases, or create release-process
  machinery.
---

# Release Triage

## Overview

Use this repo-local skill proactively after every `easyharness` PR lands, and
when reviewing whether recent work, milestone closeout, or release-scope
changes are ready to ship as a patch or minor release. The skill is advisory:
it helps the agent recommend a next release action to the human, but it does
not authorize publishing, version bumps, issue closure, or milestone creation
by itself.

Read `docs/releasing.md` first. That file is the policy source for the public
0.x release line, including minor versus patch shape, optional patch
milestones, and release-readiness judgment.

## Inputs

- the relevant merged PRs, closed issues, or active milestone
- current `VERSION`
- the latest GitHub Release, when release recency matters
- `docs/releasing.md`
- any known release blockers, failed CI, or post-publication repair context

## Workflow

1. Identify the question: recent land follow-up, milestone closeout, patch
   consideration, minor release consideration, or release repair.
2. Read `docs/releasing.md` before making a recommendation.
3. Inspect the relevant GitHub issue or milestone:
   - for minor candidates, check whether the milestone represents one coherent
     user promise and whether its release-critical issues are closed
   - for patch candidates, check whether the work is a small repair,
     same-theme fast-follow, documentation/CI correction, or another low-risk
     improvement worth shipping without waiting for a later minor
4. Check whether required CI or known blocker information is already available.
   If it is not available, say what needs checking rather than pretending the
   release is ready.
5. Recommend exactly one next action:
   - `No release recommended yet`
   - `Consider a patch release PR`
   - `Consider a minor release PR`
   - `Shape or adjust the milestone first`
6. Explain the recommendation in a short paragraph with concrete references to
   the issue, milestone, PR, or release policy that drove the judgment.

## Recommendation Guidance

Recommend `No release recommended yet` when the merged work is internal,
incomplete, blocked, too small to matter on its own, or better batched with
nearby work.

Recommend `Consider a patch release PR` when `main` contains a small repair,
low-risk fast-follow, or same-theme improvement that users should get before
the next minor. Do not require a patch milestone unless the patch needs an
explicit coordinated bucket.

Recommend `Consider a minor release PR` when the target milestone's
release-critical issues are complete and together deliver a coherent
user-facing promise. Minor releases should feel like a complete first version
of the promise, not only a specification or a half-finished feature family.

Recommend `Shape or adjust the milestone first` when the issue list does not
yet match the release promise: the milestone includes broad backlog ideas,
misses release-critical work, carries open blockers, or needs clearer
must-deliver versus follow-up boundaries in issue bodies.

## Guardrails

- Do not edit `VERSION`.
- Do not create a release PR.
- Do not create, close, or retitle milestones unless the human explicitly asks
  for that action.
- Do not publish, rerun, or repair a release.
- Do not add priority labels, project fields, or tracker issues.
- Do not require patch releases to have milestones.
- Do not treat this skill as part of the distributed easyharness-managed skill
  pack. It belongs under `.agents/skills/` only unless the repository later
  makes a deliberate promotion decision.
