---
template_version: 0.2.0
created_at: "2026-04-19T21:40:36+08:00"
approved_at: "2026-04-19T21:41:50+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/164
size: M
---

# Add machine-local watchlist touch foundation

## Goal

Add the first machine-local watchlist write path so successful core harness
workflow commands can silently register or refresh the current workspace
without changing their user-facing output semantics. The implementation should
make the watchlist a best-effort dashboard/index side effect rather than a new
workflow control-plane dependency.

This slice should keep the persisted watchlist shape small and stable for the
first dashboard foundation. Use a single `watchlist.json` index under an
easyharness home directory, support an `EASYHARNESS_HOME` override for that
root, and defer any future per-workspace directory layer until the dashboard
actually needs richer workspace-local data.

## Scope

### In Scope

- Add a shared machine-local watchlist writer that persists the current
  workspace in `watchlist.json` under `EASYHARNESS_HOME` when set or
  `~/.easyharness` by default.
- Normalize the current workspace path before duplicate detection and store one
  record per canonical workspace path with stable `watched_at` plus refreshed
  `last_seen_at`.
- Make watchlist touch a best-effort side effect of successful core workflow
  services rather than a command-name allowlist or low-level file-access hook.
- Treat `harness status`, lifecycle commands, review commands, and evidence
  submit as the initial core workflow surfaces that confirm the current
  workspace locally.
- Keep `harness ui`, bootstrap/resource commands, `harness plan template`,
  `harness plan lint`, `--version`, and help paths out of the touch flow.
- Update the watchlist spec and README anywhere the new home-directory
  override or best-effort writer semantics become part of the public contract.
- Add focused tests for path normalization, record convergence, concurrent-safe
  rewrite behavior, core-versus-non-core command boundaries, and non-fatal
  watchlist write failures.

### Out of Scope

- Introducing persisted `workspace_key` fields or a `workspaces/<key>/`
  directory layer in this slice.
- Building `harness dashboard`, changing `harness ui` routing, or adding UI
  write actions such as `Unwatch`.
- Adding watchlist touch to non-core utility/bootstrap commands just because
  they run in a repository.
- Treating watchlist persistence failures as command failures or surfacing new
  banners/prompts in ordinary command output.

## Acceptance Criteria

- [x] Successful completion of core workflow services silently registers or
      refreshes the current workspace in machine-local `watchlist.json`.
- [x] Repeated touch of the same canonical workspace path converges to one
      record, preserves `watched_at`, and may refresh `last_seen_at`.
- [x] The watchlist root resolves from `EASYHARNESS_HOME` when set and falls
      back to `~/.easyharness` otherwise; the config surface is documented in
      the README and any affected spec text.
- [x] `harness status` still registers idle workspaces, while `harness ui`,
      `harness plan lint`, bootstrap/resource commands, `--version`, and help
      do not touch the watchlist.
- [x] Watchlist rewrites are crash-safe and do not silently drop unrelated
      workspace records during ordinary concurrent use.
- [x] Watchlist write failures are treated as best-effort side effects: the
      primary command still succeeds and preserves its existing output/exit
      semantics.
- [x] The tracked plan and tests make the future two-layer direction explicit
      as deferred work, without prematurely persisting workspace keys.

## Deferred Items

- Add a future per-workspace local data layer such as
  `~/.easyharness/workspaces/<derived-key>/` if the dashboard later needs
  richer workspace-local caches or remote-signal materialization.
- Add explicit `unwatch` behavior, degraded-route cleanup actions, or any
  archive/GC policy for unwatched workspaces.
- Revisit whether future dashboard routing should persist a workspace key once
  the read-time route-key contract and any per-workspace storage layer are
  ready to converge together.

## Work Breakdown

### Step 1: Define the watchlist storage and configuration contract

- Done: [x]

#### Objective

Lock the v1 watchlist foundation around one machine-local `watchlist.json`
index plus an overridable easyharness home root, without expanding into a
persisted workspace-key or per-workspace directory contract yet.

#### Details

Update the normative watchlist contract to describe the default home root plus
`EASYHARNESS_HOME` override, clarify that repeated touch converges on one
canonical workspace record while preserving `watched_at`, and make best-effort
touch semantics explicit where the command-trigger language becomes normative.
Keep the file shape minimal: `version`, `workspace_path`, `watched_at`, and
`last_seen_at`.

