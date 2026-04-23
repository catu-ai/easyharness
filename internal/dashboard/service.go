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
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/watchlist"
)

const (
	StateActive    = "active"
	StateCompleted = "completed"
	StateIdle      = "idle"
	StateMissing   = "missing"
	StateInvalid   = "invalid"

	InvalidUnreadable        = "unreadable"
	InvalidNotGitWorkspace   = "not_git_workspace"
	InvalidStatusError       = "status_error"
	InvalidMalformedPath     = "malformed_path"
	InvalidRouteKeyCollision = "route_key_collision"
)

var dashboardStateOrder = []string{StateActive, StateCompleted, StateIdle, StateMissing, StateInvalid}

type Service struct {
	LookupEnv         func(string) (string, bool)
	UserHomeDir       func() (string, error)
	ReadStatus        func(string) contracts.StatusResult
	Stat              func(string) (os.FileInfo, error)
	CheckGitWorkspace func(string) error
	LoadPlan          func(string) (*plan.Document, error)
}

type Result = contracts.DashboardResult
type Group = contracts.DashboardGroup
type Workspace = contracts.DashboardWorkspace
type ErrorDetail = contracts.ErrorDetail
type WorkspaceResult = contracts.DashboardWorkspaceResult
type Progress = contracts.DashboardProgress
type ProgressNode = contracts.DashboardProgressNode

const (
	progressStatePending = "pending"
	progressStateCurrent = "current"
	progressStateDone    = "done"
)

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
	markRouteKeyCollisions(entries)
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

func (s Service) ReadWorkspace(key string) WorkspaceResult {
	key = strings.TrimSpace(key)
	if key == "" {
		return WorkspaceResult{
			OK:       false,
			Resource: "workspace",
			Summary:  "Workspace route key is empty.",
			Errors:   []ErrorDetail{{Path: "workspace_key", Message: "workspace route key is empty"}},
		}
	}

	file, err := watchlist.Service{
		LookupEnv:   s.LookupEnv,
		UserHomeDir: s.UserHomeDir,
	}.Read()
	if err != nil {
		return WorkspaceResult{
			OK:       false,
			Resource: "workspace",
			Summary:  "Unable to load the machine-local watchlist.",
			Errors:   []ErrorDetail{{Path: "watchlist", Message: err.Error()}},
		}
	}

	matches := make([]watchlist.Workspace, 0, 2)
	for _, watched := range file.Workspaces {
		if WorkspaceKey(watched.WorkspacePath) == key {
			matches = append(matches, watched)
		}
	}
	if len(matches) == 0 {
		return WorkspaceResult{
			OK:       true,
			Resource: "workspace",
			Summary:  "Workspace is not currently watched.",
			Watched:  false,
		}
	}

	entry := s.readWorkspace(matches[0])
	if len(matches) > 1 {
		entry.DashboardState = StateInvalid
		entry.InvalidReason = InvalidRouteKeyCollision
		entry.CurrentNode = ""
		entry.NextAction = nil
		entry.Warnings = nil
		entry.Blockers = nil
		entry.Artifacts = nil
		entry.Facts = nil
		entry.Progress = nil
		entry.Summary = "Watched workspace route key collides with another watchlist record."
		entry.Errors = append(entry.Errors, ErrorDetail{
			Path:    "workspace_key",
			Message: fmt.Sprintf("workspace_key %q is shared by %d watchlist records", key, len(matches)),
		})
	}
	return WorkspaceResult{
		OK:        true,
		Resource:  "workspace",
		Summary:   fmt.Sprintf("Loaded watched workspace %s.", entry.WorkspaceName),
		Watched:   true,
		Workspace: &entry,
	}
}

func (s Service) readWorkspace(watched watchlist.Workspace) Workspace {
	path := strings.TrimSpace(watched.WorkspacePath)
	entry := Workspace{
		WorkspaceKey:   workspaceKey(path),
		WorkspaceName:  workspaceName(path),
		WorkspacePath:  path,
		WatchedAt:      strings.TrimSpace(watched.WatchedAt),
		LastSeenAt:     strings.TrimSpace(watched.LastSeenAt),
		DashboardState: StateInvalid,
		Summary:        "Watched workspace is invalid.",
	}

	if path == "" {
		entry.InvalidReason = InvalidMalformedPath
		entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: "watched workspace path is empty"}}
		return entry
	}
	if !filepath.IsAbs(path) {
		entry.InvalidReason = InvalidMalformedPath
		entry.Errors = []ErrorDetail{{Path: "workspace_path", Message: "watched workspace path must be absolute"}}
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
		entry.Facts = statusResult.Facts
		entry.NextAction = statusResult.NextAction
		entry.Warnings = statusResult.Warnings
		entry.Blockers = statusResult.Blockers
		entry.Errors = statusResult.Errors
		entry.Artifacts = statusResult.Artifacts
		entry.PlanTitle, entry.Progress = s.planContext(path, statusResult)
		return entry
	}

	entry.DashboardState = dashboardState(statusResult)
	entry.Summary = statusResult.Summary
	entry.CurrentNode = statusResult.State.CurrentNode
	entry.Facts = statusResult.Facts
	entry.NextAction = statusResult.NextAction
	entry.Warnings = statusResult.Warnings
	entry.Blockers = statusResult.Blockers
	entry.Errors = statusResult.Errors
	entry.Artifacts = statusResult.Artifacts
	entry.PlanTitle, entry.Progress = s.planContext(path, statusResult)
	return entry
}

