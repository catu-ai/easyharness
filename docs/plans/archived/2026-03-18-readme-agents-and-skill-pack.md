---
status: archived
lifecycle: awaiting_merge_approval
revision: 1
template_version: 0.1.0
created_at: "2026-03-18T22:25:00+08:00"
updated_at: "2026-03-18T23:49:13+08:00"
source_type: direct_request
source_refs: []
---

# Bootstrap README, AGENTS, and skill pack

## Goal

Turn `superharness` from a repository that merely contains a CLI into a
repository that can explain and drive its own workflow. This slice should make
the repo legible to both humans and agents by adding a human-facing README, an
agent-facing `AGENTS.md`, and the first repo-local skill pack built around the
existing harness lifecycle.

The outcome should be strong enough that a fresh Codex session can enter the
repo, learn how work is supposed to flow, invoke `harness` directly in the
current development environment, and dogfood the repository using the same
contracts it is trying to establish.

## Scope

### In Scope

- Add a `README.md` that explains what `superharness` is, which commands exist
  today, how the workflow is intended to operate, and how to use the repo in
  development.
- Add an `AGENTS.md` that records the repository's human/agent working
  agreement and points execution detail to the new skill pack.
- Add the first repo-local skill pack with exactly five top-level skills:
  `discovery`, `plan`, `execute`, `land`, and `reviewer`.
- Organize `execute` with references instead of proliferating more top-level
  loop skills.
- Make `harness` invocable as a direct command in this development environment
  without requiring users or skills to spell `go run ./cmd/harness ...`.
- Encode the current review orchestration rule in the skills:
  one active review round at a time, wait for every spawned reviewer subagent
  to finish before aggregating, and explicitly close reviewer subagents after
  their results are consumed.
- Dogfood the new docs and skills against this repo.

### Out of Scope

- `harness ui` and any web UI implementation.
- Formal release packaging, Homebrew publishing, or installer support beyond
  development-time setup for this repository.
- Supporting overlapping active review rounds in v0.1/v0.2 skills.
- Broad test-fixture infrastructure beyond the tests or validation directly
  needed for this slice.

## Acceptance Criteria

- [x] The repository contains a `README.md` that explains the project,
      development-time setup, current command surface, and current workflow in
      a way that a new human collaborator can follow.
- [x] The repository contains an `AGENTS.md` that defines the repo-level
      working agreement, source-of-truth split, and lifecycle expectations for
      Codex agents in this repo.
- [x] The repository contains repo-local skills for `discovery`, `plan`,
      `execute`, `land`, and `reviewer`, with `execute` decomposed through
      references rather than more top-level loop skills.
- [x] The `execute` and `reviewer` skill contracts explicitly document the
      current Codex-specific reviewer orchestration rules:
      wait until all reviewer subagents finish before aggregation, and close
      reviewer subagents after their results are recorded.
- [x] There is a documented and working development-time path to run `harness`
      directly as a command inside this repo without requiring an alias.
- [x] Any behavior-changing implementation in this slice has automated test
      coverage or a clearly justified deterministic validation path.

## Deferred Items

- `harness ui` remains deferred to #2.
- `harness plan list` and docs-navigation follow-ups remain deferred to #4.
- Later skill-system expansion beyond the first dogfoodable pack remains
  deferred to #5.
- Shared test infrastructure remains deferred to #6.

## Work Breakdown

### Step 1: Define the dogfoodable repository entrypoints

- Status: completed

#### Objective

Decide how `harness` should be invoked in development and document that choice
clearly enough that both humans and skills can rely on a stable command name.

#### Details

Do not rely on shell aliases. Prefer a development-time setup path that makes
`harness` directly executable in this repo while staying close to how a future
released binary would be used.

#### Expected Files

- `README.md`
- `AGENTS.md`
- `scripts/install-dev-harness`
- additional repo-local wrapper or setup files only if they are strictly
  needed

#### Validation

- A documented development flow can install or expose a working `harness`
  command in the current environment.
- If setup logic has non-trivial behavior, add or update automated tests where
  practical; otherwise provide deterministic smoke validation steps.

#### Execution Notes

Defined the dev-time entrypoint as `scripts/install-dev-harness` rather than a
shell alias. The installer builds `.local/bin/harness`, links `harness` into a
writable directory on `PATH` when possible, and falls back to `~/.local/bin`
with explicit PATH guidance when no writable PATH entry exists.

#### Review Notes

