package reviewcoverage

import (
	"os"
	"path/filepath"
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

func TestValidateArchiveWorktreeDoesNotMaskFrontmatterCloseoutComment(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	reviewed := strings.Replace(reviewedPlan, "size: S", "## Closeout\nsize: S", 1)
	planPath := writeFile(t, root, "docs/plans/active/2026-07-13-test.md", reviewed)
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")

	changed := strings.Replace(reviewed, "size: S", "size: M", 1)
	changed = strings.Replace(changed, "# Plan", "# Unreviewed Plan", 1)
	writeFile(t, root, "docs/plans/active/2026-07-13-test.md", changed)
	if err := ValidateArchiveWorktree(root, planPath, covered); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("expected frontmatter/title change after YAML comment to be rejected, got %v", err)
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

func TestValidateArchivedCandidateAllowsMechanicalPlanAndSupplementMove(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	activePlan := "docs/plans/active/2026-07-13-test.md"
	archivedPlan := "docs/plans/archived/2026-07-13-test.md"
	activeSupplement := "docs/plans/active/supplements/2026-07-13-test/notes.md"
	archivedSupplement := "docs/plans/archived/supplements/2026-07-13-test/notes.md"
	writeFile(t, root, activePlan, reviewedPlan)
	writeFile(t, root, activeSupplement, "reviewed supplement\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")

	commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
	if err != nil {
		t.Fatalf("render command-owned archive baseline: %v", err)
	}
	archived := strings.ReplaceAll(string(commandRendered), "PENDING_UNTIL_ARCHIVE", "Recorded at archive.")
	writeFile(t, root, archivedPlan, archived)
	writeFile(t, root, archivedSupplement, "reviewed supplement\n")
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
		t.Fatalf("remove active plan: %v", err)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activeSupplement))); err != nil {
		t.Fatalf("remove active supplement: %v", err)
	}

	chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
	if err := ValidateArchivedCandidate(root, archivedPlan, chain); err != nil {
		t.Fatalf("expected mechanical archive move to preserve review coverage: %v", err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "archive plan")
	if err := ValidateArchivedCandidate(root, archivedPlan, chain); err != nil {
		t.Fatalf("expected committed archive move to preserve review coverage: %v", err)
	}
}

func TestValidateArchivedCandidateAgainstBaseAllowsUnchangedCandidateDeltaAfterBaseSync(t *testing.T) {
	for _, syncMode := range []string{"merge", "rebase"} {
		t.Run(syncMode, func(t *testing.T) {
			root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
			git(t, root, "checkout", "candidate")
			if syncMode == "merge" {
				git(t, root, "merge", "--no-ff", "-m", "sync upstream", "upstream")
			} else {
				git(t, root, "rebase", "upstream")
			}

			if err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream); err != nil {
				t.Fatalf("expected unchanged candidate delta after %s sync to pass: %v", syncMode, err)
			}
		})
	}
}

func TestValidateArchivedCandidateAgainstBaseRejectsUpstreamCandidatePathOverlap(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, true)
	git(t, root, "checkout", "candidate")
	git(t, root, "merge", "--no-ff", "-m", "sync overlapping upstream", "upstream")

	err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream)
	if err == nil || !strings.Contains(err.Error(), "overlaps reviewed candidate path product.go") {
		t.Fatalf("expected same-path upstream overlap rejection, got %v", err)
	}
}

func TestValidateArchivedCandidateAgainstBaseRejectsChangedCandidateDelta(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	git(t, root, "merge", "--no-ff", "-m", "sync upstream", "upstream")
	writeFile(t, root, "product.go", "package product\n\nconst Candidate = 2\n")
	git(t, root, "add", "product.go")
	git(t, root, "commit", "-m", "change reviewed candidate")

	err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream)
	if err == nil || !strings.Contains(err.Error(), "candidate-owned delta changed") || !strings.Contains(err.Error(), "product.go") {
		t.Fatalf("expected changed candidate delta rejection, got %v", err)
	}
}

