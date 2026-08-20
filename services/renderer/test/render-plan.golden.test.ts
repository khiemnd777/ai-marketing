import type { RenderManifest } from "@studio/video-templates";
import { describe, expect, it } from "vitest";
import { buildRenderPlan, captionBottom } from "../src/plan.js";

const source = { objectKey: "workspace/source.mp4", sha256: "a".repeat(64), contentType: "video/mp4" };

const manifest: RenderManifest = {
  renderId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
  manifestVersion: 1,
  workspaceId: "018f47a0-7b60-7e88-b3e7-e48888855073",
  campaignId: "018f47a0-7b61-7349-a334-c4a837951586",
  videoProjectId: "018f47a0-7b62-7e8b-b385-3899d9735865",
  videoProjectVersion: 1,
  videoProjectHash: "b".repeat(64),
  language: "vi",
  output: { width: 1080, height: 1920, fps: 30, durationSeconds: 30, codec: "h264" },
  scenes: [
    { sceneId: "018f47a0-7b63-7f0b-b2a1-190eb96d84b9", sceneVersion: 1, source, durationMs: 12_000, trimStartMs: 500, trimEndMs: 12_500, muted: false, transition: "fade", productMedia: [] },
    { sceneId: "018f47a0-7b64-7f0b-b2a1-190eb96d84b9", sceneVersion: 1, source, durationMs: 18_000, trimStartMs: 0, trimEndMs: 18_000, muted: true, transition: "cut", productMedia: [] },
  ],
  overlays: [
    { type: "headline", value: "Bền bỉ mỗi hành trình", startFrame: 0, endFrame: 120, safeZone: "title", sourceFactId: null },
    { type: "qr_code", value: "https://example.test", startFrame: 600, endFrame: 900, safeZone: "action", sourceFactId: null },
  ],
  captions: [{ startMs: 1_000, endMs: 2_500, text: "Xin chào", speaker: "Host" }],
  burnCaptions: true,
  logo: null,
  music: null,
  soundEffects: [],
  musicGainDb: -18,
  dialogueDuckingDb: -9,
  outputObjectKey: "workspace/output.mp4",
  thumbnailObjectKey: "workspace/thumbnail.jpg",
  createdAt: "2026-08-20T00:00:00Z",
};

describe("renderer plan golden", () => {
  it("keeps sequence, safe-zone, caption, and audio calculations stable", () => {
    const plan = buildRenderPlan(manifest);
    expect({ ...plan, audio: { musicGain: Number(plan.audio.musicGain.toFixed(6)), duckedMusicGain: Number(plan.audio.duckedMusicGain.toFixed(6)) } }).toEqual({
      durationInFrames: 900,
      scenes: [
        { sceneId: manifest.scenes[0]!.sceneId, from: 0, durationInFrames: 360, trimStartFrame: 15, trimEndFrame: 375, transition: "fade" },
        { sceneId: manifest.scenes[1]!.sceneId, from: 360, durationInFrames: 540, trimStartFrame: 0, trimEndFrame: 540, transition: "cut" },
      ],
      overlays: [
        { type: "headline", startFrame: 0, endFrame: 120, bottom: 760, fontSize: 58, prominent: true },
        { type: "qr_code", startFrame: 600, endFrame: 900, bottom: 170, fontSize: 38, prominent: false },
      ],
      captions: [{ text: "Xin chào", startFrame: 30, endFrame: 75 }],
      audio: { musicGain: 0.125893, duckedMusicGain: 0.044668 },
    });
    expect(captionBottom(manifest, 30)).toBe(120);
    expect(captionBottom(manifest, 650)).toBe(800);
  });
});
