package reviewcoverage

import (
	"strings"
	"testing"
)

const reviewedPlan = `---
template_version: 0.3.0
created_at: "2026-07-13T00:00:00Z"
source_type: direct_request
source_refs: [test]
size: S
---

# Plan

## Goal

Ship the candidate.

### Decisions and Constraints

- Keep the closeout narrow.

## Scope

### In Scope

- Candidate behavior.

### Out of Scope

- Unrelated behavior.

## Acceptance Criteria

- [x] Candidate works.

## Review Focus

- Check candidate coverage.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Ship candidate

- Done: [x]
- Outcome: Candidate shipped.
- Covers: Candidate works.

## Validation Strategy

- Run focused tests.

## Closeout

- Validation: PENDING_UNTIL_ARCHIVE
- Review: PENDING_UNTIL_ARCHIVE
- Delivered: PENDING_UNTIL_ARCHIVE
- Not Delivered: PENDING_UNTIL_ARCHIVE
- Follow-Up Issues: NONE
- PR: PENDING_UNTIL_ARCHIVE
- Ready: PENDING_UNTIL_ARCHIVE
- Merge Handoff: PENDING_UNTIL_ARCHIVE
`

func TestValidateArchiveWorktreeAllowsOnlyCloseoutSectionBodies(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	planPath := writeFile(t, root, "docs/plans/active/2026-07-13-test.md", reviewedPlan)
	writeFile(t, root, "product.go", "package product\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")

	closeout := strings.ReplaceAll(reviewedPlan, "PENDING_UNTIL_ARCHIVE", "Completed closeout evidence.")
	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", closeout)
	if err := ValidateArchiveWorktree(root, planPath, covered); err != nil {
		t.Fatalf("expected closeout-only edits to pass: %v", err)
	}

	writeFile(t, root, "product.go", "package product\n\nconst Changed = true\n")
	if err := ValidateArchiveWorktree(root, planPath, covered); err == nil || !strings.Contains(err.Error(), "product.go") {
		t.Fatalf("expected product edit rejection, got %v", err)
	}
}

func TestValidateArchiveWorktreeRejectsPlanStructureAndSupplements(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	planPath := writeFile(t, root, "docs/plans/active/2026-07-13-test.md", reviewedPlan)
	writeFile(t, root, "docs/plans/active/supplements/2026-07-13-test/review-guidance/risk.md", "reviewed guidance\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")

	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", strings.Replace(reviewedPlan, "Ship the candidate.", "Changed goal.", 1))
	if err := ValidateArchiveWorktree(root, planPath, covered); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("expected non-closeout plan edit rejection, got %v", err)
	}
	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", reviewedPlan)
	writeFile(t, root, "docs/plans/active/supplements/2026-07-13-test/review-guidance/risk.md", "changed guidance\n")
	if err := ValidateArchiveWorktree(root, planPath, covered); err == nil || !strings.Contains(err.Error(), "supplements") {
		t.Fatalf("expected supplement rejection, got %v", err)
	}
}

func TestValidateArchiveWorktreeRejectsChangeAfterFencedCloseoutHeading(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	planWithHeadingExample := strings.Replace(reviewedPlan, "Ship the candidate.", `Ship the candidate.

## Design Notes

The review contract includes this non-closeout example:

~~~markdown
## Closeout

Example content only.
~~~

This text is still part of Design Notes.`, 1)
	planPath := writeFile(t, root, "docs/plans/active/2026-07-13-test.md", planWithHeadingExample)
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")

	changed := strings.Replace(planWithHeadingExample, "This text is still part of Design Notes.", "This unreviewed design text changed.", 1)
	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", changed)
	if err := ValidateArchiveWorktree(root, planPath, covered); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("expected change after fenced closeout heading to be rejected, got %v", err)
	}
}

func TestValidateArchiveWorktreeAllowsCommittedCloseoutOnlyDescendant(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	planPath := writeFile(t, root, "docs/plans/active/2026-07-13-test.md", reviewedPlan)
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")
	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", strings.ReplaceAll(reviewedPlan, "PENDING_UNTIL_ARCHIVE", "Closeout complete."))
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "closeout")
	if err := ValidateArchiveWorktree(root, planPath, covered); err != nil {
		t.Fatalf("expected committed closeout-only descendant to pass: %v", err)
	}
}