The first delta review found that installer success could be shadowed by a
different `harness` earlier on `PATH`. The installer now verifies the direct
binary path first, fails if `command -v harness` resolves to a different
binary, and then confirms the repo build is the one the shell will run.

### Step 2: Add human-facing repository docs

- Status: completed

#### Objective

Create the first `README.md` and `AGENTS.md` for `superharness`.

#### Details

`README.md` and `AGENTS.md` should not duplicate each other. README teaches
humans what the project is and how to run it; `AGENTS.md` teaches agents how
to work in the repo and where to find the operational detail.

#### Expected Files

- `README.md`
- `AGENTS.md`

#### Validation

- The README clearly explains the repo purpose, current command surface, and
  development-time setup.
- `AGENTS.md` clearly explains the working agreement, source-of-truth split,
  and lifecycle expectations without embedding every execution detail inline.

#### Execution Notes

Drafted `README.md` and `AGENTS.md` with separate roles: README is for human
onboarding and development setup; `AGENTS.md` defines the repo-level working
agreement, source-of-truth split, lifecycle, and local-skill entrypoints.

#### Review Notes

Fresh-agent dogfood confirmed the README and `AGENTS.md` split is legible.
After the first onboarding pass, the docs were tightened so the controller /
reviewer skill boundary is explicit instead of only inferable from
`harness status`.

### Step 3: Add the first repo-local skill pack

- Status: completed

#### Objective

Add `discovery`, `plan`, `execute`, `land`, and `reviewer` as the first
repo-local skills for `superharness`.

#### Details

Keep the top-level skill surface intentionally small. `execute` should own the
large loop but delegate detail to references. `reviewer` should be specialized
for reviewer subagents and should not assume the main agent is doing review
submission itself.

#### Expected Files

- `.agents/skills/discovery/SKILL.md`
- `.agents/skills/plan/SKILL.md`
- `.agents/skills/execute/SKILL.md`
- `.agents/skills/execute/references/*.md`
- `.agents/skills/land/SKILL.md`
- `.agents/skills/reviewer/SKILL.md`
- `.agents/skills/reviewer/references/*.md`

#### Validation

- The skill pack is internally coherent and references `harness --help` and
  `harness <subcommand> --help` instead of duplicating CLI truth unnecessarily.
- `execute` explicitly documents one active review round at a time.
- `execute` explicitly documents that reviewer fan-out must wait for all
  reviewer subagents to finish before `harness review aggregate`.
- `execute` or `reviewer` explicitly documents that reviewer subagents must be
  closed after their results are captured to avoid dangling background agents.

#### Execution Notes

Drafted the first repo-local skill pack with the five agreed top-level skills:
`discovery`, `plan`, `execute`, `land`, and `reviewer`. `execute` now points to
references for resume/status, step inner loop, review orchestration,
publish/CI/sync, and closeout/archive. The review orchestration draft includes
the Codex-specific rules to wait for all reviewer subagents, then close them
after their results are consumed.

#### Review Notes

Two reviewer subagents and two pure-context execute testers validated the skill
pack. The first pass surfaced one important installer issue, one minor stale
binary documentation issue, and one discoverability ambiguity around
controller-vs-reviewer responsibilities. A second delta review passed cleanly
after those fixes, and the archive-gate full review (`review-003-full`) also
passed cleanly.

### Step 4: Dogfood the docs and skills against this repository

- Status: completed

#### Objective

Use the new docs and skill contracts to confirm that the repository can steer
its own next loop coherently.

#### Details

The validation should exercise the human-facing entrypoints and the agent-facing
contracts together, not just lint markdown in isolation.

#### Expected Files

- `README.md`
- `AGENTS.md`
- `.agents/skills/**`
- `internal/plan/current.go`
- `internal/plan/current_test.go`
- optional small supporting docs if they improve dogfood legibility

#### Validation

- A fresh agent can identify the intended workflow from `README.md`,
  `AGENTS.md`, and the skill pack without repository-specific hidden context.
- The documented `harness` invocation path works in the current repo.
- Any claimed behavior-changing setup or helper logic is validated by tests or
  deterministic smoke runs.

#### Execution Notes

Dogfooding used both repository-local commands and pure-context subagents:

- ran `go test ./...`
- ran `scripts/install-dev-harness --help`
- ran `scripts/install-dev-harness`
- verified `command -v harness`, `harness --help`, and `harness status`
- started `review-001-delta`, waited for all reviewer subagents, explicitly
  closed them, and aggregated the round
