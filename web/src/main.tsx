import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import "./styles.css";

type Page = "status" | "timeline" | "review" | "diff" | "files";
type PageDef = { id: Page; label: string; href: string };

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

function repoNameLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.repoName) || "harness";
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

function App() {
  const [page, setPage] = useState<Page>(() => readPageFromLocation());
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);

  useEffect(() => {
    const onLocationChange = () => setPage(readPageFromLocation());
    window.addEventListener("popstate", onLocationChange);
    window.addEventListener("hashchange", onLocationChange);
    return () => {
      window.removeEventListener("popstate", onLocationChange);
      window.removeEventListener("hashchange", onLocationChange);
    };
  }, []);

  const navigateToPage = (nextPage: Page) => {
    const next = pageDefinition(nextPage);
    if (window.location.pathname !== next.href || window.location.hash) {
      window.history.pushState({}, "", next.href);
    }
    setPage(nextPage);
  };

  useEffect(() => {
    if (pageFromPathname(window.location.pathname) === null && !window.location.hash) {
      window.history.replaceState({}, "", pageDefinition(page).href);
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

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{repoNameLabel()}</span>
          <span class="brand-subtitle">harness ui</span>
        </div>
        <div class="workspace-path">{workdirLabel()}</div>
        <div class="topbar-meta">
          <span class="status-pill">read-only</span>
          <span class="status-pill">local</span>
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
                onClick={(event) => {
                  event.preventDefault();
                  navigateToPage(item.id);
                }}
              >
                <span class="rail-dot" />
                <span>{item.label}</span>
              </a>
            );
          })}
        </aside>

        <main class="content">
          <section class="page-header">
            <div>
              <p class="eyebrow">Local workbench</p>
              <h1>{pageDefinition(page).label}</h1>
            </div>
            <p class="lede">
              Agent executes. Human steers. This shell stays read-only and reads from the current worktree.
            </p>
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
            />
          ) : (
            <PlaceholderPage title={pageDefinition(page).label} />
          )}
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
}) {
  const { loading, error, summary, currentNode, nextActions, blockers, warnings, errors, facts, artifacts } = props;

  return (
    <section class="workspace">
      <article class="panel panel-main">
        <div class="panel-head">
          <h2>Summary</h2>
          {loading ? <span class="muted">loading</span> : null}
        </div>
        <p class="summary-text">{summary}</p>
        <div class="status-grid">
          <div class="status-block">
            <span class="label">current_node</span>
            <strong>{currentNode}</strong>
          </div>
          <div class="status-block">
            <span class="label">next_actions</span>
            <strong>{nextActions.length} item(s)</strong>
          </div>
          <div class="status-block">
            <span class="label">warnings</span>
            <strong>{warnings.length}</strong>
          </div>
          <div class="status-block">
            <span class="label">blockers</span>
            <strong>{blockers.length}</strong>
          </div>
        </div>

        {error ? <div class="notice notice-error">{error}</div> : null}

        <div class="two-column">
          <section>
            <h3>Next actions</h3>
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

          <section>
            <h3>Warnings & blockers</h3>
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
        </div>

        <div class="two-column">
          <section>
            <h3>Facts</h3>
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

          <section>
            <h3>Artifacts</h3>
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
        </div>
      </article>
    </section>
  );
}

function PlaceholderPage(props: { title: string }) {
  return (
    <section class="workspace">
      <article class="panel panel-placeholder">
        <p class="eyebrow">{props.title}</p>
        <h2>WIP</h2>
        <p class="summary-text">
          This page is intentionally scaffolded but not yet wired to a full data view. The first release keeps
          the shell in place while the underlying contracts settle.
        </p>
      </article>
    </section>
  );
}

render(<App />, document.getElementById("app") as HTMLElement);
