---
template_version: 0.2.0
created_at: "2026-05-29T00:05:04+08:00"
approved_at: "2026-05-29T00:05:56+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Tighten Reviewer Prompt Boundaries

## Goal

Refine the managed `harness-reviewer` skill so reviewer subagents give sharper
findings without adding new review machinery. The change should borrow the
useful prompt-level lessons from `autoreview`: report actionable defects tied
to the reviewed change, use the anchor diff as the starting lens, avoid vague
security alarms, and point findings at the smallest useful location.

This plan intentionally does not integrate external review CLIs or add a review
helper. Codex subagents remain the review runtime.

## Scope

### In Scope

- Update the managed reviewer skill source under `assets/bootstrap/`.
- Sync the materialized `.agents/skills/` output from the bootstrap asset.
- Clarify changed-path awareness for `delta` review as a starting point, not a
  hard boundary.
- Clarify that findings outside directly changed files must explain why the
  reviewed change introduced, exposed, or made the issue relevant.
- Clarify high-signal security finding expectations.

### Out of Scope

- No Codex CLI, Claude CLI, or other external review-engine integration.
- No new `harness review` command behavior.
- No schema or aggregate contract changes for category, confidence, or scope
  metadata.
- No automatic out-of-scope finding filtering.
- No first-class review evidence packet design in this slice.

## Acceptance Criteria

- [x] `assets/bootstrap/skills/harness-reviewer/SKILL.md` includes concise
  reviewer guidance for actionable findings, security findings, smallest useful
  locations, and changed-path awareness.
- [x] The synced `.agents/skills/harness-reviewer/SKILL.md` matches the
  bootstrap source after running `scripts/sync-bootstrap-assets`.
- [x] The wording preserves the current reviewer workflow shape: reviewer
  subagents inspect the repo directly and submit through `harness review
  submit`.
- [x] No CLI behavior, schema, or generated contract artifact changes are made.

## Deferred Items

- Open or update a follow-up issue to research first-class review
  evidence/context packets for reviewer handoff.
- Open or update a follow-up issue to research optional finding metadata such
  as category and confidence, likely starting as reviewer `worklog` data before
  any contract decision.

## Work Breakdown

### Step 1: Tighten Managed Reviewer Prompt Guidance

- Done: [x]

#### Objective

Update the managed reviewer skill text with focused prompt rules for
high-signal review findings.

#### Details

Keep the change small and prompt-only. The added guidance should make reviewer
expectations clearer without turning the skill into a long checklist.

The intended behavior is:

- findings should be actionable defects introduced, exposed, or made relevant
  by the reviewed change
- security findings should describe a concrete exploitable risk, removed safety
  check, or missing trust-boundary validation
- locations should point at the smallest useful repo-relative anchor
- `delta` review should begin from the anchor diff and then inspect related
  code when plan intent, contract meaning, or runtime behavior warrants it
- findings in unchanged files should explain their relationship to the reviewed
  change instead of being automatically discarded

#### Expected Files

- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `.agents/skills/harness-reviewer/SKILL.md`

#### Validation

- Run `scripts/sync-bootstrap-assets`.
- Run `harness plan lint docs/plans/active/2026-05-29-tighten-reviewer-prompt-boundaries.md`.
- Inspect the synced diff to confirm only intended prompt text changed.

#### Execution Notes

Updated the managed reviewer skill prompt guidance in
`assets/bootstrap/skills/harness-reviewer/SKILL.md` and synced the materialized
`.agents/skills/harness-reviewer/SKILL.md` output with
`scripts/sync-bootstrap-assets`. The change is prompt-only and does not alter
CLI behavior, schemas, or review artifacts. TDD is not applicable because this
slice changes reviewer instructions rather than executable behavior; validation
used bootstrap sync, plan lint, and diff inspection.

Round `review-001-delta` found one agent UX issue: the phrase "made relevant to
the current plan" could broaden review scope beyond change-tied defects. The
repair changed that sentence to keep relevance tied to the reviewed change and
resynced the materialized skill output.

#### Review Notes

Delta review `review-001-delta` found one important agent UX issue: "made
relevant to the current plan" could broaden scope beyond defects made relevant
by the reviewed change. The wording was repaired and resynced.

Follow-up delta review `review-002-delta` passed with no findings.

## Validation Strategy

- Use plan lint for the tracked plan.
- Use bootstrap sync to verify the materialized skill output matches the
  managed source.
- Use git diff review to confirm this remains a skill-only prompt adjustment.

## Risks

- Risk: The reviewer skill becomes too verbose and increases reviewer cognitive
  load.
  - Mitigation: Keep the new text short and place it near existing severity and
    workflow guidance.
- Risk: Changed-path awareness is misread as a hard scope filter.
  - Mitigation: State explicitly that the anchor diff is a starting lens, not a
    boundary, and that related unchanged-code findings are allowed when
    explained.

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

PENDING_UNTIL_ARCHIVE