func markRouteKeyCollisions(entries []Workspace) {
	byKey := make(map[string][]int, len(entries))
	for index, entry := range entries {
		byKey[entry.WorkspaceKey] = append(byKey[entry.WorkspaceKey], index)
	}
	for key, indexes := range byKey {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			entry := &entries[index]
			entry.DashboardState = StateInvalid
			entry.InvalidReason = InvalidRouteKeyCollision
			entry.CurrentNode = ""
			entry.NextAction = nil
			entry.Warnings = nil
			entry.Blockers = nil
			entry.Artifacts = nil
			entry.Summary = "Watched workspace route key collides with another watchlist record."
			entry.Errors = append(entry.Errors, ErrorDetail{
				Path:    "workspace_key",
				Message: fmt.Sprintf("workspace_key %q is shared by %d watchlist records", key, len(indexes)),
			})
		}
	}
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

func WorkspaceKey(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(strings.TrimSpace(path))))
	return "wk_" + hex.EncodeToString(sum[:])[:16]
}

func workspaceKey(path string) string {
	return WorkspaceKey(path)
}

func workspaceName(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return ""
	}
	return filepath.Base(path)
}

func (s Service) loadPlan(path string) (*plan.Document, error) {
	if s.LoadPlan != nil {
		return s.LoadPlan(path)
	}
	return plan.LoadFile(path)
}

func (s Service) planContext(workspacePath string, statusResult contracts.StatusResult) (string, *Progress) {
	if statusResult.Artifacts == nil {
		return "", nil
	}
	relPlanPath := strings.TrimSpace(statusResult.Artifacts.PlanPath)
	if relPlanPath == "" {
		return "", nil
	}
	absPlanPath := filepath.Join(workspacePath, filepath.FromSlash(relPlanPath))
	doc, err := s.loadPlan(absPlanPath)
	if err != nil || doc == nil {
		return "", nil
	}
	return doc.Title, buildProgress(doc, statusResult)
}

func buildProgress(doc *plan.Document, statusResult contracts.StatusResult) *Progress {
	if doc == nil || len(doc.Steps) == 0 {
		return nil
	}

	nodes := make([]ProgressNode, 0, len(doc.Steps)+2)
	for _, step := range doc.Steps {
		nodes = append(nodes, ProgressNode{Label: step.Title, State: progressStatePending})
	}
	nodes = append(nodes, ProgressNode{Label: "Finalize", State: progressStatePending})
	nodes = append(nodes, ProgressNode{Label: "Await merge", State: progressStatePending})

	currentIndex, allDone := progressPosition(doc, statusResult)
	if allDone {
		for i := range nodes {
			nodes[i].State = progressStateDone
		}
		return &Progress{Nodes: nodes}
	}
	if currentIndex < 0 {
		return &Progress{Nodes: nodes}
	}
	if currentIndex >= len(nodes) {
		currentIndex = len(nodes) - 1
	}
	for i := range nodes {
		switch {
		case i < currentIndex:
			nodes[i].State = progressStateDone
		case i == currentIndex:
			nodes[i].State = progressStateCurrent
		default:
			nodes[i].State = progressStatePending
		}
	}
	return &Progress{Nodes: nodes}
}

func progressPosition(doc *plan.Document, statusResult contracts.StatusResult) (int, bool) {
	currentNode := strings.TrimSpace(statusResult.State.CurrentNode)
	if currentNode == "land" {
		return len(doc.Steps) + 1, true
	}
	if currentNode == "idle" {
		if statusResult.Artifacts != nil && strings.TrimSpace(statusResult.Artifacts.LastLandedAt) != "" {
			return len(doc.Steps) + 1, true
		}
		if statusResult.Facts != nil {
			if index := stepIndexForTitle(doc, statusResult.Facts.CurrentStep); index >= 0 {
				return index, false
			}
		}
		return -1, false
	}
	if index, ok := stepIndexFromNode(currentNode); ok {
		return index, false
	}
	if strings.HasPrefix(currentNode, "execution/finalize/") {
		if strings.HasSuffix(currentNode, "/await_merge") {
			return len(doc.Steps) + 1, false
		}
		return len(doc.Steps), false
	}
	if currentNode == "plan" && statusResult.Facts != nil {
		if index := stepIndexForTitle(doc, statusResult.Facts.CurrentStep); index >= 0 {
			return index, false
		}
	}
	return -1, false
}

func stepIndexFromNode(currentNode string) (int, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(currentNode), "execution/step-")
	if trimmed == currentNode {
		return 0, false
	}
	stepPart, _, ok := strings.Cut(trimmed, "/")
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(stepPart)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value - 1, true
}

func stepIndexForTitle(doc *plan.Document, title string) int {
	title = strings.TrimSpace(title)
	if title == "" || doc == nil {
		return -1
	}
	for index, step := range doc.Steps {
		if strings.TrimSpace(step.Title) == title {
			return index
		}
	}
	return -1
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
