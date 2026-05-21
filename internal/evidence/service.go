package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/inputschema"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/remote"
	"github.com/catu-ai/easyharness/internal/runstate"
)

var recordIDPattern = regexp.MustCompile(`^(ci|publish|sync)-([0-9]+)\.json$`)
var saveState = runstate.SaveState

type Service struct {
	Workdir       string
	Now           func() time.Time
	AfterMutation func(Result) error
	AfterRefresh  func(RefreshResult) error
	AfterSuccess  func(Result)
	RunCommand    remote.CommandRunner
}

type Result = contracts.EvidenceSubmitResult
type RefreshResult = contracts.EvidenceRefreshResult
type Artifacts = contracts.EvidenceArtifacts
type RefreshArtifacts = contracts.EvidenceRefreshArtifacts
type NextAction = contracts.NextAction
type CommandError = contracts.ErrorDetail
type CIInput = contracts.EvidenceCIInput
type PublishInput = contracts.EvidencePublishInput
type SyncInput = contracts.EvidenceSyncInput
type CIRecord = contracts.EvidenceCIRecord
type PublishRecord = contracts.EvidencePublishRecord
type SyncRecord = contracts.EvidenceSyncRecord

func (s Service) Submit(kind string, inputBytes []byte) Result {
	now := s.now().Format(time.RFC3339)
	planPath, relPlanPath, planStem, state, _, release, result := s.loadCurrentArchivedPlan()
	if result != nil {
		result.Command = "evidence submit"
		return *result
	}
	defer release()

	kind = strings.TrimSpace(strings.ToLower(kind))
	switch kind {
	case "ci":
		var input CIInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.ci", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("ci", issues)
		}
		if issues := validateCIInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("ci", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := CIRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			Provider:   strings.TrimSpace(input.Provider),
			URL:        strings.TrimSpace(input.URL),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		return s.finalizeMutation(successResult(planPath, kind, recordID, "Recorded CI evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(recordPath)
		})
	case "publish":
		var input PublishInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.publish", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("publish", issues)
		}
		if issues := validatePublishInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("publish", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := PublishRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			PRURL:      strings.TrimSpace(input.PRURL),
			Branch:     strings.TrimSpace(input.Branch),
			Base:       strings.TrimSpace(input.Base),
			Commit:     strings.TrimSpace(input.Commit),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		return s.finalizeMutation(successResult(planPath, kind, recordID, "Recorded publish evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(recordPath)
		})
	case "sync":
		var input SyncInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.sync", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("sync", issues)
		}
		if issues := validateSyncInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("sync", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := SyncRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			BaseRef:    strings.TrimSpace(input.BaseRef),
			HeadRef:    strings.TrimSpace(input.HeadRef),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		return s.finalizeMutation(successResult(planPath, kind, recordID, "Recorded sync evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(recordPath)
		})
	default:
		return errorResult("evidence submit", "Evidence kind is invalid.", []CommandError{{
			Path:    "kind",
			Message: "kind must be one of: ci, publish, sync",
		}})
	}
}

