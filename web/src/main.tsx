import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import "./styles.css";

type Page = "status" | "timeline" | "review" | "diff" | "files";
type PageDef = { id: Page; label: string; href: string };
type SectionLink = { id: string; label: string; meta?: string };

type NextAction = {
  command: string | null;
  description: string;
};

type ErrorDetail = {
  path: string;
  message: string;
};

type StatusResult = {
  ok: boolean;
  command: string;
  summary: string;
  state?: {
    current_node?: string;
  };
  facts?: Record<string, unknown> | null;
  artifacts?: Record<string, unknown> | null;
  next_actions?: NextAction[] | null;
  blockers?: ErrorDetail[] | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

type TimelineDetail = {
  key: string;
  value: string;
};

type TimelineArtifactRef = {
  label: string;
  value: string;
  path?: string;
  content_type?: string;
  content?: unknown;
};

type TimelineEvent = {
  event_id: string;
  sequence: number;
  recorded_at: string;
  kind: string;
  command: string;
  summary: string;
  synthetic?: boolean;
  plan_path?: string;
  plan_stem: string;
  revision?: number;
  from_node?: string;
  to_node?: string;
  details?: TimelineDetail[] | null;
  artifact_refs?: TimelineArtifactRef[] | null;
  input?: unknown;
  output?: unknown;
  artifacts?: unknown;
  payload?: unknown;
  raw_input?: unknown;
  raw_output?: unknown;
  raw_artifacts?: unknown;
  [key: string]: unknown;
};

type TimelineResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
    local_state_path?: string;
    event_index_path?: string;
  } | null;
  events?: TimelineEvent[] | null;
  errors?: ErrorDetail[] | null;
};

declare global {
  interface Window {
    __HARNESS_UI__?: {
      workdir?: string;
      repoName?: string;
      productName?: string;
    };
  }
}

const pages: PageDef[] = [
  { id: "status", label: "Status", href: "/status" },
  { id: "timeline", label: "Timeline", href: "/timeline" },
  { id: "review", label: "Review", href: "/review" },
  { id: "diff", label: "Diff", href: "/diff" },
  { id: "files", label: "Files", href: "/files" },
];

function isPage(value: string | null): value is Page {
  return value === "status" || value === "timeline" || value === "review" || value === "diff" || value === "files";
}

function pageFromPathname(pathname: string): Page | null {
  const trimmed = pathname.replace(/\/+$/, "");
  const value = trimmed.split("/").filter(Boolean).pop() ?? "";
  return isPage(value) ? value : null;
}

function readPageFromLocation(): Page {
  const pathnamePage = pageFromPathname(window.location.pathname);
  if (pathnamePage) return pathnamePage;
  const hashValue = window.location.hash.replace(/^#/, "");
  return isPage(hashValue) ? hashValue : "status";
}

function pageDefinition(page: Page): PageDef {
  return pages.find((item) => item.id === page) ?? pages[0];
}

function RailIcon(props: { page: Page }) {
  switch (props.page) {
    case "status":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3 3.5h10v3H3zM3 8.5h10v4H3z" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "timeline":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3 4.5h3v3H3zM10 9.5h3v3h-3z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M6 6h2.5v5H10" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "review":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M4 3.5h8v9H4z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M6 6.5h4M6 9.5h3" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "diff":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M5 3v10M11 3v10M3.5 5.5H6.5M9.5 10.5h3" fill="none" stroke="currentColor" stroke-width="1.2" />
          <circle cx="5" cy="5.5" r="1.4" fill="currentColor" />
          <circle cx="11" cy="10.5" r="1.4" fill="currentColor" />
        </svg>
      );
    case "files":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3.5 4.5h3l1 1h5v6.5h-9z" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
  }
}

function metadataValue(value: string | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed || /^__HARNESS_UI_[A-Z0-9_]+__$/.test(trimmed)) {
    return "";
  }
  return trimmed;
}

function workdirLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.workdir) || "unknown worktree";
}

function productNameLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.productName) || "easyharness";
}

function formatValue(value: unknown): string {
  if (value === null) return "null";
  if (value === undefined) return "undefined";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `[${value.map(formatValue).join(", ")}]`;
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

function pickEntries(value: Record<string, unknown> | null | undefined): Array<[string, unknown]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value);
}

