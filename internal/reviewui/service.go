package reviewui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

type Service struct {
	Workdir string
}

type Result = contracts.ReviewResult
type Round = contracts.ReviewRoundView
type Reviewer = contracts.ReviewReviewerView
type ErrorDetail = contracts.ErrorDetail
type Manifest = contracts.ReviewManifest
type ManifestAssignment = contracts.ReviewAssignment
type Ledger = contracts.ReviewLedger
type LedgerAssignment = contracts.ReviewLedgerAssignment
type Submission = contracts.ReviewSubmission
type Aggregate = contracts.ReviewAggregate
type FindingView = contracts.ReviewFindingView

type artifact struct {
	Status string
}

type reviewerState struct {
	Slot        string
	Role        string
	Status      string
	SubmittedAt string
	Name        string
	Summary     string
	Warnings    []string
}

func (s Service) Read() Result {
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			return Result{
				OK:       true,
				Resource: "review",
				Summary:  "No current plan review data is available in this worktree.",
				Rounds:   []Round{},
			}
		}
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to determine the current plan for review loading.",
			Errors:   []ErrorDetail{{Path: "plan", Message: "Unable to determine the current plan for review loading."}},
			Rounds:   []Round{},
		}
	}

	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to determine the current plan path for review loading.",
			Errors:   []ErrorDetail{{Path: "plan", Message: "Unable to determine the current plan path for review loading."}},
			Rounds:   []Round{},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	if !isSupportedReviewPlanPath(s.Workdir, relPlanPath) {
		return Result{
			OK:       true,
			Resource: "review",
			Summary:  "Review data is only shown for the current tracked plan.",
			Rounds:   []Round{},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
	state, _, err := runstate.LoadState(s.Workdir, planStem)
	warnings := make([]string, 0, 2)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read local review state for %s; active-round hints may be incomplete.", planStem))
	}
	if isArchivedReviewPlanPath(s.Workdir, relPlanPath) && archivedReviewHiddenDuringLand(state) {
		return Result{
			OK:       true,
			Resource: "review",
			Summary:  "Review data is hidden once required post-merge bookkeeping begins.",
			Rounds:   []Round{},
			Warnings: warnings,
		}
	}

	reviewsDir := runstate.ReviewsDir(s.Workdir, planStem)
	roundIDs, discoverWarnings, discoverErr := discoverRoundIDs(reviewsDir, state)
	warnings = append(warnings, discoverWarnings...)
	if discoverErr != nil {
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to enumerate review rounds for the current plan.",
			Errors:   []ErrorDetail{{Path: "reviews", Message: "Unable to enumerate review rounds for the current plan."}},
			Rounds:   []Round{},
		}
	}

	rounds := make([]Round, 0, len(roundIDs))
	activeRoundID := ""
	if state != nil && state.ActiveReviewRound != nil {
		activeRoundID = strings.TrimSpace(state.ActiveReviewRound.RoundID)
	}
	for _, roundID := range roundIDs {
		rounds = append(rounds, s.readRound(planStem, roundID, activeRoundID))
	}
	sortRounds(rounds)

	summary := "No review rounds recorded yet for the current plan."
	if len(rounds) > 0 {
		summary = fmt.Sprintf("Loaded %d review round(s) for %s.", len(rounds), filepath.Base(relPlanPath))
	}

	return Result{
		OK:       true,
		Resource: "review",
		Summary:  summary,
		Rounds:   rounds,
		Warnings: warnings,
	}
}

func archivedReviewHiddenDuringLand(state *runstate.State) bool {
	if state == nil {
		return false
	}
	return state.Land != nil &&
		strings.TrimSpace(state.Land.LandedAt) != "" &&
		strings.TrimSpace(state.Land.CompletedAt) == ""
}

func isSupportedReviewPlanPath(workdir, relPlanPath string) bool {
	switch plan.PathKindFor(filepath.Join(workdir, filepath.FromSlash(relPlanPath))) {
	case "active", "archived":
		return true
	default:
		return false
	}
}

func isArchivedReviewPlanPath(workdir, relPlanPath string) bool {
	return plan.PathKindFor(filepath.Join(workdir, filepath.FromSlash(relPlanPath))) == "archived"
}

