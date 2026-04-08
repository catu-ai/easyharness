package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/timeline"
)

func readStatusSnapshot(workdir string) *status.Result {
	result := status.Service{Workdir: workdir}.Read()
	if !result.OK {
		return nil
	}
	return &result
}

func readUnlockedStatusSnapshot(workdir string) *status.Result {
	result := status.Service{Workdir: workdir}.ReadUnlocked()
	if !result.OK {
		return nil
	}
	return &result
}

func appendTimelineEvent(workdir string, before, after *status.Result, event timeline.Event, recordedAt string) error {
	planPath := strings.TrimSpace(event.PlanPath)
	if planPath == "" {
		planPath = statusPlanPath(after)
	}
	if planPath == "" {
		planPath = statusPlanPath(before)
	}
	if planPath == "" {
		return fmt.Errorf("unable to determine timeline plan path")
	}
	event.PlanPath = normalizeRepoPath(workdir, planPath)
	event.PlanStem = strings.TrimSuffix(filepath.Base(event.PlanPath), filepath.Ext(event.PlanPath))
	if event.PlanStem == "" {
		return fmt.Errorf("unable to determine plan stem from %q", planPath)
	}
	if event.Revision == 0 {
		event.Revision = statusRevision(after)
	}
	if event.Revision == 0 {
		event.Revision = statusRevision(before)
	}
	if strings.TrimSpace(event.RecordedAt) == "" {
		event.RecordedAt = strings.TrimSpace(recordedAt)
	}
	if event.FromNode == "" && before != nil {
		event.FromNode = strings.TrimSpace(before.State.CurrentNode)
	}
	if event.ToNode == "" && after != nil {
		event.ToNode = strings.TrimSpace(after.State.CurrentNode)
	}
	for i := range event.ArtifactRefs {
		ref := &event.ArtifactRefs[i]
		ref.Value = normalizeRepoPath(workdir, ref.Value)
		if ref.Path != "" {
			ref.Path = normalizeRepoPath(workdir, ref.Path)
		}
	}
	_, _, err := timeline.AppendEvent(workdir, event.PlanStem, event)
	return err
}

func lifecycleTimelineHook(workdir string, before *status.Result, recordedAt string, input any) func(lifecycle.Result) error {
	return func(result lifecycle.Result) error {
		return appendTimelineEvent(
			workdir,
			before,
			readUnlockedStatusSnapshot(workdir),
			attachRawPayloads(timelineEventFromLifecycle(result), input, result, result.Artifacts),
			recordedAt,
		)
	}
}

func reviewStartTimelineHook(workdir string, before *status.Result, recordedAt string, input []byte) func(reviewStartResult) error {
	return func(result reviewStartResult) error {
		return appendTimelineEvent(
			workdir,
			before,
			readUnlockedStatusSnapshot(workdir),
			attachRawPayloads(timelineEventFromReviewStart(result), input, result, result.Artifacts),
			recordedAt,
		)
	}
}

func reviewSubmitTimelineHook(workdir string, before *status.Result, recordedAt string, input []byte) func(reviewSubmitResult) error {
	return func(result reviewSubmitResult) error {
		return appendTimelineEvent(
			workdir,
			before,
			readUnlockedStatusSnapshot(workdir),
			attachRawPayloads(timelineEventFromReviewSubmit(result), input, result, result.Artifacts),
			recordedAt,
		)
	}
}

func reviewAggregateTimelineHook(workdir string, before *status.Result, recordedAt string, input any) func(reviewAggregateResult) error {
	return func(result reviewAggregateResult) error {
		return appendTimelineEvent(
			workdir,
			before,
			readUnlockedStatusSnapshot(workdir),
			attachRawPayloads(timelineEventFromReviewAggregate(result), input, result, result.Artifacts),
			recordedAt,
		)
	}
}

func evidenceTimelineHook(workdir string, before *status.Result, recordedAt, kind string, input any) func(evidence.Result) error {
	return func(result evidence.Result) error {
		return appendTimelineEvent(
			workdir,
			before,
			readUnlockedStatusSnapshot(workdir),
			attachRawPayloads(timelineEventFromEvidence(result, kind), input, result, result.Artifacts),
			recordedAt,
		)
	}
}

