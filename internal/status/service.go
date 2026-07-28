package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/install"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/remote"
	"github.com/catu-ai/easyharness/internal/reviewcoverage"
	"github.com/catu-ai/easyharness/internal/runstate"
)

type Service struct {
	Workdir       string
	PlanSelector  string
	ObserveRemote bool
	RunCommand    remote.CommandRunner
}

type Result = contracts.StatusResult
type State = contracts.StatusState
type Facts = contracts.StatusFacts
type Artifacts = contracts.StatusArtifacts
type NextAction = contracts.NextAction
type StatusError = contracts.ErrorDetail

type reviewContext struct {
	RoundID         string
	Kind            string
	Revision        int
	ReviewedHeadSHA string
	SubmissionPath  string
	Aggregated      bool
	InFlight        bool
	Decision        string
	DecisionKnown   bool
}

type evidenceContext struct {
	Publish *evidence.PublishRecord
	CI      *evidence.CIRecord
	Sync    *evidence.SyncRecord
}

func (s Service) Snapshot() Result {
	currentPlan, err := runstate.LoadCurrentPlan(s.Workdir)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read current worktree state.",
			Errors:  []StatusError{{Path: "state", Message: "Unable to read current worktree state."}},
		}
	}

	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			if strings.TrimSpace(s.PlanSelector) != "" {
				return Result{
					OK:      false,
					Command: "status",
					Summary: "Unable to resolve the selected coordinated subplan.",
					Errors:  []StatusError{{Path: "plan", Message: "no current coordinated root plan is available"}},
				}
			}
			return idleResult(s.Workdir, currentPlan)
		}
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to determine the current plan.",
			Errors:  []StatusError{{Path: "plan", Message: "Unable to determine the current plan."}},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	doc, err := plan.LoadFile(planPath)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read the current plan.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, planPath),
			},
			Errors: []StatusError{{Path: "plan", Message: "Unable to read the current plan."}},
		}
	}
	state, _, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read local harness state.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, planPath),
			},
			Errors: []StatusError{{Path: "state", Message: "Unable to read local harness state."}},
		}
	}

	if strings.TrimSpace(s.PlanSelector) != "" {
		return s.snapshotSelectedSubplan(planPath, doc, state)
	}

	result := Result{
		OK:      true,
		Command: "status",
		Artifacts: &Artifacts{
			PlanPath: repoFacingPath(s.Workdir, planPath),
		},
	}
	supplementsPath := plan.SupplementsDirForPlanPath(planPath)
	if info, err := os.Stat(supplementsPath); err == nil && info.IsDir() {
		result.Artifacts.SupplementsPath = repoFacingPath(s.Workdir, supplementsPath)
	} else if err != nil && !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unable to inspect supplements path %s: %v", repoFacingPath(s.Workdir, supplementsPath), err))
	} else if err == nil && !info.IsDir() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("supplements path is not a directory: %s", repoFacingPath(s.Workdir, supplementsPath)))
	}

	reviewCtx, reviewWarnings := loadReviewContext(s.Workdir, planStem, state)
	result.Warnings = append(result.Warnings, reviewWarnings...)
	planApproved := doc.ExplicitlyApproved()
	if reviewCtx != nil && strings.TrimSpace(reviewCtx.RoundID) != "" {
		result.Artifacts.ReviewRoundID = reviewCtx.RoundID
		if reviewCtx.InFlight && strings.TrimSpace(reviewCtx.SubmissionPath) != "" {
			result.Artifacts.ReviewSubmissionPath = repoFacingPath(s.Workdir, reviewCtx.SubmissionPath)
		}
	}

	facts := &Facts{}
	applyPlanProgressFacts(facts, doc)
	if state != nil && state.Revision > 0 {
		facts.Revision = state.Revision
	}
	if reopenMode := effectiveReopenMode(doc, state); reopenMode != "" {
		facts.ReopenMode = reopenMode
	}
	if reviewCtx != nil {
		facts.ReviewKind = reviewCtx.Kind
		facts.ReviewedHeadSHA = reviewCtx.ReviewedHeadSHA
		switch {
		case reviewCtx.InFlight:
			facts.ReviewStatus = "in_progress"
		case reviewCtx.DecisionKnown:
			facts.ReviewStatus = reviewCtx.Decision
		case reviewCtx.Aggregated:
			facts.ReviewStatus = "unknown"
		}
	}

	var blockers []StatusError
	switch {
	case landInProgress(state):
		result.State.CurrentNode = "land"
		if state != nil && state.Land != nil {
			facts.LandPRURL = state.Land.PRURL
			facts.LandCommit = state.Land.Commit
		}
	case doc.DerivedPlanStatus() == "active" && !doc.ExecutionStarted(state):
		result.State.CurrentNode = "plan"
	case doc.DerivedPlanStatus() == "active":
		if doc.UsesCoordinatedProfile() {
			pkg, packageResult := loadCoordinatedPackageResult(s.Workdir, planPath)
			if packageResult != nil {
				return *packageResult
			}
			progress := pkg.Progress()
			facts.Subplans = &contracts.StatusSubplansFacts{
				Total:     progress.Total,
				Completed: progress.Completed,
				Runnable:  progress.Runnable,
				Waiting:   progress.Waiting,
			}
			blockers = append(blockers, documentIssuesToStatusErrors(pkg.DependencyIssues())...)
			reopenedNewSubplanPending := coordinatedNewSubplanPending(state, progress)
			if !reopenedNewSubplanPending && facts.ReopenMode == "new-step" {
				facts.ReopenMode = ""
			}
			if progress.Total == 0 || len(blockers) > 0 || !pkg.AllSubplansCompleted() || reopenedNewSubplanPending {
				result.State.CurrentNode = "execution/coordinate"
			} else {
				result.State.CurrentNode, blockers = resolveFinalizeNode(s.Workdir, planStem, doc, state, reviewCtx)
				if len(blockers) > 0 {
					facts.ArchiveBlockerCount = len(blockers)
				}
			}
		} else {
			stepIdx, stepNode := resolveStepNode(doc)
			if stepNode != "" {
				result.State.CurrentNode = stepNode
				facts.CurrentStep = doc.Steps[stepIdx].Title
			} else {
				result.State.CurrentNode, blockers = resolveFinalizeNode(s.Workdir, planStem, doc, state, reviewCtx)
				if len(blockers) > 0 {
					facts.ArchiveBlockerCount = len(blockers)
				}
			}
		}
	case doc.DerivedPlanStatus() == "archived":
		evidenceCtx, evidenceWarnings := loadEvidenceContext(s.Workdir, planStem, runstate.CurrentRevision(state))
		result.Warnings = append(result.Warnings, evidenceWarnings...)
		applyEvidenceFacts(facts, evidenceCtx)
		applyRemoteHandoffFacts(s, facts, evidenceCtx)
		result.State.CurrentNode = "execution/finalize/publish"
		if archivedCandidateReadyForMerge(evidenceCtx) {
			coverageIssues := lifecycle.EvaluateArchivedReviewCoverage(s.Workdir, planStem, doc, state)
			if mergedRemotePR(remoteEvidenceFacts(facts)) {
				coverageIssues = lifecycle.EvaluatePublishedReviewCoverage(s.Workdir, planStem, doc, state)
			}
			blockers = append(blockers, commandErrorsToStatusErrors(coverageIssues)...)
			if len(blockers) == 0 {
				result.State.CurrentNode = "execution/finalize/await_merge"
			}
		}
	default:
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to classify the current plan path.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, planPath),
			},
			Errors: []StatusError{{Path: "plan", Message: "unsupported plan path kind for current plan"}},
		}
	}

	result.Blockers = blockers
	result.Summary = buildSummary(result.State.CurrentNode, facts, reviewCtx, blockers, currentPlan, planApproved)
	result.NextAction = buildNextActions(result.State.CurrentNode, facts, reviewCtx, blockers, planApproved)
	if doc.UsesLightweightProfile() &&
		(result.State.CurrentNode == "execution/finalize/publish" || result.State.CurrentNode == "execution/finalize/await_merge") {
		action := NextAction{
			Command:     nil,
			Description: "Leave or verify the agreed repo-visible breadcrumb, such as a PR body note explaining why the lightweight path was used, before waiting for merge approval.",
		}
		result.NextAction = append([]NextAction{action}, result.NextAction...)
		if !strings.Contains(result.Summary, "lightweight path") {
			result.Summary += " The lightweight path still needs its repo-visible breadcrumb."
		}
	}
	if factsEmpty(facts) {
		result.Facts = nil
	} else {
		result.Facts = facts
	}
	decorateRepoBootstrapDrift(s.Workdir, &result, false)

	if result.Artifacts != nil && result.Artifacts.ProjectRoot == "" &&
		result.Artifacts.PlanPath == "" && result.Artifacts.SupplementsPath == "" &&
		result.Artifacts.ReviewRoundID == "" && result.Artifacts.ReviewSubmissionPath == "" &&
		result.Artifacts.LastLandedAt == "" {
		result.Artifacts = nil
	}

	return result
}

