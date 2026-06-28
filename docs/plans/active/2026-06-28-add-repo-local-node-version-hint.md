---
template_version: 0.2.0
created_at: "2026-06-28T11:28:23+08:00"
approved_at: "2026-06-28T11:29:51+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/266
size: XXS
---

# Add Repo-Local Node Version Hint

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Add the smallest repo-local hint that tells contributors and agents to use
Node 22 for easyharness development and release validation, closing issue
#266 without changing release behavior.

The expected implementation is a root `.nvmrc` containing the Node major `22`,
plus short documentation that explains the Corepack-backed pnpm flow already
pinned by `web/package.json`. This plan is intentionally standard workflow:
the work is sized `XXS`, but the human did not request
`workflow_profile: lightweight`.

Revision 3 reopened the archived candidate after PR #268 landed the
contract-sync hardening that had temporarily lived in PR #267. The repair
merges the latest `main` and keeps PR #267 focused on the Node version hint.

## Scope

### In Scope

- Add a root `.nvmrc` that selects Node 22.
- Update contributor documentation so local setup says to use Node 22 and then
  rely on Corepack to honor `web/package.json`'s `pnpm@10.32.1` pin.
- Update release/contributor baseline wording if needed so release validation
  guidance matches the new repo-local hint.
- Keep the change narrow and issue-focused.

### Out of Scope

- Changing release behavior, release scripts, CI workflows, or validation
  gates.
- Adding compatibility machinery for multiple Node versions.
- Adding multiple tool-version conventions such as `.node-version`,
  `.tool-versions`, Volta, or mise unless implementation finds that `.nvmrc`
  cannot fit this repository.
- Changing frontend dependencies, pnpm lockfiles, Vite config, or installer
  smoke behavior.
- Carrying the contract-sync validation-stability fix that landed separately in
  PR #268.

## Acceptance Criteria

- [x] The repository root contains a committed `.nvmrc` that clearly selects
      Node 22.
- [x] `docs/development.md` tells contributors to use the repo-local Node 22
      hint and to use Corepack/pnpm through the existing `web/package.json`
      package-manager pin.
- [x] `docs/releasing.md` no longer leaves the Node major implicit in the
      contributor baseline; it names Node 22 consistently with `.nvmrc`.
- [x] The diff contains no release-script, CI-workflow, dependency, or
      generated-asset changes.
- [x] Issue #266 can be closed by the PR body or archive summary.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Add Node 22 Hint And Docs

- Done: [x]

#### Objective

Add the root Node version hint and align local development/release docs with
the Node 22 plus Corepack/pnpm convention.

#### Details

Issue #266 came from the `0.5.1` patch-release cleanup: local release
validation differed when the developer machine used Node 24, while CI and the
release profile were aligned around Node 22. The prior release plan deferred
this tooling hint so the patch release could remain minimal.

Use `.nvmrc` rather than introducing several parallel tool selectors. The
value should pin the expected Node major, `22`, so patch-level Node updates do
not require repo churn. Keep pnpm versioning delegated to the existing
`packageManager` field in `web/package.json`.

#### Expected Files

- `.nvmrc`
- `docs/development.md`
- `docs/releasing.md`
- this tracked plan, including its archived form after closeout

#### Validation

- `git diff --check`
- `harness plan lint docs/plans/active/2026-06-28-add-repo-local-node-version-hint.md`
- `git diff --name-status origin/main...HEAD`
- If local Node tooling is available, optionally confirm `nvm use` selects
  Node 22 and Corepack resolves the pinned pnpm version from `web/package.json`.

#### Execution Notes

Added a root `.nvmrc` with Node major `22`, updated `docs/development.md` to
point contributors at Node 22 plus Corepack-backed pnpm resolution from
`web/package.json`, and updated `docs/releasing.md` so the contributor baseline
names Node 22 explicitly. TDD was not applicable because this slice changes
documentation and a tool-version hint rather than executable behavior.

Revision 3 merged `origin/main` after PR #268 landed the contract-sync
hardening, then removed the duplicate contract-sync test/change from PR #267 so
this candidate returns to the approved Node hint scope.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This single-step XXS documentation/tooling hint is
covered by the required finalize full review for the complete candidate.

## Validation Strategy

