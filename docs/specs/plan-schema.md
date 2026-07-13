---
title: Plan Schema
created_at: "2026-03-17T14:00:00+08:00"
updated_at: "2026-07-13T23:55:00+08:00"
reviewed_at: "2026-07-13T23:55:00+08:00"
status: active
---

# Plan Schema

## Purpose

A tracked plan preserves the decisions a human approved and the progress a
future agent needs. It describes outcomes and boundaries, not predicted files,
implementation recipes, or an execution diary.

The current template version is `0.3.0`. Generate it with:

```bash
harness plan template
```

## Location and Naming

Active and archived roots come from repository configuration. Their defaults
are:

```text
docs/plans/active/
docs/plans/archived/
```

Standard plans use `YYYY-MM-DD-short-topic.md`. A plan Markdown file cannot
live below a `supplements/` directory.

## Frontmatter

```yaml
---
template_version: 0.3.0
created_at: "2026-07-13T23:30:00+08:00"
approved_at: "2026-07-13T23:35:00+08:00"
source_type: direct_request
source_refs: []
size: M
---
```

Required fields:

- `template_version`: semantic version no newer than the installed template
- `created_at`: RFC3339 creation time
- `source_type`: non-empty origin label
- `source_refs`: source identifiers or URLs, possibly empty
- `size`: one of `XXS`, `XS`, `S`, `M`, `L`, `XL`, or `XXL`

`approved_at` is added only after explicit human approval. Runtime lifecycle,
revision, and update state do not belong in tracked frontmatter.

Standard plans omit `workflow_profile`. An explicitly approved `XXS`
lightweight plan uses `workflow_profile: lightweight`; no other workflow
profile is currently supported. Goal-oriented authoring and execution are
deferred to `v0.7.0`.

## Canonical Body

Top-level sections appear exactly in this order:

1. `Goal`
2. `Scope`
3. `Acceptance Criteria`
4. `Review Focus`
5. `Deferred Items`
6. `Work Breakdown`
7. `Validation Strategy`
8. `Closeout`

### Goal

State the intended outcome. Include one `### Decisions and Constraints`
subsection containing the durable choices, rejected directions, authority
boundaries, or compatibility decisions that execution must preserve.

### Scope

Contain `### In Scope` followed by `### Out of Scope`. These boundaries are the
approved slice; changing them requires renewed human steering.

### Acceptance Criteria

Use one or more Markdown checkboxes. These are the observable completion
contract and the source for acceptance progress.

### Review Focus

Record candidate-specific invariants, failure modes, or questions that the
mandatory final reviewer must examine. This section is additive to the fixed
integrated review rubric and is included automatically; it is not a reviewer
dimension selector.

### Deferred Items

Name work intentionally excluded from the slice, or use `- None.`. Deferred
work that remains at archive must have a concrete follow-up in `Closeout`.

### Work Breakdown

Every plan has at least one outcome boundary:

```markdown
### Step 1: Establish the compact contract

- Done: [ ]
- Outcome: The template and lint contract use the compact plan shape.
- Covers: Acceptance criteria 1 and 2.
- Check: Focused template and lint tests pass.
```

Each step contains only:

- a numbered `### Step N: Title` heading
- `- Done: [ ]` or `- Done: [x]`
- one non-empty `Outcome`
- one non-empty `Covers` reference to the acceptance criteria advanced by the
  outcome
- an optional non-empty `Check`

The first unfinished `Done` marker is the current step. Steps are meaningful
progress boundaries, not review units or prescribed implementation sequences.
Do not add expected-file lists, details, execution notes, review notes, or
step-local acceptance checklists.

### Validation Strategy

Describe the whole-plan evidence needed for confidence. Keep command-level
mechanics in repository skills or tooling unless a command is itself part of
the approved outcome.

### Closeout

The template starts with compact archive-time fields:

```markdown
## Closeout

- Validation: PENDING_UNTIL_ARCHIVE
- Review: PENDING_UNTIL_ARCHIVE
- Delivered: PENDING_UNTIL_ARCHIVE
- Not Delivered: PENDING_UNTIL_ARCHIVE
- Follow-Up Issues: NONE
- PR: PENDING_UNTIL_ARCHIVE
- Ready: PENDING_UNTIL_ARCHIVE
- Merge Handoff: PENDING_UNTIL_ARCHIVE
```

Before archive, replace every placeholder with a concise durable result. The
archive command adds `Archived At` and `Revision`. `PR`, `Ready`, and
`Merge Handoff` preserve the publish boundary; `Follow-Up Issues` cannot remain
`NONE` when deferred items remain.

## Plan Packages

A plan may own durable supporting material under:

```text
supplements/<plan-stem>/
```

Approval covers the Markdown plan and its matching supplements directory as
one package. Supplements hold bulky approved context, not scratch work or the
only copy of repository-facing normative behavior. Absorb normative content
into code, tests, or formal specs before archive and summarize that absorption
in `Closeout`.

Plan-specific final-review input belongs in the Markdown `Review Focus`
section. Supplements do not define additional reviewer guidance, topology, or
selection machinery.

## Lightweight Profile

`workflow_profile: lightweight` is available only when a human explicitly
approves it for one bounded, low-risk `XXS` change. It keeps the same compact
plan contract and mandatory steering boundaries, normally uses one step, and
archives its lightweight snapshot beneath the configured local runtime root.

## Lint and Archive Rules

`harness plan lint <path>` validates frontmatter, section order, scope shape,
acceptance checkboxes, compact step fields, closeout fields, filename/location,
profile eligibility, and supplement placement.

Archived plans additionally require:

- every acceptance criterion checked
- every step done
- no archive placeholder remaining in `Closeout`
- `Archived At`, `Revision`, and all durable closeout fields present
- follow-up issue information when deferred work remains

The schema deliberately does not preserve the pre-`0.3.0` step subsection or
goal-oriented preview shapes. This repository is in rapid development and uses
the clean current contract without compatibility shims.