- used a fresh subagent that only read `AGENTS.md`, the execute skill, and
  `harness status` to verify resumability

Dogfooding also surfaced a real status-hand-off bug: an archived
`.local/harness/current-plan.json` could mask a newer active plan. Fixed that in
`internal/plan/current.go` and added regression coverage in
`internal/plan/current_test.go`.

After fixing the review findings, ran `review-002-delta` with the same
wait-all / close-all reviewer flow. That round passed cleanly. A second fresh
execute tester confirmed the controller remains in `execute` during active
review while spawned reviewer subagents use `reviewer`.

#### Review Notes

Dogfood validation passed after the follow-up fixes. The remaining deferred work
is the already-tracked backlog in #2, #4, #5, and #6 rather than new findings
from this slice.

## Validation Strategy

- Keep repo-level truth layered:
  README for humans, `AGENTS.md` for repo-level agent norms, and skills for
  execution detail.
- Prefer deterministic command validation over vague prose claims.
- Any new executable setup path for `harness` should be smoke-tested in this
  repo.
- Review the skill pack for clarity, trigger conditions, and hidden-context
  leakage before treating it as dogfood-ready.

## Risks

- Risk: The new docs and skills may reintroduce the same top-level complexity
  that `superharness` is meant to remove from `missless`.
  - Mitigation: Keep the top-level skill set to five entries and push detail
    into `execute` references instead of more peer skills.
- Risk: The repo may document `harness` as a direct command without actually
  making that command reliable in development.
  - Mitigation: Treat the invocation path as part of the deliverable and test
    it explicitly.
- Risk: Reviewer orchestration guidance may stay too generic and fail to encode
  Codex-specific realities around async subagents.
  - Mitigation: Write the wait-for-all and explicit-close rules directly into
    the `execute` and `reviewer` skills.

## Validation Summary

- Automated:
  - `go test ./...`
- Deterministic smoke:
  - `scripts/install-dev-harness --help`
  - `scripts/install-dev-harness`
  - `command -v harness`
  - `harness --help`
  - `harness status`
  - `harness plan lint /Users/yaozhang/Workspace/superharness/docs/plans/active/2026-03-18-readme-agents-and-skill-pack.md`

The new direct-command install path works in this repo, and the dogfood-driven
current-plan detection fix is covered by an automated test.

## Review Summary

- `review-001-delta` found:
  - one blocking `important` finding: installer success could be shadowed by a
    different `harness` earlier on `PATH`
  - one `minor` finding: docs did not say to rerun the installer after Go code
    changes
- A fresh execute tester also found that the controller / reviewer skill split
  was discoverable but too implicit.
- Fixed those issues in the installer, README, `AGENTS.md`, and execute skill.
- `review-002-delta` passed with no blocking or non-blocking findings.
- `review-003-full` passed with no blocking or non-blocking findings and now
  satisfies the revision-1 archive gate.
- Fresh execute testing after the follow-up fixes confirmed the controller
  stays in `execute` while spawned review agents use `reviewer`.

## Archive Summary

- Archived At: 2026-03-18T23:49:13+08:00
- Revision: 1
moved to `docs/plans/archived/`
- PR: not opened yet; open the PR after committing and pushing the archive move
- Ready:
  - README, `AGENTS.md`, the first skill pack, and the dev install path are in
    place
  - dogfood review, fresh-agent testing, and the archive-gate full review
    passed after follow-up fixes
  - deferred items already point at GitHub issues
- Merge Handoff:
  - archive the plan, commit the tracked move plus repo changes, push the
    branch, open the PR, and then wait for merge approval or run `land` if
    explicitly asked to merge

## Outcome Summary

### Delivered

- first `README.md` for human-facing onboarding
- first `AGENTS.md` for repo-level Codex working agreement
- first repo-local skill pack with `discovery`, `plan`, `execute`, `land`, and
  `reviewer`
- execute references for resume/status, step loop, review orchestration,
  publish/CI/sync, and closeout/archive
- `scripts/install-dev-harness` for development-time direct-command setup
- a dogfood-driven fix in `internal/plan/current.go` plus regression coverage in
  `internal/plan/current_test.go`

### Not Delivered

- `harness ui`
- `harness plan list`
- broader skill-system expansion beyond the first dogfoodable pack
- shared test-fixture infrastructure

### Follow-Up Issues

- #2 `harness ui`
- #4 `harness plan list` / docs navigation
- #5 later skill-system expansion
- #6 shared test infrastructure
