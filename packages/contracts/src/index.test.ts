import { describe, expect, it } from "vitest";
import { sceneDirectionSchema } from "./index";

const validScene = {
  sceneId: "scene-01",
  order: 1,
  durationSeconds: 8,
  generationMethod: "seedance",
  speakerCharacterId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
  listenerCharacterId: "018f47a0-7b60-7e88-b3e7-e48888855073",
  dialogue: "Điều gì khiến anh chọn chiếc vali này?",
  speakerAction: "turns and asks naturally",
  listenerAction: "keeps mouth closed and gives one subtle nod",
  camera: "medium two-shot",
  environment: "modern airport lounge",
  productPlacement: "suitcase beside the traveler",
  referenceAssetIds: [],
  requiredProductFacts: [],
  expectedCost: 0.8,
} as const;

describe("sceneDirectionSchema", () => {
  it("accepts one speaker and one distinct listener", () => {
    expect(sceneDirectionSchema.safeParse(validScene).success).toBe(true);
  });

  it("rejects a duplicate character", () => {
    const result = sceneDirectionSchema.safeParse({
      ...validScene,
      listenerCharacterId: validScene.speakerCharacterId,
    });
    expect(result.success).toBe(false);
  });
});