func TestValidateArchivedCandidateAgainstBaseRejectsCandidateModeChange(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	git(t, root, "merge", "--no-ff", "-m", "sync upstream", "upstream")
	if err := os.Chmod(filepath.Join(root, "product.go"), 0o755); err != nil {
		t.Fatalf("change candidate file mode: %v", err)
	}
	git(t, root, "add", "product.go")
	git(t, root, "commit", "-m", "change reviewed candidate mode")

	err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream)
	if err == nil || !strings.Contains(err.Error(), "candidate-owned delta changed") || !strings.Contains(err.Error(), "product.go") {
		t.Fatalf("expected candidate mode change rejection, got %v", err)
	}
}

func TestValidateArchivedCandidateAgainstBaseRejectsDirtyWorktree(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	git(t, root, "merge", "--no-ff", "-m", "sync upstream", "upstream")
	writeFile(t, root, "untracked.txt", "not reviewed\n")

	err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream)
	if err == nil || !strings.Contains(err.Error(), "clean candidate worktree") || !strings.Contains(err.Error(), "untracked.txt") {
		t.Fatalf("expected dirty worktree rejection, got %v", err)
	}
}

func TestValidateArchivedCandidateAgainstBaseRequiresCurrentBaseInHead(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")

	err := ValidateArchivedCandidateAgainstBase(root, archivedPlan, chain, upstream)
	if err == nil || !strings.Contains(err.Error(), "does not contain base") {
		t.Fatalf("expected unsynchronized base rejection, got %v", err)
	}
}

func TestValidatePublishedCandidateDoesNotDependOnPostSquashCheckout(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	published := git(t, root, "rev-parse", "HEAD")

	git(t, root, "checkout", "-b", "landed", upstream)
	git(t, root, "merge", "--squash", "candidate")
	git(t, root, "commit", "-m", "squash candidate")
	if ancestor, err := IsAncestor(root, chain.CoveredHeadSHA, "HEAD"); err != nil {
		t.Fatalf("inspect squash ancestry: %v", err)
	} else if ancestor {
		t.Fatal("test setup expected landed squash commit not to descend from reviewed head")
	}

	if err := ValidatePublishedCandidate(root, archivedPlan, chain, published); err != nil {
		t.Fatalf("expected recorded published candidate to validate from landed checkout: %v", err)
	}
}

func TestValidatePublishedCandidateAgainstBaseAllowsEquivalentRebase(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	git(t, root, "rebase", "upstream")
	published := git(t, root, "rev-parse", "HEAD")
	if ancestor, err := IsAncestor(root, chain.CoveredHeadSHA, published); err != nil {
		t.Fatalf("inspect rebased ancestry: %v", err)
	} else if ancestor {
		t.Fatal("test setup expected rebase to rewrite reviewed ancestry")
	}

	git(t, root, "checkout", "upstream")
	if err := ValidatePublishedCandidateAgainstBase(root, archivedPlan, chain, published, upstream); err != nil {
		t.Fatalf("expected equivalent rebased published candidate to preserve coverage: %v", err)
	}
}

func TestValidatePublishedCandidateAgainstBaseRejectsRebasedCandidateDrift(t *testing.T) {
	root, archivedPlan, chain, upstream := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	git(t, root, "rebase", "upstream")
	writeFile(t, root, "product.go", "package product\n\nconst Candidate = 2\n\nconst Upstream = 0\n")
	git(t, root, "add", "product.go")
	git(t, root, "commit", "-m", "candidate drift")
	published := git(t, root, "rev-parse", "HEAD")

	err := ValidatePublishedCandidateAgainstBase(root, archivedPlan, chain, published, upstream)
	if err == nil || !strings.Contains(err.Error(), "candidate-owned delta changed") {
		t.Fatalf("expected rebased published candidate drift rejection, got %v", err)
	}
}

func TestValidatePublishedCandidateRejectsUnreviewedPublishedChange(t *testing.T) {
	root, archivedPlan, chain, _ := baseAwareArchivedCandidate(t, false)
	git(t, root, "checkout", "candidate")
	writeFile(t, root, "product.go", "package product\n\nconst Candidate = 2\n")
	git(t, root, "add", "product.go")
	git(t, root, "commit", "-m", "unreviewed published change")
	published := git(t, root, "rev-parse", "HEAD")

	err := ValidatePublishedCandidate(root, archivedPlan, chain, published)
	if err == nil || !strings.Contains(err.Error(), "unreviewed product change") || !strings.Contains(err.Error(), "product.go") {
		t.Fatalf("expected unreviewed published change rejection, got %v", err)
	}
}