The README should explain the environment variable only as a configuration
surface for the machine-local easyharness home root; it should not imply that
dashboard or workspace-local storage is already broader than this slice.

#### Expected Files

- `docs/specs/watchlist-contract.md`
- `README.md`

#### Validation

- A cold reader can tell where `watchlist.json` lives by default, how
  `EASYHARNESS_HOME` overrides that root, and that v1 still persists only the
  single-file watchlist index.
- The contract language matches the accepted direction: no persisted
  `workspace_key`, no per-workspace folder yet, and watchlist touch is
  best-effort.

#### Execution Notes

Updated `docs/specs/watchlist-contract.md` so the storage root now supports
`EASYHARNESS_HOME` with `~/.easyharness` as the default, repeated touch is
described as canonical-path convergence plus `last_seen_at` refresh, and
watchlist persistence is explicitly documented as a best-effort side effect
for successful core workflow commands. Updated `README.md` with the new
machine-local home override note without overclaiming that a per-workspace data
layer already exists.

#### Review Notes

PASSED after `review-001-full` requested changes on duplicate convergence and
command-boundary coverage, `review-002-delta` passed correctness but left one
remaining tests gap on excluded-surface coverage, and `review-003-delta`
passed after the final CLI no-touch matrix repair.

### Step 2: Add a shared best-effort watchlist writer

- Done: [x]

#### Objective

Implement one shared watchlist persistence path that can register or refresh
the current workspace safely without changing the behavior of core workflow
commands when the local watchlist write fails.

#### Details

Add a small machine-local watchlist package that resolves the easyharness home
root, normalizes the current workspace path, loads the current watchlist,
converges duplicate registration on the canonical path, preserves
`watched_at`, refreshes `last_seen_at`, and rewrites the file with crash-safe
replacement. The writer should coordinate concurrent writes so one command does
not silently lose another workspace record.

The API should make it easy for callers to treat touch as best-effort. Prefer
returning an error that the caller may intentionally suppress rather than
mixing watchlist failures into lifecycle/review/evidence/status result
contracts.

#### Expected Files

- new watchlist package under `internal/`
- any shared atomic-write or lock helpers that need a narrow extension
- focused tests for watchlist persistence behavior

#### Validation

- Tests cover default-root resolution, `EASYHARNESS_HOME` override, path
  convergence, `watched_at` preservation, `last_seen_at` refresh, and
  concurrent-safe rewrite behavior.
- A simulated watchlist write failure can be observed by the caller without
  forcing the watchlist package itself to own command-level rollback behavior.

#### Execution Notes

Added `internal/watchlist` with an overridable home-root resolver, canonical
workspace-path normalization via absolute-path plus symlink resolution, one
record per canonical path, stable `watched_at`, refreshed `last_seen_at`,
crash-safe atomic rewrite, and a serialized lock file under the easyharness
home root. Focused tests cover the default root, `EASYHARNESS_HOME` override,
canonical path convergence, repeated touch semantics, concurrent writes that
preserve unrelated workspace records, and non-adjacent duplicate-record
coalescing after the first review round surfaced a merge bug. Finalize repair
after `review-007-full` tightened the watched path to the git workspace root
instead of an arbitrary successful command cwd, and the package now returns a
non-git sentinel when a caller points it at a directory outside any checkout.
After `review-009-full` found that a fake `.git` marker could still slip
through, the writer now resolves the workspace root through
`git rev-parse --show-toplevel` so only real git-backed checkouts qualify for
registration. After `review-013-full`, the writer also resolves relative
`EASYHARNESS_HOME` overrides under the user's home directory so one override
value still points at one stable machine-local root instead of fragmenting by
caller cwd. Follow-up delta repair now also rejects parent-directory forms
such as `../escape` so a relative override cannot silently walk back out of
the user-home anchor.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 shipped as part of the same integrated watchlist
foundation slice that Step 1 review already examined broadly, including the
writer implementation and its regression tests.

### Step 3: Hook core workflow services without command-shape heuristics

- Done: [x]

#### Objective

Touch the watchlist only when a core workflow service successfully confirms the
current workspace, while keeping non-core commands and `harness ui` outside the
flow.

#### Details

Wire a shared best-effort touch callback into the existing service-success
boundaries instead of matching on command names or guessing from result JSON
shapes. Use the existing lifecycle/review/evidence post-success hooks where
available, and extend `status` with a similarly narrow success hook so idle
status reads still register the workspace.

