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
8. once those remote facts exist, run the `Pre-Land` scan from
   [controller-truth-surfaces.md](controller-truth-surfaces.md) before treating
   the archived candidate as genuinely merge-ready
9. only then treat the candidate as ready to enter
   `execution/finalize/await_merge`

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
