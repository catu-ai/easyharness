package runstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/repoconfig"
)

var renameFile = os.Rename

type CurrentPlan = contracts.CurrentPlanFile
type State = contracts.LocalStateFile
type ReopenState = contracts.ReopenState
type ReviewRound = contracts.ReviewRoundState
type LandState = contracts.LandState

type reviewAggregate struct {
	Decision string `json:"decision"`
}

type reviewManifest struct {
	ReviewTitle string `json:"review_title,omitempty"`
	Step        *int   `json:"step,omitempty"`
	Revision    int    `json:"revision,omitempty"`
}

func LoadCurrentPlan(workdir string) (*CurrentPlan, error) {
	path := CurrentPlanPath(workdir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var current CurrentPlan
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, fmt.Errorf("parse current-plan.json: %w", err)
	}
	return &current, nil
}

func SaveCurrentPlan(workdir, planPath string) (string, error) {
	return saveCurrentPlan(workdir, CurrentPlan{PlanPath: planPath})
}

func SaveLandedPlan(workdir, planPath, landedAt string) (string, error) {
	return saveCurrentPlan(workdir, CurrentPlan{
		LastLandedPlanPath: planPath,
		LastLandedAt:       landedAt,
	})
}

func WriteCurrentPlan(workdir string, current *CurrentPlan) (string, error) {
	path := CurrentPlanPath(workdir)
	if current == nil {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		return path, nil
	}
	return saveCurrentPlan(workdir, *current)
}

func saveCurrentPlan(workdir string, current CurrentPlan) (string, error) {
	path := CurrentPlanPath(workdir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal current-plan.json: %w", err)
	}
	if err := writeJSONAtomic(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func LoadState(workdir, planStem string) (*State, string, error) {
	path := StatePath(workdir, planStem)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, nil
		}
		return nil, path, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, path, fmt.Errorf("parse state.json: %w", err)
	}
	return &state, path, nil
}

func SaveState(workdir, planStem string, state *State) (string, error) {
	path := StatePath(workdir, planStem)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal state.json: %w", err)
	}
	if err := writeJSONAtomic(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func AcquireStateMutationLock(workdir, planStem string) (func(), error) {
	return acquirePlanFileLock(workdir, planStem, ".state-mutation.lock",
		fmt.Sprintf("another command is already mutating local state for plan %q; retry after it finishes", planStem))
}

func StateMutationLockHeld(workdir, planStem string) (bool, error) {
	return planFileLockHeld(workdir, planStem, ".state-mutation.lock")
}

func WaitForStateMutationLockRelease(workdir, planStem string, timeout, pollInterval time.Duration) (bool, error) {
	if timeout <= 0 {
		return StateMutationLockHeld(workdir, planStem)
	}
	if pollInterval <= 0 {
		pollInterval = 25 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for {
		held, err := StateMutationLockHeld(workdir, planStem)
		if err != nil || !held {
			return held, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return true, nil
		}
		sleepFor := pollInterval
		if remaining < sleepFor {
			sleepFor = remaining
		}
		time.Sleep(sleepFor)
	}
}

func AcquireTimelineMutationLock(workdir, planStem string) (func(), error) {
	return acquirePlanFileLock(workdir, planStem, ".timeline-mutation.lock",
		fmt.Sprintf("another command is already appending timeline events for plan %q; retry after it finishes", planStem))
}

func writeJSONAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		if err == nil {
			return
		}
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(perm); err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := renameFile(tempPath, path); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	if err := dirFile.Sync(); err != nil {
		return err
	}
	return nil
}

func acquirePlanFileLock(workdir, planStem, lockFileName, contentionMessage string) (func(), error) {
	lockPath := PlanRuntimePath(workdir, planStem, lockFileName)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.New(contentionMessage)
		}
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func planFileLockHeld(workdir, planStem, lockFileName string) (bool, error) {
	lockPath := PlanRuntimePath(workdir, planStem, lockFileName)
	file, err := os.OpenFile(lockPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return true, nil
		}
		return false, err
	}
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false, nil
}

func RuntimeRoot(workdir string) string {
	config := repoconfig.Load(workdir).Config
	return filepath.Join(workdir, filepath.FromSlash(config.Paths.LocalRuntime))
}

func CurrentPlanPath(workdir string) string {
	return filepath.Join(RuntimeRoot(workdir), "current-plan.json")
}

