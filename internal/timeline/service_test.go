package timeline_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/timeline"
)

func TestReadLoadsCurrentPlanTimelineEvents(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-04-01-timeline-plan", &runstate.State{
		ExecutionStartedAt: "2026-04-01T10:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "lifecycle",
		Command:    "execute start",
		Summary:    "Execution started for the current active plan.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ToNode:     "execution/step-1/implement",
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected timeline read success, got %#v", result)
	}
	if result.Resource != "timeline" {
		t.Fatalf("expected timeline resource, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus recorded event, got %#v", result.Events)
	}
	if result.Events[0].Command != "plan" {
		t.Fatalf("expected leading plan event, got %#v", result.Events[0])
	}
	if result.Events[1].Command != "execute start" {
		t.Fatalf("unexpected event order: %#v", result.Events)
	}
	if result.Artifacts == nil || result.Artifacts.PlanPath != relPlanPath {
		t.Fatalf("expected plan-path artifact, got %#v", result.Artifacts)
	}
}

func TestReadReturnsEmptyTimelineWithoutCurrentPlan(t *testing.T) {
	result := timeline.Service{Workdir: t.TempDir()}.Read()
	if !result.OK {
		t.Fatalf("expected empty timeline success, got %#v", result)
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected no events, got %#v", result.Events)
	}
}

func TestReadLoadsEventsWhenStateCacheIsMissing(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "lifecycle",
		Command:    "execute start",
		Summary:    "Execution started for the current active plan.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ToNode:     "execution/step-1/implement",
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected timeline read success without state cache, got %#v", result)
	}
	if len(result.Events) != 2 || result.Events[0].Command != "plan" || result.Events[1].Command != "execute start" {
		t.Fatalf("unexpected timeline events without state cache: %#v", result.Events)
	}
	if result.Artifacts == nil || result.Artifacts.PlanPath != relPlanPath {
		t.Fatalf("expected timeline to keep the plan path artifact even when cache is missing, got %#v", result.Artifacts)
	}
}

func TestReadSynthesizesImplementBootstrapWhenExecutionStartedPredatesEventIndex(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-04-01-timeline-plan", &runstate.State{
		ExecutionStartedAt: "2026-04-01T10:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected synthesized bootstrap timeline read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected plan and implement bootstrap events, got %#v", result.Events)
	}
	if result.Events[0].Command != "plan" || result.Events[1].Command != "implement" {
		t.Fatalf("expected plan then implement bootstrap events, got %#v", result.Events)
	}
	if len(result.Events[1].Output) == 0 {
		t.Fatalf("expected raw output payload on bootstrap implement event, got %#v", result.Events[1])
	}
}

func TestReadLoadsLargeTimelineEventPayload(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	rawOutput, err := json.Marshal(map[string]string{
		"blob": strings.Repeat("x", 2*1024*1024),
	})
	if err != nil {
		t.Fatalf("marshal large output: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review submit",
		Summary:    "Recorded large review submission payload.",
		PlanPath:   relPlanPath,
		Revision:   1,
		Output:     rawOutput,
	}); err != nil {
		t.Fatalf("append large timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected large timeline payload read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus large recorded event, got %#v", result.Events)
	}
	if result.Events[1].Command != "review submit" {
		t.Fatalf("unexpected large payload event: %#v", result.Events[1])
	}
	if len(result.Events[1].Output) == 0 {
		t.Fatalf("expected output payload on large event, got %#v", result.Events[1])
	}
	var decoded map[string]string
	if err := json.Unmarshal(result.Events[1].Output, &decoded); err != nil {
		t.Fatalf("unmarshal large output payload: %v", err)
	}
	if decoded["blob"] != strings.Repeat("x", 2*1024*1024) {
		t.Fatalf("expected large payload integrity to survive round-trip, got %d bytes", len(decoded["blob"]))
	}
}

