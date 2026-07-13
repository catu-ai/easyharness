# State Transitions

## Purpose

This is the normative transition matrix. If a move is absent, it is not a
supported workflow transition. Exact command payloads live in the CLI contract.

`step-<n>` is the current unfinished tracked step and `step-<m>` is the next
unfinished step. A step is complete when its `Done` marker is checked.

## Entering and Executing Work

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `idle` | `plan` | Active plan presence | Exactly one tracked active plan exists. |
| `plan` | `execution/step-<n>/implement` | `harness execute start` | The plan has explicit human approval and an unfinished step. |
| `execution/step-<n>/implement` | `execution/step-<m>/implement` | Plan edit | Step `<n>` becomes complete and another unfinished step exists. |
| `execution/step-<n>/implement` | `execution/finalize/review` | Plan edit | Every step is complete. Formal review has not yet established clean coverage for the candidate. |

Steps are human-visible implementation and validation boundaries. They never
create formal review nodes. Intermediate uncertainty uses focused validation or
ordinary advisor subagents; the independent harness review belongs to the
complete finalize candidate.

## Finalize

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `execution/finalize/review` | `execution/finalize/fix` | `harness review submit` | The integrated reviewer reports blocking findings or a conservative failure. |
| `execution/finalize/review` | `execution/finalize/archive` | `harness review submit` or derived status | A clean full root, optionally extended by clean linked deltas, covers current candidate HEAD. |
| `execution/finalize/fix` | `execution/finalize/review` | `harness review start` | A committed repair starts an inferred linked delta, or `--full` explicitly resets materially invalidated coverage. |
| `execution/finalize/fix` | `execution/step-<m>/implement` | Plan edit after `reopen --mode new-step` | The first new unfinished step is added. |

`review start` is finalize-only. The first round is full. When prior coverage
and unresolved findings exist, the ordinary next round is an inferred linked
delta; another full root is reserved for a material design, scope, or risk
change. The sole reviewer submission verifies the captured HEAD and completes
the decision and coverage transaction without a controller aggregate action.

## Archive, Publish, and Land

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `execution/finalize/archive` | `execution/finalize/publish` | `harness archive` | Acceptance, steps, Closeout, and finalize coverage are complete. |
| `execution/finalize/publish` | `execution/finalize/await_merge` | Evidence | Current publish, CI, and sync evidence supports merge readiness. |
| `execution/finalize/publish` or `execution/finalize/await_merge` | `execution/finalize/fix` | `harness reopen` with `finalize-fix` or `new-step` | Feedback or remote change invalidates the archived candidate. |
| `execution/finalize/await_merge` | `land` | `harness land` | Human merge approval exists and the PR has merged. |
| `land` | `idle` | `harness land complete` | Required post-merge bookkeeping and release verification are complete. |

## State-Preserving Work

Implementation, validation, plan refinement, reviewer investigation, repair,
Closeout preparation, evidence refresh, and land bookkeeping preserve their
current node until the corresponding durable condition above changes. Review
start and advisor progress do not create extra public nodes.

## Invalid Shortcuts

- execution without explicit plan approval
- skipping unfinished steps or the mandatory finalize review
- archiving with unresolved findings, moved or uncovered HEAD, incomplete
  acceptance, incomplete steps, or Closeout placeholders
- treating an advisor as the formal reviewer or controller
- replacing a narrow linked delta with a ceremonial full review without a
  material coverage invalidation
- reaching `await_merge` without current remote evidence
- merging without explicit human approval or returning to `idle` without land
  completion
