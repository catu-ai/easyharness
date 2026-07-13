# Goal-Oriented Workflow

## Purpose

This document defines the normative `goal_oriented` workflow profile for
`easyharness`.

Use `workflow_profile: goal_oriented` when the objective and success scorecard
are explicit, but the path to the result must adapt through hypotheses,
probes, checkpoint rounds, optional challenge, and synthesis.

`goal_oriented` is a profile on top of the existing harness workflow. It does
not create a separate workflow engine. The completed v0.6.0 profile is
designed to use explicit human approval, stable plan steps, formal
finalize review, optional risk-boundary step review, archive, evidence, and the
canonical node state model. In the current authoring preview, agents can draft
and lint active tracked goal-oriented plans, but must not rely on full lifecycle
support until the follow-up implementation slices land.

`goal_oriented` is a recognized preview workflow profile for v0.6.0 authoring.
`harness plan template --goal-oriented` can seed the goal-oriented plan shape,
and `harness plan lint` recognizes active tracked plans that declare
`workflow_profile: goal_oriented` with shallow preview validation. Full
execution support is still being completed: follow-up implementation slices own
status next actions, structural lint coverage, archive/reopen profile
preservation, docs/examples, and UI rendering.

## When To Use It

Use `goal_oriented` for work where the plan can name the desired outcome but
cannot honestly predeclare the exact execution path. Typical examples include:

- performance or reliability investigation with a measurable target
- design exploration where several plausible contracts need comparison
- root-cause analysis when the bug or failure mode is not yet isolated
- benchmark, evaluation, or selection work where evidence decides the path
- research-like product work that must converge on a durable decision or
  artifact

Keep the ordinary standard workflow when the task is delivery-oriented and the
path is already clear enough to plan as implementation steps. Examples include
adding a known CLI flag, fixing an already-localized bug, updating narrow docs,
or applying a known template change.

Do not use `goal_oriented` as a heavier name for ordinary standard work. If
the work does not need hypotheses, probes, checkpoint reports, or final
synthesis, the ordinary standard workflow is clearer.

## Workflow Profile Semantics

`workflow_profile: goal_oriented` means the active plan is a standard tracked
plan with an adaptive authoring contract. It is not a lightweight shortcut.

The completed profile preserves these core harness boundaries once full
execution support lands:

- human approval still approves the tracked plan package
- plan steps still represent stable approved phase boundaries
- `harness execute start` still begins execution
- finalize full review is the default formal review gate; a step review is
  binding only when the controller intentionally starts one at a concrete
  intermediate risk boundary
- archive still preserves the durable outcome summary and review history
- command-owned runtime state still lives under the local runtime root resolved
  by `harness repo config get paths.local_runtime`
- `current_node` still comes from the canonical state model, not from plan
  prose or checkpoint markdown

The current authoring preview seeds these concepts so future work can be
planned credibly, while archive/reopen behavior, status next actions, and full
structural lint remain follow-up work.
Goal-oriented execution will add a disciplined layer inside approved steps:
the controller runs bounded checkpoint rounds, records concise tracked
checkpoint reports when the work reaches meaningful decision points,
optionally requests challenge when hypotheses or evidence need stress testing,
and writes a final synthesis before closeout.

## Plan Requirements

A goal-oriented plan must define these concepts clearly enough that a future
agent can resume without hidden chat context:

- `objective`
  - the concrete result to achieve or question to answer
  - not merely "explore" or "investigate"
- `success scorecard`
  - the criteria, metrics, observations, or decision rules used to decide
    whether the work improved, failed, stopped, or completed
- `hypotheses or candidate directions`
  - the current explanations, approaches, or options that the checkpoint loop
    may test or revise
- `probe/checkpoint loop`
  - how the controller should move from hypothesis to bounded probe, observed
    signal, scorecard comparison, and next decision
