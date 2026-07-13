# Closeout and Archive

Archive freezes the reviewed outcome and its durable handoff.

## Archive

1. Run `harness status` and resolve its archive blockers.
2. Confirm acceptance criteria and steps are complete and finalize coverage
   contains a passing full root plus any linked repair deltas.
3. Update the tracked plan's `Validation Summary`, `Review Summary`, `Archive
   Summary`, and `Outcome Summary` from repository and review evidence, not
   memory.
4. Record durable follow-up details when `Deferred Items` contains real work.
5. Run `harness plan lint <plan-path>` and `harness archive`.

Keep summaries concise: delivered outcome, meaningful validation, review result
and repairs, known limits, and merge/release handoff. Do not paste execution
transcripts.

## Handoff

Commit the archive move, publish the branch and PR, and record publish evidence.
Then follow [publish-ci-sync.md](publish-ci-sync.md) until current CI and sync
facts support `execution/finalize/await_merge`.

Lightweight work still needs the agreed repo-visible breadcrumb, such as a
readable PR body memo.

If feedback or remote changes invalidate the archived candidate, use:

```bash
harness reopen --mode <finalize-fix|new-step>
```

Do not claim archive or merge readiness while review findings, placeholders,
deferred handoff, CI, or sync blockers remain.
