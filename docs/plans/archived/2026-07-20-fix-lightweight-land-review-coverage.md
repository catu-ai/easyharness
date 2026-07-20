---
template_version: 0.3.0
created_at: "2026-07-20T21:49:42+08:00"
approved_at: "2026-07-20T21:51:14+08:00"
source_type: direct_request
source_refs:
    - user request
size: S
---

# Fix lightweight land review coverage

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb repository-facing normative content into formal tracked locations before
archive, and record supplement absorption in Closeout. Lightweight plans should
normally avoid supplements. -->

## Goal

Make `harness land` complete successfully for a correctly archived lightweight
candidate whose publish or fresh sync evidence records an immutable commit.
Preserve review-coverage guarantees without requiring the ignored local
lightweight archive snapshot to exist in a Git tree.

### Decisions and Constraints

- Lightweight archive snapshots remain command-owned local runtime state and
  must not become tracked repository artifacts.
- Published-candidate validation must remain fail-closed for unreviewed product
  changes, rewritten candidate drift, missing active-plan deletion, and
  supplement drift.
- Standard workflow validation keeps its existing tracked archive semantics.
- This repair does not add compatibility shims, evidence fallbacks, or manual
  state-recovery behavior.

## Scope

### In Scope

- Distinguish standard and lightweight archive mechanics when validating an
  immutable published candidate during merge handoff.
- Validate the local lightweight archive snapshot against the reviewed plan
  while validating only repository-visible lightweight transitions in Git.
- Cover ordinary and base-aware published candidate validation, including
  lightweight plan supplements.
- Exercise the lightweight workflow through successful land entry and land
  completion with commit-bearing evidence.

### Out of Scope

- Changing where lightweight plans are archived.
- Tracking lightweight archive snapshots or adding a second tracked archive.
- Changing publish, CI, or sync evidence schemas.
- Migrating or rewriting existing standard archived plans.
- Automating recovery of already hand-edited command-owned runtime state.

## Acceptance Criteria

- [x] A valid lightweight candidate with commit-bearing publish evidence can
      enter and complete land without looking for its `.local` archive path in
      the published Git tree.
- [x] Lightweight land still rejects a published candidate that retains the
      reviewed active plan, changes reviewed plan or supplement content
      outside allowed archive mechanics, or introduces an unreviewed product
      delta.
- [x] Base-aware validation accepts an equivalent rewritten lightweight
      candidate and rejects candidate-owned drift.
- [x] Standard published-candidate and land behavior remains unchanged.
- [x] Automated coverage includes a full lightweight lifecycle through
      `land complete` with an immutable published commit.

## Review Focus

- Challenge the split between local snapshot integrity and immutable Git-tree
  validation: no ignored `.local` path may be required from a commit, and no
  repository-visible transition may escape review coverage.
- Verify lightweight supplements obey the same local-target/tracked-source
  boundary as the plan itself.
- Check both ancestry-preserving and base-aware rewritten-candidate paths for
  accidental weakening of standard validation.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Make published coverage profile-aware

- Done: [x]
- Outcome: Published review coverage models lightweight archive mechanics as a
  local snapshot plus repository-visible source removal, while retaining the
  existing tracked target model for standard plans.
- Covers: Acceptance criteria 1, 2, 3, and 4.
- Check: Focused review-coverage and lifecycle tests exercise valid and invalid
  standard and lightweight candidates.

### Step 2: Close the lifecycle regression gap

- Done: [x]
- Outcome: The built CLI proves a commit-bearing lightweight candidate can
  progress from archive through completed land bookkeeping.
- Covers: Acceptance criterion 5.
- Check: The lightweight E2E and the relevant package suites pass from a clean
  candidate.

## Validation Strategy

- Run focused unit tests for review coverage and lifecycle land readiness,
  including standard, lightweight, supplements, ancestry-preserving, and
  base-aware cases.
- Run the built-binary lightweight E2E through `land complete` with populated
  publish commit evidence.
- Run the full Go test suite and plan lint before finalize review.

## Closeout

- Archived At: 2026-07-20T22:13:14+08:00
- Revision: 1
- Validation: `go test ./... -count=1`, focused review-coverage and lifecycle
  suites, the commit-bearing lightweight `land complete` E2E, and
  `git diff --check` passed.
- Review: Integrated full review requested one fail-closed repair for archive
  targets already present in a reviewed or base tree; linked delta
  `review-002-delta` verified the repair, resolved the finding, and passed with
  no new findings.
- Delivered: Profile-aware published coverage now validates lightweight local
  plan and supplement snapshots separately from Git-visible source removals,
  rejects tracked `.local` archive targets directly, preserves standard and
  base-aware validation, and exercises the complete lightweight land lifecycle.
- Not Delivered: No archive-storage, evidence-schema, migration, or manual
  runtime-state recovery changes.
- Follow-Up Issues: NONE
