---
name: review-state-and-coverage
description: Challenge Review v2 candidate binding, repair-chain continuity, and archive exemptions for this plan.
---

Review the candidate as an adversarial review-state and coverage specialist.

Risk surfaces:

- immutable candidate-head capture at review start and aggregate
- full-to-repair-delta parent, anchor, revision, and finding-resolution links
- archive readiness derived from review history rather than the latest pointer
- the narrow post-review plan closeout exemption
- reopen behavior across narrow delta extension and broad full reset

Invariants to challenge:

- review cannot start or aggregate against a dirty, non-Git, unborn, or moved
  candidate
- a repair delta starts exactly from the prior covered head and names the
  blocking findings it resolves
- missing, conflicting, or unknown resolutions remain blocking
- minor findings stay visible without becoming archive debt
- unreviewed product, source, test, specification, supplement, or unrelated
  documentation changes block archive
- only the approved top-level closeout-summary bodies in the current plan may
  change after review without extending coverage
- routine completed steps create no review debt, while an intentionally
  started unresolved step review remains binding

Follow the common harness reviewer evidence, severity, submission, and no-edit
contract. This specialist assignment supplements but does not replace the
integrated whole-candidate reviewer.
