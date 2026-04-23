import type {
  DashboardGroup,
  DashboardWorkspace,
  ErrorDetail,
  LiveFreshness,
  LiveFreshnessKind,
  PlanResult,
  ReviewAggregateFinding,
  ReviewArtifact,
  ReviewFinding,
  ReviewReviewer,
  ReviewRound,
  StatusResult,
  TimelineArtifactRef,
  TimelineEvent,
  Tone,
  ReviewResult,
  WorkspaceRouteResult,
} from "./types";

export type TimelineTab = {
  id: string;
  label: string;
  value: unknown;
  mode: "json" | "text";
};

const hiddenArtifactTabLabels = new Set([
  "plan_path",
  "local_state_path",
  "current_plan_path",
  "from_plan_path",
  "to_plan_path",
  "manifest_path",
  "ledger_path",
  "aggregate_path",
  "event_index_path",
  "reviews_dir",
  "record_path",
  "publish_record",
  "ci_record",
  "sync_record",
]);

export function metadataValue(value: string | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed || /^__HARNESS_UI_[A-Z0-9_]+__$/.test(trimmed)) {
    return "";
  }
  return trimmed;
}

export function workdirLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.workdir) || "unknown worktree";
}

export function productNameLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.productName) || "easyharness";
}

export function formatRelativeTimestamp(value: string | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed) return "unknown";
  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) return trimmed;
  const diffMs = Date.now() - parsed.getTime();
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (diffMs < minute) return "just now";
  if (diffMs < hour) return `${Math.max(1, Math.floor(diffMs / minute))}m ago`;
  if (diffMs < day) return `${Math.max(1, Math.floor(diffMs / hour))}h ago`;
  if (diffMs < 2 * day) return "yesterday";
  return `${Math.max(2, Math.floor(diffMs / day))}d ago`;
}

export function formatValue(value: unknown): string {
  if (value === null) return "null";
  if (value === undefined) return "undefined";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `[${value.map(formatValue).join(", ")}]`;
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

export function pickEntries(value: Record<string, unknown> | null | undefined): Array<[string, unknown]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value);
}

