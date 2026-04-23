import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import "./styles.css";

import {
  combineLiveFreshness,
  dashboardWorkspaces,
  formatDashboardError,
  formatPlanError,
  formatReviewError,
  formatStatusError,
  formatTimelineError,
  pickEntries,
  productNameLabel,
  workdirLabel,
} from "./helpers";
import { useLiveResource } from "./live-resource";
import { DashboardHome, PlanWorkspace, ReviewWorkspace, StatusWorkspace, TimelineWorkspace, WorkspaceDegradedPage } from "./pages";
import type {
  DashboardResult,
  DashboardWorkspace,
  Page,
  PlanResult,
  ReviewResult,
  StatusResult,
  TimelineResult,
  WorkspaceRouteResult,
} from "./types";
import { buildWorkspaceUnwatchRequest } from "./workspace-actions";
import { RailIcon, TopbarFreshness, TopbarMetric } from "./workbench";

const pages: Array<{ id: Page; label: string }> = [
  { id: "status", label: "Status" },
  { id: "plan", label: "Plan" },
  { id: "timeline", label: "Timeline" },
  { id: "review", label: "Review" },
];

type AppRoute =
  | { kind: "dashboard" }
  | {
      kind: "workspace";
      workspaceKey: string;
      page: Page;
    };

function isPage(value: string | null): value is Page {
  return value === "status" || value === "plan" || value === "timeline" || value === "review";
}

function sectionIDsForPage(page: Page): string[] {
  if (page === "status") {
    return ["summary", "next-actions", "warnings", "facts", "artifacts"];
  }
  return ["overview"];
}

function readRouteFromLocation(): AppRoute {
  const trimmed = window.location.pathname.replace(/\/+$/, "");
  const parts = trimmed.split("/").filter(Boolean);
  if (parts.length === 0 || (parts.length === 1 && parts[0] === "dashboard")) {
    return { kind: "dashboard" };
  }
  if (parts[0] === "workspace" && parts[1]) {
    const page = isPage(parts[2] ?? "status") ? (parts[2] as Page) : "status";
    return { kind: "workspace", workspaceKey: parts[1], page };
  }
  return { kind: "dashboard" };
}

