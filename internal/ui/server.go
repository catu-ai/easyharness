package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/dashboard"
	"github.com/catu-ai/easyharness/internal/planui"
	"github.com/catu-ai/easyharness/internal/reviewui"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/timeline"
	"github.com/catu-ai/easyharness/internal/watchlist"
)

//go:embed generated
var embeddedStatic embed.FS

const productDisplayName = "easyharness"

type Server struct {
	Workdir     string
	Host        string
	Port        int
	Stdout      io.Writer
	Stderr      io.Writer
	OpenBrowser bool
	OpenPath    string
}

func (s Server) Run(ctx context.Context) error {
	host := strings.TrimSpace(s.Host)
	if host == "" {
		host = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(s.Port)))
	if err != nil {
		return fmt.Errorf("listen for harness ui: %w", err)
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String()
	if s.Stdout != nil {
		_, _ = fmt.Fprintf(s.Stdout, "Harness UI listening at %s\n", url)
	}

	if s.OpenBrowser {
		target := browserTarget(url, s.OpenPath)
		if err := openBrowser(target); err != nil && s.Stderr != nil {
			_, _ = fmt.Fprintf(s.Stderr, "open browser: %v\n", err)
		}
	}

	handler, err := NewHandler(s.Workdir)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve harness ui: %w", err)
	}
	return nil
}

func NewHandler(workdir string) (http.Handler, error) {
	staticFS, err := fs.Sub(embeddedStatic, "generated/build")
	if err != nil {
		return nil, fmt.Errorf("load embedded ui assets: %w (run scripts/install-dev-harness or scripts/build-embedded-ui first)", err)
	}

	watchlistSvc := watchlist.Service{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeDashboardJSON(w, dashboard.Service{}.Read())
	})
	mux.HandleFunc("/api/workspace/", func(w http.ResponseWriter, r *http.Request) {
		key, resource, ok := parseWorkspaceAPIPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		dashboardSvc := dashboard.Service{}
		workspaceResult := dashboardSvc.ReadWorkspace(key)
		if resource == "" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			writeWorkspaceJSON(w, workspaceResult)
			return
		}
		if resource == "unwatch" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			file, err := watchlistSvc.Read()
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":       false,
					"resource": "workspace",
					"summary":  "Unable to load the machine-local watchlist.",
					"errors": []map[string]string{{
						"path":    "watchlist",
						"message": err.Error(),
					}},
				})
				return
			}
			request, err := decodeWorkspaceActionRequest(r)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":       false,
					"resource": "workspace",
					"summary":  "Workspace action request body is invalid.",
					"errors": []map[string]string{{
						"path":    "workspace_path",
						"message": err.Error(),
					}},
				})
				return
			}
			targetPath, resolveErr := resolveWorkspaceActionTarget(workspaceMatchesByKey(file.Workspaces, key), request.WorkspacePath)
			if resolveErr != nil {
				statusCode := http.StatusConflict
				if errors.Is(resolveErr, errWorkspaceActionTargetNotFound) {
					statusCode = http.StatusNotFound
				}
				writeJSON(w, statusCode, map[string]any{
					"ok":       false,
					"resource": "workspace",
					"summary":  resolveErr.Error(),
				})
				return
			}
			if err := watchlistSvc.Unwatch(targetPath); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":       false,
					"resource": "workspace",
					"summary":  "Unable to remove workspace from the machine-local watchlist.",
					"errors": []map[string]string{{
						"path":    "watchlist",
						"message": err.Error(),
					}},
				})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":       true,
				"resource": "workspace",
				"summary":  "Removed workspace from the machine-local watchlist.",
			})
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !workspaceResult.OK {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":       false,
				"resource": resource,
				"summary":  workspaceResult.Summary,
				"errors":   workspaceResult.Errors,
			})
			return
		}
		if !workspaceResult.Watched || workspaceResult.Workspace == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":       false,
				"resource": resource,
				"summary":  "Workspace is not currently watched.",
			})
			return
		}
		if state := workspaceResult.Workspace.DashboardState; state == dashboard.StateMissing || state == dashboard.StateInvalid {
			writeJSON(w, http.StatusConflict, map[string]any{
				"ok":       false,
				"resource": resource,
				"summary":  workspaceResult.Workspace.Summary,
				"errors":   workspaceResult.Workspace.Errors,
			})
			return
		}

		workspacePath := workspaceResult.Workspace.WorkspacePath
		switch resource {
		case "status":
			writeStatusJSON(w, status.Service{Workdir: workspacePath}.Read())
		case "plan":
			writePlanJSON(w, planui.Service{Workdir: workspacePath}.Read())
		case "timeline":
			writeTimelineJSON(w, timeline.Service{Workdir: workspacePath}.Read())
		case "review":
			writeReviewJSON(w, reviewui.Service{Workdir: workspacePath}.Read())
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeStatusJSON(w, status.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/plan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writePlanJSON(w, planui.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/timeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeTimelineJSON(w, timeline.Service{Workdir: workdir}.Read())
	})
	mux.HandleFunc("/api/review", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeReviewJSON(w, reviewui.Service{Workdir: workdir}.Read())
	})
	mux.Handle("/", spaHandler(staticFS, workdir))
	return mux, nil
}

