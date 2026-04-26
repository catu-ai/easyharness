package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/dashboard"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/timeline"
	"github.com/catu-ai/easyharness/internal/watchlist"
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

func TestNewHandlerStatusDoesNotTouchWatchlist(t *testing.T) {
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)
	home := t.TempDir()
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if _, err := os.Stat(filepath.Join(home, "watchlist.json")); !os.IsNotExist(err) {
		t.Fatalf("expected UI status request to avoid watchlist writes, err=%v", err)
	}
}

func TestNewHandlerStatusForActivePlanDoesNotMutateWorkflowOrWatchlist(t *testing.T) {
	workdir := filepath.Join(t.TempDir(), "workspace-active-status")
	seedGitWorkspace(t, workdir)
	home := t.TempDir()
	t.Setenv("EASYHARNESS_HOME", home)
	relPlanPath, planStem := writeUIActivePlan(t, workdir, "UI Active Status")
	currentPlanPath, statePath, lockPath := seedUIActiveState(t, workdir, relPlanPath, planStem)
	before := snapshotStateFiles(t, currentPlanPath, statePath)

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OK    bool `json:"ok"`
		State struct {
			CurrentNode string `json:"current_node"`
		} `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
	assertStateFilesUnchanged(t, before)
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected UI status request to avoid creating state lock, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "watchlist.json")); !os.IsNotExist(err) {
		t.Fatalf("expected UI status request to avoid watchlist writes, err=%v", err)
	}
}

func TestUIReadSurfacesDoNotTouchWatchlist(t *testing.T) {
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)
	home := t.TempDir()
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	for _, path := range []string{"/api/status", "/api/plan", "/api/review", "/api/timeline", "/"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(recorder, request)
		if recorder.Code >= http.StatusInternalServerError {
			t.Fatalf("expected %s to avoid server error, got %d", path, recorder.Code)
		}
		if _, err := os.Stat(filepath.Join(home, "watchlist.json")); !os.IsNotExist(err) {
			t.Fatalf("expected %s to avoid watchlist writes, err=%v", path, err)
		}
	}
}

func TestNewHandlerServesDashboardJSON(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{{
		WorkspacePath: workspace,
		WatchedAt:     "2026-04-22T09:00:00Z",
		LastSeenAt:    "2026-04-22T12:00:00Z",
	}})
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var payload struct {
		OK       bool                 `json:"ok"`
		Resource string               `json:"resource"`
		Groups   []dashboardTestGroup `json:"groups"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "dashboard" {
		t.Fatalf("unexpected dashboard payload: %#v", payload)
	}
	entry := dashboardWorkspaceInGroup(t, payload.Groups, "idle", workspace)
	if entry.DashboardState != "idle" || entry.CurrentNode != "idle" {
		t.Fatalf("unexpected dashboard entry: %#v", entry)
	}
}

func TestNewHandlerRejectsDashboardNonGET(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/dashboard", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestNewHandlerDashboardDoesNotRewriteWatchlist(t *testing.T) {
	home := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing")
	writeWatchlist(t, home, []watchlist.Workspace{{
		WorkspacePath: missing,
		WatchedAt:     "2026-04-22T09:00:00Z",
		LastSeenAt:    "2026-04-22T12:00:00Z",
	}})
	t.Setenv("EASYHARNESS_HOME", home)
	watchlistPath := filepath.Join(home, "watchlist.json")
	fixedTime := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	if err := os.Chtimes(watchlistPath, fixedTime, fixedTime); err != nil {
		t.Fatalf("set watchlist timestamp: %v", err)
	}
	before := snapshotFile(t, watchlistPath)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OK       bool                 `json:"ok"`
		Resource string               `json:"resource"`
		Groups   []dashboardTestGroup `json:"groups"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "dashboard" {
		t.Fatalf("unexpected dashboard payload: %#v", payload)
	}
	entry := dashboardWorkspaceInGroup(t, payload.Groups, "missing", missing)
	if entry.DashboardState != "missing" || entry.CurrentNode != "" {
		t.Fatalf("expected missing degraded entry, got %#v", entry)
	}
	assertFileUnchanged(t, watchlistPath, before)
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

func TestNewHandlerRedirectsRootToDashboard(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", location)
	}
}

func TestNewHandlerRedirectsWorkspaceRouteToStatus(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/workspace/wk_abc123", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/workspace/wk_abc123/status" {
		t.Fatalf("expected redirect to /workspace/<key>/status, got %q", location)
	}
}

func TestNewHandlerServesWorkspaceLookupJSON(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{{
		WorkspacePath: workspace,
		WatchedAt:     "2026-04-22T09:00:00Z",
		LastSeenAt:    "2026-04-22T12:00:00Z",
	}})
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace), nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OK        bool `json:"ok"`
		Watched   bool `json:"watched"`
		Workspace *struct {
			WorkspacePath string `json:"workspace_path"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || !payload.Watched || payload.Workspace == nil || payload.Workspace.WorkspacePath != workspace {
		t.Fatalf("unexpected workspace payload: %#v", payload)
	}
}