function readSectionFromLocation(route: AppRoute): string {
  if (route.kind !== "workspace") return "overview";
  const section = window.location.hash.replace(/^#/, "");
  return sectionIDsForPage(route.page).includes(section) ? section : sectionIDsForPage(route.page)[0];
}

function formatTimelineResourceError(result: TimelineResult | null, statusCode?: number): string {
  return formatTimelineError(result?.summary, result?.errors, statusCode);
}

function App() {
  const [route, setRoute] = useState<AppRoute>(() => readRouteFromLocation());
  const [section, setSection] = useState<string>(() => readSectionFromLocation(readRouteFromLocation()));
  const [busyWorkspaceKey, setBusyWorkspaceKey] = useState<string | null>(null);

  useEffect(() => {
    const onLocationChange = () => {
      const nextRoute = readRouteFromLocation();
      setRoute(nextRoute);
      setSection(readSectionFromLocation(nextRoute));
    };
    window.addEventListener("popstate", onLocationChange);
    window.addEventListener("hashchange", onLocationChange);
    return () => {
      window.removeEventListener("popstate", onLocationChange);
      window.removeEventListener("hashchange", onLocationChange);
    };
  }, []);

  useEffect(() => {
    if (route.kind === "workspace" && !window.location.hash) {
      window.history.replaceState({}, "", `${workspacePageHref(route.workspaceKey, route.page)}#${sectionIDsForPage(route.page)[0]}`);
    }
  }, [route]);

  const navigateToDashboard = () => {
    if (window.location.pathname !== "/dashboard") {
      window.history.pushState({}, "", "/dashboard");
    }
    setRoute({ kind: "dashboard" });
    setSection("overview");
  };

  const navigateToWorkspacePage = (workspaceKey: string, page: Page, nextSection = sectionIDsForPage(page)[0]) => {
    const nextURL = `${workspacePageHref(workspaceKey, page)}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setRoute({ kind: "workspace", workspaceKey, page });
    setSection(nextSection);
  };

  const dashboardResource = useLiveResource<DashboardResult>({
    enabled: route.kind === "dashboard",
    path: "/api/dashboard",
    formatError: (result, statusCode) => result?.summary?.trim() || (statusCode ? `GET /api/dashboard failed with ${statusCode}` : "Unable to load dashboard"),
  });
  const workspaceResource = useLiveResource<WorkspaceRouteResult>({
    enabled: route.kind === "workspace",
    path: route.kind === "workspace" ? `/api/workspace/${route.workspaceKey}` : "/api/workspace/_",
    formatError: formatDashboardError,
  });

  const selectedWorkspace = route.kind === "workspace" ? workspaceResource.data?.workspace ?? null : null;
  const workspaceReadable =
    route.kind === "workspace" &&
    workspaceResource.data?.watched === true &&
    selectedWorkspace !== null &&
    selectedWorkspace.dashboard_state !== "missing" &&
    selectedWorkspace.dashboard_state !== "invalid";

  const statusResource = useLiveResource<StatusResult>({
    enabled: workspaceReadable,
    path: route.kind === "workspace" ? `/api/workspace/${route.workspaceKey}/status` : "/api/workspace/_/status",
    formatError: formatStatusError,
  });
  const planResource = useLiveResource<PlanResult>({
    enabled: workspaceReadable && route.kind === "workspace" && route.page === "plan",
    path: route.kind === "workspace" ? `/api/workspace/${route.workspaceKey}/plan` : "/api/workspace/_/plan",
    formatError: formatPlanError,
  });
  const timelineResource = useLiveResource<TimelineResult>({
    enabled: workspaceReadable && route.kind === "workspace" && route.page === "timeline",
    path: route.kind === "workspace" ? `/api/workspace/${route.workspaceKey}/timeline` : "/api/workspace/_/timeline",
    formatError: formatTimelineResourceError,
  });
  const reviewResource = useLiveResource<ReviewResult>({
    enabled: workspaceReadable && route.kind === "workspace" && route.page === "review",
    path: route.kind === "workspace" ? `/api/workspace/${route.workspaceKey}/review` : "/api/workspace/_/review",
    formatError: formatReviewError,
  });

  const { data: dashboard, error: dashboardError, loading: dashboardLoading, freshness: dashboardFreshness } = dashboardResource;
  const { data: status, error: statusError, loading: statusLoading, freshness: statusFreshness } = statusResource;
  const { data: plan, error: planError, loading: planLoading, freshness: planFreshness } = planResource;
  const { data: timeline, error: timelineError, loading: timelineLoading, freshness: timelineFreshness } = timelineResource;
  const { data: review, error: reviewError, loading: reviewLoading, freshness: reviewFreshness } = reviewResource;

  const activeStatus = useMemo(
    () => ({
      summary: status?.summary ?? "Waiting for status data.",
      currentNode: status?.state?.current_node ?? selectedWorkspace?.current_node ?? "unknown",
      nextActions: Array.isArray(status?.next_actions) ? status.next_actions ?? [] : [],
      blockers: Array.isArray(status?.blockers) ? status.blockers ?? [] : [],
      warnings: Array.isArray(status?.warnings) ? status.warnings ?? [] : [],
      errors: Array.isArray(status?.errors) ? status.errors ?? [] : [],
      facts: pickEntries((status?.facts as Record<string, unknown>) ?? null),
      artifacts: pickEntries(status?.artifacts),
    }),
    [selectedWorkspace?.current_node, status],
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
    if (route.kind === "dashboard") return dashboardFreshness;
    if (!workspaceReadable) return workspaceResource.freshness;
    if (route.page === "status") return statusFreshness;
    if (route.page === "plan") return combineLiveFreshness([statusFreshness, planFreshness]);
    if (route.page === "timeline") return combineLiveFreshness([statusFreshness, timelineFreshness]);
    return combineLiveFreshness([statusFreshness, reviewFreshness]);
  }, [
    dashboardFreshness,
    planFreshness,
    reviewFreshness,
    route,
    statusFreshness,
    timelineFreshness,
    workspaceReadable,
    workspaceResource.freshness,
  ]);

  const dashboardEntries = useMemo(() => dashboardWorkspaces(dashboard?.groups), [dashboard?.groups]);

  const unwatchWorkspace = (workspace: DashboardWorkspace) => {
    setBusyWorkspaceKey(workspace.workspace_key);
    const request = buildWorkspaceUnwatchRequest(workspace);
    fetch(request.url, request.init)
      .then(async (response) => {
        const payload = (await response.json()) as { ok?: boolean; summary?: string };
        if (!response.ok || payload.ok === false) {
          throw new Error(payload.summary || "Unable to remove workspace from the machine-local watchlist.");
        }
        if (route.kind === "workspace" && route.workspaceKey === workspace.workspace_key) {
          window.location.assign("/dashboard");
          return;
        }
        window.location.reload();
      })
      .catch((nextError: unknown) => {
        const message = nextError instanceof Error ? nextError.message : "Unable to remove workspace from the machine-local watchlist.";
        window.alert(message);
        setBusyWorkspaceKey(null);
      });
  };

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{productNameLabel()}</span>
        </div>
        <div class="workspace-path" title={route.kind === "workspace" ? selectedWorkspace?.workspace_path || workdirLabel() : "Dashboard"}>
          {route.kind === "workspace" ? selectedWorkspace?.workspace_path || workdirLabel() : "Dashboard"}
        </div>
        <TopbarFreshness freshness={shellFreshness} />
        {route.kind === "workspace" && workspaceReadable ? (
          <div class="topbar-summary">
            <TopbarMetric
              kind="node"
              label="Node"
              value={activeStatus.currentNode}
              onClick={() => navigateToWorkspacePage(route.workspaceKey, "status", "summary")}
            />
            {activeStatus.blockers.length > 0 ? (
              <TopbarMetric
                kind="blockers"
                label="Blockers"
                value={String(activeStatus.blockers.length)}
                tone="danger"
                onClick={() => navigateToWorkspacePage(route.workspaceKey, "status", "warnings")}
              />
            ) : null}
            <TopbarMetric
              kind="warnings"
              label="Warnings"
              value={String(activeStatus.warnings.length)}
              tone={activeStatus.warnings.length > 0 ? "warning" : "muted"}
              onClick={() => navigateToWorkspacePage(route.workspaceKey, "status", "warnings")}
            />
            <TopbarMetric
              kind="actions"
              label="Actions"
              value={String(activeStatus.nextActions.length)}
              tone={activeStatus.nextActions.length > 0 ? "good" : "muted"}
              onClick={() => navigateToWorkspacePage(route.workspaceKey, "status", "next-actions")}
            />
          </div>
        ) : null}
      </header>

      {route.kind === "dashboard" ? (
        <main class="dashboard-stage">
          <DashboardHome
            loading={dashboardLoading}
            error={dashboardError}
            workspaces={dashboardEntries}
            onOpenWorkspace={(workspaceKey) => navigateToWorkspacePage(workspaceKey, "status")}
            onUnwatch={unwatchWorkspace}
            busyWorkspaceKey={busyWorkspaceKey}
          />
        </main>
      ) : workspaceReadable ? (
        <div class="layout">
          <aside class="rail" aria-label="Pages">
            {pages.map((item) => {
              const selected = route.page === item.id;
              return (
                <a
                  key={item.id}
                  class={`rail-item${selected ? " is-active" : ""}`}
                  href={workspacePageHref(route.workspaceKey, item.id)}
                  aria-current={selected ? "page" : undefined}
                  aria-label={item.label}
                  title={item.label}
                  onClick={(event) => {
                    event.preventDefault();
                    navigateToWorkspacePage(route.workspaceKey, item.id);
                  }}
                >
                  <span class="rail-icon">
                    <RailIcon page={item.id} />
                  </span>
                  <span class="sr-only">{item.label}</span>
                </a>
              );
            })}
            <div class="rail-spacer" />
            <button type="button" class="rail-item" aria-label="Home" title="Home" onClick={navigateToDashboard}>
              <span class="rail-icon">
                <RailIcon page="home" />
              </span>
              <span class="sr-only">Home</span>
            </button>
          </aside>

          <main class="main-stage">
            {route.page === "plan" ? (
              <PlanWorkspace
                loading={planLoading}
                error={planError}
                summary={activePlan.summary}
                document={activePlan.document}
                supplements={activePlan.supplements}
                warnings={activePlan.warnings}
              />
            ) : route.page === "timeline" ? (
              <TimelineWorkspace loading={timelineLoading} error={timelineError} events={activeTimeline.events} />
            ) : route.page === "review" ? (
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
                onSelectSection={(nextSection) => {
                  setSection(nextSection);
                  const nextURL = `${workspacePageHref(route.workspaceKey, "status")}#${nextSection}`;
                  if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
                    window.history.pushState({}, "", nextURL);
                  }
                }}
              />
            )}
          </main>
        </div>
      ) : (
        <main class="dashboard-stage">
          <WorkspaceDegradedPage
            loading={workspaceResource.loading}
            error={workspaceResource.error}
            result={workspaceResource.data}
            onReturnDashboard={navigateToDashboard}
            onUnwatch={unwatchWorkspace}
            busyWorkspaceKey={busyWorkspaceKey}
          />
        </main>
      )}
    </div>
  );
}

function workspacePageHref(workspaceKey: string, page: Page): string {
  return `/workspace/${workspaceKey}/${page}`;
}

render(<App />, document.getElementById("app") as HTMLElement);
