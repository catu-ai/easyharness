package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/catu-ai/easyharness/internal/reviewdimensions"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/stepcloseout"
)

var compactRoundIDPattern = regexp.MustCompile(`^review-([0-9]+)-([a-z0-9-]+)$`)
var saveState = runstate.SaveState
var writeJSON = writeJSONFile
var resolveDimensions = resolveReviewDimensions

type Service struct {
	Workdir               string
	Now                   func() time.Time
	AfterStart            func(StartResult) error
	AfterSubmit           func(SubmitResult) error
	AfterAggregate        func(AggregateResult) error
	AfterStartSuccess     func(StartResult)
	AfterSubmitSuccess    func(SubmitResult)
	AfterAggregateSuccess func(AggregateResult)
}

type Spec = contracts.ReviewSpec
type AssignmentSpec = contracts.ReviewAssignmentSpec
type RepairReference = contracts.ReviewRepairReference
type RiskBrief = contracts.ReviewRiskBrief
type ResolvedDimension = contracts.ReviewResolvedDimension
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
type AggregateResult = contracts.ReviewAggregateResult
type AggregateArtifacts = contracts.ReviewAggregateArtifacts

type reviewDualMutationLocks struct {
	PlanPath string
	release  func()
}

type reviewMutationLockFailure struct {
	Summary string
	Issue   CommandError
}

func (s Service) Start(specBytes []byte) StartResult {
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
	planPath, doc, planStem, relPlanPath, state, statePath, errResult := s.loadCurrentExecutingPlan(locks.PlanPath)
	if errResult != nil {
		return *errResult
	}

	var spec Spec
	if issues := inputschema.DecodeAndValidate("inputs.review.spec", "spec", specBytes, &spec); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateSpec(spec); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateDeltaAnchor(s.Workdir, spec); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	inferredStep, revision, reviewTitle, err := inferReviewBinding(s.Workdir, planStem, doc, state, spec)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec does not match the current workflow state.",
			Errors:  []CommandError{{Path: "spec", Message: err.Error()}},
		}
	}
	if issues := validateAssignmentTopology(spec, inferredStep); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	materializedAssignments, issues := materializeAssignments(s.Workdir, planPath, spec.Assignments)
	if len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review assignment guidance could not be resolved.",
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
		RoundID:     roundID,
		Kind:        spec.Kind,
		AnchorSHA:   strings.TrimSpace(spec.AnchorSHA),
		Step:        inferredStep,
		Revision:    revision,
		ReviewTitle: reviewTitle,
		Repair:      cloneRepairReference(spec.Repair),
		PlanPath:    relPlanPath,
		PlanStem:    planStem,
		CreatedAt:   now.Format(time.RFC3339),
		Assignments: assignments,
		LedgerPath:  ledgerPath,
		Aggregate:   aggregatePath,
		Submissions: submissionsDir,
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
		Step:       inferredStep,
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
			ProjectRoot: s.Workdir,
			PlanPath:    relPlanPath,
			RoundID:     roundID,
			Assignments: surfacedReviewAssignments(s.Workdir, assignments),
		},
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Launch reviewer subagents for the returned assignments and have each reviewer submit structured results for its assigned slot.",
			},
			{
				Command:     strPtr(fmt.Sprintf("harness review aggregate --round %s", roundID)),
				Description: "Aggregate the round once every expected reviewer submission has landed.",
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

func (s Service) Submit(roundID, slot, reviewerName string, inputBytes []byte) SubmitResult {
	lockedPlanPath, release, err := s.acquireReviewMutationLock()
	if err == nil {
		defer release()
	} else {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Another review state mutation is already in progress.",
			Errors:  []CommandError{{Path: "review", Message: "Another review state mutation is already in progress."}},
		}
	}
	_, _, planStem, _, _, _, errResult := s.loadCurrentExecutingPlan(lockedPlanPath)
	if errResult != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: errResult.Summary,
			Errors:  errResult.Errors,
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
	assignmentDef := findAssignment(manifest, slot)
	if assignmentDef == nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Submission does not match an expected reviewer slot.",
			Errors:  []CommandError{{Path: "slot", Message: fmt.Sprintf("unknown slot %q for review round %q", slot, roundID)}},
		}
	}
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
		ExtraFields: cloneRawFields(input.ExtraFields),
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

	return s.finalizeSubmit(SubmitResult{
		OK:      true,
		Command: "review submit",
		Summary: fmt.Sprintf("Recorded submission for slot %q in review round %q.", assignmentDef.Slot, roundID),
		Artifacts: &SubmitArtifacts{
			ProjectRoot:    s.Workdir,
			RoundID:        roundID,
			Slot:           assignmentDef.Slot,
			SubmissionPath: repoFacingReviewPath(s.Workdir, assignmentDef.SubmissionPath),
		},
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Report the submission receipt to the controller agent and end the reviewer thread. If the same slot later needs a narrow follow-up for the same tracked step or the same finalize review title in the same revision, the controller may reopen this reviewer through the runtime's native resume mechanism only after this submission is verified and only while the slot instructions still materially match.",
			},
		},
	}, func() []CommandError {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "ledger")
		issues = append(issues, restoreJSONFileSnapshot(assignmentDef.SubmissionPath, previousSubmission, previousSubmissionExists, "submission")...)
		return issues
	})
}

