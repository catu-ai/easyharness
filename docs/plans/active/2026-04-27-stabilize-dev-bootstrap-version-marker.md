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

- [ ] A dev binary with `mode: dev` and `version: v0.2.5-dev` keeps rendering
      managed bootstrap version markers as `dev`.
- [ ] Release binaries still render managed bootstrap markers from their
      concrete release version.
- [ ] `harness init --dry-run` is all noop for the current clean dogfood
      bootstrap outputs after reinstalling the dev binary.
- [ ] `harness status` no longer emits the false stale bootstrap warning in the
      current idle worktree after reinstalling the dev binary.
- [ ] Focused tests cover the dev-build-with-version regression.

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

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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
