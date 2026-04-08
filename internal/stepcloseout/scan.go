package stepcloseout

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/catu-ai/easyharness/internal/plan"
)

type ActiveReviewContext struct {
	RoundID         string
	Trigger         string
	TargetStepIndex int
}

type Reminder struct {
	MissingIndexes  []int
	MissingTitles   []string
	UnscopedRoundID string
	Warnings        []string
}

type HistoricalReviewManifest struct {
	ReviewTitle string `json:"review_title,omitempty"`
	Step        *int   `json:"step,omitempty"`
	Revision    int    `json:"revision,omitempty"`
}

type RoundRecord struct {
	RoundID  string
	Sequence int
	Decision string
}

type UnknownHistoricalRound struct {
	RoundID  string
	Sequence int
}

type Scan struct {
	LatestByStepIndex   map[int]RoundRecord
	Warnings            []string
	LatestUnscopedRound *UnknownHistoricalRound
}

func LoadReminder(workdir, planStem string, doc *plan.Document, currentNode string, active *ActiveReviewContext) Reminder {
	candidateIndexes := completedStepIndexesBeforeCurrentPosition(doc, currentNode)
	if len(candidateIndexes) == 0 {
		return Reminder{}
	}

	scan := LoadLatestScan(workdir, planStem, doc, active)
	latestSteps := scan.LatestByStepIndex
	reminder := Reminder{
		MissingIndexes: make([]int, 0),
		MissingTitles:  make([]string, 0),
		Warnings:       scan.Warnings,
	}
	for _, index := range candidateIndexes {
		step := doc.Steps[index]
		if !step.Done {
			continue
		}
		if latest, ok := latestSteps[index]; ok {
			if latest.Decision == "pass" {
				if scan.LatestUnscopedRound != nil && isUnknownHistoricalReviewRoundLaterThanStepCloseout(*scan.LatestUnscopedRound, latest) {
					reminder.UnscopedRoundID = scan.LatestUnscopedRound.RoundID
				}
				continue
			}
			reminder.MissingIndexes = append(reminder.MissingIndexes, index)
			reminder.MissingTitles = append(reminder.MissingTitles, step.Title)
			continue
		}
		if HasNoStepReviewNeededMarker(step.Sections["Review Notes"]) {
			if scan.LatestUnscopedRound != nil {
				reminder.UnscopedRoundID = scan.LatestUnscopedRound.RoundID
			}
			continue
		}
		reminder.MissingIndexes = append(reminder.MissingIndexes, index)
		reminder.MissingTitles = append(reminder.MissingTitles, step.Title)
	}
	return reminder
}

