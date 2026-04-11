---
template_version: 0.2.0
created_at: "2026-04-08T22:27:00+08:00"
source_type: direct_request
source_refs:
    - chat://current-session
size: S
---

# Add Explicit Global Fallback Control To Dev Wrapper Install

## Goal

Adjust `scripts/install-dev-harness` so parallel easyharness worktrees can keep
their current worktree-local behavior without letting an arbitrary linked
worktree become the default fallback used from unrelated repositories.

The installer should keep the worktree-aware wrapper model, add an explicit
`--global` path for refreshing the fallback used outside easyharness source
trees, and keep easyharness source trees pinned to their own local
`./.local/bin/harness` binaries. This slice also resolves issue `#31` by
locking the default wrapper install directory policy to `~/.local/bin` and
documenting that development installs require that directory on `PATH`.

## Scope

### In Scope

- Add explicit `--global` installer behavior for updating the out-of-tree
  fallback binary used by the wrapper.
- Keep easyharness source-tree detection authoritative so any easyharness
  checkout still prefers its own local binary and does not silently run the
  global fallback.
- Update installer smoke coverage for the new `--global` behavior and the
  revised fallback rules.
- Update README install guidance to state the `~/.local/bin` default and the
  development PATH prerequisite.
- Close the open decision tracked by issue `#31` via docs and installer policy
  updates in this slice.

### Out of Scope

- Changing release/distribution channels such as GitHub Releases or Homebrew.
- Replacing the wrapper model with a standalone globally installed binary.
- Broad PATH scanning, `GOBIN` detection, or new automatic install-location
  heuristics beyond the chosen `~/.local/bin` default and existing explicit
  `--install-dir` override.

## Acceptance Criteria

- [x] `scripts/install-dev-harness` accepts `--global` and only updates the
      out-of-tree fallback binary when that flag is present.
- [x] The installed wrapper continues to use the current easyharness source
      tree's `./.local/bin/harness` whenever the current directory resolves to
      an easyharness checkout, even without Git metadata.
- [x] Outside easyharness source trees, the wrapper uses the explicit global
      fallback when one has been installed and reports a clear actionable error
      when none exists.
- [x] The default wrapper install directory policy is documented as
      `~/.local/bin`, with `--install-dir` remaining the explicit override.
- [x] Smoke coverage demonstrates both the worktree-local preference and the
      explicit global fallback behavior.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Rework installer fallback ownership

- Done: [x]

#### Objective

Refactor `scripts/install-dev-harness` and the generated wrapper so global
fallback refresh becomes explicit with `--global` while easyharness source
trees remain strictly local-binary-first.

#### Details

The wrapper should keep its existing source-tree detection approach: if the
current directory belongs to an easyharness checkout, resolve
`<repo>/.local/bin/harness` and fail locally if that binary is missing. Only
when the current directory is not part of an easyharness source tree should the
wrapper attempt the installed global fallback path. The installer should stop
rewriting the out-of-tree fallback on every run; instead, `--global` should be
required to refresh that fallback and normal installs should leave it alone.

#### Expected Files

- `scripts/install-dev-harness`

#### Validation

- `bash -n scripts/install-dev-harness`
- Manual or automated verification that a normal install leaves the previously
  configured global fallback untouched.
- Automated verification that `--global` refreshes the fallback path used
  outside easyharness source trees.

#### Execution Notes

Added `--global` to `scripts/install-dev-harness`, moved the wrapper's
out-of-tree fallback to a dedicated user-level path, and changed ordinary
installs so they no longer overwrite that fallback implicitly. The wrapper now
always prefers the current easyharness source tree's `.local/bin/harness` and
only uses the global fallback when invoked outside an easyharness source tree.
Validated with `bash -n scripts/install-dev-harness`, a local rerun of
`scripts/install-dev-harness`, and the installer-focused smoke suite.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 shipped as one bounded installer slice
and were reviewed together at the candidate level instead of forcing an
artificial mid-slice review boundary.

### Step 2: Cover behavior and document the policy

- Done: [x]

#### Objective

Update smoke coverage and README guidance so the new installer contract and the
`#31` default-directory decision are explicit and testable.

#### Details

Tests should cover source-tree detection with and without Git metadata, confirm
that easyharness source trees never silently use the global fallback, and prove
that unrelated repositories can use the global fallback after an explicit
`--global` install. README updates should state that development installs place
the wrapper in `~/.local/bin` by default, require that directory on `PATH`, and
use `--install-dir` only for explicit overrides.

