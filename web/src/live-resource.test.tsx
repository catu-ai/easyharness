import { cleanup, render, screen, waitFor } from "@testing-library/preact";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

import { useLiveResource } from "./live-resource";

function ResourceProbe(props: {
  resource: { key: string; path: string } | null;
  mode?: "live" | "paused";
  intervalMs?: number;
}) {
  const resource = useLiveResource<{ ok?: boolean; value?: string }>({
    resource: props.resource,
    mode: props.mode ?? "live",
    formatError: (result, statusCode) => result?.value || (statusCode ? `failed:${statusCode}` : "failed"),
    intervalMs: props.intervalMs ?? 60_000,
  });

  return (
    <div>
      <span data-testid="value">{resource.data?.value ?? "none"}</span>
      <span data-testid="error">{resource.error ?? ""}</span>
      <span data-testid="loading">{String(resource.loading)}</span>
      <span data-testid="freshness">{resource.freshness.kind}</span>
    </div>
  );
}

describe("useLiveResource", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.useRealTimers();
    cleanup();
    vi.unstubAllGlobals();
  });

  test("clears the previous resource when the path changes", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockImplementation((input) => {
      const path = String(input);
      if (path === "/api/workspace/alpha") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ ok: true, value: "alpha" }),
        } as Response);
      }
      if (path === "/api/workspace/beta") {
        return Promise.resolve({
          ok: false,
          status: 404,
          json: async () => ({ ok: false, value: "beta missing" }),
        } as Response);
      }
      return Promise.reject(new Error(`unexpected path ${path}`));
    });

    const rendered = render(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));

    rendered.rerender(<ResourceProbe resource={{ key: "workspace:beta", path: "/api/workspace/beta" }} />);

    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("loading").textContent).toBe("true");

    await waitFor(() => expect(screen.getByTestId("freshness").textContent).toBe("disconnected"));
    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("error").textContent).toBe("beta missing");
  });

  test("retains successful data while paused without polling", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ ok: true, value: "alpha" }) } as Response);

    const rendered = render(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} intervalMs={10} />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));
    expect(fetchMock).toHaveBeenCalledTimes(1);

    rendered.rerender(
      <ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} mode="paused" intervalMs={10} />,
    );

    expect(screen.getByTestId("value").textContent).toBe("alpha");
    expect(screen.getByTestId("loading").textContent).toBe("false");

    await new Promise((resolve) => window.setTimeout(resolve, 40));
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  test("refreshes in the background when paused data returns to live mode", async () => {
    const refresh = { resolve: null as ((response: Response) => void) | null };
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce({ ok: true, json: async () => ({ ok: true, value: "alpha" }) } as Response)
      .mockImplementationOnce(
        () =>
          new Promise<Response>((resolve) => {
            refresh.resolve = resolve;
          }),
      );

    const rendered = render(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));

    rendered.rerender(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} mode="paused" />);
    rendered.rerender(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(screen.getByTestId("value").textContent).toBe("alpha");
    expect(screen.getByTestId("loading").textContent).toBe("false");

    if (!refresh.resolve) throw new Error("Expected pending refresh");
    refresh.resolve({ ok: true, json: async () => ({ ok: true, value: "alpha refreshed" }) } as Response);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha refreshed"));
  });

  test("keeps retained data and reports stale when a resumed refresh fails", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce({ ok: true, json: async () => ({ ok: true, value: "alpha" }) } as Response)
      .mockResolvedValueOnce({ ok: false, status: 500, json: async () => ({ ok: false, value: "refresh failed" }) } as Response);

    const rendered = render(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));

    rendered.rerender(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} mode="paused" />);
    rendered.rerender(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);

    await waitFor(() => expect(screen.getByTestId("freshness").textContent).toBe("stale"));
    expect(screen.getByTestId("value").textContent).toBe("alpha");
    expect(screen.getByTestId("error").textContent).toBe("refresh failed");
  });

  test("renders disconnected empty state when the first live load fails", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue({ ok: false, status: 404, json: async () => ({ ok: false, value: "missing" }) } as Response);

    render(<ResourceProbe resource={{ key: "workspace:missing", path: "/api/workspace/missing" }} />);

    await waitFor(() => expect(screen.getByTestId("freshness").textContent).toBe("disconnected"));
    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("error").textContent).toBe("missing");
  });

  test("clears retained data when the resource becomes invalid", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ ok: true, value: "alpha" }) } as Response);

    const rendered = render(<ResourceProbe resource={{ key: "workspace:alpha", path: "/api/workspace/alpha" }} />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));

    rendered.rerender(<ResourceProbe resource={null} />);

    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("freshness").textContent).toBe("idle");
  });
});