func (s Service) snapshotSelectedSubplan(rootPath string, rootDoc *plan.Document, rootState *runstate.State) Result {
	selector := strings.TrimSpace(s.PlanSelector)
	if rootDoc == nil || !rootDoc.UsesCoordinatedProfile() {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to select a subplan from the current plan.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, rootPath),
			},
			Errors: []StatusError{{
				Path:    "plan",
				Message: "`--plan` selects subplans only when the current root uses workflow_profile: coordinated",
			}},
		}
	}

	pkg, packageResult := loadCoordinatedPackageResult(s.Workdir, rootPath)
	if packageResult != nil {
		return *packageResult
	}
	selectedPath, err := plan.ResolveSubplanPath(rootPath, selector)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to resolve the selected coordinated subplan.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, rootPath),
			},
			Errors: []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}

	var selected *plan.SubplanDocument
	for _, child := range pkg.Subplans {
		if child != nil && filepath.Clean(child.Path) == filepath.Clean(selectedPath) {
			selected = child
			break
		}
	}
	if selected == nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to resolve the selected coordinated subplan.",
			Artifacts: &Artifacts{
				ProjectRoot: s.Workdir,
				PlanPath:    repoFacingPath(s.Workdir, rootPath),
			},
			Errors: []StatusError{{Path: "plan", Message: fmt.Sprintf("subplan %q is not part of the current coordinated package", selector)}},
		}
	}

	facts := &Facts{
		SelectedSubplan: &contracts.StatusSelectedSubplanFacts{
			ID:           selected.ID,
			Dependencies: append([]string(nil), selected.DependsOn...),
		},
	}
	applyStepProgressFacts(facts, selected.Steps)

	waitingOn := make([]string, 0, len(selected.DependsOn))
	for _, dependencyID := range selected.DependsOn {
		dependency := pkg.Subplan(dependencyID)
		if dependency == nil || !dependency.Completed() {
			waitingOn = append(waitingOn, dependencyID)
		}
	}
	facts.SelectedSubplan.WaitingOn = waitingOn

	blockers := documentIssuesToStatusErrors(pkg.DependencyIssues())
	result := Result{
		OK:       true,
		Command:  "status",
		Facts:    facts,
		Blockers: blockers,
		Artifacts: &Artifacts{
			PlanPath: repoFacingPath(s.Workdir, selected.Path),
		},
	}

	switch {
	case rootDoc.DerivedPlanStatus() == "active" && !rootDoc.ExecutionStarted(rootState):
		result.State.CurrentNode = "plan"
		result.Summary = fmt.Sprintf("Subplan %s belongs to a coordinated root whose execution has not started yet.", selected.ID)
		result.NextAction = []NextAction{{Command: nil, Description: "Wait for the controller to start execution for the approved coordinated root."}}
	case selected.Completed():
		result.State.CurrentNode = "execution/complete"
		result.Summary = fmt.Sprintf("Subplan %s has completed its ordered steps and final result.", selected.ID)
		result.NextAction = []NextAction{{Command: nil, Description: "The subplan is complete; return its result to the coordinated root and continue root-level integration."}}
	case len(waitingOn) > 0:
		result.State.CurrentNode = "execution/waiting"
		result.Summary = fmt.Sprintf("Subplan %s is waiting on %s.", selected.ID, strings.Join(waitingOn, ", "))
		result.NextAction = []NextAction{{Command: nil, Description: "Complete the unresolved sibling dependencies before executing this subplan."}}
	default:
		stepDoc := &plan.Document{Steps: selected.Steps}
		stepIdx, node := resolveStepNode(stepDoc)
		if node != "" {
			result.State.CurrentNode = node
			facts.CurrentStep = selected.Steps[stepIdx].Title
			result.Summary = fmt.Sprintf("Subplan %s is executing %s.", selected.ID, facts.CurrentStep)
			result.NextAction = []NextAction{{Command: nil, Description: "Continue the current subplan outcome and mark the step done after its concise check passes."}}
		} else {
			result.State.CurrentNode = "execution/subplan/closeout"
			result.Summary = fmt.Sprintf("Subplan %s has finished its ordered steps and needs its final result completed.", selected.ID)
			result.NextAction = []NextAction{{Command: nil, Description: "Record the subplan validation and delivered result so the coordinated root can count it as complete."}}
		}
	}

	decorateRepoBootstrapDrift(s.Workdir, &result, false)
	return result
}

func loadCoordinatedPackageResult(workdir, rootPath string) (*plan.CoordinatedPackage, *Result) {
	pkg, err := plan.LoadCoordinatedPackage(rootPath)
	if err == nil {
		return pkg, nil
	}
	result := Result{
		OK:      false,
		Command: "status",
		Summary: "Unable to read the coordinated plan package.",
		Artifacts: &Artifacts{
			ProjectRoot: workdir,
			PlanPath:    repoFacingPath(workdir, rootPath),
		},
		Errors: []StatusError{{Path: "plan", Message: err.Error()}},
	}
	return nil, &result
}

