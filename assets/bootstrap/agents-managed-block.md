## Harness Working Agreement

1. Humans steer. Agents execute.
2. Approved scope lives in a git-tracked plan.
3. Raw execution trajectory lives under the local runtime root resolved with
   `harness repo config get paths.local_runtime`, and is disposable.
4. Durable summaries, contracts, and behavior changes must be written back to
   tracked docs or code before archive.
5. Evidence beats memory. Use `harness status`, tracked plans, and owned local
   artifacts instead of relying on long-session recall.
6. Keep harness workflow artifacts such as plans, summaries, and review notes
   in English unless repo-owned instructions explicitly set a different
   language policy.

## Harness Source of Truth

Harness resolves these workflow roots from repo config:

- active plan root, resolved with
  `harness repo config get paths.plans.active` and defaulting to
  `docs/plans/active/`: active markdown-led plan packages and any matching
  `supplements/` companion directories
- archived plan root, resolved with
  `harness repo config get paths.plans.archived` and defaulting to
  `docs/plans/archived/`: standard archived plan packages and durable
  closeout summaries
- local runtime root, resolved with
  `harness repo config get paths.local_runtime` and defaulting to
  `.local/harness/`: disposable runtime state, review artifacts, evidence
  artifacts, trajectory, and `plans/archived/` lightweight archived snapshots

Repository-specific durable contracts may live in `docs/specs/` or anywhere
else repo-owned instructions, plans, docs, or code say they live. Do not assume
`docs/specs/` exists, or that any repo-local specs are upstream easyharness
product contracts.

For Codex, bootstrap installs managed harness workflow skills under
`.agents/skills/` by default unless a different skills target was used. When
bootstrap-installed skills are present, easyharness-managed `harness-*` skills
are refreshed by bootstrap. Other repo-owned skills stay outside easyharness
ownership. If a tracked plan conflicts with a repo-local skill, the tracked
plan wins.

## Harness Product Help

When easyharness product behavior, command concepts, or repo customization
syntax is unclear, use `harness help` or `harness help <topic>` from the
installed binary. Use command `--help` for syntax and flags. After reading
topic guidance, apply it to the requested outcome and report the effective
result back to the human.

## Harness Workflow

For work coordinated through harness that is not already clear enough to
execute directly:

1. Discovery
2. Plan
3. Plan approval
4. Execute
5. Archive / publish / await merge approval
6. Land

Plan approval is explicit. Writing a plan or hearing the original task request
does not by itself approve execution. After the plan is shown and the human
approves it, the agent should record that boundary with
`harness plan approve --by human` before `harness execute start`.

For approved low-risk work that explicitly uses `workflow_profile:
lightweight`, keep the same workflow shape but store the active plan under the
active plan root resolved by `harness repo config get paths.plans.active` like
any other plan. Only the archived lightweight snapshot moves under the local
runtime root resolved by `harness repo config get paths.local_runtime`. That
shortcut does not remove human steering, low-risk eligibility checks, or the
requirement to leave a repo-visible breadcrumb such as a readable PR body
merge memo.

Use `lightweight` only when all of these are true:

- the human explicitly approves using `workflow_profile: lightweight`
- the plan is sized `XXS`
- the whole slice is one bounded low-risk change
- the edits stay within a narrow surface such as README/docs/comments/copy, a
  small CI condition adjustment, a tiny helper-script fix, or another similarly
  small change whose blast radius is easy to explain
- no schema-meaning changes, core state/review/archive/evidence changes,
  release-safety changes, or security-sensitive logic changes
- if the boundary is unclear, default to `standard`

`size` and `workflow_profile` are separate decisions. `XXS` is the only size
eligible for `lightweight`, and `XXS` may still use the ordinary `standard`
workflow when that is the approved path.

When drafting a new plan, estimate `size` early. If the initial estimate is
`XXL`, stop and confirm with the human whether the work should be split first;
if the split is unclear, return to discovery to settle a better split before
execution approval. If the work still proceeds as `XXL`, move obvious spillover
into `Deferred Items` or follow-up issues instead of letting the oversized plan
quietly absorb extra scope. `XXL` remains available for truthful historical
sizing and rare coherent large slices, but it should not be the routine
starting point for new work.

Use `harness reopen --mode finalize-fix|new-step` when an archived candidate
is no longer merge-ready because of new feedback, remote changes, or other
invalidation.

## Harness Subagent Use

The controller owns shared repository context and the final workflow judgment.
Subagents are a normal part of harness work, not an exceptional fallback. When
a harness workflow skill is first used in a thread, ask once whether the human
authorizes specific, well-scoped subagents for this harness run unless that
authorization has already been explicit in the conversation. This authorization
covers explorer, worker, and reviewer subagents.

After authorization, actively look for bounded, independent work that
subagents can handle in parallel or with useful separation. Spawn subagents
only for concrete subproblems; do not split one shared context bundle across
multiple subagents just to get summaries back.

Discovery and execution may still stay local, use one subagent, or use
multiple subagents in parallel according to the current question shape:

- stay local when the controller can answer the next question from the shared
  context it already needs to hold
- use `1` when one bounded question or hypothesis needs independent repo
  checking
- use multiple subagents in parallel only when multiple hypotheses or
  questions are genuinely independent

If subagent authorization is missing when it becomes useful, ask for that
authorization before spawning. If authorization is denied, continue locally
until the human changes that boundary.

In Codex, spawned subagents are not fire-and-forget memory. Once a bounded
subagent task is complete and the controller has received the result, close
that subagent promptly by default. Reuse `resume_agent` only when a later
narrow follow-up makes continuity materially more valuable than a fresh agent.

## Harness Review Execution

When work enters review orchestration, spawned reviewer subagents are the
default path. The controller agent stays in `harness-execute`, reviewer work
belongs to spawned `harness-reviewer` subagents, and the repo-local review
skills must be followed strictly. The shared rules in `Harness Subagent Use`
still apply here; review-specific docs add reviewer-slot orchestration,
aggregation, and same-slot resume rules on top of that shared baseline.

The controller must not submit reviewer results on a reviewer's behalf. Each
reviewer submission should be recorded through
`harness review submit --by <reviewer-name>`
from the bounded reviewer thread that owns that slot.

Routine review progression is controller-owned once a tracked plan is approved.
The controller should not stop to ask the human whether ordinary step-closeout
or finalize review should begin.

For `delta` review, use a real git commit anchor so later agents know the
default starting point for the reviewed change.

Use `harness status` at routine checkpoints:

- when starting or resuming execution
- before marking a step done
- after each review aggregate
- before relying on later-step or finalize progression after a warning or fix

Human confirmation is still required for real blockers, scope changes, and
merge approval, but not for ordinary review closeout.

## Harness Start Points

When entering the repository or resuming after compaction:

1. Read `README.md` if you need repository purpose or setup context.
2. Run `harness status`.
3. If `harness status` reports a current plan artifact, open that plan.
   Active work always uses a tracked plan under the active plan root resolved
   by `harness repo config get paths.plans.active`; archived lightweight
   candidates may live under the local runtime root resolved by
   `harness repo config get paths.local_runtime`.
   If status reports `idle`, there is no current plan to resume yet.
4. If status reports approved or in-progress harness execution, most resumed
   work should continue in `harness-execute`.
5. Switch only when `harness status` and the workflow boundary clearly call for
   a different skill:
   - `harness-discovery` when direction is unclear
   - `harness-plan` when creating or revising a tracked plan
   - `harness-land` only when `state.current_node` is
     `execution/finalize/await_merge` and a human has explicitly approved
     merge
   - `harness-reviewer` only inside spawned reviewer subagents
