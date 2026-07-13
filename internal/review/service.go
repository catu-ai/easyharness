package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/inputschema"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/reviewcoverage"
	"github.com/catu-ai/easyharness/internal/runstate"
)

var compactRoundIDPattern = regexp.MustCompile(`^review-([0-9]+)-([a-z0-9-]+)$`)
var findingAreaPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var saveState = runstate.SaveState
var writeJSON = writeJSONFile

type Service struct {
	Workdir            string
	Now                func() time.Time
	AfterStart         func(StartResult) error
	AfterSubmit        func(SubmitResult) error
	AfterStartSuccess  func(StartResult)
	AfterSubmitSuccess func(SubmitResult)
}

type StartOptions = contracts.ReviewStartOptions
type RepairReference = contracts.ReviewRepairReference
type Manifest = contracts.ReviewManifest
type ManifestAssignment = contracts.ReviewAssignment
type Ledger = contracts.ReviewLedger
type LedgerAssignment = contracts.ReviewLedgerAssignment
type SubmissionInput = contracts.ReviewSubmissionInput
type Submission = contracts.ReviewSubmission
type Finding = contracts.ReviewFinding
type FindingResolution = contracts.ReviewFindingResolution
type Aggregate = contracts.ReviewAggregate
type AggregateFinding = contracts.ReviewAggregateFinding
type CommandError = contracts.ErrorDetail
type NextAction = contracts.NextAction
type StartResult = contracts.ReviewStartResult
type StartArtifacts = contracts.ReviewStartArtifacts
type SubmitResult = contracts.ReviewSubmitResult
type SubmitArtifacts = contracts.ReviewSubmitArtifacts

type inferredSpec struct {
	Kind      string
	AnchorSHA string
	Repair    *RepairReference
}

type completionResult struct {
	OK      bool
	Summary string
	Errors  []CommandError
	Review  *Aggregate
}

type reviewDualMutationLocks struct {
	PlanPath string
	release  func()
}

type reviewMutationLockFailure struct {
	Summary string
	Issue   CommandError
}

