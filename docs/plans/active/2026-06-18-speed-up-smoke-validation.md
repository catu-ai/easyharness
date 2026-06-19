---
template_version: 0.2.0
created_at: "2026-06-18T23:54:11+08:00"
approved_at: "2026-06-18T23:56:10+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/257
    - https://github.com/catu-ai/easyharness/pull/259
size: M
---

# Speed Up Smoke Validation

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Make routine validation fast and predictable again by separating ordinary
development validation from full release-ready validation, while keeping the
installer and release confidence checks available through purpose-named,
deterministic entrypoints.

Issue 257 was opened because `tests/smoke` dominated `go test ./...` during
release validation and could exceed Go's default 10 minute package timeout.
Discovery on 2026-06-18 confirmed the problem still exists: a partial
`go test -json -count=1 -timeout=20m ./tests/smoke` run was stopped after
completed tests had already accumulated 430.24s, and the release-build smoke
tests had not started yet. The five completed installer tests alone accounted
for about 419.75s.

Revision 2 reopened the archived PR because the first implementation framed
the split as quick versus slow. That made the release/installer checks
available, but the human clarified that test selection should not depend on a
subjective judgment that a change is "small" or on duration-oriented names
such as `slow_smoke`. The remaining work is to express the split as two clear
validation profiles: ordinary development validation and full release-ready
validation.

## Scope

### In Scope

- Split the current smoke coverage into an explicit quick/default validation
  path and one or more slow opt-in smoke paths for installer and release
  workflows.
- Replace the duration-oriented validation framing with purpose-oriented
  validation profiles: ordinary development validation and full release-ready
  validation.
- Provide simple script entrypoints for those profiles so humans and agents do
  not need to remember build tags or make subjective risk calls for `VERSION`
  and release PRs.
- Rename opt-in build tags away from `slow_smoke` toward functional meaning,
  such as release and installer smoke coverage.
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
- Update release guidance so `VERSION` and release PRs use the release-ready
  validation profile before merge, while the post-merge Release workflow
  remains publish validation.

### Out of Scope

- Changing `harness` product behavior outside what is necessary to keep tests
  passing after the validation split.
- Preserving old command shapes or compatibility aliases for any test scripts
  introduced by this work.
- Removing installer or release smoke coverage merely to make validation look
  faster.
- Reworking unrelated Playwright UI smoke scripts except where docs need to
  describe the overall validation split accurately.
- Adding a complex path-filter matrix that tries to infer every possible
  release or installer surface. The intended rule is simpler: ordinary
  development uses the default profile; release-ready work, including
  `VERSION` PRs, uses the full release profile.

## Acceptance Criteria

- [x] The ordinary quick validation command no longer runs the full cold
      installer and release smoke suite by accident.
- [x] Full slow smoke coverage remains available through an explicit command or
      documented command set.
- [x] `go test ./...` or its documented replacement no longer requires raising
      the package timeout just to get past the old monolithic `tests/smoke`
      package.
- [x] Smoke subprocess failures identify the specific slow or stuck command
      instead of surfacing only as a Go package-level timeout.
- [x] CI and release workflow validation match the documented quick/full split.
- [x] Documentation no longer describes the current full `tests/smoke` package
      as fast repo-level smoke coverage unless the implementation makes that
      true again.
- [x] User-facing validation commands are named by purpose rather than runtime,
      with an ordinary development profile and a full release-ready profile.
- [x] Opt-in installer and release test tags use functional names instead of
      `slow_smoke`.
- [x] `VERSION` and release PR documentation requires the full release-ready
      validation profile before merge.
- [x] The release workflow is documented as post-merge publish validation, not
      as a substitute for release-ready PR validation.
- [x] CI, docs, and tests avoid relying on an agent or maintainer's subjective
      judgment that a release-adjacent change is "small".

## Deferred Items

- Broader test taxonomy changes beyond issue 257, such as adding a complete
  end-to-end or resilience suite, remain deferred unless they are the smallest
  clean way to express this validation split.
- Fine-grained path-specific validation profiles remain deferred. This plan
  should not introduce separate release-surface and installer-surface CI
  matrices unless they become the simplest way to express the two approved
  profiles.

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

