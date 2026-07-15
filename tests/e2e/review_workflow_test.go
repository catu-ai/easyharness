package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	reviewWorkflowTitle = "Review Workflow Plan"
	stepOneTitle        = "Build the candidate"
	stepTwoTitle        = "Validate the integrated workflow"
)

func TestReviewWorkflowWithBuiltBinary(t *testing.T) {
	workspace, _, planStem := prepareFinalizeReviewFixture(t)

	preReview := runStatus(t, workspace.Root)
	assertNode(t, preReview, "execution/finalize/review")
	if preReview.Facts.StepCompleted != 2 || preReview.Facts.StepTotal != 2 || preReview.Facts.AcceptanceCompleted != 2 || preReview.Facts.AcceptanceTotal != 2 {
		t.Fatalf("expected compact plan progress facts, got %#v", preReview.Facts)
	}

	full := startReviewRound(t, workspace, false)
	if !strings.HasSuffix(full.Artifacts.RoundID, "-full") {
		t.Fatalf("expected mandatory initial full review, got %#v", full)
	}
	if full.Artifacts.Reviewer.ReviewFocus == "" {
		t.Fatalf("expected plan-scoped Review Focus in reviewer handoff, got %#v", full.Artifacts.Reviewer)
	}
	manifest := support.ReadJSONFile[reviewManifest](t, reviewRoundArtifactPath(workspace.Root, planStem, full.Artifacts.RoundID, "manifest.json"))
	if manifest.Kind != "full" || manifest.ReviewedHeadSHA != full.Artifacts.ReviewedHeadSHA || manifest.ReviewFocus == "" {
		t.Fatalf("unexpected full review manifest: %#v", manifest)
	}
	generatedSubmissionPath := resolveRepoPath(workspace.Root, full.Artifacts.Reviewer.SubmissionPath)
	generatedSubmission := support.ReadJSONFile[map[string]any](t, generatedSubmissionPath)
	generatedSubmission["summary"] = "One candidate defect remains."
	generatedSubmission["findings"] = []map[string]any{{
		"area": "review-lifecycle", "severity": "important", "title": "Repair the candidate", "details": "The complete candidate needs a narrow repair.",
	}}
	editedSubmission, err := json.MarshalIndent(generatedSubmission, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	editedSubmission = append(editedSubmission, '\n')
	if err := os.WriteFile(generatedSubmissionPath, editedSubmission, 0o644); err != nil {
		t.Fatal(err)
	}

	blockedCommand := support.Run(t, workspace.Root, "review", "submit", "--round", full.Artifacts.RoundID, "--by", "integrated-reviewer", "--input", generatedSubmissionPath)
	support.RequireSuccess(t, blockedCommand)
	blocked := support.RequireJSONResult[submitResult](t, blockedCommand)
	if blocked.Review.Decision != "changes_requested" || len(blocked.Review.BlockingFindings) != 1 {
		t.Fatalf("expected blocking submission to decide the round immediately, got %#v", blocked)
	}
	findingID := blocked.Review.BlockingFindings[0].FindingID
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/fix")

	workspace.WriteFile(t, "candidate/narrow-fix.txt", []byte("fixed\n"))
	delta := startReviewRound(t, workspace, false)
	if !strings.HasSuffix(delta.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected inferred linked delta after a narrow repair, got %#v", delta)
	}
	deltaManifest := support.ReadJSONFile[reviewManifest](t, reviewRoundArtifactPath(workspace.Root, planStem, delta.Artifacts.RoundID, "manifest.json"))
	if deltaManifest.Kind != "delta" || deltaManifest.AnchorSHA != full.Artifacts.ReviewedHeadSHA {
		t.Fatalf("expected delta anchored to the full review, got %#v", deltaManifest)
	}

	passing := submitReview(t, workspace, delta.Artifacts.RoundID, "integrated-reviewer", "The narrow repair closes the finding.", nil, []map[string]any{{
		"finding_id": findingID, "status": "resolved", "details": "The committed repair removes the defect.",
	}})
	if passing.Review.Decision != "pass" || len(passing.Review.UnresolvedFindingIDs) != 0 {
		t.Fatalf("expected linked repair submission to complete coverage, got %#v", passing)
	}
	assertNode(t, runStatus(t, workspace.Root), "execution/finalize/archive")
}

func TestReviewFullResetAndCandidateBoundaryWithBuiltBinary(t *testing.T) {
	workspace, _, _ := prepareFinalizeReviewFixture(t)

	workspace.WriteFile(t, "candidate/dirty.txt", []byte("dirty\n"))
	dirty := support.Run(t, workspace.Root, "review", "start")
	support.RequireExitCode(t, dirty, 1)
	os.Remove(workspace.Path("candidate/dirty.txt"))

	full := startReviewRound(t, workspace, false)
	blocked := submitReview(t, workspace, full.Artifacts.RoundID, "integrated-reviewer", "Broad design work remains.", []map[string]any{{
		"area": "candidate-design", "severity": "blocker", "title": "Reset the design", "details": "The repair changes the candidate design broadly.",
	}}, nil)
	if blocked.Review.Decision != "changes_requested" {
		t.Fatalf("expected blocking full review, got %#v", blocked)
	}

	workspace.WriteFile(t, "candidate/broad-redesign.txt", []byte("redesigned\n"))
	reset := startReviewRound(t, workspace, true)
	if !strings.HasSuffix(reset.Artifacts.RoundID, "-full") {
		t.Fatalf("expected explicit --full to reset broad repair coverage, got %#v", reset)
	}

	workspace.WriteFile(t, "candidate/moved-after-review.txt", []byte("changed\n"))
	workspace.CommitAll(t, "move candidate after review start")
	changed := support.Run(t, workspace.Root, "review", "submit", "--round", reset.Artifacts.RoundID, "--by", "integrated-reviewer", "--input", workspace.WriteJSON(t, "tmp/changed-head-submission.json", map[string]any{
		"summary": "This stale judgment must not persist.", "findings": []map[string]any{},
	}))
	support.RequireExitCode(t, changed, 1)
}

func TestReviewRewrittenAncestryFallsBackToFullWithBuiltBinary(t *testing.T) {
	workspace, _, planStem := prepareFinalizeReviewFixture(t)
	workspace.CommitAll(t, "base fixture")
	workspace.WriteFile(t, "candidate/product.txt", []byte("candidate\n"))
	workspace.CommitAll(t, "candidate under review")

	full := startReviewRound(t, workspace, false)
	passing := submitReview(t, workspace, full.Artifacts.RoundID, "integrated-reviewer", "The candidate passes.", nil, nil)
	if passing.Review.Decision != "pass" {
		t.Fatalf("expected initial full review to pass, got %#v", passing)
	}
	reviewedHead := full.Artifacts.ReviewedHeadSHA
	base := runReviewGit(t, workspace.Root, "rev-parse", reviewedHead+"^")
	runReviewGit(t, workspace.Root, "branch", "candidate", reviewedHead)
	runReviewGit(t, workspace.Root, "checkout", "-qb", "upstream", base)
	workspace.WriteFile(t, "upstream.txt", []byte("upstream\n"))
	runReviewGit(t, workspace.Root, "add", "upstream.txt")
	runReviewGit(t, workspace.Root, "commit", "-qm", "advance upstream")
	runReviewGit(t, workspace.Root, "checkout", "-q", "candidate")
	runReviewGit(t, workspace.Root, "rebase", "upstream")
	if runReviewGitExitCode(t, workspace.Root, "merge-base", "--is-ancestor", reviewedHead, "HEAD") == 0 {
		t.Fatal("test setup expected rebase to rewrite reviewed ancestry")
	}

	replacement := startReviewRound(t, workspace, false)
	if !strings.HasSuffix(replacement.Artifacts.RoundID, "-full") {
		t.Fatalf("expected rewritten ancestry to establish a full root, got %#v", replacement)
	}
	manifest := support.ReadJSONFile[reviewManifest](t, reviewRoundArtifactPath(workspace.Root, planStem, replacement.Artifacts.RoundID, "manifest.json"))
	if manifest.Kind != "full" || manifest.AnchorSHA != "" {
		t.Fatalf("unexpected replacement review manifest: %#v", manifest)
	}
}

func runReviewGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func runReviewGitExitCode(t *testing.T, root string, args ...string) int {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	err := command.Run()
	if err == nil {
		return 0
	}
	exit, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return exit.ExitCode()
}

func prepareFinalizeReviewFixture(t *testing.T) (*support.Workspace, string, string) {
	t.Helper()
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-22-review-workflow.md"
	planPath := workspace.Path(planRelPath)
	template := support.Run(t, workspace.Root, "plan", "template", "--title", reviewWorkflowTitle, "--timestamp", "2026-03-22T00:00:00Z", "--source-type", "issue", "--source-ref", "#6", "--output", planRelPath)
	support.RequireSuccess(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, reviewWorkflowTitle, reviewWorkflowPlanBody())
	support.ApprovePlan(t, planPath, "2026-03-22T00:05:00Z")
	support.RequireSuccess(t, support.Run(t, workspace.Root, "execute", "start"))
	support.CompleteStep(t, planPath, 1)
	support.CompleteStep(t, planPath, 2)
	support.CheckAllAcceptanceCriteria(t, planPath)
	return workspace, planPath, "2026-03-22-review-workflow"
}

func reviewWorkflowPlanBody() string {
	return compactPlanFixture(stepOneTitle, stepTwoTitle)
}