func baseAwareArchivedCandidate(t *testing.T, overlap bool) (string, string, *Chain, string) {
	t.Helper()
	root := t.TempDir()
	initGit(t, root)
	activePlan := "docs/plans/active/2026-07-13-test.md"
	archivedPlan := "docs/plans/archived/2026-07-13-test.md"
	writeFile(t, root, activePlan, reviewedPlan)
	writeFile(t, root, "product.go", "package product\n\nconst Candidate = 0\n\nconst Upstream = 0\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")
	base := git(t, root, "rev-parse", "HEAD")

	git(t, root, "checkout", "-b", "candidate")
	writeFile(t, root, "product.go", "package product\n\nconst Candidate = 1\n\nconst Upstream = 0\n")
	git(t, root, "add", "product.go")
	git(t, root, "commit", "-m", "reviewed candidate")
	reviewed := git(t, root, "rev-parse", "HEAD")
	commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
	if err != nil {
		t.Fatalf("render command-owned archive baseline: %v", err)
	}
	writeFile(t, root, archivedPlan, string(commandRendered))
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
		t.Fatalf("remove active plan: %v", err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "archive plan")

	git(t, root, "checkout", "-b", "upstream", base)
	if overlap {
		writeFile(t, root, "product.go", "package product\n\nconst Candidate = 0\n\nconst Upstream = 1\n")
		git(t, root, "add", "product.go")
	} else {
		writeFile(t, root, "upstream.txt", "unrelated upstream work\n")
		git(t, root, "add", "upstream.txt")
	}
	git(t, root, "commit", "-m", "upstream advance")
	upstream := git(t, root, "rev-parse", "HEAD")

	return root, archivedPlan, &Chain{CoveredHeadSHA: reviewed, ReviewedPlanPath: activePlan}, upstream
}

func TestValidateArchivedCandidateRejectsPostArchiveProductChange(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	activePlan := "docs/plans/active/2026-07-13-test.md"
	archivedPlan := "docs/plans/archived/2026-07-13-test.md"
	writeFile(t, root, activePlan, reviewedPlan)
	writeFile(t, root, "product.go", "package product\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")
	commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
	if err != nil {
		t.Fatalf("render command-owned archive baseline: %v", err)
	}
	writeFile(t, root, archivedPlan, string(commandRendered))
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
		t.Fatalf("remove active plan: %v", err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "archive plan")

	writeFile(t, root, "product.go", "package product\n\nconst Unreviewed = true\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "unreviewed product change")
	chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
	if err := ValidateArchivedCandidate(root, archivedPlan, chain); err == nil || !strings.Contains(err.Error(), "product.go") {
		t.Fatalf("expected post-archive product change rejection, got %v", err)
	}
}

func TestValidateArchivedCandidateRejectsFrontmatterChangesOutsideCloseout(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	activePlan := "docs/plans/active/2026-07-13-test.md"
	archivedPlan := "docs/plans/archived/2026-07-13-test.md"
	writeFile(t, root, activePlan, reviewedPlan)
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
		t.Fatalf("remove active plan: %v", err)
	}

	chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
	commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
	if err != nil {
		t.Fatalf("render command-owned archive baseline: %v", err)
	}
	canonicalPlan := string(commandRendered)
	for name, mutation := range map[string]string{
		"unknown key":       strings.Replace(canonicalPlan, "size: S", "size: S\nunreviewed_key: true", 1),
		"comment":           strings.Replace(canonicalPlan, "size: S", "size: S\n# unreviewed comment", 1),
		"opening delimiter": strings.Replace(canonicalPlan, "---\n", " --- \n", 1),
		"closing delimiter": strings.Replace(canonicalPlan, "\n---\n\n# Plan", "\n --- \n\n# Plan", 1),
	} {
		t.Run(name, func(t *testing.T) {
			writeFile(t, root, archivedPlan, mutation)
			if err := ValidateArchivedCandidate(root, archivedPlan, chain); err == nil || !strings.Contains(err.Error(), "outside the allowed Closeout body") {
				t.Fatalf("expected raw frontmatter change rejection, got %v", err)
			}
		})
	}
}

