import { describe, expect, it } from "vitest";
import { renderManifestSchema } from "./manifest";

const object = { objectKey: "workspaces/a/source.mp4", sha256: "a".repeat(64), contentType: "video/mp4" };

describe("renderManifestSchema", () => {
  it("rejects captions outside the approved output duration", () => {
    const result = renderManifestSchema.safeParse({
      renderId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
      manifestVersion: 1,
      workspaceId: "018f47a0-7b60-7e88-b3e7-e48888855073",
      campaignId: "018f47a0-7b61-7349-a334-c4a837951586",
      videoProjectId: "018f47a0-7b62-7e8b-b385-3899d9735865",
      videoProjectVersion: 1,
      videoProjectHash: "b".repeat(64),
      language: "vi",
      output: { width: 1080, height: 1920, fps: 30, durationSeconds: 30, codec: "h264" },
      scenes: [{ sceneId: "018f47a0-7b63-7f0b-b2a1-190eb96d84b9", sceneVersion: 1, source: object, durationMs: 30_000, trimStartMs: 0, trimEndMs: 30_000, muted: false, transition: "cut", productMedia: [] }],
      overlays: [],
      captions: [{ startMs: 29_000, endMs: 31_000, text: "Ngoài thời lượng", speaker: null }],
      burnCaptions: true,
      logo: null,
      music: null,
      soundEffects: [],
      musicGainDb: -18,
      dialogueDuckingDb: -9,
      outputObjectKey: "workspaces/a/output.mp4",
      thumbnailObjectKey: "workspaces/a/thumbnail.jpg",
      createdAt: "2026-08-20T00:00:00Z",
    });
    expect(result.success).toBe(false);
  });
});