func TestNewHandlerServesWorkspaceStatusJSONByKey(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-status")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace)+"/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Command != "status" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
}

func TestNewHandlerWorkspaceStatusForActivePlanDoesNotMutateWorkflowOrWatchlist(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-status-active")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)
	relPlanPath, planStem := writeUIActivePlan(t, workspace, "Workspace Active Status")
	currentPlanPath, statePath, lockPath := seedUIActiveState(t, workspace, relPlanPath, planStem)
	watchlistPath := filepath.Join(home, "watchlist.json")
	fixedTime := time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(watchlistPath, fixedTime, fixedTime); err != nil {
		t.Fatalf("set watchlist timestamp: %v", err)
	}
	before := snapshotStateFiles(t, currentPlanPath, statePath)
	watchlistBefore := snapshotFile(t, watchlistPath)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace)+"/status", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OK    bool `json:"ok"`
		State struct {
			CurrentNode string `json:"current_node"`
		} `json:"state"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
	assertStateFilesUnchanged(t, before)
	assertFileUnchanged(t, watchlistPath, watchlistBefore)
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace status request to avoid creating state lock, err=%v", err)
	}
}

func TestNewHandlerServesWorkspacePlanJSONByKey(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-plan")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	relPlanPath := "docs/plans/active/2026-04-10-ui-plan.md"
	path := filepath.Join(workspace, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "Workspace Plan")
	rendered = strings.Replace(rendered, "Describe the intended outcome in one or two short paragraphs.", "Read the plan.\n", 1)
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workspace, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace)+"/plan", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Document *struct {
			Title string `json:"title"`
		} `json:"document"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "plan" || payload.Document == nil || payload.Document.Title != "Workspace Plan" {
		t.Fatalf("unexpected workspace plan payload: %#v", payload)
	}
}

func TestNewHandlerServesWorkspaceTimelineJSONByKey(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-timeline")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-plan.md"
	path := filepath.Join(workspace, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "Workspace Timeline Plan")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workspace, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workspace, "2026-04-01-ui-timeline-plan", &runstate.State{
		Revision: 1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if _, _, err := timeline.AppendEvent(workspace, "2026-04-01-ui-timeline-plan", timeline.Event{
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

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace)+"/timeline", nil)
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
		t.Fatalf("unexpected workspace timeline payload: %#v", payload)
	}
	if len(payload.Events) != 2 || payload.Events[1].Command != "execute start" {
		t.Fatalf("unexpected timeline events: %#v", payload.Events)
	}
}

