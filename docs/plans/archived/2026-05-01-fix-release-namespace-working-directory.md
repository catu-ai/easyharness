---
template_version: 0.2.0
created_at: "2026-05-01T22:23:00+08:00"
approved_at: "2026-05-01T23:18:04+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/actions/runs/25217687431
size: XS
---

# Fix release namespace verification working directory

## Goal

Repair the `v0.3.0` release automation after the release bump landed. The
`v0.3.0` tag and GitHub Release assets were created, but the `Release` workflow
failed in `Verify published release namespace` before Homebrew formula update
and Homebrew install verification could run.

The failure was not in artifact build or upload. The namespace smoke test ran
from the workflow root checkout and saw Go `1.24.13` with `GOTOOLCHAIN=local`,
while the repository now requires Go `1.25.0`. The release job already checks
out the tagged source at `dist/release-source` and configures Go from
`dist/release-source/go.mod`, so the live namespace verification should run
from that release-source checkout like the preceding release tests do.

## Scope

### In Scope

- Update `.github/workflows/release.yml` so `Verify published release
  namespace` runs from `dist/release-source`.
- Add focused smoke coverage that anchors the release workflow wiring for that
  live namespace verification step.
- Leave a clear merge handoff that the `Release` workflow should be rerun for
  `v0.3.0` after this fix merges, so it can clobber the already-created
  assets, verify the namespace, update the Homebrew formula, and run Homebrew
  install verification.

### Out of Scope

- Rebuilding or replacing the `v0.3.0` tag manually.
- Deleting the already-published `v0.3.0` GitHub Release or assets.
- Changing release artifact contents, checksum semantics, Homebrew formula
  rendering logic, or release version policy.
- Broad refactors of the release workflow.

## Acceptance Criteria

- [x] The release workflow runs the live namespace smoke test from
      `dist/release-source`.
- [x] Focused smoke coverage fails before the workflow wiring fix and passes
      after it.
- [x] The fix candidate is ready for a dedicated PR; post-archive PR CI must
      pass before the candidate is treated as waiting for merge approval.
- [x] The merge handoff clearly says to rerun the `Release` workflow for
      `v0.3.0` after the fix lands, and explains that any remaining release or
      Homebrew failure should become a new explicit handoff.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Run namespace verification from release source

- Done: [x]

#### Objective

Patch the release workflow and its smoke coverage so live namespace
verification uses the tagged release-source checkout and its configured Go
toolchain.

#### Details

Keep the fix narrow. The workflow already runs `Build embedded UI assets` and
`Run tests` from `dist/release-source`; apply the same working-directory
discipline to `Verify published release namespace`. Update the existing
release workflow smoke test so future workflow edits preserve that placement.

#### Expected Files

- `.github/workflows/release.yml`
- `tests/smoke/homebrew_formula_test.go`
- `docs/plans/active/2026-05-01-fix-release-namespace-working-directory.md`

#### Validation

- Run the focused smoke test that checks release/Homebrew workflow wiring.
- Run `harness plan lint` for this plan before approval and again before
  archive.

#### Execution Notes

Added focused smoke coverage to `TestReleaseWorkflowWiresHomebrewTapPublishing`
that requires the `Verify published release namespace` step to run with
`working-directory: dist/release-source`. The test failed before the workflow
change, proving the regression. Updated `.github/workflows/release.yml` to run
that live namespace smoke from `dist/release-source`. Focused validation then
passed with `go test ./tests/smoke -run TestReleaseWorkflowWiresHomebrewTapPublishing -count=1`
and `ruby -e 'require "yaml"; YAML.load_file(ARGV[0]); puts "yaml-ok #{ARGV[0]}"' .github/workflows/release.yml`.

#### Review Notes

Step-closeout review `review-001-delta` passed with 0 blocking and 0
non-blocking findings. The `correctness` reviewer confirmed the workflow now
runs namespace verification from `dist/release-source` without disturbing
adjacent release/Homebrew steps. The `tests` reviewer confirmed the smoke
coverage would catch the missing-working-directory regression and that the
focused validation is appropriate for this narrow workflow-only fix.

### Step 2: Prepare the 0.3.0 release rerun handoff

- Done: [x]

#### Objective

Make the post-merge rerun requirement explicit before the fix waits for merge
approval.

#### Details

