---
template_version: 0.2.0
created_at: "2026-04-24T23:49:44+08:00"
approved_at: "2026-04-24T23:51:38+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/193
    - https://github.com/catu-ai/easyharness/issues/194
size: XS
---

# Fix Actions Node 24 readiness and Homebrew verification

## Goal

Resolve the CI and release workflow maintenance issues tracked by #193 and
#194 in one narrow release-automation slice.

The intended end state is that GitHub Actions jobs no longer use JavaScript
actions that emit the Node.js 20 deprecation annotation, and the Release
workflow's `Verify Homebrew Install` job provisions the same `pnpm` dependency
that the live Homebrew smoke test expects on `PATH`.

## Scope

### In Scope

- Upgrade the GitHub Actions versions used by CI, Release, and VERSION-driven
  release-tag automation to Node.js 24-ready action major versions.
- Add `pnpm` setup to the Release workflow's `verify-homebrew-install` job
  before running `TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled`.
- Update workflow smoke tests so they assert the new action versions and the
  Homebrew verification job setup.
- Validate the updated workflows and focused smoke tests locally.
- After publish, confirm the relevant PR CI and release verification workflows
  no longer emit the Node.js 20 deprecation annotation and that Homebrew
  verification can pass against the published `v0.2.5` release and tap formula.

### Out of Scope

- Changing release asset build logic, archive contents, checksum generation, or
  Homebrew formula rendering semantics.
- Changing the published `v0.2.5` assets or rewriting release history.
- Adding new CI jobs, matrices, operating systems, or broader workflow
  restructuring.
- Changing the pinned tool versions for Go, Node.js, or `pnpm` unless required
  by the action-version upgrade.
- Altering GitHub repository branch protection, secrets, or Homebrew tap
  repository policy.

## Acceptance Criteria

- [x] CI and release workflows use Node.js 24-ready versions of
      `actions/checkout`, `actions/setup-go`, `actions/setup-node`, and
      `pnpm/action-setup`.
- [x] `.github/workflows/release.yml` sets up `pnpm` in
      `verify-homebrew-install` before running the Homebrew tap smoke test.
- [x] Smoke tests assert the upgraded workflow action versions and the added
      Homebrew verification `pnpm` setup.
- [x] Local focused validation passes for the workflow smoke tests touched by
      this plan.
- [ ] Post-publish evidence shows PR CI and release-relevant workflows no
      longer emit the Node.js 20 deprecation annotation.
- [ ] Post-publish evidence shows the Release workflow's `Verify Homebrew
      Install` job passes, either by rerunning the `v0.2.5` Release workflow or
      by an equivalent live verification against the published release and tap
      formula.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Upgrade workflow action versions

- Done: [x]

#### Objective

Move the repository's GitHub Actions workflow references from Node.js 20 action
majors to Node.js 24-ready action majors.

#### Details

Discovery found Node.js 20 deprecation annotations in PR CI run
`24897276878` and Release run `24897631690`. At discovery time, the relevant
latest releases were:

- `actions/checkout@v6.0.2`
- `actions/setup-go@v6.4.0`
- `actions/setup-node@v6.4.0`
- `pnpm/action-setup@v5.0.0`

Each referenced `action.yml` declared `node24`. During execution, re-check the
current latest compatible versions before editing if enough time has passed for
the answer to plausibly change. Prefer pinned major or full tag style
consistently with the existing workflows after deciding the local convention.

#### Expected Files

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.github/workflows/tag-release-from-version.yml`
- `tests/smoke/ci_workflow_test.go`
- `tests/smoke/homebrew_formula_test.go`
- `tests/smoke/release_version_file_test.go`

#### Validation

- Workflow files no longer reference the old action majors that produced the
  Node.js 20 annotation.
- Smoke tests assert the new action references.
- Workflow syntax remains valid YAML.

#### Execution Notes

Updated `.github/workflows/ci.yml`, `.github/workflows/release.yml`, and
`.github/workflows/tag-release-from-version.yml` to use the repository's
existing major-tag style with Node.js 24-ready actions:
`actions/checkout@v6`, `actions/setup-go@v6`, `actions/setup-node@v6`, and
`pnpm/action-setup@v5`.

Updated the workflow smoke tests to assert the upgraded action references.
TDD was compressed because the behavior is workflow shape rather than runtime
code, but focused tests caught an overly broad Homebrew verification ordering
assertion during validation and the assertion was tightened before closeout.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 were implemented as one tightly
coupled workflow-maintenance slice and should be reviewed together in finalize
review after full local validation.

### Step 2: Provision pnpm for Homebrew verification

- Done: [x]

#### Objective

Fix the Release workflow's `Verify Homebrew Install` job so the live Homebrew
smoke test can find `pnpm` on `PATH`.

#### Details