func discoverRoundIDs(reviewsDir string, state *runstate.State) ([]string, []string, error) {
	roundSet := map[string]bool{}
	warnings := []string{}

	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, warnings, err
		}
		if state != nil && state.ActiveReviewRound != nil && strings.TrimSpace(state.ActiveReviewRound.RoundID) != "" {
			roundID := strings.TrimSpace(state.ActiveReviewRound.RoundID)
			roundSet[roundID] = true
			warnings = append(warnings, fmt.Sprintf("Active review round %s is tracked in local state, but the review directory is missing.", roundID))
		}
		return sortedRoundIDs(roundSet), warnings, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		roundSet[name] = true
	}

	if state != nil && state.ActiveReviewRound != nil && strings.TrimSpace(state.ActiveReviewRound.RoundID) != "" {
		roundID := strings.TrimSpace(state.ActiveReviewRound.RoundID)
		if !roundSet[roundID] {
			warnings = append(warnings, fmt.Sprintf("Active review round %s is tracked in local state, but no matching round directory was found.", roundID))
		}
		roundSet[roundID] = true
	}

	return sortedRoundIDs(roundSet), warnings, nil
}

func sortedRoundIDs(values map[string]bool) []string {
	roundIDs := make([]string, 0, len(values))
	for roundID := range values {
		roundIDs = append(roundIDs, roundID)
	}
	sort.Slice(roundIDs, func(i, j int) bool {
		return compareRoundIDs(roundIDs[i], roundIDs[j]) < 0
	})
	return roundIDs
}

func compareRoundIDs(left, right string) int {
	leftSeq, leftOK := parseRoundSequence(left)
	rightSeq, rightOK := parseRoundSequence(right)
	switch {
	case leftOK && rightOK && leftSeq != rightSeq:
		if leftSeq > rightSeq {
			return -1
		}
		return 1
	case leftOK && !rightOK:
		return -1
	case !leftOK && rightOK:
		return 1
	case left == right:
		return 0
	case left > right:
		return -1
	default:
		return 1
	}
}

func parseRoundSequence(roundID string) (int, bool) {
	trimmed := strings.TrimSpace(roundID)
	if !strings.HasPrefix(trimmed, "review-") {
		return 0, false
	}
	remainder := strings.TrimPrefix(trimmed, "review-")
	sequence, _, found := strings.Cut(remainder, "-")
	if !found || sequence == "" {
		return 0, false
	}
	value, err := strconv.Atoi(sequence)
	if err != nil {
		return 0, false
	}
	return value, true
}

func sortRounds(rounds []Round) {
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].IsActive != rounds[j].IsActive {
			return rounds[i].IsActive
		}
		return compareRoundIDs(rounds[i].RoundID, rounds[j].RoundID) < 0
	})
}

