---
template_version: 0.3.0
created_at: "2026-07-14T11:07:14+08:00"
approved_at: "2026-07-14T11:09:19+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/293
size: M
---

# Correct RC workflow edges from dogfood

## Goal

Ship `v0.6.0-rc.2` with the lean review model intact while removing workflow
friction exposed by the first Grove dogfood runs. A complete candidate should
enter review only after its outcomes are accepted, plans should use natural
Markdown without lifecycle ceremony, stale managed resources should be visible,
and an already merged PR should recover into land instead of looking invalid.

### Decisions and Constraints

- Keep one mandatory integrated finalize reviewer and linked repair deltas.
- Prefer direct workflow cues and useful state checks over provenance or
  anti-agent machinery.
- Treat post-review Closeout edits as allowed archive preparation; do not weaken
  reviewed coverage for product, plan-structure, or acceptance changes.
- Warn clearly about managed-resource drift without silently rewriting a user's
  worktree or mixing bootstrap refreshes into an unrelated candidate.
- Keep stable `v0.6.0` and goal-oriented work out of this RC correction.

## Scope

### In Scope

- Require completed acceptance criteria, in addition to completed steps, before
  starting finalize review and reflect the blocker in status guidance.
- Clarify that plan steps and acceptance criteria describe candidate outcomes,
  not review, archive, publish, merge, or land milestones.
- Accept properly indented continuation lines for `Outcome`, `Covers`, and
  `Check`, while retaining deterministic parsing and useful errors for malformed
  step bodies.
- Surface packaged-binary versus repository-managed instruction/skill drift in
  status facts and next actions, including while a plan is active.
- Distinguish a merged recorded PR from a merely closed or invalidated PR and
  guide recovery through `harness land`.
- Add regression coverage, update managed assets/contracts, bump the RC version,
  and prepare release and Homebrew handoff evidence.

### Out of Scope

- Automatically refreshing, rebasing, or modifying stale managed resources.
- Semantic natural-language policing of arbitrary plan prose beyond clear
  lifecycle guidance and enforceable state invariants.
- Removing the mandatory finalize reviewer or reviewed git boundary.
- Stable `v0.6.0`, goal-oriented workflow, or unrelated dashboard redesign.
- Repairing or landing the separate Grove worktrees that exposed these cases.

## Acceptance Criteria

- [x] `harness review start` rejects a plan with any unchecked acceptance
      criterion before creating review state, and status tells the controller
      what remains incomplete.
- [x] The managed planning guidance prevents review/archive/publish/merge/land
      milestones from being authored as ordinary step or acceptance outcomes.
- [x] Canonical indented continuation lines round-trip through plan lint,
      document parsing, progress/status, review handoff, and archive checks;
      malformed or ambiguous step text still fails clearly.
- [x] `harness status` reports when packaged managed assets differ from the
      current repository copy and gives a non-mutating refresh/rebase-oriented
      next action without changing the current lifecycle node.
- [x] A recorded PR observed as `MERGED` receives land-recovery guidance and is
      not classified as a candidate needing publish repair; an unmerged closed
      PR remains invalid.
- [x] Representative Grove dogfood traces are encoded as regression tests,
      including the no-extra-delta happy path and recovery after API merge.
- [x] Managed bootstrap assets, CLI/state contracts, release metadata, and
      Homebrew/release handoff describe `v0.6.0-rc.2` consistently.
- [x] Full repository and release validation pass for the complete candidate.

## Review Focus

- Verify that acceptance gating prevents premature review without introducing
  a new ceremonial plan-write loop.
- Challenge continuation parsing around indentation, blank lines, duplicate
  fields, headings, fenced examples, and archive Closeout boundaries.
- Check managed-resource drift detection for clean, dirty, detached, active-plan,
  and unavailable-resource cases without hidden writes or excessive status cost.
- Verify remote `OPEN`, `CLOSED`, and `MERGED` states remain distinct and that
  land still requires explicit human merge approval and an explicit command.