func (s Service) Start(options StartOptions) StartResult {
	locks, failure := s.acquireReviewAndStateMutationLocks()
	if failure != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: failure.Summary,
			Errors:  []CommandError{failure.Issue},
		}
	}
	defer locks.release()

	now := s.now()
	_, doc, planStem, relPlanPath, state, statePath, errResult := s.loadCurrentExecutingPlan(locks.PlanPath)
	if errResult != nil {
		return *errResult
	}

	if !doc.AllStepsCompleted() {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Finalize review is not ready to start.",
			Errors:  []CommandError{{Path: "plan.steps", Message: "complete every tracked step before starting finalize review"}},
		}
	}
	if pendingNewStepReopen(doc, state) {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Finalize review is not ready to start.",
			Errors:  []CommandError{{Path: "plan.steps", Message: "reopen mode new-step still requires a new unfinished step before review can start"}},
		}
	}
	if state != nil && state.ActiveReviewRound != nil && !state.ActiveReviewRound.Aggregated {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Finalize review is already in progress.",
			Errors:  []CommandError{{Path: "state.active_review_round", Message: fmt.Sprintf("review round %s must be submitted before another review starts", state.ActiveReviewRound.RoundID)}},
		}
	}
	spec, err := inferFinalizeSpec(s.Workdir, planStem, state, options.ForceFull)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to infer finalize review coverage.",
			Errors:  []CommandError{{Path: "review.coverage", Message: err.Error()}},
		}
	}
	revision := runstate.CurrentRevision(state)
	reviewTitle := "Complete candidate before archive"
	reviewFocus := strings.TrimSpace(doc.SectionText("Review Focus"))
	materializedAssignments := []ManifestAssignment{{
		Slot:         "integrated",
		Role:         "integrated",
		Instructions: "Use the harness-reviewer skill to review the complete candidate against its fixed integrated rubric. Spawn bounded advisor subagents only when they materially improve coverage; advisors report only to you.",
		ReviewFocus:  reviewFocus,
	}}
	candidate, err := reviewcoverage.CaptureCandidate(s.Workdir)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review candidate is not a clean committed Git boundary.",
			Errors:  []CommandError{{Path: "review.candidate", Message: err.Error()}},
		}
	}
	if issues := validateRepairLink(s.Workdir, planStem, state, spec, revision); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review repair does not continuously extend finalize coverage.",
			Errors:  issues,
		}
	}

	roundID, err := nextRoundID(s.Workdir, planStem, spec.Kind)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to determine the next review round identifier.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedReadMessage("the next review round identifier", err)}},
		}
	}
	roundDir := runstate.ReviewRoundDir(s.Workdir, planStem, roundID)
	submissionsDir := filepath.Join(roundDir, "submissions")
	manifestPath := filepath.Join(roundDir, "manifest.json")
	ledgerPath := filepath.Join(roundDir, "ledger.json")
	aggregatePath := filepath.Join(roundDir, "aggregate.json")
	if err := os.MkdirAll(submissionsDir, 0o755); err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to initialize review artifacts.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedWriteMessage("review round artifacts", err)}},
		}
	}

	assignments := make([]ManifestAssignment, 0, len(materializedAssignments))
	ledger := Ledger{
		RoundID:     roundID,
		Kind:        spec.Kind,
		UpdatedAt:   now.Format(time.RFC3339),
		Assignments: make([]LedgerAssignment, 0, len(materializedAssignments)),
	}
	for _, assignment := range materializedAssignments {
		submissionPath := filepath.Join(submissionsDir, assignment.Slot, "submission.json")
		assignment.SubmissionPath = submissionPath
		assignments = append(assignments, assignment)
		ledger.Assignments = append(ledger.Assignments, LedgerAssignment{
			Slot:           assignment.Slot,
			Role:           assignment.Role,
			Status:         "pending",
			SubmissionPath: submissionPath,
		})
	}
	for _, assignment := range assignments {
		if err := writeJSON(assignment.SubmissionPath, newSubmissionSkeleton(roundID, assignment)); err != nil {
			_ = os.RemoveAll(roundDir)
			return StartResult{
				OK:      false,
				Command: "review start",
				Summary: "Unable to create reviewer submission skeletons.",
				Errors:  []CommandError{{Path: "review.submission", Message: sanitizedWriteMessage("reviewer submission skeletons", err)}},
			}
		}
	}

	manifest := Manifest{
		RoundID:         roundID,
		Kind:            spec.Kind,
		AnchorSHA:       strings.TrimSpace(spec.AnchorSHA),
		ReviewedHeadSHA: candidate.HeadSHA,
		Revision:        revision,
		ReviewTitle:     reviewTitle,
		ReviewFocus:     reviewFocus,
		Repair:          cloneRepairReference(spec.Repair),
		PlanPath:        relPlanPath,
		PlanStem:        planStem,
		CreatedAt:       now.Format(time.RFC3339),
		Assignments:     assignments,
		LedgerPath:      ledgerPath,
		Aggregate:       aggregatePath,
		Submissions:     submissionsDir,
	}
	confirmedCandidate, confirmErr := reviewcoverage.CaptureCandidate(s.Workdir)
	if confirmErr != nil || confirmedCandidate.HeadSHA != candidate.HeadSHA {
		_ = os.RemoveAll(roundDir)
		message := "candidate worktree is no longer clean"
		if confirmErr != nil {
			message = confirmErr.Error()
		} else {
			message = fmt.Sprintf("candidate HEAD moved from %s to %s", candidate.HeadSHA, confirmedCandidate.HeadSHA)
		}
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review candidate changed before round persistence.",
			Errors:  []CommandError{{Path: "review.candidate", Message: message}},
		}
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		_ = os.RemoveAll(roundDir)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist review round metadata.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedWriteMessage("review round metadata", err)}},
		}
	}
	if err := writeJSON(ledgerPath, ledger); err != nil {
		_ = os.RemoveAll(roundDir)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist reviewer assignment state.",
			Errors:  []CommandError{{Path: "review.assignments", Message: sanitizedWriteMessage("reviewer assignment state", err)}},
		}
	}

	originalState := cloneState(state)
	if state == nil {
		state = &runstate.State{}
	}
	state.ActiveReviewRound = &runstate.ReviewRound{
		RoundID:    roundID,
		Kind:       spec.Kind,
		Revision:   revision,
		Aggregated: false,
		Decision:   "",
	}
	statePath, err = saveState(s.Workdir, planStem, state)
	if err != nil {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		if removeErr := os.RemoveAll(roundDir); removeErr != nil && !os.IsNotExist(removeErr) {
			issues = append(issues, CommandError{Path: "review.round", Message: sanitizedRollbackMessage("review round artifacts", removeErr)})
		}
		issues = append([]CommandError{{Path: "state", Message: sanitizedWriteMessage("local harness state", err)}}, issues...)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist local harness state.",
			Errors:  issues,
		}
	}

	_ = doc

	return s.finalizeStart(StartResult{
		OK:      true,
		Command: "review start",
		Summary: fmt.Sprintf("Created %s review round %q.", spec.Kind, roundID),
		Artifacts: &StartArtifacts{
			ProjectRoot:     s.Workdir,
			PlanPath:        relPlanPath,
			RoundID:         roundID,
			ReviewedHeadSHA: candidate.HeadSHA,
			Reviewer:        surfacedReviewAssignment(s.Workdir, assignments[0]),
		},
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Launch the returned integrated reviewer. Its single submission completes the review round and records the decision.",
			},
		},
	}, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		if err := os.RemoveAll(roundDir); err != nil && !os.IsNotExist(err) {
			issues = append(issues, CommandError{Path: "review.round", Message: sanitizedRollbackMessage("review round artifacts", err)})
		}
		return issues
	})
}