func timelineEventFromLifecycle(result lifecycle.Result) timeline.Event {
	event := timeline.Event{
		Kind:    "lifecycle",
		Command: result.Command,
		Summary: result.Summary,
		Details: []timeline.Detail{
			{Key: "current_node", Value: result.State.CurrentNode},
		},
	}
	if result.Facts != nil {
		event.Revision = result.Facts.Revision
		if result.Facts.ReopenMode != "" {
			event.Details = append(event.Details, timeline.Detail{Key: "reopen_mode", Value: result.Facts.ReopenMode})
		}
		if result.Facts.LandPRURL != "" {
			event.Details = append(event.Details, timeline.Detail{Key: "land_pr_url", Value: result.Facts.LandPRURL})
		}
		if result.Facts.LandCommit != "" {
			event.Details = append(event.Details, timeline.Detail{Key: "land_commit", Value: result.Facts.LandCommit})
		}
	}
	if result.Artifacts != nil {
		if result.Artifacts.ToPlanPath != "" {
			event.PlanPath = result.Artifacts.ToPlanPath
		} else {
			event.PlanPath = result.Artifacts.FromPlanPath
		}
		event.ArtifactRefs = append(event.ArtifactRefs,
			pathRef("from_plan_path", result.Artifacts.FromPlanPath),
			pathRef("to_plan_path", result.Artifacts.ToPlanPath),
			pathRef("local_state_path", result.Artifacts.LocalStatePath),
			pathRef("current_plan_path", result.Artifacts.CurrentPlanPath),
		)
	}
	return pruneTimelineEvent(event)
}

func timelineEventFromReviewStart(result reviewStartResult) timeline.Event {
	event := timeline.Event{
		Kind:    "review",
		Command: result.Command,
		Summary: result.Summary,
	}
	if result.Artifacts != nil {
		event.PlanPath = result.Artifacts.PlanPath
		event.ArtifactRefs = append(event.ArtifactRefs,
			pathRef("plan_path", result.Artifacts.PlanPath),
			pathRef("local_state_path", result.Artifacts.LocalStatePath),
			valueRef("round_id", result.Artifacts.RoundID),
			pathRef("manifest_path", result.Artifacts.ManifestPath),
			pathRef("ledger_path", result.Artifacts.LedgerPath),
			pathRef("aggregate_path", result.Artifacts.AggregatePath),
		)
		event.Details = append(event.Details, timeline.Detail{
			Key:   "slot_count",
			Value: strconv.Itoa(len(result.Artifacts.Slots)),
		})
	}
	return pruneTimelineEvent(event)
}

func timelineEventFromReviewSubmit(result reviewSubmitResult) timeline.Event {
	event := timeline.Event{
		Kind:    "review",
		Command: result.Command,
		Summary: result.Summary,
	}
	if result.Artifacts != nil {
		event.ArtifactRefs = append(event.ArtifactRefs,
			valueRef("round_id", result.Artifacts.RoundID),
			valueRef("slot", result.Artifacts.Slot),
			pathRef("submission_path", result.Artifacts.SubmissionPath),
			pathRef("ledger_path", result.Artifacts.LedgerPath),
		)
		event.Details = append(event.Details, timeline.Detail{Key: "slot", Value: result.Artifacts.Slot})
	}
	return pruneTimelineEvent(event)
}

func timelineEventFromReviewAggregate(result reviewAggregateResult) timeline.Event {
	event := timeline.Event{
		Kind:    "review",
		Command: result.Command,
		Summary: result.Summary,
	}
	if result.Artifacts != nil {
		event.ArtifactRefs = append(event.ArtifactRefs,
			valueRef("round_id", result.Artifacts.RoundID),
			pathRef("aggregate_path", result.Artifacts.AggregatePath),
			pathRef("local_state_path", result.Artifacts.LocalStatePath),
		)
	}
	if result.Review != nil {
		event.Revision = result.Review.Revision
		event.Details = append(event.Details,
			timeline.Detail{Key: "decision", Value: result.Review.Decision},
			timeline.Detail{Key: "blocking_findings", Value: strconv.Itoa(len(result.Review.BlockingFindings))},
			timeline.Detail{Key: "non_blocking_findings", Value: strconv.Itoa(len(result.Review.NonBlockingFindings))},
		)
	}
	return pruneTimelineEvent(event)
}

