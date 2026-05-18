---
template_version: 0.2.0
created_at: "2026-05-18T23:44:43+08:00"
approved_at: "2026-05-18T23:45:57+08:00"
source_type: github_issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/203
size: M
---

# Define Read-Only PR Handoff Identity

<!-- If this plan uses supplements/<plan-stem>/, keep the markdown concise,
absorb any repository-facing normative content into formal tracked locations
before archive, and record archive-time supplement absorption in Archive
Summary or Outcome Summary. Lightweight plans should normally avoid
supplements. -->

## Goal

Resolve issue #203 by defining the read-only identity model that later
remote-handoff work can rely on without guessing. The core contract is that a
recorded publish evidence PR URL is the authoritative remote handoff anchor
once a candidate has reached the publish phase.

The end state should let future PR and CI status work observe a known PR with
`gh` when available, degrade clearly when `gh` or auth is unavailable, and keep
the existing manual evidence-submit workflow as the fallback. Local git and
remote facts remain useful context, but they must not become a branch-based PR
guessing system.

## Scope

### In Scope

- Document that `publish` evidence `pr_url` is the authoritative remote
  handoff identity for an archived candidate.
- Document that local git branch, commit, remote, and GitHub owner/repo facts
  are contextual observations rather than PR association authority.
- Define degraded read states for missing recorded PR URL, invalid PR URL,
  missing `gh`, missing `gh` auth, unreadable PR, unsupported or unavailable
  remote context, detached HEAD, and mismatched contextual facts.
- Add a small internal read-model skeleton that can load the current local git
  context, parse a recorded GitHub PR URL, and call `gh` only to observe that
  recorded PR.
- Add tests for the read-only model, including success and degraded paths,
  without requiring real GitHub network access or a real authenticated `gh`
  session.

### Out of Scope

- Introducing `.harness` customization or any tracked configuration schema.
- Discovering or guessing a PR from the current branch when publish evidence
  does not already record a PR URL.
- Reading CI/check status, merge policy, review state, or full PR readiness.
- Adding `harness publish`, opening PRs, updating PRs, commenting, rerunning
  checks, merging, or performing any git/GitHub write operation.
- Changing `harness status` workflow-node progression to depend on live
  GitHub reads.
- Supporting non-GitHub hosts beyond explicit degraded states.

## Acceptance Criteria

- [ ] Specs identify recorded publish `pr_url` as the remote handoff anchor for
      later PR and CI reads.
- [ ] Specs clearly distinguish authoritative harness evidence from contextual
      local git/remote observations.
- [ ] The implementation does not perform branch-based PR discovery when no
      recorded PR URL exists; it degrades to manual publish evidence fallback.
- [ ] The read model uses `gh` only for read-only observation of a recorded PR
      URL and treats missing `gh`, missing auth, network/API failure, and
      unreadable PRs as degraded observations rather than workflow failures.
- [ ] Tests cover recorded PR URL parsing, local git context success and
      detached/missing-remote degradation, `gh` success, and `gh` degraded
      paths using fake command execution.
- [ ] Existing manual `harness evidence submit --kind publish|ci|sync` flows
      continue to work unchanged.

## Deferred Items

- Add PR and CI fact reading for a recorded PR URL in issue #199.
- Surface remote handoff facts and next actions through `harness status` in
  issue #200.
- Add repo-level `.harness` customization for complex remote mappings in
  issue #71 or a follow-up customization slice.

## Work Breakdown

### Step 1: Specify the no-guess remote handoff contract

- Done: [x]

#### Objective

Update the durable specs so future agents can tell which remote handoff facts
are authoritative, which are contextual, and when the workflow should fall
back to manual evidence rather than guessing.

#### Details

`docs/specs/state-model.md` should define the publish-phase identity rule:
once publish evidence records a PR URL, that URL is the candidate's
authoritative remote handoff anchor. Local git branch, commit, upstream,
origin, and remote repository facts are context that may explain warnings but
must not replace recorded evidence.

`docs/specs/cli-contract.md` should describe the intended read-only boundary
for later status integration: live `gh` reads may observe a recorded PR, but
status must continue to degrade without failing local workflow state when
`gh`, auth, or the network is unavailable. When no recorded PR URL exists, the
next action should remain to open/update the PR and record publish evidence,
not to discover or guess a PR from the current branch.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`

#### Validation

- The specs are understandable without this plan or chat history.
- `git diff --check` passes for the edited specs and plan.
- `harness plan lint docs/plans/active/2026-05-18-define-read-only-pr-handoff-identity.md` still passes after the edits.

#### Execution Notes

Updated `docs/specs/state-model.md` and `docs/specs/cli-contract.md` to define
recorded publish `pr_url` as the remote handoff anchor, local git/remote facts
as contextual observations, and `gh` as an optional read-through provider that
must degrade without blocking local workflow state. Validation:
`git diff --check -- docs/specs/state-model.md docs/specs/cli-contract.md
docs/plans/active/2026-05-18-define-read-only-pr-handoff-identity.md`;
`harness plan lint
docs/plans/active/2026-05-18-define-read-only-pr-handoff-identity.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 only records the approved no-guess handoff
contract in specs; implementation and behavior tests begin in Step 2.