func TestNewHandlerServesWorkspaceReviewJSONByKey(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-review")
	seedGitWorkspace(t, workspace)
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(workspace, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	relPlanPath := "docs/plans/active/2026-04-02-ui-review-plan.md"
	path := filepath.Join(workspace, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "Workspace Review Plan")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workspace, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workspace, "2026-04-02-ui-review-plan", &runstate.State{
		Revision: 2,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-full",
			Kind:       "full",
			Revision:   2,
			Aggregated: false,
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	reviewDir := filepath.Join(workspace, ".local", "harness", "plans", "2026-04-02-ui-review-plan", "reviews", "review-002-full")
	if err := os.MkdirAll(filepath.Join(reviewDir, "submissions"), 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	manifestPath := filepath.Join(reviewDir, "manifest.json")
	ledgerPath := filepath.Join(reviewDir, "ledger.json")
	submissionPath := filepath.Join(reviewDir, "submissions", "ux.json")
	if err := os.WriteFile(manifestPath, []byte(`{"round_id":"review-002-full","kind":"delta","anchor_sha":"abc123def","revision":2,"review_title":"Finalize review","plan_path":"`+relPlanPath+`","plan_stem":"2026-04-02-ui-review-plan","created_at":"2026-04-02T12:00:00Z","ledger_path":"`+ledgerPath+`","aggregate_path":"`+filepath.Join(reviewDir, "aggregate.json")+`","submissions_dir":"`+filepath.Join(reviewDir, "submissions")+`","dimensions":[{"name":"UX","slot":"ux","instructions":"Check the interface hierarchy.","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(ledgerPath, []byte(`{"round_id":"review-002-full","kind":"full","updated_at":"2026-04-02T12:10:00Z","slots":[{"name":"UX","slot":"ux","status":"submitted","submitted_at":"2026-04-02T12:08:00Z","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte(`{"round_id":"review-002-full","slot":"ux","dimension":"UX","submitted_at":"2026-04-02T12:08:00Z","summary":"Hierarchy is clear.","findings":[],"worklog":{"full_plan_read":true,"checked_areas":["web/src/pages.tsx"]},"coverage":{"review_kind":"delta","anchor_sha":"abc123def"}}`), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspace/"+dashboard.WorkspaceKey(workspace)+"/review", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Rounds   []struct {
			RoundID string `json:"round_id"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "review" || len(payload.Rounds) != 1 || payload.Rounds[0].RoundID != "review-002-full" {
		t.Fatalf("unexpected workspace review payload: %#v", payload)
	}
}

func TestNewHandlerWorkspaceUnwatchRemovesEntry(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-unwatch")
	seedGitWorkspace(t, workspace)
	canonicalPath, err := watchlist.CanonicalWorkspacePath(workspace)
	if err != nil {
		t.Fatalf("canonical workspace path: %v", err)
	}
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(canonicalPath, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/workspace/"+dashboard.WorkspaceKey(canonicalPath)+"/unwatch", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	after := readWatchlist(t, home)
	if len(after.Workspaces) != 0 {
		t.Fatalf("expected watched workspace to be removed, got %#v", after.Workspaces)
	}
}

func TestResolveWorkspaceActionTargetRejectsAmbiguousMatchWithoutExplicitPath(t *testing.T) {
	matches := []watchlist.Workspace{
		{WorkspacePath: "/tmp/alpha"},
		{WorkspacePath: "/tmp/beta"},
	}

	target, err := resolveWorkspaceActionTarget(matches, "")
	if !errors.Is(err, errWorkspaceActionTargetAmbiguous) {
		t.Fatalf("expected ambiguous target error, got target=%q err=%v", target, err)
	}
}

func TestResolveWorkspaceActionTargetAcceptsExplicitPathFromCollisionSet(t *testing.T) {
	matches := []watchlist.Workspace{
		{WorkspacePath: "/tmp/alpha"},
		{WorkspacePath: "/tmp/beta"},
	}

	target, err := resolveWorkspaceActionTarget(matches, "/tmp/beta")
	if err != nil {
		t.Fatalf("resolve explicit target: %v", err)
	}
	if target != "/tmp/beta" {
		t.Fatalf("expected explicit target path, got %q", target)
	}
}

func TestNewHandlerWorkspaceUnwatchUsesExplicitWorkspacePath(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(t.TempDir(), "workspace-unwatch-explicit")
	seedGitWorkspace(t, workspace)
	canonicalPath, err := watchlist.CanonicalWorkspacePath(workspace)
	if err != nil {
		t.Fatalf("canonical workspace path: %v", err)
	}
	writeWatchlist(t, home, []watchlist.Workspace{workspaceRecord(canonicalPath, "2026-04-22T12:00:00Z")})
	t.Setenv("EASYHARNESS_HOME", home)

	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	body := strings.NewReader(`{"workspace_path":"` + canonicalPath + `"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/workspace/"+dashboard.WorkspaceKey(canonicalPath)+"/unwatch", body)
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	after := readWatchlist(t, home)
	if len(after.Workspaces) != 0 {
		t.Fatalf("expected watched workspace to be removed, got %#v", after.Workspaces)
	}
}

func TestNewHandlerServesPlanJSON(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-10-ui-plan.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Plan")
	rendered = strings.Replace(rendered, "Describe the intended outcome in one or two short paragraphs.", "Read the plan.\n", 1)
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	supplementsDir := filepath.Join(workdir, "docs", "plans", "active", "supplements", "2026-04-10-ui-plan")
	if err := os.MkdirAll(supplementsDir, 0o755); err != nil {
		t.Fatalf("mkdir supplements: %v", err)
	}
	if err := os.WriteFile(filepath.Join(supplementsDir, "notes.txt"), []byte("hello plan page\n"), 0o644); err != nil {
		t.Fatalf("write supplement: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/plan", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Document *struct {
			Title    string `json:"title"`
			Markdown string `json:"markdown"`
			Headings []struct {
				Label    string `json:"label"`
				Children []struct {
					Label string `json:"label"`
				} `json:"children"`
			} `json:"headings"`
		} `json:"document"`
		Supplements *struct {
			Label    string `json:"label"`
			Children []struct {
				Label   string `json:"label"`
				Preview *struct {
					Status      string `json:"status"`
					ContentType string `json:"content_type"`
				} `json:"preview"`
			} `json:"children"`
		} `json:"supplements"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "plan" {
		t.Fatalf("unexpected plan payload: %#v", payload)
	}
	if payload.Document == nil || payload.Document.Title != "UI Plan" || strings.Contains(payload.Document.Markdown, "template_version") {
		t.Fatalf("unexpected document payload: %#v", payload.Document)
	}
	if len(payload.Document.Headings) < 2 || payload.Document.Headings[0].Label != "Goal" || payload.Document.Headings[1].Label != "Scope" {
		t.Fatalf("unexpected heading tree: %#v", payload.Document.Headings)
	}
	if payload.Supplements == nil || payload.Supplements.Label != "2026-04-10-ui-plan" || len(payload.Supplements.Children) != 1 {
		t.Fatalf("unexpected supplements payload: %#v", payload.Supplements)
	}
	if payload.Supplements.Children[0].Preview == nil || payload.Supplements.Children[0].Preview.Status != "supported" || payload.Supplements.Children[0].Preview.ContentType != "text" {
		t.Fatalf("unexpected supplement preview: %#v", payload.Supplements.Children[0])
	}
}

func TestNewHandlerServesArchivedCurrentPlanJSON(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/archived/2026-04-10-archived-ui-plan.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir archived dir: %v", err)
	}
	rendered := renderPlanFixture(t, "Archived UI Plan")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write archived plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	supplementsDir := filepath.Join(workdir, "docs", "plans", "archived", "supplements", "2026-04-10-archived-ui-plan")
	if err := os.MkdirAll(supplementsDir, 0o755); err != nil {
		t.Fatalf("mkdir archived supplements: %v", err)
	}
	if err := os.WriteFile(filepath.Join(supplementsDir, "notes.txt"), []byte("archived plan page\n"), 0o644); err != nil {
		t.Fatalf("write archived supplement: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/plan", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK       bool   `json:"ok"`
		Summary  string `json:"summary"`
		Document *struct {
			Path string `json:"path"`
		} `json:"document"`
		Artifacts *struct {
			PlanPath string `json:"plan_path"`
		} `json:"artifacts"`
		Supplements *struct {
			Label string `json:"label"`
		} `json:"supplements"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK {
		t.Fatalf("expected ok=true, got %#v", payload)
	}
	if payload.Document == nil || payload.Document.Path != relPlanPath {
		t.Fatalf("expected archived pointer document, got %#v", payload.Document)
	}
	if payload.Artifacts == nil || payload.Artifacts.PlanPath != relPlanPath {
		t.Fatalf("expected archived pointer to return artifacts, got %#v", payload.Artifacts)
	}
	if payload.Supplements == nil || payload.Supplements.Label != "2026-04-10-archived-ui-plan" {
		t.Fatalf("expected archived supplements payload, got %#v", payload.Supplements)
	}
	if !strings.Contains(payload.Summary, "Loaded the current plan package") {
		t.Fatalf("unexpected summary: %q", payload.Summary)
	}
}

func TestNewHandlerServesTimelineJSON(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-plan.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Timeline Plan")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-01-ui-timeline-plan", &runstate.State{
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

func TestNewHandlerServesReviewJSON(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-02-ui-review-plan.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Review Plan")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-02-ui-review-plan", &runstate.State{
		Revision: 2,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-full",
			Kind:       "full",
			Revision:   2,
			Aggregated: false,
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	reviewDir := filepath.Join(workdir, ".local", "harness", "plans", "2026-04-02-ui-review-plan", "reviews", "review-002-full")
	if err := os.MkdirAll(filepath.Join(reviewDir, "submissions"), 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	manifestPath := filepath.Join(reviewDir, "manifest.json")
	ledgerPath := filepath.Join(reviewDir, "ledger.json")
	submissionPath := filepath.Join(reviewDir, "submissions", "ux.json")
	if err := os.WriteFile(manifestPath, []byte(`{"round_id":"review-002-full","kind":"delta","anchor_sha":"abc123def","revision":2,"review_title":"Finalize review","plan_path":"docs/plans/active/2026-04-02-ui-review-plan.md","plan_stem":"2026-04-02-ui-review-plan","created_at":"2026-04-02T12:00:00Z","ledger_path":"`+ledgerPath+`","aggregate_path":"`+filepath.Join(reviewDir, "aggregate.json")+`","submissions_dir":"`+filepath.Join(reviewDir, "submissions")+`","dimensions":[{"name":"UX","slot":"ux","instructions":"Check the interface hierarchy.","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(ledgerPath, []byte(`{"round_id":"review-002-full","kind":"full","updated_at":"2026-04-02T12:10:00Z","slots":[{"name":"UX","slot":"ux","status":"submitted","submitted_at":"2026-04-02T12:08:00Z","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte(`{"round_id":"review-002-full","slot":"ux","dimension":"UX","submitted_at":"2026-04-02T12:08:00Z","summary":"Hierarchy is clear.","findings":[],"worklog":{"full_plan_read":true,"checked_areas":["web/src/pages.tsx"],"open_questions":["Should the review summary stay compact?"],"candidate_findings":["Hierarchy polish"]},"coverage":{"review_kind":"delta","anchor_sha":"abc123def"}}`), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/review", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Rounds   []struct {
			RoundID   string `json:"round_id"`
			Status    string `json:"status"`
			AnchorSHA string `json:"anchor_sha"`
			Reviewers []struct {
				Instructions  string          `json:"instructions"`
				Summary       string          `json:"summary"`
				RawSubmission json.RawMessage `json:"raw_submission"`
				Worklog       struct {
					ReviewKind string   `json:"review_kind"`
					AnchorSHA  string   `json:"anchor_sha"`
					Checked    []string `json:"checked_areas"`
				} `json:"worklog"`
			} `json:"reviewers"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || payload.Resource != "review" {
		t.Fatalf("unexpected review payload: %#v", payload)
	}
	if len(payload.Rounds) != 1 || payload.Rounds[0].RoundID != "review-002-full" {
		t.Fatalf("unexpected rounds: %#v", payload.Rounds)
	}
	if payload.Rounds[0].Status != "waiting_for_aggregation" {
		t.Fatalf("expected waiting_for_aggregation status, got %#v", payload.Rounds[0])
	}
	if len(payload.Rounds[0].Reviewers) != 1 || payload.Rounds[0].Reviewers[0].Instructions == "" || payload.Rounds[0].Reviewers[0].Summary == "" {
		t.Fatalf("expected reviewer content, got %#v", payload.Rounds[0].Reviewers)
	}
	if payload.Rounds[0].AnchorSHA != "abc123def" {
		t.Fatalf("expected round anchor SHA, got %#v", payload.Rounds[0])
	}
	if payload.Rounds[0].Reviewers[0].Worklog.ReviewKind != "delta" || payload.Rounds[0].Reviewers[0].Worklog.AnchorSHA != "abc123def" {
		t.Fatalf("expected reviewer worklog context, got %#v", payload.Rounds[0].Reviewers[0].Worklog)
	}
	if len(payload.Rounds[0].Reviewers[0].Worklog.Checked) != 1 || payload.Rounds[0].Reviewers[0].Worklog.Checked[0] != "web/src/pages.tsx" {
		t.Fatalf("expected reviewer checked areas, got %#v", payload.Rounds[0].Reviewers[0].Worklog)
	}
	if len(payload.Rounds[0].Reviewers[0].RawSubmission) == 0 || !strings.Contains(string(payload.Rounds[0].Reviewers[0].RawSubmission), "Hierarchy polish") {
		t.Fatalf("expected raw submission payload, got %#v", string(payload.Rounds[0].Reviewers[0].RawSubmission))
	}
}

func TestNewHandlerServesReviewJSONWithMalformedWorklogWarnings(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-10-ui-review-malformed-worklog.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Review Malformed Worklog")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-10-ui-review-malformed-worklog", &runstate.State{
		Revision: 1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-delta",
			Kind:       "delta",
			Revision:   1,
			Aggregated: false,
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	reviewDir := filepath.Join(workdir, ".local", "harness", "plans", "2026-04-10-ui-review-malformed-worklog", "reviews", "review-001-delta")
	if err := os.MkdirAll(filepath.Join(reviewDir, "submissions"), 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	manifestPath := filepath.Join(reviewDir, "manifest.json")
	ledgerPath := filepath.Join(reviewDir, "ledger.json")
	submissionPath := filepath.Join(reviewDir, "submissions", "risk.json")
	if err := os.WriteFile(manifestPath, []byte(`{"round_id":"review-001-delta","kind":"delta","anchor_sha":"abc123def","revision":1,"review_title":"Malformed worklog review","plan_path":"`+relPlanPath+`","plan_stem":"2026-04-10-ui-review-malformed-worklog","created_at":"2026-04-10T12:00:00Z","ledger_path":"`+ledgerPath+`","aggregate_path":"`+filepath.Join(reviewDir, "aggregate.json")+`","submissions_dir":"`+filepath.Join(reviewDir, "submissions")+`","dimensions":[{"name":"Risk","slot":"risk","instructions":"Check degraded worklog handling.","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(ledgerPath, []byte(`{"round_id":"review-001-delta","kind":"delta","updated_at":"2026-04-10T12:10:00Z","slots":[{"name":"Risk","slot":"risk","status":"submitted","submitted_at":"2026-04-10T12:08:00Z","submission_path":"`+submissionPath+`"}]}`), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte(`{"round_id":"review-001-delta","slot":"risk","dimension":"Risk","submitted_at":"2026-04-10T12:08:00Z","summary":"Malformed worklog fields should not crash the review API.","findings":[],"worklog":{"full_plan_read":"yes","checked_areas":["web/src/pages.tsx"],"open_questions":"still investigating","candidate_findings":["Candidate trail"]},"coverage":{"review_kind":7,"anchor_sha":"abc123def"}}`), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/review", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		OK     bool `json:"ok"`
		Rounds []struct {
			Reviewers []struct {
				Warnings []string `json:"warnings"`
				Worklog  struct {
					AnchorSHA  string   `json:"anchor_sha"`
					ReviewKind string   `json:"review_kind"`
					Checked    []string `json:"checked_areas"`
				} `json:"worklog"`
			} `json:"reviewers"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if !payload.OK || len(payload.Rounds) != 1 || len(payload.Rounds[0].Reviewers) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	reviewer := payload.Rounds[0].Reviewers[0]
	if reviewer.Worklog.AnchorSHA != "abc123def" || reviewer.Worklog.ReviewKind != "" {
		t.Fatalf("expected partial worklog recovery, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.Checked) != 1 || reviewer.Worklog.Checked[0] != "web/src/pages.tsx" {
		t.Fatalf("expected checked areas to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Warnings) == 0 || !strings.Contains(strings.Join(reviewer.Warnings, " "), "malformed") {
		t.Fatalf("expected malformed worklog warnings, got %#v", reviewer.Warnings)
	}
}

func TestNewHandlerServesReviewJSONFailureAs503(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-02-ui-review-error.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Review Error")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	planStem := "2026-04-02-ui-review-error"
	if _, err := runstate.SaveState(workdir, planStem, &runstate.State{
		Revision: 1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	reviewsPath := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews")
	if err := os.MkdirAll(filepath.Dir(reviewsPath), 0o755); err != nil {
		t.Fatalf("mkdir reviews parent: %v", err)
	}
	if err := os.WriteFile(reviewsPath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write invalid reviews path: %v", err)
	}

	handler, err := NewHandler(workdir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/review", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}

	var payload struct {
		OK       bool   `json:"ok"`
		Resource string `json:"resource"`
		Summary  string `json:"summary"`
		Errors   []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, recorder.Body.String())
	}
	if payload.OK || payload.Resource != "review" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if !strings.Contains(payload.Summary, "Unable to enumerate review rounds") {
		t.Fatalf("unexpected summary: %#v", payload)
	}
	if len(payload.Errors) != 1 || payload.Errors[0].Path != "reviews" {
		t.Fatalf("unexpected errors: %#v", payload.Errors)
	}
}

func TestNewHandlerServesLargeTimelinePayloadWithoutTruncation(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-01-ui-timeline-large.md"
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered := renderPlanFixture(t, "UI Timeline Large Payload")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-01-ui-timeline-large", &runstate.State{
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
	rendered := renderPlanFixture(t, "UI Timeline Artifact Tabs")
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	submissionRelPath := ".local/harness/plans/2026-04-01-ui-timeline-artifacts/reviews/review-001-full/submissions/correctness/submission.json"
	submissionPath := filepath.Join(workdir, filepath.FromSlash(submissionRelPath))
	if err := os.MkdirAll(filepath.Dir(submissionPath), 0o755); err != nil {
		t.Fatalf("mkdir submission dir: %v", err)
	}
	if err := os.WriteFile(submissionPath, []byte("{\"summary\":\"Artifact tabs\"}\n"), 0o644); err != nil {
		t.Fatalf("write submission: %v", err)
	}

	if _, _, err := timeline.AppendEvent(workdir, "2026-04-01-ui-timeline-artifacts", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review start",
		Summary:    "Created review round.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ArtifactRefs: []timeline.ArtifactRef{
			{Label: "submission_correctness_path", Value: submissionRelPath, Path: submissionRelPath},
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
	if content["summary"] != "Artifact tabs" {
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

func TestServerRunStartupDoesNotTouchWatchlist(t *testing.T) {
	logs := &lockedBuffer{}
	workdir := filepath.Join(t.TempDir(), "workspace")
	seedGitWorkspace(t, workdir)
	home := t.TempDir()
	t.Setenv("EASYHARNESS_HOME", home)

	server := Server{
		Workdir:     workdir,
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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(logs.String(), "Harness UI listening at http://") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(logs.String(), "Harness UI listening at http://") {
		t.Fatalf("expected listening URL in stdout, got %q", logs.String())
	}
	if _, err := os.Stat(filepath.Join(home, "watchlist.json")); !os.IsNotExist(err) {
		t.Fatalf("expected UI startup to avoid watchlist writes, err=%v", err)
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

func seedGitWorkspace(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir git workspace root %q: %v", root, err)
	}
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Codex Test")
	runGit(t, root, "config", "user.email", "codex@example.com")
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func renderPlanFixture(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: title})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	return strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
}

func writeUIActivePlan(t *testing.T, root, title string) (string, string) {
	t.Helper()
	planStem := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	relPath := filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-04-22-"+planStem+".md"))
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(renderPlanFixture(t, title)), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath, "2026-04-22-" + planStem
}

func seedUIActiveState(t *testing.T, root, relPlanPath, planStem string) (string, string, string) {
	t.Helper()
	currentPlanPath, err := runstate.SaveCurrentPlan(root, relPlanPath)
	if err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	statePath, err := runstate.SaveState(root, planStem, &runstate.State{
		ExecutionStartedAt: "2026-04-22T09:00:00Z",
		Revision:           1,
	})
	if err != nil {
		t.Fatalf("save state: %v", err)
	}
	fixedTime := time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC)
	for _, path := range []string{currentPlanPath, statePath} {
		if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
			t.Fatalf("set state timestamp %s: %v", path, err)
		}
	}
	lockPath := filepath.Join(root, ".local", "harness", "plans", planStem, ".state-mutation.lock")
	return currentPlanPath, statePath, lockPath
}

func snapshotStateFiles(t *testing.T, paths ...string) map[string]fileSnapshot {
	t.Helper()
	before := make(map[string]fileSnapshot, len(paths))
	for _, path := range paths {
		before[path] = snapshotFile(t, path)
	}
	return before
}

func assertStateFilesUnchanged(t *testing.T, before map[string]fileSnapshot) {
	t.Helper()
	for path, snapshot := range before {
		assertFileUnchanged(t, path, snapshot)
	}
}

type dashboardTestGroup struct {
	State      string                   `json:"state"`
	Workspaces []dashboardTestWorkspace `json:"workspaces"`
}

type dashboardTestWorkspace struct {
	WorkspacePath  string `json:"workspace_path"`
	DashboardState string `json:"dashboard_state"`
	CurrentNode    string `json:"current_node"`
}

func dashboardWorkspaceInGroup(t *testing.T, groups []dashboardTestGroup, state, path string) dashboardTestWorkspace {
	t.Helper()
	for _, group := range groups {
		if group.State != state {
			continue
		}
		for _, workspace := range group.Workspaces {
			if workspace.WorkspacePath == path {
				return workspace
			}
		}
	}
	t.Fatalf("missing workspace %q in dashboard state %q", path, state)
	return dashboardTestWorkspace{}
}

type fileSnapshot struct {
	data    []byte
	modTime time.Time
}

func snapshotFile(t *testing.T, path string) fileSnapshot {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file %s: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return fileSnapshot{data: data, modTime: info.ModTime()}
}

func assertFileUnchanged(t *testing.T, path string, before fileSnapshot) {
	t.Helper()
	after := snapshotFile(t, path)
	if string(after.data) != string(before.data) {
		t.Fatalf("expected %s bytes to remain unchanged", path)
	}
	if !after.modTime.Equal(before.modTime) {
		t.Fatalf("expected %s mtime to remain unchanged, got %s want %s", path, after.modTime, before.modTime)
	}
}

func writeWatchlist(t *testing.T, home string, workspaces []watchlist.Workspace) {
	t.Helper()
	payload := struct {
		Version    int                   `json:"version"`
		Workspaces []watchlist.Workspace `json:"workspaces"`
	}{
		Version:    1,
		Workspaces: workspaces,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal watchlist: %v", err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir watchlist home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "watchlist.json"), data, 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
}

func readWatchlist(t *testing.T, home string) watchlist.File {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, "watchlist.json"))
	if err != nil {
		t.Fatalf("read watchlist: %v", err)
	}
	var file watchlist.File
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("decode watchlist: %v\n%s", err, data)
	}
	return file
}

func workspaceRecord(path, seenAt string) watchlist.Workspace {
	return watchlist.Workspace{
		WorkspacePath: path,
		WatchedAt:     "2026-04-22T09:00:00Z",
		LastSeenAt:    seenAt,
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