func resolveStepNode(doc *plan.Document) (int, string) {
	currentStepIndex := currentStepIndex(doc)
	if currentStepIndex < 0 {
		return -1, ""
	}
	return currentStepIndex, stepNode(currentStepIndex)
}

func applyPlanProgressFacts(facts *Facts, doc *plan.Document) {
	if facts == nil || doc == nil {
		return
	}
	applyStepProgressFacts(facts, doc.Steps)
	for _, rawLine := range strings.Split(doc.SectionText("Acceptance Criteria"), "\n") {
		line := strings.TrimSpace(rawLine)
		if len(line) < 6 || !strings.HasPrefix(line, "- [") || line[4] != ']' {
			continue
		}
		marker := line[3]
		if marker != ' ' && marker != 'x' && marker != 'X' {
			continue
		}
		facts.AcceptanceTotal++
		if marker == 'x' || marker == 'X' {
			facts.AcceptanceCompleted++
		}
	}
}

func applyStepProgressFacts(facts *Facts, steps []plan.DocumentStep) {
	if facts == nil {
		return
	}
	facts.StepTotal = len(steps)
	for index, step := range steps {
		if step.Done {
			facts.StepCompleted++
			continue
		}
		if facts.CurrentStepNumber == 0 {
			facts.CurrentStepNumber = index + 1
			facts.CurrentStep = step.Title
		}
	}
}

func resolveFinalizeNode(workdir, planStem string, doc *plan.Document, state *runstate.State, reviewCtx *reviewContext) (string, []StatusError) {
	reopenedNewStepPending := state != nil &&
		state.Reopen != nil &&
		state.Reopen.Mode == "new-step" &&
		state.Reopen.BaseStepCount > 0 &&
		len(doc.Steps) <= state.Reopen.BaseStepCount &&
		doc.CurrentStep() == nil &&
		doc.AllStepsCompleted()

	if reviewCtx != nil && reviewCtx.InFlight {
		return "execution/finalize/review", nil
	}
	if reopenedNewStepPending {
		return "execution/finalize/fix", nil
	}
	if state != nil && state.Reopen != nil && state.Reopen.Mode == "finalize-fix" {
		if !finalizeReviewSatisfied(workdir, planStem, state) {
			return "execution/finalize/fix", nil
		}
	}
	if reviewCtx != nil && reviewCtx.Aggregated &&
		(!reviewCtx.DecisionKnown || reviewCtx.Decision != "pass") {
		return "execution/finalize/fix", nil
	}
	if finalizeReviewSatisfied(workdir, planStem, state) {
		return "execution/finalize/archive", commandErrorsToStatusErrors(lifecycle.EvaluateArchiveReadiness(workdir, planStem, doc, state))
	}
	return "execution/finalize/review", nil
}

func coordinatedNewSubplanPending(state *runstate.State, progress plan.CoordinatedProgress) bool {
	return state != nil &&
		state.Reopen != nil &&
		state.Reopen.Mode == "new-step" &&
		progress.Total <= state.Reopen.BaseStepCount
}

func finalizeReviewSatisfied(workdir, planStem string, state *runstate.State) bool {
	if state == nil || state.FinalizeCoverage == nil || strings.TrimSpace(state.FinalizeCoverage.TipRoundID) == "" {
		return false
	}
	chain, err := reviewcoverage.Resolve(workdir, planStem, state.FinalizeCoverage.TipRoundID, runstate.CurrentRevision(state))
	if err != nil {
		return false
	}
	return chain.Decision == "pass" && chain.UnresolvedBlockingCount == 0
}

func effectiveReopenMode(doc *plan.Document, state *runstate.State) string {
	if state == nil || state.Reopen == nil {
		return ""
	}
	if state.Reopen.Mode != "new-step" {
		return state.Reopen.Mode
	}
	if state.Reopen.BaseStepCount > 0 && doc != nil && len(doc.Steps) > state.Reopen.BaseStepCount {
		return ""
	}
	return state.Reopen.Mode
}

func loadReviewContext(workdir, planStem string, state *runstate.State) (*reviewContext, []string) {
	if state == nil || state.ActiveReviewRound == nil {
		return nil, nil
	}

	round := state.ActiveReviewRound
	ctx := &reviewContext{
		RoundID:    round.RoundID,
		Kind:       round.Kind,
		Aggregated: round.Aggregated,
		InFlight:   !round.Aggregated,
	}
	warnings := make([]string, 0)

	revision, revisionKnown, err := runstate.EffectiveReviewRevision(workdir, planStem, round)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review revision for %s; status may be conservative.", round.RoundID))
	} else if revisionKnown {
		ctx.Revision = revision
	}

	manifestPath := filepath.Join(runstate.ReviewRoundDir(workdir, planStem, round.RoundID), "manifest.json")
	if data, err := os.ReadFile(manifestPath); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review handoff for %s; status may be incomplete.", round.RoundID))
	} else {
		var manifest contracts.ReviewManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			warnings = append(warnings, fmt.Sprintf("Unable to parse the review handoff for %s; status may be incomplete.", round.RoundID))
		} else {
			ctx.ReviewedHeadSHA = manifest.ReviewedHeadSHA
			if len(manifest.Assignments) == 1 {
				ctx.SubmissionPath = manifest.Assignments[0].SubmissionPath
			}
		}
	}

	if round.Aggregated {
		decision, known, err := runstate.EffectiveReviewDecision(workdir, planStem, round)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Unable to read the completed review decision for %s; review status may be stale.", round.RoundID))
		} else {
			ctx.Decision = decision
			ctx.DecisionKnown = known
		}
		if !ctx.DecisionKnown {
			warnings = append(warnings, fmt.Sprintf("The latest review outcome for %s could not be recovered; status is staying conservative.", round.RoundID))
		}
	}

	return ctx, warnings
}

func loadEvidenceContext(workdir, planStem string, revision int) (*evidenceContext, []string) {
	ctx := &evidenceContext{}
	warnings := make([]string, 0)

	if publish, err := evidence.LoadLatestPublish(workdir, planStem, revision); err != nil {
		warnings = append(warnings, "Unable to read publish evidence; publish status may be incomplete.")
	} else {
		ctx.Publish = publish
	}
	if ci, err := evidence.LoadLatestCI(workdir, planStem, revision); err != nil {
		warnings = append(warnings, "Unable to read CI evidence; CI status may be incomplete.")
	} else {
		ctx.CI = ci
	}
	if sync, err := evidence.LoadLatestSync(workdir, planStem, revision); err != nil {
		warnings = append(warnings, "Unable to read sync evidence; sync status may be incomplete.")
	} else {
		ctx.Sync = sync
	}

	return ctx, warnings
}