export function formatTimestamp(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

export function formatFreshnessTimestamp(value: string | null | undefined): string {
  if (!value) return "never";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  const now = new Date();
  const sameDay = parsed.toDateString() === now.toDateString();
  return parsed.toLocaleString(undefined, sameDay ? { timeStyle: "short" } : { dateStyle: "medium", timeStyle: "short" });
}

function freshnessTone(kind: LiveFreshnessKind): Tone {
  switch (kind) {
    case "live":
      return "good";
    case "stale":
      return "warning";
    case "disconnected":
      return "danger";
    default:
      return "muted";
  }
}

export function describeLiveFreshness(
  kind: LiveFreshnessKind,
  lastSuccessAt: string | null = null,
  error: string | null = null,
): LiveFreshness {
  const lastUpdated = formatFreshnessTimestamp(lastSuccessAt);
  switch (kind) {
    case "connecting":
      return {
        kind,
        label: "Connecting",
        detail: "Waiting for the first harness UI sync.",
        tone: freshnessTone(kind),
        lastSuccessAt,
      };
    case "updating":
      return {
        kind,
        label: "Updating",
        detail: lastSuccessAt ? `Refreshing now. Last synced ${lastUpdated}.` : "Refreshing harness state now.",
        tone: freshnessTone(kind),
        lastSuccessAt,
      };
    case "live":
      return {
        kind,
        label: "Live",
        detail: lastSuccessAt ? `Auto-refresh is healthy. Last synced ${lastUpdated}.` : "Auto-refresh is healthy.",
        tone: freshnessTone(kind),
        lastSuccessAt,
      };
    case "stale":
      return {
        kind,
        label: "Stale",
        detail: lastSuccessAt
          ? `${error || "The latest refresh failed."} Showing data from ${lastUpdated}.`
          : error || "The latest refresh failed.",
        tone: freshnessTone(kind),
        lastSuccessAt,
      };
    case "disconnected":
      return {
        kind,
        label: "Disconnected",
        detail: error || "Unable to reach the harness UI data endpoints.",
        tone: freshnessTone(kind),
        lastSuccessAt,
      };
    default:
      return {
        kind: "idle",
        label: "Idle",
        detail: "Live refresh is idle.",
        tone: freshnessTone("idle"),
        lastSuccessAt,
      };
  }
}

const freshnessPriority: Record<LiveFreshnessKind, number> = {
  idle: 0,
  live: 1,
  updating: 2,
  connecting: 3,
  stale: 4,
  disconnected: 5,
};

export function combineLiveFreshness(states: LiveFreshness[]): LiveFreshness {
  return states.reduce((current, candidate) =>
    freshnessPriority[candidate.kind] > freshnessPriority[current.kind] ? candidate : current,
  );
}

export function dashboardWorkspaces(groups: DashboardGroup[] | null | undefined): DashboardWorkspace[] {
  const flattened: DashboardWorkspace[] = [];
  for (const group of groups ?? []) {
    for (const workspace of group.workspaces ?? []) {
      flattened.push(workspace);
    }
  }
  return flattened.sort((left, right) => compareDashboardRecency(left.last_seen_at, right.last_seen_at) || left.workspace_path.localeCompare(right.workspace_path));
}

function compareDashboardRecency(left: string | undefined, right: string | undefined): number {
  const leftTime = left ? Date.parse(left) : Number.NaN;
  const rightTime = right ? Date.parse(right) : Number.NaN;
  if (!Number.isNaN(leftTime) && !Number.isNaN(rightTime) && leftTime !== rightTime) {
    return rightTime - leftTime;
  }
  if (!Number.isNaN(leftTime) && Number.isNaN(rightTime)) return -1;
  if (Number.isNaN(leftTime) && !Number.isNaN(rightTime)) return 1;
  return 0;
}

export function dashboardRowKey(workspace: DashboardWorkspace, index: number): string {
  return `${workspace.workspace_key}:${workspace.workspace_path}:${index}`;
}

export function dashboardStateTone(state: string): Tone {
  switch (state) {
    case "active":
      return "good";
    case "completed":
      return "good";
    case "missing":
    case "invalid":
      return "danger";
    case "idle":
      return "muted";
    default:
      return "muted";
  }
}

export function dashboardStateLabel(state: string): string {
  return humanizeLabel(state);
}

export function formatDashboardError(result: WorkspaceRouteResult | null, statusCode?: number): string {
  const fallback = result?.summary?.trim() || (statusCode ? `GET workspace failed with ${statusCode}` : "Unable to load workspace");
  return fallback;
}

export function humanizeLabel(value: string): string {
  const normalized = value.replace(/[_-]+/g, " ").trim();
  return normalized ? normalized.charAt(0).toUpperCase() + normalized.slice(1) : value;
}

export function titleizeLabel(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function timelineEventTitle(event: TimelineEvent): string {
  const command = event.command.trim();
  if (command) return command;
  const kind = event.kind.trim();
  if (kind) return humanizeLabel(kind);
  return `event ${event.sequence}`;
}

export function timelineEventSubtitle(event: TimelineEvent): string {
  const parts = [event.synthetic ? "bootstrap" : humanizeLabel(event.kind)];
  if (event.revision !== undefined) {
    parts.push(`rev ${event.revision}`);
  }
  return parts.join(" · ");
}

export function sortTimelineEvents(events: TimelineEvent[]): TimelineEvent[] {
  return [...events].sort((left, right) => {
    const leftTime = Date.parse(left.recorded_at);
    const rightTime = Date.parse(right.recorded_at);
    if (!Number.isNaN(leftTime) && !Number.isNaN(rightTime) && leftTime !== rightTime) {
      return rightTime - leftTime;
    }
    if (!Number.isNaN(leftTime) && Number.isNaN(rightTime)) return -1;
    if (Number.isNaN(leftTime) && !Number.isNaN(rightTime)) return 1;
    if (left.sequence !== right.sequence) return right.sequence - left.sequence;
    if (left.synthetic !== right.synthetic) return left.synthetic ? 1 : -1;
    return right.event_id.localeCompare(left.event_id);
  });
}

export function pickDefaultTimelineEvent(events: TimelineEvent[]): TimelineEvent | null {
  if (events.length === 0) return null;
  for (const event of events) {
    if (!event.synthetic) return event;
  }
  return events[0];
}

export function jsonStringify(value: unknown): string {
  if (value === undefined) return "";
  if (typeof value === "string") return JSON.stringify(value, null, 2);
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function firstDefinedValue(event: TimelineEvent, keys: string[]): unknown {
  for (const key of keys) {
    const next = event[key];
    if (next !== undefined && next !== null) {
      if (typeof next === "string" && next.trim() === "") continue;
      return next;
    }
  }
  return undefined;
}

export function artifactTabLabel(artifactRef: TimelineArtifactRef, index: number): string {
  const rawLabel = artifactRef.label?.trim();
  if (!rawLabel) return `Artifact ${index + 1}`;
  const normalized = rawLabel.replace(/_path$/i, "");
  return titleizeLabel(normalized);
}

export function timelineEventRecord(event: TimelineEvent): Record<string, unknown> {
  const artifactRefs = Array.isArray(event.artifact_refs)
    ? event.artifact_refs.map((artifactRef) => ({
        label: artifactRef.label,
        value: artifactRef.value,
        ...(artifactRef.path ? { path: artifactRef.path } : {}),
      }))
    : undefined;
  const value: Record<string, unknown> = {};
  Object.entries(event).forEach(([key, next]) => {
    if (next === undefined) return;
    if (key === "artifact_refs") {
      if (artifactRefs && artifactRefs.length > 0) value.artifact_refs = artifactRefs;
      return;
    }
    value[key] = next;
  });
  return value;
}

export function buildTimelineTabs(event: TimelineEvent | null): TimelineTab[] {
  if (!event) return [];

  const tabs: TimelineTab[] = [{ id: "event", label: "Event", value: timelineEventRecord(event), mode: "json" }];

  const inputValue = firstDefinedValue(event, ["input", "raw_input"]);
  if (inputValue !== undefined) {
    tabs.push({ id: "input", label: "Input", value: inputValue, mode: "json" });
  }

  const outputValue = firstDefinedValue(event, ["output", "raw_output"]);
  if (outputValue !== undefined) {
    tabs.push({ id: "output", label: "Output", value: outputValue, mode: "json" });
  }

  const artifactsValue = firstDefinedValue(event, ["artifacts", "raw_artifacts"]);
  if (artifactsValue !== undefined) {
    tabs.push({ id: "artifacts", label: "Artifacts", value: artifactsValue, mode: "json" });
  }

  const payloadValue = firstDefinedValue(event, ["payload"]);
  if (payloadValue !== undefined) {
    tabs.push({ id: "payload", label: "Payload", value: payloadValue, mode: "json" });
  }

  if (Array.isArray(event.artifact_refs)) {
    event.artifact_refs.forEach((artifactRef, index) => {
      if (hiddenArtifactTabLabels.has(artifactRef.label)) return;
      if (!artifactRef.path || artifactRef.content === undefined) return;
      tabs.push({
        id: `artifact-ref-${index}`,
        label: artifactTabLabel(artifactRef, index),
        value: artifactRef.content,
        mode: artifactRef.content_type === "text" ? "text" : "json",
      });
    });
  }

  return tabs;
}

export function timelineTabText(value: unknown, mode: "json" | "text"): string {
  if (mode === "text" && typeof value === "string") return value;
  return jsonStringify(value);
}

export function reviewRoundTitle(round: ReviewRound): string {
  const title = round.review_title?.trim();
  if (title) return title;
  if (typeof round.step === "number") return `Step ${round.step} review`;
  if (round.kind?.trim() === "full") return "Finalize review";
  return round.round_id;
}

export function reviewRoundSequenceLabel(round: ReviewRound): string {
  const match = round.round_id.trim().match(/^review-(\d+)(?:-.+)?$/i);
  if (!match) return round.round_id;
  return `Round ${match[1]}`;
}

export function reviewRoundListLabel(round: ReviewRound): string {
  const parts = [reviewRoundSequenceLabel(round)];
  if (round.kind?.trim()) parts.push(humanizeLabel(round.kind));
  if (typeof round.step === "number") {
    parts.push(`step ${round.step}`);
  } else if (round.kind?.trim() === "full") {
    parts.push("finalize");
  }
  if (typeof round.revision === "number" && round.revision > 0) parts.push(`rev ${round.revision}`);
  return parts.join(" · ");
}

export function reviewRoundCompactMeta(round: ReviewRound): string {
  const parts = [reviewRoundSequenceLabel(round)];
  if (round.kind?.trim()) parts.push(humanizeLabel(round.kind));
  if (round.kind?.trim() === "full") parts.push("finalize");
  if (typeof round.revision === "number" && round.revision > 0) parts.push(`rev ${round.revision}`);
  return parts.join(" · ");
}

export function reviewRoundCompactStatusLabel(round: ReviewRound): string {
  switch (round.status) {
    case "pass":
      return "PASS";
    case "changes_requested":
      return "CHG";
    case "waiting_for_submissions":
      return "WAIT";
    case "waiting_for_aggregation":
      return "AGGR";
    case "degraded":
      return "DEG";
    case "in_progress":
      return "WIP";
    case "complete":
      return "DONE";
    case "aggregated":
      return "AGGR";
    case "incomplete":
      return "PART";
    default:
      return reviewRoundStatusLabel(round).toUpperCase();
  }
}

export function reviewRoundSubtitle(round: ReviewRound): string {
  const parts: string[] = [];
  if (typeof round.step === "number") {
    parts.push(`Step ${round.step}`);
  } else if (round.kind?.trim() === "full") {
    parts.push("Finalize scope");
  }
  return parts.join(" · ") || round.round_id;
}

export function reviewRoundExplorerMetaLabel(round: ReviewRound): string {
  const parts: string[] = [];
  if (typeof round.step === "number") {
    parts.push(`Step ${round.step}`);
  } else if (round.kind?.trim() === "full") {
    parts.push("Finalize");
  } else {
    parts.push(reviewRoundSequenceLabel(round));
  }
  parts.push(`${reviewCountLabel(round.submitted_slots)}/${reviewCountLabel(round.total_slots)}`);
  return parts.join(" · ");
}

export function reviewCountLabel(value: number | undefined): string {
  if (typeof value !== "number") return "0";
  return String(value);
}

export function reviewRoundAriaLabel(round: ReviewRound): string {
  const parts = [
    reviewRoundTitle(round),
    reviewRoundStatusLabel(round),
    reviewRoundCompactMeta(round),
    reviewRoundSubtitle(round),
  ].filter(Boolean);
  const timestamp = round.created_at || round.updated_at || round.aggregated_at;
  if (timestamp) {
    parts.push(formatTimestamp(timestamp));
  }
  parts.push(`${reviewCountLabel(round.submitted_slots)}/${reviewCountLabel(round.total_slots)} submitted`);
  return parts.join(" ");
}

export function reviewRoundStatusLabel(round: ReviewRound): string {
  const status = round.status?.trim();
  if (!status) return "Unknown";
  switch (status) {
    case "pass":
      return "Pass";
    case "changes_requested":
      return "Changes requested";
    case "waiting_for_submissions":
      return "Waiting for submissions";
    case "waiting_for_aggregation":
      return "Waiting for aggregation";
    case "degraded":
      return "Degraded";
    case "in_progress":
      return "In progress";
    case "complete":
      return "Complete";
    case "aggregated":
      return "Aggregated";
    case "incomplete":
      return "Incomplete";
    default:
      return humanizeLabel(status);
  }
}

export function reviewRoundStatusTone(round: ReviewRound): "good" | "danger" | "warning" | "muted" {
  switch (round.status) {
    case "pass":
    case "complete":
      return "good";
    case "changes_requested":
    case "degraded":
      return "danger";
    case "waiting_for_submissions":
    case "waiting_for_aggregation":
    case "incomplete":
      return "warning";
    default:
      return "muted";
  }
}

export function reviewReviewerLabel(reviewer: ReviewReviewer): string {
  return reviewer.name?.trim() || reviewer.slot;
}

export function reviewReviewerStatusLabel(reviewer: ReviewReviewer): string {
  const status = reviewer.status?.trim();
  if (!status) return reviewer.summary?.trim() ? "Submitted" : "Pending";
  return humanizeLabel(status);
}

export function reviewReviewerStatusTone(reviewer: ReviewReviewer): "good" | "danger" | "warning" | "muted" {
  const status = reviewer.status?.trim().toLowerCase();
  const hasWarnings = Array.isArray(reviewer.warnings) && reviewer.warnings.length > 0;
  if (!status) {
    if (reviewer.summary?.trim()) return hasWarnings ? "warning" : "good";
    return "warning";
  }
  if (status === "submitted") return hasWarnings ? "warning" : "good";
  if (status === "pending") return "warning";
  return hasWarnings ? "danger" : "warning";
}

export function reviewFindingBadgeTone(severity: string): "danger" | "warning" {
  return severity === "minor" ? "warning" : "danger";
}

export function reviewFindingKey(finding: ReviewFinding, index: number): string {
  const aggregateFinding = finding as ReviewAggregateFinding;
  return [aggregateFinding.slot, aggregateFinding.dimension, finding.title, finding.details, String(index)].filter(Boolean).join("::");
}

export function reviewAggregateFindingSource(finding: ReviewAggregateFinding): string | null {
  const dimension = finding.dimension?.trim() ? humanizeLabel(finding.dimension) : "";
  const slot = finding.slot?.trim() ? humanizeLabel(finding.slot) : "";
  if (dimension && slot && dimension.toLowerCase() !== slot.toLowerCase()) {
    return `${dimension} · slot ${slot}`;
  }
  if (dimension) return dimension;
  if (slot) return `slot ${slot}`;
  return null;
}

export function reviewAggregateFindingLabels(finding: ReviewAggregateFinding): string[] {
  const labels: string[] = [];
  const dimension = finding.dimension?.trim() ? humanizeLabel(finding.dimension) : "";
  const slot = finding.slot?.trim() ? humanizeLabel(finding.slot) : "";
  if (dimension) labels.push(dimension);
  if (slot && slot.toLowerCase() !== dimension.toLowerCase()) {
    labels.push(`slot ${slot}`);
  }
  return labels;
}

export function reviewArtifactLabel(artifact: ReviewArtifact): string {
  return artifact.label?.trim() || "Artifact";
}

export function reviewArtifactKey(artifact: ReviewArtifact, index: number): string {
  const path = artifact.path?.trim();
  if (path) return path;
  return `${reviewArtifactLabel(artifact)}-${index}`;
}

export function reviewArtifactText(artifact: ReviewArtifact | null): string {
  if (!artifact) return "";
  if (artifact.content_type === "text" && typeof artifact.content === "string") return artifact.content;
  return jsonStringify(artifact.content ?? { status: artifact.status, summary: artifact.summary, path: artifact.path });
}

export function formatPlanError(result: PlanResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result.errors
        ?.map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const summary = result?.summary?.trim();
  if (summary && details.length > 0) return `${summary} ${details.join("; ")}`;
  if (summary) return summary;
  if (details.length > 0) return details.join("; ");
  if (statusCode) return `GET /api/plan failed with ${statusCode}`;
  return "Unable to load plan";
}

export function reviewRawSubmissionText(value: unknown): string {
  if (typeof value === "string") return value;
  return jsonStringify(value);
}

export function formatReviewError(result: ReviewResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result.errors
        ?.map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const summary = result?.summary?.trim();
  if (summary && details.length > 0) return `${summary} ${details.join("; ")}`;
  if (summary) return summary;
  if (details.length > 0) return details.join("; ");
  if (statusCode) return `GET /api/review failed with ${statusCode}`;
  return "Unable to load review";
}

export function formatStatusError(result: StatusResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result.errors
        ?.map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const summary = result?.summary?.trim();
  if (summary && details.length > 0) return `${summary} ${details.join("; ")}`;
  if (summary) return summary;
  if (details.length > 0) return details.join("; ");
  if (statusCode) return `GET /api/status failed with ${statusCode}`;
  return "Unable to load status";
}

export function formatTimelineError(summary: string | undefined, errors: ErrorDetail[] | null | undefined, statusCode?: number): string {
  const details = Array.isArray(errors)
    ? errors
        .map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const fallback = summary?.trim() || (statusCode ? `GET /api/timeline failed with ${statusCode}` : "Unable to load timeline");
  return details.length > 0 ? `${fallback} ${details.join("; ")}` : fallback;
}