func TestValidateArchivedCandidateDoesNotMaskFrontmatterCloseoutComment(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	activePlan := "docs/plans/active/2026-07-13-test.md"
	archivedPlan := "docs/plans/archived/2026-07-13-test.md"
	reviewed := strings.Replace(reviewedPlan, "size: S", "## Closeout\nsize: S", 1)
	writeFile(t, root, activePlan, reviewed)
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "reviewed")
	covered := git(t, root, "rev-parse", "HEAD")
	changed := strings.Replace(reviewed, "size: S", "size: M", 1)
	changed = strings.Replace(changed, "# Plan", "# Unreviewed Plan", 1)
	writeFile(t, root, archivedPlan, changed)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
		t.Fatalf("remove active plan: %v", err)
	}
	chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
	if err := ValidateArchivedCandidate(root, archivedPlan, chain); err == nil || !strings.Contains(err.Error(), "outside the allowed Closeout body") {
		t.Fatalf("expected archived frontmatter/title change after YAML comment to be rejected, got %v", err)
	}
}

func TestValidateArchivedCandidateModelsCommandBodyNormalization(t *testing.T) {
	for name, separator := range map[string]string{
		"no leading newline":       "---\n# Plan",
		"several leading newlines": "---\n\n\n\n# Plan",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			initGit(t, root)
			activePlan := "docs/plans/active/2026-07-13-test.md"
			archivedPlan := "docs/plans/archived/2026-07-13-test.md"
			reviewed := strings.Replace(reviewedPlan, "---\n\n# Plan", separator, 1)
			writeFile(t, root, activePlan, reviewed)
			git(t, root, "add", ".")
			git(t, root, "commit", "-m", "reviewed")
			covered := git(t, root, "rev-parse", "HEAD")
			commandRendered, err := commandRenderedArchivePlan([]byte(reviewed))
			if err != nil {
				t.Fatalf("render command-owned archive baseline: %v", err)
			}
			writeFile(t, root, archivedPlan, string(commandRendered))
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
				t.Fatalf("remove active plan: %v", err)
			}
			chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
			if err := ValidateArchivedCandidate(root, archivedPlan, chain); err != nil {
				t.Fatalf("expected command-normalized archive to pass: %v", err)
			}
		})
	}
}

func TestValidateArchivedCandidateRejectsPlanModeOrTypeChange(t *testing.T) {
	for _, mutation := range []string{"executable", "symlink"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			initGit(t, root)
			activePlan := "docs/plans/active/2026-07-13-test.md"
			archivedPlan := "docs/plans/archived/2026-07-13-test.md"
			writeFile(t, root, activePlan, reviewedPlan)
			git(t, root, "add", ".")
			git(t, root, "commit", "-m", "reviewed")
			covered := git(t, root, "rev-parse", "HEAD")
			commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
			if err != nil {
				t.Fatalf("render command-owned archive baseline: %v", err)
			}
			archivedPath := writeFile(t, root, archivedPlan, string(commandRendered))
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
				t.Fatalf("remove active plan: %v", err)
			}
			if mutation == "executable" {
				if err := os.Chmod(archivedPath, 0o755); err != nil {
					t.Fatalf("chmod archived plan: %v", err)
				}
			} else {
				target := archivedPath + ".target"
				if err := os.Rename(archivedPath, target); err != nil {
					t.Fatalf("move archived plan target: %v", err)
				}
				if err := os.Symlink(filepath.Base(target), archivedPath); err != nil {
					t.Fatalf("symlink archived plan: %v", err)
				}
			}
			chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
			if err := ValidateArchivedCandidate(root, archivedPlan, chain); err == nil || !strings.Contains(err.Error(), "git mode") {
				t.Fatalf("expected archived plan mode rejection, got %v", err)
			}
		})
	}
}

