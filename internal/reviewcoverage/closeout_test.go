package reviewcoverage

import (
	"strings"
	"testing"
)

const reviewedPlan = `---
template_version: 0.2.0
created_at: "2026-07-13T00:00:00Z"
source_type: direct_request
source_refs: [test]
size: S
---

# Plan

## Goal

Ship the candidate.

## Acceptance Criteria

- [x] Candidate works.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
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
