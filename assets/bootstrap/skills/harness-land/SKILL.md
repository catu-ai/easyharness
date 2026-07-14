---
name: harness-land
description: Merge an archived candidate after explicit human approval and complete required post-merge bookkeeping.
---

# Harness Land

## Authority

Use this skill only after the human explicitly approves merge and `harness
status` reports `execution/finalize/await_merge`. If the candidate is no longer
valid, reopen it and return to `harness-execute` instead of merging.

## Land

1. Verify the recorded PR, current CI, sync, and mergeability facts.
2. Merge using the repository's preferred strategy unless the human specifies
   another.
3. Record merge confirmation:

   ```bash
   harness land --pr <url> [--commit <sha>]
   ```

4. Complete the permanent PR and issue handoff: record the merged reference,
   close fully resolved issues, and leave explicit pointers for partial or
   deferred work.
5. Finish local branch cleanup and sync.
6. Run `harness land complete` only after required bookkeeping is done. Confirm
   `harness status` returns to `idle` with landed context preserved.

Do not merge without explicit approval, edit the archived plan to add merge
metadata, silently close partial issues, or complete land before its durable
bookkeeping is finished.