- Done: [x]

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

Updated durable validation docs in `docs/development.md`, `docs/releasing.md`,
and `docs/specs/proposals/testing-structure.md` to describe the quick default
path and the explicit slow smoke command:
`go test -tags slow_smoke ./tests/installer ./tests/release -count=1`.

The first full documented slow-smoke attempt exposed that installer tests were
forcing a temporary empty `GOMODCACHE`, which made parallel cold installs depend
on fresh network downloads and hit Go proxy TLS timeouts. Repaired this by
leaving `GOMODCACHE` on the normal warmed default unless a test explicitly
overrides it.

Validation: doc/workflow smoke tests passed with
`go test ./tests/smoke ./tests/release -run 'TestReleaseDocsPresentStableOnboardingSurface|TestCIWorkflowBuildsEmbeddedUIBeforeGoTests|TestReleaseWorkflowWiresHomebrewTapPublishing' -count=1`.
The full documented slow path passed in about 3:01.16:
`go test -tags slow_smoke ./tests/installer ./tests/release -count=1`
reported `tests/installer` at 180.956s and `tests/release` at 92.849s.
Focused quick validation passed with `go test ./tests/smoke ./tests/release ./tests/support`.
After `review-004-delta` flagged that the exact documented quick path was not
recorded, ran `scripts/build-embedded-ui` and `go test ./...`; they passed in
about 3.520s and 1:15.78 respectively. The same review also flagged that the
testing-structure proposal layout omitted `tests/release/` and
`tests/installer/`; the layout now includes both.

#### Review Notes

`review-004-delta` found one blocking tests issue and one minor
docs-consistency issue: the Step 3 notes had not recorded the exact documented
quick path, and the testing-structure proposal layout omitted `tests/release/`
and `tests/installer/`. Repaired both by recording
`scripts/build-embedded-ui` plus `go test ./...` evidence and updating the
layout. Follow-up `review-005-delta` passed with no findings.

### Step 4: Replace Duration-Based Smoke Selection with Validation Profiles

- Done: [x]

#### Objective

Turn the reopened split into two purpose-named validation profiles so release
readiness is deterministic and not hidden behind subjective "small change" or
"slow test" judgment.

#### Details

The approved direction is:

- ordinary development validation: a default profile for normal feature,
  documentation, and bug-fix work;
- full release-ready validation: a profile for `VERSION` PRs, release PRs, and
  pre-release confidence checks.

Add simple script entrypoints so humans and agents do not need to remember
build tags directly. Preferred names are:

- `scripts/validate`
- `scripts/validate-release`

`scripts/validate` should run the ordinary development profile, currently
expected to include `scripts/build-embedded-ui` and `go test ./...`.

`scripts/validate-release` should include `scripts/validate` and then run the
installer and release archive/package smoke coverage that is intentionally
outside ordinary development validation.

Rename `slow_smoke` build tags to functional names. The exact names can be
chosen during implementation, but the intended meaning is release smoke and
installer smoke, not generic slowness. A likely clean shape is:

- `//go:build release_smoke` for release archive/package construction tests;
- `//go:build installer_smoke` for installer and wrapper PATH tests;
- `scripts/validate-release` invokes the needed tags/packages directly.

Update documentation and workflow checks so `VERSION` and release PRs are
instructed to run the release-ready validation profile before merge. The
post-merge Release workflow should remain described as publish validation: it
builds and publishes from the tag, verifies published release assets, and
checks Homebrew behavior after assets exist.

Do not add a path-filter matrix for this step unless it becomes simpler than
the two-profile rule. The accepted model is deliberately coarse: ordinary
development gets the default profile; release-ready work gets the full release
profile.

#### Expected Files

- `scripts/validate`
- `scripts/validate-release`
- `tests/installer/**`
- `tests/release/**`
- `docs/releasing.md`
- `docs/development.md`
- `docs/specs/proposals/testing-structure.md`
- `.github/workflows/ci.yml` if CI should call `scripts/validate` instead of
  spelling out its commands
- `docs/plans/active/2026-06-18-speed-up-smoke-validation.md`

#### Validation

