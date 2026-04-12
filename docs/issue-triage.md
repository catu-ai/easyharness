# GitHub Issue Triage

This repository keeps GitHub issue triage intentionally lightweight.

The operational contract for backlog triage lives in the repo-local skill
package at [`.agents/skills/issue-triage/`](../.agents/skills/issue-triage/).
Use that skill when reviewing new issues, revisiting deferred backlog items, or
doing broader backlog sweeps.

This page is only a thin discovery note for maintainers:

- labels and milestones live as GitHub repository metadata, not as a
  `.github` file that GitHub reads automatically
- the repo-local `issue-triage` skill is intentionally repo-owned rather than
  part of the distributed `harness-*` bootstrap pack
- if the triage rules change, update the skill package first and keep this page
  as a short pointer rather than a second full policy surface