func (s Service) Submit(roundID, reviewerName string, inputBytes []byte) SubmitResult {
	locks, failure := s.acquireReviewAndStateMutationLocks()
	if failure != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: failure.Summary,
			Errors:  []CommandError{failure.Issue},
		}
	}
	defer locks.release()
	_, _, planStem, _, state, _, errResult := s.loadCurrentExecutingPlan(locks.PlanPath)
	if errResult != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: errResult.Summary,
			Errors:  errResult.Errors,
		}
	}
	if guard := activeCompletionRoundError(state, roundID); guard != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: guard.Summary,
			Errors:  guard.Errors,
		}
	}

	manifestPath := filepath.Join(runstate.ReviewRoundDir(s.Workdir, planStem, roundID), "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to load review round metadata.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedReadMessage("review round metadata", err)}},
		}
	}
	if len(manifest.Assignments) != 1 || manifest.Assignments[0].Slot != "integrated" || manifest.Assignments[0].Role != "integrated" {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Review round does not contain exactly one integrated reviewer.",
			Errors:  []CommandError{{Path: "review.reviewer", Message: fmt.Sprintf("round %q must contain exactly one integrated reviewer", roundID)}},
		}
	}
	assignmentDef := &manifest.Assignments[0]
	if strings.TrimSpace(reviewerName) == "" {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer identity is required.",
			Errors:  []CommandError{{Path: "by", Message: "review submit requires a non-empty reviewer identity label"}},
		}
	}

	var input SubmissionInput
	if issues := inputschema.DecodeAndValidate("inputs.review.submission", "submission", inputBytes, &input); len(issues) > 0 {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer submission is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateSubmission(input); len(issues) > 0 {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer submission is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateSubmissionForManifest(input, manifest); len(issues) > 0 {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer submission does not match the review repair contract.",
			Errors:  issues,
		}
	}

	now := s.now().Format(time.RFC3339)
	submission := Submission{
		RoundID:     roundID,
		Slot:        assignmentDef.Slot,
		Role:        assignmentDef.Role,
		By:          strings.TrimSpace(reviewerName),
		SubmittedAt: now,
		Summary:     strings.TrimSpace(input.Summary),
		Resolutions: cloneResolutions(input.Resolutions),
		Findings:    input.Findings,
	}
	previousSubmission, previousSubmissionExists, err := readFileIfExists(assignmentDef.SubmissionPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to snapshot the previous reviewer submission.",
			Errors:  []CommandError{{Path: "review.submission", Message: sanitizedReadMessage("the previous reviewer submission", err)}},
		}
	}
	ledger, err := loadLedger(manifest.LedgerPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to load reviewer assignment state.",
			Errors:  []CommandError{{Path: "review.assignments", Message: sanitizedReadMessage("reviewer assignment state", err)}},
		}
	}
	previousLedger, previousLedgerExists, err := readFileIfExists(manifest.LedgerPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to snapshot the review ledger before writing the submission.",
			Errors:  []CommandError{{Path: "review.assignments", Message: sanitizedReadMessage("reviewer assignment state", err)}},
		}
	}
	if err := writeJSON(assignmentDef.SubmissionPath, submission); err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to persist the reviewer submission.",
			Errors:  []CommandError{{Path: "review.submission", Message: sanitizedWriteMessage("the reviewer submission", err)}},
		}
	}
	for i := range ledger.Assignments {
		if ledger.Assignments[i].Slot == assignmentDef.Slot {
			ledger.Assignments[i].Status = "submitted"
			ledger.Assignments[i].SubmittedAt = now
		}
	}
	ledger.UpdatedAt = now
	if err := writeJSON(manifest.LedgerPath, ledger); err != nil {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "review.assignments")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "review.submission")...)
		issues = append([]CommandError{{Path: "review.assignments", Message: sanitizedWriteMessage("reviewer assignment state", err)}}, issues...)
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to persist reviewer assignment state.",
			Errors:  issues,
		}
	}
	originalState, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "review.assignments")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "review.submission")...)
		return SubmitResult{OK: false, Command: "review submit", Summary: "Unable to snapshot local harness state before completing review.", Errors: append([]CommandError{{Path: "state", Message: sanitizedReadMessage("local harness state", err)}}, issues...)}
	}
	previousAggregate, previousAggregateExists, err := readFileIfExists(manifest.Aggregate)
	if err != nil {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "review.assignments")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "review.submission")...)
		return SubmitResult{OK: false, Command: "review submit", Summary: "Unable to snapshot the review decision before completing review.", Errors: append([]CommandError{{Path: "review.decision", Message: sanitizedReadMessage("the review decision", err)}}, issues...)}
	}
	completion := s.completeRoundLocked(roundID, planStem)
	if !completion.OK || completion.Review == nil {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "review.assignments")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "review.submission")...)
		return SubmitResult{OK: false, Command: "review submit", Summary: completion.Summary, Errors: append(completion.Errors, issues...)}
	}

	return s.finalizeSubmit(SubmitResult{
		OK:      true,
		Command: "review submit",
		Summary: fmt.Sprintf("Recorded the integrated reviewer submission for review round %q.", roundID),
		Artifacts: &SubmitArtifacts{
			ProjectRoot:    s.Workdir,
			RoundID:        roundID,
			SubmissionPath: repoFacingReviewPath(s.Workdir, assignmentDef.SubmissionPath),
		},
		Review: completion.Review,
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Report the completed review decision to the controller and end the reviewer thread.",
			},
		},
	}, func() []CommandError {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "ledger")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "submission")...)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "review.decision")...)
		issues = append(issues, restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)...)
		return issues
	})
}