- Run `scripts/validate`.
- Run `scripts/validate-release`.
- Run `go test -list` with the new functional tags to confirm installer and
  release archive tests are reachable through the expected profile.
- Run focused docs/workflow smoke tests that assert release checklist and CI
  guidance.
- Run `harness plan lint docs/plans/active/2026-06-18-speed-up-smoke-validation.md`.

#### Execution Notes

Added purpose-named validation entrypoints. `scripts/validate` now owns the
ordinary development profile by building embedded UI assets and running
`go test ./...`; CI invokes that script directly. `scripts/validate-release`
includes `scripts/validate`, then runs installer smoke with
`installer_smoke` and release archive smoke with `release_smoke`.

Renamed the opt-in build tags from duration language to functional meaning and
updated release/development docs plus workflow smoke assertions. `VERSION` and
release PR guidance now requires `scripts/validate-release` before merge, and
the post-merge Release workflow is described as publish validation from the
packaged source tree rather than a substitute for PR validation.

Validation passed:
`go test -list . ./tests/smoke ./tests/release`;
`go test -tags installer_smoke -list . ./tests/installer`;
`go test -tags release_smoke -list . ./tests/release`;
`go test ./tests/smoke ./tests/release -count=1`;
`scripts/validate`; and `scripts/validate-release`. The release-ready profile
ran installer smoke in 250.429s and release archive smoke in 66.450s after the
ordinary development validation portion passed.

#### Review Notes

`review-016-delta` found two blocking issues: the active plan closeout
summaries still described the old duration-oriented validation contract, and
the smoke tests did not prove `installer_smoke` and `release_smoke` selected
the intended tests. Repaired both by rewriting the current closeout summaries
and adding `TestValidationProfileTagsSelectReleaseReadySmokeTests`.

Follow-up `review-017-delta` passed with `docs-consistency` and `tests`
reviewer slots reporting no findings.

## Validation Strategy

- Use `go test -list` before and after the split to confirm tests moved into
  the intended command surfaces.
- Run ordinary development validation with elapsed-time output and confirm it
  avoids the old monolithic smoke timeout risk.
- Run the full release-ready validation profile to prove installer and release
  confidence checks still exist.
- Run workflow/documentation checks as plain file inspection plus relevant Go
  smoke tests that assert workflow wiring where such tests already exist.

## Risks

- Risk: The split could make important installer or release coverage less
  visible.
  - Mitigation: Keep explicit release-ready validation commands and update
    release docs so `VERSION` and release PRs use the full profile before
    merge.
- Risk: Reducing repeated installer work could accidentally stop testing the
  real cold install path.
  - Mitigation: Preserve at least one true cold installer smoke and convert only
    narrower wrapper behavior checks to lighter fixtures.
- Risk: New timeout values could be too aggressive on slower machines.
  - Mitigation: Choose conservative defaults, include useful diagnostics, and
    allow only intentional opt-in overrides if the repository already has a
    clear pattern for that.
- Risk: Functional profiles could grow into another ambiguous taxonomy.
  - Mitigation: Keep this step to two user-facing profiles and defer
    fine-grained path-specific matrices to follow-up work.

## Validation Summary

Ordinary development validation is now `scripts/validate`, which builds
embedded UI assets and runs `go test ./...`. It passed locally after Step 4,
and after the finalize repair it also passed when invoked from `docs/` as
`../scripts/validate`, proving the profile anchors Go tests to the repository
root instead of the caller directory.

Full release-ready validation is now `scripts/validate-release`, which runs
`scripts/validate` and then the purpose-tagged installer and release archive
smoke suites. It passed locally after Step 4, and again after the finalize
repair with warmed installer Corepack/pnpm caches; the latest installer
profile ran with `go test -tags installer_smoke ./tests/installer -count=1`
in 182.964s, and the latest release archive profile ran with
`go test -tags release_smoke ./tests/release -count=1` in 113.788s.

