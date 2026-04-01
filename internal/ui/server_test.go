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
