/**
 * Custom fetcher for the orval-generated API client. Calls go through the
 * web's same-origin core proxy (apps/web routes/api/core.$.ts), which attaches
 * the person's urbangate token from the session and forwards to the Go core:
 * no CORS, no token in the browser. The core mounts /api/v1 (apps/core/main.go),
 * and orval-generated urls are relative to it (e.g. /examples).
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
const API_BASE = "/api/core/api/v1";

export async function coreFetcher<T>(
  url: string,
  init?: RequestInit,
): Promise<T> {
  const fullUrl = url.startsWith("http") ? url : `${API_BASE}${url}`;

  // FormData uploads need the browser to set Content-Type (with its boundary);
  // forcing application/json would corrupt them.
  const isFormData =
    typeof FormData !== "undefined" && init?.body instanceof FormData;

  const res = await fetch(fullUrl, {
    ...init,
    credentials: "include",
    headers: {
      ...(isFormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers as Record<string, string>),
    },
  });

  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  const json = body ? JSON.parse(body) : {};

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("auth:unauthorized"));
    }
    throw { message: "Unauthorized", status: 401, data: json };
  }

  if (!res.ok) {
    const errorMessage =
      ((json as Record<string, unknown>)?.message as string) ||
      ((json as Record<string, unknown>)?.error as string) ||
      "Request failed";
    throw { message: errorMessage, status: res.status, data: json };
  }

  return { data: json, status: res.status, headers: res.headers } as T;
}
