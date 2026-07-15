---
template_version: 0.3.0
created_at: "2026-07-15T10:30:00+08:00"
approved_at: "2026-07-15T09:11:19+08:00"
source_type: direct_request
source_refs:
    - Codex task 019f606f-3398-7840-a444-f7445512e958
size: S
---

# Recover Review Rounds After Rewritten Ancestry

## Goal

Prevent finalize review from entering an unrecoverable linked-delta round after
a rebase or other history rewrite breaks the reviewed ancestry chain. Make the
ordinary command choose a valid full-review boundary before creating artifacts,
and provide a supported way to abandon an unfinished review round without
editing local harness state by hand.

### Decisions and Constraints

- Preserve linked delta review when the prior reviewed head is an ancestor of
  the current committed candidate.
- When prior coverage cannot be extended because ancestry was rewritten,
  ordinary review start automatically chooses a full review instead of creating
  a doomed delta or requiring the controller to diagnose Git topology.
- Add an explicit review-abort operation for the active, unaggregated round.
  Preserve its artifacts and record the abort; clear only the control-plane
  blockage needed to start a replacement round.
- Completed review decisions remain immutable and cannot be aborted.
- Keep RC3 base-aware publish/sync coverage behavior unchanged: an equivalent,
  non-overlapping base sync still preserves review, while overlapping upstream
  edits still require a synchronized-candidate review.

## Scope

### In Scope

- Review-kind inference and preflight validation for rewritten Git ancestry.
- A command-owned recovery path for an active unaggregated review round.
- Status, help, state/artifact contracts, and tests needed to make the new
  behavior discoverable and replayable.
- Bootstrap-managed skill guidance where the workflow instructions need to
  mention the recovery behavior.

### Out of Scope

- Weakening candidate cleanliness, fixed-HEAD, or continuous-coverage checks.
- Preserving linked-delta identity across rewritten commits by patch similarity.
- Changing base-aware publish/sync overlap policy or merge/land behavior.
- Migrating old runtime state or supporting legacy command shapes.

## Acceptance Criteria

- [x] Starting finalize review after a history rewrite that makes the prior
      reviewed head non-ancestral creates a full review boundary before any
      impossible delta round becomes active.
- [x] A normal narrow repair whose current head descends from the prior reviewed
      head still creates and completes a linked delta with existing finding and
      revision semantics intact.
- [x] An active unaggregated review round can be explicitly aborted through the
      CLI; its artifacts and abort fact remain inspectable, and a replacement
      full or inferred review can start without manual state edits.
- [x] Aborting a completed, non-active, or mismatched round fails safely without
      changing finalize coverage or the active round.
- [x] Status/help and managed workflow guidance lead an agent from an invalid or
      abandoned round to the supported next action without suggesting local
      state surgery.
- [x] Unit and built-binary E2E coverage reproduces the rebase deadlock, proves
      the repaired transition, and keeps existing review, publish/sync, and
      archive behavior green.

## Review Focus

- Confirm the automatic full-review fallback cannot silently discard valid
  unresolved finding obligations or weaken whole-candidate coverage.
- Check that abort is atomic across state, round metadata, timeline output, and
  failure rollback, and that it never mutates an aggregated decision.
- Verify the rebase test genuinely rewrites ancestry rather than merely adding
  an upstream merge ancestor.
- Ensure RC3 base-aware non-overlap preservation and overlap rejection remain
  distinct from review-round ancestry recovery.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Select a valid review boundary before round creation

- Done: [x]
- Outcome: Finalize review inference distinguishes extendable ancestry from rewritten history and creates either a linked delta or a full review that can establish valid coverage.
- Covers: Acceptance criteria 1, 2, and 6.
- Check: Exercise narrow-repair and rewritten-ancestry cases through service and built-binary workflow tests.

### Step 2: Add supported unfinished-round recovery

- Done: [x]
- Outcome: Agents can abort an active unaggregated round through a documented command and immediately follow status guidance into a valid replacement review, with artifacts and prior coverage preserved.
- Covers: Acceptance criteria 3, 4, 5, and 6.
- Check: Cover successful abort, invalid targets, rollback behavior, status/help output, schemas, and managed bootstrap synchronization.

## Validation Strategy

- Run focused review service, runstate, CLI, status, contract, and E2E tests for
  both ancestry inference and abort transitions.
- Run bootstrap and generated-contract synchronization checks after changing
  managed guidance or command schemas.
- Reinstall the development harness after Go CLI changes, then run the full Go
  and built-binary E2E suites plus `scripts/validate-release`.
- Complete one independent integrated finalize review against the fixed final
  candidate before archive and publish.

## Closeout

- Archived At: 2026-07-15T09:46:51+08:00
- Revision: 1
- Validation: `scripts/validate-release` passed on the final candidate,
  including the complete Go suite, built-binary E2E, installer and release
  smoke tests, UI build, and schema/bootstrap synchronization checks.
- Review: `review-001-full` passed at
  `50f47a6fb7989ffdd7b83390b415cd6ba2c1a8f8` with no blocking or non-blocking
  findings; the independent reviewer also reran release, race, and targeted
  rebase/abort validation.
- Delivered: Review start now rejects impossible deltas before artifact
  creation, automatically establishes a full root after clean rewritten
  ancestry, preserves unresolved-finding obligations, and provides atomic
  `review abort` recovery with historical UI/timeline visibility and agent
  guidance.
- Not Delivered: Patch-similarity ancestry bridging, legacy runtime-state
  migration, and changes to RC3 publish/sync overlap policy remain outside the
  approved scope.
- Follow-Up Issues: NONE
