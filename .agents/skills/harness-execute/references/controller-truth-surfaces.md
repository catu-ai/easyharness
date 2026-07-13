# Controller Truth Surfaces

Use this checklist at high-risk controller transitions. It is a short
self-check for the controller, not a second reviewer protocol.

## Pre-Review

- scope truth
  - decide whether this round is really `delta` or `full`, and say why a
    narrower or broader pass would be less trustworthy
- anchor and diff truth
  - ensure intended candidate changes are committed and the worktree is clean
  - for `delta`, verify the review anchor is the prior covered git head and the
    repair link matches the intended findings
- contract scan
  - scan the active plan, touched contracts, docs wording, and focused
    validation so the controller does not outsource all completeness checking
    to reviewers
- dispatch sanity
  - make sure one integrated assignment owns whole-candidate review
  - add only specialists justified by concrete risk surfaces and invariants;
    plan size alone is not a trigger
  - make sure the review spec and reviewer prompt carry the actual round
    context, captured head, anchor, repair link, role, guidance, and bounded
    change summary instead of forcing reviewers to guess

## Pre-Aggregate

- reviewer identity truth
  - verify each submission came from a bounded reviewer slot using
    `harness review submit --by <reviewer-name>`, not from the controller thread standing
    in as its own reviewer
- submission truth
  - verify every expected slot submitted a real result rather than a missing,
    invalid, or still-skeleton artifact
- round-state truth
  - verify you are aggregating the current active round and not mixing older
    findings, newer repairs, or the wrong revision
  - verify candidate HEAD still equals the round's captured reviewed head
- synthesis sanity
  - read the submitted findings once before aggregation so obvious duplicates,
    missing severities, or malformed claims do not slide through by inertia

## Pre-Archive

- placeholder debt
  - replace placeholder summaries, unchecked acceptance criteria, and step
    markers before archive instead of letting `archive` discover them late
- narrative debt
  - make sure the tracked plan tells a durable story of what changed, how it
    was validated, what review concluded, and what follow-up remains
- publish-readiness sanity
  - confirm the branch is truly in archive closeout rather than still needing
    review, repair, or unresolved handoff work
  - confirm review coverage is a continuous full-plus-repair-delta chain and
    product/source changes have not moved beyond its covered head

## Pre-Land

- PR truth
  - run `harness status` first and inspect the recorded PR handoff facts instead
    of trusting a stale local impression of readiness
- CI truth
  - verify the latest relevant runs through status or refresh output and
    distinguish `success` from cancelled, stale, superseded, failed, or
    still-running checks
- sync truth
  - confirm status shows fresh local sync evidence and no live remote stale or
    conflicted handoff warning before merge-sensitive handoff or merge work
- merge and bookkeeping truth
  - remember that status is read-only: PR creation, reruns, comments, issue
    follow-up, merge, and land bookkeeping remain human/agent actions outside
    harness core commands