func (s Service) completeRoundLocked(roundID, planStem string) completionResult {
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: sanitizedReadMessage("local harness state", err)}},
		}
	}
	if guard := activeCompletionRoundError(state, roundID); guard != nil {
		return *guard
	}

	manifestPath := filepath.Join(runstate.ReviewRoundDir(s.Workdir, planStem, roundID), "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to load review round metadata.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedReadMessage("review round metadata", err)}},
		}
	}
	candidate, err := reviewcoverage.CaptureCandidate(s.Workdir)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Review candidate changed while the round was in progress.",
			Errors:  []CommandError{{Path: "review.candidate", Message: err.Error()}},
		}
	}
	if candidate.HeadSHA != manifest.ReviewedHeadSHA {
		return completionResult{
			OK:      false,
			Summary: "Review candidate HEAD moved while the round was in progress.",
			Errors: []CommandError{{
				Path:    "review.reviewed_head_sha",
				Message: fmt.Sprintf("round captured %s but current HEAD is %s; start a new review round", manifest.ReviewedHeadSHA, candidate.HeadSHA),
			}},
		}
	}

	ledger, err := loadLedger(manifest.LedgerPath)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to load reviewer assignment state.",
			Errors:  []CommandError{{Path: "review.assignments", Message: sanitizedReadMessage("reviewer assignment state", err)}},
		}
	}
	ledgerBySlot := map[string]LedgerAssignment{}
	for _, assignment := range ledger.Assignments {
		ledgerBySlot[assignment.Slot] = assignment
	}

	blocking := make([]AggregateFinding, 0)
	nonBlocking := make([]AggregateFinding, 0)
	missing := make([]string, 0)
	unresolvedByID := map[string]AggregateFinding{}
	if manifest.Repair != nil {
		parent, err := reviewcoverage.LoadRound(s.Workdir, planStem, manifest.Repair.RoundID)
		if err != nil {
			return completionResult{
				OK:      false,
				Summary: "Unable to load the repair parent review round.",
				Errors:  []CommandError{{Path: "review.repair.round_id", Message: err.Error()}},
			}
		}
		for _, finding := range parent.Aggregate.UnresolvedBlockingFindings {
			unresolvedByID[finding.FindingID] = finding
		}
	}
	resolutionVerdicts := map[string]string{}
	for _, assignmentDef := range manifest.Assignments {
		ledgerSlot, ok := ledgerBySlot[assignmentDef.Slot]
		if !ok || ledgerSlot.Status != "submitted" {
			missing = append(missing, assignmentDef.Slot)
			continue
		}
		submission, err := loadSubmission(assignmentDef.SubmissionPath)
		if err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, assignmentDef.Slot)
				continue
			}
			return completionResult{
				OK:      false,
				Summary: "Unable to load reviewer submissions.",
				Errors:  []CommandError{{Path: "review.submission", Message: sanitizedReadMessage("reviewer submissions", err)}},
			}
		}
		if issues := validateStoredSubmission(*submission); len(issues) > 0 {
			return completionResult{
				OK:      false,
				Summary: "Reviewer submission artifact is invalid for aggregation.",
				Errors:  issues,
			}
		}
		if issues := validateStoredSubmissionForAssignment(*submission, manifest, assignmentDef); len(issues) > 0 {
			return completionResult{
				OK:      false,
				Summary: "Reviewer submission artifact does not match its assignment or repair contract.",
				Errors:  issues,
			}
		}
		for _, resolution := range submission.Resolutions {
			findingID := strings.TrimSpace(resolution.FindingID)
			if prior, exists := resolutionVerdicts[findingID]; exists {
				return completionResult{
					OK:      false,
					Summary: "Repair finding resolution ownership is ambiguous.",
					Errors:  []CommandError{{Path: "review.resolutions", Message: fmt.Sprintf("finding %q has multiple resolution verdicts (%s and %s)", findingID, prior, resolution.Status)}},
				}
			}
			resolutionVerdicts[findingID] = resolution.Status
		}
		for findingIndex, finding := range submission.Findings {
			aggregateFinding := AggregateFinding{
				FindingID:    findingID(roundID, submission.Slot, findingIndex),
				Slot:         submission.Slot,
				Role:         submission.Role,
				Area:         finding.Area,
				Severity:     finding.Severity,
				Title:        finding.Title,
				Details:      finding.Details,
				Locations:    cloneLocations(finding.Locations, finding.HasLocations),
				HasLocations: finding.HasLocations,
			}
			if isBlockingSeverity(finding.Severity) {
				blocking = append(blocking, aggregateFinding)
			} else {
				nonBlocking = append(nonBlocking, aggregateFinding)
			}
		}
	}
	if len(missing) > 0 {
		return completionResult{
			OK:      false,
			Summary: "Review round is missing required submissions.",
			Errors:  []CommandError{{Path: "submissions", Message: fmt.Sprintf("missing submissions for slots: %s", strings.Join(missing, ", "))}},
		}
	}

	resolvedFindingIDs := make([]string, 0)
	for findingID, verdict := range resolutionVerdicts {
		if verdict == "resolved" {
			resolvedFindingIDs = append(resolvedFindingIDs, findingID)
			delete(unresolvedByID, findingID)
		}
	}
	for _, finding := range blocking {
		unresolvedByID[finding.FindingID] = finding
	}
	slices.Sort(resolvedFindingIDs)
	unresolvedBlocking := make([]AggregateFinding, 0, len(unresolvedByID))
	for _, finding := range unresolvedByID {
		unresolvedBlocking = append(unresolvedBlocking, finding)
	}
	slices.SortFunc(unresolvedBlocking, func(left, right AggregateFinding) int {
		return strings.Compare(left.FindingID, right.FindingID)
	})
	unresolvedFindingIDs := make([]string, 0, len(unresolvedBlocking))
	for _, finding := range unresolvedBlocking {
		unresolvedFindingIDs = append(unresolvedFindingIDs, finding.FindingID)
	}
	decision := "pass"
	if len(unresolvedFindingIDs) > 0 {
		decision = "changes_requested"
	}

	aggregate := Aggregate{
		RoundID:                    roundID,
		Kind:                       manifest.Kind,
		Revision:                   manifest.Revision,
		ReviewTitle:                manifest.ReviewTitle,
		ReviewedHeadSHA:            manifest.ReviewedHeadSHA,
		Repair:                     cloneRepairReference(manifest.Repair),
		Decision:                   decision,
		BlockingFindings:           blocking,
		NonBlockingFindings:        nonBlocking,
		ResolvedFindingIDs:         resolvedFindingIDs,
		UnresolvedFindingIDs:       unresolvedFindingIDs,
		UnresolvedBlockingFindings: unresolvedBlocking,
		AggregatedAt:               s.now().Format(time.RFC3339),
	}
	state, _, err = runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to reload local harness state before persisting the aggregate.",
			Errors:  []CommandError{{Path: "state", Message: sanitizedReadMessage("local harness state", err)}},
		}
	}
	if guard := activeCompletionRoundError(state, roundID); guard != nil {
		return *guard
	}
	originalState := cloneState(state)
	previousAggregate, previousAggregateExists, err := readFileIfExists(manifest.Aggregate)
	if err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to snapshot the previous aggregate artifact.",
			Errors:  []CommandError{{Path: "review.decision", Message: sanitizedReadMessage("the previous review decision", err)}},
		}
	}
	confirmedCandidate, err := reviewcoverage.CaptureCandidate(s.Workdir)
	if err != nil || confirmedCandidate.HeadSHA != manifest.ReviewedHeadSHA {
		message := "candidate worktree is no longer clean"
		if err != nil {
			message = err.Error()
		} else {
			message = fmt.Sprintf("round captured %s but current HEAD is %s", manifest.ReviewedHeadSHA, confirmedCandidate.HeadSHA)
		}
		return completionResult{
			OK:      false,
			Summary: "Review candidate changed before aggregate persistence.",
			Errors:  []CommandError{{Path: "review.reviewed_head_sha", Message: message}},
		}
	}
	if err := writeJSON(manifest.Aggregate, aggregate); err != nil {
		return completionResult{
			OK:      false,
			Summary: "Unable to persist the aggregate review result.",
			Errors:  []CommandError{{Path: "review.decision", Message: sanitizedWriteMessage("the persisted review decision", err)}},
		}
	}

	if state == nil {
		state = &runstate.State{}
	}
	state.ActiveReviewRound = &runstate.ReviewRound{
		RoundID:    manifest.RoundID,
		Kind:       manifest.Kind,
		Revision:   manifest.Revision,
		Aggregated: true,
		Decision:   decision,
	}
	chain, coverageErr := reviewcoverage.Resolve(s.Workdir, planStem, manifest.RoundID, manifest.Revision)
	if coverageErr != nil {
		issues := restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")
		return completionResult{
			OK:      false,
			Summary: "Unable to establish continuous finalize review coverage.",
			Errors:  append([]CommandError{{Path: "review.coverage", Message: coverageErr.Error()}}, issues...),
		}
	}
	state.FinalizeCoverage = reviewcoverage.StateFromChain(chain)
	statePath, err = saveState(s.Workdir, planStem, state)
	if err != nil {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")...)
		issues = append([]CommandError{{Path: "state", Message: sanitizedWriteMessage("local harness state", err)}}, issues...)
		return completionResult{
			OK:      false,
			Summary: "Unable to persist local harness state.",
			Errors:  issues,
		}
	}

	return completionResult{
		OK:      true,
		Summary: buildAggregateSummary(manifest, decision, len(unresolvedBlocking), len(nonBlocking)),
		Review:  &aggregate,
	}
}

