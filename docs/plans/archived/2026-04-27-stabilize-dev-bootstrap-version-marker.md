---
template_version: 0.2.0
created_at: "2026-04-27T09:21:53+08:00"
approved_at: "2026-04-27T09:23:20+08:00"
source_type: direct_request
source_refs: []
size: XS
---

# Stabilize Dev Bootstrap Version Marker

## Goal

Fix the false bootstrap drift warning produced by dev-built `harness` binaries
that expose a `vX.Y.Z-dev` build version. The direct `harness --version`
metadata should keep reporting the dev build version for diagnostics, while
managed bootstrap instructions and skill packages continue to use the stable
`dev` marker in development mode.

## Scope

### In Scope

- Keep bootstrap managed asset rendering stable when the running binary is in
  dev mode, even if its build metadata includes a concrete `vX.Y.Z-dev`
  version.
- Add focused regression coverage for a dev build that has both `Mode: dev`
  and a non-empty `Version`.
- Reinstall the repo-local dev binary after the Go CLI change and verify the
  real `harness` command no longer reports bootstrap drift on the clean
  dogfood outputs.

### Out of Scope

- Changing the public `harness --version` output shape or removing dev build
  version metadata.
- Refreshing managed bootstrap files to `vX.Y.Z-dev` markers.
- Changing release-mode bootstrap markers.
- Redesigning the worktree-aware development wrapper.

## Acceptance Criteria

- [x] A dev binary with `mode: dev` and `version: v0.2.5-dev` keeps rendering
      managed bootstrap version markers as `dev`.
- [x] Release binaries still render managed bootstrap markers from their
      concrete release version.
- [x] `harness init --dry-run` is all noop for the current clean dogfood
      bootstrap outputs after reinstalling the dev binary.
- [x] `harness status` no longer emits the false stale bootstrap warning in the
      current idle worktree after reinstalling the dev binary.
- [x] Focused tests cover the dev-build-with-version regression.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Separate Dev Build Metadata From Bootstrap Markers

- Done: [x]

#### Objective

Teach bootstrap marker selection to prefer the stable `dev` marker whenever
the running harness binary is in dev mode.

#### Details

The regression appeared because `scripts/install-dev-harness` now injects
`BuildVersion=vX.Y.Z-dev` for better `harness --version` diagnostics, while
bootstrap marker rendering currently prefers any non-empty version before
considering dev-mode stability. Keep the diagnostic version visible through
`harness --version`, but prevent that diagnostic value from becoming a managed
asset compatibility marker.

#### Expected Files

- `internal/install/service.go`
- `internal/install/service_test.go`

#### Validation

- Add or update focused install tests covering `versioninfo.Info{Mode: "dev",
  Version: "v0.2.5-dev"}`.
- Run `go test ./internal/install -count=1`.

#### Execution Notes

Added a focused regression test that reproduces marker churn when a dev build
has `Mode: dev` and `Version: v0.2.5-dev`, then updated bootstrap marker
selection so dev mode always renders the stable `dev` marker. Focused
validation passed with `go test ./internal/install -run
'TestInitUsesStableDevVersionMarkerWhenDevBuildHasVersion|TestInitUsesStableDevVersionMarkerAcrossCommitChanges|TestInitRefreshesVersionMarkersAcrossVersionChanges'
-count=1`.

#### Review Notes

Step-closeout delta review `review-001-delta` passed with no findings across
`correctness` and `tests`.

### Step 2: Verify Real Dev Binary Behavior

- Done: [x]

#### Objective

Rebuild the repo-local dev binary and verify the real `harness` command no
longer reports false bootstrap drift.

#### Details

Because the user-facing failure is visible through the installed worktree
wrapper and repo-local binary, source-level `go run` validation is not enough.
After the code change, rerun `scripts/install-dev-harness` and validate the
direct command on PATH.

#### Expected Files

- `internal/install/service.go`
- `internal/install/service_test.go`

#### Validation

- Run `scripts/install-dev-harness`.
- Run `harness --version` and confirm it still reports a dev version.
- Run `harness init --dry-run` and confirm all actions are `noop`.
- Run `harness status` and confirm the stale bootstrap warning is absent.
- Run a broader relevant test sweep such as `go test ./internal/status
  ./internal/install -count=1`, with `go test ./... -count=1` preferred when
  time permits.

#### Execution Notes

Reinstalled the repo-local dev binary with `scripts/install-dev-harness`.
Verified the real `harness` command still reports `version: v0.2.5-dev` and
`mode: dev`, while `harness init --dry-run` reports all noop actions and
`harness status` no longer emits the stale bootstrap warning. Validation also
passed with `go test ./internal/status ./internal/install -count=1` and
`go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only rebuilt and validated the dev binary after
the Step 1 code change, and introduced no additional code or contract edits.

## Validation Strategy

Use focused install tests to lock the marker-selection semantics, then validate
the actual repo-local dev binary because this bug only became visible after the
development installer injected build version metadata. Finish by checking the
idle status path and init dry-run path that originally exposed the false
warning.

## Risks

- Risk: Release bootstrap refresh behavior could accidentally lose concrete
  release version markers.
  - Mitigation: Keep existing release-version tests and add only a dev-mode
    special case.
- Risk: Source-level tests could pass while the installed wrapper still uses a
  stale binary.
  - Mitigation: Re-run `scripts/install-dev-harness` after the Go change and
    validate with the actual `harness` command.

## Validation Summary

Validation covered the marker-selection unit path, the status/init drift path,
the installed development binary, and the full Go test suite:

- `go test ./internal/install -run 'TestInitUsesStableDevVersionMarkerWhenDevBuildHasVersion|TestInitUsesStableDevVersionMarkerAcrossCommitChanges|TestInitRefreshesVersionMarkersAcrossVersionChanges' -count=1`
- `go test ./internal/status ./internal/install -count=1`
- `scripts/install-dev-harness`
- `harness --version`
- `harness init --dry-run`
- `harness status`
- `go test ./... -count=1`

## Review Summary

Step-closeout delta review `review-001-delta` passed with no findings across
`correctness` and `tests`. Finalize full review `review-002-full` passed
`correctness` and `tests`, and found one docs-consistency archive-readiness
issue: the durable archive summary sections still contained placeholders. This
revision removed those placeholders and recorded the closeout summaries.
Follow-up finalize full review `review-003-full` passed with no findings.

## Archive Summary

- Archived At: 2026-04-27T09:42:59+08:00
- Revision: 1
The candidate is ready to archive as a standard tracked plan after clean
follow-up finalize review `review-003-full`. There are no deferred items or
follow-up issues for this slice.

- PR: pending post-archive publish handoff from branch
  `codex/stabilize-dev-bootstrap-version-marker`.
- Ready: Acceptance criteria are satisfied, step-closeout review
  `review-001-delta` passed, finalize repair review `review-003-full` passed,
  and validation includes the installed dev binary plus `go test ./... -count=1`.
- Merge Handoff: Archive the active plan, commit the tracked archive move, push
  branch `codex/stabilize-dev-bootstrap-version-marker`, open a PR, and record
  publish/CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Dev-mode bootstrap marker rendering now returns the stable `dev` marker even
  when the dev binary exposes a diagnostic version such as `v0.2.5-dev`.
- Release-mode bootstrap marker rendering continues to use concrete release
  versions.
- Regression coverage now proves the dev-build-with-version case remains noop
  for managed bootstrap assets.
- The repo-local dev binary was rebuilt and verified to keep `harness
  --version` diagnostics while clearing the false `harness status` bootstrap
  drift warning.

### Not Delivered

No planned scope was left undelivered.

### Follow-Up Issues

NONE
