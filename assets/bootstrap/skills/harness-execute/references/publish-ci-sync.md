# Publish, CI, and Sync

Publish turns the archived local candidate into a current, externally verified
merge handoff.

## Flow

1. Commit and push the archived candidate.
2. Open or update the PR and write a readable merge memo.
3. Record the handoff with `harness evidence submit --kind publish`.
4. When a supported PR URL is recorded, run `harness evidence refresh` to
   capture CI and sync facts. Use manual `publish|ci|sync` evidence only when
   provider refresh is unavailable or unsupported.
5. Follow `harness status` next actions until recorded evidence is current and
   the candidate reaches `execution/finalize/await_merge`.

Live remote observation is read-only context; recorded evidence drives durable
workflow progression. Do not infer a PR from the branch, record running checks
as successful, or claim merge readiness while checks fail, sync is stale or
conflicted, provider reads are degraded, or live facts have drifted.

If remote changes require candidate edits, validate them and extend review with
a linked delta unless design, scope, or risk changed enough to require a new
full root. Use `harness reopen` when an archived candidate must change.

## PR Body

Write for the human making the merge decision, not as an execution log:

- `What Changed`: the outcome now delivered
- `Confidence`: a few high-signal validation and review facts
- `Handoff`: only remaining merge, release, or deferred notes; omit when empty

Do not paste file lists, command transcripts, or the tracked plan. Lightweight
work may use this memo as its required repo-visible breadcrumb.

Direct provider inspection is a diagnostic fallback when status or evidence
refresh is unclear. External PR actions, reruns, conflict resolution, and merge
remain outside harness core commands.