Do not add touch behavior in low-level runstate or UI read paths. `harness ui`
must remain excluded even though it reads workspace state, so dashboard/UI
polling cannot inflate recency.

#### Expected Files

- `internal/cli/app.go`
- `internal/status/service.go`
- `internal/lifecycle/service.go`
- `internal/review/service.go`
- `internal/evidence/service.go`
- command/service tests that pin core versus non-core behavior

#### Validation

- Successful `status`, lifecycle, review, and evidence flows touch the
  watchlist.
- `harness ui`, `harness plan lint`, bootstrap/resource commands, `--version`,
  and help do not.
- Watchlist touch failures do not change command JSON payloads or exit codes on
  otherwise successful commands.

#### Execution Notes

Added success-only best-effort callbacks to `status`, `lifecycle`, `review`,
and `evidence` services so watchlist touch can run after the existing strict
timeline hooks without changing command failure behavior. `internal/cli/app.go`
now wires those callbacks through one shared watchlist toucher using the
current workdir plus injected env/home resolvers. Non-core commands remain
untouched because they never receive the new success callback. Added CLI tests
that prove idle `harness status` creates a watchlist record, `harness plan
lint` does not, and a watchlist home-resolution failure still leaves `status`
successful. Follow-up delta repairs expanded the CLI matrix so `plan template`,
`skills install --dry-run`, `--version`, root help, `init --dry-run`, and
`ui --help` are all pinned as no-touch surfaces, while lifecycle, review, and
evidence families each have positive touch coverage. Finalize repair after
`review-007-full` moved the best-effort touch trigger into one shared
successful-result postprocessor in `internal/cli/app.go`, which keeps the
core-service boundary intact while removing repeated per-entrypoint callback
plumbing. After `review-009-full`, the CLI regression matrix now also pins
watchlist registration for `plan approve`, `archive`, `land`, `land complete`,
and both `reopen` modes so the centralized lifecycle route stays explicit in
tests rather than relying on one representative command. The last repair also
adds linked git worktree coverage plus a real `/api/status` UI no-touch test
so the core workflow contract is pinned for both accepted workspace forms and
the excluded steering surface. Follow-up repair broadens that UI coverage
across `/api/status`, `/api/plan`, `/api/review`, `/api/timeline`, `/`, and
normal `harness ui` startup so the entire excluded steering surface is pinned
against accidental watchlist writes.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 was not an independently reviewable slice; the
same integrated Step 1 review boundary covered the service-hook wiring and CLI
command-boundary behavior together with the storage contract changes.

### Step 4: Prove the contract with focused command and documentation coverage

- Done: [x]

#### Objective

Close the slice with repository-visible proof that the watchlist foundation is
 durable, non-invasive, and understandable without discovery chat.

#### Details

Add or update focused tests around the watchlist package plus CLI/service
boundaries, and run the relevant Go test targets that exercise the new
best-effort behavior. Validation should prove the command-success path for
status and at least one representative lifecycle/review/evidence command, plus
the exclusion path for `harness ui` or another non-core command.

Make sure the final tracked docs explain the new home-root override and do not
overclaim that the per-workspace storage layer already exists.

#### Expected Files

- watchlist package tests
- affected service or CLI tests
- `README.md`
- `docs/specs/watchlist-contract.md`
- `docs/plans/active/2026-04-19-add-machine-local-watchlist-touch-foundation.md`

#### Validation

- Focused Go tests for the watchlist package and affected command/service
  boundaries pass.
- `harness plan lint` passes on this tracked plan.
- A future agent could execute the slice from this plan alone and understand
  both the current single-file contract and the deferred two-layer direction.

#### Execution Notes

Validated the slice with `go test ./internal/watchlist ./internal/cli
./internal/status ./internal/lifecycle ./internal/review ./internal/evidence
-count=1` and `go test ./... -count=1`. Also re-ran `harness status` as a
controller checkpoint after implementation to confirm the tracked plan still
resolves cleanly before review orchestration. After `review-007-full`
requested changes, added focused CLI coverage for non-git `status` no-touch
behavior and upgraded the review command fixture to use a real git repository
anchor before rerunning `go test ./internal/watchlist ./internal/cli
./internal/status ./internal/lifecycle ./internal/review ./internal/evidence
-count=1`. After `review-009-full`, replaced fake git markers in the watchlist
and CLI fixtures with real initialized repositories, added explicit lifecycle
watchlist assertions for the remaining routed commands, and reran the same
focused Go test set to confirm the repair.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 4 is repository-level validation and closeout
proof for the integrated slice rather than a separate behavior surface, so it
rides on the Step 1 review boundary plus the recorded validation commands.