- `checkpoint cadence or budget`
  - the expected checkpoint rhythm for adaptive steps, such as "write 2-4
    checkpoint reports for the main exploration step"
  - a cadence is guidance for keeping the work bounded, not a global hard
    minimum or maximum
  - the preview template gives this cadence a stable authoring anchor, and
    follow-up lint/status work may tighten the parseable contract as needed
- `challenge triggers`
  - when advisory challenge should be considered or is required by the plan
- `evidence requirements`
  - what kind of durable support is needed for the plan's conclusions
- `stopping conditions`
  - when the controller should stop probing and synthesize, close the step,
    defer remaining uncertainty, or ask the human about scope
- `final synthesis`
  - the closeout explanation of accepted conclusions, rejected hypotheses,
    residual uncertainty, evidence, and follow-up work

The plan may express these concepts in dedicated sections, step details, or
the goal-oriented template shape. The preview template uses stable authoring
anchors, but the full structural lint contract remains follow-up work.

Future lint and status support must not rely on unstructured prose guessing.
Those implementation slices may consume stable parseable anchors for the
fields status and lint need, especially the success scorecard, checkpoint
cadence, tracked checkpoint report index, and final synthesis. Prefer
plan-body structure for goal-oriented working concepts; reserve frontmatter for
durable command-level metadata such as profile selection unless a field truly
affects command resolution.

## Steps, Checkpoint Rounds, And Model Turns

A harness step is a durable approved phase boundary. A model turn is a runtime
or accounting unit. A checkpoint round is the goal-oriented layer between them.

Do not make one harness step per model turn the default shape. A single
adaptive step may contain several model continuations, probes, observations,
and pivots. Durable learning from that work belongs in tracked checkpoint
reports or final synthesis, not in a growing list of tiny plan steps.

Use stable steps for boundaries that need approval, validation, and archive
visibility. Use checkpoint rounds for meaningful intermediate learning inside
those steps.

Before an adaptive step closes, the plan must contain at least one tracked
checkpoint report or equivalent final synthesis explaining the adaptive work
that happened in the step. A step that performs no adaptive work should not use
goal-oriented ceremony just to satisfy the profile.

## Checkpoint Drafts

A checkpoint draft is execution working memory. It may live under the local
runtime root and may be messy enough to help the controller resume after
context compaction.

Checkpoint drafts may contain transient details such as:

- raw observations not yet curated
- failed probes that may or may not matter
- command summaries
- temporary hypothesis notes
- reminders about what to test next

Checkpoint drafts are local and disposable by default. They do not enter git,
archive, or review unless their content is promoted into a tracked checkpoint
report, an approved supplement, a formal evidence artifact, or another
approved deliverable.

## Tracked Checkpoint Reports

A tracked checkpoint report is a concise, git-tracked structured narrative
digest of a meaningful checkpoint round. It records the hypotheses or
candidate directions considered, probe or experiment, observed result,
scorecard movement, decision or next mutation, residual uncertainty, and
evidence pointers needed for future resume, review, and archive.

A tracked checkpoint report is not:

- a raw log
- a transcript
- a list of every command
- a record of every small attempt
- a bullet-only form
- a formal review gate
- the canonical evidence report unless the plan explicitly designates it as
  such

Tracked checkpoint reports should normally live in the plan body under a
`Checkpoint Reports` section. For step-local reports, use a dedicated
`#### Checkpoint Reports` subsection inside the adaptive step. Use one
subsection per report with a stable checkpoint ID, such as `CP1`, `CP2`, or
`S2-CP1`, so a single checkpoint can compare several hypotheses without
turning the plan into a nested bullet log or a series of tiny pseudo-steps.

Tracked checkpoint reports should be self-contained enough that a future agent
can understand the decision trail from the plan body. Do not push essential
checkpoint meaning into a separate file merely to keep the plan short.

Supplements are allowed only for curated durable material that the approved
plan deliberately keeps with the plan package. Do not use tracked supplements
as the default home for bulky raw experiment data, JSON/CSV dumps, transcripts,
or command logs. Large or messy probe artifacts should stay local,
regenerable, external, or be summarized into a smaller approved deliverable
unless the user objective explicitly requires committing them.