func (s Service) Aggregate(roundID string) AggregateResult {
	locks, failure := s.acquireReviewAndStateMutationLocks()
	if failure != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: failure.Summary,
			Errors:  []CommandError{failure.Issue},
		}
	}
	defer locks.release()

	_, _, planStem, _, state, statePath, errResult := s.loadCurrentExecutingPlan(locks.PlanPath)
	if errResult != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: errResult.Summary,
			Errors:  errResult.Errors,
		}
	}
	if guard := activeAggregateRoundError(state, roundID); guard != nil {
		return *guard
	}

	manifestPath := filepath.Join(runstate.ReviewRoundDir(s.Workdir, planStem, roundID), "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to load review round metadata.",
			Errors:  []CommandError{{Path: "review.round", Message: sanitizedReadMessage("review round metadata", err)}},
		}
	}

	ledger, err := loadLedger(manifest.LedgerPath)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
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
	resolutionStatus := initialResolutionStatus(manifest.Repair)
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
			return AggregateResult{
				OK:      false,
				Command: "review aggregate",
				Summary: "Unable to load reviewer submissions.",
				Errors:  []CommandError{{Path: "review.submission", Message: sanitizedReadMessage("reviewer submissions", err)}},
			}
		}
		if issues := validateStoredSubmission(*submission); len(issues) > 0 {
			return AggregateResult{
				OK:      false,
				Command: "review aggregate",
				Summary: "Reviewer submission artifact is invalid for aggregation.",
				Errors:  issues,
			}
		}
		if issues := validateStoredSubmissionForAssignment(*submission, manifest, assignmentDef); len(issues) > 0 {
			return AggregateResult{
				OK:      false,
				Command: "review aggregate",
				Summary: "Reviewer submission artifact does not match its assignment or repair contract.",
				Errors:  issues,
			}
		}
		applyResolutions(resolutionStatus, submission.Resolutions)
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
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Review round is missing required submissions.",
			Errors:  []CommandError{{Path: "submissions", Message: fmt.Sprintf("missing submissions for slots: %s", strings.Join(missing, ", "))}},
		}
	}

	decision := "pass"
	resolvedFindingIDs, unresolvedFindingIDs := partitionResolutionStatus(manifest.Repair, resolutionStatus)
	if len(blocking) > 0 || len(unresolvedFindingIDs) > 0 {
		decision = "changes_requested"
	}

	aggregate := Aggregate{
		RoundID:              roundID,
		Kind:                 manifest.Kind,
		Step:                 manifest.Step,
		Revision:             manifest.Revision,
		ReviewTitle:          manifest.ReviewTitle,
		Repair:               cloneRepairReference(manifest.Repair),
		Decision:             decision,
		BlockingFindings:     blocking,
		NonBlockingFindings:  nonBlocking,
		ResolvedFindingIDs:   resolvedFindingIDs,
		UnresolvedFindingIDs: unresolvedFindingIDs,
		AggregatedAt:         s.now().Format(time.RFC3339),
	}
	state, _, err = runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to reload local harness state before persisting the aggregate.",
			Errors:  []CommandError{{Path: "state", Message: sanitizedReadMessage("local harness state", err)}},
		}
	}
	if guard := activeAggregateRoundError(state, roundID); guard != nil {
		return *guard
	}
	originalState := cloneState(state)
	previousAggregate, previousAggregateExists, err := readFileIfExists(manifest.Aggregate)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to snapshot the previous aggregate artifact.",
			Errors:  []CommandError{{Path: "review.decision", Message: sanitizedReadMessage("the previous review decision", err)}},
		}
	}
	if err := writeJSON(manifest.Aggregate, aggregate); err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
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
		Step:       manifest.Step,
		Revision:   manifest.Revision,
		Aggregated: true,
		Decision:   decision,
	}
	statePath, err = saveState(s.Workdir, planStem, state)
	if err != nil {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")...)
		issues = append([]CommandError{{Path: "state", Message: sanitizedWriteMessage("local harness state", err)}}, issues...)
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to persist local harness state.",
			Errors:  issues,
		}
	}

	return s.finalizeAggregate(AggregateResult{
		OK:      true,
		Command: "review aggregate",
		Summary: buildAggregateSummary(manifest.Kind, decision, len(blocking), len(nonBlocking)),
		Artifacts: &AggregateArtifacts{
			ProjectRoot: s.Workdir,
			RoundID:     roundID,
		},
		Review:     &aggregate,
		NextAction: buildAggregateNextActions(manifest.Kind, decision),
	}, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")...)
		return issues
	})
}