Focused validation also passed:
`go test -list . ./tests/smoke ./tests/release`,
`go test -tags installer_smoke -list . ./tests/installer`,
`go test -tags release_smoke -list . ./tests/release`,
`go test ./tests/smoke ./tests/release -count=1`, and
`harness plan lint docs/plans/active/2026-06-18-speed-up-smoke-validation.md`.
Finalize repair validation also covered
`go test ./tests/smoke -run 'TestSyncContractArtifactsCheckFailsOnStaleGeneratedFiles|TestSyncContractArtifactsCheckFailsOnDeprecatedGeneratedDocs|TestValidationScriptsDefineDevelopmentAndReleaseProfiles|TestValidationProfileTagsSelectReleaseReadySmokeTests|TestCIWorkflowUsesDevelopmentValidationProfile' -count=1`,
`go test ./tests/support -count=1`, and
`go test -tags installer_smoke ./tests/installer -run 'TestInstallDevHarnessWrapperSkipsOtherManagedWrappersOnPathOutsideWorktree' -count=1`.
Timeout regression tests still cover the command helper repairs from revision
1, and the current profile tests prove the user-facing scripts plus functional
build tags select the intended release-ready smoke tests.

## Review Summary

Step reviews passed after targeted repairs:
`review-001-delta` passed the validation split, `review-003-delta` passed the
timeout regression repair for `RunWithOptions`, and `review-005-delta` passed
the documentation/evidence repair for the exact quick path and test layout.

Finalize review intentionally found additional timeout gaps. Repairs were
made and re-reviewed for release archive subprocesses (`review-008-delta`
passed after the live release follow-up), default smoke subprocesses
(`review-011-delta` passed after the tar stderr fix), and support setup
subprocesses plus installer slow-smoke repeatability. The final full review
round, `review-015-full`, passed with correctness, tests, docs-consistency,
and risk-scan slots reporting no findings.

After revision 2 reopened the plan for the purpose-named validation profile
repair, Step 4 review `review-016-delta` found two blocking issues: closeout
summaries still described the old duration-oriented contract, and automated
tests did not prove the functional build tags selected the intended tests.
Both were repaired in the active plan and smoke test coverage before follow-up
review.

Finalize review `review-018-full` found three archive-blocking issues and one
duplicate non-blocking issue: the validation scripts ran Go tests from the
caller directory instead of the repository root, installer smoke could still
fall through to a live Corepack/pnpm download from temporary homes, and the
Archive Summary still said the Step 4 repair review was pending. The repairs
anchor validation scripts to `repo_root`, share warmed installer Corepack and
pnpm caches, and update the archive handoff text before follow-up review.

## Archive Summary

- Archived At: PENDING_ARCHIVE
- Revision: 2
Ready for archive once the finalize repair review passes.
The active plan contains checked acceptance criteria, completed tracked steps
through Step 4, validation evidence, review history through
`review-018-full`, and no deferred follow-up issue requirement beyond the
existing broader test-taxonomy deferral. No supplements were used.

- PR: NONE
- Ready: Pending finalize repair review, archive, publish/CI, and sync
  evidence.
- Merge Handoff: After archive, update PR #259 for issue 257, record
  publish/CI/sync evidence, and wait for explicit human merge approval.

## Outcome Summary

### Delivered

- Ordinary development validation no longer runs installer and release archive
  smoke coverage by accident.
- User-facing validation commands are now named by purpose:
  `scripts/validate` for daily development and `scripts/validate-release` for
  full release-ready validation.
- Installer and release archive smoke coverage moved behind functional build
  tags, `installer_smoke` and `release_smoke`, invoked by
  `scripts/validate-release`.
- Smoke command helpers gained conservative command-level timeouts with
  captured stdout/stderr diagnostics.
- Installer smoke tests now run in parallel with shared setup and warmed
  dependencies, while preserving real `scripts/install-dev-harness` coverage.
- Documentation now requires `scripts/validate-release` for `VERSION` and
  release PRs before merge, and describes the post-merge Release workflow as
  publish validation rather than a substitute for release-ready PR validation.

### Not Delivered

No product behavior changes were delivered outside the validation split. A
broader end-to-end test taxonomy remains outside this issue slice.

### Follow-Up Issues

- https://github.com/catu-ai/easyharness/issues/258 tracks the broader
  validation taxonomy work that remains outside issue 257.
