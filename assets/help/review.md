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
finding and extends coverage. Use `harness review start --full` only when the
repair materially changes candidate design, scope, or risk and therefore
invalidates the earlier full root.
