# Step Inner Loop

The inner loop is how you finish one plan step cleanly.

## Inner Loop

1. Confirm the current step from `harness status` and the tracked plan.
2. For behavior changes, run Red/Green/Refactor:
   - Red: write or update a test that fails for the intended behavior.
   - Green: implement the smallest change that makes the test pass.
   - Refactor: improve structure without changing the behavior you just proved.
3. If TDD is genuinely impractical for this slice, record the reason in
   `Execution Notes` before continuing.
4. Run focused validation for the slice.
5. Update the step's `Execution Notes` with a concise summary.
6. If the slice is green and meaningfully reviewable, make a small commit so a
   later review has a durable git boundary.
7. Run `harness status` before step closeout so the next action reflects the
   current step, any active review, and any warning-driven follow-up.
8. Mark the step complete when its objective, validation, and durable notes are
   genuinely satisfied. Routine step completion creates no review debt and
   needs no no-review marker.
9. Start a step-bound review only when an intermediate artifact crosses a
   concrete risk boundary that should be independently frozen before later
   work, such as a schema/API contract, security boundary, migration, or
   irreversible side effect. First run the `Pre-Review` scan in
   [controller-truth-surfaces.md](controller-truth-surfaces.md).
10. If an intentional step review finds blocking issues, fix them, rerun
    focused validation, complete a linked repair review, and update `Review
    Notes` before advancing. Once started, that review remains binding.

## Step Notes

Keep step-local notes useful to the next agent:

- `Execution Notes`
  - what changed, what was validated, what remains
- `Review Notes`
  - record the latest review outcome, major findings, and repair only when an
    optional step review actually ran
  - otherwise a concise statement that formal review is deferred to finalize
    is enough; no magic marker or justification is required

Keep these notes high-signal and brief. Summarize the core change and outcome;
do not turn them into transcripts.

Do not wait until archive to reconstruct step history from memory.
