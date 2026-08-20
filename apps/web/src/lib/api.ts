import { createStudioClient } from "@studio/api-client";

function readCookie(name: string): string | undefined {
  if (typeof document === "undefined") return undefined;
  const prefix = `${encodeURIComponent(name)}=`;
  const entry = document.cookie.split("; ").find((item) => item.startsWith(prefix));
  return entry ? decodeURIComponent(entry.slice(prefix.length)) : undefined;
}

const csrfFetch: typeof fetch = async (input, init = {}) => {
  const method = (init.method ?? "GET").toUpperCase();
  const headers = new Headers(init.headers);
  if (!["GET", "HEAD", "OPTIONS"].includes(method)) {
    const csrfToken = readCookie("studio_csrf");
    if (csrfToken) headers.set("X-CSRF-Token", csrfToken);
  }
  headers.set("Accept", "application/json, application/problem+json");
  return fetch(input, { ...init, headers, credentials: "include" });
};

export const api = createStudioClient("/api/studio", csrfFetch);
