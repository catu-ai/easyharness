---
template_version: 0.2.0
created_at: "2026-06-07T00:00:00Z"
approved_at: "2026-06-07T23:53:24+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/229
size: XL
---

# Support Custom Harness Path Roots

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Implement the first real `.harness/config.yaml` customization field by letting
repositories configure harness path roots for tracked plan storage and
command-owned local runtime storage.

After this slice, core harness workflows should behave the same under the
built-in defaults and under configured path roots. The feature should close
issue #229 without adding compatibility shims for obsolete layouts.

## Scope

### In Scope

- Define the v1 repo config shape for custom harness path roots.
- Add a central path resolver with built-in defaults for:
  - active tracked plan root, defaulting to `docs/plans/active`
  - standard archived tracked plan root, defaulting to `docs/plans/archived`
  - command-owned local runtime root, defaulting to `.local/harness`
- Derive supplements paths from the configured active and archived plan roots.
- Derive local runtime contents from the configured runtime root, including
  current-plan pointer, plan state, review artifacts, evidence artifacts,
  timeline events, locks, and lightweight archived plan snapshots.
- Update status, archive, reopen, lint, plan UI, review UI, review, evidence,
  timeline, lifecycle, and related command read models to respect configured
  roots.
- Keep path validation strict enough to reject escaping or ambiguous roots with
  clear user-facing errors.
- Preserve the clean target behavior only; no migration bridge or fallback reads
  for older custom layouts are needed.

### Out of Scope

- Dashboard/watchlist workspace grouping and dashboard-specific reads; issue
  #232 tracks that follow-up surface.
- Repo instruction references, review defaults, remote mappings, hooks,
  provider mappings, or other `.harness` customization fields.
- A standalone `harness repo config lint` command; issue #235 tracks that
  follow-up command.
- Migrating existing repositories or dual-reading old and new runtime roots.

## Acceptance Criteria

- [ ] A valid `.harness/config.yaml` can configure active tracked plans,
      standard archived tracked plans, and command-owned local runtime roots.
- [ ] Missing config and default config preserve the current default paths and
      existing tests continue to pass.
- [ ] Invalid path config rejects absolute paths, `..` escape, ambiguous or
      overlapping roots, and unsupported path shapes with clear errors.
- [ ] `harness status`, `archive`, `reopen`, `plan lint`, plan UI, review UI,
      review commands, evidence commands, timeline reads, and lifecycle outputs
      all use the configured roots for their own plan/runtime artifacts.
- [ ] Supplements are derived from the configured plan roots rather than
      configured independently.
- [ ] Lightweight archived snapshots and all local runtime artifacts live under
      the configured local runtime root.
- [ ] Focused unit and end-to-end coverage proves default-root behavior and a
      custom-root workflow through active plan detection, runtime writes,
      archive/reopen, review/evidence reads, and UI read models.

## Deferred Items

- Dashboard reads respecting repo-level path customization remain deferred to
  #232.
- Repo config lint remains deferred to #235.
- Additional `.harness` customization fields remain deferred to their own
  issues, including review dimensions and repo instruction references.

## Work Breakdown

### Step 1: Specify and parse configurable path roots

- Done: [x]

#### Objective

Define the repo config contract for path roots and teach `internal/repoconfig`
to parse validated path settings with default values.

#### Details

Use a compact config shape that keeps `version: 1` as the manifest entrypoint
and introduces a `paths` object for the first real customization field. The
accepted shape should be documented in `docs/specs/repo-config.md` before
behavioral code depends on it.

The expected configuration shape is:

```yaml
version: 1
paths:
  plans:
    active: docs/plans/active
    archived: docs/plans/archived
  local_runtime: .local/harness
```

All path values should be repo-relative slash paths. Reject absolute paths,
empty paths, `..` escape, paths that resolve outside the worktree, ambiguous
root overlap, plan roots inside the local runtime root, and local runtime roots
inside tracked plan roots. Keep supplements derived from plan roots rather than
making them independently configurable.

#### Expected Files

- `docs/specs/repo-config.md`
- `internal/repoconfig/config.go`
- `internal/repoconfig/config_test.go`

#### Validation

- `go test ./internal/repoconfig`
- Focused tests cover defaults, valid custom roots, unsupported fields, invalid
  path strings, escaping paths, and ambiguous root combinations.

#### Execution Notes

Added the v1 `paths` contract to `docs/specs/repo-config.md` and extended
`internal/repoconfig` to parse default and custom active, archived, and local
runtime roots. Validation now rejects unsafe, escaping, empty, unknown, and
overlapping path roots while preserving defaults for missing fields.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is covered by focused config parser tests and
the final integrated review for the path-roots slice.

### Step 2: Centralize harness path resolution

- Done: [x]

#### Objective

Introduce a single path resolver used by plan and runtime packages instead of
hard-coded `docs/plans/...` and `.local/harness/...` strings.

#### Details

The resolver should expose repo-facing and absolute paths for the configured
roots and all command-owned derived artifacts. It should keep existing default
behavior when config is missing, and it should be small enough for callers to
depend on without each package reimplementing config loading.

