# Review Orchestration

Every candidate receives one independent integrated finalize review. The
controller stays in `harness-execute`; the reviewer subagent uses
`harness-reviewer` and owns the only submission.

## Start and Dispatch

Commit the candidate and keep the worktree clean. At finalize, run:

```bash
harness review start
```

The first round establishes full whole-candidate coverage. After a blocking
review and a narrow committed repair, `review start` infers the linked delta
from current coverage and unresolved findings. If rewritten ancestry makes the
clean covered tip no longer an ancestor of the current candidate, `review
start` automatically establishes a new full root before it creates any round.
When unresolved findings remain, restore ancestry or explicitly choose a full
replacement; Harness fails before creating a round rather than silently
discarding those obligations. Use
`harness review start --full` only when the repair changes design, scope, or
risk enough to invalidate the prior full judgment.

The command captures HEAD and returns the integrated reviewer handoff,
including the plan and Review Focus. Spawn one clean reviewer subagent and tell
it to use `harness-reviewer`. Pass the returned round ID, plan path, reviewed
HEAD, submission path, repair link, and concise change context. Do not invent
or select dimensions, slots, specialists, or instructions.

The reviewer may create bounded advisor subagents for concrete investigations.
Advisors report only to that reviewer. They do not call harness review commands
or submit separate verdicts.

## Completion and Repair

Reviewer completion is not review completion; wait for its successful
reviewer-owned `harness review submit`. Submission atomically verifies the
captured HEAD, records findings and resolutions, derives the decision, and
updates coverage. Run `harness status` after submission.

The controller must not submit, edit, or synthesize the reviewer result. If the
review requests changes, repair every blocking finding and validate the repair.
Then commit and start the inferred linked delta. Reuse the reviewer only when a
narrow follow-up benefits materially from continuity; otherwise use a fresh
reviewer.

A clean linked delta extends the full root without another full review.
Non-blocking findings do not require repair. Archive remains blocked when
findings are unresolved, candidate changes are uncovered, or the reviewed git
boundary moved.

If an unfinished round genuinely cannot be completed, do not edit local
runtime state. Run `harness review abort --round <round-id>`. The command marks
the historical round aborted, preserves its artifacts and existing finalize
coverage, clears only the active pointer, and allows a replacement `review
start` to select a valid delta or full boundary.