func (s Service) readRound(planStem, roundID, activeRoundID string) Round {
	roundDir := runstate.ReviewRoundDir(s.Workdir, planStem, roundID)
	manifestPath := filepath.Join(roundDir, "manifest.json")
	ledgerPath := filepath.Join(roundDir, "ledger.json")
	aggregatePath := filepath.Join(roundDir, "aggregate.json")

	manifestArtifact, manifest, manifestWarning := readJSONArtifact[Manifest]("Review metadata", manifestPath, validateManifestArtifact)
	ledgerArtifact, ledger, ledgerWarning := readJSONArtifact[Ledger]("Review progress", ledgerPath, validateLedgerArtifact)
	decisionArtifact, aggregate, decisionWarning := readJSONArtifact[Aggregate]("Decision", aggregatePath, validateAggregateArtifact)

	warnings := make([]string, 0, 8)
	appendWarning := func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			return
		}
		warnings = append(warnings, message)
	}

	appendWarning(manifestWarning)
	appendWarning(ledgerWarning)
	if decisionWarning != "" && decisionArtifact.Status != "missing" {
		appendWarning(decisionWarning)
	}

	round := Round{
		RoundID:  roundID,
		Status:   "unknown",
		IsActive: roundID == strings.TrimSpace(activeRoundID),
	}

	if manifest != nil {
		round.Kind = manifest.Kind
		round.AnchorSHA = manifest.AnchorSHA
		round.ReviewedHeadSHA = manifest.ReviewedHeadSHA
		if manifest.Repair != nil {
			round.RepairsRoundID = manifest.Repair.RoundID
			round.RepairFindingIDs = append([]string(nil), manifest.Repair.FindingIDs...)
		}
		round.Step = manifest.Step
		round.Revision = manifest.Revision
		round.ReviewTitle = manifest.ReviewTitle
		round.CreatedAt = manifest.CreatedAt
	}
	if ledger != nil {
		if round.Kind == "" {
			round.Kind = ledger.Kind
		}
		round.UpdatedAt = ledger.UpdatedAt
	}
	if aggregate != nil {
		if round.Kind == "" {
			round.Kind = aggregate.Kind
		}
		if round.Step == nil {
			round.Step = aggregate.Step
		}
		if round.Revision == 0 {
			round.Revision = aggregate.Revision
		}
		if round.ReviewTitle == "" {
			round.ReviewTitle = aggregate.ReviewTitle
		}
		round.DecidedAt = aggregate.AggregatedAt
		round.Decision = aggregate.Decision
		if round.ReviewedHeadSHA == "" {
			round.ReviewedHeadSHA = aggregate.ReviewedHeadSHA
		}
		if round.RepairsRoundID == "" && aggregate.Repair != nil {
			round.RepairsRoundID = aggregate.Repair.RoundID
			round.RepairFindingIDs = append([]string(nil), aggregate.Repair.FindingIDs...)
		}
		if aggregate.UnresolvedBlockingFindings != nil {
			round.BlockingFindings = findingViews(aggregate.UnresolvedBlockingFindings)
		} else {
			round.BlockingFindings = findingViews(aggregate.BlockingFindings)
		}
		round.NonBlockingFindings = findingViews(aggregate.NonBlockingFindings)
		round.ResolvedFindingIDs = append([]string(nil), aggregate.ResolvedFindingIDs...)
		round.UnresolvedFindingIDs = append([]string(nil), aggregate.UnresolvedFindingIDs...)
		switch {
		case len(round.UnresolvedFindingIDs) > 0 || len(round.BlockingFindings) > 0 || aggregate.Decision == "changes_requested":
			round.CoverageStatus = "blocked"
		case aggregate.Decision == "pass":
			round.CoverageStatus = "clean"
		default:
			round.CoverageStatus = "pending"
		}
	}

	reviewers, reviewerWarnings := s.readReviewers(roundDir, manifest, ledger)
	warnings = append(warnings, reviewerWarnings...)

	pendingReviewers := 0
	for _, reviewer := range reviewers {
		if normalizeSlotStatus(reviewer.Status) != "submitted" {
			pendingReviewers++
		}
	}
	round.Reviewer = selectReviewer(reviewers)

	status, summary := resolveRoundStatus(round, len(reviewers), pendingReviewers, manifestArtifact, ledgerArtifact, decisionArtifact)
	round.Status = status
	round.StatusSummary = summary
	if manifestArtifact.Status == "invalid" || ledgerArtifact.Status == "invalid" || decisionArtifact.Status == "invalid" {
		appendWarning("One or more review artifacts are malformed; the round is shown conservatively.")
	}
	if manifestArtifact.Status == "missing" {
		appendWarning("Review metadata is missing, so round context may be incomplete.")
	}
	if ledgerArtifact.Status == "missing" && len(reviewers) > 0 {
		appendWarning("Review progress is missing, so reviewer state is inferred conservatively.")
	}
	if decisionArtifact.Status == "missing" && pendingReviewers == 0 && len(reviewers) > 0 {
		appendWarning("Reviewer results are present, but the review decision is still missing.")
	}
	round.Warnings = dedupeStrings(warnings)
	return round
}