func applyEvidenceFacts(facts *Facts, evidenceCtx *evidenceContext) {
	if evidenceCtx == nil {
		return
	}
	if facts.Evidence == nil {
		facts.Evidence = &contracts.StatusEvidenceFacts{}
	}
	recorded := &contracts.StatusRecordedEvidence{}
	if evidenceCtx.Publish != nil {
		recorded.Publish = &contracts.StatusRecordedPublishEvidence{
			Status: evidenceCtx.Publish.Status,
			PRURL:  evidenceCtx.Publish.PRURL,
		}
	}
	if evidenceCtx.CI != nil {
		recorded.CI = &contracts.StatusRecordedEvidenceStatus{Status: evidenceCtx.CI.Status}
	}
	if evidenceCtx.Sync != nil {
		recorded.Sync = &contracts.StatusRecordedEvidenceStatus{Status: evidenceCtx.Sync.Status}
	}
	if recorded.Publish != nil || recorded.CI != nil || recorded.Sync != nil {
		facts.Evidence.Recorded = recorded
	}
}

func applyRemoteHandoffFacts(s Service, facts *Facts, evidenceCtx *evidenceContext) {
	if !s.ObserveRemote || facts == nil || evidenceCtx == nil || evidenceCtx.Publish == nil {
		return
	}
	if evidenceCtx.Publish.Status != "recorded" || strings.TrimSpace(evidenceCtx.Publish.PRURL) == "" {
		return
	}
	identity := remote.ParseRecordedPRURL(evidenceCtx.Publish.PRURL)
	if identity.Status != remote.PRStatusRecorded {
		ensureEvidenceFacts(facts).Remote = remoteEvidenceUnavailable(facts, identity.Degraded)
		return
	}
	observation := remote.Service{Workdir: s.Workdir, RunCommand: s.RunCommand}.ObserveHandoff(identity)
	ensureEvidenceFacts(facts).Remote = mapRemoteEvidenceObservation(facts, observation)
}

func ensureEvidenceFacts(facts *Facts) *contracts.StatusEvidenceFacts {
	if facts.Evidence == nil {
		facts.Evidence = &contracts.StatusEvidenceFacts{}
	}
	return facts.Evidence
}

func mapRemoteEvidenceObservation(facts *Facts, observation remote.HandoffObservation) *contracts.StatusRemoteEvidence {
	out := &contracts.StatusRemoteEvidence{
		Observation: remoteObservationCompleteness(observation),
		Degraded:    mapRemoteDegradations(observation.Degraded),
	}
	if observation.PR.Status == remote.PRObservationAvailable {
		out.PR = &contracts.StatusRemotePRSummary{
			State: observation.PR.State,
			Draft: observation.PR.IsDraft,
		}
	}
	if observation.CI.Status == remote.RemoteCIAvailable {
		out.CI = &contracts.StatusRemoteEvidenceStatus{Status: observation.CI.EvidenceStatus}
	}
	if observation.Sync.Status == remote.RemoteSyncAvailable {
		out.Sync = &contracts.StatusRemoteEvidenceStatus{Status: observation.Sync.EvidenceStatus}
	}
	if strings.TrimSpace(observation.Sync.MergePolicy) != "" {
		out.MergePolicy = &contracts.StatusRemoteEvidenceStatus{Status: observation.Sync.MergePolicy}
	}
	out.Assessment = remoteAssessment(facts, out)
	out.Message = remoteMessage(out)
	return out
}

func remoteAssessment(facts *Facts, remoteFacts *contracts.StatusRemoteEvidence) string {
	if remoteFacts == nil || remoteFacts.Observation == "unavailable" {
		return "manual_evidence_required"
	}
	if mergedRemotePR(remoteFacts) {
		return "merged_pending_land"
	}
	if remoteMergePolicyStatus(remoteFacts) == "blocked" {
		return "merge_policy_blocked"
	}
	if remoteMergePolicyStatus(remoteFacts) == "unknown" {
		return "wait_for_remote"
	}
	remoteCI := remoteCIStatusFrom(remoteFacts)
	remoteSync := remoteSyncStatusFrom(remoteFacts)
	if remoteCI == "" && remoteSync == "" {
		return "manual_evidence_required"
	}
	recordedReady := recordedPublishStatus(facts) == "recorded" &&
		strings.TrimSpace(recordedPRURL(facts)) != "" &&
		(recordedCIStatus(facts) == "success" || recordedCIStatus(facts) == "not_applied") &&
		(recordedSyncStatus(facts) == "fresh" || recordedSyncStatus(facts) == "not_applied")
	if recordedReady && remoteNonReady(remoteFacts) {
		return "candidate_invalidated"
	}
	if unusableRemotePR(remoteFacts) {
		if recordedReady {
			return "candidate_invalidated"
		}
		return "manual_evidence_required"
	}
	if remoteFacts.PR != nil && remoteFacts.PR.Draft {
		return "wait_for_remote"
	}
	if remoteCI == "pending" {
		return "wait_for_remote"
	}
	if remoteCI == "failed" || remoteSync == "stale" || remoteSync == "conflicted" {
		return "repair_remote"
	}
	if remoteCanRefreshRecorded(facts, remoteFacts) {
		return "refresh_available"
	}
	if remoteFacts.Observation == "partial" {
		return "manual_evidence_required"
	}
	return "matches_recorded"
}

func unusableRemotePR(remoteFacts *contracts.StatusRemoteEvidence) bool {
	if remoteFacts == nil || remoteFacts.PR == nil {
		return false
	}
	state := strings.ToUpper(strings.TrimSpace(remoteFacts.PR.State))
	return state != "" && state != "OPEN"
}

func mergedRemotePR(remoteFacts *contracts.StatusRemoteEvidence) bool {
	return remoteFacts != nil && remoteFacts.PR != nil &&
		strings.EqualFold(strings.TrimSpace(remoteFacts.PR.State), "MERGED")
}

func remoteMessage(remoteFacts *contracts.StatusRemoteEvidence) string {
	if remoteFacts == nil {
		return ""
	}
	switch remoteFacts.Assessment {
	case "matches_recorded":
		return "Remote PR facts match the recorded evidence that already drives the workflow node."
	case "refresh_available":
		return "Remote PR facts can be recorded as durable CI and sync evidence."
	case "wait_for_remote":
		return "Remote PR facts are not ready yet; wait before recording final evidence."
	case "repair_remote":
		return "Remote PR facts show CI or sync repair is needed before merge-ready handoff."
	case "merge_policy_blocked":
		return "The branch is current, but provider approvals or merge policy still block merge."
	case "manual_evidence_required":
		return "Remote PR facts are unavailable or incomplete; use manual evidence fallback when the facts are known."
	case "candidate_invalidated":
		return "Recorded evidence is merge-ready, but live remote facts show the candidate should be repaired or refreshed before merge approval."
	case "merged_pending_land":
		return "The recorded PR is already merged; record the merge through harness and finish post-merge bookkeeping."
	default:
		return ""
	}
}

