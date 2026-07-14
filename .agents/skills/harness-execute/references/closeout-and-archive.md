# Closeout and Archive

Archive freezes the reviewed outcome and its durable handoff.

## Archive

1. Run `harness status` and resolve its archive blockers.
2. Confirm acceptance criteria and steps are complete and finalize coverage
   contains a passing full root plus any linked repair deltas.
3. Replace the tracked plan's `Closeout` placeholders from repository and
   review evidence, not memory.
4. Record a concrete issue URL or `#number` reference when `Deferred Items`
   contains real work. If no future commitment exists, keep that work in Out of
   Scope and use `- None.` under Deferred Items instead of inventing an issue.
5. Run `harness plan lint <plan-path>` and `harness archive`.

Keep summaries concise: delivered outcome, meaningful validation, review result
and repairs, known limits, and concrete follow-up issues. The tracked Closeout
does not duplicate PR, CI, sync, merge, or land facts. Its labeled values may
wrap onto immediately following ordinary prose lines indented by at least two
spaces. Do not paste execution transcripts.

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
