---
template_version: 0.2.0
created_at: "2026-06-18T23:54:11+08:00"
approved_at: "2026-06-18T23:56:10+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/257
size: M
---

# Speed Up Smoke Validation

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Make routine validation fast and predictable again by separating quick
package-local tests from slower binary-driven smoke coverage, while keeping the
full installer and release confidence checks available on an explicit path.

Issue 257 was opened because `tests/smoke` dominated `go test ./...` during
release validation and could exceed Go's default 10 minute package timeout.
Discovery on 2026-06-18 confirmed the problem still exists: a partial
`go test -json -count=1 -timeout=20m ./tests/smoke` run was stopped after
completed tests had already accumulated 430.24s, and the release-build smoke
tests had not started yet. The five completed installer tests alone accounted
for about 419.75s.

## Scope

### In Scope

- Split the current smoke coverage into an explicit quick/default validation
  path and one or more slow opt-in smoke paths for installer and release
  workflows.
- Keep important installer, wrapper PATH, release archive, and live release
  workflow coverage, but make slow cold-path tests intentional rather than an
  accidental part of every quick validation run.
- Add focused subprocess timeouts to smoke/support helpers so a stuck command
  fails with a useful command-level diagnostic.
- Reduce repeated cold `scripts/install-dev-harness` work where coverage can be
  preserved with lighter fixtures, shared setup, or narrower tests.
- Update CI, release workflow, and docs so agents and humans know which command
  is the ordinary quick validation path and which command runs full slow smoke
  coverage.

### Out of Scope

- Changing `harness` product behavior outside what is necessary to keep tests
  passing after the validation split.
- Preserving old command shapes or compatibility aliases for any test scripts
  introduced by this work.
- Removing installer or release smoke coverage merely to make validation look
  faster.
- Reworking unrelated Playwright UI smoke scripts except where docs need to
  describe the overall validation split accurately.

## Acceptance Criteria

- [ ] The ordinary quick validation command no longer runs the full cold
      installer and release smoke suite by accident.
- [ ] Full slow smoke coverage remains available through an explicit command or
      documented command set.
- [ ] `go test ./...` or its documented replacement no longer requires raising
      the package timeout just to get past the old monolithic `tests/smoke`
      package.
- [ ] Smoke subprocess failures identify the specific slow or stuck command
      instead of surfacing only as a Go package-level timeout.
- [ ] CI and release workflow validation match the documented quick/full split.
- [ ] Documentation no longer describes the current full `tests/smoke` package
      as fast repo-level smoke coverage unless the implementation makes that
      true again.

## Deferred Items

- Broader test taxonomy changes beyond issue 257, such as adding a complete
  end-to-end or resilience suite, remain deferred unless they are the smallest
  clean way to express this validation split.

## Work Breakdown

### Step 1: Define and Implement the Validation Split

- Done: [x]

#### Objective

Separate quick smoke/default validation from slow installer and release smoke
coverage in code and command structure.

#### Details

Use a clean target shape rather than compatibility shims. The exact mechanics
can be chosen during execution, but the preferred shape is:

- keep fast package-local and lightweight smoke coverage in the default path;
- move cold installer and release archive checks into clearly named slow smoke
  packages or build-tagged suites;
- update any workflow references that currently assume live release smoke tests
  live under `./tests/smoke`;
- avoid leaving a monolithic package where unrelated smoke tests block each
  other from package-level parallelism.

The resulting commands should be understandable from the repository alone. If a
new script such as `scripts/test-quick` or `scripts/test-smoke` is introduced,
it should be short, explicit, and tested or smoke-checked where practical.

#### Expected Files

- `tests/smoke/**`
- `tests/installer/**`
- `tests/release/**`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `scripts/test-*` if scripts are the cleanest command surface

#### Validation

- Run the new quick validation command and confirm it excludes the slow cold
  installer/release path.
- Run focused moved-package tests or build-tagged suites to confirm the
  preserved smoke coverage still executes.
- Use `go test -list` or the equivalent command output to verify workflow test
  references still point at real tests.

#### Execution Notes

Implemented the first validation split. Cold installer smoke moved from
`tests/smoke` to `tests/installer` behind `//go:build slow_smoke`; release
script/namespace/Homebrew smoke moved to `tests/release`, and release archive
construction moved behind `slow_smoke` inside that package. Shared command,
fixture, PATH, version-field, and ordering helpers were promoted into
`tests/support` so quick smoke, release smoke, and opt-in installer smoke no
longer depend on one monolithic package.

Updated release workflow live smoke invocations to run the moved release tests
from `./tests/release`. Validation:
`go test -list . ./tests/smoke ./tests/release`,
`go test -tags slow_smoke -list . ./tests/installer ./tests/release`,
`go test ./tests/smoke ./tests/release`, and `go test ./...` all pass. The
default `go test ./...` run completed in about 1:07.51 and no longer executed
the slow installer package or tagged release archive tests.

#### Review Notes

`review-001-delta` passed on 2026-06-19 with `correctness` and `tests`
reviewer slots. Both reviewers reported no findings.

