---
name: issue-triage
description: Triage GitHub issues for the easyharness repository by following the tracked policy in docs/issue-triage.md. Use when reviewing a new issue, backfilling labels on existing issues, revisiting deferred or needs-info backlog items, or deciding whether an issue should stay open, move into a concrete version milestone, or close as not planned.
---

# Issue Triage

## Overview

Use this skill only for `easyharness` repository backlog work. Follow the
policy in [docs/issue-triage.md](../../../docs/issue-triage.md)
and leave a short rationale comment whenever you first triage an issue or
change its triage state later.

## Workflow

1. Read the issue body, relevant comments, and any linked plan or release
   context before choosing labels.
2. Apply or correct the default GitHub type label when the issue clearly fits
   `bug`, `enhancement`, `documentation`, or `question`.
3. Use `docs/issue-triage.md` as the policy source of truth to decide whether
   the issue belongs in a `state/*` label, a concrete version milestone such as
   `v0.x.y`, or a close-as-not-planned outcome.
4. Close the issue as not planned when that is the honest outcome instead of
   inventing another open-state label.
5. Leave a short rationale comment that records the judgment, the main reason,
   and what would cause a revisit when that matters.
6. When revisiting `state/deferred` or `state/needs-info`, read the earlier
   rationale comment first and update the state only when the earlier reason no
   longer holds.

## Rationale Comments

Use the comment guidance in `docs/issue-triage.md`. Keep the comment short and
specific. Future backlog sweeps should be able to answer two questions from it:

- why was this state chosen then?
- what would make it reasonable to revisit now?

## Guardrails

- Keep this skill repo-local. Do not add `easyharness-managed` metadata or move
  it into `assets/bootstrap/` unless the repository explicitly decides to ship
  it later.
- Do not duplicate the policy rules here when `docs/issue-triage.md` already
  states them; update the policy doc first when the rule changes.
- Do not rely on labels alone when a short rationale comment would prevent
  future confusion.
