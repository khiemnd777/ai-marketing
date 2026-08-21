import { describe, expect, it } from "vitest";
import { toTravelLuggageData, toTravelLuggageForm, validateTravelLuggageForm } from "./travel-luggage-form";

describe("travel luggage form", () => {
  it("round-trips structured vertical data through user-friendly string controls", () => {
    const source = {
      luggageType: "CHECKED",
      sizeInches: 24,
      externalDimensions: { heightCm: 65, widthCm: 42, depthCm: 27 },
      emptyWeightKg: 3.7,
      capacityLiters: 68,
      shellMaterial: "Polycarbonate",
      wheelType: "Spinner 360°",
      wheelCount: 4,
      lockType: "TSA",
      handleType: "Telescopic",
      interiorCompartments: ["Ngăn lưới", "Dây đai"],
      expandable: true,
      waterResistance: "Chống bắn nước",
      warranty: "5 năm",
      availableColors: ["Đen", "Xanh navy"],
    };

    const form = toTravelLuggageForm(source);
    expect(validateTravelLuggageForm(form)).toBeNull();
    expect(toTravelLuggageData(form)).toEqual(source);
  });

  it("rejects invalid numeric fields and an empty color list", () => {
    const form = toTravelLuggageForm({});
    expect(validateTravelLuggageForm({ ...form, emptyWeightKg: "0" })).toMatch(/số hợp lệ/);
    expect(validateTravelLuggageForm({ ...form, availableColors: "" })).toMatch(/ít nhất một màu/);
  });
});
