import { z } from "zod";

const objectReferenceSchema = z.object({
  objectKey: z.string().min(1).max(1024),
  sha256: z.string().regex(/^[a-f0-9]{64}$/),
  contentType: z.string().min(1).max(150),
});

const overlaySchema = z.object({
  type: z.enum(["logo", "headline", "lower_third", "price", "discount_code", "cta", "website", "phone", "qr_code", "disclaimer"]),
  value: z.string().min(1).max(1000),
  startFrame: z.number().int().nonnegative(),
  endFrame: z.number().int().positive(),
  safeZone: z.enum(["title", "action", "bottom"]),
  sourceFactId: z.string().uuid().nullable(),
});

const captionCueSchema = z.object({
  startMs: z.number().int().nonnegative(),
  endMs: z.number().int().positive(),
  text: z.string().min(1).max(240),
  speaker: z.string().max(120).nullable(),
});

const renderSceneSchema = z.object({
  sceneId: z.string().uuid(),
  sceneVersion: z.number().int().positive(),
  source: objectReferenceSchema,
  durationMs: z.number().int().positive(),
  trimStartMs: z.number().int().nonnegative(),
  trimEndMs: z.number().int().nonnegative(),
  muted: z.boolean(),
  transition: z.enum(["cut", "fade", "slide"]),
  productMedia: z.array(objectReferenceSchema).max(5),
});

export const renderManifestSchema = z
  .object({
    renderId: z.string().uuid(),
    manifestVersion: z.literal(1),
    workspaceId: z.string().uuid(),
    campaignId: z.string().uuid(),
    videoProjectId: z.string().uuid(),
    videoProjectVersion: z.number().int().positive(),
    videoProjectHash: z.string().regex(/^[a-f0-9]{64}$/),
    language: z.enum(["vi", "en"]),
    output: z.object({
      width: z.literal(1080),
      height: z.literal(1920),
      fps: z.literal(30),
      durationSeconds: z.union([z.literal(30), z.literal(45)]),
      codec: z.literal("h264"),
    }),
    scenes: z.array(renderSceneSchema).min(1).max(20),
    overlays: z.array(overlaySchema).max(30),
    captions: z.array(captionCueSchema).max(300),
    burnCaptions: z.boolean(),
    logo: objectReferenceSchema.nullable(),
    music: objectReferenceSchema.nullable(),
    soundEffects: z
      .array(
        z.object({
          source: objectReferenceSchema,
          startMs: z.number().int().nonnegative(),
          gainDb: z.number().min(-60).max(0),
        }),
      )
      .max(10),
    musicGainDb: z.number().min(-60).max(0),
    dialogueDuckingDb: z.number().min(-30).max(0),
    outputObjectKey: z.string().min(1).max(1024),
    thumbnailObjectKey: z.string().min(1).max(1024),
    createdAt: z.iso.datetime({ offset: true }),
  })
  .superRefine((manifest, context) => {
    const finalFrame = manifest.output.durationSeconds * manifest.output.fps;
    for (const [index, overlay] of manifest.overlays.entries()) {
      if (overlay.endFrame <= overlay.startFrame || overlay.endFrame > finalFrame) {
        context.addIssue({ code: "custom", path: ["overlays", index], message: "Overlay frames fall outside the output" });
      }
    }
    for (const [index, cue] of manifest.captions.entries()) {
      if (cue.endMs <= cue.startMs || cue.endMs > manifest.output.durationSeconds * 1000) {
        context.addIssue({ code: "custom", path: ["captions", index], message: "Caption cue falls outside the output" });
      }
    }
    for (const [index, effect] of manifest.soundEffects.entries()) {
      if (effect.startMs >= manifest.output.durationSeconds * 1000) {
        context.addIssue({ code: "custom", path: ["soundEffects", index], message: "Sound effect falls outside the output" });
      }
    }
    const sceneDuration = manifest.scenes.reduce((total, scene) => total + scene.durationMs, 0);
    if (Math.abs(sceneDuration - manifest.output.durationSeconds * 1000) > 1_000) {
      context.addIssue({ code: "custom", path: ["scenes"], message: "Scene durations must match the output duration" });
    }
  });

export type RenderManifest = z.infer<typeof renderManifestSchema>;
export type ObjectReference = z.infer<typeof objectReferenceSchema>;