At minimum, centralize active-plan discovery, active/archived path kind
classification, standard archive targets, lightweight archive targets,
supplements path derivation, current-plan pointer paths, per-plan local state,
review artifact paths, evidence artifact paths, timeline event paths, and lock
paths.

#### Expected Files

- `internal/plan/profile.go`
- `internal/plan/current.go`
- `internal/runstate/state.go`
- `internal/repoconfig/config.go`
- New or existing focused path-resolution helper files under `internal/`

#### Validation

- `go test ./internal/plan ./internal/runstate ./internal/repoconfig`
- Focused tests prove default and custom roots classify active, archived, and
  lightweight archived paths correctly and derive matching supplements/runtime
  paths.

#### Execution Notes

Centralized default/custom path derivation in plan and runstate helpers. Active
plan discovery, path-kind classification, archive targets, lightweight archive
targets, current-plan pointers, state files, locks, review dirs, evidence dirs,
and timeline event paths now derive from repo config defaults or custom roots.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Focused `plan`, `runstate`, and `repoconfig` tests cover
this step, and the integrated finalize review will check the full wiring.

### Step 3: Wire lifecycle, status, review, evidence, timeline, and UI reads

- Done: [x]

#### Objective

Update the command services and read models that issue #229 names so they
consume the resolver instead of hard-coded defaults.

#### Details

This step should cover status, archive, reopen, lint, plan UI, review UI,
review start/submit/aggregate, evidence submit/refresh, timeline reads and
writes, lifecycle outputs, and any adjacent helpers that surface artifact
paths. Command outputs should continue to show repo-facing slash paths.

Do not widen into dashboard/watchlist behavior here; keep that separated for
#232.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/status/service.go`
- `internal/review/service.go`
- `internal/reviewui/service.go`
- `internal/evidence/service.go`
- `internal/timeline/service.go`
- `internal/planui/service.go`
- `internal/cli/app.go`
- Related service tests

#### Validation

- `go test ./internal/lifecycle ./internal/status ./internal/review ./internal/reviewui ./internal/evidence ./internal/timeline ./internal/planui ./internal/cli`
- Existing default-root tests remain green.
- New focused tests prove each major service reads and writes under custom
  roots.

#### Execution Notes

Updated lifecycle-facing services and read models to consume the configured
path helpers. Review, review UI, status, evidence, step-closeout scanning,
timeline, plan UI, archive, reopen, and lint behavior now work under custom
plan and runtime roots while default-root behavior remains green.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Package tests and custom-root e2e coverage exercise the
wired services; final review will inspect the cross-package behavior together.

### Step 4: Prove the end-to-end custom-root workflow

- Done: [x]

#### Objective

Add end-to-end or smoke coverage that exercises a configured custom-root
workflow across the main command surfaces and closes issue #229 with durable
evidence.

#### Details

Build a custom-root fixture that creates `.harness/config.yaml`, stores the
active plan outside `docs/plans/active`, stores runtime outside
`.local/harness`, and drives the workflow far enough to prove active detection,
runtime state, archive/reopen, review/evidence, and read-model behavior.

The validation should also prove default-root behavior still works, so the
feature is a clean extension of the current defaults rather than a rewrite that
only works in the new layout.

#### Expected Files

- `tests/e2e/`
- `tests/smoke/`
- Any focused support helpers needed for custom-root fixtures

#### Validation

- `go test ./tests/e2e ./tests/smoke`
- Full relevant package test set after implementation.
- Manual probe equivalent to: custom config plus custom active plan makes
  `harness status` report `plan` instead of `idle`.

#### Execution Notes

Added built-binary e2e coverage for custom-root workflow and custom-root
reopen behavior, plus focused plan UI and review UI tests for configured roots.
Validation passed for focused internal packages, all e2e tests, and smoke tests.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The e2e and smoke validation is complete; use the final
full review as the review gate for this integrated behavior slice.

## Validation Strategy

- Start with unit coverage for config parsing and path resolution.
- Keep default-root regression tests green throughout, since default behavior
  remains the primary no-config path.
- Add custom-root service tests before or alongside wiring changes so hard-coded
  path regressions are caught close to their package.
- Finish with e2e/smoke coverage for a full configured-root workflow and run the
  focused internal package set plus relevant e2e/smoke tests before archive.

## Risks

- Risk: Hard-coded path assumptions are spread across lifecycle, read models,
  review, evidence, timeline, CLI tests, and UI resource services.
  - Mitigation: Introduce a central resolver early and move packages through it
    in focused steps with default-root regression coverage.
- Risk: Custom roots can become ambiguous or unsafe if validation accepts
  escaping or overlapping paths.
  - Mitigation: Reject unsafe path shapes in `internal/repoconfig` before any
    command consumes them, and cover the rejections with tests.
- Risk: Supplements or lightweight archived snapshots could drift from the
  configured roots and make plan packages hard to reason about.
  - Mitigation: Derive those paths from the configured plan/runtime roots and
    test active, standard archived, and lightweight archived cases explicitly.

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