// Review start and reviewer submission completion both mutate review artifacts
// plus state.json.
// Keep the review lock outermost so future edits cannot drift into a different
// acquisition order than the rest of the review command family expects.
func (s Service) acquireReviewAndStateMutationLocks() (*reviewDualMutationLocks, *reviewMutationLockFailure) {
	planPath, releaseReview, err := s.acquireReviewMutationLock()
	if err != nil {
		return nil, &reviewMutationLockFailure{
			Summary: "Another review state mutation is already in progress.",
			Issue:   CommandError{Path: "review", Message: "Another review state mutation is already in progress."},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	releaseState, err := runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		releaseReview()
		return nil, &reviewMutationLockFailure{
			Summary: "Another local state mutation is already in progress.",
			Issue:   CommandError{Path: "state", Message: "Another local state mutation is already in progress."},
		}
	}
	return &reviewDualMutationLocks{
		PlanPath: planPath,
		release: func() {
			releaseState()
			releaseReview()
		},
	}, nil
}

func activeCompletionRoundError(state *runstate.State, roundID string) *completionResult {
	if state == nil || state.ActiveReviewRound == nil {
		return &completionResult{
			OK:      false,
			Summary: "No active review round is available to complete.",
			Errors:  []CommandError{{Path: "round", Message: "review submit only supports the current active review round"}},
		}
	}
	if state.ActiveReviewRound.RoundID == roundID && !state.ActiveReviewRound.Aggregated {
		return nil
	}
	if state.ActiveReviewRound.RoundID == roundID {
		return &completionResult{
			OK:      false,
			Summary: "Review round is already complete.",
			Errors: []CommandError{{
				Path:    "round",
				Message: fmt.Sprintf("round %q is complete and immutable; start a new review round for later candidate changes", roundID),
			}},
		}
	}
	return &completionResult{
		OK:      false,
		Summary: "Only the current active review round can be completed.",
		Errors: []CommandError{{
			Path:    "round",
			Message: fmt.Sprintf("round %q is not the current active review round %q", roundID, state.ActiveReviewRound.RoundID),
		}},
	}
}

func (s Service) acquireReviewMutationLock() (string, func(), error) {
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", func() {}, nil
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	lockPath := runstate.PlanRuntimePath(s.Workdir, planStem, ".review-mutation.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return "", nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return "", nil, fmt.Errorf("another review command is already mutating plan %q; retry after it finishes", planStem)
		}
		return "", nil, err
	}
	return planPath, func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func (s Service) loadCurrentExecutingPlan(lockedPlanPath string) (string, *plan.Document, string, string, *runstate.State, string, *StartResult) {
	planPath := strings.TrimSpace(lockedPlanPath)
	if planPath == "" {
		var err error
		planPath, err = plan.DetectCurrentPath(s.Workdir)
		if err != nil {
			return "", nil, "", "", nil, "", &StartResult{
				OK:      false,
				Command: "review start",
				Summary: "Unable to determine the current plan.",
				Errors:  []CommandError{{Path: "plan", Message: sanitizedReadMessage("the current plan", err)}},
			}
		}
	}
	doc, err := plan.LoadFile(planPath)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: sanitizedReadMessage("the current plan", err)}},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "plan", Message: "Unable to relativize the current plan path."}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: sanitizedReadMessage("local harness state", err)}},
		}
	}
	if doc.DerivedPlanStatus() != "active" || doc.DerivedLifecycle(state) != "executing" {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review commands require an active executing plan.",
			Errors: []CommandError{{
				Path:    "plan.lifecycle",
				Message: fmt.Sprintf("current plan is status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
			}},
		}
	}
	return planPath, doc, planStem, relPlanPath, state, statePath, nil
}

