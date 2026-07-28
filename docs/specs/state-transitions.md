# State Transitions

## Purpose

This is the normative transition matrix. If a move is absent, it is not a
supported workflow transition. Exact command payloads live in the CLI contract.

`step-<n>` is the current unfinished tracked step and `step-<m>` is the next
unfinished step for a standard or lightweight root. A step is complete when
its `Done` marker is checked. Coordinated roots use `execution/coordinate`;
their subplans use ordered child steps internally.

## Entering and Executing Work

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `idle` | `plan` | Active plan presence | Exactly one tracked active plan exists. |
| `plan` | `execution/step-<n>/implement` | `harness execute start` | The plan has explicit human approval and an unfinished step. |
| `plan` | `execution/coordinate` | `harness execute start` | The coordinated root has explicit human approval. |
| `execution/step-<n>/implement` | `execution/step-<m>/implement` | Plan edit | Step `<n>` becomes complete and another unfinished step exists. |
| `execution/step-<n>/implement` | `execution/finalize/review` | Plan edit | Every step is complete. Formal review has not yet established clean coverage for the candidate. |
| `execution/coordinate` | `execution/coordinate` | Subplan edit | At least one subplan is incomplete, waiting on a sibling, or structurally blocked. |
| `execution/coordinate` | `execution/finalize/review` | Subplan edit | At least one subplan exists, every subplan step and Result is complete, and the package graph is valid. |

Root or child steps are implementation and validation boundaries. They never
create formal review nodes. Coordinated children may progress concurrently but
share one candidate and controller-owned Git integration. Intermediate
uncertainty uses focused validation or ordinary advisor subagents; the
independent harness review belongs to the complete root candidate.

## Finalize

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `execution/finalize/review` | `execution/finalize/fix` | `harness review submit` | The integrated reviewer reports blocking findings or a conservative failure. |
| `execution/finalize/review` | `execution/finalize/archive` | `harness review submit` or derived status | A clean full root, optionally extended by clean linked deltas, covers current candidate HEAD. |
| `execution/finalize/review` | `execution/finalize/review` | `harness review abort` | The exact active unfinished round is preserved as aborted history and its active pointer is cleared; completed coverage is unchanged. |
| `execution/finalize/fix` | `execution/finalize/review` | `harness review start` | A committed repair starts an inferred linked delta, or `--full` explicitly resets materially invalidated coverage. |
| `execution/finalize/fix` | `execution/step-<m>/implement` | Plan edit after `reopen --mode new-step` | The first new unfinished step is added. |

`review start` is finalize-only. The first round is full. When prior coverage
and unresolved findings exist, the ordinary next round is an inferred linked
delta. Rewritten ancestry over clean coverage automatically establishes another
full root before a round is created. Rewritten ancestry with unresolved
findings fails safely until ancestry is restored or a human explicitly requests
a replacement full root. A human may also request a full root for a material
design, scope, or risk change. An unfinished round may be explicitly aborted without
deleting its history or changing prior coverage. The sole reviewer submission
verifies the captured HEAD and completes
the decision and coverage transaction without a controller aggregate action.

## Archive, Publish, and Land

| From | To | Driver | Requirement |
| --- | --- | --- | --- |
| `execution/finalize/archive` | `execution/finalize/publish` | `harness archive` | Acceptance, ordinary root steps or the complete coordinated package, Closeout, and finalize coverage are complete. |
| `execution/finalize/publish` | `execution/finalize/await_merge` | Evidence | Current publish, CI, and sync evidence supports merge readiness, and the archived branch differs from the reviewed candidate only by the allowed archive move and Closeout update. |
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
- skipping unfinished root/child steps or the mandatory root finalize review
- archiving with unresolved findings, moved or uncovered HEAD, incomplete
  acceptance, incomplete steps or coordinated Results, an invalid subplan
  graph, or Closeout placeholders
- using child-specific approval, execute-start, review, archive, publish, or
  land as if a subplan owned an independent candidate
- treating an advisor as the formal reviewer or controller
- replacing a narrow linked delta with a ceremonial full review without a
  material coverage invalidation
- reaching `await_merge` without current remote evidence
- merging without explicit human approval or returning to `idle` without land
  completion