### Step 2: Add Command Timeouts and Reduce Repeated Cold Work

- Done: [x]

#### Objective

Make smoke helpers fail at the command boundary when subprocesses hang, and
reduce avoidable repeated cold installer cost without weakening coverage.

#### Details

The current `tests/support.Run` helper uses `exec.Command` without a context,
and many smoke tests use direct `exec.Command` calls. Introduce a small,
consistent timeout mechanism for harness subprocesses and the slow smoke
helpers. Keep timeout values generous enough for real release/installer work,
but short enough that a stuck command points at the offending command.

Installer smoke should keep at least one true cold `scripts/install-dev-harness`
end-to-end check. Other wrapper PATH behavior tests should avoid rebuilding the
embedded UI and Go binary repeatedly when the same confidence can be achieved
with a prepared binary, a shared fixture, or a smaller wrapper fixture.

#### Expected Files

- `tests/support/run.go`
- `tests/support/binary.go`
- `tests/smoke/**`
- `tests/installer/**`
- `tests/release/**`
- `scripts/install-dev-harness` only if a tiny test-only-neutral improvement is
  needed to make the split clean

#### Validation

- Add or update focused tests that prove timeout diagnostics include the command
  and captured output when a subprocess exceeds its timeout.
- Run the installer-focused slow smoke command and compare elapsed time against
  the issue 257 baseline enough to show repeated cold work was reduced.
- Confirm at least one cold installer path still runs `scripts/install-dev-harness`
  end to end.

#### Execution Notes

Added default command-level timeouts to `tests/support.Run`,
`tests/support.RunWithOptions`, and the generic smoke command helpers. Timeout
failures now include the command plus captured stdout/stderr, and
`tests/support` has a self-spawning regression test that proves the diagnostic
includes timeout text and child process output.

Removed global test environment mutation from installer cache setup and marked
the opt-in installer tests parallel. The full installer slow-smoke package now
passes with `go test -tags slow_smoke ./tests/installer -count=1` in about
3:29.01. That keeps a true cold `scripts/install-dev-harness` path while
materially reducing wall-clock time from the earlier serial baseline, where the
first five installer tests alone had already accumulated roughly 7 minutes.

Validation: `go test ./tests/support ./tests/smoke ./tests/release`,
`go test -tags slow_smoke ./tests/release -run TestBuildReleaseHelpUsesStableExampleVersion -count=1`,
`go test -tags slow_smoke ./tests/installer -count=1`, and `go test ./...`
all pass. The final default `go test ./...` run completed in about 1:30.78
with `tests/e2e` uncached.

#### Review Notes

`review-002-delta` found one blocking tests issue and one minor tests issue:
the timeout regression covered `RunCommandWithTimeout` but not
`RunWithOptions`, and the nested timeout used `50ms`. Repaired both by adding a
`RunWithOptions` timeout regression and increasing the nested timeout to
`500ms`. Follow-up `review-003-delta` passed with no findings.

### Step 3: Update Documentation, Workflows, and Evidence

- Done: [ ]

#### Objective

Make the new validation contract durable in CI, release workflow, and docs, and
record evidence that issue 257's failure mode is addressed.

#### Details

Update docs that currently imply `go test ./tests/smoke -count=1` is fast
repo-level smoke coverage if that is no longer the command shape. The release
checklist should tell maintainers which quick validation to run before merge
and when to run the full slow smoke path.

The CI and release workflows should follow the same split. If full slow smoke
is too expensive for every PR, keep that explicit and leave a documented manual
or release-only command instead of hiding the cost inside `go test ./...`.

#### Expected Files

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `docs/releasing.md`
- `docs/development.md`
- `docs/specs/proposals/testing-structure.md`
- `docs/plans/active/2026-06-18-speed-up-smoke-validation.md`

#### Validation

- Run the documented quick validation command.
- Run the documented slow smoke command or a focused representative subset if
  the full slow command remains intentionally long.
- Run `harness plan lint docs/plans/active/2026-06-18-speed-up-smoke-validation.md`.
- Record elapsed-time evidence in this plan before archive.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Use `go test -list` before and after the split to confirm tests moved into
  the intended command surfaces.
- Run quick validation with elapsed-time output and confirm it avoids the old
  monolithic smoke timeout risk.
- Run the slow smoke command or focused slow suites to prove installer and
  release confidence checks still exist.
- Run workflow/documentation checks as plain file inspection plus relevant Go
  smoke tests that assert workflow wiring where such tests already exist.

## Risks

- Risk: The split could make important installer or release coverage less
  visible.
  - Mitigation: Keep explicit slow smoke commands and update CI/release docs so
    the coverage is opt-in by design, not forgotten.
- Risk: Reducing repeated installer work could accidentally stop testing the
  real cold install path.
  - Mitigation: Preserve at least one true cold installer smoke and convert only
    narrower wrapper behavior checks to lighter fixtures.
- Risk: New timeout values could be too aggressive on slower machines.
  - Mitigation: Choose conservative defaults, include useful diagnostics, and
    allow only intentional opt-in overrides if the repository already has a
    clear pattern for that.

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