var (
	errWorkspaceActionTargetNotFound = errors.New("workspace action target is not currently watched")
	errWorkspaceActionTargetAmbiguous = errors.New("workspace route key is ambiguous; specify the exact workspace_path")
)

type workspaceActionRequest struct {
	WorkspacePath string `json:"workspace_path"`
}

func decodeWorkspaceActionRequest(r *http.Request) (workspaceActionRequest, error) {
	if r.Body == nil {
		return workspaceActionRequest{}, nil
	}
	defer r.Body.Close()

	var request workspaceActionRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			return workspaceActionRequest{}, nil
		}
		return workspaceActionRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		return workspaceActionRequest{}, errors.New("request body must contain a single JSON object")
	}
	request.WorkspacePath = strings.TrimSpace(request.WorkspacePath)
	return request, nil
}

func workspaceMatchesByKey(entries []watchlist.Workspace, key string) []watchlist.Workspace {
	matches := make([]watchlist.Workspace, 0, 2)
	for _, entry := range entries {
		if dashboard.WorkspaceKey(entry.WorkspacePath) == key {
			matches = append(matches, entry)
		}
	}
	return matches
}

func resolveWorkspaceActionTarget(matches []watchlist.Workspace, requestedPath string) (string, error) {
	if requestedPath != "" {
		for _, match := range matches {
			if strings.TrimSpace(match.WorkspacePath) == requestedPath {
				return requestedPath, nil
			}
		}
		return "", fmt.Errorf("%w: %s", errWorkspaceActionTargetNotFound, requestedPath)
	}
	if len(matches) == 0 {
		return "", errWorkspaceActionTargetNotFound
	}
	if len(matches) > 1 {
		return "", errWorkspaceActionTargetAmbiguous
	}
	return strings.TrimSpace(matches[0].WorkspacePath), nil
}

func spaHandler(staticFS fs.FS, workdir string) http.Handler {
	files := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/api" {
			http.NotFound(w, r)
			return
		}

		requestPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		switch requestPath {
		case ".", "":
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		if requestPath == "dashboard" {
			serveIndex(staticFS, workdir, w)
			return
		}
		if nextPath, ok := workspacePagePath(requestPath); ok {
			if nextPath != "" {
				http.Redirect(w, r, nextPath, http.StatusFound)
				return
			}
			serveIndex(staticFS, workdir, w)
			return
		}

		if entry, err := fs.Stat(staticFS, requestPath); err == nil && !entry.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		serveIndex(staticFS, workdir, w)
	})
}

func browserTarget(baseURL, openPath string) string {
	openPath = strings.TrimSpace(openPath)
	if openPath == "" || openPath == "/" {
		return baseURL
	}
	if strings.HasPrefix(openPath, "/") {
		return baseURL + openPath
	}
	return baseURL + "/" + openPath
}

func parseWorkspaceAPIPath(rawPath string) (key, resource string, ok bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(rawPath), "/api/workspace/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		return parts[0], "", true
	}
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func workspacePagePath(requestPath string) (redirectPath string, ok bool) {
	parts := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(parts) < 2 || parts[0] != "workspace" {
		return "", false
	}
	if len(parts) == 2 {
		return "/" + path.Join("workspace", parts[1], "status"), true
	}
	if len(parts) == 3 && isWorkspacePage(parts[2]) {
		return "", true
	}
	return "", false
}

func isWorkspacePage(value string) bool {
	switch value {
	case "status", "plan", "timeline", "review":
		return true
	default:
		return false
	}
}

func writeWorkspaceJSON(w http.ResponseWriter, result dashboard.WorkspaceResult) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func serveIndex(staticFS fs.FS, workdir string, w http.ResponseWriter) {
	data, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		http.Error(w, "missing embedded ui index", http.StatusInternalServerError)
		return
	}
	workdirJSON, err := json.Marshal(filepath.Clean(workdir))
	if err != nil {
		http.Error(w, "encode workdir", http.StatusInternalServerError)
		return
	}
	repoNameJSON, err := json.Marshal(filepath.Base(filepath.Clean(workdir)))
	if err != nil {
		http.Error(w, "encode repo name", http.StatusInternalServerError)
		return
	}
	productNameJSON, err := json.Marshal(productDisplayName)
	if err != nil {
		http.Error(w, "encode product name", http.StatusInternalServerError)
		return
	}
	page := strings.ReplaceAll(string(data), "\"__HARNESS_UI_WORKDIR__\"", string(workdirJSON))
	page = strings.ReplaceAll(page, "\"__HARNESS_UI_REPO_NAME__\"", string(repoNameJSON))
	page = strings.ReplaceAll(page, "\"__HARNESS_UI_PRODUCT_NAME__\"", string(productNameJSON))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

func writeStatusJSON(w http.ResponseWriter, result status.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeDashboardJSON(w http.ResponseWriter, result dashboard.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writePlanJSON(w http.ResponseWriter, result planui.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeTimelineJSON(w http.ResponseWriter, result timeline.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeReviewJSON(w http.ResponseWriter, result reviewui.Result) {
	statusCode := http.StatusOK
	if !result.OK {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, result)
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
