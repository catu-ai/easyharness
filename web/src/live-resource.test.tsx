import { render, screen, waitFor } from "@testing-library/preact";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

import { useLiveResource } from "./live-resource";

function ResourceProbe(props: { path: string }) {
  const resource = useLiveResource<{ ok?: boolean; value?: string }>({
    enabled: true,
    path: props.path,
    formatError: (result, statusCode) => result?.value || (statusCode ? `failed:${statusCode}` : "failed"),
    intervalMs: 60_000,
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

    const rendered = render(<ResourceProbe path="/api/workspace/alpha" />);
    await waitFor(() => expect(screen.getByTestId("value").textContent).toBe("alpha"));

    rendered.rerender(<ResourceProbe path="/api/workspace/beta" />);

    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("loading").textContent).toBe("true");

    await waitFor(() => expect(screen.getByTestId("freshness").textContent).toBe("disconnected"));
    expect(screen.getByTestId("value").textContent).toBe("none");
    expect(screen.getByTestId("error").textContent).toBe("beta missing");
  });
});