func (s Service) Refresh() RefreshResult {
	now := s.now().Format(time.RFC3339)
	_, relPlanPath, planStem, state, _, release, result := s.loadCurrentArchivedPlan()
	if result != nil {
		return refreshErrorResult("Evidence refresh requires the current archived candidate.", result.Errors)
	}
	defer release()

	revision := runstate.CurrentRevision(state)
	publish, err := LoadLatestPublish(s.Workdir, planStem, revision)
	if err != nil {
		return refreshErrorResult("Unable to read publish evidence for refresh.", []CommandError{{Path: "publish", Message: err.Error()}})
	}
	if publish == nil || strings.TrimSpace(publish.PRURL) == "" {
		return refreshErrorResult("Publish evidence has no recorded PR URL to refresh from.", []CommandError{{
			Path:    "publish.pr_url",
			Message: "record publish evidence with a PR URL before refreshing CI and sync evidence",
		}})
	}

	identity := remote.ParseRecordedPRURL(publish.PRURL)
	if identity.Status != remote.PRStatusRecorded {
		return refreshErrorResult("Publish evidence PR URL is not supported for refresh.", []CommandError{{
			Path:    "publish.pr_url",
			Message: identity.Degraded.Message,
		}})
	}

	observation := remote.Service{Workdir: s.Workdir, RunCommand: s.RunCommand}.ObserveHandoff(identity)
	artifacts := &RefreshArtifacts{
		PlanPath: relPlanPath,
		PRURL:    identity.URL,
	}
	warnings := make([]string, 0, 2)
	rollbackPaths := make([]string, 0, 2)

	if observation.CI.Status == remote.RemoteCIAvailable {
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, "ci")
		if err != nil {
			return refreshErrorResult("Unable to determine the next CI evidence record ID.", []CommandError{{Path: "ci.record_id", Message: err.Error()}})
		}
		record := CIRecord{
			RecordID:   recordID,
			Kind:       "ci",
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   revision,
			RecordedAt: now,
			Status:     observation.CI.EvidenceStatus,
			Provider:   "github-actions",
			URL:        firstCheckURL(observation.CI.Checks),
			Reason:     "Refreshed from recorded pull request checks.",
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return refreshErrorResult("Unable to persist CI evidence refresh.", []CommandError{{Path: "ci.record", Message: err.Error()}})
		}
		artifacts.CIRecordID = recordID
		rollbackPaths = append(rollbackPaths, recordPath)
	} else {
		warnings = append(warnings, degradedWarning("CI refresh unavailable", observation.CI.Degraded))
	}

	if observation.Sync.Status == remote.RemoteSyncAvailable {
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, "sync")
		if err != nil {
			return refreshErrorResult("Unable to determine the next sync evidence record ID.", []CommandError{{Path: "sync.record_id", Message: err.Error()}})
		}
		record := SyncRecord{
			RecordID:   recordID,
			Kind:       "sync",
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   revision,
			RecordedAt: now,
			Status:     observation.Sync.EvidenceStatus,
			BaseRef:    observation.PR.BaseRefName,
			HeadRef:    observation.PR.HeadRefName,
			Reason:     "Refreshed from recorded pull request merge state.",
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return refreshErrorResult("Unable to persist sync evidence refresh.", []CommandError{{Path: "sync.record", Message: err.Error()}})
		}
		artifacts.SyncRecordID = recordID
		rollbackPaths = append(rollbackPaths, recordPath)
	} else {
		warnings = append(warnings, degradedWarning("Sync refresh unavailable", observation.Sync.Degraded))
	}

	if artifacts.CIRecordID == "" && artifacts.SyncRecordID == "" {
		return RefreshResult{
			OK:       false,
			Command:  "evidence refresh",
			Summary:  "Remote evidence refresh could not write CI or sync evidence.",
			Warnings: warnings,
			Errors: []CommandError{{
				Path:    "remote",
				Message: "remote PR facts were unavailable; use manual harness evidence submit fallback",
			}},
			NextAction: manualRefreshFallbackActions(),
		}
	}

	resultOut := RefreshResult{
		OK:        true,
		Command:   "evidence refresh",
		Summary:   refreshSummary(artifacts),
		Artifacts: artifacts,
		Warnings:  warnings,
		NextAction: []NextAction{
			{Command: nil, Description: "Run harness status to refresh the archived candidate summary and next actions."},
		},
	}
	return s.finalizeRefreshMutation(resultOut, rollbackPaths)
}

func LoadLatestCI(workdir, planStem string, revision int) (*CIRecord, error) {
	return loadLatestRecord[CIRecord](workdir, planStem, "ci", revision)
}

func LoadLatestPublish(workdir, planStem string, revision int) (*PublishRecord, error) {
	return loadLatestRecord[PublishRecord](workdir, planStem, "publish", revision)
}

