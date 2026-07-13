package status_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/install"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/remote"
	"github.com/catu-ai/easyharness/internal/review"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
)

const (
	stepOneTitle = "Step 1: Replace with first step title"
	stepTwoTitle = "Step 2: Replace with second step title"
)

func TestStatusPlanNodeForActivePlan(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.NextAction) < 2 || result.NextAction[0].Command != nil {
		t.Fatalf("expected human-approval guidance first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness plan approve --by human" {
		t.Fatalf("expected explicit approval guidance, got %#v", result.NextAction)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected status read to avoid caching plan node, got %#v", state)
	}

	doc, err := plan.LoadFile(filepath.Join(root, "docs/plans/active/2026-03-18-status-plan.md"))
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if got := doc.DerivedLifecycle(nil); got != "awaiting_plan_approval" {
		t.Fatalf("expected lifecycle to derive from the plan alone, got %q", got)
	}
}

func TestStatusPlanNodeForApprovedActivePlan(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return approvePlanContent(content, "2026-03-18T10:05:00+08:00")
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness execute start" {
		t.Fatalf("expected execute-start guidance for approved plan, got %#v", result.NextAction)
	}
}

func TestStatusPlanNodeForTrackedLightweightPlan(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
		return strings.Replace(content, "size: M", "size: XXS", 1)
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.PlanPath, relPath) {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
	if result.Artifacts.SupplementsPath != "" {
		t.Fatalf("expected no supplements path for markdown-only plan, got %#v", result.Artifacts)
	}
}

func TestStatusSurfacesSupplementsDirectoryForCurrentPlanPackage(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-18-status-plan.md"
	writePlan(t, root, relPath, func(content string) string {
		return content
	})
	supplementPath := filepath.Join(root, "docs/plans/active/supplements/2026-03-18-status-plan/spec.md")
	if err := os.MkdirAll(filepath.Dir(supplementPath), 0o755); err != nil {
		t.Fatalf("mkdir supplements dir: %v", err)
	}
	if err := os.WriteFile(supplementPath, []byte("# spec draft\n"), 0o644); err != nil {
		t.Fatalf("write supplements file: %v", err)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.SupplementsPath, "docs/plans/active/supplements/2026-03-18-status-plan") {
		t.Fatalf("unexpected supplements artifacts: %#v", result.Artifacts)
	}
}

func TestStatusSurfacesSupplementsDirectoryForArchivedPlanPackage(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/archived/2026-03-18-status-plan.md"
	writePlan(t, root, relPath, func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, relPath)
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
	})
	supplementPath := filepath.Join(root, "docs/plans/archived/supplements/2026-03-18-status-plan/spec.md")
	if err := os.MkdirAll(filepath.Dir(supplementPath), 0o755); err != nil {
		t.Fatalf("mkdir archived supplements dir: %v", err)
	}
	if err := os.WriteFile(supplementPath, []byte("# archived spec draft\n"), 0o644); err != nil {
		t.Fatalf("write archived supplements file: %v", err)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.SupplementsPath, "docs/plans/archived/supplements/2026-03-18-status-plan") {
		t.Fatalf("unexpected archived supplements artifacts: %#v", result.Artifacts)
	}
}

func TestStatusLightweightPublishPromptsForBreadcrumb(t *testing.T) {
	root := t.TempDir()
	relPath := ".local/harness/plans/archived/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
		content = strings.Replace(content, "size: M", "size: XXS", 1)
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, relPath)
	writeState(t, root, "2026-03-18-status-lightweight", map[string]any{
		"revision": 1,
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance first, got %#v", result.NextAction)
	}
	if result.Artifacts == nil || result.Artifacts.SupplementsPath != "" {
		t.Fatalf("expected no lightweight supplements path when no supplements directory exists, got %#v", result.Artifacts)
	}
	foundCommitPush := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Commit and push the tracked plan change created by archiving") {
			foundCommitPush = true
			break
		}
	}
	if !foundCommitPush {
		t.Fatalf("expected publish guidance to mention commit/push for the tracked archive change, got %#v", result.NextAction)
	}
	if !strings.Contains(result.Summary, "repo-visible breadcrumb") {
		t.Fatalf("expected summary to mention breadcrumb, got %q", result.Summary)
	}
}

func TestStatusLightweightArchivedPlanSurfacesSupplementsDirectory(t *testing.T) {
	root := t.TempDir()
	relPath := ".local/harness/plans/archived/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
		content = strings.Replace(content, "size: M", "size: XXS", 1)
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, relPath)
	writeState(t, root, "2026-03-18-status-lightweight", map[string]any{
		"revision": 1,
	})
	supplementPath := filepath.Join(root, ".local/harness/plans/archived/supplements/2026-03-18-status-lightweight/spec.md")
	if err := os.MkdirAll(filepath.Dir(supplementPath), 0o755); err != nil {
		t.Fatalf("mkdir lightweight supplements dir: %v", err)
	}
	if err := os.WriteFile(supplementPath, []byte("# lightweight archived spec draft\n"), 0o644); err != nil {
		t.Fatalf("write lightweight supplements file: %v", err)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.SupplementsPath, ".local/harness/plans/archived/supplements/2026-03-18-status-lightweight") {
		t.Fatalf("unexpected lightweight archived supplements artifacts: %#v", result.Artifacts)
	}
}

func TestStatusExecutionStepImplementNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T10:05:00+08:00" {
		t.Fatalf("expected execution-start state to remain available, got %#v", state)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "current_node")
}

func TestStatusSnapshotDoesNotCompeteForStateMutationLock(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected status snapshot to remain read-only while state lock is held, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
}

func TestStatusFinalizeReviewNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness review start" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusFinalizeReviewInFlightIncludesReviewFacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": false,
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewKind != "full" || result.Facts.ReviewStatus != "in_progress" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusFinalizeFixNodeAfterFailedFinalizeReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "changes_requested" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusFinalizeArchiveNodeAfterCleanFinalizeReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	initCommittedGitCandidate(t, root)
	completePassingFinalizeReview(t, root)

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/archive" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.Blockers) != 0 {
		t.Fatalf("expected no archive blockers, got %#v", result.Blockers)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness archive" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusInvalidFinalizeCoverageFallsBackToReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	initCommittedGitCandidate(t, root)
	completePassingFinalizeReview(t, root)

	aggregatePath := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-001-full", "aggregate.json")
	if err := os.WriteFile(aggregatePath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("corrupt finalize aggregate: %v", err)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected invalid coverage to conservatively require finalize review, got %#v", result.State)
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness archive" {
			t.Fatalf("did not expect archive guidance for invalid finalize coverage, got %#v", result.NextAction)
		}
	}
}

func TestStatusArchivedPlanNeedsPublishEvidence(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	foundPublishSubmit := false
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness evidence submit --kind publish --input <json>" {
			foundPublishSubmit = true
			break
		}
	}
	if !foundPublishSubmit {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanWithRecordedPRSuggestsEvidenceRefresh(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if !statusNextActionsContain(result, "harness evidence refresh") {
		t.Fatalf("expected evidence refresh guidance after recorded PR, got %#v", result.NextAction)
	}
	if !statusNextActionsContain(result, "harness evidence submit --kind ci --input <json>") {
		t.Fatalf("expected manual CI fallback guidance to remain, got %#v", result.NextAction)
	}
	if !statusNextActionsContain(result, "harness evidence submit --kind sync --input <json>") {
		t.Fatalf("expected manual sync fallback guidance to remain, got %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanSurfacesRemoteHandoffObservation(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommands(`"CLEAN"`, `[{"name":"Go Test","workflow":"CI","bucket":"pass","state":"SUCCESS","link":"https://ci.example/run"}]`),
	}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("live remote facts must not advance without local evidence, got %#v", result.State)
	}
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Observation != "complete" || remoteEvidence.Assessment != "refresh_available" {
		t.Fatalf("unexpected remote assessment: %#v", remoteEvidence)
	}
	if remoteEvidence.PR == nil || remoteEvidence.PR.State != "OPEN" || remoteEvidence.PR.Draft {
		t.Fatalf("unexpected remote PR summary: %#v", remoteEvidence.PR)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal status result: %v", err)
	}
	if !strings.Contains(string(payload), `"draft":false`) {
		t.Fatalf("expected status JSON to include explicit draft:false, got %s", string(payload))
	}
	if remoteEvidence.CI == nil || remoteEvidence.CI.Status != "success" {
		t.Fatalf("unexpected remote CI summary: %#v", remoteEvidence.CI)
	}
	if remoteEvidence.Sync == nil || remoteEvidence.Sync.Status != "fresh" {
		t.Fatalf("unexpected remote sync summary: %#v", remoteEvidence.Sync)
	}
	if !statusNextActionsContain(result, "harness evidence refresh") {
		t.Fatalf("expected evidence refresh guidance for recorded remote facts, got %#v", result.NextAction)
	}
}

func TestStatusRemoteHandoffObservationDegradesWithoutFailingLocalStatus(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand: func(name string, args ...string) remote.CommandResult {
			return remote.CommandResult{Err: exec.ErrNotFound}
		},
	}.Snapshot()
	if !result.OK || result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("remote degradation should not fail local status, got %#v", result)
	}
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Observation != "unavailable" || remoteEvidence.Assessment != "manual_evidence_required" {
		t.Fatalf("expected unavailable remote evidence, got %#v", remoteEvidence)
	}
	if len(remoteEvidence.Degraded) != 1 || remoteEvidence.Degraded[0].Code != "gh_missing" {
		t.Fatalf("expected gh_missing degradation, got %#v", remoteEvidence.Degraded)
	}
	if !statusNextActionsContain(result, "harness evidence submit --kind ci --input <json>") {
		t.Fatalf("expected manual CI fallback guidance, got %#v", result.NextAction)
	}
}