func timelineEventFromEvidence(result evidence.Result, kind string) timeline.Event {
	event := timeline.Event{
		Kind:    "evidence",
		Command: result.Command,
		Summary: result.Summary,
		Details: []timeline.Detail{{Key: "evidence_kind", Value: strings.TrimSpace(kind)}},
	}
	if result.Artifacts != nil {
		event.PlanPath = result.Artifacts.PlanPath
		event.ArtifactRefs = append(event.ArtifactRefs,
			pathRef("plan_path", result.Artifacts.PlanPath),
			pathRef("local_state_path", result.Artifacts.LocalStatePath),
			valueRef("record_id", result.Artifacts.RecordID),
			pathRef("record_path", result.Artifacts.RecordPath),
		)
	}
	return pruneTimelineEvent(event)
}

type reviewStartResult = contracts.ReviewStartResult
type reviewSubmitResult = contracts.ReviewSubmitResult
type reviewAggregateResult = contracts.ReviewAggregateResult

func pathRef(label, path string) timeline.ArtifactRef {
	if strings.TrimSpace(path) == "" {
		return timeline.ArtifactRef{}
	}
	return timeline.ArtifactRef{Label: label, Value: path, Path: path}
}

func valueRef(label, value string) timeline.ArtifactRef {
	if strings.TrimSpace(value) == "" {
		return timeline.ArtifactRef{}
	}
	return timeline.ArtifactRef{Label: label, Value: value}
}

func pruneTimelineEvent(event timeline.Event) timeline.Event {
	filteredRefs := make([]timeline.ArtifactRef, 0, len(event.ArtifactRefs))
	for _, ref := range event.ArtifactRefs {
		if strings.TrimSpace(ref.Label) == "" || strings.TrimSpace(ref.Value) == "" {
			continue
		}
		filteredRefs = append(filteredRefs, ref)
	}
	event.ArtifactRefs = filteredRefs

	filteredDetails := make([]timeline.Detail, 0, len(event.Details))
	for _, detail := range event.Details {
		if strings.TrimSpace(detail.Key) == "" || strings.TrimSpace(detail.Value) == "" {
			continue
		}
		filteredDetails = append(filteredDetails, detail)
	}
	event.Details = filteredDetails
	return event
}

func attachRawPayloads(event timeline.Event, input, output, artifacts any) timeline.Event {
	event.Input = marshalRawPayload(input)
	event.Output = marshalRawPayload(output)
	event.Artifacts = marshalRawPayload(artifacts)
	return event
}

func marshalRawPayload(value any) json.RawMessage {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		trimmed := strings.TrimSpace(string(typed))
		if trimmed == "" || trimmed == "null" {
			return nil
		}
		return append(json.RawMessage(nil), typed...)
	case json.RawMessage:
		trimmed := strings.TrimSpace(string(typed))
		if trimmed == "" || trimmed == "null" {
			return nil
		}
		return append(json.RawMessage(nil), typed...)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		if string(data) == "null" {
			return nil
		}
		return data
	}
}

func normalizeRepoPath(workdir, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !filepath.IsAbs(trimmed) {
		return filepath.ToSlash(trimmed)
	}
	if rel, err := filepath.Rel(workdir, trimmed); err == nil && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(trimmed)
}

func statusPlanPath(result *status.Result) string {
	if result == nil || result.Artifacts == nil {
		return ""
	}
	if strings.TrimSpace(result.Artifacts.PlanPath) != "" {
		return result.Artifacts.PlanPath
	}
	if strings.TrimSpace(result.Artifacts.LastLandedPlanPath) != "" {
		return result.Artifacts.LastLandedPlanPath
	}
	return ""
}

func statusRevision(result *status.Result) int {
	if result == nil || result.Facts == nil {
		return 0
	}
	return result.Facts.Revision
}