func LoadLatestSync(workdir, planStem string, revision int) (*SyncRecord, error) {
	return loadLatestRecord[SyncRecord](workdir, planStem, "sync", revision)
}

func loadRecord[T any](workdir, relPath string) (*T, error) {
	if strings.TrimSpace(relPath) == "" {
		return nil, nil
	}
	path := filepath.Join(workdir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record T
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &record, nil
}

func loadLatestRecord[T any](workdir, planStem, kind string, revision int) (*T, error) {
	if strings.TrimSpace(planStem) == "" {
		return nil, nil
	}
	dir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "evidence", kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	maxID := 0
	var latestRecord *T
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := recordIDPattern.FindStringSubmatch(entry.Name())
		if matches == nil || matches[1] != kind {
			continue
		}
		n, err := strconv.Atoi(matches[2])
		if err != nil || n <= maxID {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var header struct {
			Revision int `json:"revision"`
		}
		if err := json.Unmarshal(data, &header); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if revision > 0 && header.Revision != revision {
			continue
		}
		var record T
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		maxID = n
		latestRecord = &record
	}
	return latestRecord, nil
}

func validateCIInput(input CIInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "pending", "success", "failed":
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be pending, success, failed, or not_applied"}}
	}
}

func validatePublishInput(input PublishInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "recorded":
		if strings.TrimSpace(input.PRURL) == "" {
			return []CommandError{{Path: "input.pr_url", Message: "pr_url is required when status=recorded"}}
		}
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be recorded or not_applied"}}
	}
}

func validateSyncInput(input SyncInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "fresh", "stale", "conflicted":
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be fresh, stale, conflicted, or not_applied"}}
	}
}

func nextRecordLocation(workdir, planStem, kind string) (string, string, error) {
	dir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "evidence", kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	maxID := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := recordIDPattern.FindStringSubmatch(entry.Name())
		if matches == nil || matches[1] != kind {
			continue
		}
		n, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		if n > maxID {
			maxID = n
		}
	}
	recordID := fmt.Sprintf("%s-%03d", kind, maxID+1)
	return recordID, filepath.Join(dir, recordID+".json"), nil
}

func successResult(planPath, kind, recordID, summary string) Result {
	return Result{
		OK:      true,
		Command: "evidence submit",
		Summary: summary,
		Artifacts: &Artifacts{
			PlanPath: planPath,
			RecordID: recordID,
			Kind:     kind,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Run harness status to refresh the archived candidate summary and next actions."},
		},
	}
}

func invalidInputResult(kind string, err error) Result {
	return errorResult("evidence submit", fmt.Sprintf("%s evidence input is invalid.", kind), []CommandError{{Path: "input", Message: err.Error()}})
}

func invalidInputIssuesResult(kind string, issues []CommandError) Result {
	return Result{
		OK:      false,
		Command: "evidence submit",
		Summary: fmt.Sprintf("%s evidence input is invalid.", kind),
		Errors:  issues,
	}
}

func errorResult(command, summary string, errors []CommandError) Result {
	return Result{
		OK:      false,
		Command: command,
		Summary: summary,
		Errors:  errors,
	}
}

func refreshErrorResult(summary string, errors []CommandError) RefreshResult {
	return RefreshResult{
		OK:         false,
		Command:    "evidence refresh",
		Summary:    summary,
		Errors:     errors,
		NextAction: manualRefreshFallbackActions(),
	}
}

func (s Service) finalizeMutation(result Result, rollback func() []CommandError) Result {
	if !result.OK || s.AfterMutation == nil {
		if result.OK && s.AfterSuccess != nil {
			s.AfterSuccess(result)
		}
		return result
	}
	if err := s.AfterMutation(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return errorResult(result.Command, "Unable to record the timeline event for the successful command result.", issues)
	}
	if s.AfterSuccess != nil {
		s.AfterSuccess(result)
	}
	return result
}