func TestStatusRemoteHandoffNextActionsExplainNonReadyRemoteFacts(t *testing.T) {
	tests := []struct {
		name           string
		mergeState     string
		checks         string
		wantCue        string
		wantAssessment string
	}{
		{
			name:           "pending checks",
			mergeState:     `"CLEAN"`,
			checks:         `[{"name":"Go Test","bucket":"pending","state":"IN_PROGRESS"}]`,
			wantCue:        "Remote PR checks are still pending",
			wantAssessment: "wait_for_remote",
		},
		{
			name:           "failed checks",
			mergeState:     `"CLEAN"`,
			checks:         `[{"name":"Go Test","bucket":"fail","state":"FAILURE"}]`,
			wantCue:        "Remote PR checks are failing",
			wantAssessment: "repair_remote",
		},
		{
			name:           "stale sync",
			mergeState:     `"BEHIND"`,
			checks:         `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`,
			wantCue:        "Remote PR merge state is stale",
			wantAssessment: "repair_remote",
		},
		{
			name:           "conflicted sync",
			mergeState:     `"DIRTY"`,
			checks:         `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`,
			wantCue:        "Remote PR merge state is conflicted",
			wantAssessment: "repair_remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
				return completeAllSteps(content, true)
			})
			writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
			if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
				t.Fatalf("publish evidence: %#v", result)
			}

			result := status.Service{
				Workdir:       root,
				ObserveRemote: true,
				RunCommand:    fakeStatusRemoteCommands(tt.mergeState, tt.checks),
			}.Snapshot()
			remoteEvidence := requireRemoteEvidence(t, result)
			if remoteEvidence.Assessment != tt.wantAssessment {
				t.Fatalf("expected remote assessment %q, got %#v", tt.wantAssessment, remoteEvidence)
			}
			found := false
			for _, action := range result.NextAction {
				if strings.Contains(action.Description, tt.wantCue) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected next action containing %q, got %#v", tt.wantCue, result.NextAction)
			}
			if statusNextActionsContain(result, "harness evidence refresh") {
				t.Fatalf("remote assessment %s must not suggest immediate evidence refresh, got %#v", tt.wantAssessment, result.NextAction)
			}
		})
	}
}

func TestStatusDraftRemoteAssessmentDoesNotSuggestImmediateRefresh(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommandsWithPRState(`"OPEN"`, true, `"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Snapshot()
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Assessment != "wait_for_remote" || remoteEvidence.PR == nil || !remoteEvidence.PR.Draft {
		t.Fatalf("expected draft PR wait assessment, got %#v", remoteEvidence)
	}
	if statusNextActionsContain(result, "harness evidence refresh") {
		t.Fatalf("draft PR must not suggest immediate evidence refresh, got %#v", result.NextAction)
	}
}

func TestStatusRemoteAssessmentMatchesRecordedEvidence(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{Workdir: root}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommands(`"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Snapshot()
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Assessment != "matches_recorded" {
		t.Fatalf("expected matching recorded and remote evidence assessment, got %#v", remoteEvidence)
	}
}

func TestStatusRemoteAssessmentHandlesClosedPR(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommandsWithPRState(`"CLOSED"`, false, `"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Snapshot()
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Assessment != "manual_evidence_required" {
		t.Fatalf("closed PR without ready recorded evidence should require manual handoff repair, got %#v", remoteEvidence)
	}
	found := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Recorded PR is no longer open") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected closed PR guidance, got %#v", result.NextAction)
	}
	if statusNextActionsContain(result, "harness evidence refresh") {
		t.Fatalf("closed PR must not suggest evidence refresh, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "harness evidence refresh") {
			t.Fatalf("closed PR must not mention evidence refresh guidance, got %#v", result.NextAction)
		}
	}
}

func TestStatusRemoteClosedPRDoesNotMentionRefreshForPendingOrStaleRemoteFacts(t *testing.T) {
	tests := []struct {
		name       string
		mergeState string
		checks     string
	}{
		{
			name:       "pending checks",
			mergeState: `"CLEAN"`,
			checks:     `[{"name":"Go Test","bucket":"pending","state":"IN_PROGRESS"}]`,
		},
		{
			name:       "stale sync",
			mergeState: `"BEHIND"`,
			checks:     `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
				return completeAllSteps(content, true)
			})
			writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
			if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
				t.Fatalf("publish evidence: %#v", result)
			}

			result := status.Service{
				Workdir:       root,
				ObserveRemote: true,
				RunCommand:    fakeStatusRemoteCommandsWithPRState(`"CLOSED"`, false, tt.mergeState, tt.checks),
			}.Snapshot()
			remoteEvidence := requireRemoteEvidence(t, result)
			if remoteEvidence.Assessment != "manual_evidence_required" {
				t.Fatalf("closed PR should require manual handoff repair, got %#v", remoteEvidence)
			}
			for _, action := range result.NextAction {
				if strings.Contains(action.Description, "harness evidence refresh") {
					t.Fatalf("closed PR must not mention evidence refresh guidance, got %#v", result.NextAction)
				}
			}
		})
	}
}

