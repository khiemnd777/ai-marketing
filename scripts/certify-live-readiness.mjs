#!/usr/bin/env node

const requiredProviders = ["OPENAI", "SEEDANCE", "R2", "META", "RENDERER"];
const timeoutMs = Number.parseInt(process.env.STUDIO_CERT_TIMEOUT_MS ?? "10000", 10);

function required(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function target(name, { https = false } = {}) {
  const value = new URL(required(name));
  if (!["http:", "https:"].includes(value.protocol) || value.username || value.password || value.search || value.hash) {
    throw new Error(`${name} must be a plain HTTP(S) origin without credentials, query, or fragment`);
  }
  if (https && value.protocol !== "https:") {
    const local = ["localhost", "127.0.0.1", "::1"].includes(value.hostname);
    if (!(local && process.env.STUDIO_CERT_ALLOW_INSECURE_LOCAL === "true")) {
      throw new Error(`${name} must use HTTPS (only explicit local certification may use HTTP)`);
    }
  }
  value.pathname = value.pathname.replace(/\/$/, "");
  return value;
}

async function request(url, init, label) {
  const response = await fetch(url, { ...init, signal: AbortSignal.timeout(timeoutMs), redirect: "error" });
  if (!response.ok) throw new Error(`${label} failed with HTTP ${response.status}`);
  return response;
}

async function json(url, init, label) {
  const response = await request(url, init, label);
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) throw new Error(`${label} returned a non-JSON response`);
  return { response, body: await response.json() };
}

function cookies(response) {
  const values = typeof response.headers.getSetCookie === "function"
    ? response.headers.getSetCookie()
    : [response.headers.get("set-cookie")].filter(Boolean);
  return values.map((value) => value.split(";", 1)[0]).join("; ");
}

const api = target("STUDIO_CERT_API_URL", { https: true });
const web = target("STUDIO_CERT_WEB_URL", { https: true });
const rendererValue = process.env.STUDIO_CERT_RENDERER_URL?.trim();
const renderer = rendererValue ? target("STUDIO_CERT_RENDERER_URL") : null;
const email = required("STUDIO_CERT_ADMIN_EMAIL");
const password = required("STUDIO_CERT_ADMIN_PASSWORD");
const clientId = required("STUDIO_CERT_CLIENT_ID");
if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(clientId)) {
  throw new Error("STUDIO_CERT_CLIENT_ID must be a UUID");
}

let cookie = "";
let csrfToken = "";
let failure;

try {
  const ready = await json(new URL("/v1/health/ready", api), { headers: { accept: "application/json" } }, "API readiness");
  if (ready.body?.status !== "ok" || ready.body?.checks?.database !== "ok") throw new Error("API readiness did not confirm database health");
  await request(new URL("/login", web), { headers: { accept: "text/html" } }, "Web login page");
  if (renderer) {
    const rendererReady = await json(new URL("/health/ready", renderer), { headers: { accept: "application/json" } }, "Renderer readiness");
    if (rendererReady.body?.status !== "ok") throw new Error("Renderer readiness returned an unexpected status");
  }

  const login = await json(new URL("/v1/auth/login", api), {
    method: "POST",
    headers: { accept: "application/json", "content-type": "application/json" },
    body: JSON.stringify({ email, password }),
  }, "Admin login");
  cookie = cookies(login.response);
  csrfToken = String(login.body?.csrfToken ?? "");
  if (!cookie || !csrfToken) throw new Error("Admin login did not return a complete session");
  if (login.body?.role !== "ADMIN" || login.body?.requiresPasswordChange) throw new Error("Certification account must be an active Admin without a forced password change");

  const status = await json(new URL(`/v1/clients/${clientId}/provider-configuration`, api), {
    headers: { accept: "application/json", cookie },
  }, "Provider readiness");
  if (status.body?.clientId !== clientId) throw new Error("Provider readiness returned a different client scope");
  if (status.body?.demoMode !== false) throw new Error("Live certification requires this client provider profile to be in LIVE mode");
  const providers = new Map((status.body?.providers ?? []).map((provider) => [provider.provider, provider]));
  const missing = requiredProviders.filter((name) => providers.get(name)?.configured !== true);
  if (missing.length) throw new Error(`Providers not configured: ${missing.join(", ")}`);
  const endpointByProvider = new Map([
    ["OPENAI", providers.get("OPENAI")?.settings?.baseUrl],
    ["SEEDANCE", providers.get("SEEDANCE")?.settings?.baseUrl],
    ["R2", providers.get("R2")?.settings?.endpoint],
    ["META", providers.get("META")?.settings?.graphBaseUrl],
  ]);
  const insecureProviders = [...endpointByProvider].filter(([, endpoint]) => {
    try { return new URL(endpoint).protocol !== "https:"; } catch { return true; }
  }).map(([name]) => name);
  if (insecureProviders.length) throw new Error(`Provider endpoints must use HTTPS: ${insecureProviders.join(", ")}`);

  process.stdout.write(`${JSON.stringify({
    status: "passed",
    mode: "read-only-no-spend",
    api: api.host,
    web: web.host,
    renderer: renderer?.host ?? "verified-by-api-configuration",
    clientId,
    providers: requiredProviders,
    timestamp: new Date().toISOString(),
  })}\n`);
} catch (error) {
  failure = error;
} finally {
  if (cookie && csrfToken) {
    try {
      const logout = await fetch(new URL("/v1/auth/logout", api), {
        method: "POST",
        headers: { accept: "application/json", cookie, "x-csrf-token": csrfToken },
        signal: AbortSignal.timeout(timeoutMs),
      });
      if (logout.status !== 204 && !failure) failure = new Error(`Session cleanup failed with HTTP ${logout.status}`);
    } catch (error) {
      if (!failure) failure = new Error(`Session cleanup failed: ${error instanceof Error ? error.message : "unknown error"}`);
    }
  }
}

if (failure) {
  process.stderr.write(`Live readiness certification failed: ${failure instanceof Error ? failure.message : "unknown error"}\n`);
  process.exitCode = 1;
}
