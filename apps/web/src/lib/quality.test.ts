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

  it("keeps an approved take actionable until it is selected for Composer", () => {
    expect(qualityNeedsAction({ status: "APPROVED", selected: false, qualityCheck: { status: "PASSED", findings: [] } })).toBe(true);
    expect(qualityNeedsAction({ status: "APPROVED", selected: true, qualityCheck: { status: "PASSED", findings: [] } })).toBe(false);
  });
});
