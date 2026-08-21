const storageOriginPattern = /^https?:\/\/(?:\*\.)?[a-z0-9.-]+(?::\d{1,5})?$/i;

export function parseBrowserStorageOrigins(raw: string | undefined): string[] {
  const values = (raw ?? "").split(/\s+/).filter(Boolean);
  const unique = new Set<string>();
  for (const value of values) {
    if (!storageOriginPattern.test(value)) {
      throw new Error(`Invalid BROWSER_STORAGE_ORIGINS entry: ${value}`);
    }
    const parsed = new URL(value.replace("://*.", "://wildcard."));
    const port = parsed.port ? Number(parsed.port) : null;
    if (parsed.username || parsed.password || parsed.pathname !== "/" || parsed.search || parsed.hash || (port !== null && (port < 1 || port > 65_535))) {
      throw new Error(`Invalid BROWSER_STORAGE_ORIGINS entry: ${value}`);
    }
    unique.add(value);
  }
  return [...unique];
}

export function contentSecurityPolicy(browserStorageOrigins: string | undefined): string {
  const storageOrigins = parseBrowserStorageOrigins(browserStorageOrigins);
  const sources = storageOrigins.length > 0 ? ` ${storageOrigins.join(" ")}` : "";
  return [
    "default-src 'self'",
    "base-uri 'self'",
    "frame-ancestors 'none'",
    "object-src 'none'",
    `img-src 'self' data: blob:${sources}`,
    `media-src 'self' blob:${sources}`,
    `connect-src 'self'${sources}`,
    "style-src 'self' 'unsafe-inline'",
    "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'",
    "form-action 'self'",
  ].join("; ");
}