func (s Service) finalizeRefreshMutation(result RefreshResult, rollbackPaths []string) RefreshResult {
	if !result.OK || s.AfterRefresh == nil {
		return result
	}
	if err := s.AfterRefresh(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		for _, path := range rollbackPaths {
			issues = append(issues, rollbackEvidenceMutation(path)...)
		}
		return refreshErrorResult("Unable to record the timeline event for the successful evidence refresh.", issues)
	}
	return result
}

func firstCheckURL(checks []remote.CheckRun) string {
	for _, check := range checks {
		if strings.TrimSpace(check.Link) != "" {
			return strings.TrimSpace(check.Link)
		}
	}
	return ""
}

func degradedWarning(prefix string, degradation remote.Degradation) string {
	message := strings.TrimSpace(degradation.Message)
	if message == "" {
		message = strings.TrimSpace(degradation.Code)
	}
	if message == "" {
		message = "remote facts are unavailable"
	}
	return prefix + ": " + message
}

func refreshSummary(artifacts *RefreshArtifacts) string {
	switch {
	case artifacts != nil && artifacts.CIRecordID != "" && artifacts.SyncRecordID != "":
		return "Refreshed CI and sync evidence from the recorded pull request."
	case artifacts != nil && artifacts.CIRecordID != "":
		return "Refreshed CI evidence from the recorded pull request; sync evidence still needs manual follow-up."
	case artifacts != nil && artifacts.SyncRecordID != "":
		return "Refreshed sync evidence from the recorded pull request; CI evidence still needs manual follow-up."
	default:
		return "Remote evidence refresh did not write evidence."
	}
}

func manualRefreshFallbackActions() []NextAction {
	return []NextAction{
		{Command: strPtr("harness evidence submit --kind ci --input <json>"), Description: "Manually record CI evidence when remote checks cannot be refreshed."},
		{Command: strPtr("harness evidence submit --kind sync --input <json>"), Description: "Manually record sync evidence when remote merge state cannot be refreshed."},
	}
}

func strPtr(value string) *string {
	return &value
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s Service) loadCurrentArchivedPlan() (string, string, string, *runstate.State, string, func(), *Result) {
	release := func() {}
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", "", "", nil, "", release, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	release, err = runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Another local state mutation is already in progress.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}

	planPath, err = plan.DetectCurrentPathLocked(s.Workdir, planStem)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	doc, err := plan.LoadFile(planPath)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	if doc.DerivedPlanStatus() != "archived" || doc.DerivedLifecycle(state) != "awaiting_merge_approval" {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Evidence commands require the current archived candidate.",
			Errors: []CommandError{{
				Path:    "plan.lifecycle",
				Message: fmt.Sprintf("current plan is status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
			}},
		}
	}
	if state != nil && state.Land != nil &&
		strings.TrimSpace(state.Land.LandedAt) != "" &&
		strings.TrimSpace(state.Land.CompletedAt) == "" {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Evidence commands are not allowed after merge confirmation enters required post-merge bookkeeping.",
			Errors: []CommandError{{
				Path:    "state.land",
				Message: "current archived candidate is already in required post-merge bookkeeping",
			}},
		}
	}
	return planPath, relPlanPath, planStem, state, statePath, release, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func cloneState(state *runstate.State) *runstate.State {
	if state == nil {
		return nil
	}
	cloned := *state
	if state.ActiveReviewRound != nil {
		round := *state.ActiveReviewRound
		cloned.ActiveReviewRound = &round
	}
	if state.Reopen != nil {
		reopen := *state.Reopen
		cloned.Reopen = &reopen
	}
	if state.Land != nil {
		land := *state.Land
		cloned.Land = &land
	}
	return &cloned
}

func rollbackEvidenceMutation(recordPath string) []CommandError {
	issues := make([]CommandError, 0, 1)
	if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "record", Message: fmt.Sprintf("rollback evidence artifact: %v", err)})
	}
	return issues
}