Issue #194 is an actual release workflow failure: the `Build and Publish
Release Assets` job succeeded for `v0.2.5`, but `Verify Homebrew Install`
failed because `TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled` calls
the shared `installerPath(t)` helper, and that helper requires both `go` and
`pnpm` on `PATH`.

Keep this fix scoped to verification setup. Add the same minimal `pnpm` setup
pattern already used by the build job, adjusted only as needed for the macOS
verification job. Do not change the smoke test's dependency expectation unless
investigation proves the test is the wrong boundary.

#### Expected Files

- `.github/workflows/release.yml`
- `tests/smoke/homebrew_formula_test.go`

#### Validation

- The `verify-homebrew-install` job provisions `pnpm` before the `Run Homebrew
  tap smoke` step.
- The workflow smoke test asserts this setup and the relevant step ordering.

#### Execution Notes

Added a `Set up pnpm` step to the Release workflow's
`verify-homebrew-install` job before `Run Homebrew tap smoke`, preserving the
existing pinned `pnpm` version and `run_install: false` configuration already
used by the release build job.

Updated `TestReleaseWorkflowWiresHomebrewTapPublishing` to assert the
verification job provisions `pnpm` after Go setup and before the live Homebrew
smoke test.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This setup fix is inseparable from the action-version
workflow update in Step 1; finalize review will cover the full candidate.

### Step 3: Validate and prepare publish evidence

- Done: [x]

#### Objective

Run local validation for the workflow changes and record the remote evidence
needed to close #193 and #194 after publish.

#### Details

Local validation should focus on the smoke tests that assert workflow shape,
plus YAML parsing or an equivalent workflow syntax check. Full `go test ./...`
is acceptable if practical and gives better confidence, but the plan does not
require broad code changes outside workflow assertions.

Remote closeout matters here because both issues were observed in GitHub
Actions. After the branch is published, gather evidence from PR CI and from
the release-relevant path. For #194, rerun the Release workflow for `v0.2.5`
when that is the cleanest confirmation, or record an equivalent live Homebrew
verification against the already-published `v0.2.5` release and updated tap
formula if rerunning the release workflow is not appropriate.

#### Expected Files

- `docs/plans/active/2026-04-24-fix-actions-node24-and-homebrew-verification.md`

#### Validation

- Focused smoke tests pass locally.
- Workflow YAML validates locally.
- Plan execution notes record the local validation commands and the limits of
  local validation.
- Archive-time evidence records the relevant remote run URLs or the reason an
  equivalent confirmation was used instead.

#### Execution Notes

Validated the workflow updates locally with:

- `ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path); puts "yaml-ok #{path}" }' .github/workflows/ci.yml .github/workflows/release.yml .github/workflows/tag-release-from-version.yml`
- `go test ./tests/smoke -run 'TestCIWorkflowBuildsEmbeddedUIBeforeGoTests|TestReleaseWorkflowWiresHomebrewTapPublishing|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
- `go test ./...`

The old Node.js 20 action major scan returned no matches for
`actions/checkout@v4`, `actions/setup-go@v5`, `actions/setup-node@v4`, or
`pnpm/action-setup@v4` under `.github` and `tests/smoke`.

Remote evidence remains intentionally deferred to the publish phase because
the Node.js deprecation annotation and Homebrew verification failure are hosted
GitHub Actions behaviors.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 is validation and evidence preparation for the
same workflow slice; finalize review will cover the complete candidate.

## Validation Strategy

- Parse the touched workflow YAML files locally after editing.
- Run focused smoke tests for workflow assertions, expected to include:
  `go test ./tests/smoke -run 'TestCIWorkflowBuildsEmbeddedUIBeforeGoTests|TestReleaseWorkflowWiresHomebrewTapPublishing|TestVersionTagWorkflowUsesRepositoryVersionFile' -count=1`
- Run broader `go test ./...` if practical after the focused workflow tests
  pass.
- After publish, inspect PR CI logs for the absence of the Node.js 20
  deprecation annotation.
- After publish, confirm the release-relevant Homebrew verification path
  passes against the published `v0.2.5` release and Homebrew tap formula.

## Risks

- Risk: Upgrading action majors may introduce input or behavior changes in
  checkout, setup-go, setup-node, or pnpm setup.
  - Mitigation: keep workflow edits narrow, preserve existing inputs, and
    validate the affected workflow assertions before publish.
- Risk: The Homebrew verification job can pass setup but still fail later due
  to live Homebrew, GitHub release, or tap-state conditions.
  - Mitigation: distinguish the `pnpm` setup fix from unrelated live service
    failures and record any new failure as follow-up scope only if it is
    outside this plan's acceptance target.
- Risk: Local validation cannot prove GitHub's hosted runner no longer emits
  the deprecation warning.
  - Mitigation: require post-publish Actions log evidence before treating the
    candidate as ready for merge approval.

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
