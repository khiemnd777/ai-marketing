#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash, createHmac } from "node:crypto";
import { createReadStream } from "node:fs";
import { readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import {
  CreateBucketCommand,
  GetObjectCommand,
  HeadObjectCommand,
  PutObjectCommand,
  S3Client,
} from "@aws-sdk/client-s3";

const sourcePath = path.resolve(process.argv[2] ?? "");
assert(process.argv[2], "Usage: node services/renderer/scripts/smoke-renderer.mjs <30-second-source.mp4>");

const rendererUrl = new URL(process.env.STUDIO_RENDER_SMOKE_URL ?? "http://127.0.0.1:18090");
const storageEndpoint = process.env.STUDIO_RENDER_SMOKE_STORAGE_URL ?? "http://127.0.0.1:19000";
const accessKeyId = process.env.STUDIO_RENDER_SMOKE_ACCESS_KEY ?? "studio";
const secretAccessKey = process.env.STUDIO_RENDER_SMOKE_SECRET_KEY ?? "studio-local-secret";
const bucket = process.env.STUDIO_RENDER_SMOKE_BUCKET ?? "studio-render-smoke";
const sharedSecret = process.env.STUDIO_RENDER_SMOKE_SHARED_SECRET ?? "renderer-smoke-secret-at-least-32-bytes";
const sourceObjectKey = "fixtures/source-30s.mp4";
const outputObjectKey = "outputs/smoke-final.mp4";
const thumbnailObjectKey = "outputs/smoke-thumbnail.jpg";

const storage = new S3Client({
  region: "auto",
  endpoint: storageEndpoint,
  forcePathStyle: true,
  credentials: { accessKeyId, secretAccessKey },
});

async function bodyBytes(body) {
  assert(body, "Object storage returned an empty response body");
  const chunks = [];
  for await (const chunk of body) chunks.push(Buffer.from(chunk));
  return Buffer.concat(chunks);
}

async function render(manifest) {
  const body = JSON.stringify(manifest);
  const signature = createHmac("sha256", sharedSecret).update(body).digest("hex");
  const response = await fetch(new URL("/v1/renders", rendererUrl), {
    method: "POST",
    headers: {
      accept: "application/json",
      "content-type": "application/json",
      "x-render-signature": signature,
    },
    body,
    signal: AbortSignal.timeout(15 * 60 * 1000),
  });
  const responseText = await response.text();
  assert.equal(response.status, 200, `Renderer returned HTTP ${response.status}: ${responseText}`);
  return JSON.parse(responseText);
}

const source = await readFile(sourcePath);
const sourceSha256 = createHash("sha256").update(source).digest("hex");
await storage.send(new CreateBucketCommand({ Bucket: bucket }));
await storage.send(new PutObjectCommand({
  Bucket: bucket,
  Key: sourceObjectKey,
  Body: createReadStream(sourcePath),
  ContentLength: (await stat(sourcePath)).size,
  ContentType: "video/mp4",
}));

const manifest = {
  renderId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
  manifestVersion: 1,
  workspaceId: "018f47a0-7b60-7e88-b3e7-e48888855073",
  campaignId: "018f47a0-7b61-7349-a334-c4a837951586",
  videoProjectId: "018f47a0-7b62-7e8b-b385-3899d9735865",
  videoProjectVersion: 1,
  videoProjectHash: "a".repeat(64),
  language: "vi",
  output: { width: 1080, height: 1920, fps: 30, durationSeconds: 30, codec: "h264" },
  scenes: [{
    sceneId: "018f47a0-7b63-7f0b-b2a1-190eb96d84b9",
    sceneVersion: 1,
    source: { objectKey: sourceObjectKey, sha256: sourceSha256, contentType: "video/mp4" },
    durationMs: 30_000,
    trimStartMs: 0,
    trimEndMs: 30_000,
    muted: false,
    transition: "fade",
    productMedia: [],
  }],
  overlays: [
    { type: "headline", value: "Bền bỉ mỗi hành trình", startFrame: 0, endFrame: 180, safeZone: "title", sourceFactId: null },
    { type: "price", value: "2.490.000 ₫", startFrame: 180, endFrame: 450, safeZone: "action", sourceFactId: "018f47a0-7b65-7f0b-b2a1-190eb96d84b9" },
    { type: "cta", value: "Khám phá ngay", startFrame: 450, endFrame: 720, safeZone: "action", sourceFactId: null },
    { type: "qr_code", value: "https://example.test/luggage", startFrame: 720, endFrame: 900, safeZone: "action", sourceFactId: null },
  ],
  captions: [
    { startMs: 1_000, endMs: 4_000, text: "Chiếc vali đồng hành cùng mọi chuyến đi.", speaker: "Host" },
    { startMs: 12_000, endMs: 15_000, text: "Thông tin sản phẩm luôn được kiểm chứng.", speaker: "Traveler" },
  ],
  burnCaptions: true,
  logo: null,
  music: null,
  soundEffects: [],
  musicGainDb: -18,
  dialogueDuckingDb: -9,
  outputObjectKey,
  thumbnailObjectKey,
  createdAt: "2026-08-20T00:00:00Z",
};

const first = await render(manifest);
assert.equal(first.reused, false);
assert.equal(first.outputObjectKey, outputObjectKey);
assert.equal(first.thumbnailObjectKey, thumbnailObjectKey);
assert.equal(first.width, 1080);
assert.equal(first.height, 1920);
assert.ok(Math.abs(first.fps - 30) <= 0.1, `Unexpected frame rate: ${first.fps}`);
assert.ok(Math.abs(first.durationMs - 30_000) <= 750, `Unexpected duration: ${first.durationMs}`);
assert.ok(["h264", "avc1"].includes(first.codec), `Unexpected codec: ${first.codec}`);
assert.equal(first.audioCodec, "aac");
assert.match(first.checksumSha256, /^[a-f0-9]{64}$/);
assert.ok(first.fileSizeBytes > 0);

const output = await storage.send(new GetObjectCommand({ Bucket: bucket, Key: outputObjectKey }));
const outputBytes = await bodyBytes(output.Body);
assert.equal(createHash("sha256").update(outputBytes).digest("hex"), first.checksumSha256);
await writeFile(path.join(path.dirname(sourcePath), "output.mp4"), outputBytes, { mode: 0o600 });

const thumbnail = await storage.send(new HeadObjectCommand({ Bucket: bucket, Key: thumbnailObjectKey }));
assert.ok((thumbnail.ContentLength ?? 0) > 0, "Renderer did not persist a thumbnail");

const second = await render(manifest);
assert.equal(second.reused, true);
assert.equal(second.checksumSha256, first.checksumSha256);
assert.equal(second.outputObjectKey, outputObjectKey);

process.stdout.write(`${JSON.stringify({
  status: "passed",
  mode: "local-no-cost",
  width: first.width,
  height: first.height,
  fps: first.fps,
  durationMs: first.durationMs,
  codec: first.codec,
  audioCodec: first.audioCodec,
  checksumVerified: true,
  thumbnailVerified: true,
  reuseVerified: true,
})}\n`);