func surfacedReviewAssignment(workdir string, assignment ManifestAssignment) *contracts.ReviewHandle {
	return &contracts.ReviewHandle{
		Instructions:   assignment.Instructions,
		ReviewFocus:    assignment.ReviewFocus,
		SubmissionPath: repoFacingReviewPath(workdir, assignment.SubmissionPath),
	}
}

func repoFacingReviewPath(workdir, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if !filepath.IsAbs(trimmed) {
		return filepath.ToSlash(filepath.Clean(trimmed))
	}
	relPath, err := filepath.Rel(workdir, trimmed)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(trimmed))
	}
	if relPath == "." || relPath == "" {
		return "."
	}
	if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		return filepath.ToSlash(filepath.Clean(trimmed))
	}
	return filepath.ToSlash(filepath.Clean(relPath))
}

func inferFinalizeSpec(workdir, planStem string, state *runstate.State, forceFull bool) (inferredSpec, error) {
	if forceFull || state == nil || state.FinalizeCoverage == nil || strings.TrimSpace(state.FinalizeCoverage.TipRoundID) == "" {
		return inferredSpec{Kind: "full"}, nil
	}
	coverage := state.FinalizeCoverage
	parent, err := reviewcoverage.LoadRound(workdir, planStem, coverage.TipRoundID)
	if err != nil {
		return inferredSpec{}, fmt.Errorf("load current finalize coverage tip: %w", err)
	}
	anchor := strings.TrimSpace(coverage.CoveredHeadSHA)
	if anchor == "" {
		anchor = strings.TrimSpace(parent.Manifest.ReviewedHeadSHA)
	}
	if anchor == "" {
		return inferredSpec{}, fmt.Errorf("current finalize coverage tip has no reviewed head")
	}
	return inferredSpec{
		Kind:      "delta",
		AnchorSHA: anchor,
		Repair: &RepairReference{
			RoundID:    coverage.TipRoundID,
			FindingIDs: append([]string(nil), parent.Aggregate.UnresolvedFindingIDs...),
		},
	}, nil
}

func validateRepairLink(workdir, planStem string, state *runstate.State, spec inferredSpec, revision int) []CommandError {
	if spec.Kind == "full" {
		if spec.Repair != nil {
			return []CommandError{{Path: "spec.repair", Message: "must be omitted when a full review resets finalize coverage"}}
		}
		return nil
	}
	if spec.Repair == nil {
		return []CommandError{{Path: "spec.repair", Message: "is required for a finalize delta so the coverage parent is explicit"}}
	}
	parentID := strings.TrimSpace(spec.Repair.RoundID)
	if state == nil || state.FinalizeCoverage == nil || strings.TrimSpace(state.FinalizeCoverage.TipRoundID) == "" {
		return []CommandError{{Path: "spec.repair.round_id", Message: "cannot extend finalize coverage before a full root is aggregated"}}
	}
	if parentID != state.FinalizeCoverage.TipRoundID {
		return []CommandError{{Path: "spec.repair.round_id", Message: fmt.Sprintf("must directly reference current finalize coverage tip %q", state.FinalizeCoverage.TipRoundID)}}
	}
	parent, err := reviewcoverage.LoadRound(workdir, planStem, parentID)
	if err != nil {
		return []CommandError{{Path: "spec.repair.round_id", Message: err.Error()}}
	}
	if spec.AnchorSHA != parent.Manifest.ReviewedHeadSHA {
		return []CommandError{{Path: "spec.anchor_sha", Message: fmt.Sprintf("must equal parent reviewed head %s", parent.Manifest.ReviewedHeadSHA)}}
	}
	if revision < parent.Manifest.Revision || revision > parent.Manifest.Revision+1 {
		return []CommandError{{Path: "spec.repair.round_id", Message: "parent revision does not continuously precede this review"}}
	}
	if revision == parent.Manifest.Revision+1 && (parent.Aggregate.Decision != "pass" || len(parent.Aggregate.UnresolvedFindingIDs) > 0) {
		return []CommandError{{Path: "spec.repair.round_id", Message: "a reopened revision can extend only a clean prior-revision coverage tip"}}
	}
	targeted := map[string]bool{}
	for _, findingID := range spec.Repair.FindingIDs {
		targeted[strings.TrimSpace(findingID)] = true
	}
	known := map[string]bool{}
	for _, finding := range parent.Aggregate.BlockingFindings {
		known[finding.FindingID] = true
	}
	for _, finding := range parent.Aggregate.NonBlockingFindings {
		known[finding.FindingID] = true
	}
	for _, finding := range parent.Aggregate.UnresolvedBlockingFindings {
		known[finding.FindingID] = true
	}
	issues := make([]CommandError, 0)
	for _, findingID := range parent.Aggregate.UnresolvedFindingIDs {
		if !targeted[findingID] {
			issues = append(issues, CommandError{Path: "spec.repair.finding_ids", Message: fmt.Sprintf("must cover unresolved parent finding %q", findingID)})
		}
	}
	for index, findingID := range spec.Repair.FindingIDs {
		findingID = strings.TrimSpace(findingID)
		if !known[findingID] {
			issues = append(issues, CommandError{Path: fmt.Sprintf("spec.repair.finding_ids[%d]", index), Message: fmt.Sprintf("finding %q does not exist in parent round %s", findingID, parentID)})
		}
	}
	return issues
}