// review start and review aggregate both mutate review artifacts plus state.json.
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

func activeAggregateRoundError(state *runstate.State, roundID string) *AggregateResult {
	if state == nil || state.ActiveReviewRound == nil {
		return &AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "No active review round is available to aggregate.",
			Errors:  []CommandError{{Path: "round", Message: "review aggregate only supports the current active review round"}},
		}
	}
	if state.ActiveReviewRound.RoundID == roundID {
		return nil
	}
	return &AggregateResult{
		OK:      false,
		Command: "review aggregate",
		Summary: "Only the current active review round can be aggregated.",
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
			return "", nil, fmt.Errorf("another review start or aggregate command is already mutating plan %q; retry after it finishes", planStem)
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

func surfacedReviewAssignments(workdir string, assignments []ManifestAssignment) []ManifestAssignment {
	if len(assignments) == 0 {
		return nil
	}
	surfaced := make([]ManifestAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		next := assignment
		next.SubmissionPath = repoFacingReviewPath(workdir, assignment.SubmissionPath)
		surfaced = append(surfaced, next)
	}
	return surfaced
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

func validateSpec(spec Spec) []CommandError {
	issues := make([]CommandError, 0)
	if !slices.Contains([]string{"delta", "full"}, spec.Kind) {
		issues = append(issues, CommandError{Path: "spec.kind", Message: "must be delta or full"})
	}
	if spec.Kind == "delta" && strings.TrimSpace(spec.AnchorSHA) == "" {
		issues = append(issues, CommandError{Path: "spec.anchor_sha", Message: "must not be empty for delta review"})
	}
	if spec.Step != nil && *spec.Step <= 0 {
		issues = append(issues, CommandError{Path: "spec.step", Message: "must be a positive 1-based step number"})
	}
	if spec.Repair != nil {
		if spec.Kind != "delta" {
			issues = append(issues, CommandError{Path: "spec.repair", Message: "is only valid for delta review"})
		}
		if strings.TrimSpace(spec.Repair.RoundID) == "" {
			issues = append(issues, CommandError{Path: "spec.repair.round_id", Message: "must not be empty"})
		}
		if len(spec.Repair.FindingIDs) == 0 {
			issues = append(issues, CommandError{Path: "spec.repair.finding_ids", Message: "must contain at least one finding id"})
		}
		seenFindingIDs := map[string]bool{}
		for i, findingID := range spec.Repair.FindingIDs {
			findingID = strings.TrimSpace(findingID)
			if findingID == "" {
				issues = append(issues, CommandError{Path: fmt.Sprintf("spec.repair.finding_ids[%d]", i), Message: "must not be empty"})
			} else if seenFindingIDs[findingID] {
				issues = append(issues, CommandError{Path: fmt.Sprintf("spec.repair.finding_ids[%d]", i), Message: fmt.Sprintf("duplicates finding id %q", findingID)})
			}
			seenFindingIDs[findingID] = true
		}
	}
	if len(spec.Assignments) == 0 {
		issues = append(issues, CommandError{Path: "spec.assignments", Message: "must contain at least one reviewer assignment"})
	}
	seenSlots := map[string]bool{}
	for i, assignment := range spec.Assignments {
		pathPrefix := fmt.Sprintf("spec.assignments[%d]", i)
		slot := strings.TrimSpace(assignment.Slot)
		if !reviewdimensions.ValidName(slot) {
			issues = append(issues, CommandError{Path: pathPrefix + ".slot", Message: "must use lowercase alphanumeric segments separated by single hyphens"})
		}
		if !slices.Contains([]string{"integrated", "specialist"}, assignment.Role) {
			issues = append(issues, CommandError{Path: pathPrefix + ".role", Message: "must be integrated or specialist"})
		}
		if strings.TrimSpace(assignment.Instructions) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".instructions", Message: "must not be empty"})
		}
		if len(assignment.Dimensions) == 0 {
			issues = append(issues, CommandError{Path: pathPrefix + ".dimensions", Message: "must contain at least one guidance dimension"})
		}
		if seenSlots[slot] {
			issues = append(issues, CommandError{Path: pathPrefix + ".slot", Message: fmt.Sprintf("duplicates slot %q", slot)})
		}
		seenSlots[slot] = true
		seenDimensions := map[string]bool{}
		for j, dimension := range assignment.Dimensions {
			dimension = strings.TrimSpace(dimension)
			if !reviewdimensions.ValidName(dimension) {
				issues = append(issues, CommandError{Path: fmt.Sprintf("%s.dimensions[%d]", pathPrefix, j), Message: "must use lowercase alphanumeric segments separated by single hyphens"})
			} else if seenDimensions[dimension] {
				issues = append(issues, CommandError{Path: fmt.Sprintf("%s.dimensions[%d]", pathPrefix, j), Message: fmt.Sprintf("duplicates dimension %q", dimension)})
			}
			seenDimensions[dimension] = true
		}
		if assignment.Role == "specialist" {
			if assignment.RiskBrief == nil {
				issues = append(issues, CommandError{Path: pathPrefix + ".risk_brief", Message: "is required for specialist assignments"})
			} else {
				issues = append(issues, validateRiskBrief(pathPrefix+".risk_brief", assignment.RiskBrief)...)
			}
		} else if assignment.RiskBrief != nil {
			issues = append(issues, CommandError{Path: pathPrefix + ".risk_brief", Message: "must be omitted for integrated assignments"})
		}
	}
	return issues
}

