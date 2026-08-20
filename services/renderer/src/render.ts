import { createHash, randomUUID } from "node:crypto";
import { createReadStream, createWriteStream, existsSync } from "node:fs";
import { mkdir, mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import { basename, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import { GetObjectCommand, HeadObjectCommand, PutObjectCommand, S3Client } from "@aws-sdk/client-s3";
import { bundle } from "@remotion/bundler";
import { renderMedia, selectComposition } from "@remotion/renderer";
import type { ObjectReference, RenderManifest } from "@studio/video-templates";
import QRCode from "qrcode";

type RenderResult = { requestId: string; reused: boolean; outputObjectKey: string; thumbnailObjectKey: string; checksumSha256: string; fileSizeBytes: number; width: number; height: number; fps: number; durationMs: number; codec: string; audioCodec: string | null };

let bundlePromise: Promise<string> | undefined;

function storageClient() {
  const endpoint = process.env.R2_ENDPOINT;
  const accessKeyId = process.env.R2_ACCESS_KEY_ID;
  const secretAccessKey = process.env.R2_SECRET_ACCESS_KEY;
  const bucket = process.env.R2_BUCKET;
  if (!endpoint || !accessKeyId || !secretAccessKey || !bucket) throw new Error("Renderer object storage is not configured");
  return { bucket, client: new S3Client({ region: "auto", endpoint, forcePathStyle: endpoint.includes("localhost") || endpoint.includes("minio") || endpoint.includes("127.0.0.1"), credentials: { accessKeyId, secretAccessKey } }) };
}

async function getBundle() {
  if (!bundlePromise) {
    const compiled = fileURLToPath(new URL("./composition.js", import.meta.url));
    const source = fileURLToPath(new URL("./composition.tsx", import.meta.url));
    bundlePromise = bundle({ entryPoint: existsSync(compiled) ? compiled : source, onProgress: () => undefined });
  }
  return bundlePromise;
}

function uniqueReferences(manifest: RenderManifest): ObjectReference[] {
  const values = [manifest.logo, manifest.music, ...manifest.soundEffects.map((effect) => effect.source), ...manifest.scenes.flatMap((scene) => [scene.source, ...scene.productMedia])].filter((value): value is ObjectReference => value !== null);
  return [...new Map(values.map((value) => [value.objectKey, value])).values()];
}

async function downloadAndVerify(client: S3Client, bucket: string, reference: ObjectReference, destination: string) {
  const output = await client.send(new GetObjectCommand({ Bucket: bucket, Key: reference.objectKey }));
  if (!output.Body) throw new Error("Object storage returned an empty body");
  const hash = createHash("sha256");
  const writer = createWriteStream(destination, { flags: "wx", mode: 0o600 });
  for await (const chunk of output.Body as AsyncIterable<Uint8Array>) {
    hash.update(chunk);
    if (!writer.write(chunk)) await new Promise<void>((resolve) => writer.once("drain", resolve));
  }
  await new Promise<void>((resolve, reject) => writer.end((error?: Error | null) => error ? reject(error) : resolve()));
  if (hash.digest("hex") !== reference.sha256) throw new Error(`Input checksum mismatch for ${reference.objectKey}`);
}

async function serveAssets(files: Map<string, string>) {
  const server = createServer((request, response) => {
    const file = files.get(request.url ?? "");
    if (!file) { response.writeHead(404); response.end(); return; }
    response.writeHead(200, { "cache-control": "no-store" });
    createReadStream(file).pipe(response);
  });
  await new Promise<void>((resolve, reject) => { server.once("error", reject); server.listen(0, "127.0.0.1", resolve); });
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("Asset server failed to bind");
  return { base: `http://127.0.0.1:${address.port}`, close: () => new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve())) };
}

async function command(name: string, args: string[]) {
  return new Promise<string>((resolve, reject) => {
    const child = spawn(name, args, { stdio: ["ignore", "pipe", "pipe"] });
    const stdout: Buffer[] = []; const stderr: Buffer[] = [];
    child.stdout.on("data", (chunk) => stdout.push(Buffer.from(chunk)));
    child.stderr.on("data", (chunk) => stderr.push(Buffer.from(chunk)));
    child.once("error", reject);
    child.once("close", (code) => code === 0 ? resolve(Buffer.concat(stdout).toString("utf8")) : reject(new Error(`${name} failed: ${Buffer.concat(stderr).toString("utf8").slice(0, 500)}`)));
  });
}

async function inspectVideo(filename: string) {
  const raw = await command("ffprobe", ["-v", "error", "-show_entries", "format=duration:stream=codec_type,codec_name,width,height,r_frame_rate", "-of", "json", filename]);
  const document = JSON.parse(raw) as { streams: Array<{ codec_type: string; codec_name: string; width?: number; height?: number; r_frame_rate?: string }>; format: { duration: string } };
  const video = document.streams.find((stream) => stream.codec_type === "video"); const audio = document.streams.find((stream) => stream.codec_type === "audio");
  if (!video?.width || !video.height) throw new Error("Rendered output has no video stream");
  const [numerator = 0, denominator = 1] = (video.r_frame_rate ?? "0/1").split("/").map(Number);
  return { width: video.width, height: video.height, fps: denominator ? numerator / denominator : 0, durationMs: Math.round(Number(document.format.duration) * 1000), codec: video.codec_name, audioCodec: audio?.codec_name ?? null };
}