func TestStatusRemoteAssessmentInvalidatesReadyClosedPR(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	svc := evidence.Service{Workdir: root}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommandsWithPRState(`"CLOSED"`, false, `"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("remote facts must not regress evidence-driven node, got %#v", result.State)
	}
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.Assessment != "candidate_invalidated" {
		t.Fatalf("closed PR should invalidate recorded merge-ready candidate, got %#v", remoteEvidence)
	}
	if statusNextActionsContain(result, "harness land --pr https://github.com/catu-ai/easyharness/pull/13 [--commit <sha>]") {
		t.Fatalf("closed PR must not suggest land guidance, got %#v", result.NextAction)
	}
}

func TestStatusSuggestsRefreshWhenCleanRemoteCanReplaceNonReadyEvidence(t *testing.T) {
	tests := []struct {
		name      string
		ciInput   string
		syncInput string
	}{
		{
			name:      "failed CI evidence",
			ciInput:   `{"status":"failed","provider":"github-actions","url":"https://ci.example/run"}`,
			syncInput: `{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`,
		},
		{
			name:      "conflicted sync evidence",
			ciInput:   `{"status":"success","provider":"github-actions","url":"https://ci.example/run"}`,
			syncInput: `{"status":"conflicted","base_ref":"main","head_ref":"codex/test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
				return completeAllSteps(content, true)
			})
			writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

			svc := evidence.Service{
				Workdir: root,
				Now: func() time.Time {
					return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
				},
			}
			if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
				t.Fatalf("publish evidence: %#v", result)
			}
			if result := svc.Submit("ci", []byte(tt.ciInput)); !result.OK {
				t.Fatalf("ci evidence: %#v", result)
			}
			if result := svc.Submit("sync", []byte(tt.syncInput)); !result.OK {
				t.Fatalf("sync evidence: %#v", result)
			}

			result := status.Service{
				Workdir:       root,
				ObserveRemote: true,
				RunCommand:    fakeStatusRemoteCommands(`"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
			}.Snapshot()
			if result.State.CurrentNode != "execution/finalize/publish" {
				t.Fatalf("clean live facts must not advance stale local evidence, got %#v", result.State)
			}
			if !statusNextActionsContain(result, "harness evidence refresh") {
				t.Fatalf("expected evidence refresh guidance when clean remote facts can replace non-ready evidence, got %#v", result.NextAction)
			}
		})
	}
}

func TestStatusAwaitMergeIncludesRemoteHandoffWarningsWithoutRegressingNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand:    fakeStatusRemoteCommands(`"CLEAN"`, `[{"name":"Go Test","bucket":"fail","state":"FAILURE"}]`),
	}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("live remote facts must not regress evidence-driven await_merge node, got %#v", result.State)
	}
	remoteEvidence := requireRemoteEvidence(t, result)
	if remoteEvidence.CI == nil || remoteEvidence.CI.Status != "failed" || remoteEvidence.Assessment != "candidate_invalidated" {
		t.Fatalf("expected invalidating failed live remote CI facts, got %#v", remoteEvidence)
	}
	foundRemoteGuidance := false
	foundLandGuidance := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Remote PR checks are failing") {
			foundRemoteGuidance = true
		}
		if action.Command != nil && strings.Contains(*action.Command, "harness land --pr https://github.com/catu-ai/easyharness/pull/13") {
			foundLandGuidance = true
		}
	}
	if !foundRemoteGuidance {
		t.Fatalf("expected await_merge next actions to include remote failure guidance, got %#v", result.NextAction)
	}
	if foundLandGuidance {
		t.Fatalf("invalidating remote facts must suppress ordinary land guidance, got %#v", result.NextAction)
	}
}

func TestStatusRemoteHandoffDoesNotGuessPRWhenPublishEvidenceMissing(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	called := false
	result := status.Service{
		Workdir:       root,
		ObserveRemote: true,
		RunCommand: func(name string, args ...string) remote.CommandResult {
			called = true
			return remote.CommandResult{}
		},
	}.Snapshot()
	if called {
		t.Fatal("status should not call gh without recorded publish PR evidence")
	}
	if result.Facts != nil && result.Facts.Evidence != nil && result.Facts.Evidence.Remote != nil {
		t.Fatalf("did not expect remote facts without recorded PR evidence, got %#v", result.Facts.Evidence.Remote)
	}
}

func TestStatusArchivedPlanKeepsEvidenceRefreshForNonReadyRecordedPR(t *testing.T) {
	tests := []struct {
		name            string
		ciInput         string
		syncInput       string
		wantStatus      string
		fallbackCommand string
	}{
		{
			name:            "pending CI",
			ciInput:         `{"status":"pending","provider":"github-actions","url":"https://ci.example/run"}`,
			syncInput:       `{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`,
			wantStatus:      "pending",
			fallbackCommand: "harness evidence submit --kind ci --input <json>",
		},
		{
			name:            "stale sync",
			ciInput:         `{"status":"success","provider":"github-actions","url":"https://ci.example/run"}`,
			syncInput:       `{"status":"stale","base_ref":"main","head_ref":"codex/test"}`,
			wantStatus:      "stale",
			fallbackCommand: "harness evidence submit --kind sync --input <json>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
				return completeAllSteps(content, true)
			})
			writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

			svc := evidence.Service{Workdir: root}
			if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
				t.Fatalf("publish evidence: %#v", result)
			}
			if result := svc.Submit("ci", []byte(tt.ciInput)); !result.OK {
				t.Fatalf("ci evidence: %#v", result)
			}
			if result := svc.Submit("sync", []byte(tt.syncInput)); !result.OK {
				t.Fatalf("sync evidence: %#v", result)
			}

			result := status.Service{Workdir: root}.Snapshot()
			if result.State.CurrentNode != "execution/finalize/publish" {
				t.Fatalf("unexpected node: %#v", result.State)
			}
			if !statusNextActionsContain(result, "harness evidence refresh") {
				t.Fatalf("expected evidence refresh guidance for %s handoff, got %#v", tt.wantStatus, result.NextAction)
			}
			if !statusNextActionsContain(result, tt.fallbackCommand) {
				t.Fatalf("expected manual fallback command for %s handoff, got %#v", tt.wantStatus, result.NextAction)
			}
		})
	}
}

func TestStatusArchivedPlanReadyForAwaitMerge(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if recordedPublishURL(result) != "https://github.com/catu-ai/easyharness/pull/13" || recordedCIStatus(result) != "not_applied" || recordedSyncStatus(result) != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PlanPath == "" {
		t.Fatalf("expected plan artifact in status, got %#v", result.Artifacts)
	}
}

func TestStatusPostArchiveProductCommitCannotReachAwaitMerge(t *testing.T) {
	root := t.TempDir()
	archivedRelPath := "docs/plans/archived/2026-03-18-status-plan.md"
	writePlan(t, root, archivedRelPath, func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, archivedRelPath)
	prepareReviewedArchivedStatusCandidate(t, root, archivedRelPath)
	commitStatusCandidate(t, root, "archive reviewed plan")
	if err := os.WriteFile(filepath.Join(root, "product.go"), []byte("package product\n\nconst Unreviewed = true\n"), 0o644); err != nil {
		t.Fatalf("write post-archive product change: %v", err)
	}
	commitStatusCandidate(t, root, "unreviewed post-archive product change")

	svc := evidence.Service{Workdir: root}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unreviewed post-archive change must stay in publish, got %#v", result.State)
	}
	if len(result.Blockers) != 1 || result.Blockers[0].Path != "review.coverage" || !strings.Contains(result.Blockers[0].Message, "product.go") {
		t.Fatalf("expected reviewed-head coverage blocker naming product.go, got %#v", result.Blockers)
	}
	if !statusNextActionsContainDescription(result, "harness reopen --mode finalize-fix") {
		t.Fatalf("expected reopen-and-review guidance, got %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeFromEvidenceArtifacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("expected evidence artifacts to reach await_merge, got %#v", result.State)
	}
	if recordedPublishURL(result) != "https://github.com/catu-ai/easyharness/pull/13" || recordedCIStatus(result) != "success" || recordedSyncStatus(result) != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PlanPath == "" {
		t.Fatalf("expected plan artifact in status, got %#v", result.Artifacts)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "latest_publish", "latest_ci", "latest_evidence")
}

func TestStatusArchivedPlanIgnoresOlderRevisionEvidenceArtifacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
	})

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 2,
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected older revision evidence to keep publish state, got %#v", result.State)
	}
	if recordedPublishURL(result) != "" || recordedCIStatus(result) != "" || recordedSyncStatus(result) != "" {
		t.Fatalf("expected older revision evidence to stay hidden, got %#v", result.Facts)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeWithSyncNotApplied(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no meaningful merge-base freshness signal in this test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if recordedCIStatus(result) != "success" || recordedSyncStatus(result) != "not_applied" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeWhenCIAndSyncAreBothNotApplied(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	prepareReviewedArchivedStatusCandidate(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no meaningful merge-base freshness signal in this test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if recordedCIStatus(result) != "not_applied" || recordedSyncStatus(result) != "not_applied" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusArchivedPlanStaysInPublishFromEvidenceArtifactsWhenDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
	})

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"failed","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty evidence artifacts to stay in publish, got %#v", result.State)
	}
	if recordedCIStatus(result) != "failed" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundFixCI := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Fix the CI failures") {
			foundFixCI = true
			break
		}
	}
	if !foundFixCI {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "latest_publish", "latest_ci", "latest_evidence")
}

func TestStatusArchivedPlanStaysInPublishWhenEvidenceIsDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"pending","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty evidence to stay in publish, got %#v", result.State)
	}
	if recordedCIStatus(result) != "pending" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundPendingCI := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Wait for the relevant post-archive CI") {
			foundPendingCI = true
			break
		}
	}
	if !foundPendingCI {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanStaysInPublishWhenSyncIsDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"conflicted","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty sync evidence to stay in publish, got %#v", result.State)
	}
	if recordedSyncStatus(result) != "conflicted" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundResolveConflicts := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Resolve merge conflicts") {
			foundResolveConflicts = true
			break
		}
	}
	if !foundResolveConflicts {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusLandNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"land": map[string]any{
			"pr_url":    "https://github.com/catu-ai/easyharness/pull/99",
			"commit":    "abc123",
			"landed_at": "2026-03-18T12:00:00Z",
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "land" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if !strings.Contains(result.Summary, "required post-merge bookkeeping is still in progress") {
		t.Fatalf("expected land summary to mention required bookkeeping, got %#v", result)
	}
	if result.Facts == nil || result.Facts.LandPRURL == "" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness land complete" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "required post-merge bookkeeping") {
		t.Fatalf("expected bookkeeping guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "final PR comment") {
		t.Fatalf("expected final PR comment guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "follow-up references") {
		t.Fatalf("expected linked issue follow-up guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "only after the required PR and issue bookkeeping is done") {
		t.Fatalf("expected land complete gate guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "required post-merge bookkeeping completion") {
		t.Fatalf("expected land complete action to mention required bookkeeping completion, got %#v", result.NextAction)
	}
}

func TestStatusIdleNodeAfterLand(t *testing.T) {
	root := t.TempDir()
	writeCurrentPlanPayload(t, root, map[string]any{
		"last_landed_plan_path": "docs/plans/archived/2026-03-18-status-plan.md",
		"last_landed_at":        "2026-03-19T12:00:00Z",
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if result.State.CurrentNode != "idle" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Artifacts == nil || result.Artifacts.PlanPath != "docs/plans/archived/2026-03-18-status-plan.md" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusIdleIgnoresEscapedLastLandedPlanPath(t *testing.T) {
	root := t.TempDir()
	writeCurrentPlanPayload(t, root, map[string]any{
		"last_landed_plan_path": "../sibling/docs/plans/archived/2026-03-18-status-plan.md",
		"last_landed_at":        "2026-03-19T12:00:00Z",
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if result.State.CurrentNode != "idle" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Artifacts != nil {
		t.Fatalf("expected escaped landed pointer to be ignored, got %#v", result.Artifacts)
	}
}

func TestStatusIdleSurfacesNonBlockingBootstrapReminderWhenManagedAssetsAreStale(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	staleManagedInstructions(t, root)
	staleManagedSkill(t, root, "harness-discovery")

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if result.State.CurrentNode != "idle" {
		t.Fatalf("expected idle node, got %#v", result.State)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected idle bootstrap reminder warning, got %#v", result)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "non-blocking reminder") {
		t.Fatalf("expected non-blocking reminder wording, got %#v", result.Warnings)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness repo init --dry-run" {
		t.Fatalf("expected optional bootstrap refresh guidance first, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "Optionally inspect") {
		t.Fatalf("expected optional reminder phrasing, got %#v", result.NextAction)
	}
}

func TestStatusIdleSurfacesReminderWhenManagedInstructionsAreStale(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	staleManagedInstructions(t, root)

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "AGENTS.md managed block") {
		t.Fatalf("expected stale instructions reminder, got %#v", result.Warnings)
	}
}

func TestStatusIdleSurfacesReminderWhenManagedSkillsAreStale(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	staleManagedSkill(t, root, "harness-discovery")

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "managed skill package") {
		t.Fatalf("expected stale managed-skill reminder, got %#v", result.Warnings)
	}
}

func TestStatusIdleSkipsBootstrapReminderWhenManagedAssetsAreFresh(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected fresh idle bootstrap state to stay quiet, got %#v", result.Warnings)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command != nil {
		t.Fatalf("expected ordinary idle guidance first, got %#v", result.NextAction)
	}
}

func TestStatusActivePlanDoesNotSurfaceIdleBootstrapReminder(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	staleManagedInstructions(t, root)

	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected plan result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("expected plan node, got %#v", result.State)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "bootstrap assets for Codex") {
			t.Fatalf("did not expect idle-only bootstrap reminder during active work, got %#v", result.Warnings)
		}
	}
}

func TestStatusActiveExecutionDoesNotSurfaceIdleBootstrapReminder(t *testing.T) {
	root := t.TempDir()
	svc := install.Service{Workdir: root}
	if result := svc.Init(install.Options{}); !result.OK {
		t.Fatalf("init failed: %#v", result)
	}

	staleManagedInstructions(t, root)

	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Snapshot()
	if !result.OK {
		t.Fatalf("expected execution result, got %#v", result)
	}
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected execution node, got %#v", result.State)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "bootstrap assets for Codex") {
			t.Fatalf("did not expect idle-only bootstrap reminder during execution, got %#v", result.Warnings)
		}
	}
}

func TestStatusReopenedFinalizeFixNeedsReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "finalize-fix",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusReopenedNewStepPendingPromptsForNewStep(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if !strings.Contains(result.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusReopenedNewStepContinuesOnceStepExists(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeAllSteps(content, true)
		return appendThirdStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/step-3/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != "Step 3: Follow-up reopened work" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusConsumedReopenedNewStepDoesNotForceAnotherStepAfterLaterFinding(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeAllSteps(content, true)
		content = appendThirdStep(content)
		content = replaceOnce(content, "- Done: [ ]", "- Done: [x]")
		content = replaceOnce(content, "PENDING_STEP_EXECUTION", "Done.")
		content = replaceOnce(content, "PENDING_STEP_REVIEW", "Reviewed.")
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Snapshot()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("expected finalize fix node, got %#v", result.State)
	}
	if strings.Contains(result.Summary, "needs a new unfinished step") {
		t.Fatalf("expected consumed new-step reopen mode to stop forcing another step, got %q", result.Summary)
	}
	if len(result.NextAction) == 0 || strings.Contains(result.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected finalize repair guidance instead of another new-step demand, got %#v", result.NextAction)
	}
	if result.Facts != nil && result.Facts.ReopenMode == "new-step" {
		t.Fatalf("expected consumed reopen mode to stop surfacing raw new-step guidance, got %#v", result.Facts)
	}
}

func writePlan(t *testing.T, root, relPath string, mutate func(string) string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Status Plan",
		Timestamp:  time.Date(2026, 3, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	content := mutate(rendered)
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return path
}

func approvePlanContent(content, approvedAt string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "approved_at:") {
			lines[i] = "approved_at: " + approvedAt
			return strings.Join(lines, "\n")
		}
		if strings.HasPrefix(line, "created_at:") {
			lines = append(lines[:i+1], append([]string{"approved_at: " + approvedAt}, lines[i+1:]...)...)
			return strings.Join(lines, "\n")
		}
	}
	return content
}

func writeCurrentPlan(t *testing.T, root, relPath string) {
	t.Helper()
	writeCurrentPlanPayload(t, root, map[string]any{"plan_path": relPath})
}

func staleManagedInstructions(t *testing.T, root string) {
	t.Helper()
	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	staleAgents := strings.Replace(string(agentsData), `<!-- easyharness:begin version="`, `<!-- easyharness:begin version="stale-`, 1)
	if err := os.WriteFile(agentsPath, []byte(staleAgents), 0o644); err != nil {
		t.Fatalf("write stale AGENTS.md: %v", err)
	}
}

func staleManagedSkill(t *testing.T, root, skillName string) {
	t.Helper()
	skillPath := filepath.Join(root, ".agents/skills", skillName, "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill %s: %v", skillName, err)
	}
	staleSkill := strings.Replace(string(skillData), "easyharness-version:", "easyharness-version: stale-", 1)
	if err := os.WriteFile(skillPath, []byte(staleSkill), 0o644); err != nil {
		t.Fatalf("write stale skill %s: %v", skillName, err)
	}
}

func writeCurrentPlanPayload(t *testing.T, root string, payloadMap map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		t.Fatalf("marshal current-plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "current-plan.json"), payload, 0o644); err != nil {
		t.Fatalf("write current-plan: %v", err)
	}
}

func writeState(t *testing.T, root, planStem string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

func assertStateJSONLacksKeys(t *testing.T, root, planStem string, keys ...string) {
	t.Helper()
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse state json: %v", err)
	}
	requiredAbsent := []string{
		"current_node",
		"plan_path",
		"plan_stem",
		"latest_evidence",
		"latest_ci",
		"sync",
		"latest_publish",
	}
	keys = append(requiredAbsent, keys...)
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected state.json to omit %q, got %#v", key, payload)
		}
	}
}

func statusNextActionsContain(result status.Result, command string) bool {
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == command {
			return true
		}
	}
	return false
}

func statusNextActionsContainDescription(result status.Result, snippet string) bool {
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, snippet) {
			return true
		}
	}
	return false
}

func recordedPublishURL(result status.Result) string {
	if result.Facts == nil || result.Facts.Evidence == nil || result.Facts.Evidence.Recorded == nil || result.Facts.Evidence.Recorded.Publish == nil {
		return ""
	}
	return result.Facts.Evidence.Recorded.Publish.PRURL
}

func recordedCIStatus(result status.Result) string {
	if result.Facts == nil || result.Facts.Evidence == nil || result.Facts.Evidence.Recorded == nil || result.Facts.Evidence.Recorded.CI == nil {
		return ""
	}
	return result.Facts.Evidence.Recorded.CI.Status
}

func recordedSyncStatus(result status.Result) string {
	if result.Facts == nil || result.Facts.Evidence == nil || result.Facts.Evidence.Recorded == nil || result.Facts.Evidence.Recorded.Sync == nil {
		return ""
	}
	return result.Facts.Evidence.Recorded.Sync.Status
}

func requireRemoteEvidence(t *testing.T, result status.Result) *contracts.StatusRemoteEvidence {
	t.Helper()
	if result.Facts == nil || result.Facts.Evidence == nil || result.Facts.Evidence.Remote == nil {
		t.Fatalf("expected remote evidence facts, got %#v", result.Facts)
	}
	return result.Facts.Evidence.Remote
}

func fakeStatusRemoteCommands(mergeStateJSON, checksJSON string) remote.CommandRunner {
	return fakeStatusRemoteCommandsWithPRState(`"OPEN"`, false, mergeStateJSON, checksJSON)
}

func fakeStatusRemoteCommandsWithPRState(prStateJSON string, draft bool, mergeStateJSON, checksJSON string) remote.CommandRunner {
	return func(name string, args ...string) remote.CommandResult {
		if name != "gh" {
			return remote.CommandResult{}
		}
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			return remote.CommandResult{Stdout: `{
				"url":"https://github.com/catu-ai/easyharness/pull/13",
				"number":13,
				"state":` + prStateJSON + `,
				"isDraft":` + fmt.Sprintf("%t", draft) + `,
				"mergeStateStatus":` + mergeStateJSON + `,
				"mergeable":"MERGEABLE",
				"reviewDecision":"APPROVED",
				"headRefName":"codex/test",
				"headRefOid":"abc123",
				"baseRefName":"main"
			}`}
		}
		if len(args) >= 3 && args[0] == "pr" && args[1] == "checks" {
			return remote.CommandResult{Stdout: checksJSON}
		}
		return remote.CommandResult{}
	}
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func reviewIntPtr(value int) *int {
	return &value
}

func initCommittedGitCandidate(t *testing.T, root string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".local/\n"), 0o644); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Status Test"},
		{"config", "user.email", "status-test@example.com"},
		{"add", "."},
		{"commit", "-qm", "candidate"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
		}
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve committed candidate: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func completePassingFinalizeReview(t *testing.T, root string) {
	t.Helper()
	svc := review.Service{Workdir: root}
	start := svc.Start(review.StartOptions{})
	if !start.OK {
		t.Fatalf("start finalize review: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "reviewer-integrated", mustJSONBytes(t, review.SubmissionInput{
		Summary:  "The complete candidate is ready for archive.",
		Findings: nil,
	}))
	if !submit.OK || submit.Review == nil || submit.Review.Decision != "pass" {
		t.Fatalf("submit finalize review: %#v", submit)
	}
}

func prepareReviewedArchivedStatusCandidate(t *testing.T, root, archivedRelPath string) {
	t.Helper()
	archivedPath := filepath.Join(root, filepath.FromSlash(archivedRelPath))
	content, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("read archived status candidate: %v", err)
	}
	activeRelPath := strings.Replace(archivedRelPath, "/archived/", "/active/", 1)
	activePath := filepath.Join(root, filepath.FromSlash(activeRelPath))
	if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
		t.Fatalf("mkdir active status plan: %v", err)
	}
	if err := os.WriteFile(activePath, content, 0o644); err != nil {
		t.Fatalf("write active status candidate: %v", err)
	}
	if err := os.Remove(archivedPath); err != nil {
		t.Fatalf("remove pre-review archived status plan: %v", err)
	}
	writeCurrentPlan(t, root, activeRelPath)
	planStem := strings.TrimSuffix(filepath.Base(archivedRelPath), filepath.Ext(archivedRelPath))
	writeState(t, root, planStem, map[string]any{
		"revision":             1,
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	initCommittedGitCandidate(t, root)
	completePassingFinalizeReview(t, root)
	if err := os.MkdirAll(filepath.Dir(archivedPath), 0o755); err != nil {
		t.Fatalf("mkdir archived status plan: %v", err)
	}
	if err := os.Rename(activePath, archivedPath); err != nil {
		t.Fatalf("mechanically archive reviewed status plan: %v", err)
	}
	writeCurrentPlan(t, root, archivedRelPath)
}

func commitStatusCandidate(t *testing.T, root, message string) {
	t.Helper()
	for _, args := range [][]string{{"add", "--all"}, {"commit", "-qm", message}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
		}
	}
}

func writeReviewManifest(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func writeReviewAggregate(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir aggregate dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal aggregate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aggregate.json"), data, 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func completeFirstStep(content string) string {
	content = replaceOnce(content, "- Done: [ ]", "- Done: [x]")
	return content
}

func completeAllSteps(content string, archiveReady bool) string {
	return completeAllStepsWithoutCloseout(content, archiveReady)
}

func completeAllStepsWithoutCloseout(content string, archiveReady bool) string {
	content = stringsReplaceAll(content, "- Done: [ ]", "- Done: [x]")
	content = stringsReplaceAll(content, "- [ ]", "- [x]")
	if archiveReady {
		content = stringsReplaceAll(content, "- Validation: PENDING_UNTIL_ARCHIVE", "- Validation: Validated the implementation.")
		content = stringsReplaceAll(content, "- Review: PENDING_UNTIL_ARCHIVE", "- Review: No blocking review findings remain.")
		content = stringsReplaceAll(content, "- Delivered: PENDING_UNTIL_ARCHIVE", "- Delivered: Delivered the planned slice.")
		content = stringsReplaceAll(content, "- Not Delivered: PENDING_UNTIL_ARCHIVE", "- Not Delivered: NONE.")
		content = stringsReplaceAll(content, "- PR: PENDING_UNTIL_ARCHIVE", "- PR: NONE")
		content = stringsReplaceAll(content, "- Ready: PENDING_UNTIL_ARCHIVE", "- Ready: The candidate is ready for archive.")
		content = stringsReplaceAll(content, "- Merge Handoff: PENDING_UNTIL_ARCHIVE", "- Merge Handoff: Commit and push the archive move before merge approval.")
	}
	return content
}

func appendThirdStep(content string) string {
	insert := `### Step 3: Follow-up reopened work

- Done: [ ]
- Outcome: Carry the reopened follow-up work as a proper third step.
- Covers: Criterion 2
- Check: Verify the reopened scope is complete.

## Validation Strategy`
	return strings.Replace(content, "## Validation Strategy", insert, 1)
}

func replaceOnce(content, old, new string) string {
	return strings.Replace(content, old, new, 1)
}

func stringsReplaceAll(content, old, new string) string {
	return strings.ReplaceAll(content, old, new)
}