func remoteCanRefreshRecorded(facts *Facts, remoteFacts *contracts.StatusRemoteEvidence) bool {
	remoteCI := remoteCIStatusFrom(remoteFacts)
	remoteSync := remoteSyncStatusFrom(remoteFacts)
	recordedCI := recordedCIStatus(facts)
	recordedSync := recordedSyncStatus(facts)
	ciCanRefresh := remoteCI != "" && remoteCI != "pending" && recordedCI != "not_applied" && remoteCI != recordedCI
	syncCanRefresh := remoteSync != "" && recordedSync != "not_applied" && remoteSync != recordedSync
	return ciCanRefresh || syncCanRefresh
}

func remoteNonReady(remoteFacts *contracts.StatusRemoteEvidence) bool {
	ci := remoteCIStatusFrom(remoteFacts)
	sync := remoteSyncStatusFrom(remoteFacts)
	return ci == "pending" || ci == "failed" || sync == "stale" || sync == "conflicted" ||
		(remoteFacts != nil && remoteFacts.PR != nil && remoteFacts.PR.Draft)
}

func recordedPublishStatus(facts *Facts) string {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Recorded == nil || facts.Evidence.Recorded.Publish == nil {
		return ""
	}
	return facts.Evidence.Recorded.Publish.Status
}

func recordedPRURL(facts *Facts) string {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Recorded == nil || facts.Evidence.Recorded.Publish == nil {
		return ""
	}
	return facts.Evidence.Recorded.Publish.PRURL
}

func recordedCIStatus(facts *Facts) string {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Recorded == nil || facts.Evidence.Recorded.CI == nil {
		return ""
	}
	return facts.Evidence.Recorded.CI.Status
}

func recordedSyncStatus(facts *Facts) string {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Recorded == nil || facts.Evidence.Recorded.Sync == nil {
		return ""
	}
	return facts.Evidence.Recorded.Sync.Status
}

func remoteCIStatus(facts *Facts) string {
	if facts == nil || facts.Evidence == nil {
		return ""
	}
	return remoteCIStatusFrom(facts.Evidence.Remote)
}

func remoteSyncStatus(facts *Facts) string {
	if facts == nil || facts.Evidence == nil {
		return ""
	}
	return remoteSyncStatusFrom(facts.Evidence.Remote)
}

func remoteCIStatusFrom(remoteFacts *contracts.StatusRemoteEvidence) string {
	if remoteFacts == nil || remoteFacts.CI == nil {
		return ""
	}
	return remoteFacts.CI.Status
}

func remoteSyncStatusFrom(remoteFacts *contracts.StatusRemoteEvidence) string {
	if remoteFacts == nil || remoteFacts.Sync == nil {
		return ""
	}
	return remoteFacts.Sync.Status
}

func remoteMergePolicyStatus(remoteFacts *contracts.StatusRemoteEvidence) string {
	if remoteFacts == nil || remoteFacts.MergePolicy == nil {
		return ""
	}
	return strings.TrimSpace(remoteFacts.MergePolicy.Status)
}

func remoteEvidenceUnavailable(facts *Facts, degradation remote.Degradation) *contracts.StatusRemoteEvidence {
	out := &contracts.StatusRemoteEvidence{
		Observation: "unavailable",
		Degraded:    mapRemoteDegradations([]remote.Degradation{degradation}),
	}
	out.Assessment = remoteAssessment(facts, out)
	out.Message = remoteMessage(out)
	return out
}

func remoteObservationCompleteness(observation remote.HandoffObservation) string {
	switch observation.Status {
	case remote.HandoffObservationAvailable:
		return "complete"
	case remote.HandoffObservationDegraded:
		return "partial"
	default:
		return "unavailable"
	}
}