async function upload(client: S3Client, bucket: string, key: string, file: string, contentType: string, metadata: Record<string, string>) {
  const fileStat = await stat(file);
  await client.send(new PutObjectCommand({ Bucket: bucket, Key: key, Body: createReadStream(file), ContentLength: fileStat.size, ContentType: contentType, Metadata: metadata }));
  return fileStat.size;
}

export async function renderFinalVideo(manifest: RenderManifest): Promise<RenderResult> {
  const requestId = randomUUID();
  const { client, bucket } = storageClient();
  try {
    const head = await client.send(new HeadObjectCommand({ Bucket: bucket, Key: manifest.outputObjectKey }));
    const metadata = head.Metadata;
    if (metadata?.["render-id"] === manifest.renderId && metadata.sha256) {
      return { requestId, reused: true, outputObjectKey: manifest.outputObjectKey, thumbnailObjectKey: manifest.thumbnailObjectKey, checksumSha256: metadata.sha256, fileSizeBytes: head.ContentLength ?? 0, width: 1080, height: 1920, fps: 30, durationMs: manifest.output.durationSeconds * 1000, codec: "h264", audioCodec: metadata["audio-codec"] ?? null };
    }
  } catch { /* output is not present yet */ }
  const tempBase = process.env.RENDER_TEMP_DIR ?? tmpdir(); await mkdir(tempBase, { recursive: true });
  const root = await mkdtemp(join(tempBase, "studio-render-"));
  const inputRoot = join(root, "inputs"); await mkdir(inputRoot, { recursive: true });
  const output = join(root, "final.mp4"); const thumbnail = join(root, "thumbnail.jpg");
  let assetServer: Awaited<ReturnType<typeof serveAssets>> | undefined;
  try {
    const routes = new Map<string, string>(); const assetUrls: Record<string, string> = {};
    for (const [index, reference] of uniqueReferences(manifest).entries()) {
      const destination = join(inputRoot, `${index}-${basename(reference.objectKey)}`);
      await downloadAndVerify(client, bucket, reference, destination);
      const route = `/asset/${index}`; routes.set(route, destination); assetUrls[reference.objectKey] = route;
    }
    assetServer = await serveAssets(routes);
    for (const key of Object.keys(assetUrls)) assetUrls[key] = assetServer.base + assetUrls[key];
    const qrValues = [...new Set(manifest.overlays.filter((overlay) => overlay.type === "qr_code").map((overlay) => overlay.value))];
    const qrCodes = Object.fromEntries(await Promise.all(qrValues.map(async (value) => [value, await QRCode.toDataURL(value, { errorCorrectionLevel: "M", margin: 1, width: 512 })])));
    const serveUrl = await getBundle();
    const inputProps = { manifest, assetUrls, qrCodes };
    const browserExecutable = process.env.REMOTION_BROWSER_EXECUTABLE;
    const composition = await selectComposition({ serveUrl, id: "StudioFinalVideo", inputProps, browserExecutable, logLevel: "warn" });
    await renderMedia({ serveUrl, composition, codec: "h264", audioCodec: "aac", outputLocation: output, inputProps, pixelFormat: "yuv420p", overwrite: true, browserExecutable, concurrency: Number(process.env.RENDER_CONCURRENCY ?? "2"), logLevel: "warn", enforceAudioTrack: true });
    await command("ffmpeg", ["-hide_banner", "-loglevel", "error", "-y", "-ss", "1", "-i", output, "-frames:v", "1", "-vf", "scale=540:960", thumbnail]);
    const inspected = await inspectVideo(output);
    if (inspected.width !== 1080 || inspected.height !== 1920 || Math.abs(inspected.fps - 30) > 0.1 || Math.abs(inspected.durationMs - manifest.output.durationSeconds * 1000) > 750 || !["h264", "avc1"].includes(inspected.codec)) throw new Error("Rendered output failed format validation");
    const bytes = await readFile(output); const checksumSha256 = createHash("sha256").update(bytes).digest("hex");
    const fileSizeBytes = await upload(client, bucket, manifest.outputObjectKey, output, "video/mp4", { "render-id": manifest.renderId, sha256: checksumSha256, "audio-codec": inspected.audioCodec ?? "" });
    await upload(client, bucket, manifest.thumbnailObjectKey, thumbnail, "image/jpeg", { "render-id": manifest.renderId });
    return { requestId, reused: false, outputObjectKey: manifest.outputObjectKey, thumbnailObjectKey: manifest.thumbnailObjectKey, checksumSha256, fileSizeBytes, ...inspected };
  } finally {
    await assetServer?.close().catch(() => undefined);
    await rm(root, { recursive: true, force: true });
  }
}