func (s Service) readReviewers(roundDir string, manifest *Manifest, ledger *Ledger) ([]reviewerState, []string) {
	slotOrder := make([]string, 0)
	slotSeen := map[string]bool{}
	manifestBySlot := map[string]ManifestAssignment{}
	ledgerBySlot := map[string]LedgerAssignment{}
	submissionPathBySlot := map[string]string{}
	addSlot := func(slot string) {
		slot = strings.TrimSpace(slot)
		if slot == "" || slotSeen[slot] {
			return
		}
		slotOrder = append(slotOrder, slot)
		slotSeen[slot] = true
	}

	if manifest != nil {
		for _, item := range manifest.Assignments {
			slot := strings.TrimSpace(item.Slot)
			if slot == "" {
				continue
			}
			addSlot(slot)
			manifestBySlot[slot] = item
			if path := strings.TrimSpace(item.SubmissionPath); path != "" {
				submissionPathBySlot[slot] = path
			}
		}
	}
	if ledger != nil {
		for _, item := range ledger.Assignments {
			slot := strings.TrimSpace(item.Slot)
			if slot == "" {
				continue
			}
			addSlot(slot)
			ledgerBySlot[slot] = item
			if path := strings.TrimSpace(item.SubmissionPath); path != "" {
				submissionPathBySlot[slot] = path
			}
		}
	}
	discoveredSubmissionPaths, discoveryWarnings := discoverSubmissionPaths(filepath.Join(roundDir, "submissions"))
	discoveredSlots := make([]string, 0, len(discoveredSubmissionPaths))
	for slot := range discoveredSubmissionPaths {
		discoveredSlots = append(discoveredSlots, slot)
	}
	sort.Strings(discoveredSlots)
	for _, slot := range discoveredSlots {
		path := discoveredSubmissionPaths[slot]
		addSlot(slot)
		if _, exists := submissionPathBySlot[slot]; !exists {
			submissionPathBySlot[slot] = path
		}
	}

	reviewers := make([]reviewerState, 0, len(slotOrder))
	warnings := make([]string, 0, len(slotOrder)+len(discoveryWarnings))
	warnings = append(warnings, discoveryWarnings...)
	for _, slot := range slotOrder {
		reviewer := reviewerState{Slot: slot}
		ledgerClaimedSubmitted := false
		ledgerStatusWarning := ""
		hasLedgerEntry := false
		artifactPath := filepath.Join(roundDir, "submissions", slot, "submission.json")
		if path, ok := submissionPathBySlot[slot]; ok && strings.TrimSpace(path) != "" {
			artifactPath = path
		}
		if item, ok := manifestBySlot[slot]; ok {
			reviewer.Role = item.Role
		}
		if item, ok := ledgerBySlot[slot]; ok {
			hasLedgerEntry = true
			if reviewer.Role == "" {
				reviewer.Role = item.Role
			}
			reviewer.Status = item.Status
			reviewer.SubmittedAt = item.SubmittedAt
			normalizedLedgerStatus, warning := canonicalSlotStatus(item.Status)
			ledgerClaimedSubmitted = normalizedLedgerStatus == "submitted"
			ledgerStatusWarning = warning
		}
		reviewerWarnings := make([]string, 0, 4)
		_, submission, submissionWarning := readJSONArtifact[Submission]("Reviewer result", artifactPath, validateSubmissionArtifact)
		if submission != nil {
			if reviewer.Role == "" {
				reviewer.Role = submission.Role
			}
			reviewer.Name = strings.TrimSpace(submission.By)
			reviewer.Summary = submission.Summary
			if reviewer.SubmittedAt == "" {
				reviewer.SubmittedAt = submission.SubmittedAt
			}
			if reviewer.Status == "" {
				if submissionLooksSubmitted(*submission) {
					reviewer.Status = "submitted"
				} else {
					reviewer.Status = "pending"
				}
			}
		} else if reviewer.Status == "" || ledgerClaimedSubmitted {
			reviewer.Status = "pending"
		}

		if ledgerStatusWarning != "" {
			reviewerWarnings = append(reviewerWarnings, ledgerStatusWarning)
		}
		if submissionWarning != "" {
			reviewerWarnings = append(reviewerWarnings, submissionWarning)
		}
		if ledgerClaimedSubmitted && submission == nil {
			reviewerWarnings = append(reviewerWarnings, "Ledger marks this reviewer as submitted, but the submission artifact is unavailable.")
		}
		if hasLedgerEntry && !ledgerClaimedSubmitted && submission != nil && submissionLooksSubmitted(*submission) {
			reviewerWarnings = append(reviewerWarnings, "Submission artifact exists even though the ledger does not mark this reviewer as submitted.")
		}
		reviewer.Warnings = dedupeStrings(reviewerWarnings)
		if len(reviewer.Warnings) > 0 {
			warnings = append(warnings, fmt.Sprintf("Reviewer result: %s", strings.Join(reviewer.Warnings, " ")))
		}
		reviewers = append(reviewers, reviewer)
	}
	return reviewers, warnings
}

