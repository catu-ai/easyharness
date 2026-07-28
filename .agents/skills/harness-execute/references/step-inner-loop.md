# Step Inner Loop

Finish the current step's outcome, not a prescribed sequence of model actions.

For a standard or lightweight root, read the current root step. For a
coordinated child, use `harness status --plan <subplan-id-or-path>` and read
that child's current step. Coordinated children do not receive separate
lifecycle commands or formal reviews.

1. Read the step outcome, covered acceptance criteria, and optional check.
2. Implement the smallest coherent change that satisfies them.
3. Validate the changed behavior and relevant failure paths.
4. Commit a useful boundary when the slice is coherent, then mark the step
   complete when its outcome is genuinely satisfied. When the final child step
   completes, replace its Result placeholders with concise validation and
   delivery outcomes.

Use focused validation or bounded advisor subagents for intermediate
uncertainty. Formal review belongs to the complete finalize candidate. Do not
turn step progress into a tool transcript or frequent plan writes.