func TestReadResolvesArtifactRefFileContents(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	submissionPath := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-timeline-plan", "reviews", "review-001-full", "submissions", "correctness", "submission.json")
	if err := os.MkdirAll(filepath.Dir(submissionPath), 0o755); err != nil {
		t.Fatalf("mkdir submission dir: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte("{\"summary\":\"Timeline artifacts\"}\n"), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review start",
		Summary:    "Created review round.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ArtifactRefs: []timeline.ArtifactRef{
			{Label: "round_id", Value: "review-001-full"},
			{Label: "submission_correctness_path", Value: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions/correctness/submission.json", Path: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions/correctness/submission.json"},
		},
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected resolved artifact timeline read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus review start event, got %#v", result.Events)
	}
	refs := result.Events[1].ArtifactRefs
	if len(refs) != 2 {
		t.Fatalf("expected artifact refs to survive read, got %#v", refs)
	}
	if len(refs[0].Content) != 0 {
		t.Fatalf("expected value-only ref to remain unresolved, got %#v", refs[0])
	}
	if refs[1].ContentType != "json" {
		t.Fatalf("expected resolved submission ref content type json, got %#v", refs[1])
	}
	var decoded map[string]string
	if err := json.Unmarshal(refs[1].Content, &decoded); err != nil {
		t.Fatalf("unmarshal resolved submission content: %v", err)
	}
	if decoded["summary"] != "Timeline artifacts" {
		t.Fatalf("unexpected submission content: %#v", decoded)
	}
}

func TestReadFiltersLegacyControlPathRefsAndPayloadKeysFromTimelineUI(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	submissionPath := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-timeline-plan", "reviews", "review-001-full", "submissions", "correctness", "submission.json")
	if err := os.MkdirAll(filepath.Dir(submissionPath), 0o755); err != nil {
		t.Fatalf("mkdir submission dir: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte("{\"summary\":\"Timeline artifacts\"}\n"), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	output, err := json.Marshal(map[string]any{
		"artifacts": map[string]any{
			"local_state_path": ".local/harness/plans/2026-04-01-timeline-plan/state.json",
			"record_path":      ".local/harness/plans/2026-04-01-timeline-plan/evidence/ci/ci-001.json",
			"submissions_dir":  ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions",
			"refs": []map[string]any{
				{
					"label": "record_path",
					"path":  ".local/harness/plans/2026-04-01-timeline-plan/evidence/ci/ci-001.json",
				},
				{
					"label": "publish_record",
					"path":  ".local/harness/plans/2026-04-01-timeline-plan/evidence/publish/publish-001.json",
				},
				{
					"label": "submission_path",
					"path":  ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions/correctness/submission.json",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}
	artifacts, err := json.Marshal(map[string]any{
		"local_state_path": ".local/harness/plans/2026-04-01-timeline-plan/state.json",
		"record_path":      ".local/harness/plans/2026-04-01-timeline-plan/evidence/ci/ci-001.json",
		"submissions_dir":  ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions",
	})
	if err != nil {
		t.Fatalf("marshal artifacts: %v", err)
	}

	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review start",
		Summary:    "Created review round.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ArtifactRefs: []timeline.ArtifactRef{
			{Label: "manifest_path", Value: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/manifest.json", Path: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/manifest.json"},
			{Label: "publish_record", Value: ".local/harness/plans/2026-04-01-timeline-plan/evidence/publish/publish-001.json", Path: ".local/harness/plans/2026-04-01-timeline-plan/evidence/publish/publish-001.json"},
			{Label: "submission_correctness_path", Value: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions/correctness/submission.json", Path: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/submissions/correctness/submission.json"},
		},
		Output:    output,
		Artifacts: artifacts,
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected timeline read success, got %#v", result)
	}
	event := result.Events[1]
	if len(event.ArtifactRefs) != 1 || event.ArtifactRefs[0].Label != "submission_correctness_path" {
		t.Fatalf("expected only reviewer-owned artifact refs to remain, got %#v", event.ArtifactRefs)
	}
	if strings.Contains(string(event.Output), "local_state_path") || strings.Contains(string(event.Artifacts), "local_state_path") {
		t.Fatalf("expected control-path keys to be scrubbed from timeline payloads, got output=%s artifacts=%s", event.Output, event.Artifacts)
	}
	if strings.Contains(string(event.Output), "submissions_dir") || strings.Contains(string(event.Artifacts), "submissions_dir") {
		t.Fatalf("expected submissions_dir to be scrubbed from timeline payloads, got output=%s artifacts=%s", event.Output, event.Artifacts)
	}
	if strings.Contains(string(event.Output), "\"label\":\"record_path\"") || strings.Contains(string(event.Output), "\"label\":\"submission_path\"") {
		t.Fatalf("expected hidden legacy artifact-ref objects to be scrubbed from timeline payloads, got output=%s", event.Output)
	}
	if strings.Contains(string(event.Output), "\"label\":\"publish_record\"") {
		t.Fatalf("expected legacy evidence artifact-ref objects to be scrubbed from timeline payloads, got output=%s", event.Output)
	}
}

func writeActivePlanForTimeline(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: "Timeline Test Plan"})
	if err != nil {
		t.Fatalf("render plan template: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath
}

func stringsHasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
