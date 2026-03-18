package runstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CurrentPlan struct {
	PlanPath string `json:"plan_path"`
}

type State struct {
	PlanPath          string       `json:"plan_path,omitempty"`
	PlanStem          string       `json:"plan_stem,omitempty"`
	ActiveReviewRound *ReviewRound `json:"active_review_round,omitempty"`
	LatestCI          *CIState     `json:"latest_ci,omitempty"`
	Sync              *SyncState   `json:"sync,omitempty"`
	LatestPublish     *Publish     `json:"latest_publish,omitempty"`
}

type ReviewRound struct {
	RoundID    string `json:"round_id"`
	Kind       string `json:"kind"`
	Aggregated bool   `json:"aggregated"`
	Decision   string `json:"decision,omitempty"`
}

type CIState struct {
	SnapshotID string `json:"snapshot_id"`
	Status     string `json:"status"`
}

type SyncState struct {
	Freshness string `json:"freshness"`
	Conflicts bool   `json:"conflicts"`
}

type Publish struct {
	AttemptID string `json:"attempt_id"`
	PRURL     string `json:"pr_url"`
}

type reviewAggregate struct {
	Decision string `json:"decision"`
}

func LoadCurrentPlan(workdir string) (*CurrentPlan, error) {
	path := filepath.Join(workdir, ".local", "harness", "current-plan.json")
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
	path := filepath.Join(workdir, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(CurrentPlan{PlanPath: planPath}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal current-plan.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func LoadState(workdir, planStem string) (*State, string, error) {
	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "state.json")
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
	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal state.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
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

	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", round.RoundID, "aggregate.json")
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