func mapRemoteDegradations(degradations []remote.Degradation) []contracts.StatusRemoteDegradation {
	out := make([]contracts.StatusRemoteDegradation, 0, len(degradations))
	for _, degradation := range degradations {
		if strings.TrimSpace(degradation.Code) == "" {
			continue
		}
		out = append(out, contracts.StatusRemoteDegradation{
			Code:    degradation.Code,
			Message: degradation.Message,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func archivedCandidateReadyForMerge(evidenceCtx *evidenceContext) bool {
	if evidenceCtx == nil || evidenceCtx.Publish == nil || evidenceCtx.CI == nil || evidenceCtx.Sync == nil {
		return false
	}
	if evidenceCtx.Publish.Status != "recorded" || strings.TrimSpace(evidenceCtx.Publish.PRURL) == "" {
		return false
	}
	if evidenceCtx.CI.Status != "success" && evidenceCtx.CI.Status != "not_applied" {
		return false
	}
	if evidenceCtx.Sync.Status != "fresh" && evidenceCtx.Sync.Status != "not_applied" {
		return false
	}
	return true
}

func buildSummary(node string, facts *Facts, reviewCtx *reviewContext, blockers []StatusError, currentPlan *runstate.CurrentPlan, planApproved bool) string {

	switch node {
	case "idle":
		if currentPlan != nil && strings.TrimSpace(currentPlan.LastLandedPlanPath) != "" {
			return "No current plan is active in this worktree. The most recent landed candidate is recorded for handoff context."
		}
		return "No current plan is active in this worktree."
	case "plan":
		if planApproved {
			return "Current plan is approved and ready for execution to start."
		}
		return "Current plan exists, but execution is still waiting for explicit human approval."
	case "execution/coordinate":
		if facts == nil || facts.Subplans == nil || facts.Subplans.Total == 0 {
			return "Coordinated execution has started, but the root does not have any subplans yet."
		}
		if facts.ReopenMode == "new-step" && facts.Subplans.Completed == facts.Subplans.Total {
			return "The coordinated root was reopened for new work and needs a new subplan before it can return to finalize review."
		}
		if len(blockers) > 0 {
			return fmt.Sprintf("Coordinated execution has %d dependency blocker(s) that must be fixed before the root can finalize.", len(blockers))
		}
		return fmt.Sprintf(
			"Coordinated execution is progressing across %d subplans: %d complete, %d runnable, and %d waiting on dependencies.",
			facts.Subplans.Total,
			facts.Subplans.Completed,
			facts.Subplans.Runnable,
			facts.Subplans.Waiting,
		)
	case "execution/finalize/review":
		if reviewCtx != nil && reviewCtx.InFlight {
			return "Plan is in finalize review and waiting for the integrated reviewer to submit its complete judgment."
		}
		if acceptanceIncomplete(facts) {
			if facts != nil && facts.Subplans != nil {
				return "Plan has finished its coordinated subplans, but acceptance criteria are still incomplete and finalize review cannot start yet."
			}
			return "Plan has finished its tracked steps, but acceptance criteria are still incomplete and finalize review cannot start yet."
		}
		if facts != nil && facts.Subplans != nil {
			return "Plan has finished its coordinated subplans and needs finalize review before archive."
		}
		return "Plan has finished its tracked steps and needs finalize review before archive."
	case "execution/finalize/fix":
		if facts != nil && facts.ReopenMode == "new-step" && facts.CurrentStep == "" {
			return "Plan was reopened for new-scope work and needs a new unfinished step before implementation can continue."
		}
		if facts != nil && facts.ReopenMode == "finalize-fix" {
			return "Plan was reopened into finalize-scope repair and needs follow-up fixes plus a fresh finalize review before archive."
		}
		if facts != nil && facts.ReviewStatus == "unknown" && reviewCtx != nil {
			return fmt.Sprintf("Plan needs finalize follow-up because the latest review decision (%s) could not be recovered from local state.", reviewCtx.RoundID)
		}
		if reviewCtx != nil && facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" {
			return fmt.Sprintf("Plan needs finalize-scope repair because the latest finalize review (%s) requested changes.", reviewCtx.RoundID)
		}
		return "Plan needs finalize-scope repair before archive."
	case "execution/finalize/archive":
		if len(blockers) > 0 {
			return fmt.Sprintf("Plan has a clean finalize review and is in archive closeout, but %d archive blocker(s) still need to be fixed before `harness archive`.", len(blockers))
		}
		return "Plan has a clean finalize review and is ready to archive."
	case "execution/finalize/publish":
		return "Plan is archived, but external publish, CI, or sync evidence is still keeping it from merge-ready handoff."
	case "execution/finalize/await_merge":
		return "Plan is archived, published, and merge-ready; waiting for human merge approval."
	case "land":
		return "Merge has been recorded and required post-merge bookkeeping is still in progress."
	}

	if strings.HasSuffix(node, "/implement") {
		return fmt.Sprintf("Plan is executing %s.", facts.CurrentStep)
	}

	return fmt.Sprintf("Plan is at %s.", node)
}

func buildNextActions(node string, facts *Facts, reviewCtx *reviewContext, blockers []StatusError, planApproved bool) []NextAction {
	switch node {
	case "idle":
		return []NextAction{
			{Command: nil, Description: "Start discovery or create a new tracked plan when the next slice is ready."},
		}
	case "plan":
		if !planApproved {
			return []NextAction{
				{Command: nil, Description: "Ask the human to approve the tracked plan before execution begins."},
				{Command: strPtr("harness plan approve --by human"), Description: "After the human approves in chat, record that approval boundary on the tracked plan."},
				{Command: nil, Description: "If scope changed before approval, update the tracked plan first."},
			}
		}
		return []NextAction{
			{Command: strPtr("harness execute start"), Description: "Start execution now that the tracked plan is approved for implementation."},
			{Command: nil, Description: "If scope changed after approval but before implementation begins, update the tracked plan and refresh approval before executing."},
		}
	case "execution/coordinate":
		if len(blockers) > 0 {
			return []NextAction{{Command: nil, Description: "Fix the coordinated subplan dependency blockers, then rerun `harness status` before continuing toward finalize review."}}
		}
		if facts != nil && facts.Subplans != nil &&
			facts.ReopenMode == "new-step" &&
			facts.Subplans.Completed == facts.Subplans.Total {
			return []NextAction{{Command: nil, Description: "Add a new sibling subplan for the reopened work, then complete its ordered steps and result before returning to finalize review."}}
		}
		if facts == nil || facts.Subplans == nil || facts.Subplans.Total == 0 {
			return []NextAction{{Command: nil, Description: "Create at least one flat subplan inside the coordinated root package before continuing execution."}}
		}
		return []NextAction{
			{Command: nil, Description: "Continue the runnable subplans in parallel where their file ownership does not overlap, and serialize Git mutations through the controller."},
			{Command: strPtr("harness status --plan <subplan-id-or-path>"), Description: "Inspect one subplan's ordered steps and unresolved dependencies without changing the current root plan."},
		}
	case "execution/finalize/review":
		if reviewCtx != nil && reviewCtx.InFlight {
			return []NextAction{
				{Command: reviewSubmitCommand(reviewCtx), Description: "Have the independent integrated reviewer submit its complete judgment for the active finalize round."},
				{Command: strPtr(fmt.Sprintf("harness review abort --round %s", reviewCtx.RoundID)), Description: "Abort this unfinished round only when it cannot be completed, then start a replacement review without editing local state."},
			}
		}
		if acceptanceIncomplete(facts) {
			return []NextAction{{Command: nil, Description: "Validate the remaining outcomes and check every acceptance criterion before starting finalize review."}}
		}
		return []NextAction{
			{Command: strPtr("harness review start"), Description: "Start the mandatory integrated full review for the committed complete candidate."},
		}
	case "execution/finalize/fix":
		if facts != nil && facts.ReopenMode == "new-step" && facts.CurrentStep == "" {
			return []NextAction{
				{Command: nil, Description: "Add a new unfinished step for the reopened scope before continuing implementation; do not fold the new work into already completed steps."},
			}
		}
		description := "Repair the finalize-scope issues, refresh the Closeout as needed, rerun focused validation, and start the linked finalize review before archive."
		if reviewCtx != nil && facts != nil && facts.ReviewStatus == "unknown" {
			description = fmt.Sprintf("Recover or replace %s before continuing. Its finalize review decision could not be recovered, so archive-sensitive guidance is intentionally blocked.", reviewCtx.RoundID)
		}
		if reviewCtx != nil && facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" && facts.ReviewStatus != "unknown" {
			description = fmt.Sprintf("Address the findings from %s, refresh the Closeout as needed, rerun focused validation, and start the linked delta once the narrow repair is committed.", reviewCtx.RoundID)
		}
		return []NextAction{
			{Command: nil, Description: description},
			{Command: strPtr("harness review start"), Description: "Start the inferred linked delta for a narrow repair; use `harness review start --full` only after a material design, scope, or risk change."},
		}
	case "execution/finalize/archive":
		if len(blockers) > 0 {
			return []NextAction{
				{Command: nil, Description: "Fix the archive blockers surfaced below, refresh the durable summaries, and rerun `harness status` before archiving."},
			}
		}
		return []NextAction{
			{Command: nil, Description: "Archive-ready closeout is complete; archive the plan and then commit and push the tracked move."},
			{Command: strPtr("harness archive"), Description: "Archive the current plan now that the closeout notes and follow-up links are ready."},
		}
	case "execution/finalize/publish":
		if len(blockers) > 0 {
			return []NextAction{
				{Command: nil, Description: "The archived branch no longer matches its reviewed candidate. Reopen with `harness reopen --mode finalize-fix`, review the current candidate, and archive it again before merge handoff."},
			}
		}
		return buildPublishNextActions(facts)
	case "execution/finalize/await_merge":
		actions := remoteHandoffNextActions(facts)
		if !mergedRemotePR(remoteEvidenceFacts(facts)) {
			if remoteBlocksMergeApproval(facts) {
				actions = append(actions, NextAction{
					Command:     nil,
					Description: "Repair or refresh the remote handoff before asking for merge approval.",
				})
			} else {
				actions = append(actions, NextAction{Command: nil, Description: "Wait for explicit human approval before merging the PR."})
				if facts != nil && strings.TrimSpace(recordedPRURL(facts)) != "" {
					actions = append(actions, NextAction{
						Command:     strPtr(fmt.Sprintf("harness land --pr %s [--commit <sha>]", recordedPRURL(facts))),
						Description: "After the PR is merged outside harness and the worktree is synced, record merge confirmation and enter required post-merge bookkeeping.",
					})
				}
			}
		}
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "If new feedback or remote changes invalidate the archived candidate, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step.",
		})
		return actions
	case "land":
		return []NextAction{
			{Command: nil, Description: "Finish required post-merge bookkeeping and cleanup while the plan is in land: rely on the forge merge record when it is sufficient; add a PR comment or issue update only for material unresolved, deployment, or follow-up context; then sync local branches."},
			{Command: strPtr("harness land complete"), Description: "Record required post-merge bookkeeping completion only after any necessary durable handoff and local cleanup are done, then restore the worktree to idle."},
		}
	}

	if strings.HasSuffix(node, "/implement") {
		return []NextAction{
			{Command: nil, Description: "Continue the current outcome and mark the step done after its concise check passes."},
		}
	}

	return nil
}

