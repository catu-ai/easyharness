package timeline

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

type Service struct {
	Workdir string
}

type Result = contracts.TimelineResult
type Event = contracts.TimelineEvent
type ArtifactRef = contracts.TimelineArtifactRef
type Detail = contracts.TimelineDetail
type Artifacts = contracts.TimelineArtifacts
type TimelineError = contracts.ErrorDetail

func (s Service) Read() Result {
	currentPlan, err := runstate.LoadCurrentPlan(s.Workdir)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "timeline",
			Summary:  "Unable to read current worktree state.",
			Errors:   []TimelineError{{Path: "state", Message: err.Error()}},
			Events:   []Event{},
		}
	}

	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			if currentPlan != nil && strings.TrimSpace(currentPlan.LastLandedPlanPath) != "" {
				return s.readPlanTimeline(filepath.ToSlash(strings.TrimSpace(currentPlan.LastLandedPlanPath)))
			}
			return Result{
				OK:       true,
				Resource: "timeline",
				Summary:  "No current or recent plan timeline is available in this worktree.",
				Events:   []Event{},
			}
		}
		return Result{
			OK:       false,
			Resource: "timeline",
			Summary:  "Unable to determine the current plan for timeline loading.",
			Errors:   []TimelineError{{Path: "plan", Message: err.Error()}},
			Events:   []Event{},
		}
	}

	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "timeline",
			Summary:  "Unable to determine the current plan path for timeline loading.",
			Errors:   []TimelineError{{Path: "plan", Message: err.Error()}},
			Events:   []Event{},
		}
	}
	return s.readPlanTimeline(filepath.ToSlash(relPlanPath))
}

func (s Service) readPlanTimeline(relPlanPath string) Result {
	planStem := strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
	eventIndexPath := EventIndexPath(s.Workdir, planStem)
	events, err := loadEvents(eventIndexPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "timeline",
			Summary:  "Unable to read the timeline event index.",
			Artifacts: &Artifacts{
				PlanPath:       relPlanPath,
				EventIndexPath: eventIndexPath,
			},
			Errors: []TimelineError{{Path: "events", Message: err.Error()}},
			Events: []Event{},
		}
	}

	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	errors := make([]TimelineError, 0, 1)
	if err != nil {
		errors = append(errors, TimelineError{Path: "state", Message: err.Error()})
	}
	doc, err := plan.LoadFile(resolvePlanPath(s.Workdir, relPlanPath))
	if err != nil {
		errors = append(errors, TimelineError{Path: "plan", Message: err.Error()})
	}
	bootstrapEvents := buildBootstrapEvents(relPlanPath, planStem, doc, state, events)
	mergedEvents := mergeTimelineEvents(bootstrapEvents, events)
	mergedEvents = enrichArtifactRefs(s.Workdir, mergedEvents)

	summary := "No timeline events recorded yet for the current plan."
	if len(mergedEvents) > 0 {
		summary = fmt.Sprintf("Loaded %d timeline event(s) for %s.", len(mergedEvents), filepath.Base(relPlanPath))
	}

	return Result{
		OK:       true,
		Resource: "timeline",
		Summary:  summary,
		Artifacts: &Artifacts{
			PlanPath:       relPlanPath,
			LocalStatePath: statePath,
			EventIndexPath: eventIndexPath,
		},
		Events: mergedEvents,
		Errors: errors,
	}
}

func enrichArtifactRefs(workdir string, events []Event) []Event {
	cache := map[string]ArtifactRef{}
	enriched := make([]Event, 0, len(events))
	for _, event := range events {
		if len(event.ArtifactRefs) == 0 {
			enriched = append(enriched, event)
			continue
		}
		next := event
		next.ArtifactRefs = make([]ArtifactRef, 0, len(event.ArtifactRefs))
		for _, ref := range event.ArtifactRefs {
			enrichedRef := ref
			if strings.TrimSpace(ref.Path) != "" {
				normalizedPath := filepath.ToSlash(strings.TrimSpace(ref.Path))
				if cached, ok := cache[normalizedPath]; ok {
					enrichedRef.ContentType = cached.ContentType
					enrichedRef.Content = append(json.RawMessage(nil), cached.Content...)
				} else if content, contentType, err := readArtifactRefContent(workdir, normalizedPath); err == nil {
					enrichedRef.ContentType = contentType
					enrichedRef.Content = content
					cache[normalizedPath] = ArtifactRef{
						ContentType: contentType,
						Content:     append(json.RawMessage(nil), content...),
					}
				}
			}
			next.ArtifactRefs = append(next.ArtifactRefs, enrichedRef)
		}
		enriched = append(enriched, next)
	}
	return enriched
}

func readArtifactRefContent(workdir, refPath string) (json.RawMessage, string, error) {
	resolvedPath := refPath
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(workdir, filepath.FromSlash(refPath))
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, "", err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, "", nil
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage([]byte(trimmed)), "json", nil
	}
	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return nil, "", err
	}
	return encoded, "text", nil
}

