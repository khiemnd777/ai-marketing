import { z } from "zod";

export const internalRoleSchema = z.enum(["ADMIN", "OPERATOR", "REVIEWER"]);
export const languageSchema = z.enum(["vi", "en"]);
export const videoFormatSchema = z.enum(["INTERVIEW_REVIEW", "PROBLEM_SOLUTION"]);
export const videoDurationSchema = z.union([z.literal(30), z.literal(45)]);
export const aspectRatioSchema = z.literal("9:16");
export const approvalStateSchema = z.enum([
  "DRAFT",
  "SCRIPT_READY",
  "SCRIPT_APPROVED",
  "SCENES_GENERATING",
  "SCENE_REVIEW",
  "FINAL_RENDERING",
  "FINAL_REVIEW",
  "APPROVED",
  "READY_TO_PUBLISH",
]);

export const generationJobStateSchema = z.enum([
  "DRAFT",
  "READY",
  "QUEUED",
  "SUBMITTING",
  "PROVIDER_QUEUED",
  "PROVIDER_PROCESSING",
  "SUCCEEDED",
  "DOWNLOADING",
  "VALIDATING",
  "REVIEW_REQUIRED",
  "APPROVED",
  "REJECTED",
  "FAILED",
  "CANCELLED",
]);

export const sceneDirectionSchema = z
  .object({
    sceneId: z.string().min(1).max(100),
    order: z.number().int().positive(),
    durationSeconds: z.number().int().min(3).max(15),
    generationMethod: z.enum(["seedance", "product_footage", "still_image"]),
    speakerCharacterId: z.string().uuid().nullable(),
    listenerCharacterId: z.string().uuid().nullable(),
    dialogue: z.string().max(500),
    speakerAction: z.string().max(500),
    listenerAction: z.string().max(500),
    camera: z.string().max(200),
    environment: z.string().max(300),
    productPlacement: z.string().max(300),
    referenceAssetIds: z.array(z.string().uuid()).max(12),
    requiredProductFacts: z.array(z.string().uuid()).max(50),
    expectedCost: z.number().nonnegative(),
  })
  .superRefine((scene, context) => {
    if (scene.generationMethod === "seedance") {
      if (!scene.speakerCharacterId || !scene.listenerCharacterId) {
        context.addIssue({ code: "custom", message: "Seedance scenes require exactly two characters" });
      } else if (scene.speakerCharacterId === scene.listenerCharacterId) {
        context.addIssue({ code: "custom", message: "Speaker and listener must be different" });
      }
    }
  });

export type InternalRole = z.infer<typeof internalRoleSchema>;
export type SceneDirection = z.infer<typeof sceneDirectionSchema>;
