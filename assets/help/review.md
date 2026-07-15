Review Help

Every completed standard candidate receives one independent integrated
finalize review. The reviewer owns the complete judgment across correctness,
acceptance, failure and permission behavior, tests and evidence, contracts,
scope, and residual risk. Harness automatically includes the plan's
`Review Focus`; the controller does not select dimensions or write a review
spec.

Start review after every tracked step is done:

```bash
harness review start
```

The first round is full. The returned assignment belongs to one spawned
reviewer using the repo-local `harness-reviewer` skill. That reviewer may spawn
bounded advisor subagents for focused checks, but advisors create no harness
slots or submissions and do not split ownership of the final judgment.

The reviewer records its own result with the returned round and submission
path:

```bash
harness review submit --round <round-id> --by <reviewer-name> --input <path>
```

A valid submission records the decision and coverage atomically; there is no
separate aggregate command. Candidate changes after the captured git head are
not covered and block archive.

If review finds a narrow issue, commit the repair and run
`harness review start` again. Harness infers a linked delta that resolves the
finding and extends coverage. If rewritten Git ancestry means the covered tip
is no longer an ancestor of the candidate, Harness automatically starts a new
full root instead of creating an unusable delta when the covered tip is clean.
If unresolved findings remain, automatic reset fails before round creation;
restore ancestry or explicitly use `harness review start --full` when a new
whole-candidate review should supersede them. Explicit full is also appropriate
when the repair materially changes candidate design, scope, or risk.

If an unfinished round cannot be completed, preserve it as aborted and clear
the active pointer through the supported recovery command:

```bash
harness review abort --round <round-id>
```

Abort never changes completed coverage or deletes round artifacts. Afterward,
run `harness review start`; Harness will infer a valid delta or full boundary.
