import { createStudioClient } from "@studio/api-client";

function readCookie(name: string): string | undefined {
  if (typeof document === "undefined") return undefined;
  const prefix = `${encodeURIComponent(name)}=`;
  const entry = document.cookie.split("; ").find((item) => item.startsWith(prefix));
  return entry ? decodeURIComponent(entry.slice(prefix.length)) : undefined;
}

export const csrfFetch: typeof fetch = async (input, init) => {
  // openapi-fetch normally passes a fully constructed Request rather than a URL
  // plus RequestInit. Read the effective request so its method and headers are
  // preserved before adding browser-only security headers.
  const request = new Request(input, init);
  const method = request.method.toUpperCase();
  const headers = new Headers(request.headers);
  if (!["GET", "HEAD", "OPTIONS"].includes(method)) {
    const csrfToken = readCookie("studio_csrf");
    if (csrfToken) headers.set("X-CSRF-Token", csrfToken);
  }
  headers.set("Accept", "application/json, application/problem+json");
  return fetch(request, { headers, credentials: "include" });
};

export const api = createStudioClient("/api/studio", csrfFetch);
