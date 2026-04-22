package dashboard

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/watchlist"
)

const (
	StateActive    = "active"
	StateCompleted = "completed"
	StateIdle      = "idle"
	StateMissing   = "missing"
	StateInvalid   = "invalid"

	InvalidUnreadable      = "unreadable"
	InvalidNotGitWorkspace = "not_git_workspace"
	InvalidStatusError     = "status_error"
)

var dashboardStateOrder = []string{StateActive, StateCompleted, StateIdle, StateMissing, StateInvalid}

type Service struct {
	LookupEnv         func(string) (string, bool)
	UserHomeDir       func() (string, error)
	ReadStatus        func(string) contracts.StatusResult
	Stat              func(string) (os.FileInfo, error)
	CheckGitWorkspace func(string) error
}

type Result = contracts.DashboardResult
type Group = contracts.DashboardGroup
type Workspace = contracts.DashboardWorkspace
type ErrorDetail = contracts.ErrorDetail

func (s Service) Read() Result {
	file, err := watchlist.Service{
		LookupEnv:   s.LookupEnv,
		UserHomeDir: s.UserHomeDir,
	}.Read()
	if err != nil {
		return Result{
			OK:       false,
			Resource: "dashboard",
			Summary:  "Unable to load the machine-local watchlist.",
			Groups:   emptyGroups(),
			Errors:   []ErrorDetail{{Path: "watchlist", Message: err.Error()}},
		}
	}

	entries := make([]Workspace, 0, len(file.Workspaces))
	for _, watched := range file.Workspaces {
		entries = append(entries, s.readWorkspace(watched))
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entryLess(entries[i], entries[j])
	})

	groups := groupEntries(entries)
	return Result{
		OK:       true,
		Resource: "dashboard",
		Summary:  fmt.Sprintf("Loaded %d watched workspace(s).", len(entries)),
		Groups:   groups,
	}
}

func (s Service) readWorkspace(watched watchlist.Workspace) Workspace {
	path := strings.TrimSpace(watched.WorkspacePath)
	entry := Workspace{
		WorkspaceKey:   workspaceKey(path),
		WorkspacePath:  path,
		WatchedAt:      strings.TrimSpace(watched.WatchedAt),
		LastSeenAt:     strings.TrimSpace(watched.LastSeenAt),
		DashboardState: StateInvalid,
		Summary:        "Watched workspace is invalid.",
	}

	if path == "" {
		entry.InvalidReason = InvalidNotGitWorkspace
		entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: "watched workspace path is empty"}}
		return entry
	}

	info, err := s.stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			entry.DashboardState = StateMissing
			entry.Summary = "Watched workspace path is missing."
			entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: "watched workspace path is missing"}}
			return entry
		}
		entry.InvalidReason = InvalidUnreadable
		entry.Summary = "Watched workspace path is unreadable."
		entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: err.Error()}}
		return entry
	}
	if !info.IsDir() {
		entry.InvalidReason = InvalidNotGitWorkspace
		entry.Summary = "Watched workspace path is not a Git workspace."
		entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: "watched workspace path is not a directory"}}
		return entry
	}
	if err := s.checkGitWorkspace(path); err != nil {
		if errors.Is(err, watchlist.ErrNotGitWorkspace) {
			entry.InvalidReason = InvalidNotGitWorkspace
			entry.Summary = "Watched workspace path is not a Git workspace."
		} else {
			entry.InvalidReason = InvalidUnreadable
			entry.Summary = "Unable to inspect watched workspace Git metadata."
		}
		entry.Errors = []ErrorDetail{{Path: "git", Message: err.Error()}}
		return entry
	}

	statusResult := s.readStatus(path)
	if !statusResult.OK {
		entry.InvalidReason = InvalidStatusError
		entry.Summary = statusResult.Summary
		entry.CurrentNode = statusResult.State.CurrentNode
		entry.NextAction = statusResult.NextAction
		entry.Warnings = statusResult.Warnings
		entry.Blockers = statusResult.Blockers
		entry.Errors = statusResult.Errors
		entry.Artifacts = statusResult.Artifacts
		return entry
	}

	entry.DashboardState = dashboardState(statusResult)
	entry.Summary = statusResult.Summary
	entry.CurrentNode = statusResult.State.CurrentNode
	entry.NextAction = statusResult.NextAction
	entry.Warnings = statusResult.Warnings
	entry.Blockers = statusResult.Blockers
	entry.Errors = statusResult.Errors
	entry.Artifacts = statusResult.Artifacts
	return entry
}

func (s Service) stat(path string) (os.FileInfo, error) {
	if s.Stat != nil {
		return s.Stat(path)
	}
	return os.Stat(path)
}

func (s Service) checkGitWorkspace(path string) error {
	if s.CheckGitWorkspace != nil {
		return s.CheckGitWorkspace(path)
	}
	return requireGitWorkspace(path)
}

func (s Service) readStatus(path string) contracts.StatusResult {
	if s.ReadStatus != nil {
		return s.ReadStatus(path)
	}
	return status.Service{Workdir: path}.ReadUnlocked()
}

func dashboardState(result contracts.StatusResult) string {
	if result.State.CurrentNode != "idle" {
		return StateActive
	}
	if result.Artifacts != nil && strings.TrimSpace(result.Artifacts.LastLandedAt) != "" {
		return StateCompleted
	}
	return StateIdle
}

func requireGitWorkspace(path string) error {
	output, err := exec.Command("git", "-C", filepath.Clean(path), "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return classifyGitProbeExit(path, message)
		}
		return fmt.Errorf("inspect git workspace: %w", err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return watchlist.ErrNotGitWorkspace
	}
	return nil
}

func classifyGitProbeExit(path, message string) error {
	lowerMessage := strings.ToLower(message)
	if strings.Contains(lowerMessage, "not a git repository") ||
		strings.Contains(lowerMessage, "not a git work tree") {
		if gitMarkerExists(path) {
			return fmt.Errorf("inspect git workspace: %s", message)
		}
		return fmt.Errorf("%w: %s", watchlist.ErrNotGitWorkspace, message)
	}
	return fmt.Errorf("inspect git workspace: %s", message)
}

func gitMarkerExists(path string) bool {
	_, err := os.Lstat(filepath.Join(path, ".git"))
	return err == nil
}

func workspaceKey(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(strings.TrimSpace(path))))
	return "wk_" + hex.EncodeToString(sum[:])[:16]
}

func emptyGroups() []Group {
	return groupEntries(nil)
}

func groupEntries(entries []Workspace) []Group {
	byState := make(map[string][]Workspace, len(dashboardStateOrder))
	for _, state := range dashboardStateOrder {
		byState[state] = []Workspace{}
	}
	for _, entry := range entries {
		state := entry.DashboardState
		if _, ok := byState[state]; !ok {
			state = StateInvalid
			entry.DashboardState = StateInvalid
			if entry.InvalidReason == "" {
				entry.InvalidReason = InvalidStatusError
			}
		}
		byState[state] = append(byState[state], entry)
	}

	groups := make([]Group, 0, len(dashboardStateOrder))
	for _, state := range dashboardStateOrder {
		groups = append(groups, Group{State: state, Workspaces: byState[state]})
	}
	return groups
}

func entryLess(a, b Workspace) bool {
	at, aOK := parseTimestamp(a.LastSeenAt)
	bt, bOK := parseTimestamp(b.LastSeenAt)
	if aOK && bOK && !at.Equal(bt) {
		return at.After(bt)
	}
	if aOK != bOK {
		return aOK
	}
	return a.WorkspacePath < b.WorkspacePath
}

func parseTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	return parsed, err == nil
}