func LoadLatestScan(workdir, planStem string, doc *plan.Document, active *ActiveReviewContext) Scan {
	reviewsDir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews")
	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return Scan{LatestByStepIndex: map[int]RoundRecord{}}
		}
		return Scan{
			LatestByStepIndex: map[int]RoundRecord{},
			Warnings:          []string{fmt.Sprintf("Unable to inspect historical step-closeout reviews: %v", err)},
		}
	}

	latestByStepIndex := map[int]RoundRecord{}
	warnings := make([]string, 0)
	var latestUnscopedRound *UnknownHistoricalRound
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roundID := entry.Name()
		sequence := historicalReviewRoundSequence(roundID)
		manifestPath := filepath.Join(reviewsDir, roundID, "manifest.json")
		aggregatePath := filepath.Join(reviewsDir, roundID, "aggregate.json")

		var manifest HistoricalReviewManifest
		manifestErr := readJSONFile(manifestPath, &manifest)
		record := RoundRecord{
			RoundID:  roundID,
			Sequence: sequence,
			Decision: "",
		}
		var aggregate struct {
			Decision string `json:"decision"`
		}
		if readJSONFile(aggregatePath, &aggregate) == nil {
			record.Decision = strings.TrimSpace(aggregate.Decision)
		}

		if manifestErr != nil {
			warnings = append(warnings, fmt.Sprintf("Unable to read historical review manifest for %s; status may miss older step-closeout evidence.", roundID))
			if active != nil && active.RoundID == roundID {
				if active.Trigger == "step_closeout" && active.TargetStepIndex >= 0 {
					existing, ok := latestByStepIndex[active.TargetStepIndex]
					if ok && !isLaterHistoricalStepCloseoutRound(record, existing) {
						continue
					}
					latestByStepIndex[active.TargetStepIndex] = record
					continue
				}
				if active.Trigger == "pre_archive" {
					continue
				}
			}
			candidate := UnknownHistoricalRound{
				RoundID:  roundID,
				Sequence: sequence,
			}
			if latestUnscopedRound == nil || isLaterUnknownHistoricalReviewRound(candidate, *latestUnscopedRound) {
				latestUnscopedRound = &candidate
			}
			continue
		}

		if manifest.Revision <= 0 {
			warnings = append(warnings, fmt.Sprintf("Historical review round %s is invalid and cannot be mapped to a tracked step; it is being ignored and you do not need to do anything.", roundID))
			candidate := UnknownHistoricalRound{
				RoundID:  roundID,
				Sequence: sequence,
			}
			if latestUnscopedRound == nil || isLaterUnknownHistoricalReviewRound(candidate, *latestUnscopedRound) {
				latestUnscopedRound = &candidate
			}
			continue
		}
		if manifest.Step == nil {
			continue
		}
		if *manifest.Step <= 0 || *manifest.Step > len(doc.Steps) {
			warnings = append(warnings, fmt.Sprintf("Historical review round %s is invalid and cannot be mapped to a tracked step; it is being ignored and you do not need to do anything.", roundID))
			candidate := UnknownHistoricalRound{
				RoundID:  roundID,
				Sequence: sequence,
			}
			if latestUnscopedRound == nil || isLaterUnknownHistoricalReviewRound(candidate, *latestUnscopedRound) {
				latestUnscopedRound = &candidate
			}
			continue
		}
		stepIndex := *manifest.Step - 1
		existing, ok := latestByStepIndex[stepIndex]
		if ok && !isLaterHistoricalStepCloseoutRound(record, existing) {
			continue
		}
		latestByStepIndex[stepIndex] = record
	}

	return Scan{
		LatestByStepIndex:   latestByStepIndex,
		Warnings:            warnings,
		LatestUnscopedRound: latestUnscopedRound,
	}
}

func HasNoStepReviewNeededMarker(reviewNotes string) bool {
	for _, line := range strings.Split(reviewNotes, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "NO_STEP_REVIEW_NEEDED:") {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(line, "NO_STEP_REVIEW_NEEDED:")) != "" {
			return true
		}
	}
	return false
}

func completedStepIndexesBeforeCurrentPosition(doc *plan.Document, currentNode string) []int {
	switch {
	case strings.HasPrefix(currentNode, "execution/finalize/"):
		indexes := make([]int, 0, len(doc.Steps))
		for index, step := range doc.Steps {
			if step.Done {
				indexes = append(indexes, index)
			}
		}
		return indexes
	case strings.HasPrefix(currentNode, "execution/step-"):
		index, ok := stepIndexFromNode(currentNode)
		if !ok || index <= 0 {
			return nil
		}
		indexes := make([]int, 0, index)
		for candidate := 0; candidate < index; candidate++ {
			if doc.Steps[candidate].Done {
				indexes = append(indexes, candidate)
			}
		}
		return indexes
	default:
		return nil
	}
}

func stepIndexFromNode(node string) (int, bool) {
	node = strings.TrimSpace(node)
	if !strings.HasPrefix(node, "execution/step-") {
		return -1, false
	}
	remainder := strings.TrimPrefix(node, "execution/step-")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 {
		return -1, false
	}
	stepNumber, err := strconv.Atoi(parts[0])
	if err != nil || stepNumber <= 0 {
		return -1, false
	}
	return stepNumber - 1, true
}

func historicalReviewRoundSequence(roundID string) int {
	parts := strings.Split(strings.TrimSpace(roundID), "-")
	if len(parts) < 3 || parts[0] != "review" {
		return 0
	}
	sequence, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return sequence
}

func isLaterHistoricalStepCloseoutRound(candidate, existing RoundRecord) bool {
	if candidate.Sequence != existing.Sequence {
		return candidate.Sequence > existing.Sequence
	}
	return candidate.RoundID > existing.RoundID
}

func isLaterUnknownHistoricalReviewRound(candidate, existing UnknownHistoricalRound) bool {
	if candidate.Sequence != existing.Sequence {
		return candidate.Sequence > existing.Sequence
	}
	return candidate.RoundID > existing.RoundID
}

func isUnknownHistoricalReviewRoundLaterThanStepCloseout(unknown UnknownHistoricalRound, known RoundRecord) bool {
	if unknown.Sequence != known.Sequence {
		return unknown.Sequence > known.Sequence
	}
	return unknown.RoundID > known.RoundID
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}
