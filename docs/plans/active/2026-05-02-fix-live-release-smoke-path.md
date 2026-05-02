---
template_version: 0.2.0
created_at: "2026-05-02T22:43:58+08:00"
approved_at: "2026-05-02T22:45:01+08:00"
source_type: direct_request
source_refs:
    - release-v0.3.0
    - https://github.com/catu-ai/easyharness/actions/runs/25253755723
size: S
---

# Fix Live Release Smoke PATH And Checkout

## Goal

Finish the `v0.3.0` release recovery path by fixing the live release smoke
checks that still fail after the release assets are uploaded. The latest manual
rerun of the `Release` workflow for `v0.3.0` failed at
`Verify published release namespace` with:

```text
go.mod requires go >= 1.25.0 (running go 1.24.13; GOTOOLCHAIN=local)
```

The current failure points at test-side PATH construction, not asset creation:
the live GitHub smoke test narrows PATH to the directory containing `gh` plus
the dev harness installer path, which can put an older system `go` ahead of
the Go version installed by `actions/setup-go`. The previous workflow fix also
made namespace verification run from `dist/release-source`, which means a
future main-branch test fix would not affect a rerun for the already-created
`v0.3.0` tag.

## Scope

### In Scope

- Fix live namespace release smoke PATH construction so the setup-go toolchain
  remains available and is not shadowed by the directory containing `gh`.
- Fix live Homebrew tap smoke PATH construction similarly, since it also
  prepends tool directories while running after asset upload.
- Adjust the release workflow so the live namespace verification uses the
  main/root checkout test logic while still validating the `v0.3.0` release
  assets and module namespace.
- Add focused smoke coverage for the workflow wiring and PATH ordering
  behavior.
- Prepare a merge handoff that explicitly reruns the `Release` workflow for
  `v0.3.0` after the fix lands.

### Out of Scope

- Rewriting, deleting, or retagging `v0.3.0`.
- Changing the version number or release artifact contents beyond whatever the
  existing workflow rebuilds with `--clobber`.
- Broad release workflow refactors unrelated to live verification.
- Changing Homebrew formula generation or publishing behavior unless a new
  validation failure proves that surface is also broken.

## Acceptance Criteria

- [ ] The namespace live smoke test preserves the setup-go PATH and cannot
      accidentally choose an older system `go` solely because `gh` lives in
      that directory.
- [ ] The Homebrew live smoke test uses the same PATH-preserving approach for
      `brew` and `gh`.
- [ ] The `Release` workflow runs namespace verification from the main/root
      checkout test logic while still targeting the requested release version
      and uploaded assets.
- [ ] Focused tests or smoke assertions cover the PATH-ordering and workflow
      checkout expectations.
- [ ] PR CI passes, the candidate is archived, and the workflow is ready to
      wait for human merge approval.
- [ ] The merge handoff records that the post-merge action is to rerun
      `Release` on `main` with `version=v0.3.0` and inspect any remaining live
      failure separately.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Correct live verification PATH and checkout wiring

- Done: [x]

#### Objective

Make release live smoke checks use the intended Go toolchain and the current
main-branch test logic when validating already-published release assets.

#### Details

The implementation should avoid a fragile PATH overwrite in the live smoke
tests. It may introduce a small helper if that keeps the namespace and Homebrew
tests consistent, but should keep the change local to release smoke surfaces.
The workflow should continue to build release artifacts from the requested
release source while running the namespace verification logic from the root
checkout so a main-branch fix applies to a `v0.3.0` rerun.

#### Expected Files

- `.github/workflows/release.yml`
- `tests/smoke/verify_release_namespace_test.go`
- `tests/smoke/homebrew_formula_test.go`

#### Validation

- Add or update focused tests that fail for the old PATH or workflow behavior
  and pass with the corrected behavior.
- Run the relevant smoke test package targets locally without requiring live
  GitHub credentials.
- Parse the release workflow YAML after editing.

#### Execution Notes

Implemented a release-smoke PATH helper that places the current setup-go Go
and pnpm directories before live external tool directories, then preserves the
runner PATH. Updated the namespace and Homebrew live smoke tests to use that
helper, removed the namespace verification `working-directory:
dist/release-source` so reruns use root/main checkout test logic, and added
coverage for both the PATH ordering and workflow checkout expectation.
Validated with:
`ruby -e 'require "yaml"; YAML.load_file(ARGV[0]); puts "ok"' .github/workflows/release.yml`,
`go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestReleaseSmokePathKeepsSetupGoAheadOfExternalTools' -count=1`,
and `go test ./tests/smoke -count=1`.

#### Review Notes

`review-001-delta` passed with 0 blocking and 0 non-blocking findings across
the `release_correctness` and `tests` slots. Reviewers confirmed the workflow
keeps tag-based build/publish behavior while root checkout verification applies
main-branch smoke fixes, and that the smoke coverage catches the old PATH
shadowing and checkout-wiring behavior.

### Step 2: Prepare release rerun handoff

- Done: [x]

#### Objective

Leave the archived candidate and PR handoff ready for human merge approval,
with a concrete post-merge rerun command for `v0.3.0`.

#### Details

Do not attempt to rerun the live `Release` workflow before this fix is merged
to `main`, because the current failure occurs in the workflow/test logic used
by the live run. The PR body or archive notes should make clear that the
existing `v0.3.0` tag and release assets are kept, and that the workflow's
existing upload path may clobber assets during the rerun.

#### Expected Files

- `docs/plans/active/2026-05-02-fix-live-release-smoke-path.md`

#### Validation

- Confirm the candidate reaches `execution/finalize/await_merge`.
- Confirm PR CI is green before treating it as ready for merge approval.

#### Execution Notes

Prepared the release rerun handoff in this plan: keep the existing `v0.3.0`
tag and release assets, merge the workflow/test fix to `main`, then rerun
`gh workflow run release.yml --ref main -f version=v0.3.0`. The rerun may
clobber existing release assets through the workflow's existing upload behavior.
Do not run the live `Release` workflow before the fix lands on `main`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only records the release rerun handoff. The
handoff will be included in archive/publish notes and covered by finalize
review.

## Validation Strategy

- Run focused Go smoke tests covering release workflow wiring and PATH
  construction, for example the release workflow smoke test and the namespace
  and Homebrew live-smoke helper tests.
- Run a YAML parse check for `.github/workflows/release.yml`.
- Use harness step and finalize reviews with release-safety attention before
  archive.
- After merge approval and landing, rerun:

```bash
gh workflow run release.yml --ref main -f version=v0.3.0
```

## Risks

- Risk: Running verification from the wrong checkout could leave a main-only
  fix unused by the `v0.3.0` rerun.
  - Mitigation: Add workflow wiring coverage that distinguishes the root
    checkout from `dist/release-source` for namespace verification.
- Risk: PATH fixes could hide missing tool dependencies by preserving too much
  of the runner environment.
  - Mitigation: Keep explicit checks for required tools while preserving the
    existing setup-go PATH order.
- Risk: The Homebrew live smoke may expose another issue after the namespace
  check passes.
  - Mitigation: Fix its identical PATH-risk pattern in this slice and record
    any new failure from the post-merge rerun as separate evidence.

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

NONE
