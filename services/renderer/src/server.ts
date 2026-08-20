import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { renderManifestSchema } from "@studio/video-templates";
import { z } from "zod";
import { verifyManifestSignature } from "./auth.js";
import { renderFinalVideo } from "./render.js";

const renderStorageSchema = z.object({
  endpoint: z.url(),
  accessKeyId: z.string().min(1).max(4096),
  secretAccessKey: z.string().min(1).max(16384),
  bucket: z.string().min(1).max(255),
});
const renderRequestSchema = z.object({ manifest: renderManifestSchema, storage: renderStorageSchema });

const port = Number.parseInt(process.env.RENDERER_PORT ?? process.env.PORT ?? "8090", 10);
const host = process.env.RENDERER_HOST ?? process.env.HOST ?? "0.0.0.0";
const sharedSecret = process.env.RENDERER_INTERNAL_AUTH_SECRET ?? "";

function safeTraceparent(request: IncomingMessage): string | undefined {
  const value = String(request.headers.traceparent ?? "");
  return /^00-[a-f0-9]{32}-[a-f0-9]{16}-[a-f0-9]{2}$/.test(value) ? value : undefined;
}

function json(response: ServerResponse, status: number, body: unknown) {
  response.writeHead(status, { "content-type": status >= 400 ? "application/problem+json" : "application/json", "cache-control": "no-store" });
  response.end(JSON.stringify(body));
}

async function readBody(request: IncomingMessage): Promise<string> {
  const chunks: Buffer[] = [];
  let length = 0;
  for await (const chunk of request) {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    length += buffer.length;
    if (length > 2 * 1024 * 1024) throw new Error("Manifest exceeds 2 MiB");
    chunks.push(buffer);
  }
  return Buffer.concat(chunks).toString("utf8");
}

const server = createServer(async (request, response) => {
  const requestUrl = new URL(request.url ?? "/", `http://${request.headers.host ?? "localhost"}`);
  if (request.method === "GET" && (requestUrl.pathname === "/health/live" || requestUrl.pathname === "/health/ready")) {
    json(response, 200, { status: "ok", timestamp: new Date().toISOString() });
    return;
  }
  if (request.method === "POST" && requestUrl.pathname === "/v1/renders") {
    const startedAt = Date.now();
    const traceparent = safeTraceparent(request);
    try {
      const body = await readBody(request);
      const signature = String(request.headers["x-render-signature"] ?? "");
      if (sharedSecret.length < 32 || !verifyManifestSignature(body, signature, sharedSecret)) {
        json(response, 401, { type: "https://studio.internal/problems/renderer-auth", title: "Renderer authentication failed", status: 401, detail: "Manifest signature is invalid" });
        return;
      }
      const requestBody = renderRequestSchema.parse(JSON.parse(body) as unknown);
      const manifest = requestBody.manifest;
      const result = await renderFinalVideo(manifest, requestBody.storage);
      process.stdout.write(`${JSON.stringify({ level: "info", message: "render completed", traceparent, renderId: manifest.renderId, requestId: result.requestId, reused: result.reused, durationMs: Date.now() - startedAt })}\n`);
      json(response, 200, result);
    } catch (error) {
      const errorType = error instanceof Error ? error.name : "UnknownError";
      process.stderr.write(`${JSON.stringify({ level: "error", message: "render failed", traceparent, errorType, durationMs: Date.now() - startedAt })}\n`);
      json(response, 422, { type: "https://studio.internal/problems/render-failed", title: "Render failed", status: 422, detail: "Renderer could not validate or render the manifest" });
    }
    return;
  }
  json(response, 404, { type: "https://studio.internal/problems/not-found", title: "Not found", status: 404, detail: "Renderer route does not exist" });
});

server.listen(port, host, () => {
  process.stdout.write(`${JSON.stringify({ level: "info", message: "renderer listening", host, port })}\n`);
});

function shutdown() {
  server.close((error) => {
    if (error) {
      process.stderr.write(`${JSON.stringify({ level: "error", message: "renderer shutdown failed", error: error.message })}\n`);
      process.exitCode = 1;
    }
  });
}

process.on("SIGTERM", shutdown);
process.on("SIGINT", shutdown);
