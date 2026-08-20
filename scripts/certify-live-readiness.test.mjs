import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { createServer } from "node:http";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const certificationScript = path.join(repositoryRoot, "scripts", "certify-live-readiness.mjs");
const certificationClientId = "018f47a0-7b5f-7d5f-9d2a-c5939813086f";

function sendJSON(response, status, body, headers = {}) {
  response.writeHead(status, { "content-type": "application/json", ...headers });
  response.end(JSON.stringify(body));
}

async function listen(handler) {
  const server = createServer(handler);
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const address = server.address();
  assert(address && typeof address !== "string");
  return { server, origin: `http://127.0.0.1:${address.port}` };
}

async function close(server) {
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
}

async function fixture({ demoMode = false } = {}) {
  const state = { loginBody: "", logoutHeaders: undefined };
  const providers = [
    ["OPENAI", { baseUrl: "https://api.openai.com" }],
    ["SEEDANCE", { baseUrl: "https://ark.cn-beijing.volces.com" }],
    ["R2", { endpoint: "https://account.r2.cloudflarestorage.com" }],
    ["META", { graphBaseUrl: "https://graph.facebook.com" }],
    ["RENDERER", { baseUrl: "http://renderer:8090" }],
  ].map(([provider, settings]) => ({ provider, settings, configured: true }));

  const api = await listen(async (request, response) => {
    if (request.url === "/v1/health/ready") {
      sendJSON(response, 200, { status: "ok", checks: { database: "ok" } });
      return;
    }
    if (request.url === "/v1/auth/login" && request.method === "POST") {
      const chunks = [];
      for await (const chunk of request) chunks.push(chunk);
      state.loginBody = Buffer.concat(chunks).toString("utf8");
      sendJSON(response, 200, { role: "ADMIN", requiresPasswordChange: false, csrfToken: "certification-csrf" }, {
        "set-cookie": "studio_session=certification-session; HttpOnly; SameSite=Strict",
      });
      return;
    }
    if (request.url === `/v1/clients/${certificationClientId}/provider-configuration`) {
      sendJSON(response, 200, { clientId: certificationClientId, demoMode, providers });
      return;
    }
    if (request.url === "/v1/auth/logout" && request.method === "POST") {
      state.logoutHeaders = request.headers;
      response.writeHead(204);
      response.end();
      return;
    }
    response.writeHead(404);
    response.end();
  });
  const web = await listen((request, response) => {
    if (request.url === "/login") {
      response.writeHead(200, { "content-type": "text/html; charset=utf-8" });
      response.end("<!doctype html><title>Staging login</title>");
      return;
    }
    response.writeHead(404);
    response.end();
  });
  const renderer = await listen((request, response) => {
    if (request.url === "/health/ready") {
      sendJSON(response, 200, { status: "ok" });
      return;
    }
    response.writeHead(404);
    response.end();
  });

  return {
    api,
    web,
    renderer,
    state,
    async close() {
      await Promise.all([close(api.server), close(web.server), close(renderer.server)]);
    },
  };
}

async function runCertification(targets) {
  const email = "certification-admin@example.com";
  const password = "certification-password-that-must-not-leak";
  const child = spawn(process.execPath, [certificationScript], {
    cwd: repositoryRoot,
    env: {
      ...process.env,
      STUDIO_CERT_API_URL: targets.api.origin,
      STUDIO_CERT_WEB_URL: targets.web.origin,
      STUDIO_CERT_RENDERER_URL: targets.renderer.origin,
      STUDIO_CERT_ADMIN_EMAIL: email,
      STUDIO_CERT_ADMIN_PASSWORD: password,
      STUDIO_CERT_CLIENT_ID: certificationClientId,
      STUDIO_CERT_ALLOW_INSECURE_LOCAL: "true",
      STUDIO_CERT_TIMEOUT_MS: "2000",
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let stdout = "";
  let stderr = "";
  child.stdout.setEncoding("utf8");
  child.stderr.setEncoding("utf8");
  child.stdout.on("data", (chunk) => { stdout += chunk; });
  child.stderr.on("data", (chunk) => { stderr += chunk; });
  const [code, signal] = await once(child, "exit");
  return { code, signal, stdout, stderr, email, password };
}

test("certification passes without spending and always revokes its Admin session", async () => {
  const targets = await fixture();
  try {
    const result = await runCertification(targets);
    assert.equal(result.code, 0, result.stderr);
    assert.equal(result.signal, null);
    const report = JSON.parse(result.stdout.trim());
    assert.equal(report.status, "passed");
    assert.equal(report.mode, "read-only-no-spend");
    assert.deepEqual(report.providers, ["OPENAI", "SEEDANCE", "R2", "META", "RENDERER"]);
    assert.equal(report.clientId, certificationClientId);
    assert.deepEqual(JSON.parse(targets.state.loginBody), { email: result.email, password: result.password });
    assert.match(targets.state.logoutHeaders?.cookie ?? "", /studio_session=certification-session/);
    assert.equal(targets.state.logoutHeaders?.["x-csrf-token"], "certification-csrf");
    assert.equal(`${result.stdout}${result.stderr}`.includes(result.email), false);
    assert.equal(`${result.stdout}${result.stderr}`.includes(result.password), false);
  } finally {
    await targets.close();
  }
});

test("certification fails closed in demo mode and still revokes its session", async () => {
  const targets = await fixture({ demoMode: true });
  try {
    const result = await runCertification(targets);
    assert.equal(result.code, 1);
    assert.match(result.stderr, /client provider profile.*LIVE mode/);
    assert.match(targets.state.logoutHeaders?.cookie ?? "", /studio_session=certification-session/);
    assert.equal(`${result.stdout}${result.stderr}`.includes(result.email), false);
    assert.equal(`${result.stdout}${result.stderr}`.includes(result.password), false);
  } finally {
    await targets.close();
  }
});
