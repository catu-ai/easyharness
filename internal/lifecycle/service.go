package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/runstate"
	"gopkg.in/yaml.v3"
)

type Service struct {
	Workdir string
	Now     func() time.Time
}

type Result struct {
	OK         bool           `json:"ok"`
	Command    string         `json:"command"`
	Summary    string         `json:"summary"`
	State      State          `json:"state"`
	Artifacts  *Artifacts     `json:"artifacts,omitempty"`
	NextAction []NextAction   `json:"next_actions"`
	Errors     []CommandError `json:"errors,omitempty"`
}

type State struct {
	PlanStatus string `json:"plan_status"`
	Lifecycle  string `json:"lifecycle"`
	Revision   int    `json:"revision"`
}

type Artifacts struct {
	FromPlanPath    string `json:"from_plan_path"`
	ToPlanPath      string `json:"to_plan_path"`
	LocalStatePath  string `json:"local_state_path,omitempty"`
	CurrentPlanPath string `json:"current_plan_path,omitempty"`
}

type NextAction struct {
	Command     *string `json:"command"`
	Description string  `json:"description"`
}

type CommandError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type editablePlan struct {
	Frontmatter plan.Frontmatter
	Body        string
}

func (s Service) Archive() Result {
	now := s.now()
	currentPath, doc, editable, planStem, relCurrentPath, state, statePath, result := s.loadCurrentPlan()
	if result != nil {
		result.Command = "archive"
		return *result
	}
	if doc.Frontmatter.Status != "active" || doc.Frontmatter.Lifecycle != "executing" {
		return errorResult("archive", "Current plan is not archive-ready.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("archive requires status=active and lifecycle=executing, got status=%q lifecycle=%q", doc.Frontmatter.Status, doc.Frontmatter.Lifecycle),
		}})
	}
	if !doc.AllAcceptanceChecked() {
		return errorResult("archive", "Current plan is not archive-ready.", []CommandError{{Path: "section.Acceptance Criteria", Message: "all acceptance criteria must be checked before archive"}})
	}
	if !doc.AllStepsCompleted() {
		return errorResult("archive", "Current plan is not archive-ready.", []CommandError{{Path: "section.Work Breakdown", Message: "all steps must be completed before archive"}})
	}
	if doc.HasPendingArchivePlaceholders() {
		return errorResult("archive", "Current plan still contains archive placeholders.", []CommandError{{Path: "sections", Message: "replace every PENDING_UNTIL_ARCHIVE token before archive"}})
	}
	if doc.CompletedStepsHavePendingPlaceholders() {
		return errorResult("archive", "Current plan still contains completed-step placeholders.", []CommandError{{Path: "steps", Message: "replace every PENDING_STEP_EXECUTION and PENDING_STEP_REVIEW token before archive"}})
	}
	if issues := archiveStateIssues(s.Workdir, planStem, doc.Frontmatter.Revision, state); len(issues) > 0 {
		return errorResult("archive", "Current plan is not archive-ready.", issues)
	}

	archiveSummary := doc.SectionText("Archive Summary")
	missingLabels := missingArchiveSummaryLabels(archiveSummary, []string{"PR", "Ready", "Merge Handoff"})
	if len(missingLabels) > 0 {
		return errorResult("archive", "Archive Summary is missing required fields.", []CommandError{{
			Path:    "section.Archive Summary",
			Message: fmt.Sprintf("add archive summary lines for: %s", strings.Join(missingLabels, ", ")),
		}})
	}

	archiveSummary = stripArchiveSummaryLines(archiveSummary, []string{"Archived At", "Revision"})
	archiveSummary = strings.TrimSpace(strings.Join([]string{
		fmt.Sprintf("- Archived At: %s", now.Format(time.RFC3339)),
		fmt.Sprintf("- Revision: %d", doc.Frontmatter.Revision),
		archiveSummary,
	}, "\n"))

	body, err := replaceTopLevelSection(editable.Body, "Archive Summary", archiveSummary)
	if err != nil {
		return errorResult("archive", "Unable to update Archive Summary.", []CommandError{{Path: "section.Archive Summary", Message: err.Error()}})
	}

	editable.Frontmatter.Status = "archived"
	editable.Frontmatter.Lifecycle = "awaiting_merge_approval"
	editable.Frontmatter.UpdatedAt = now.Format(time.RFC3339)

	targetPath := filepath.Join(s.Workdir, "docs", "plans", "archived", filepath.Base(currentPath))
	if _, err := os.Stat(targetPath); err == nil {
		return errorResult("archive", "Archived target path already exists.", []CommandError{{Path: "path", Message: fmt.Sprintf("target already exists: %s", targetPath)}})
	}

	content, err := renderEditablePlan(editable.Frontmatter, body)
	if err != nil {
		return errorResult("archive", "Unable to render archived plan.", []CommandError{{Path: "frontmatter", Message: err.Error()}})
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errorResult("archive", "Unable to create archived plan directory.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return errorResult("archive", "Unable to write archived plan.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if lint := plan.LintFile(targetPath); !lint.OK {
		_ = os.Remove(targetPath)
		return errorResult("archive", "Archived plan did not pass validation.", lintErrorsToCommandErrors(lint.Errors))
	}
	if err := os.Remove(currentPath); err != nil {
		_ = os.Remove(targetPath)
		return errorResult("archive", "Unable to remove the active plan after archiving.", []CommandError{{Path: "path", Message: err.Error()}})
	}

	relTargetPath, err := filepath.Rel(s.Workdir, targetPath)
	if err != nil {
		return errorResult("archive", "Unable to relativize archived plan path.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	relTargetPath = filepath.ToSlash(relTargetPath)

	currentPlanPath, err := runstate.SaveCurrentPlan(s.Workdir, relTargetPath)
	if err != nil {
		return errorResult("archive", "Unable to update current-plan pointer.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	if state != nil {
		state.PlanPath = relTargetPath
		state.PlanStem = planStem
		statePath, err = runstate.SaveState(s.Workdir, planStem, state)
		if err != nil {
			return errorResult("archive", "Unable to update local state after archiving.", []CommandError{{Path: "state", Message: err.Error()}})
		}
	}

	return Result{
		OK:      true,
		Command: "archive",
		Summary: "Plan archived and frozen for merge handoff.",
		State: State{
			PlanStatus: "archived",
			Lifecycle:  "awaiting_merge_approval",
			Revision:   doc.Frontmatter.Revision,
		},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			ToPlanPath:      relTargetPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Commit and push the archived plan move before treating the candidate as truly waiting for merge approval."},
			{Command: nil, Description: "Wait for human merge approval or merge manually from the PR once checks are green."},
			{Command: strPtr("harness reopen"), Description: "Reopen the plan if new feedback or remote changes invalidate the archived candidate."},
		},
	}
}

func (s Service) Reopen() Result {
	now := s.now()
	currentPath, doc, editable, planStem, relCurrentPath, state, statePath, result := s.loadCurrentPlan()
	if result != nil {
		result.Command = "reopen"
		return *result
	}
	if doc.Frontmatter.Status != "archived" || doc.Frontmatter.Lifecycle != "awaiting_merge_approval" {
		return errorResult("reopen", "Current plan is not archived.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("reopen requires status=archived and lifecycle=awaiting_merge_approval, got status=%q lifecycle=%q", doc.Frontmatter.Status, doc.Frontmatter.Lifecycle),
		}})
	}

	body, err := replaceTopLevelSection(editable.Body, "Validation Summary", "PENDING_UNTIL_ARCHIVE")
	if err != nil {
		return errorResult("reopen", "Unable to reset Validation Summary.", []CommandError{{Path: "section.Validation Summary", Message: err.Error()}})
	}
	body, err = replaceTopLevelSection(body, "Review Summary", "PENDING_UNTIL_ARCHIVE")
	if err != nil {
		return errorResult("reopen", "Unable to reset Review Summary.", []CommandError{{Path: "section.Review Summary", Message: err.Error()}})
	}
	body, err = replaceTopLevelSection(body, "Archive Summary", "PENDING_UNTIL_ARCHIVE")
	if err != nil {
		return errorResult("reopen", "Unable to reset Archive Summary.", []CommandError{{Path: "section.Archive Summary", Message: err.Error()}})
	}
	body, err = replaceTopLevelSection(body, "Outcome Summary", strings.Join([]string{
		"### Delivered",
		"",
		"PENDING_UNTIL_ARCHIVE",
		"",
		"### Not Delivered",
		"",
		"PENDING_UNTIL_ARCHIVE",
		"",
		"### Follow-Up Issues",
		"",
		"NONE",
	}, "\n"))
	if err != nil {
		return errorResult("reopen", "Unable to reset Outcome Summary.", []CommandError{{Path: "section.Outcome Summary", Message: err.Error()}})
	}

	editable.Frontmatter.Status = "active"
	editable.Frontmatter.Lifecycle = "executing"
	editable.Frontmatter.Revision++
	editable.Frontmatter.UpdatedAt = now.Format(time.RFC3339)

	targetPath := filepath.Join(s.Workdir, "docs", "plans", "active", filepath.Base(currentPath))
	if _, err := os.Stat(targetPath); err == nil {
		return errorResult("reopen", "Active target path already exists.", []CommandError{{Path: "path", Message: fmt.Sprintf("target already exists: %s", targetPath)}})
	}

	content, err := renderEditablePlan(editable.Frontmatter, body)
	if err != nil {
		return errorResult("reopen", "Unable to render reopened plan.", []CommandError{{Path: "frontmatter", Message: err.Error()}})
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errorResult("reopen", "Unable to create active plan directory.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return errorResult("reopen", "Unable to write reopened plan.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if lint := plan.LintFile(targetPath); !lint.OK {
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Reopened plan did not pass validation.", lintErrorsToCommandErrors(lint.Errors))
	}
	if err := os.Remove(currentPath); err != nil {
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Unable to remove the archived plan after reopening.", []CommandError{{Path: "path", Message: err.Error()}})
	}

	relTargetPath, err := filepath.Rel(s.Workdir, targetPath)
	if err != nil {
		return errorResult("reopen", "Unable to relativize active plan path.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	relTargetPath = filepath.ToSlash(relTargetPath)

	currentPlanPath, err := runstate.SaveCurrentPlan(s.Workdir, relTargetPath)
	if err != nil {
		return errorResult("reopen", "Unable to update current-plan pointer.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	if state != nil {
		state.PlanPath = relTargetPath
		state.PlanStem = planStem
		state.ActiveReviewRound = nil
		state.LatestCI = nil
		state.Sync = nil
		statePath, err = runstate.SaveState(s.Workdir, planStem, state)
		if err != nil {
			return errorResult("reopen", "Unable to update local state after reopen.", []CommandError{{Path: "state", Message: err.Error()}})
		}
	}

	return Result{
		OK:      true,
		Command: "reopen",
		Summary: "Archived plan reopened for active execution.",
		State: State{
			PlanStatus: "active",
			Lifecycle:  "executing",
			Revision:   editable.Frontmatter.Revision,
		},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			ToPlanPath:      relTargetPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Review the feedback or remote change that caused reopen."},
			{Command: nil, Description: "Update the plan content if scope or acceptance criteria changed."},
			{Command: nil, Description: "Continue the inferred current step, or set awaiting_plan_approval if the plan contract needs fresh approval."},
		},
	}
}

func (s Service) loadCurrentPlan() (string, *plan.Document, *editablePlan, string, string, *runstate.State, string, *Result) {
	currentPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", nil, nil, "", "", nil, "", &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	doc, err := plan.LoadFile(currentPath)
	if err != nil {
		return "", nil, nil, "", "", nil, "", &Result{
			OK:      false,
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	editable, err := loadEditablePlan(currentPath)
	if err != nil {
		return "", nil, nil, "", "", nil, "", &Result{
			OK:      false,
			Summary: "Unable to load the editable plan representation.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))
	relCurrentPath, err := filepath.Rel(s.Workdir, currentPath)
	if err != nil {
		return "", nil, nil, "", "", nil, "", &Result{
			OK:      false,
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "path", Message: err.Error()}},
		}
	}
	relCurrentPath = filepath.ToSlash(relCurrentPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return "", nil, nil, "", "", nil, "", &Result{
			OK:      false,
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	return currentPath, doc, editable, planStem, relCurrentPath, state, statePath, nil
}

func loadEditablePlan(path string) (*editablePlan, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}
	var frontmatter plan.Frontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &editablePlan{Frontmatter: frontmatter, Body: strings.TrimLeft(body, "\n")}, nil
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("file must start with YAML frontmatter delimited by ---")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", fmt.Errorf("frontmatter is missing a closing --- delimiter")
}

func replaceTopLevelSection(body, sectionName, newContent string) (string, error) {
	header := "## " + sectionName + "\n\n"
	start := strings.Index(body, header)
	if start == -1 {
		return "", fmt.Errorf("missing ## %s section", sectionName)
	}

	searchStart := start + len(header)
	nextRelative := strings.Index(body[searchStart:], "\n## ")
	end := len(body)
	if nextRelative != -1 {
		end = searchStart + nextRelative + 1
	}

	replacement := fmt.Sprintf("## %s\n\n%s\n\n", sectionName, strings.TrimSpace(newContent))
	return body[:start] + replacement + strings.TrimLeft(body[end:], "\n"), nil
}

func renderEditablePlan(frontmatter plan.Frontmatter, body string) (string, error) {
	data, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\n%s---\n\n%s", string(data), strings.TrimLeft(body, "\n")), nil
}

func missingArchiveSummaryLabels(content string, labels []string) []string {
	missing := make([]string, 0)
	for _, label := range labels {
		if !strings.Contains(content, "- "+label+":") {
			missing = append(missing, label)
		}
	}
	return missing
}

func stripArchiveSummaryLines(content string, labels []string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		keep := true
		for _, label := range labels {
			if strings.HasPrefix(strings.TrimSpace(line), "- "+label+":") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func lintErrorsToCommandErrors(issues []plan.LintIssue) []CommandError {
	errors := make([]CommandError, 0, len(issues))
	for _, issue := range issues {
		errors = append(errors, CommandError{Path: issue.Path, Message: issue.Message})
	}
	return errors
}

func errorResult(command, summary string, errors []CommandError) Result {
	return Result{
		OK:      false,
		Command: command,
		Summary: summary,
		Errors:  errors,
	}
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func strPtr(value string) *string {
	return &value
}

func archiveStateIssues(workdir, planStem string, revision int, state *runstate.State) []CommandError {
	issues := make([]CommandError, 0)
	if state == nil || state.ActiveReviewRound == nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: requiredReviewMessage(revision),
		})
		return issues
	}

	if !state.ActiveReviewRound.Aggregated {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "aggregate or clear the active review round before archive",
		})
	}
	decision, known, err := runstate.EffectiveReviewDecision(workdir, planStem, state.ActiveReviewRound)
	if err != nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("unable to read the latest aggregate artifact for %s: %v", state.ActiveReviewRound.RoundID, err),
		})
		return issues
	}
	if !known {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "latest review decision is unknown; rerun or re-aggregate the latest review before archive",
		})
	}
	if known && decision != "pass" {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("latest review decision %q is not archive-ready; fix findings or rerun review", decision),
		})
	}
	if revision <= 1 && state.ActiveReviewRound.Kind != "full" {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "revision 1 requires a passing full review before archive",
		})
	}
	if state.LatestCI != nil && !ciStatusAllowsArchive(state.LatestCI.Status) {
		issues = append(issues, CommandError{
			Path:    "state.latest_ci",
			Message: fmt.Sprintf("latest CI status %q is not archive-ready; wait for green CI or fix failures first", state.LatestCI.Status),
		})
	}
	if state.Sync != nil {
		if state.Sync.Conflicts {
			issues = append(issues, CommandError{
				Path:    "state.sync",
				Message: "resolve merge conflicts before archive",
			})
		}
		if freshnessBlocksArchive(state.Sync.Freshness) {
			issues = append(issues, CommandError{
				Path:    "state.sync",
				Message: fmt.Sprintf("remote freshness %q is not archive-ready; refresh remote state before archive", state.Sync.Freshness),
			})
		}
	}
	return issues
}

func ciStatusAllowsArchive(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "passed", "green", "succeeded":
		return true
	default:
		return false
	}
}

func freshnessBlocksArchive(freshness string) bool {
	switch strings.ToLower(strings.TrimSpace(freshness)) {
	case "", "fresh":
		return false
	default:
		return true
	}
}

func requiredReviewMessage(revision int) string {
	if revision <= 1 {
		return "revision 1 requires a passing full review before archive"
	}
	return "archive requires a passing aggregated review before archive"
}