func validateAssignmentTopology(spec Spec, inferredStep *int) []CommandError {
	if spec.Kind != "full" || inferredStep != nil {
		return nil
	}
	count := 0
	for _, assignment := range spec.Assignments {
		if assignment.Role == "integrated" {
			count++
		}
	}
	if count != 1 {
		return []CommandError{{Path: "spec.assignments", Message: "a full finalize review requires exactly one integrated assignment"}}
	}
	return nil
}

func validateRiskBrief(path string, brief *RiskBrief) []CommandError {
	issues := make([]CommandError, 0)
	if len(brief.RiskSurfaces) == 0 {
		issues = append(issues, CommandError{Path: path + ".risk_surfaces", Message: "must contain at least one concrete risk surface"})
	}
	if len(brief.Invariants) == 0 {
		issues = append(issues, CommandError{Path: path + ".invariants", Message: "must contain at least one invariant"})
	}
	for field, values := range map[string][]string{"risk_surfaces": brief.RiskSurfaces, "invariants": brief.Invariants, "failure_modes": brief.FailureModes} {
		for i, value := range values {
			if strings.TrimSpace(value) == "" {
				issues = append(issues, CommandError{Path: fmt.Sprintf("%s.%s[%d]", path, field, i), Message: "must not be empty"})
			}
		}
	}
	return issues
}