func acceptanceIncomplete(facts *Facts) bool {
	return facts != nil && facts.AcceptanceTotal > 0 && facts.AcceptanceCompleted < facts.AcceptanceTotal
}

func buildPublishNextActions(facts *Facts) []NextAction {
	actions := []NextAction{
		{
			Command:     nil,
			Description: "Commit and push the tracked plan change created by archiving before treating the candidate as merge-ready.",
		},
	}
	actions = append(actions, remoteHandoffNextActions(facts)...)

	switch {
	case facts == nil || recordedPublishStatus(facts) == "":
		actions = append(actions,
			NextAction{Command: nil, Description: "Open or update the PR for the archived candidate, then record publish evidence with the PR URL."},
			NextAction{Command: strPtr("harness evidence submit --kind publish --input <json>"), Description: "Record publish evidence for the archived candidate once the PR or handoff record exists."},
		)
	case recordedPublishStatus(facts) == "not_applied":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Publish was marked not_applied, but v0.2 land still requires a PR URL; record publish evidence with a PR URL or reopen if the workflow changed.",
		})
	case shouldSuggestEvidenceRefresh(facts):
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence refresh"),
			Description: "Refresh CI and sync evidence from the recorded PR URL, including re-checking pending CI or non-fresh sync state.",
		})
	}

	switch {
	case facts == nil || recordedCIStatus(facts) == "":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind ci --input <json>"),
			Description: "Record CI evidence once the relevant post-archive check result is known.",
		})
	case recordedCIStatus(facts) == "pending":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind ci --input <json>"),
			Description: "Wait for the relevant post-archive CI to finish, then manually record the updated result if refresh is unavailable.",
		})
	case recordedCIStatus(facts) == "failed":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Fix the CI failures or record an explicit not_applied decision before treating the candidate as merge-ready.",
		})
	}

	switch {
	case facts == nil || recordedSyncStatus(facts) == "":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind sync --input <json>"),
			Description: "Record sync evidence after checking freshness and conflict status against the merge base.",
		})
	case recordedSyncStatus(facts) == "stale":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind sync --input <json>"),
			Description: "Refresh the branch against the merge base, then manually record a fresh sync result if refresh is unavailable.",
		})
	case recordedSyncStatus(facts) == "conflicted":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Resolve merge conflicts or otherwise repair the branch, then record a fresh sync result before merge approval.",
		})
	}

	actions = append(actions, NextAction{
		Command:     nil,
		Description: "If the archived candidate is invalidated, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step.",
	})

	return actions
}

func remoteHandoffNextActions(facts *Facts) []NextAction {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Remote == nil {
		return nil
	}
	remoteFacts := facts.Evidence.Remote
	actions := make([]NextAction, 0, 2)
	if mergedRemotePR(remoteFacts) {
		return append(actions, NextAction{
			Command:     strPtr(fmt.Sprintf("harness land --pr %s [--commit <sha>]", recordedPRURL(facts))),
			Description: "The recorded PR is already merged. After confirming the merge was explicitly human-approved, record it and enter required post-merge bookkeeping.",
		})
	}
	if unusableRemotePR(remoteFacts) {
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Recorded PR is no longer open; repair or replace the publish handoff before recording merge-ready evidence.",
		})
		return actions
	}
	if remoteFacts.CI != nil {
		switch remoteFacts.CI.Status {
		case "pending":
			actions = append(actions, NextAction{
				Command:     nil,
				Description: "Remote PR checks are still pending; wait for them to finish before recording final CI and sync evidence.",
			})
		case "failed":
			actions = append(actions, NextAction{
				Command:     nil,
				Description: "Remote PR checks are failing; fix CI before recording merge-ready evidence.",
			})
		}
	}
	if remoteFacts.Sync != nil {
		switch remoteFacts.Sync.Status {
		case "stale":
			actions = append(actions, NextAction{
				Command:     nil,
				Description: "Remote PR merge state is stale; refresh the branch against the base before recording fresh sync evidence.",
			})
		case "conflicted":
			actions = append(actions, NextAction{
				Command:     nil,
				Description: "Remote PR merge state is conflicted; resolve the conflict before recording fresh sync evidence.",
			})
		}
	}
	if remoteMergePolicyStatus(remoteFacts) == "blocked" {
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "The branch is current, but provider approvals or merge policy still block merge; satisfy that policy without refreshing the branch.",
		})
	}
	return actions
}

func shouldSuggestEvidenceRefresh(facts *Facts) bool {
	if facts == nil || recordedPublishStatus(facts) != "recorded" || strings.TrimSpace(recordedPRURL(facts)) == "" {
		return false
	}
	if remoteFacts := remoteEvidenceFacts(facts); remoteFacts != nil {
		return remoteFacts.Assessment == "refresh_available"
	}
	return recordedCIStatus(facts) == "" ||
		recordedCIStatus(facts) == "pending" ||
		recordedSyncStatus(facts) == "" ||
		recordedSyncStatus(facts) == "stale" ||
		cleanRemoteCanRefreshNonReadyEvidence(facts)
}

func remoteBlocksMergeApproval(facts *Facts) bool {
	remoteFacts := remoteEvidenceFacts(facts)
	if remoteFacts == nil {
		return false
	}
	switch remoteFacts.Assessment {
	case "candidate_invalidated", "repair_remote", "wait_for_remote", "merge_policy_blocked":
		return true
	default:
		return unusableRemotePR(remoteFacts)
	}
}

func remoteEvidenceFacts(facts *Facts) *contracts.StatusRemoteEvidence {
	if facts == nil || facts.Evidence == nil {
		return nil
	}
	return facts.Evidence.Remote
}

func cleanRemoteCanRefreshNonReadyEvidence(facts *Facts) bool {
	if facts == nil || facts.Evidence == nil || facts.Evidence.Remote == nil {
		return false
	}
	return (recordedCIStatus(facts) == "failed" && remoteCIStatus(facts) == "success") ||
		(recordedSyncStatus(facts) == "conflicted" && remoteSyncStatus(facts) == "fresh")
}