func selectReviewer(reviewers []reviewerState) *Reviewer {
	if len(reviewers) == 0 {
		return nil
	}
	selected := reviewers[0]
	for _, reviewer := range reviewers {
		if strings.EqualFold(strings.TrimSpace(reviewer.Role), "integrated") {
			selected = reviewer
			break
		}
	}
	return &Reviewer{
		Name:        selected.Name,
		Status:      normalizeSlotStatus(selected.Status),
		SubmittedAt: selected.SubmittedAt,
		Summary:     selected.Summary,
		Warnings:    selected.Warnings,
	}
}

func findingViews(findings []contracts.ReviewAggregateFinding) []FindingView {
	views := make([]FindingView, 0, len(findings))
	for _, finding := range findings {
		views = append(views, FindingView{
			FindingID: finding.FindingID,
			Area:      finding.Area,
			Severity:  finding.Severity,
			Title:     finding.Title,
			Details:   finding.Details,
			Locations: append([]string(nil), finding.Locations...),
		})
	}
	return views
}

func submissionLooksSubmitted(submission Submission) bool {
	return strings.TrimSpace(submission.SubmittedAt) != ""
}

func normalizeSlotStatus(status string) string {
	value, _ := canonicalSlotStatus(status)
	return value
}

func canonicalSlotStatus(status string) (string, string) {
	value := strings.TrimSpace(strings.ToLower(status))
	switch value {
	case "", "pending":
		return "pending", ""
	case "submitted":
		return "submitted", ""
	default:
		return "pending", fmt.Sprintf("Review progress reports unknown reviewer status %q, so the result is shown conservatively as pending.", strings.TrimSpace(status))
	}
}

func resolveRoundStatus(round Round, reviewerCount, pendingReviewers int, manifestArtifact, ledgerArtifact, decisionArtifact artifact) (string, string) {
	if manifestArtifact.Status == "invalid" || ledgerArtifact.Status == "invalid" || decisionArtifact.Status == "invalid" {
		return "degraded", "One or more artifacts are malformed; review state is shown conservatively."
	}
	if manifestArtifact.Status == "missing" && ledgerArtifact.Status == "missing" {
		return "degraded", "Core review artifacts are missing."
	}
	if reviewerCount == 0 {
		if round.IsActive {
			return "in_progress", "Review is active, but the independent reviewer could not be recovered yet."
		}
		return "incomplete", "Review metadata is incomplete."
	}
	if pendingReviewers > 0 {
		return "waiting_for_review", "Waiting for independent review."
	}
	if manifestArtifact.Status != "available" || ledgerArtifact.Status != "available" {
		return "degraded", "Core review artifacts are incomplete, so the decision is shown conservatively."
	}
	if decisionArtifact.Status == "available" && strings.TrimSpace(round.Decision) != "" {
		switch round.Decision {
		case "pass":
			return "pass", "Independent review passed."
		case "changes_requested":
			if len(round.UnresolvedFindingIDs) > 0 {
				return "changes_requested", fmt.Sprintf("Review requested changes; %d blocking finding(s) remain unresolved.", len(round.UnresolvedFindingIDs))
			}
			return "changes_requested", "Review requested changes."
		default:
			return "complete", fmt.Sprintf("Review decision: %s.", round.Decision)
		}
	}
	if decisionArtifact.Status == "missing" {
		return "waiting_for_decision", "Independent review is complete; waiting for its decision."
	}
	if round.IsActive {
		return "in_progress", "Review round is still active."
	}
	return "complete", "Review data is available."
}

