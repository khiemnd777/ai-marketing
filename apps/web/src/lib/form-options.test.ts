import { describe, expect, it } from "vitest";
import { countryOptions, includeCurrentOption, mediaCategoryOptions, timeZoneOptions } from "./form-options";

describe("form options", () => {
  it("keeps a legacy value selectable while editing existing data", () => {
    const options = includeCurrentOption([{ value: "seedance", label: "Seedance" }], "legacy-provider");

    expect(options).toContainEqual({ value: "legacy-provider", label: "legacy-provider · Giá trị hiện tại" });
  });

  it("does not duplicate an existing current value", () => {
    const options = includeCurrentOption([{ value: "vi", label: "Tiếng Việt" }], "vi");

    expect(options).toHaveLength(1);
  });

  it("derives media categories from the travel-luggage vertical pack", () => {
    expect(mediaCategoryOptions.map((option) => option.value)).toEqual(expect.arrayContaining(["HERO_IMAGE", "FRONT_VIEW", "WHEEL_CLOSE_UP"]));
  });

  it("provides standards-backed country and timezone choices", () => {
    expect(countryOptions.find((option) => option.value === "VN")?.label).toBeTruthy();
    expect(timeZoneOptions.some((option) => option.value === "UTC")).toBe(true);
  });
});
