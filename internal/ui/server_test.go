package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/timeline"
)

func TestNewHandlerServesStatusJSON(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		State   struct {
			CurrentNode string `json:"current_node"`
		} `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true, got %#v", payload)
	}
	if payload.Command != "status" {
		t.Fatalf("expected command=status, got %#v", payload)
	}
	if payload.State.CurrentNode == "" {
		t.Fatalf("expected current_node, got %#v", payload)
	}
}

func TestNewHandlerFallsBackToIndexForSPAPath(t *testing.T) {
	workdir := t.TempDir()
	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/review", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected HTML content type, got %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "<div id=\"app\"></div>") {
		t.Fatalf("expected embedded index, got %s", body)
	}
	if !strings.Contains(body, workdir) {
		t.Fatalf("expected injected workdir %q in embedded index, got %s", workdir, body)
	}
	if !strings.Contains(body, "repoName: \""+filepath.Base(workdir)+"\"") {
		t.Fatalf("expected injected repo name %q in embedded index, got %s", filepath.Base(workdir), body)
	}
	if !strings.Contains(body, "productName: \""+productDisplayName+"\"") {
		t.Fatalf("expected injected product name %q in embedded index, got %s", productDisplayName, body)
	}
}

func TestNewHandlerServesTimelineJSON(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-plan.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: "UI Timeline Plan"})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-01-ui-timeline-plan", &runstate.State{
		PlanPath: relPlanPath,
		PlanStem: "2026-04-01-ui-timeline-plan",
		Revision: 1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if _, _, err := timeline.AppendEvent(workdir, "2026-04-01-ui-timeline-plan", timeline.Event{
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

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Events   []struct {
			Command string `json:"command"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "timeline" {
		t.Fatalf("unexpected timeline payload: %#v", payload)
	}
	if len(payload.Events) != 2 || payload.Events[0].Command != "plan" || payload.Events[1].Command != "execute start" {
		t.Fatalf("unexpected timeline events: %#v", payload.Events)
	}
}

func TestNewHandlerServesLargeTimelinePayloadWithoutTruncation(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-large.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: "UI Timeline Large Payload"})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-01-ui-timeline-large", &runstate.State{
		PlanPath: relPlanPath,
		PlanStem: "2026-04-01-ui-timeline-large",
		Revision: 1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	blob := strings.Repeat("x", 2*1024*1024)
	rawOutput, err := json.Marshal(map[string]string{"blob": blob})
	if err != nil {
		t.Fatalf("marshal large output: %v", err)
	}
	if _, _, err := timeline.AppendEvent(workdir, "2026-04-01-ui-timeline-large", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review submit",
		Summary:    "Recorded large review submission payload.",
		PlanPath:   relPlanPath,
		Revision:   1,
		Output:     rawOutput,
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK     bool `json:"ok"`
		Events []struct {
			Command string          `json:"command"`
			Output  json.RawMessage `json:"output"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || len(payload.Events) != 2 || payload.Events[1].Command != "review submit" {
		t.Fatalf("unexpected timeline payload: %#v", payload)
	}
	var output struct {
		Blob string `json:"blob"`
	}
	if err := json.Unmarshal(payload.Events[1].Output, &output); err != nil {
		t.Fatalf("unmarshal event output: %v", err)
	}
	if output.Blob != blob {
		t.Fatalf("expected large payload to survive api serialization, got %d bytes", len(output.Blob))
	}
}

func TestNewHandlerServesResolvedArtifactFileContents(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-artifacts.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: "UI Timeline Artifact Tabs"})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	manifestRelPath := ".local/harness/plans/2026-04-01-ui-timeline-artifacts/reviews/review-001-full/manifest.json"
	manifestPath := filepath.Join(workdir, filepath.FromSlash(manifestRelPath))
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{\"review_title\":\"Artifact tabs\"}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, _, err := timeline.AppendEvent(workdir, "2026-04-01-ui-timeline-artifacts", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review start",
		Summary:    "Created review round.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ArtifactRefs: []timeline.ArtifactRef{
			{Label: "manifest_path", Value: manifestRelPath, Path: manifestRelPath},
		},
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK     bool `json:"ok"`
		Events []struct {
			Command      string `json:"command"`
			ArtifactRefs []struct {
				Label       string          `json:"label"`
				ContentType string          `json:"content_type"`
				Content     json.RawMessage `json:"content"`
			} `json:"artifact_refs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || len(payload.Events) != 2 || payload.Events[1].Command != "review start" {
		t.Fatalf("unexpected timeline payload: %#v", payload)
	}
	if len(payload.Events[1].ArtifactRefs) != 1 {
		t.Fatalf("expected one resolved artifact ref, got %#v", payload.Events[1].ArtifactRefs)
	}
	if payload.Events[1].ArtifactRefs[0].ContentType != "json" {
		t.Fatalf("expected json content type, got %#v", payload.Events[1].ArtifactRefs[0])
	}
	var content map[string]string
	if err := json.Unmarshal(payload.Events[1].ArtifactRefs[0].Content, &content); err != nil {
		t.Fatalf("unmarshal resolved artifact content: %v", err)
	}
	if content["review_title"] != "Artifact tabs" {
		t.Fatalf("unexpected resolved artifact content: %#v", content)
	}
}

func TestNewHandlerReturnsNotFoundForAPINamespaceRoot(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestNewHandlerReturnsServiceUnavailableForStatusReadFailure(t *testing.T) {
	workdir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(workdir, []byte("blocking file"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d\n%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		OK     bool `json:"ok"`
		Errors []struct {
			Path string `json:"path"`
		} `json:"errors"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if payload.OK {
		t.Fatalf("expected ok=false, got %#v", payload)
	}
	if payload.Summary == "" {
		t.Fatalf("expected failure summary, got %#v", payload)
	}
	if len(payload.Errors) == 0 {
		t.Fatalf("expected status errors, got %#v", payload)
	}
}

func TestServerRunPrintsListeningURLWithoutOpeningBrowser(t *testing.T) {
	logs := &lockedBuffer{}
	server := Server{
		Workdir:     t.TempDir(),
		Host:        "127.0.0.1",
		Port:        0,
		Stdout:      logs,
		Stderr:      io.Discard,
		OpenBrowser: false,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	var listeningURL string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		output := logs.String()
		if strings.Contains(output, "Harness UI listening at http://") {
			listeningURL = strings.TrimSpace(strings.TrimPrefix(output, "Harness UI listening at "))
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if listeningURL == "" {
		t.Fatalf("expected listening URL in stdout, got %q", logs.String())
	}

	response, err := http.Get(listeningURL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}
