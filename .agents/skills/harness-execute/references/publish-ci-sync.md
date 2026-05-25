# Publish, CI, and Sync

Once implementation is materially complete, the execute loop expands beyond the
current step and eventually into archived-candidate handoff.

## Publish and CI

1. commit reviewable progress
2. push the branch
3. open or update the PR
4. wait for required CI
5. fix failures
6. decide whether the repair needs delta review or full review

For archived candidates, use the same sequence as post-archive handoff work,
but record the observed external facts through harness-owned evidence:

1. commit the archive move
2. push the branch
3. open or update the PR
4. run `harness evidence submit --kind publish` once the PR or handoff target
   exists
5. when publish evidence records a supported PR URL, run
   `harness evidence refresh` to read CI and sync facts from that recorded PR
6. run `harness status` after refresh so the local summary and next actions
   reflect the evidence that was just written
7. if refresh degrades, is unavailable, or publish evidence lacks a recorded
   PR URL, use manual
   `harness evidence submit --kind publish|ci|sync --input <json>` for the
   affected evidence domains
8. write or update the PR body as a readable merge memo; see
   [PR Body Handoff](#pr-body-handoff)
9. once those remote facts exist, run the `Pre-Land` scan from
   [controller-truth-surfaces.md](controller-truth-surfaces.md) before treating
   the archived candidate as genuinely merge-ready
10. only then treat the candidate as ready to enter
   `execution/finalize/await_merge`

## PR Body Handoff

Treat the PR body as a human merge memo, not as a copied execution log. The
tracked plan and harness evidence already hold the full audit trail: scope,
acceptance criteria, validation commands, review rounds, repair history, and
publish/CI/sync facts. The PR body should summarize why the branch is
mergeable without asking the human to rereview the diff.

Use this shape unless the repository has a stricter local convention:

- `What Changed`
  - Lead with what is now true after the PR.
  - Write a readable explanation, not a file list, command list, commit-log
    rewrite, or pasted plan summary.
  - Use short paragraphs or bullets according to the change shape. Length is
    secondary to clarity.
- `Confidence`
  - Combine self-review and validation into three to five high-signal bullets.
  - Each bullet should name a checked risk surface and the result, such as
    contract sync, generated bootstrap output, remote experiment completeness,
    focused tests, review-found repairs, or diff hygiene.
  - Do not dump raw validation commands unless the command result itself is the
    user-facing fact, such as release version parsing.
- `Handoff`
  - Include only merge-time, release-time, follow-up, non-goal, known-gap, or
    deferred-work notes that still matter after reading the PR.
  - Omit the section when it has no useful content.

For lightweight work, the required repo-visible breadcrumb can be this PR body
memo. It should still explain why the lightweight path was appropriate, but it
should follow the same readable merge-memo standard.

## Status-First Remote Handoff

During archived-candidate handoff, treat `harness status` as the first
orientation surface. Read these parts together:

- `state.current_node`: the durable local workflow node
- local publish, CI, and sync evidence facts
- `facts.remote_handoff`: optional live read-only observation of the recorded PR
- `warnings` and `next_actions`: the controller-facing interpretation

The remote handoff surface is deliberately read-only and non-authoritative.
Passing checks or clean merge state in `facts.remote_handoff` explain what the
controller should do next; they do not move `state.current_node` to
`execution/finalize/await_merge` until local evidence records the facts.

Use these cases as the controller decision guide:

- No recorded PR evidence
  - Open or update the PR outside harness, then record publish evidence with
    `harness evidence submit --kind publish`.
  - Do not ask harness to infer a PR from the current branch, upstream, or local
    remote.
- Unsupported or non-PR handoff target
  - Keep using manual `harness evidence submit --kind ci|sync` for the affected
    evidence domains.
  - Use direct provider inspection only to decide what manual evidence to
    record.
- Degraded remote observation
  - Treat missing `gh`, missing auth, network or API failure, unreadable PRs, and
    malformed provider output as degraded remote reads, not as local workflow
    failures.
  - Retry `harness evidence refresh` when the provider problem is temporary, or
    manually submit the evidence when the facts are clear through another
    trusted surface.
- Pending remote checks
  - Wait for checks to finish, then rerun `harness evidence refresh`.
  - Do not mark CI successful from a still-running live observation.
- Failed or cancelled remote checks
  - Inspect the failing checks, fix the branch, rerun focused validation, and
    decide whether the repair needs delta or full review.
  - After the PR checks recover, use `harness evidence refresh` or manual CI
    evidence to update the local handoff facts.
- Passing remote checks or fresh merge state with missing or stale local
  evidence
  - Run `harness evidence refresh` so the observed facts become durable local
    CI and sync evidence.
  - Rerun `harness status` after refresh before reporting the candidate as
    merge-ready.
- Stale remote sync
  - Refresh the branch against the base outside harness, push the repaired
    branch, validate the change, and rerun `harness evidence refresh`.
- Conflicted remote sync
  - Resolve the conflict outside harness and treat the repair like any other
    merge-sensitive code or docs change: validate it and review it at the
    appropriate delta or full scope before refreshing evidence.
- Locally merge-ready but live remote facts have drifted
  - Keep `execution/finalize/await_merge` as the historical local evidence
    state, but do not tell the human the candidate is still ready to merge until
    the drift is repaired, refreshed, or intentionally recorded.
- PR already merged
  - Switch to `harness-land` only after explicit human merge approval. The land
    flow records merge confirmation and required post-merge bookkeeping; status
    observation alone is not a substitute for that boundary.

Direct `gh` inspection is a diagnostic fallback for confusing or degraded
status/refresh results. It is not the ordinary first path when `harness status`
and `harness evidence refresh` can carry the handoff.

## Remote Freshness

Refresh remote state before merge-sensitive handoff work.

Use `harness evidence refresh` as the ordinary refresh path when publish
evidence has a recorded PR URL. Direct `gh` inspection is a diagnostic fallback
for confusing or degraded refresh results, not the controller's first path for
routine CI/sync evidence refresh.

If remote changes introduce real conflict work:

- resolve the conflicts
- rerun focused validation
- run delta or full review depending on how broad the repair was

Do not create a new review round while an earlier one is still active.
