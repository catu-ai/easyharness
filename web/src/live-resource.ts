import { useEffect, useRef, useState } from "preact/hooks";

import { describeLiveFreshness } from "./helpers";
import type { LiveFreshness } from "./types";

const LIVE_REFRESH_INTERVAL_MS = 4000;
const LIVE_REFRESH_ACTIVITY_BUFFER_MS = 250;

export type LiveResourceResult<T> = {
  data: T | null;
  error: string | null;
  loading: boolean;
  freshness: LiveFreshness;
};

export type LiveResourceDescriptor = {
  key: string;
  path: string;
};

export function useLiveResource<T>(options: {
  resource: LiveResourceDescriptor | null;
  mode?: "live" | "paused";
  formatError: (result: T | null, statusCode?: number) => string;
  intervalMs?: number;
}): LiveResourceResult<T> {
  const { resource, mode = "live", formatError, intervalMs = LIVE_REFRESH_INTERVAL_MS } = options;
  const resourceKey = resource?.key ?? null;
  const resourcePath = resource?.path ?? null;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [freshness, setFreshness] = useState<LiveFreshness>(() => describeLiveFreshness(resource && mode === "live" ? "connecting" : "idle"));
  const inFlightRef = useRef(false);
  const lastSuccessAtRef = useRef<string | null>(null);
  const hasSuccessfulLoadRef = useRef(false);
  const resourceKeyRef = useRef<string | null>(null);
  const formatErrorRef = useRef(formatError);

  useEffect(() => {
    formatErrorRef.current = formatError;
  }, [formatError]);

  useEffect(() => {
    if (!resourceKey) {
      resourceKeyRef.current = null;
      hasSuccessfulLoadRef.current = false;
      lastSuccessAtRef.current = null;
      setData(null);
      setError(null);
      setLoading(false);
      setFreshness(describeLiveFreshness("idle"));
      return;
    }

    if (resourceKeyRef.current === resourceKey) return;
    resourceKeyRef.current = resourceKey;
    hasSuccessfulLoadRef.current = false;
    lastSuccessAtRef.current = null;
    setData(null);
    setError(null);
    setLoading(mode === "live");
    setFreshness(describeLiveFreshness(mode === "live" ? "connecting" : "idle"));
  }, [mode, resourceKey]);

  useEffect(() => {
    if (!resourcePath || mode !== "live") return;

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
        if (trigger === "initial") {
          setFreshness(describeLiveFreshness("updating", lastSuccessAtRef.current));
        } else {
          updatingIndicatorTimeoutID = window.setTimeout(() => {
            updatingIndicatorTimeoutID = null;
            if (disposed || controller.signal.aborted) return;
            setFreshness(describeLiveFreshness("updating", lastSuccessAtRef.current));
          }, LIVE_REFRESH_ACTIVITY_BUFFER_MS);
        }
      } else {
        setFreshness(describeLiveFreshness("connecting", lastSuccessAtRef.current));
      }

      fetch(resourcePath, { signal: controller.signal })
        .then(async (response) => {
          const payload = (await response.json()) as T & { ok?: boolean };
          if (!response.ok || payload.ok === false) {
            throw new Error(formatErrorRef.current(payload as T, response.status));
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
          const message = nextError instanceof Error ? nextError.message : `Unable to load ${resourcePath}`;
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
  }, [intervalMs, mode, resourceKey, resourcePath]);

  return { data, error, loading, freshness };
}