This is a documentation and tool-version hint change, so validation is mostly
static: lint the plan, check whitespace with `git diff --check`, and review the
diff to confirm it only touches the hint and docs. Do not run release
validation unless implementation unexpectedly changes executable release or UI
build behavior.

Revision 3 is a finalize-scope cleanup after PR #268 landed. Validate that the
remaining diff against `origin/main` contains only the Node hint, contributor
docs, release docs, and this plan.

## Risks

- Risk: Adding more than one version-management convention could create
  conflicting contributor guidance.
  - Mitigation: Use only `.nvmrc` unless a concrete repository need appears.
- Risk: Pinning an exact Node patch could create avoidable maintenance churn.
  - Mitigation: Pin the Node major `22`, matching the issue's intent.
- Risk: Documentation could imply pnpm should be installed separately from the
  repository's package-manager pin.
  - Mitigation: Point contributors at Corepack and the existing
  `web/package.json` pnpm pin.

## Validation Summary

- `git diff --check` passed.
- `harness plan lint docs/plans/active/2026-06-28-add-repo-local-node-version-hint.md`
  passed after implementation and closeout updates.
- `nvm use` was not run because `nvm` is not available in this non-interactive
  shell; the repo-level hint is still static and reviewable through `.nvmrc`.
- Revision 1 diff review confirmed the Node hint candidate touched only
  `.nvmrc`, `docs/development.md`, `docs/releasing.md`, and this tracked plan.
- PR #267 CI failed in `scripts/validate` because `contract-sync` scanned
  `web/node_modules` while loading Go comments.
- Revision 2 focused validation passed:
  `go test ./internal/contractsync ./tests/smoke -count=1`,
  `scripts/sync-contract-artifacts --check`, and `git diff --check`.
- Revision 2 full validation passed with `scripts/validate`.
- Revision 3 merged `origin/main` after PR #268 landed the contract-sync fix and
  removed duplicate contract-sync scope from PR #267.
- Revision 3 validation confirmed `git diff --name-status origin/main...HEAD`
  contains only `.nvmrc`, `docs/development.md`, `docs/releasing.md`, and this
  tracked plan, and `git diff --check` passed.

## Review Summary

- Finalize full review `review-001-full` passed with 0 blocking findings and
  0 non-blocking findings.
- `docs-consistency` found no defects and confirmed the Node 22 hint,
  development guidance, release baseline, workflow pins, and plan scope align.
- `risk-scan` found no leaked scope or release/tooling hazards; the candidate
  does not touch release scripts, CI workflows, dependencies, lockfiles, or
  generated assets.
- Revision 2 finalize full review `review-002-full` passed with 0 blocking
  findings and 0 non-blocking findings across `correctness`, `tests`, and
  `risk-scan`.
- Revision 3 finalize review `review-003-full` found one blocking
  docs-consistency issue: `docs/releasing.md` overstated CI/release pnpm
  resolution as Corepack-from-`web/package.json`, while workflows explicitly
  install the pinned pnpm version.
- Repaired the wording so release docs say CI/release use Node 22 and
  explicitly install pinned pnpm.
- Follow-up review `review-004-full` found the same stale Corepack wording in
  this plan's Outcome Summary; repaired the archive-bound plan text to match
  `docs/releasing.md`.

## Archive Summary

- PR: https://github.com/catu-ai/easyharness/pull/267
- Ready: Revision 3 merged `origin/main` after PR #268 landed and slimmed PR
  #267 back to Node hint scope; fresh finalize review is pending.
- Merge Handoff: After re-archive, commit and push the revision 3 archive move,
  refresh PR #267 publish/CI/sync evidence, and wait for explicit human merge
  approval.

## Outcome Summary

### Delivered

- Added a root `.nvmrc` selecting Node 22.
- Updated `docs/development.md` to tell contributors to use Node 22 and
  Corepack-backed pnpm resolution from `web/package.json`.
- Updated `docs/releasing.md` so the contributor baseline explicitly names
  Node 22 and pinned pnpm installation for CI/release jobs.

### Not Delivered

- No release behavior, CI workflow, dependency, lockfile, Vite, installer
  smoke, or generated-asset changes were made.
- The contract-sync hardening is no longer part of PR #267 because it landed in
  PR #268.

### Follow-Up Issues

NONE