func validateDeltaAnchor(workdir string, spec Spec) []CommandError {
	if spec.Kind != "delta" {
		return nil
	}

	anchor := strings.TrimSpace(spec.AnchorSHA)
	if anchor == "" {
		return nil
	}

	if _, err := os.Stat(filepath.Join(workdir, ".git")); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []CommandError{{
			Path:    "spec.anchor_sha",
			Message: fmt.Sprintf("unable to inspect git metadata: %v", err),
		}}
	}

	cmd := exec.Command("git", "-C", workdir, "rev-parse", "--verify", anchor+"^{commit}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return []CommandError{{
			Path:    "spec.anchor_sha",
			Message: fmt.Sprintf("must resolve to a real git commit for delta review: %s", message),
		}}
	}

	return nil
}

func validateSubmission(input SubmissionInput) []CommandError {
	issues := make([]CommandError, 0)
	if _, obsolete := input.ExtraFields["dimension"]; obsolete {
		issues = append(issues, CommandError{Path: "submission.dimension", Message: "is obsolete; findings now carry area and submissions are owned by reviewer assignments"})
	}
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
		if !reviewdimensions.ValidName(finding.Area) {
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

func materializeAssignments(workdir, planPath string, specs []AssignmentSpec) ([]ManifestAssignment, []CommandError) {
	assignments := make([]ManifestAssignment, 0, len(specs))
	for i, spec := range specs {
		dimensions, err := resolveDimensions(workdir, planPath, spec.Dimensions)
		if err != nil {
			return nil, []CommandError{{Path: fmt.Sprintf("spec.assignments[%d].dimensions", i), Message: err.Error()}}
		}
		assignments = append(assignments, ManifestAssignment{
			Slot:         strings.TrimSpace(spec.Slot),
			Role:         strings.TrimSpace(spec.Role),
			Dimensions:   dimensions,
			Instructions: strings.TrimSpace(spec.Instructions),
			RiskBrief:    cloneRiskBrief(spec.RiskBrief),
		})
	}
	return assignments, nil
}

func resolveReviewDimensions(workdir, planPath string, names []string) ([]ResolvedDimension, error) {
	service := reviewdimensions.Service{Workdir: workdir}
	resolved := make([]ResolvedDimension, 0, len(names))
	for _, name := range names {
		dimension, warnings, issues := service.ResolveForPlan(planPath, strings.TrimSpace(name))
		if len(issues) > 0 {
			return nil, fmt.Errorf("unable to resolve dimension %q: %s", name, issues[0].Message)
		}
		if len(warnings) > 0 {
			return nil, fmt.Errorf("unable to resolve dimension %q cleanly: %s", name, warnings[0])
		}
		resolved = append(resolved, ResolvedDimension{
			Name:         dimension.Name,
			Sources:      append([]string(nil), dimension.Sources...),
			Description:  dimension.Description,
			Instructions: dimension.Instructions,
			PlanPath:     dimension.PlanPath,
		})
	}
	return resolved, nil
}

func cloneRiskBrief(brief *RiskBrief) *RiskBrief {
	if brief == nil {
		return nil
	}
	return &RiskBrief{
		RiskSurfaces: append([]string(nil), brief.RiskSurfaces...),
		Invariants:   append([]string(nil), brief.Invariants...),
		FailureModes: append([]string(nil), brief.FailureModes...),
	}
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

func initialResolutionStatus(reference *RepairReference) map[string]string {
	status := map[string]string{}
	if reference == nil {
		return status
	}
	for _, findingID := range reference.FindingIDs {
		status[strings.TrimSpace(findingID)] = "unresolved"
	}
	return status
}

func applyResolutions(status map[string]string, resolutions []FindingResolution) {
	for _, resolution := range resolutions {
		findingID := strings.TrimSpace(resolution.FindingID)
		if _, ok := status[findingID]; !ok {
			continue
		}
		if resolution.Status == "unresolved" {
			status[findingID] = "unresolved-confirmed"
			continue
		}
		if status[findingID] == "unresolved" {
			status[findingID] = "resolved"
		}
	}
}

func partitionResolutionStatus(reference *RepairReference, status map[string]string) ([]string, []string) {
	if reference == nil {
		return []string{}, []string{}
	}
	resolved := make([]string, 0, len(reference.FindingIDs))
	unresolved := make([]string, 0, len(reference.FindingIDs))
	for _, findingID := range reference.FindingIDs {
		findingID = strings.TrimSpace(findingID)
		if status[findingID] == "resolved" {
			resolved = append(resolved, findingID)
		} else {
			unresolved = append(unresolved, findingID)
		}
	}
	return resolved, unresolved
}

func validateStoredSubmission(submission Submission) []CommandError {
	issues := make([]CommandError, 0)
	if strings.TrimSpace(submission.RoundID) == "" {
		issues = append(issues, CommandError{Path: "submission.round_id", Message: "must not be empty"})
	}
	if strings.TrimSpace(submission.Slot) == "" {
		issues = append(issues, CommandError{Path: "submission.slot", Message: "must not be empty"})
	}
	if !slices.Contains([]string{"integrated", "specialist"}, submission.Role) {
		issues = append(issues, CommandError{Path: "submission.role", Message: "must be integrated or specialist"})
	}
	if strings.TrimSpace(submission.SubmittedAt) == "" {
		issues = append(issues, CommandError{Path: "submission.submitted_at", Message: "must not be empty"})
	}
	issues = append(issues, validateSubmission(SubmissionInput{
		Summary:     submission.Summary,
		Resolutions: submission.Resolutions,
		Findings:    submission.Findings,
		ExtraFields: submission.ExtraFields,
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

func cloneRawFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	if len(fields) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}

func newSubmissionSkeleton(roundID string, assignment ManifestAssignment) map[string]any {
	return map[string]any{
		"round_id":    roundID,
		"slot":        assignment.Slot,
		"role":        assignment.Role,
		"summary":     "",
		"resolutions": []FindingResolution{},
		"findings":    []Finding{},
		"worklog": map[string]any{
			"full_plan_read":     false,
			"checked_areas":      []string{},
			"open_questions":     []string{},
			"candidate_findings": []string{},
		},
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

func inferReviewBinding(workdir, planStem string, doc *plan.Document, state *runstate.State, spec Spec) (*int, int, string, error) {
	revision := runstate.CurrentRevision(state)
	if stepIndex, ok, err := inferReviewStepIndex(doc, state, spec); err != nil {
		return nil, 0, "", err
	} else if ok {
		reviewTitle := strings.TrimSpace(spec.ReviewTitle)
		if reviewTitle == "" {
			reviewTitle = doc.Steps[stepIndex].Title
		}
		stepNumber := stepIndex + 1
		return &stepNumber, revision, reviewTitle, nil
	}

	if pendingNewStepReopen(doc, state) {
		return nil, 0, "", fmt.Errorf("reopen mode new-step still requires a new unfinished step before review can start")
	}
	if !doc.AllStepsCompleted() {
		return nil, 0, "", fmt.Errorf("no reviewable tracked step could be inferred; set spec.step to select a tracked step explicitly")
	}
	reminder := stepcloseout.LoadReminder(workdir, planStem, doc, "execution/finalize/review", nil)
	if len(reminder.MissingTitles) > 0 {
		earliestTitle := reminder.MissingTitles[0]
		earliestStepNumber := reminder.MissingIndexes[0] + 1
		return nil, 0, "", fmt.Errorf("earlier completed steps still need review-complete closeout; repair %s first with spec.step=%d or record NO_STEP_REVIEW_NEEDED: <reason> in Review Notes before starting default finalize review", earliestTitle, earliestStepNumber)
	}

	reviewTitle := strings.TrimSpace(spec.ReviewTitle)
	if reviewTitle == "" {
		if spec.Kind == "full" {
			reviewTitle = "Full branch candidate before archive"
		} else {
			reviewTitle = "Branch candidate before archive"
		}
	}
	return nil, revision, reviewTitle, nil
}

func inferReviewStepIndex(doc *plan.Document, state *runstate.State, spec Spec) (int, bool, error) {
	if doc == nil {
		return 0, false, fmt.Errorf("current plan is unavailable")
	}
	if spec.Step != nil {
		index := *spec.Step - 1
		if index < 0 || index >= len(doc.Steps) {
			return 0, false, fmt.Errorf("spec.step=%d does not match a tracked step", *spec.Step)
		}
		return index, true, nil
	}
	if current := currentStepIndex(doc); current >= 0 {
		return current, true, nil
	}
	return 0, false, nil
}

func currentStepIndex(doc *plan.Document) int {
	if doc == nil {
		return -1
	}
	for index, step := range doc.Steps {
		if !step.Done {
			return index
		}
	}
	return -1
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

func buildAggregateSummary(kind, decision string, blocking, nonBlocking int) string {
	if decision == "pass" {
		return fmt.Sprintf("%s review passed with %d non-blocking finding(s).", kind, nonBlocking)
	}
	return fmt.Sprintf("%s review found %d blocking and %d non-blocking finding(s).", kind, blocking, nonBlocking)
}

func buildAggregateNextActions(kind, decision string) []NextAction {
	if decision == "pass" {
		if kind == "delta" {
			return []NextAction{{
				Command:     nil,
				Description: "Continue the current step or mark it complete, then update the step's Execution Notes and Review Notes.",
			}}
		}
		return []NextAction{{
			Command:     nil,
			Description: "Move toward final CI and archive readiness for the current candidate.",
		}}
	}
	if kind == "delta" {
		return []NextAction{{
			Command:     nil,
			Description: "Fix the current slice and rerun a delta review once the blocking findings are addressed.",
		}}
	}
	return []NextAction{{
		Command:     nil,
		Description: "Fix the blocking findings before archive and rerun full review if the candidate scope changed materially.",
	}}
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

func (s Service) finalizeAggregate(result AggregateResult, rollback func() []CommandError) AggregateResult {
	if !result.OK || s.AfterAggregate == nil {
		if result.OK && s.AfterAggregateSuccess != nil {
			s.AfterAggregateSuccess(result)
		}
		return result
	}
	if err := s.AfterAggregate(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: "Unable to record the review timeline event."}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return AggregateResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	if s.AfterAggregateSuccess != nil {
		s.AfterAggregateSuccess(result)
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