When a small durable support artifact is approved for the tracked plan package,
keep the actual checkpoint report in the plan body and index the support
artifact from that report. Supplements share the same approval boundary as the
plan package; they are not free-form scratch space.

Checkpoint report structure is a shallow parseable contract, not a prose
straitjacket. Required label names are stable enough for future lint/status
support to find them, but the content under a label may be prose, bullets,
tables, or short lists according to what the checkpoint needs to explain.
Future tooling may inspect headings, IDs, and labels; it must not require
bullet-only content or judge whether a hypothesis, probe, evidence argument, or
next mutation is good.

A tracked checkpoint report must include these labels. The only interchangeable
required label names are the pairs shown explicitly below:

- `Trigger`
- `Hypotheses` or `Candidate Directions`
- `Probe` or `Experiment`
- `Observed Result`
- `Scorecard Movement`
- `Decision / Next Mutation`
- `Residuals`
- `Evidence`

A tracked checkpoint report may include these labels when useful:

- `Challenge`
- `Rejected Alternatives`
- `Human Decision Needed`
- `Follow-Up / Deferred`
- `Supplement`
- `Validation / Reproduction`

The report should explain decision movement, not activity. Many small attempts
that do not independently change the decision space should be summarized inside
one report, for example under `Probe`, `Observed Result`, or an optional
attempts summary. A meaningful pivot, plateau, challenge request, scorecard
movement, pre-synthesis boundary, or human-scope decision is usually a better
reason to write a new checkpoint report than the mere fact that another model
turn or command happened.

## Challenge

Challenge is optional advisory intervention during goal-oriented work. It is
used to stress-test hypotheses, identify missing probes, notice evidence gaps,
or broaden candidate directions before the controller commits to a synthesis
or pivot.

Use `hypothesis-challenge` as the checkpoint advisory action for this posture.
It is not a review dimension, does not start a review round, does not create a
review aggregate, and does not change `current_node`. The controller may run
it locally or ask a bounded challenger to look for alternative hypotheses,
sharper probes, weak-evidence claims, premature convergence, or useful next
mutations.

Consider `hypothesis-challenge` at checkpoint boundaries when:

- several plausible hypotheses remain and the current probes cannot distinguish
  them
- evidence is too weak for the conclusion the controller is about to write
- the scorecard has plateaued and the controller is considering a major pivot
- the synthesis is high-impact enough that alternative explanations should be
  tested first
- the human or approved plan declares a required challenge boundary

Do not require every goal-oriented plan to run at least one challenge round.
When evidence is decisive and the plan did not require challenge, the
controller may synthesize and proceed to formal review without challenge.

Record challenge output in the checkpoint report's `Challenge` field when it
sharpens the current decision. If challenge materially changes the direction,
write a new tracked checkpoint report so the pivot, evidence, and residuals are
visible to later review and archive.

## Evidence And Reports

Goal-oriented work must support final conclusions with evidence, but the
profile does not require every plan to create a standalone evidence report.

The approved plan decides whether an evidence report, decision report, spec,
code change, configuration change, documentation change, benchmark artifact,
or other file is the deliverable. Use the user's objective and plan scope to
choose the durable artifact.

When the deliverable is an evidence or decision report, checkpoint reports
should point to it or summarize key pivots without duplicating the report.
When the deliverable is code, docs, configuration, or another artifact,
checkpoint reports and ordinary validation evidence may be sufficient for
review and archive.

Archive should preserve conclusions, evidence, rejected hypotheses, residuals,
and follow-up items that still matter after synthesis. It should not preserve
duplicate process notes when those notes have been absorbed into a deliverable
or final synthesis.

## Status Guidance

Future `harness status` support for goal-oriented plans may provide advisory
structure checks and general next-action guidance. It must not become a hidden
full lint pass, and it must not infer or mutate `current_node` from checkpoint
markdown.

