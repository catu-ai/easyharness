import { render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";

import "./styles.css";

import {
  combineLiveFreshness,
  describeLiveFreshness,
  formatPlanError,
  formatReviewError,
  formatStatusError,
  formatTimelineError,
  pickEntries,
  productNameLabel,
  workdirLabel,
} from "./helpers";
import { PlanWorkspace, ReviewWorkspace, StatusWorkspace, TimelineWorkspace } from "./pages";
import type { LiveFreshness, Page, PageDef, PlanResult, ReviewResult, StatusResult, TimelineResult } from "./types";
import { RailIcon, TopbarFreshness, TopbarMetric } from "./workbench";

const pages: PageDef[] = [
  { id: "status", label: "Status", href: "/status" },
  { id: "plan", label: "Plan", href: "/plan" },
  { id: "timeline", label: "Timeline", href: "/timeline" },
  { id: "review", label: "Review", href: "/review" },
];
const LIVE_REFRESH_INTERVAL_MS = 4000;
const LIVE_REFRESH_ACTIVITY_BUFFER_MS = 250;

type LiveResourceResult<T> = {
  data: T | null;
  error: string | null;
  loading: boolean;
  freshness: LiveFreshness;
};

function isPage(value: string | null): value is Page {
  return value === "status" || value === "plan" || value === "timeline" || value === "review";
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

function sectionIDsForPage(page: Page): string[] {
  if (page === "status") {
    return ["summary", "next-actions", "warnings", "facts", "artifacts"];
  }
  return ["overview"];
}

function readSectionFromLocation(page: Page): string {
  const section = window.location.hash.replace(/^#/, "");
  return sectionIDsForPage(page).includes(section) ? section : sectionIDsForPage(page)[0];
}

function formatTimelineResourceError(result: TimelineResult | null, statusCode?: number): string {
  return formatTimelineError(result?.summary, result?.errors, statusCode);
}

function useLiveResource<T>(options: {
  enabled: boolean;
  path: string;
  formatError: (result: T | null, statusCode?: number) => string;
  intervalMs?: number;
}): LiveResourceResult<T> {
  const { enabled, path, formatError, intervalMs = LIVE_REFRESH_INTERVAL_MS } = options;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [freshness, setFreshness] = useState<LiveFreshness>(() => describeLiveFreshness(enabled ? "connecting" : "idle"));
  const inFlightRef = useRef(false);
  const lastSuccessAtRef = useRef<string | null>(null);
  const hasSuccessfulLoadRef = useRef(false);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }

    let disposed = false;
    let activeController: AbortController | null = null;
    let updatingIndicatorTimeoutID: number | null = null;

    const clearUpdatingIndicator = () => {
      if (updatingIndicatorTimeoutID !== null) {
        window.clearTimeout(updatingIndicatorTimeoutID);
        updatingIndicatorTimeoutID = null;
      }
    };

    const refresh = (trigger: "initial" | "poll" | "focus") => {
      if (disposed) return;
      if (trigger === "poll" && document.visibilityState !== "visible") return;
      if (inFlightRef.current) {
        if (trigger === "poll") return;
        clearUpdatingIndicator();
        activeController?.abort();
        inFlightRef.current = false;
      }

      const hasLiveData = hasSuccessfulLoadRef.current;
      setLoading(!hasLiveData);

      const controller = new AbortController();
      activeController = controller;
      inFlightRef.current = true;
      clearUpdatingIndicator();
      if (hasLiveData) {
        updatingIndicatorTimeoutID = window.setTimeout(() => {
          updatingIndicatorTimeoutID = null;
          if (disposed || controller.signal.aborted) return;
          setFreshness(describeLiveFreshness("updating", lastSuccessAtRef.current));
        }, LIVE_REFRESH_ACTIVITY_BUFFER_MS);
      } else {
        setFreshness(describeLiveFreshness("connecting", lastSuccessAtRef.current));
      }

      fetch(path, { signal: controller.signal })
        .then(async (response) => {
          const payload = (await response.json()) as T & { ok?: boolean };
          if (!response.ok || payload.ok === false) {
            throw new Error(formatError(payload as T, response.status));
          }
          return payload as T;
        })
        .then((payload) => {
          if (disposed || controller.signal.aborted) return;
          clearUpdatingIndicator();
          const nextSuccessAt = new Date().toISOString();
          hasSuccessfulLoadRef.current = true;
          lastSuccessAtRef.current = nextSuccessAt;
          setData(payload);
          setError(null);
          setLoading(false);
          setFreshness(describeLiveFreshness("live", nextSuccessAt));
        })
        .catch((nextError: unknown) => {
          if (disposed || controller.signal.aborted) return;
          clearUpdatingIndicator();
          const message = nextError instanceof Error ? nextError.message : `Unable to load ${path}`;
          setError(message);
          setLoading(false);
          if (!hasSuccessfulLoadRef.current) {
            setData(null);
          }
          setFreshness(
            describeLiveFreshness(hasSuccessfulLoadRef.current ? "stale" : "disconnected", lastSuccessAtRef.current, message),
          );
        })
        .finally(() => {
          if (activeController === controller) {
            activeController = null;
            inFlightRef.current = false;
          }
        });
    };

    const refreshOnFocus = () => refresh("focus");
    const refreshOnVisibility = () => {
      if (document.visibilityState === "visible") {
        refresh("focus");
      }
    };

    refresh("initial");
    const intervalID = window.setInterval(() => refresh("poll"), intervalMs);
    window.addEventListener("focus", refreshOnFocus);
    document.addEventListener("visibilitychange", refreshOnVisibility);

    return () => {
      disposed = true;
      clearUpdatingIndicator();
      window.clearInterval(intervalID);
      window.removeEventListener("focus", refreshOnFocus);
      document.removeEventListener("visibilitychange", refreshOnVisibility);
      activeController?.abort();
      inFlightRef.current = false;
    };
  }, [enabled, formatError, intervalMs, path]);

  return { data, error, loading, freshness };
}