The first release run already created tag `v0.3.0` and uploaded the release
assets. The workflow's publish step is designed to upload with `--clobber`
when the release already exists, so rerunning the workflow for the same tag is
the intended recovery path after this fix merges. Do not create a new version,
rewrite the tag, or rerun the release workflow before explicit merge approval.

#### Expected Files

- none, unless a rerun exposes a new code or workflow issue that requires
  reopening before archive

#### Validation

- Confirm the archived plan and PR body identify the required post-merge
  `Release` workflow rerun for `v0.3.0`.
- Confirm the handoff says any remaining release/Homebrew failure should be
  captured as a new explicit follow-up.

#### Execution Notes

Prepared the merge handoff for the already-published `v0.3.0` release:
after this fix merges, rerun the `Release` workflow for `v0.3.0`; the existing
publish step will upload to the already-created release with `--clobber`, then
the fixed namespace verification can run from `dist/release-source`, followed
by Homebrew formula update and Homebrew install verification. If that rerun
still fails, capture the remaining release/Homebrew failure as a new explicit
handoff rather than rewriting the tag or creating a new version.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only records the post-merge release rerun
handoff and does not change code beyond the already reviewed Step 1 fix.

## Validation Strategy

- Use focused workflow smoke coverage for the code change.
- Use the PR body and archive summary as the release recovery handoff before
  merge approval.
- Keep the already-published `v0.3.0` release artifacts intact and rely on the
  workflow's clobber behavior during rerun.

## Risks

- Risk: The release workflow could keep using a non-release checkout for live
  verification.
  - Mitigation: add a smoke assertion tying the namespace verification step to
    `working-directory: dist/release-source`.
- Risk: Rerunning the release could duplicate or corrupt assets.
  - Mitigation: preserve the existing workflow behavior that uploads to an
    existing release with `--clobber`.
- Risk: Fixing namespace verification may reveal a later Homebrew failure.
  - Mitigation: keep rerun verification in scope and record any remaining
    failure as an explicit handoff instead of treating the release as complete.

## Validation Summary

- Red/green validation proved the focused workflow smoke test failed before
  the workflow fix and passed after it:
  `go test ./tests/smoke -run TestReleaseWorkflowWiresHomebrewTapPublishing -count=1`.
- `.github/workflows/release.yml` parsed successfully with
  `ruby -e 'require "yaml"; YAML.load_file(ARGV[0]); puts "yaml-ok #{ARGV[0]}"' .github/workflows/release.yml`.
- `harness plan lint docs/plans/active/2026-05-01-fix-release-namespace-working-directory.md`
  passed before approval and again before archive.

## Review Summary

- Step-closeout review `review-001-delta` passed with 0 blocking and 0
  non-blocking findings across `correctness` and `tests`.
- Finalize review `review-002-full` passed with 0 blocking and 0 non-blocking
  findings across `release_safety` and `handoff_quality`.
- Reviewers confirmed the workflow fix keeps release recovery on the
  already-published `v0.3.0` tag/assets and that the post-merge rerun handoff
  is clear.

## Archive Summary

- Archived At: 2026-05-01T23:26:35+08:00
- Revision: 1
- PR: pending archive publication. After archive, push branch
  `codex/fix-release-namespace-working-dir` and open the dedicated fix PR.
- Ready: the candidate is ready for PR publication; it narrowly adds
  `working-directory: dist/release-source` to the live namespace verification
  step and anchors that wiring with focused smoke coverage.
- Merge Handoff: after the fix PR merges, rerun the `Release` workflow for
  `v0.3.0`. Do not rewrite `v0.3.0` or delete the already-published release
  assets; rely on the workflow's existing `--clobber` upload path. If that
  rerun still fails in release or Homebrew verification, capture the remaining
  failure as a new explicit handoff.

## Outcome Summary

### Delivered

- `.github/workflows/release.yml` now runs `Verify published release namespace`
  from `dist/release-source`, matching the tagged checkout and Go toolchain
  configured earlier in the release job.
- `tests/smoke/homebrew_formula_test.go` now asserts that release workflow
  wiring so the missing-working-directory regression is caught by focused
  smoke coverage.
- The plan records the `v0.3.0` recovery path without manual tag rewriting or
  asset deletion.

### Not Delivered

- The `Release` workflow for `v0.3.0` was not rerun before merge approval;
  rerun is intentionally post-merge handoff work for this fix.
- No release assets, Homebrew formula rendering logic, release tag, or release
  version policy changed.

### Follow-Up Issues

NONE