At checkpoint-round boundaries, status guidance may present a short list of
general next actions, such as:

- continue with another bounded checkpoint round
- synthesize if the scorecard is decisive
- request challenge if uncertainty or competing hypotheses remain
- close the step if synthesis and evidence are ready

The controller remains responsible for deciding which action fits the current
evidence. The CLI should not attempt to judge hypothesis quality from prose.

Status may later warn about shallow structural gaps, such as no checkpoint
cadence, zero tracked checkpoint reports in an adaptive step, or missing
report anchors. Those warnings are navigation guidance. They are not a
substitute for explicit plan lint or formal review.

## Lint Boundary

`harness plan lint` remains the explicit validation surface for tracked plans.

Current goal-oriented lint support is intentionally shallow preview
recognition for active tracked plans. Full coverage belongs in a follow-up
implementation slice. That coverage may check structural requirements such as
objective, success scorecard, checkpoint cadence, tracked checkpoint report
anchors, and final synthesis presence. It should avoid judging whether a
hypothesis is good or whether evidence is persuasive; those are review and
challenge concerns.

Status may point the controller toward lint when tracked checkpoint reports or
final synthesis change, but status should not silently run or replace full
lint.

## Review And Archive

Finalize full review remains the default formal gate. Step-closeout review is
not mandatory, including for large goal-oriented plans; it is available when
an intermediate conclusion, contract, irreversible probe, or other concrete
risk boundary must be challenged before later work depends on it. Once such a
review is started, its blocking findings must be resolved. Challenge does not
replace a formal review that was intentionally started, and checkpoint reports
do not create new review states.

Review for goal-oriented work should focus on whether the final synthesis is
supported by the scorecard, tracked checkpoint reports, durable evidence,
rejected hypotheses, residual uncertainty, and follow-up handling.

Use the built-in `evidence-validity` review dimension when an intentionally
started step review or the finalize review needs formal scrutiny of conclusions, syntheses, decision
artifacts, evidence claims, rejected alternatives, residuals, or follow-up
handling. It is especially useful for goal-oriented finalize review when the
candidate depends on adaptive conclusions rather than only implementation
correctness or documentation consistency.

Controllers still choose guidance deliberately. A finalize integrated reviewer
may receive `evidence-validity` together with correctness, tests, risk, and
documentation guidance; a dimension does not imply a separate reviewer. Do not
assign `evidence-validity`, or any other built-in dimension, merely because it
exists in the catalog.

Known experimental risks and evidence invariants may be recorded in the plan
package, including additive plan-scoped review guidance. The controller chooses
actual specialist assignments immediately before finalize from the completed
candidate. Adaptive plan size, checkpoint count, and uncertainty alone do not
trigger a specialist; a specialist requires a concrete high-risk surface and a
risk brief naming the invariants and failure modes to challenge.

At archive, the tracked plan package should preserve the durable decision
trail:

- final synthesis
- accepted conclusions
- rejected hypotheses or candidate directions
- residual uncertainty
- curated evidence or deliverable pointers
- follow-up issues for deferred work

Local checkpoint drafts remain disposable unless promoted into the tracked
plan package or another approved durable artifact before archive.

## Follow-Up Boundaries

This contract now owns the #271 checkpoint report convention. It intentionally
leaves the remaining goal-oriented implementation details to the v0.6.0
follow-up issues:

- #270 adds the goal-oriented plan template and authoring-preview workflow
  guidance
- #272 added evidence-validity formal review guidance and
  hypothesis-challenge checkpoint advisory guidance
- #273 teaches status and next-action behavior for goal-oriented plans,
  including any future hints for when to consider `hypothesis-challenge`
- #274 adds full structural lint coverage for goal-oriented plans beyond the
  shallow active-plan preview recognition
- #275 adds user-facing docs, help text, and examples

The implementation slices must also define how archive and reopen preserve the
goal-oriented profile identity without relying on hidden chat memory or
contradicting tracked archived-plan frontmatter rules.