func discoverSubmissionPaths(submissionsDir string) (map[string]string, []string) {
	paths := map[string]string{}
	warnings := []string{}

	entries, err := os.ReadDir(submissionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return paths, warnings
		}
		return paths, []string{"Unable to inspect reviewer submissions."}
	}

	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if entry.IsDir() {
			slot := name
			path := filepath.Join(submissionsDir, slot, "submission.json")
			if _, err := os.Stat(path); err == nil {
				paths[slot] = path
			}
			continue
		}
		if filepath.Ext(name) != ".json" {
			continue
		}
		slot := strings.TrimSpace(strings.TrimSuffix(name, ".json"))
		if slot == "" {
			continue
		}
		paths[slot] = filepath.Join(submissionsDir, name)
	}

	return paths, warnings
}

type artifactValidator[T any] func(*T) []string

func readJSONArtifact[T any](label, path string, validator artifactValidator[T]) (artifact, *T, string) {
	result := artifact{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = "missing"
			return result, nil, fmt.Sprintf("%s is missing.", label)
		}
		result.Status = "invalid"
		return result, nil, fmt.Sprintf("Unable to read %s.", strings.ToLower(label))
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		result.Status = "invalid"
		return result, nil, fmt.Sprintf("%s is empty.", label)
	}
	if !json.Valid([]byte(trimmed)) {
		result.Status = "invalid"
		return result, nil, fmt.Sprintf("%s is not valid JSON.", label)
	}

	var parsed T
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		result.Status = "invalid"
		return result, nil, fmt.Sprintf("%s could not be parsed cleanly.", label)
	}
	if validator != nil {
		if missing := dedupeStrings(validator(&parsed)); len(missing) > 0 {
			result.Status = "invalid"
			return result, nil, fmt.Sprintf("%s is missing required fields: %s.", label, strings.Join(missing, ", "))
		}
	}

	result.Status = "available"
	return result, &parsed, ""
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func validateManifestArtifact(manifest *Manifest) []string {
	missing := []string{}
	if strings.TrimSpace(manifest.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(manifest.Kind) == "" {
		missing = append(missing, "kind")
	}
	if manifest.Revision <= 0 {
		missing = append(missing, "revision")
	}
	if strings.TrimSpace(manifest.PlanPath) == "" {
		missing = append(missing, "plan_path")
	}
	if strings.TrimSpace(manifest.PlanStem) == "" {
		missing = append(missing, "plan_stem")
	}
	if strings.TrimSpace(manifest.CreatedAt) == "" {
		missing = append(missing, "created_at")
	}
	return missing
}

func validateLedgerArtifact(ledger *Ledger) []string {
	missing := []string{}
	if strings.TrimSpace(ledger.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(ledger.Kind) == "" {
		missing = append(missing, "kind")
	}
	if strings.TrimSpace(ledger.UpdatedAt) == "" {
		missing = append(missing, "updated_at")
	}
	return missing
}

func validateSubmissionArtifact(submission *Submission) []string {
	missing := []string{}
	if strings.TrimSpace(submission.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	return missing
}

func validateAggregateArtifact(aggregate *Aggregate) []string {
	missing := []string{}
	if strings.TrimSpace(aggregate.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(aggregate.Kind) == "" {
		missing = append(missing, "kind")
	}
	if aggregate.Revision <= 0 {
		missing = append(missing, "revision")
	}
	if strings.TrimSpace(aggregate.Decision) == "" {
		missing = append(missing, "decision")
	}
	if strings.TrimSpace(aggregate.AggregatedAt) == "" {
		missing = append(missing, "decided_at")
	}
	findings := append([]contracts.ReviewAggregateFinding(nil), aggregate.BlockingFindings...)
	findings = append(findings, aggregate.NonBlockingFindings...)
	findings = append(findings, aggregate.UnresolvedBlockingFindings...)
	for index, finding := range findings {
		prefix := fmt.Sprintf("findings[%d]", index)
		if strings.TrimSpace(finding.FindingID) == "" {
			missing = append(missing, prefix+".finding_id")
		}
		if strings.TrimSpace(finding.Area) == "" {
			missing = append(missing, prefix+".area")
		}
	}
	return missing
}
