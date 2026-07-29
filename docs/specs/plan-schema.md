---
title: Plan Schema
created_at: "2026-03-17T14:00:00+08:00"
updated_at: "2026-07-28T00:00:00+08:00"
reviewed_at: "2026-07-28T00:00:00+08:00"
status: active
---

# Plan Schema

## Purpose

A tracked plan preserves the decisions a human approved and the progress a
future agent needs. It describes outcomes and boundaries, not predicted files,
implementation recipes, or an execution diary.

The current template version is `0.3.0`. Generate an ordinary root with:

```bash
harness plan template
```

Generate a coordinated root or one of its subplans with:

```bash
harness plan template --coordinated
harness plan template --subplan
```

## Location and Naming

Active and archived roots come from repository configuration. Their defaults
are:

```text
docs/plans/active/
docs/plans/archived/
```

Root plans use `YYYY-MM-DD-short-topic.md` directly under an active or archived
root. A root filename must remain unique across the configured active,
standard-archive, and lightweight-archive roots because its stem is the stable
runtime identity across archive and reopen. Reusing an archived root filename
for new work is invalid; choose a new topic or date-qualified filename instead.
A coordinated root may additionally own flat subplans at:

```text
supplements/<root-plan-stem>/subplans/<subplan-id>.md
```

`<subplan-id>` is a lowercase hyphenated slug. Subplan directories cannot
nest. The supplements directory, `subplans/` directory, and child Markdown
files must be ordinary in-package directories or regular files rather than
symlinks. Other plan Markdown files cannot live below `supplements/`.

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
lightweight plan uses `workflow_profile: lightweight`. A root that coordinates
multiple agent-owned workstreams uses `workflow_profile: coordinated`.
Goal-oriented authoring and execution remain deferred to `v0.7.0`.

## Canonical Body

Standard and lightweight top-level sections appear exactly in this order:

1. `Goal`
2. `Scope`
3. `Acceptance Criteria`
4. `Review Focus`
5. `Deferred Items`
6. `Work Breakdown`
7. `Validation Strategy`
8. `Closeout`

A coordinated root uses the same order but omits `Work Breakdown`. Its subplan
set is the unordered or partially ordered execution breakdown; the root is not
forced to pretend that one child is its sequential current step.

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
work that remains at archive must have a concrete issue URL or `#number`
reference in `Closeout`. Work with no intended follow-up belongs in Out of
Scope with `- None.` under Deferred Items; do not invent an issue for it.

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

`Outcome`, `Covers`, and `Check` may wrap onto immediately following lines
indented by at least two spaces. Continuations are parsed as newline-separated
text in the same field. Keep them as ordinary prose: a blank line ends the
field, and ATX or Setext headings, thematic breaks, blockquotes, fenced blocks,
or nested lists are not accepted as field continuations. List markers separated
from their content by either spaces or tabs are rejected consistently with
CommonMark parsing.

For standard and lightweight roots, the first unfinished `Done` marker is the
current step. Coordinated roots do not have root steps; each subplan applies
the same ordered-step rule locally. Steps are meaningful progress boundaries,
not review units or prescribed implementation sequences. Do not add
expected-file lists, details, execution notes, review notes, or step-local
acceptance checklists.

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
```

Before archive, replace every placeholder with a concise durable result. The
archive command adds `Archived At` and `Revision`. Values may wrap onto
immediately following lines indented by at least two spaces, using the same
ordinary-prose continuation rules as step fields. `Follow-Up Issues` must name
a concrete issue URL or `#number` when deferred items remain. PR, CI, sync,
merge, and land facts belong to harness runtime evidence and forge history,
not the tracked archive-time Closeout.

## Plan Packages

A plan may own durable supporting material under:

```text
supplements/<plan-stem>/
```

For standard and lightweight plans, approval covers the root Markdown and its
matching supplements as one package. Supplements hold bulky approved context,
not scratch work or the only copy of repository-facing normative behavior.

For a coordinated plan, the human approves the root's goal, decisions and
constraints, scope, acceptance criteria, Review Focus, and authority boundary.
Subplans are agent-owned decomposition within that boundary and may be added or
refined without separate approval. A material change to the approved root
contract requires the same renewed human steering as an ordinary plan.

Formal review, archive, publish, and land still cover the complete coordinated
package. Archive moves the root and matching supplements together, and the
archived top level contains only the root plan. Absorb normative content into
code, tests, or formal specs before archive and summarize that absorption in
the root `Closeout`.

Plan-specific final-review input belongs in the Markdown `Review Focus`
section. Supplements do not define additional reviewer guidance, topology, or
selection machinery.

## Coordinated Profile

`workflow_profile: coordinated` represents one candidate-owning root plus zero
or more flat subplans. It does not allow multiple independent root candidates
in one worktree.

A subplan has minimal frontmatter:

```yaml
---
depends_on: [api]
---
```

`depends_on` is optional and names sibling subplan IDs. Missing dependencies,
self-dependencies, duplicate IDs, cycles, and nested subplans are invalid.
With no dependencies, siblings are runnable concurrently.

The body contains exactly:

1. one H1 title
2. `Outcome`
3. `Work Breakdown`
4. `Result`

`Work Breakdown` contains one or more compact ordered steps using the same
`Done`, `Outcome`, `Covers`, and optional `Check` fields as ordinary plan
steps. `Covers` identifies the root acceptance outcome advanced by the child;
the contract intentionally does not invent separate acceptance IDs.

`Result` contains exactly:

```markdown
- Validation: PENDING
- Delivered: PENDING
```

A subplan is complete only when every step is done and both Result values have
been replaced with concise, non-placeholder outcomes. Every existing subplan
is required. Subplans have no separate approval, execute-start, formal review,
archive, publish, or land lifecycle.

## Lightweight Profile

`workflow_profile: lightweight` is available only when a human explicitly
approves it for one bounded, low-risk `XXS` change. It keeps the same compact
plan contract and mandatory steering boundaries, normally uses one step, and
archives its lightweight snapshot beneath the configured local runtime root.

## Lint and Archive Rules

`harness plan lint <path>` validates frontmatter, profile-specific section
order, scope shape, acceptance checkboxes, compact step fields, closeout fields,
filename/location, profile eligibility, and supplement placement. For a
coordinated root or child, lint validates the complete package and dependency
graph.

Archived plans additionally require:

- every acceptance criterion checked
- every ordinary root step done, or every coordinated subplan complete
- no archive placeholder remaining in `Closeout`
- `Archived At`, `Revision`, and all durable closeout fields present
- follow-up issue information when deferred work remains

The schema deliberately does not preserve obsolete step subsection or
goal-oriented preview shapes. This repository is in rapid development and uses
the clean current contract without compatibility shims.
