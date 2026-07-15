package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/cli"
	"github.com/catu-ai/easyharness/internal/timeline"
)

func TestIntegratedReviewCLIStartAndSubmitCompleteRound(t *testing.T) {
	root := writeReviewV3CLIPlan(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return t.TempDir(), nil }

	if code := app.Run([]string{"execute", "start"}); code != 0 {
		t.Fatalf("execute start failed (%d): %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"obsolete":"input is not consumed"}`)
	if code := app.Run([]string{"review", "start"}); code != 0 {
		t.Fatalf("review start failed (%d): %s\n%s", code, stderr.String(), stdout.String())
	}
	var started struct {
		OK        bool   `json:"ok"`
		Command   string `json:"command"`
		Artifacts struct {
			RoundID  string `json:"round_id"`
			Reviewer *struct {
				ReviewFocus    string `json:"review_focus"`
				SubmissionPath string `json:"submission_path"`
			} `json:"reviewer"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	if !started.OK || started.Command != "review start" || started.Artifacts.RoundID != "review-001-full" || started.Artifacts.Reviewer == nil {
		t.Fatalf("unexpected start result: %#v", started)
	}
	if !strings.Contains(started.Artifacts.Reviewer.ReviewFocus, "CLI state and coverage") {
		t.Fatalf("review focus was not surfaced: %#v", started.Artifacts.Reviewer)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Complete candidate passes.","findings":[]}`)
	if code := app.Run([]string{"review", "submit", "--round", started.Artifacts.RoundID, "--by", "reviewer-integrated"}); code != 0 {
		t.Fatalf("review submit failed (%d): %s\n%s", code, stderr.String(), stdout.String())
	}
	var submitted struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Review  *struct {
			Decision string `json:"decision"`
		} `json:"review"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &submitted); err != nil {
		t.Fatal(err)
	}
	if !submitted.OK || submitted.Command != "review submit" || submitted.Review == nil || submitted.Review.Decision != "pass" {
		t.Fatalf("submit did not complete decision: %#v", submitted)
	}
	events := timeline.Service{Workdir: root}.Read()
	if !events.OK || len(events.Events) == 0 || events.Events[len(events.Events)-1].Command != "review submit" {
		t.Fatalf("expected submit timeline event: %#v", events)
	}
}

func TestIntegratedReviewCLIAbortPreservesHistoryAndAllowsReplacement(t *testing.T) {
	root := writeReviewV3CLIPlan(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return t.TempDir(), nil }

	if code := app.Run([]string{"execute", "start"}); code != 0 {
		t.Fatalf("execute start failed (%d): %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"review", "start"}); code != 0 {
		t.Fatalf("review start failed (%d): %s", code, stderr.String())
	}
	var started struct {
		Artifacts struct {
			RoundID string `json:"round_id"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &started); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"review", "abort", "--round", started.Artifacts.RoundID}); code != 0 {
		t.Fatalf("review abort failed (%d): %s\n%s", code, stderr.String(), stdout.String())
	}
	var aborted struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &aborted); err != nil {
		t.Fatal(err)
	}
	if !aborted.OK || aborted.Command != "review abort" {
		t.Fatalf("unexpected abort result: %#v", aborted)
	}
	events := timeline.Service{Workdir: root}.Read()
	if !events.OK || len(events.Events) == 0 || events.Events[len(events.Events)-1].Command != "review abort" {
		t.Fatalf("expected abort timeline event: %#v", events)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"review", "start"}); code != 0 {
		t.Fatalf("replacement review start failed (%d): %s\n%s", code, stderr.String(), stdout.String())
	}
}

func TestRemovedReviewOrchestrationCommandsAreRejected(t *testing.T) {
	for _, args := range [][]string{{"review", "aggregate", "--round", "review-001-full"}, {"review", "dimensions", "list"}} {
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		app := cli.New(stdout, stderr)
		if code := app.Run(args); code != 2 {
			t.Fatalf("%v returned %d, stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "unknown review subcommand") {
			t.Fatalf("expected removed command diagnostic for %v: %q", args, stderr.String())
		}
	}
}

func TestIntegratedReviewSubmitStillUsesSubmissionSchema(t *testing.T) {
	root := writeReviewV3CLIPlan(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Getwd = func() (string, error) { return root, nil }
	app.UserHomeDir = func() (string, error) { return t.TempDir(), nil }
	if code := app.Run([]string{"execute", "start"}); code != 0 {
		t.Fatalf("execute start failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"review", "start"}); code != 0 {
		t.Fatalf("review start failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"findings":[]}`)
	if code := app.Run([]string{"review", "submit", "--round", "review-001-full", "--by", "reviewer-integrated"}); code != 1 {
		t.Fatalf("expected schema failure, code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"path": "submission.summary"`) {
		t.Fatalf("expected summary validation error: %s", stdout.String())
	}
}

func writeReviewV3CLIPlan(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	content := `---
template_version: 0.3.0
created_at: "2026-07-13T00:00:00Z"
approved_at: "2026-07-13T00:01:00Z"
source_type: direct_request
source_refs: []
size: S
---
# CLI Review V3
## Goal
Exercise the integrated reviewer CLI.
### Decisions and Constraints
- Final review is mandatory.
## Scope
### In Scope
- Review CLI.
### Out of Scope
- UI.
## Acceptance Criteria
- [x] Review CLI completes.
## Review Focus
- Challenge CLI state and coverage.
## Deferred Items
- None.
## Work Breakdown
### Step 1: Candidate
- Done: [x]
- Outcome: Candidate exists.
- Covers: Review CLI completes.
- Check: CLI tests pass.
## Validation Strategy
- Run CLI tests.
## Closeout
- Validation: Complete.
- Review: Complete.
- Delivered: Complete.
- Not Delivered: None.
- Follow-Up Issues: NONE
- PR: Test.
- Ready: Yes.
- Merge Handoff: Test.
`
	path := filepath.Join(root, "docs", "plans", "active", "2026-07-13-cli-review-v3.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCLIReviewV3(t, root, "init", "-q")
	gitCLIReviewV3(t, root, "config", "user.name", "Codex Test")
	gitCLIReviewV3(t, root, "config", "user.email", "codex@example.com")
	gitCLIReviewV3(t, root, "add", ".")
	gitCLIReviewV3(t, root, "commit", "-qm", "fixture")
	return root
}

func gitCLIReviewV3(t *testing.T, root string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
