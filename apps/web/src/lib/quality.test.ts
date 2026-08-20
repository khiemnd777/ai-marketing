import { describe, expect, it } from "vitest";
import { qualityNeedsAction } from "./quality";

describe("qualityNeedsAction", () => {
  it.each([
    ["REVIEW_REQUIRED", null],
    ["FAILED", null],
    ["APPROVED", { status: "FAILED", findings: [] }],
    ["APPROVED", { status: "PASSED", findings: ["subtitle overflow"] }],
  ])("routes %s generations with relevant QC to the action queue", (status, qualityCheck) => {
    expect(qualityNeedsAction({ status, qualityCheck })).toBe(true);
  });

  it("keeps a clean approved take out of the action queue", () => {
    expect(qualityNeedsAction({ status: "APPROVED", qualityCheck: { status: "PASSED", findings: [] } })).toBe(false);
  });
});
