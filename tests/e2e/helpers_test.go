package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

type commandError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type reviewHandle struct {
	Instructions   string `json:"instructions"`
	ReviewFocus    string `json:"review_focus"`
	SubmissionPath string `json:"submission_path"`
}

type executeStartResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	Facts struct {
		Revision int `json:"revision"`
	} `json:"facts"`
	Artifacts struct {
		LocalStatePath string `json:"local_state_path"`
	} `json:"artifacts"`
}

type lifecycleCommandResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	Facts struct {
		Revision   int    `json:"revision"`
		ReopenMode string `json:"reopen_mode"`
		LandPRURL  string `json:"land_pr_url"`
		LandCommit string `json:"land_commit"`
	} `json:"facts"`
	Artifacts struct {
		FromPlanPath    string `json:"from_plan_path"`
		ToPlanPath      string `json:"to_plan_path"`
		LocalStatePath  string `json:"local_state_path"`
		CurrentPlanPath string `json:"current_plan_path"`
	} `json:"artifacts"`
}

type reviewStartResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Artifacts struct {
		ProjectRoot     string        `json:"project_root"`
		PlanPath        string        `json:"plan_path"`
		RoundID         string        `json:"round_id"`
		ReviewedHeadSHA string        `json:"reviewed_head_sha"`
		Reviewer        *reviewHandle `json:"reviewer"`
	} `json:"artifacts"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type evidenceSubmitResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Artifacts struct {
		PlanPath       string `json:"plan_path"`
		LocalStatePath string `json:"local_state_path"`
		RecordID       string `json:"record_id"`
		RecordPath     string `json:"record_path"`
		Kind           string `json:"kind"`
	} `json:"artifacts"`
}

type submitResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Artifacts struct {
		ProjectRoot    string `json:"project_root"`
		SubmissionPath string `json:"submission_path"`
		RoundID        string `json:"round_id"`
	} `json:"artifacts"`
	Review struct {
		RoundID          string `json:"round_id"`
		ReviewedHeadSHA  string `json:"reviewed_head_sha"`
		Decision         string `json:"decision"`
		BlockingFindings []struct {
			FindingID string `json:"finding_id"`
			Area      string `json:"area"`
			Severity  string `json:"severity"`
			Title     string `json:"title"`
		} `json:"blocking_findings"`
		NonBlockingFindings []struct {
			Severity string `json:"severity"`
			Title    string `json:"title"`
			Details  string `json:"details"`
		} `json:"non_blocking_findings"`
		UnresolvedFindingIDs []string `json:"unresolved_finding_ids"`
	} `json:"review"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type reviewManifest struct {
	RoundID         string `json:"round_id"`
	Kind            string `json:"kind"`
	AnchorSHA       string `json:"anchor_sha"`
	ReviewedHeadSHA string `json:"reviewed_head_sha"`
	ReviewFocus     string `json:"review_focus"`
}

type reviewSubmission struct {
	RoundID     string          `json:"round_id"`
	Slot        string          `json:"slot"`
	By          string          `json:"by"`
	SubmittedAt string          `json:"submitted_at"`
	Summary     string          `json:"summary"`
	Resolutions json.RawMessage `json:"resolutions"`
	Findings    []struct {
		Area     string `json:"area"`
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Details  string `json:"details"`
	} `json:"findings"`
}

type currentPlan struct {
	PlanPath           string `json:"plan_path"`
	LastLandedPlanPath string `json:"last_landed_plan_path"`
	LastLandedAt       string `json:"last_landed_at"`
}

type statusResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	Facts struct {
		CurrentStep         string `json:"current_step"`
		ReviewStatus        string `json:"review_status"`
		ReopenMode          string `json:"reopen_mode"`
		Revision            int    `json:"revision"`
		StepCompleted       int    `json:"step_completed"`
		StepTotal           int    `json:"step_total"`
		AcceptanceCompleted int    `json:"acceptance_completed"`
		AcceptanceTotal     int    `json:"acceptance_total"`
		Evidence            struct {
			Recorded struct {
				Publish struct {
					Status string `json:"status"`
					PRURL  string `json:"pr_url"`
				} `json:"publish"`
				CI struct {
					Status string `json:"status"`
				} `json:"ci"`
				Sync struct {
					Status string `json:"status"`
				} `json:"sync"`
			} `json:"recorded"`
			Remote struct {
				Observation string `json:"observation"`
				Assessment  string `json:"assessment"`
			} `json:"remote"`
		} `json:"evidence"`
		LandPRURL string `json:"land_pr_url"`
	} `json:"facts"`
	Artifacts struct {
		ProjectRoot          string `json:"project_root"`
		PlanPath             string `json:"plan_path"`
		ReviewRoundID        string `json:"review_round_id"`
		ReviewSubmissionPath string `json:"review_submission_path"`
		ReviewedHeadSHA      string `json:"reviewed_head_sha"`
		LastLandedAt         string `json:"last_landed_at"`
	} `json:"artifacts"`
	Blockers   []commandError `json:"blockers"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type runState struct {
	ExecutionStartedAt string `json:"execution_started_at"`
	Revision           int    `json:"revision"`
	FinalizeCoverage   struct {
		RootRoundID    string `json:"root_round_id"`
		TipRoundID     string `json:"tip_round_id"`
		CoveredHeadSHA string `json:"covered_head_sha"`
		Revision       int    `json:"revision"`
	} `json:"finalize_coverage"`
	ActiveReviewRound struct {
		RoundID    string `json:"round_id"`
		Aggregated bool   `json:"aggregated"`
		Decision   string `json:"decision"`
		Step       *int   `json:"step,omitempty"`
		Revision   int    `json:"revision"`
		Kind       string `json:"kind"`
	} `json:"active_review_round"`
}

func runStatus(t *testing.T, workdir string) statusResult {
	t.Helper()

	status := support.Run(t, workdir, "status")
	support.RequireSuccess(t, status)
	support.RequireNoStderr(t, status)
	return support.RequireJSONResult[statusResult](t, status)
}

func requireLifecycleResult(t *testing.T, result support.Result) lifecycleCommandResult {
	t.Helper()

	var raw map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &raw); err != nil {
		t.Fatalf("expected JSON lifecycle output: %v\n%s", err, result.Stdout)
	}
	assertNoLegacyLifecycleFields(t, raw)
	return support.RequireJSONResult[lifecycleCommandResult](t, result)
}

func requireExecuteStartResult(t *testing.T, result support.Result) executeStartResult {
	t.Helper()

	var raw map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &raw); err != nil {
		t.Fatalf("expected JSON execute-start output: %v\n%s", err, result.Stdout)
	}
	assertNoLegacyLifecycleFields(t, raw)
	return support.RequireJSONResult[executeStartResult](t, result)
}

func assertNoLegacyLifecycleFields(t *testing.T, payload map[string]any) {
	t.Helper()

	state, ok := payload["state"].(map[string]any)
	if !ok {
		t.Fatalf("expected lifecycle payload state object, got %#v", payload)
	}
	if _, ok := state["plan_status"]; ok {
		t.Fatalf("expected lifecycle payload to omit plan_status, got %#v", state)
	}
	if _, ok := state["lifecycle"]; ok {
		t.Fatalf("expected lifecycle payload to omit lifecycle, got %#v", state)
	}
	if _, ok := state["revision"]; ok {
		t.Fatalf("expected lifecycle payload to omit state.revision, got %#v", state)
	}
}

func assertLifecycleEnvelope(t *testing.T, payload lifecycleCommandResult, wantNode string, wantRevision int) {
	t.Helper()

	if payload.State.CurrentNode != wantNode {
		t.Fatalf("expected lifecycle current node %q, got %#v", wantNode, payload)
	}
	if payload.Facts.Revision != wantRevision {
		t.Fatalf("expected lifecycle revision %d, got %#v", wantRevision, payload)
	}
}

func assertNode(t *testing.T, status statusResult, want string) {
	t.Helper()
	if status.State.CurrentNode != want {
		t.Fatalf("expected current node %q, got %#v", want, status)
	}
}

func assertRawStateJSONOmitsKeys(t *testing.T, path string, keys ...string) {
	t.Helper()
	var payload map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw state json: %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse raw state json: %v", err)
	}
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected raw state json to omit %q, got %#v", key, payload)
		}
	}
}

func submitReview(t *testing.T, workspace *support.Workspace, roundID, reviewer, summary string, findings []map[string]any, resolutions []map[string]any) submitResult {
	t.Helper()

	for _, finding := range findings {
		if _, ok := finding["area"]; !ok {
			finding["area"] = "candidate-correctness"
		}
	}
	payload := map[string]any{
		"summary":  summary,
		"findings": findings,
	}
	if resolutions != nil {
		payload["resolutions"] = resolutions
	}
	submissionPath := workspace.WriteJSON(t, fmt.Sprintf("tmp/%s-submission.json", roundID), payload)

	submit := support.Run(
		t,
		workspace.Root,
		"review", "submit",
		"--round", roundID,
		"--by", reviewer,
		"--input", submissionPath,
	)
	support.RequireSuccess(t, submit)
	support.RequireNoStderr(t, submit)
	submitPayload := support.RequireJSONResult[submitResult](t, submit)
	if !submitPayload.OK || submitPayload.Command != "review submit" {
		t.Fatalf("unexpected review-submit payload: %#v", submitPayload)
	}
	if submitPayload.Artifacts.RoundID != roundID || submitPayload.Artifacts.SubmissionPath == "" {
		t.Fatalf("unexpected submit artifacts: %#v", submitPayload)
	}
	support.RequireFileExists(t, filepath.Join(workspace.Root, filepath.FromSlash(submitPayload.Artifacts.SubmissionPath)))
	return submitPayload
}

func trackedStepTitle(stepNumber int, stepTitle string) string {
	return fmt.Sprintf("Step %d: %s", stepNumber, stepTitle)
}

func resolveRepoPath(root, path string) string {
	return filepath.Join(root, filepath.FromSlash(path))
}

func reviewRoundArtifactPath(root, planStem, roundID string, elems ...string) string {
	parts := []string{root, ".local", "harness", "plans", planStem, "reviews", roundID}
	parts = append(parts, elems...)
	return filepath.Join(parts...)
}

func startReviewRound(t *testing.T, workspace *support.Workspace, forceFull bool) reviewStartResult {
	t.Helper()

	workspace.CommitAll(t, "checkpoint review candidate")
	args := []string{"review", "start"}
	if forceFull {
		args = append(args, "--full")
	}
	start := support.Run(t, workspace.Root, args...)
	support.RequireSuccess(t, start)
	support.RequireNoStderr(t, start)
	payload := support.RequireJSONResult[reviewStartResult](t, start)
	if !payload.OK || payload.Command != "review start" {
		t.Fatalf("unexpected review-start payload: %#v", payload)
	}
	if payload.Artifacts.ReviewedHeadSHA == "" {
		t.Fatalf("expected built review-start command to surface the captured candidate HEAD, got %#v", payload.Artifacts)
	}
	if payload.Artifacts.Reviewer == nil || payload.Artifacts.Reviewer.SubmissionPath == "" || payload.Artifacts.Reviewer.Instructions == "" {
		t.Fatalf("expected one integrated reviewer handoff, got %#v", payload.Artifacts)
	}
	return payload
}

func currentWorkspaceHead(t *testing.T, root string) string {
	t.Helper()

	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		if os.IsNotExist(err) {
			return "anchor-sha"
		}
		t.Fatalf("stat .git: %v", err)
	}

	return resolveWorkspaceRevision(t, root, "HEAD")
}

func resolveWorkspaceRevision(t *testing.T, root, revision string) string {
	t.Helper()

	output, err := exec.Command("git", "-C", root, "rev-parse", revision).CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v\n%s", revision, err, output)
	}
	resolved := strings.TrimSpace(string(output))
	if resolved == "" {
		t.Fatalf("git rev-parse %s returned an empty commit for %s", revision, root)
	}
	return resolved
}

func runPassingFinalizeReview(t *testing.T, workspace *support.Workspace) string {
	t.Helper()

	startPayload := startReviewRound(t, workspace, false)

	inReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, inReviewStatus, "execution/finalize/review")
	if inReviewStatus.Facts.ReviewStatus != "in_progress" || inReviewStatus.Artifacts.ReviewSubmissionPath == "" {
		t.Fatalf("expected active finalize-review facts, got %#v", inReviewStatus)
	}

	submission := submitReview(t, workspace, startPayload.Artifacts.RoundID, "integrated-reviewer", "The complete candidate is ready to archive.", nil, nil)
	if submission.Review.Decision != "pass" {
		t.Fatalf("expected passing finalize review, got %#v", submission)
	}
	return startPayload.Artifacts.RoundID
}

func drivePlanToArchivedPublishNode(t *testing.T, workspace *support.Workspace, planPath string, stepTitles ...string) lifecycleCommandResult {
	t.Helper()

	support.ApprovePlan(t, planPath, "2026-03-22T00:05:00Z")
	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	for index, stepTitle := range stepTitles {
		_ = stepTitle
		support.CompleteStep(t, planPath, index+1)
	}

	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	runPassingFinalizeReview(t, workspace)
	support.CompleteCloseout(t, planPath)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")
	stillArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, stillArchiveStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	payload := requireLifecycleResult(t, archive)
	if !payload.OK || payload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", payload)
	}

	postArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postArchiveStatus, "execution/finalize/publish")

	return payload
}

func compactPlanFixture(stepTitles ...string) string {
	if len(stepTitles) == 0 {
		panic("compact plan fixture requires at least one step title")
	}

	var body strings.Builder
	body.WriteString("## Goal\n\nExercise the built harness workflow through a compact tracked plan.\n\n")
	body.WriteString("### Decisions and Constraints\n\n- Final integrated review is mandatory.\n\n")
	body.WriteString("## Scope\n\n### In Scope\n\n- The lifecycle behavior under test.\n\n### Out of Scope\n\n- Unrelated product behavior.\n\n")
	body.WriteString("## Acceptance Criteria\n\n")
	for index := range stepTitles {
		fmt.Fprintf(&body, "- [ ] Criterion %d\n", index+1)
	}
	body.WriteString("\n## Review Focus\n\n- Challenge lifecycle correctness, review coverage, and persisted artifacts.\n\n")
	body.WriteString("## Deferred Items\n\n- None.\n\n## Work Breakdown\n")
	for index, title := range stepTitles {
		title = strings.TrimSpace(title)
		fmt.Fprintf(&body, "\n### Step %d: %s\n\n- Done: [ ]\n- Outcome: Complete %s.\n- Covers: Criterion %d\n- Check: Verify the step through the built binary.\n", index+1, title, title, index+1)
	}
	body.WriteString("\n## Validation Strategy\n\n- Run the built-binary end-to-end scenario.\n\n## Closeout\n\n")
	body.WriteString("- Validation: PENDING_UNTIL_ARCHIVE\n- Review: PENDING_UNTIL_ARCHIVE\n- Delivered: PENDING_UNTIL_ARCHIVE\n- Not Delivered: PENDING_UNTIL_ARCHIVE\n- Follow-Up Issues: NONE\n")
	return body.String()
}

func drivePlanToAwaitMergeNode(t *testing.T, workspace *support.Workspace, planPath string, stepTitles ...string) {
	t.Helper()

	drivePlanToArchivedPublishNode(t, workspace, planPath, stepTitles...)

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
		"branch": "codex/e2e-lifecycle-handoff-coverage",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/2",
	})
	submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status": "not_applied",
		"reason": "fixture does not model a remote base comparison",
	})

	postSyncStatus := runStatus(t, workspace.Root)
	assertNode(t, postSyncStatus, "execution/finalize/await_merge")
}

func submitEvidence(t *testing.T, workspace *support.Workspace, kind, inputRelPath string, payload map[string]any) evidenceSubmitResult {
	t.Helper()

	inputPath := workspace.WriteJSON(t, inputRelPath, payload)
	result := support.Run(t, workspace.Root, "evidence", "submit", "--kind", kind, "--input", inputPath)
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	parsed := support.RequireJSONResult[evidenceSubmitResult](t, result)
	if !parsed.OK || parsed.Command != "evidence submit" {
		t.Fatalf("unexpected evidence-submit payload: %#v", parsed)
	}
	return parsed
}