func TestValidateArchivedCandidateRejectsSupplementModeOrTypeChange(t *testing.T) {
	for _, mutation := range []string{"executable", "symlink"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			initGit(t, root)
			activePlan := "docs/plans/active/2026-07-13-test.md"
			archivedPlan := "docs/plans/archived/2026-07-13-test.md"
			activeSupplement := "docs/plans/active/supplements/2026-07-13-test/notes.md"
			archivedSupplement := "docs/plans/archived/supplements/2026-07-13-test/notes.md"
			writeFile(t, root, activePlan, reviewedPlan)
			writeFile(t, root, activeSupplement, "reviewed supplement\n")
			git(t, root, "add", ".")
			git(t, root, "commit", "-m", "reviewed")
			covered := git(t, root, "rev-parse", "HEAD")
			commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
			if err != nil {
				t.Fatalf("render command-owned archive baseline: %v", err)
			}
			writeFile(t, root, archivedPlan, string(commandRendered))
			archivedSupplementPath := writeFile(t, root, archivedSupplement, "reviewed supplement\n")
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
				t.Fatalf("remove active plan: %v", err)
			}
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(activeSupplement))); err != nil {
				t.Fatalf("remove active supplement: %v", err)
			}
			if mutation == "executable" {
				if err := os.Chmod(archivedSupplementPath, 0o755); err != nil {
					t.Fatalf("chmod archived supplement: %v", err)
				}
			} else {
				target := archivedSupplementPath + ".target"
				if err := os.Rename(archivedSupplementPath, target); err != nil {
					t.Fatalf("move archived supplement target: %v", err)
				}
				if err := os.Symlink(filepath.Base(target), archivedSupplementPath); err != nil {
					t.Fatalf("symlink archived supplement: %v", err)
				}
			}
			chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
			if err := ValidateArchivedCandidate(root, archivedPlan, chain); err == nil || !strings.Contains(err.Error(), "git mode") {
				t.Fatalf("expected archived supplement mode rejection, got %v", err)
			}
		})
	}
}

func TestValidateArchivedCandidateAllowsModePreservingSupplementMove(t *testing.T) {
	for _, mode := range []string{"executable", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			initGit(t, root)
			activePlan := "docs/plans/active/2026-07-13-test.md"
			archivedPlan := "docs/plans/archived/2026-07-13-test.md"
			activeSupplement := "docs/plans/active/supplements/2026-07-13-test/notes"
			archivedSupplement := "docs/plans/archived/supplements/2026-07-13-test/notes"
			writeFile(t, root, activePlan, reviewedPlan)
			activeSupplementPath := filepath.Join(root, filepath.FromSlash(activeSupplement))
			if err := os.MkdirAll(filepath.Dir(activeSupplementPath), 0o755); err != nil {
				t.Fatalf("mkdir active supplement: %v", err)
			}
			if mode == "executable" {
				writeFile(t, root, activeSupplement, "#!/bin/sh\nexit 0\n")
				if err := os.Chmod(activeSupplementPath, 0o755); err != nil {
					t.Fatalf("chmod active supplement: %v", err)
				}
			} else if err := os.Symlink("shared-notes.md", activeSupplementPath); err != nil {
				t.Fatalf("create active supplement symlink: %v", err)
			}
			git(t, root, "add", ".")
			git(t, root, "commit", "-m", "reviewed")
			covered := git(t, root, "rev-parse", "HEAD")
			commandRendered, err := commandRenderedArchivePlan([]byte(reviewedPlan))
			if err != nil {
				t.Fatalf("render command-owned archive baseline: %v", err)
			}
			writeFile(t, root, archivedPlan, string(commandRendered))
			archivedSupplementPath := filepath.Join(root, filepath.FromSlash(archivedSupplement))
			if err := os.MkdirAll(filepath.Dir(archivedSupplementPath), 0o755); err != nil {
				t.Fatalf("mkdir archived supplement: %v", err)
			}
			if err := os.Rename(activeSupplementPath, archivedSupplementPath); err != nil {
				t.Fatalf("move supplement: %v", err)
			}
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(activePlan))); err != nil {
				t.Fatalf("remove active plan: %v", err)
			}
			chain := &Chain{CoveredHeadSHA: covered, ReviewedPlanPath: activePlan}
			if err := ValidateArchivedCandidate(root, archivedPlan, chain); err != nil {
				t.Fatalf("expected mode-preserving supplement move to pass: %v", err)
			}
		})
	}
}