## Validation Strategy

- Use focused Go tests for the new watchlist package and the affected CLI or
  service boundaries instead of relying only on manual command runs.
- Re-run the most relevant package-level tests for `status`, `lifecycle`,
  `review`, and `evidence` wherever the watchlist-success hooks land.
- Lint the tracked plan before asking for approval.

## Risks

- Risk: Path canonicalization or concurrent rewrite logic could still allow
  duplicate or lost records under real-world local usage.
  - Mitigation: Centralize normalization and rewrite behavior in one shared
    package with focused concurrency/duplicate tests instead of scattering file
    mutation across commands.
- Risk: Hooking touch at the wrong layer could accidentally include `harness ui`
  or utility commands, or could turn watchlist failures into workflow
  failures.
  - Mitigation: Wire touch only at explicit core-service success boundaries and
    pin exclusion/non-fatal behavior in tests.
- Risk: Introducing `EASYHARNESS_HOME` could drift from docs or imply broader
  storage semantics than this slice actually implements.
  - Mitigation: Update the spec and README in the same slice and state
    explicitly that v1 still persists only `watchlist.json`.

## Validation Summary

- `harness plan lint docs/plans/active/2026-04-19-add-machine-local-watchlist-touch-foundation.md` passes after the final closeout notes.
- Focused suites pass for the touched surfaces, including
  `go test ./internal/watchlist ./internal/cli ./internal/status ./internal/lifecycle ./internal/review ./internal/evidence ./internal/ui -count=1`.
- Full repository validation passes with `go test ./... -count=1`.
- Targeted regressions now cover linked git worktrees, stable and rejected
  relative `EASYHARNESS_HOME` overrides, lifecycle command touch routes, and
  the excluded `harness ui` steering surface.

## Review Summary

- The candidate closed with a clean finalize full review in `review-019-full`.
- Earlier finalize rounds surfaced and repaired: duplicate-record merge
  convergence, missing no-touch and failure-path coverage, non-git workspace
  registration, incomplete lifecycle/UI coverage, fake `.git` acceptance,
  relative-home fragmentation/escape behavior, and README/spec drift around
  `EASYHARNESS_HOME`.
- Final reviewer consensus is that the watchlist writer, CLI touch routing,
  docs contract, and excluded UI surface now align with the accepted slice.

## Archive Summary

- Archived At: 2026-04-19T23:01:18+08:00
- Revision: 1
- PR: Not created yet; publish handoff after archive should push the branch and
  open or refresh the PR before evidence submission.
- Ready: Acceptance criteria are satisfied, finalize full review
  `review-019-full` passed cleanly, and local validation covers the writer,
  CLI/service boundaries, linked worktrees, and excluded UI surfaces.
- Merge Handoff: Run `harness archive`, commit the tracked archive move plus
  these closeout notes, push the branch, open/update the PR, record publish/CI/
  sync evidence, and stop once `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added a shared machine-local watchlist writer under `internal/watchlist`
  with crash-safe rewrite, duplicate convergence, stable `watched_at`,
  refreshed `last_seen_at`, and canonical git-backed workspace-root
  registration.
- Routed best-effort watchlist touching through successful core workflow
  results for status, lifecycle, review, and evidence commands without
  changing their existing output/exit behavior.
- Documented `EASYHARNESS_HOME` plus the final contract for absolute and
  relative overrides, and pinned the excluded `harness ui` surface so read-only
  steering paths do not refresh recency.

### Not Delivered

- No per-workspace `workspaces/<derived-key>/` storage layer, persisted
  workspace key, explicit `unwatch`, or archive/GC flow was added in this
  slice.
- No dashboard write actions or broader `harness ui` behavior changes were
  introduced beyond protecting the existing read-only surface from watchlist
  writes.

### Follow-Up Issues

- No GitHub follow-up issue was created during this slice. Deferred follow-up
  remains captured in `## Deferred Items`: future per-workspace local data,
  explicit membership-removal behavior, and any later route-key persistence.