### Step 2: Add the local and recorded-PR identity skeleton

- Done: [x]

#### Objective

Create the minimal internal read model for local git context and recorded PR
URL identity without adding status output behavior or GitHub writes.

#### Details

Add a small internal package or equivalent local module that reports:

- local git context for the current worktree, including branch when available,
  HEAD commit when available, selected remote context when unambiguous, and
  degraded states such as not-a-git-repository, detached HEAD, missing remote,
  unsupported host, and ambiguous remote context
- recorded PR identity parsed from a publish evidence PR URL, limited to
  supported GitHub PR URLs
- explicit degraded identity when no publish evidence PR URL exists

The model should not infer PR identity from branch names. Any current-branch
or remote information should be retained as context only, so later status work
can produce warnings without turning context into authority.

#### Expected Files

- `internal/remote/...` or another clearly named internal read-model package
- `internal/remote/..._test.go`
- Supporting contract files only if needed for generated schemas or shared
  result structs

#### Validation

- Unit tests cover URL parsing, unsupported URLs, missing recorded PR URL,
  normal branch context, detached HEAD context, missing remote context, and
  unsupported remote context.
- Tests use temporary git repositories where local git behavior matters.
- The implementation has no GitHub network dependency.

#### Execution Notes

Added `internal/remote` local identity skeleton and tests. The package now
parses recorded GitHub PR URLs, reports missing or unsupported recorded PR
identity, inspects current worktree git context, records detached HEAD and
remote degradation, and treats branch/remote facts as context only. Validation:
`go test ./internal/remote`. After `review-001-delta`, added negative coverage
and parsing guards so empty GitHub owner/repo path segments degrade instead of
being accepted as recorded or supported identity. Validation:
`go test ./internal/remote`.

#### Review Notes

`review-001-delta` found one important correctness issue: malformed GitHub PR
or remote URLs with empty repo segments could be accepted as valid identity.
The repair adds negative tests for empty repo PR and remote URLs and rejects
empty owner/repo segments before recording GitHub identity. `review-002-delta`
passed with no findings after the repair.

### Step 3: Add read-only `gh` observation with degraded paths

- Done: [ ]

#### Objective

Add a mockable `gh` observation path for a recorded PR URL so later PR/CI work
can build on a tested provider boundary.

#### Details

Use `gh` as the intended read-through provider for GitHub PR observation, but
do not make it a hard workflow dependency. The provider should call `gh` only
for read-only operations against a recorded PR URL, such as `gh pr view
<url> --json ...`, and it should classify failures into clear degraded states.

The execution boundary should allow tests to fake command execution so the
suite does not require a real `gh` binary, real auth, or network access.
Missing `gh`, auth errors, non-zero command exits, invalid JSON, and unreadable
PRs should produce degraded observations and preserve manual evidence submit
as the fallback path.

#### Expected Files

- `internal/remote/...`
- `internal/remote/..._test.go`

#### Validation

- Unit tests cover `gh` success, missing binary, auth unavailable, unreadable
  PR, invalid JSON, and generic command failure.
- The provider does not call `gh pr list` or otherwise discover PRs from the
  current branch.
- The provider performs no git or GitHub write operation.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run focused Go tests for the new read-model package.
- Run relevant status/evidence tests if shared contracts or evidence loading
  behavior changes.
- Run `go test ./...` unless the implementation remains narrowly isolated and
  a faster package set is clearly sufficient.
- Run `harness plan lint` for this plan after step closeout updates.
- Run `git diff --check` before archive.

## Risks

- Risk: The implementation could accidentally reintroduce branch-based PR
  guessing under the name of convenience.
  - Mitigation: Keep the contract explicit, add tests proving no `gh pr list`
    discovery is used, and require missing PR evidence to degrade to manual
    fallback.
- Risk: `gh` failure modes could become workflow failures and make local
  harness status brittle on unauthenticated machines.
  - Mitigation: Treat `gh` as an optional read-through provider and classify
    missing auth, missing binary, and network/API failures as degraded
    observations.
- Risk: The skeleton could overfit the current issue and make later #199/#200
  integration awkward.
  - Mitigation: Keep the package focused on identity and provider boundaries,
    not CI/readiness policy or status-node progression.

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