function formatTimestamp(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function humanizeLabel(value: string): string {
  const normalized = value.replace(/[_-]+/g, " ").trim();
  return normalized ? normalized.charAt(0).toUpperCase() + normalized.slice(1) : value;
}

function titleizeLabel(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function timelineEventTitle(event: TimelineEvent): string {
  const command = event.command.trim();
  if (command) return command;
  const kind = event.kind.trim();
  if (kind) return humanizeLabel(kind);
  return `event ${event.sequence}`;
}

function timelineEventSubtitle(event: TimelineEvent): string {
  const parts = [event.synthetic ? "bootstrap" : humanizeLabel(event.kind)];
  if (event.revision !== undefined) {
    parts.push(`rev ${event.revision}`);
  }
  return parts.join(" · ");
}

function sortTimelineEvents(events: TimelineEvent[]): TimelineEvent[] {
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

function pickDefaultTimelineEvent(events: TimelineEvent[]): TimelineEvent | null {
  if (events.length === 0) return null;
  for (const event of events) {
    if (!event.synthetic) return event;
  }
  return events[0];
}

function jsonStringify(value: unknown): string {
  if (value === undefined) return "";
  if (typeof value === "string") return JSON.stringify(value, null, 2);
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function firstDefinedValue(event: TimelineEvent, keys: string[]): unknown {
  for (const key of keys) {
    const next = event[key];
    if (next !== undefined && next !== null) {
      if (typeof next === "string" && next.trim() === "") continue;
      return next;
    }
  }
  return undefined;
}

type TimelineTab = {
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
]);

function artifactTabLabel(artifactRef: TimelineArtifactRef, index: number): string {
  const rawLabel = artifactRef.label?.trim();
  if (!rawLabel) return `Artifact ${index + 1}`;
  const normalized = rawLabel.replace(/_path$/i, "");
  return titleizeLabel(normalized);
}

function timelineEventRecord(event: TimelineEvent): Record<string, unknown> {
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

function buildTimelineTabs(event: TimelineEvent | null): TimelineTab[] {
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

function timelineTabText(value: unknown, mode: "json" | "text"): string {
  if (mode === "text" && typeof value === "string") return value;
  return jsonStringify(value);
}

function formatStatusError(result: StatusResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result?.errors
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

function sectionIDsForPage(page: Page): string[] {
  if (page === "status") {
    return ["summary", "next-actions", "warnings", "facts", "artifacts"];
  }
  if (page === "timeline") {
    return ["events"];
  }
  return ["overview", "status"];
}

function readSectionFromLocation(page: Page): string {
  const section = window.location.hash.replace(/^#/, "");
  return sectionIDsForPage(page).includes(section) ? section : sectionIDsForPage(page)[0];
}

function sectionsForPage(page: Page, status: {
  nextActions: NextAction[];
  blockers: ErrorDetail[];
  warnings: string[];
  facts: Array<[string, unknown]>;
  artifacts: Array<[string, unknown]>;
}, timeline: {
  events: TimelineEvent[];
  artifacts: Array<[string, unknown]>;
}): SectionLink[] {
  if (page === "status") {
    return [
      { id: "summary", label: "Summary" },
      { id: "next-actions", label: "Next actions", meta: String(status.nextActions.length) },
      { id: "warnings", label: "Warnings", meta: String(status.warnings.length + status.blockers.length) },
      { id: "facts", label: "Facts", meta: String(status.facts.length) },
      { id: "artifacts", label: "Artifacts", meta: String(status.artifacts.length) },
    ];
  }

  if (page === "timeline") {
    return [
      { id: "events", label: "Events", meta: String(timeline.events.length) },
    ];
  }

  return [
    { id: "overview", label: "Overview" },
    { id: "status", label: "Status" },
  ];
}

function App() {
  const [page, setPage] = useState<Page>(() => readPageFromLocation());
  const [section, setSection] = useState<string>(() => readSectionFromLocation(readPageFromLocation()));
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [timeline, setTimeline] = useState<TimelineResult | null>(null);
  const [timelineError, setTimelineError] = useState<string | null>(null);
  const [timelineLoading, setTimelineLoading] = useState(false);

  useEffect(() => {
    const onLocationChange = () => {
      const nextPage = readPageFromLocation();
      setPage(nextPage);
      setSection(readSectionFromLocation(nextPage));
    };
    window.addEventListener("popstate", onLocationChange);
    window.addEventListener("hashchange", onLocationChange);
    return () => {
      window.removeEventListener("popstate", onLocationChange);
      window.removeEventListener("hashchange", onLocationChange);
    };
  }, []);

  const navigateToPage = (nextPage: Page) => {
    const next = pageDefinition(nextPage);
    const nextSection = sectionIDsForPage(nextPage)[0];
    const nextURL = `${next.href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setPage(nextPage);
    setSection(nextSection);
  };

  const navigateToSection = (nextSection: string) => {
    const nextURL = `${pageDefinition(page).href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setSection(nextSection);
  };

  useEffect(() => {
    if (pageFromPathname(window.location.pathname) === null && !window.location.hash) {
      window.history.replaceState({}, "", `${pageDefinition(page).href}#${sectionIDsForPage(page)[0]}`);
    }
  }, [page]);

  useEffect(() => {
    if (page !== "status") return;

    const controller = new AbortController();
    setStatusLoading(true);
    setStatusError(null);

    fetch("/api/status", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as StatusResult;
        if (!response.ok || payload.ok === false) {
          throw new Error(formatStatusError(payload, response.status));
        }
        return payload;
      })
      .then((nextStatus) => {
        setStatus(nextStatus);
        setStatusLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setStatus(null);
        setStatusError(error instanceof Error ? error.message : "Unable to load status");
        setStatusLoading(false);
      });

    return () => controller.abort();
  }, [page]);

  useEffect(() => {
    if (page !== "timeline") return;

    const controller = new AbortController();
    setTimelineLoading(true);
    setTimelineError(null);

    fetch("/api/timeline", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as TimelineResult;
        if (!response.ok || payload.ok === false) {
          const summary = payload?.summary?.trim();
          const details = Array.isArray(payload?.errors)
            ? payload.errors
                ?.map((item) => {
                  const path = item.path?.trim();
                  const message = item.message?.trim();
                  if (path && message) return `${path}: ${message}`;
                  return message || path || "";
                })
                .filter(Boolean)
            : [];
          const fallback = summary || (response.status ? `GET /api/timeline failed with ${response.status}` : "Unable to load timeline");
          throw new Error(details.length > 0 ? `${fallback} ${details.join("; ")}` : fallback);
        }
        return payload;
      })
      .then((nextTimeline) => {
        setTimeline(nextTimeline);
        setTimelineLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setTimeline(null);
        setTimelineError(error instanceof Error ? error.message : "Unable to load timeline");
        setTimelineLoading(false);
      });

    return () => controller.abort();
  }, [page]);

  const activeStatus = useMemo(() => {
    return {
      summary: status?.summary ?? "Waiting for status data.",
      currentNode: status?.state?.current_node ?? "unknown",
      nextActions: Array.isArray(status?.next_actions) ? status?.next_actions ?? [] : [],
      blockers: Array.isArray(status?.blockers) ? status?.blockers ?? [] : [],
      warnings: Array.isArray(status?.warnings) ? status?.warnings ?? [] : [],
      errors: Array.isArray(status?.errors) ? status?.errors ?? [] : [],
      facts: pickEntries(status?.facts),
      artifacts: pickEntries(status?.artifacts),
    };
  }, [status]);

  const activeTimeline = useMemo(() => {
    const events = sortTimelineEvents(Array.isArray(timeline?.events) ? timeline?.events ?? [] : []);
    const artifacts = pickEntries((timeline?.artifacts as Record<string, unknown>) ?? null);
    return {
      events,
      artifacts,
      latestEvent: events.length > 0 ? events[0] : null,
    };
  }, [timeline]);

  const activeSectionLabel =
    sectionsForPage(page, activeStatus, activeTimeline).find((item) => item.id === section)?.label ?? pageDefinition(page).label;

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{productNameLabel()}</span>
        </div>
        <div class="workspace-path" title={workdirLabel()}>{workdirLabel()}</div>
        <div class="topbar-meta">
          <span>read-only</span>
          <span>local</span>
        </div>
      </header>

      <div class="layout">
        <aside class="rail" aria-label="Pages">
          {pages.map((item) => {
            const selected = page === item.id;
            return (
              <a
                key={item.id}
                class={`rail-item${selected ? " is-active" : ""}`}
                href={item.href}
                aria-current={selected ? "page" : undefined}
                aria-label={item.label}
                title={item.label}
                onClick={(event) => {
                  event.preventDefault();
                  navigateToPage(item.id);
                }}
              >
                <span class="rail-icon">
                  <RailIcon page={item.id} />
                </span>
                <span class="sr-only">{item.label}</span>
              </a>
            );
          })}
        </aside>

        {page === "timeline" ? (
          <TimelineWorkspace
            loading={timelineLoading}
            error={timelineError}
            events={activeTimeline.events}
          />
        ) : (
          <main class="content">
            <aside class="sidebar" aria-label={`${pageDefinition(page).label} sidebar`}>
              <div class="sidebar-header">
                <span class="sidebar-label">Explorer</span>
                <strong>{pageDefinition(page).label}</strong>
              </div>
              <nav class="sidebar-group" aria-label={`${pageDefinition(page).label} sections`}>
                {sectionsForPage(page, activeStatus, activeTimeline).map((item) => (
                  <a
                    key={item.id}
                    class={`sidebar-link${item.id === section ? " is-active" : ""}`}
                    href={`#${item.id}`}
                    onClick={(event) => {
                      event.preventDefault();
                      navigateToSection(item.id);
                    }}
                  >
                    <span>{item.label}</span>
                    {item.meta ? <span class="sidebar-meta">{item.meta}</span> : null}
                  </a>
                ))}
              </nav>
            </aside>

            <section class="editor">
              <section class="page-header">
                <div class="editor-tabs">
                  <div class="editor-tab is-active">{pageDefinition(page).label}</div>
                  <div class="editor-section-label">{activeSectionLabel}</div>
                </div>
                {page === "status" && statusLoading ? <span class="muted">loading</span> : null}
              </section>

              {page === "status" ? (
                <StatusPage
                  loading={statusLoading}
                  error={statusError}
                  summary={activeStatus.summary}
                  currentNode={activeStatus.currentNode}
                  nextActions={activeStatus.nextActions}
                  blockers={activeStatus.blockers}
                  warnings={activeStatus.warnings}
                  errors={Array.isArray(status?.errors) ? status?.errors ?? [] : []}
                  facts={activeStatus.facts}
                  artifacts={activeStatus.artifacts}
                  selectedSection={section}
                />
              ) : (
                <PlaceholderPage title={pageDefinition(page).label} selectedSection={section} />
              )}
            </section>
          </main>
        )}
      </div>
    </div>
  );
}

function StatusPage(props: {
  loading: boolean;
  error: string | null;
  summary: string;
  currentNode: string;
  nextActions: NextAction[];
  blockers: ErrorDetail[];
  warnings: string[];
  errors: ErrorDetail[];
  facts: Array<[string, unknown]>;
  artifacts: Array<[string, unknown]>;
  selectedSection: string;
}) {
  const { loading, error, summary, currentNode, nextActions, blockers, warnings, errors, facts, artifacts, selectedSection } = props;

  let detailPane = (
    <section id="summary" class="pane">
      <div class="section-head">
        <h2>Summary</h2>
        {loading ? <span class="muted">loading</span> : null}
      </div>
      <div class="detail-copy">{summary}</div>
    </section>
  );

  if (selectedSection === "next-actions") {
    detailPane = (
      <section id="next-actions" class="pane">
        <div class="section-head">
          <h2>Next actions</h2>
          <span class="muted">{nextActions.length}</span>
        </div>
        <ol class="stack-list">
          {nextActions.length > 0 ? (
            nextActions.map((action, index) => (
              <li key={`${action.description}-${index}`}>
                <div class="list-title">{action.description}</div>
                {action.command ? <code>{action.command}</code> : <span class="muted">no command</span>}
              </li>
            ))
          ) : (
            <li class="empty-row">No next actions surfaced yet.</li>
          )}
        </ol>
      </section>
    );
  }

  if (selectedSection === "warnings") {
    detailPane = (
      <section id="warnings" class="pane">
        <div class="section-head">
          <h2>Warnings & blockers</h2>
        </div>
        <div class="stack-list">
          {warnings.length > 0 ? warnings.map((warning, index) => <div key={`warning-${index}`} class="pill pill-warn">{warning}</div>) : <div class="empty-row">No warnings.</div>}
          {blockers.length > 0 ? (
            blockers.map((blocker, index) => (
              <div key={`${blocker.path}-${index}`} class="pill pill-blocker">
                <strong>{blocker.path}</strong>
                <span>{blocker.message}</span>
              </div>
            ))
          ) : (
            <div class="empty-row">No blockers.</div>
          )}
          {errors.length > 0 ? (
            errors.map((item, index) => (
              <div key={`${item.path}-${index}`} class="pill pill-blocker">
                <strong>{item.path}</strong>
                <span>{item.message}</span>
              </div>
            ))
          ) : null}
        </div>
      </section>
    );
  }

  if (selectedSection === "facts") {
    detailPane = (
      <section id="facts" class="pane">
        <div class="section-head">
          <h2>Facts</h2>
          <span class="muted">{facts.length}</span>
        </div>
        <dl class="kv-list">
          {facts.length > 0 ? (
            facts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))
          ) : (
            <div class="empty-row">No facts available.</div>
          )}
        </dl>
      </section>
    );
  }

  if (selectedSection === "artifacts") {
    detailPane = (
      <section id="artifacts" class="pane">
        <div class="section-head">
          <h2>Artifacts</h2>
          <span class="muted">{artifacts.length}</span>
        </div>
        <dl class="kv-list">
          {artifacts.length > 0 ? (
            artifacts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))
          ) : (
            <div class="empty-row">No artifacts available.</div>
          )}
        </dl>
      </section>
    );
  }

  return (
    <section class="workspace">
      <div class="workspace-inner">
        <section class="status-grid" aria-label="Status overview">
          <div class="status-block">
            <span class="label">current node</span>
            <strong>{currentNode}</strong>
          </div>
          <div class="status-block">
            <span class="label">next actions</span>
            <strong>{nextActions.length}</strong>
          </div>
          <div class="status-block">
            <span class="label">warnings</span>
            <strong>{warnings.length}</strong>
          </div>
          <div class="status-block">
            <span class="label">blockers</span>
            <strong>{blockers.length}</strong>
          </div>
        </section>

        {error ? <div class="notice notice-error">{error}</div> : null}

        {detailPane}
      </div>
    </section>
  );
}

function TimelineWorkspace(props: {
  loading: boolean;
  error: string | null;
  events: TimelineEvent[];
}) {
  const { loading, error, events } = props;
  const sortedEvents = useMemo(() => sortTimelineEvents(events), [events]);
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const selectedEvent = useMemo(() => {
    if (sortedEvents.length === 0) return null;
    if (selectedEventId) {
      const found = sortedEvents.find((event) => event.event_id === selectedEventId);
      if (found) return found;
    }
    return pickDefaultTimelineEvent(sortedEvents);
  }, [selectedEventId, sortedEvents]);
  const [selectedTab, setSelectedTab] = useState<string>("event");
  const timelineTabs = useMemo(() => buildTimelineTabs(selectedEvent), [selectedEvent]);

  useEffect(() => {
    if (sortedEvents.length === 0) {
      setSelectedEventId(null);
      return;
    }
    setSelectedEventId((current) => {
      if (current && sortedEvents.some((event) => event.event_id === current)) {
        return current;
      }
      return pickDefaultTimelineEvent(sortedEvents)?.event_id ?? null;
    });
  }, [sortedEvents]);

  useEffect(() => {
    if (timelineTabs.length === 0) {
      setSelectedTab("event");
      return;
    }
    setSelectedTab((current) => {
      if (timelineTabs.some((tab) => tab.id === current)) {
        return current;
      }
      return timelineTabs[0].id;
    });
  }, [timelineTabs, selectedEvent?.event_id]);

  const selectedTabValue =
    timelineTabs.find((tab) => tab.id === selectedTab)?.value ?? timelineTabs[0]?.value ?? selectedEvent ?? null;
  const selectedTabMode = timelineTabs.find((tab) => tab.id === selectedTab)?.mode ?? timelineTabs[0]?.mode ?? "json";
  const transitionLabel =
    selectedEvent && (selectedEvent.from_node || selectedEvent.to_node)
      ? `${selectedEvent.from_node || "unknown"} → ${selectedEvent.to_node || "unknown"}`
      : null;

  return (
    <section class="timeline-shell">
      <aside class="timeline-nav" aria-label="Timeline events">
        <div class="timeline-nav-header">
          <span class="sidebar-label">Explorer</span>
          <strong>Timeline</strong>
          <span class="timeline-nav-meta">{sortedEvents.length}</span>
        </div>
        <div class="timeline-nav-list">
          {sortedEvents.length > 0 ? (
            sortedEvents.map((event) => {
              const selected = event.event_id === selectedEvent?.event_id;
              return (
                <button
                  key={event.event_id}
                  class={`timeline-stream-item${selected ? " is-active" : ""}`}
                  type="button"
                  onClick={() => setSelectedEventId(event.event_id)}
                  aria-pressed={selected}
                >
                  <div class="timeline-stream-row">
                    <div class="timeline-stream-title">{timelineEventTitle(event)}</div>
                    <div class="timeline-stream-meta">
                      <span>{formatTimestamp(event.recorded_at)}</span>
                    </div>
                  </div>
                  <div class="timeline-stream-subtitle">{timelineEventSubtitle(event)}</div>
                </button>
              );
            })
          ) : (
            <div class="empty-row">No timeline events recorded yet for this plan.</div>
          )}
        </div>
      </aside>

      <section class="editor timeline-editor">
        <section class="page-header">
          <div class="editor-tabs">
            <div class="editor-tab is-active">Timeline</div>
            <div class="editor-section-label">{selectedEvent ? timelineEventTitle(selectedEvent) : "Events"}</div>
          </div>
          {loading ? <span class="muted">loading</span> : null}
        </section>

        <section class="workspace workspace-timeline">
          {error ? <div class="notice notice-error">{error}</div> : null}

          <section class="timeline-inspector" aria-label="Selected event details">
            <div class="timeline-inspector-tabs" role="tablist" aria-label="Timeline event payloads">
              {timelineTabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  class={`timeline-inspector-tab${selectedTab === tab.id ? " is-active" : ""}`}
                  onClick={() => setSelectedTab(tab.id)}
                  role="tab"
                  aria-selected={selectedTab === tab.id}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            <div class="timeline-inspector-body">
              {selectedEvent ? (
                <>
                  <div class="timeline-inspector-head">
                    <div>
                      <div class="timeline-inspector-command">{timelineEventTitle(selectedEvent)}</div>
                      {transitionLabel ? <div class="timeline-inspector-transition">{transitionLabel}</div> : null}
                      <div class="timeline-inspector-subtitle">{selectedEvent.summary}</div>
                    </div>
                    <div class="timeline-inspector-refs">
                      <span>{selectedEvent.event_id}</span>
                      <span>{formatTimestamp(selectedEvent.recorded_at)}</span>
                    </div>
                  </div>

                  <pre class="timeline-json" aria-label={`${selectedTab} payload`}>
                    {timelineTabText(selectedTabValue, selectedTabMode)}
                  </pre>
                </>
              ) : (
                <div class="empty-row">Select an event to inspect its raw payload.</div>
              )}
            </div>
          </section>
        </section>
      </section>
    </section>
  );
}

function PlaceholderPage(props: { title: string; selectedSection: string }) {
  const heading = props.selectedSection === "status" ? "Status" : "Overview";
  const copy =
    props.selectedSection === "status"
      ? "Route and shell are live. Data hookup remains deferred for this page."
      : "This page is scaffolded but not yet wired to a full data view. The first release keeps the shell in place while the underlying contracts settle.";

  return (
    <section class="workspace">
      <div class="workspace-inner">
        <section id={props.selectedSection} class="pane pane-placeholder">
          <div class="section-head">
            <h2>{heading}</h2>
            <span class="muted">WIP</span>
          </div>
          <div class="placeholder-copy">{copy}</div>
        </section>
      </div>
    </section>
  );
}

render(<App />, document.getElementById("app") as HTMLElement);