func AppendEvent(workdir, planStem string, event Event) (string, Event, error) {
	release, err := runstate.AcquireTimelineMutationLock(workdir, planStem)
	if err != nil {
		return "", Event{}, err
	}
	defer release()

	path := EventIndexPath(workdir, planStem)
	events, err := loadEvents(path)
	if err != nil {
		return path, Event{}, err
	}
	nextSequence := len(events) + 1
	event.Sequence = nextSequence
	event.EventID = fmt.Sprintf("event-%03d", nextSequence)
	if strings.TrimSpace(event.PlanStem) == "" {
		event.PlanStem = planStem
	}
	events = append(events, event)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, Event{}, err
	}
	if err := writeEventsAtomically(path, events); err != nil {
		return path, Event{}, err
	}
	return path, event, nil
}

func EventIndexPath(workdir, planStem string) string {
	return filepath.Join(workdir, ".local", "harness", "plans", planStem, "events.jsonl")
}

func loadEvents(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}
	defer file.Close()

	events := make([]Event, 0)
	reader := bufio.NewReader(file)
	lineNumber := 0
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, readErr
		}
		if errors.Is(readErr, io.EOF) && line == "" {
			break
		}
		lineNumber++
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			var event Event
			if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
				return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
			}
			if event.Sequence == 0 {
				event.Sequence = lineNumber
			}
			if strings.TrimSpace(event.EventID) == "" {
				event.EventID = "event-" + strconv.Itoa(lineNumber)
			}
			events = append(events, event)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	return events, nil
}

func writeEventsAtomically(path string, events []Event) (err error) {
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

	if err := tempFile.Chmod(0o644); err != nil {
		return err
	}
	for _, event := range events {
		data, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err := tempFile.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
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

func buildBootstrapEvents(relPlanPath, planStem string, doc *plan.Document, state *runstate.State, events []Event) []Event {
	bootstrap := make([]Event, 0, 2)
	if doc != nil && strings.TrimSpace(doc.Frontmatter.CreatedAt) != "" {
		bootstrap = append(bootstrap, Event{
			EventID:    "bootstrap-plan",
			Sequence:   0,
			RecordedAt: strings.TrimSpace(doc.Frontmatter.CreatedAt),
			Kind:       "plan",
			Command:    "plan",
			Summary:    doc.Title,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Synthetic:  true,
			Output: mustMarshalRaw(map[string]any{
				"title":            doc.Title,
				"created_at":       doc.Frontmatter.CreatedAt,
				"plan_path":        relPlanPath,
				"workflow_profile": doc.WorkflowProfile(),
			}),
		})
	}
	if hasCommand(events, "execute start") || state == nil || strings.TrimSpace(state.ExecutionStartedAt) == "" {
		return bootstrap
	}
	bootstrap = append(bootstrap, Event{
		EventID:    "bootstrap-implement",
		Sequence:   0,
		RecordedAt: strings.TrimSpace(state.ExecutionStartedAt),
		Kind:       "execution",
		Command:    "implement",
		Summary:    "Execution started for the current plan.",
		PlanPath:   relPlanPath,
		PlanStem:   planStem,
		Revision:   runstate.CurrentRevision(state),
		Synthetic:  true,
		Output: mustMarshalRaw(map[string]any{
			"execution_started_at": state.ExecutionStartedAt,
			"plan_path":            relPlanPath,
			"plan_stem":            planStem,
			"revision":             runstate.CurrentRevision(state),
			"state":                state,
		}),
	})
	return bootstrap
}

func mergeTimelineEvents(bootstrap, recorded []Event) []Event {
	merged := make([]Event, 0, len(bootstrap)+len(recorded))
	merged = append(merged, bootstrap...)
	merged = append(merged, recorded...)

	sort.SliceStable(merged, func(i, j int) bool {
		left := merged[i]
		right := merged[j]
		leftPriority := bootstrapPriority(left)
		rightPriority := bootstrapPriority(right)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}

		leftTime, leftOK := parseTimelineTime(left.RecordedAt)
		rightTime, rightOK := parseTimelineTime(right.RecordedAt)
		if leftOK && rightOK && !leftTime.Equal(rightTime) {
			return leftTime.Before(rightTime)
		}
		if leftOK != rightOK {
			return leftOK
		}
		if left.Synthetic != right.Synthetic {
			return left.Synthetic
		}
		if left.Sequence != right.Sequence {
			if left.Sequence == 0 {
				return true
			}
			if right.Sequence == 0 {
				return false
			}
			return left.Sequence < right.Sequence
		}
		return left.EventID < right.EventID
	})

	return merged
}

func bootstrapPriority(event Event) int {
	if event.Synthetic && strings.EqualFold(strings.TrimSpace(event.Command), "plan") {
		return 0
	}
	if event.Synthetic && strings.EqualFold(strings.TrimSpace(event.Command), "implement") {
		return 1
	}
	return 2
}

func hasCommand(events []Event, command string) bool {
	for _, event := range events {
		if strings.TrimSpace(event.Command) == strings.TrimSpace(command) {
			return true
		}
	}
	return false
}

func resolvePlanPath(workdir, relPlanPath string) string {
	if filepath.IsAbs(relPlanPath) {
		return relPlanPath
	}
	return filepath.Join(workdir, filepath.FromSlash(relPlanPath))
}

func parseTimelineTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func mustMarshalRaw(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}