func validateSubmission(input SubmissionInput) []CommandError {
	issues := make([]CommandError, 0)
	if strings.TrimSpace(input.Summary) == "" {
		issues = append(issues, CommandError{Path: "submission.summary", Message: "must not be empty"})
	}
	seenResolutions := map[string]bool{}
	for i, resolution := range input.Resolutions {
		pathPrefix := fmt.Sprintf("submission.resolutions[%d]", i)
		findingID := strings.TrimSpace(resolution.FindingID)
		if findingID == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".finding_id", Message: "must not be empty"})
		} else if seenResolutions[findingID] {
			issues = append(issues, CommandError{Path: pathPrefix + ".finding_id", Message: fmt.Sprintf("duplicates finding id %q", findingID)})
		}
		seenResolutions[findingID] = true
		if !slices.Contains([]string{"resolved", "unresolved"}, resolution.Status) {
			issues = append(issues, CommandError{Path: pathPrefix + ".status", Message: "must be resolved or unresolved"})
		}
		if strings.TrimSpace(resolution.Details) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".details", Message: "must not be empty"})
		}
	}
	for i, finding := range input.Findings {
		pathPrefix := fmt.Sprintf("submission.findings[%d]", i)
		if !findingAreaPattern.MatchString(strings.TrimSpace(finding.Area)) {
			issues = append(issues, CommandError{Path: pathPrefix + ".area", Message: "must use lowercase alphanumeric segments separated by single hyphens"})
		}
		if !slices.Contains([]string{"blocker", "important", "minor"}, finding.Severity) {
			issues = append(issues, CommandError{Path: pathPrefix + ".severity", Message: "must be blocker, important, or minor"})
		}
		if strings.TrimSpace(finding.Title) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".title", Message: "must not be empty"})
		}
		if strings.TrimSpace(finding.Details) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".details", Message: "must not be empty"})
		}
		if finding.HasLocations && finding.Locations == nil {
			issues = append(issues, CommandError{
				Path:    pathPrefix + ".locations",
				Message: "must be an array of strings when present",
			})
			continue
		}
		for j, location := range finding.Locations {
			if strings.TrimSpace(location) == "" {
				issues = append(issues, CommandError{
					Path:    fmt.Sprintf("%s.locations[%d]", pathPrefix, j),
					Message: "must not be empty",
				})
			}
		}
	}
	return issues
}

func validateSubmissionForManifest(input SubmissionInput, manifest *Manifest) []CommandError {
	if manifest.Repair == nil {
		if len(input.Resolutions) > 0 {
			return []CommandError{{Path: "submission.resolutions", Message: "must be omitted when the review round does not reference prior findings"}}
		}
		return nil
	}
	allowed := map[string]bool{}
	for _, findingID := range manifest.Repair.FindingIDs {
		allowed[strings.TrimSpace(findingID)] = true
	}
	issues := make([]CommandError, 0)
	for i, resolution := range input.Resolutions {
		if !allowed[strings.TrimSpace(resolution.FindingID)] {
			issues = append(issues, CommandError{Path: fmt.Sprintf("submission.resolutions[%d].finding_id", i), Message: "must reference a finding targeted by this repair round"})
		}
	}
	return issues
}

func cloneRepairReference(reference *RepairReference) *RepairReference {
	if reference == nil {
		return nil
	}
	return &RepairReference{
		RoundID:    strings.TrimSpace(reference.RoundID),
		FindingIDs: append([]string(nil), reference.FindingIDs...),
	}
}

func cloneResolutions(resolutions []FindingResolution) []FindingResolution {
	if len(resolutions) == 0 {
		return nil
	}
	return append([]FindingResolution(nil), resolutions...)
}

func findingID(roundID, slot string, findingIndex int) string {
	return fmt.Sprintf("%s/%s/%03d", roundID, slot, findingIndex+1)
}

func validateStoredSubmission(submission Submission) []CommandError {
	issues := make([]CommandError, 0)
	if strings.TrimSpace(submission.RoundID) == "" {
		issues = append(issues, CommandError{Path: "submission.round_id", Message: "must not be empty"})
	}
	if strings.TrimSpace(submission.Slot) == "" {
		issues = append(issues, CommandError{Path: "submission.slot", Message: "must not be empty"})
	}
	if submission.Role != "integrated" {
		issues = append(issues, CommandError{Path: "submission.role", Message: "must be integrated"})
	}
	if strings.TrimSpace(submission.SubmittedAt) == "" {
		issues = append(issues, CommandError{Path: "submission.submitted_at", Message: "must not be empty"})
	}
	issues = append(issues, validateSubmission(SubmissionInput{
		Summary:     submission.Summary,
		Resolutions: submission.Resolutions,
		Findings:    submission.Findings,
	})...)
	return issues
}

func validateStoredSubmissionForAssignment(submission Submission, manifest *Manifest, assignment ManifestAssignment) []CommandError {
	issues := make([]CommandError, 0)
	if submission.RoundID != manifest.RoundID {
		issues = append(issues, CommandError{Path: "submission.round_id", Message: "must match the review round manifest"})
	}
	if submission.Slot != assignment.Slot {
		issues = append(issues, CommandError{Path: "submission.slot", Message: "must match the reviewer assignment"})
	}
	if submission.Role != assignment.Role {
		issues = append(issues, CommandError{Path: "submission.role", Message: "must match the reviewer assignment"})
	}
	issues = append(issues, validateSubmissionForManifest(SubmissionInput{
		Resolutions: submission.Resolutions,
	}, manifest)...)
	return issues
}

func cloneLocations(locations []string, present bool) []string {
	if !present {
		return nil
	}
	if len(locations) == 0 {
		return []string{}
	}
	return append([]string(nil), locations...)
}

