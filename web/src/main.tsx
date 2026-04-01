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

  const activeSectionLabel =
    sectionsForPage(page, activeStatus).find((item) => item.id === section)?.label ?? pageDefinition(page).label;

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

        <main class="content">
          <aside class="sidebar" aria-label={`${pageDefinition(page).label} sidebar`}>
            <div class="sidebar-header">
              <span class="sidebar-label">Explorer</span>
              <strong>{pageDefinition(page).label}</strong>
            </div>
            <nav class="sidebar-group" aria-label={`${pageDefinition(page).label} sections`}>
              {sectionsForPage(page, activeStatus).map((item) => (
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