function App() {
  const [page, setPage] = useState<Page>(() => readPageFromLocation());
  const [section, setSection] = useState<string>(() => readSectionFromLocation(readPageFromLocation()));

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

  useEffect(() => {
    if (pageFromPathname(window.location.pathname) === null && !window.location.hash) {
      window.history.replaceState({}, "", `${pageDefinition(page).href}#${sectionIDsForPage(page)[0]}`);
    }
  }, [page]);

  const navigateToPage = (nextPage: Page, nextSection = sectionIDsForPage(nextPage)[0]) => {
    const nextURL = `${pageDefinition(nextPage).href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setPage(nextPage);
    setSection(nextSection);
  };

  const navigateToSection = (nextSection: string) => {
    navigateToPage(page, nextSection);
  };
  const statusResource = useLiveResource<StatusResult>({
    enabled: true,
    path: "/api/status",
    formatError: formatStatusError,
  });
  const planResource = useLiveResource<PlanResult>({
    enabled: page === "plan",
    path: "/api/plan",
    formatError: formatPlanError,
  });
  const timelineResource = useLiveResource<TimelineResult>({
    enabled: page === "timeline",
    path: "/api/timeline",
    formatError: formatTimelineResourceError,
  });
  const reviewResource = useLiveResource<ReviewResult>({
    enabled: page === "review",
    path: "/api/review",
    formatError: formatReviewError,
  });

  const { data: status, error: statusError, loading: statusLoading, freshness: statusFreshness } = statusResource;
  const { data: plan, error: planError, loading: planLoading, freshness: planFreshness } = planResource;
  const { data: timeline, error: timelineError, loading: timelineLoading, freshness: timelineFreshness } = timelineResource;
  const { data: review, error: reviewError, loading: reviewLoading, freshness: reviewFreshness } = reviewResource;

  const activeStatus = useMemo(
    () => ({
      summary: status?.summary ?? "Waiting for status data.",
      currentNode: status?.state?.current_node ?? "unknown",
      nextActions: Array.isArray(status?.next_actions) ? status.next_actions ?? [] : [],
      blockers: Array.isArray(status?.blockers) ? status.blockers ?? [] : [],
      warnings: Array.isArray(status?.warnings) ? status.warnings ?? [] : [],
      errors: Array.isArray(status?.errors) ? status.errors ?? [] : [],
      facts: pickEntries(status?.facts),
      artifacts: pickEntries(status?.artifacts),
    }),
    [status],
  );

  const activeTimeline = useMemo(
    () => ({
      events: Array.isArray(timeline?.events) ? timeline.events ?? [] : [],
    }),
    [timeline],
  );

  const activePlan = useMemo(
    () => ({
      summary: plan?.summary ?? "Waiting for plan data.",
      document: plan?.document ?? null,
      supplements: plan?.supplements ?? null,
      warnings: Array.isArray(plan?.warnings) ? plan.warnings ?? [] : [],
      artifacts: pickEntries((plan?.artifacts as Record<string, unknown>) ?? null),
    }),
    [plan],
  );

  const activeReview = useMemo(
    () => ({
      rounds: Array.isArray(review?.rounds) ? review.rounds ?? [] : [],
      warnings: Array.isArray(review?.warnings) ? review.warnings ?? [] : [],
      artifacts: pickEntries((review?.artifacts as Record<string, unknown>) ?? null),
      summary: review?.summary ?? "Waiting for review data.",
    }),
    [review],
  );

  const shellFreshness = useMemo(() => {
    if (page === "status") return statusFreshness;
    if (page === "plan") return combineLiveFreshness([statusFreshness, planFreshness]);
    if (page === "timeline") return combineLiveFreshness([statusFreshness, timelineFreshness]);
    return combineLiveFreshness([statusFreshness, reviewFreshness]);
  }, [page, planFreshness, reviewFreshness, statusFreshness, timelineFreshness]);

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{productNameLabel()}</span>
        </div>
        <div class="workspace-path" title={workdirLabel()}>
          {workdirLabel()}
        </div>
        <TopbarFreshness freshness={shellFreshness} />
        <div class="topbar-summary">
          <TopbarMetric kind="node" label="Node" value={activeStatus.currentNode} onClick={() => navigateToPage("status", "summary")} />
          {activeStatus.blockers.length > 0 ? (
            <TopbarMetric
              kind="blockers"
              label="Blockers"
              value={String(activeStatus.blockers.length)}
              tone="danger"
              onClick={() => navigateToPage("status", "warnings")}
            />
          ) : null}
          <TopbarMetric
            kind="warnings"
            label="Warnings"
            value={String(activeStatus.warnings.length)}
            tone={activeStatus.warnings.length > 0 ? "warning" : "muted"}
            onClick={() => navigateToPage("status", "warnings")}
          />
          <TopbarMetric
            kind="actions"
            label="Actions"
            value={String(activeStatus.nextActions.length)}
            tone={activeStatus.nextActions.length > 0 ? "good" : "muted"}
            onClick={() => navigateToPage("status", "next-actions")}
          />
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

        <main class="main-stage">
          {page === "plan" ? (
            <PlanWorkspace
              loading={planLoading}
              error={planError}
              summary={activePlan.summary}
              document={activePlan.document}
              supplements={activePlan.supplements}
              warnings={activePlan.warnings}
            />
          ) : page === "timeline" ? (
            <TimelineWorkspace loading={timelineLoading} error={timelineError} events={activeTimeline.events} />
          ) : page === "review" ? (
            <ReviewWorkspace
              loading={reviewLoading}
              error={reviewError}
              summary={activeReview.summary}
              rounds={activeReview.rounds}
              warnings={activeReview.warnings}
              artifacts={activeReview.artifacts}
            />
          ) : (
            <StatusWorkspace
              loading={statusLoading}
              error={statusError}
              summary={activeStatus.summary}
              currentNode={activeStatus.currentNode}
              nextActions={activeStatus.nextActions}
              blockers={activeStatus.blockers}
              warnings={activeStatus.warnings}
              errors={activeStatus.errors}
              facts={activeStatus.facts}
              artifacts={activeStatus.artifacts}
              selectedSection={section}
              onSelectSection={navigateToSection}
            />
          )}
        </main>
      </div>
    </div>
  );
}

render(<App />, document.getElementById("app") as HTMLElement);