func PlanRuntimeDir(workdir, planStem string) string {
	return filepath.Join(RuntimeRoot(workdir), "plans", planStem)
}

func PlanRuntimePath(workdir, planStem string, elems ...string) string {
	parts := append([]string{PlanRuntimeDir(workdir, planStem)}, elems...)
	return filepath.Join(parts...)
}

func StatePath(workdir, planStem string) string {
	return PlanRuntimePath(workdir, planStem, "state.json")
}

func ReviewsDir(workdir, planStem string) string {
	return PlanRuntimePath(workdir, planStem, "reviews")
}

func ReviewRoundDir(workdir, planStem, roundID string) string {
	return PlanRuntimePath(workdir, planStem, "reviews", roundID)
}

func EvidenceKindDir(workdir, planStem, kind string) string {
	return PlanRuntimePath(workdir, planStem, "evidence", kind)
}

func TimelineEventsPath(workdir, planStem string) string {
	return PlanRuntimePath(workdir, planStem, "events.jsonl")
}

func LightweightArchivedPlansDir(workdir string) string {
	return filepath.Join(RuntimeRoot(workdir), "plans", "archived")
}

func CurrentRevision(state *State) int {
	if state == nil || state.Revision <= 0 {
		return 1
	}
	return state.Revision
}

func EffectiveReviewDecision(workdir, planStem string, round *ReviewRound) (string, bool, error) {
	if round == nil {
		return "", false, nil
	}
	if decision := strings.TrimSpace(round.Decision); decision != "" {
		return decision, true, nil
	}
	if !round.Aggregated || strings.TrimSpace(round.RoundID) == "" {
		return "", false, nil
	}

	path := filepath.Join(ReviewRoundDir(workdir, planStem, round.RoundID), "aggregate.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read aggregate.json for %s: %w", round.RoundID, err)
	}

	var aggregate reviewAggregate
	if err := json.Unmarshal(data, &aggregate); err != nil {
		return "", false, fmt.Errorf("parse aggregate.json for %s: %w", round.RoundID, err)
	}
	if decision := strings.TrimSpace(aggregate.Decision); decision != "" {
		return decision, true, nil
	}
	return "", false, nil
}

func EffectiveReviewTitle(workdir, planStem string, round *ReviewRound) (string, bool, error) {
	if round == nil || strings.TrimSpace(round.RoundID) == "" {
		return "", false, nil
	}

	path := filepath.Join(ReviewRoundDir(workdir, planStem, round.RoundID), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read manifest.json for %s: %w", round.RoundID, err)
	}

	var manifest reviewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", false, fmt.Errorf("parse manifest.json for %s: %w", round.RoundID, err)
	}
	if reviewTitle := strings.TrimSpace(manifest.ReviewTitle); reviewTitle != "" {
		return reviewTitle, true, nil
	}
	return "", false, nil
}

func EffectiveReviewStep(workdir, planStem string, round *ReviewRound) (int, bool, error) {
	if round == nil {
		return 0, false, nil
	}
	if round.Step != nil {
		return *round.Step, true, nil
	}
	if strings.TrimSpace(round.RoundID) == "" {
		return 0, false, nil
	}

	path := filepath.Join(ReviewRoundDir(workdir, planStem, round.RoundID), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read manifest.json for %s: %w", round.RoundID, err)
	}

	var manifest reviewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return 0, false, fmt.Errorf("parse manifest.json for %s: %w", round.RoundID, err)
	}
	if manifest.Step != nil {
		return *manifest.Step, true, nil
	}
	return 0, false, nil
}

func EffectiveReviewRevision(workdir, planStem string, round *ReviewRound) (int, bool, error) {
	if round == nil {
		return 0, false, nil
	}
	if round.Revision > 0 {
		return round.Revision, true, nil
	}
	if strings.TrimSpace(round.RoundID) == "" {
		return 0, false, nil
	}

	path := filepath.Join(ReviewRoundDir(workdir, planStem, round.RoundID), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read manifest.json for %s: %w", round.RoundID, err)
	}

	var manifest reviewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return 0, false, fmt.Errorf("parse manifest.json for %s: %w", round.RoundID, err)
	}
	if manifest.Revision > 0 {
		return manifest.Revision, true, nil
	}
	return 0, false, nil
}