#### Expected Files

- `tests/smoke/install_dev_harness_test.go`
- `README.md`

#### Validation

- `go test ./tests/smoke/... -count=1`
- Any narrower targeted test invocation needed if the smoke package layout
  requires a different package path.
- Review the README install section for consistency with the new CLI behavior.

#### Execution Notes

Updated installer smoke coverage for explicit `--global` fallback setup,
outside-source-tree failure without a global fallback, preservation of an
existing global fallback on ordinary installs, and refusal to use the global
fallback from inside an easyharness source tree without a local binary. README
now states that development installs default to `~/.local/bin`, require that
directory on `PATH`, and use `--global` to refresh the fallback consumed
outside easyharness source trees. Validated with
`go test ./tests/smoke -run InstallDevHarness -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 is inseparable from Step 1's installer contract
change, so the controller deferred review to the integrated branch candidate.

## Validation Strategy

- Run shell validation for the installer script.
- Run the installer smoke suite covering wrapper refresh and fallback behavior.
- Re-run `scripts/install-dev-harness` after script changes before relying on
  direct `harness` commands in this worktree.

## Risks

- Risk: The wrapper could accidentally fall back to the global binary from
  inside an easyharness checkout and hide missing local installs.
  - Mitigation: Keep source-tree detection ahead of fallback resolution and add
    smoke coverage for both Git and non-Git source-tree layouts.
- Risk: The new `--global` flow could make the external fallback hard to
  discover or stale.
  - Mitigation: Document the flag clearly in README/help text and ensure the
    wrapper emits a clear error when no global fallback has been installed.

## Validation Summary

- `bash -n scripts/install-dev-harness`
- `scripts/install-dev-harness`
- `go test ./tests/smoke -run InstallDevHarness -count=1`

## Review Summary

- `review-001-full` requested one blocking tests finding about missing
  precedence coverage when both a worktree-local binary and a global fallback
  are present.
- Added the missing precedence assertion to
  `TestInstallDevHarnessWrapperDispatchesToCurrentWorktree` and reran
  `go test ./tests/smoke -run InstallDevHarness -count=1`.
- `review-002-delta` passed cleanly for the bounded review-fix.
- `review-003-full` passed with one non-blocking tests note about missing smoke
  coverage for the default `scripts/install-dev-harness --global` path.
- Reopened in `finalize-fix` mode for revision `2`, added the missing default
  `--global` smoke coverage, and reran
  `go test ./tests/smoke -run InstallDevHarness -count=1`.
- `review-004-full` then requested two blocking findings: validate
  wrapper-vs-fallback path conflicts before mutating the global fallback, and
  prove that `--global` refreshes an already-populated stale fallback.
- The repair now validates the wrapper target before writing the fallback and
  adds smoke coverage for stale-fallback refresh plus the conflict path.
- `review-005-delta` passed cleanly for that bounded follow-up.

## Archive Summary

- Archived At: 2026-04-08T23:22:48+08:00
- Revision: 2
- PR: existing PR `https://github.com/catu-ai/easyharness/pull/120` stays open
  for the reopened finalize-fix candidate.
- Ready: `review-005-delta` passed cleanly for the bounded revision-2 repair,
  all acceptance criteria remain satisfied, and the candidate is ready to
  re-enter publish/merge handoff after re-archive.
- Merge Handoff: Re-archive this plan, push the revision-2 repair to the
  existing branch and PR, refresh publish/CI/sync evidence, and wait for merge
  approval once status returns to `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added explicit `--global` control to `scripts/install-dev-harness` so normal
  dev installs no longer overwrite the out-of-tree fallback binary
  implicitly.
- Kept easyharness source-tree detection authoritative so any easyharness
  checkout still prefers its own `.local/bin/harness`, while unrelated
  repositories can use the explicitly installed global fallback.
- Locked the default wrapper install directory policy to `~/.local/bin` and
  documented the required PATH setup in `README.md`.
- Expanded installer smoke coverage for the explicit global fallback contract,
  preservation of an existing global fallback on ordinary installs,
  worktree-vs-global precedence, and the default
  `scripts/install-dev-harness --global` path.

### Not Delivered

None.

### Follow-Up Issues

NONE