- Confirm the RC bump and release handoff cannot publish before merge approval.

## Deferred Items

- Stable `v0.6.0` remains a feedback-driven decision under issue #293.
- Automatic managed-resource refresh or cross-worktree rebase remains explicit
  future work if warnings prove insufficient.
- Goal-oriented workflow remains targeted at `v0.7.0`.

## Work Breakdown

### Step 1: Make plan completion and parsing natural

- Done: [x]
- Outcome: Finalize review starts only for a completed candidate, and concise step fields support ordinary indented Markdown wrapping without ambiguous parsing.
- Covers: Acceptance criteria 1, 2, 3, and 6.
- Check: Focused plan, review, status, lifecycle, fuzz, and end-to-end transition tests reproduce the Grove closeout and multiline-plan cases.

### Step 2: Make stale resources and merged handoffs recoverable

- Done: [x]
- Outcome: Status exposes managed-resource drift and treats a merged PR as a land handoff while preserving human authority and closed-PR invalidation.
- Covers: Acceptance criteria 4, 5, and 6.
- Check: Focused status, bootstrap, remote-observation, evidence, and land recovery tests cover current, stale, unavailable, open, closed, and merged states.

### Step 3: Prepare the corrected RC candidate

- Done: [x]
- Outcome: Contracts, managed assets, version metadata, and release handoff consistently describe the corrected lean workflow candidate.
- Covers: Acceptance criteria 7 and 8.
- Check: Bootstrap sync, documentation/schema checks, full repository validation, installer/release validation, and version consistency checks pass.

## Validation Strategy

- Add focused Red/Green coverage for every observed dogfood failure before
  relying on broad suites.
- Exercise both CLI service boundaries and representative end-to-end state
  transitions so prompts and state enforcement agree.
- Sync bootstrap assets from their source, then run full Go, resilience, E2E,
  UI, installer, and release validation appropriate to an RC artifact.
- Use one independent integrated finalize reviewer over the complete committed
  candidate; close only narrow repairs with linked deltas.

## Closeout

- Archived At: 2026-07-14T11:48:18+08:00
- Revision: 1
- Validation: Focused plan, review, status, install, CLI, contract-sync, dashboard, and built-binary E2E tests passed; `go test ./...`, `scripts/validate`, `scripts/validate-release`, bootstrap/contract sync checks, plan lint, and diff checks passed on the complete candidate. The installed development binary reports `v0.6.0-rc.2-dev`.
- Review: Integrated full review `review-001-full` requested two important repairs for the generated reviewer skeleton and ambiguous Markdown continuations. Linked `review-002-delta` exercised the generated skeleton directly, verified the parser boundary repairs, resolved both findings, and passed with no new findings.
- Delivered: Finalize review now requires completed acceptance criteria; plans support ordinary indented continuation prose and discourage lifecycle milestones as outcomes; status surfaces stale managed assets without writes, distinguishes merged PRs for explicit land recovery, and preserves closed-PR invalidation; the directly editable reviewer skeleton now matches its submit input contract; release metadata is `0.6.0-rc.2`.
- Not Delivered: Automatic managed-resource refresh/rebase, stable `v0.6.0`, goal-oriented workflow, and repair of the external Grove dogfood worktrees remain outside this candidate.
- Follow-Up Issues: Continue RC feedback and the stable-release decision in https://github.com/catu-ai/easyharness/issues/293; goal-oriented work remains assigned to milestone `v0.7.0`.
- PR: To be created after archive from `codex/rc2-workflow-corrections`.
- Ready: Acceptance, implementation, release-level validation, bootstrap/schema sync, and full-plus-linked-delta review coverage are complete; the candidate is ready to archive and publish.
- Merge Handoff: Publish a ready PR, record current CI and base-sync evidence, and wait for explicit human merge approval. After merge, verify tag `v0.6.0-rc.2`, GitHub prerelease assets, the default Homebrew formula update, and an installed Homebrew binary reporting the RC version.