func newSubmissionSkeleton(roundID string, assignment ManifestAssignment) map[string]any {
	return map[string]any{
		"round_id":    roundID,
		"slot":        assignment.Slot,
		"role":        assignment.Role,
		"summary":     "",
		"resolutions": []FindingResolution{},
		"findings":    []Finding{},
	}
}

func nextRoundID(workdir, planStem, kind string) (string, error) {
	sequence, err := nextRoundSequence(workdir, planStem)
	if err != nil {
		return "", err
	}
	return formatRoundID(sequence, kind), nil
}

func nextRoundSequence(workdir, planStem string) (int, error) {
	reviewsDir := runstate.ReviewsDir(workdir, planStem)
	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}

	maxSequence := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := compactRoundIDPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		sequence, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("parse compact review round sequence from %q: %w", entry.Name(), err)
		}
		if sequence > maxSequence {
			maxSequence = sequence
		}
	}
	return maxSequence + 1, nil
}

func formatRoundID(sequence int, kind string) string {
	return fmt.Sprintf("review-%03d-%s", sequence, kind)
}

func pendingNewStepReopen(doc *plan.Document, state *runstate.State) bool {
	return state != nil &&
		state.Reopen != nil &&
		state.Reopen.Mode == "new-step" &&
		state.Reopen.BaseStepCount > 0 &&
		doc != nil &&
		len(doc.Steps) <= state.Reopen.BaseStepCount &&
		doc.CurrentStep() == nil &&
		doc.AllStepsCompleted()
}

func loadManifest(path string) (*Manifest, error) {
	var manifest Manifest
	if err := readJSONFile(path, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func loadLedger(path string) (*Ledger, error) {
	var ledger Ledger
	if err := readJSONFile(path, &ledger); err != nil {
		return nil, err
	}
	return &ledger, nil
}

func loadSubmission(path string) (*Submission, error) {
	var submission Submission
	if err := readJSONFile(path, &submission); err != nil {
		return nil, err
	}
	return &submission, nil
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func findAssignment(manifest *Manifest, slot string) *ManifestAssignment {
	for i := range manifest.Assignments {
		if manifest.Assignments[i].Slot == slot {
			return &manifest.Assignments[i]
		}
	}
	return nil
}

func isBlockingSeverity(severity string) bool {
	return severity == "blocker" || severity == "important"
}

func buildAggregateSummary(manifest *Manifest, decision string, unresolvedBlocking, nonBlocking int) string {
	scope := aggregateWorkflowScope(manifest)
	if decision == "pass" {
		return fmt.Sprintf("%s review passed with %d non-blocking finding(s).", scope, nonBlocking)
	}
	return fmt.Sprintf("%s review has %d unresolved blocking and %d non-blocking finding(s).", scope, unresolvedBlocking, nonBlocking)
}

func aggregateWorkflowScope(manifest *Manifest) string {
	if manifest.Repair != nil {
		return "Finalize repair"
	}
	return "Finalize"
}

func (s Service) finalizeStart(result StartResult, rollback func() []CommandError) StartResult {
	if !result.OK || s.AfterStart == nil {
		if result.OK && s.AfterStartSuccess != nil {
			s.AfterStartSuccess(result)
		}
		return result
	}
	if err := s.AfterStart(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: "Unable to record the review timeline event."}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return StartResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	if s.AfterStartSuccess != nil {
		s.AfterStartSuccess(result)
	}
	return result
}

func (s Service) finalizeSubmit(result SubmitResult, rollback func() []CommandError) SubmitResult {
	if !result.OK || s.AfterSubmit == nil {
		if result.OK && s.AfterSubmitSuccess != nil {
			s.AfterSubmitSuccess(result)
		}
		return result
	}
	if err := s.AfterSubmit(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: "Unable to record the review timeline event."}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return SubmitResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	if s.AfterSubmitSuccess != nil {
		s.AfterSubmitSuccess(result)
	}
	return result
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
	if state.FinalizeCoverage != nil {
		coverage := *state.FinalizeCoverage
		cloned.FinalizeCoverage = &coverage
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

func restoreStateSnapshot(workdir, planStem string, originalState *runstate.State, statePath string) []CommandError {
	issues := make([]CommandError, 0)
	if originalState != nil {
		if _, err := runstate.SaveState(workdir, planStem, originalState); err != nil {
			issues = append(issues, CommandError{Path: "state", Message: sanitizedRollbackMessage("local harness state", err)})
		}
		return issues
	}
	if statePath == "" {
		return issues
	}
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "state", Message: sanitizedRollbackMessage("local harness state", err)})
	}
	return issues
}

func readFileIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func restoreJSONFileSnapshot(path string, data []byte, existed bool, pathLabel string) []CommandError {
	issues := make([]CommandError, 0, 1)
	if existed {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return append(issues, CommandError{Path: pathLabel, Message: sanitizedRollbackMessage(pathLabel, err)})
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return append(issues, CommandError{Path: pathLabel, Message: sanitizedRollbackMessage(pathLabel, err)})
		}
		return issues
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: pathLabel, Message: sanitizedRollbackMessage(pathLabel, err)})
	}
	return issues
}

func sanitizedReadMessage(subject string, err error) string {
	return sanitizedStorageMessage("read", subject, err)
}

func sanitizedWriteMessage(subject string, err error) string {
	return sanitizedStorageMessage("write", subject, err)
}

func sanitizedRollbackMessage(subject string, err error) string {
	return sanitizedStorageMessage("restore", subject, err)
}

func sanitizedStorageMessage(action, subject string, err error) string {
	switch {
	case os.IsNotExist(err):
		return fmt.Sprintf("%s is missing.", subject)
	case os.IsPermission(err):
		return fmt.Sprintf("Permission denied while trying to %s %s.", action, subject)
	default:
		return fmt.Sprintf("Unable to %s %s.", action, subject)
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