func idleResult(workdir string, currentPlan *runstate.CurrentPlan) Result {
	result := Result{
		OK:      true,
		Command: "status",
		State: State{
			CurrentNode: "idle",
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Start discovery or create a new tracked plan when the next slice is ready."},
		},
	}
	if currentPlan != nil && strings.TrimSpace(currentPlan.LastLandedPlanPath) != "" {
		if lastLandedPlanPath, ok := repoRelativePointerPath(currentPlan.LastLandedPlanPath); ok {
			result.Summary = "No current plan is active in this worktree. The most recent landed candidate is recorded for handoff context."
			result.Artifacts = &Artifacts{
				PlanPath:     lastLandedPlanPath,
				LastLandedAt: currentPlan.LastLandedAt,
			}
		}
	}
	if result.Summary == "" {
		result.Summary = "No current plan is active in this worktree."
	}

	decorateRepoBootstrapDrift(workdir, &result, true)

	return result
}

func decorateRepoBootstrapDrift(workdir string, result *Result, prioritizeAction bool) {
	if result == nil || !result.OK {
		return
	}
	drift, err := install.Service{Workdir: workdir}.InspectRepoBootstrapDrift("codex")
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unable to inspect the default repo bootstrap assets: %v", err))
		return
	}
	if drift.Stale() {
		result.Warnings = append(result.Warnings, buildBootstrapDriftWarning(drift))
		if result.Facts == nil {
			result.Facts = &Facts{}
		}
		result.Facts.ManagedResources = &contracts.StatusManagedResources{
			Status:               "stale",
			Agent:                "codex",
			InstructionsStale:    drift.InstructionsStale,
			StaleSkillPackages:   drift.StaleManagedSkillPackages,
			MissingSkillPackages: drift.MissingManagedSkillPackages,
			ExtraSkillPackages:   drift.ExtraManagedSkillPackages,
		}
		command := "harness repo init --dry-run"
		action := NextAction{
			Command:     &command,
			Description: "Optionally inspect the default repo resource refresh without writing files. Keep any refresh out of the current candidate unless it is approved scope; otherwise refresh or rebase separately.",
		}
		if prioritizeAction {
			result.NextAction = append([]NextAction{action}, result.NextAction...)
		} else {
			result.NextAction = append(result.NextAction, action)
		}
	}
}

func repoFacingPath(workdir, path string) string {
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

func repoRelativePointerPath(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return "", false
	}
	relPath := filepath.Clean(filepath.FromSlash(trimmed))
	if relPath == "." || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relPath), true
}

func buildBootstrapDriftWarning(drift install.RepoBootstrapDrift) string {
	affected := make([]string, 0, 3)
	if drift.InstructionsStale {
		affected = append(affected, "the AGENTS.md managed block")
	}
	if staleCount := len(drift.StaleManagedSkillPackages); staleCount > 0 {
		if staleCount == 1 {
			affected = append(affected, "1 managed skill package")
		} else {
			affected = append(affected, fmt.Sprintf("%d managed skill packages", staleCount))
		}
	}
	if missingCount := len(drift.MissingManagedSkillPackages); missingCount > 0 {
		if missingCount == 1 {
			affected = append(affected, "1 missing managed skill package")
		} else {
			affected = append(affected, fmt.Sprintf("%d missing managed skill packages", missingCount))
		}
	}
	if extraCount := len(drift.ExtraManagedSkillPackages); extraCount > 0 {
		if extraCount == 1 {
			affected = append(affected, "1 stale managed skill package that is no longer in the packaged bootstrap set")
		} else {
			affected = append(affected, fmt.Sprintf("%d stale managed skill packages that are no longer in the packaged bootstrap set", extraCount))
		}
	}
	if len(affected) == 0 {
		return "The default repo bootstrap assets appear stale relative to the running easyharness binary. This is a non-blocking reminder for the agent and does not change workflow state."
	}
	return fmt.Sprintf("The default repo bootstrap assets for Codex are stale relative to the running easyharness binary (%s). This is a non-blocking reminder for the agent and does not change workflow state.", strings.Join(affected, ", "))
}

func currentStepIndex(doc *plan.Document) int {
	currentStep := doc.CurrentStep()
	if currentStep == nil {
		return -1
	}
	for index, step := range doc.Steps {
		if step.Title == currentStep.Title {
			return index
		}
	}
	return -1
}

func landInProgress(state *runstate.State) bool {
	return state != nil &&
		state.Land != nil &&
		strings.TrimSpace(state.Land.LandedAt) != "" &&
		strings.TrimSpace(state.Land.CompletedAt) == ""
}

func stepNode(index int) string {
	return fmt.Sprintf("execution/step-%d/implement", index+1)
}

func commandErrorsToStatusErrors(errors []lifecycle.CommandError) []StatusError {
	out := make([]StatusError, 0, len(errors))
	for _, issue := range errors {
		out = append(out, StatusError{
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	return out
}

func reviewSubmitCommand(reviewCtx *reviewContext) *string {
	if reviewCtx == nil || strings.TrimSpace(reviewCtx.RoundID) == "" {
		return strPtr("harness review submit --round <round-id> --by integrated --input <path>")
	}
	inputPath := strings.TrimSpace(reviewCtx.SubmissionPath)
	if inputPath == "" {
		inputPath = "<path>"
	}
	return strPtr(fmt.Sprintf("harness review submit --round %s --by integrated --input %s", reviewCtx.RoundID, inputPath))
}

func strPtr(value string) *string {
	return &value
}

func factsEmpty(f *Facts) bool {
	if f == nil {
		return true
	}
	return strings.TrimSpace(f.CurrentStep) == "" &&
		f.CurrentStepNumber == 0 &&
		f.StepCompleted == 0 &&
		f.StepTotal == 0 &&
		f.AcceptanceCompleted == 0 &&
		f.AcceptanceTotal == 0 &&
		f.Revision == 0 &&
		strings.TrimSpace(f.ReopenMode) == "" &&
		strings.TrimSpace(f.ReviewKind) == "" &&
		strings.TrimSpace(f.ReviewStatus) == "" &&
		strings.TrimSpace(f.ReviewedHeadSHA) == "" &&
		f.ArchiveBlockerCount == 0 &&
		f.Subplans == nil &&
		f.SelectedSubplan == nil &&
		f.Evidence == nil &&
		f.ManagedResources == nil &&
		strings.TrimSpace(f.LandPRURL) == "" &&
		strings.TrimSpace(f.LandCommit) == ""
}

func documentIssuesToStatusErrors(issues []plan.DocumentIssue) []StatusError {
	if len(issues) == 0 {
		return nil
	}
	out := make([]StatusError, 0, len(issues))
	for _, issue := range issues {
		out = append(out, StatusError{Path: issue.Path, Message: issue.Message})
	}
	return out
}
