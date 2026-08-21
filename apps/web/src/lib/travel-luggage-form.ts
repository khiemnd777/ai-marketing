export type TravelLuggageForm = {
  luggageType: "CARRY_ON" | "CHECKED" | "SET";
  sizeInches: string;
  heightCm: string;
  widthCm: string;
  depthCm: string;
  emptyWeightKg: string;
  capacityLiters: string;
  shellMaterial: string;
  wheelType: string;
  wheelCount: string;
  lockType: string;
  handleType: string;
  interiorCompartments: string;
  expandable: boolean;
  waterResistance: string;
  warranty: string;
  availableColors: string;
};

const text = (value: unknown, fallback = "") => typeof value === "string" ? value : typeof value === "number" ? String(value) : fallback;
const record = (value: unknown) => value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
const lines = (value: unknown) => Array.isArray(value) ? value.filter((item): item is string => typeof item === "string").join("\n") : "";

export function toTravelLuggageForm(value: Record<string, unknown>): TravelLuggageForm {
  const dimensions = record(value.externalDimensions);
  const luggageType = value.luggageType === "CHECKED" || value.luggageType === "SET" ? value.luggageType : "CARRY_ON";
  return {
    luggageType,
    sizeInches: text(value.sizeInches, "20"),
    heightCm: text(dimensions.heightCm, "55"),
    widthCm: text(dimensions.widthCm, "36"),
    depthCm: text(dimensions.depthCm, "23"),
    emptyWeightKg: text(value.emptyWeightKg, "2.9"),
    capacityLiters: text(value.capacityLiters, "38"),
    shellMaterial: text(value.shellMaterial, "Polycarbonate"),
    wheelType: text(value.wheelType, "Spinner 360°"),
    wheelCount: text(value.wheelCount, "4"),
    lockType: text(value.lockType, "TSA"),
    handleType: text(value.handleType, "Telescopic"),
    interiorCompartments: lines(value.interiorCompartments),
    expandable: value.expandable === true,
    waterResistance: text(value.waterResistance, "Không công bố"),
    warranty: text(value.warranty, "5 năm"),
    availableColors: lines(value.availableColors) || "Đen",
  };
}

export function toTravelLuggageData(form: TravelLuggageForm): Record<string, unknown> {
  const list = (value: string) => value.split(/\n|,/).map((item) => item.trim()).filter(Boolean);
  return {
    luggageType: form.luggageType,
    sizeInches: Number(form.sizeInches),
    externalDimensions: { heightCm: Number(form.heightCm), widthCm: Number(form.widthCm), depthCm: Number(form.depthCm) },
    emptyWeightKg: Number(form.emptyWeightKg),
    capacityLiters: Number(form.capacityLiters),
    shellMaterial: form.shellMaterial.trim(),
    wheelType: form.wheelType.trim(),
    wheelCount: Number(form.wheelCount),
    lockType: form.lockType.trim(),
    handleType: form.handleType.trim(),
    interiorCompartments: list(form.interiorCompartments),
    expandable: form.expandable,
    waterResistance: form.waterResistance.trim(),
    warranty: form.warranty.trim(),
    availableColors: list(form.availableColors),
  };
}

export function validateTravelLuggageForm(form: TravelLuggageForm): string | null {
  const positive = [form.sizeInches, form.heightCm, form.widthCm, form.depthCm, form.emptyWeightKg, form.capacityLiters].every((value) => Number.isFinite(Number(value)) && Number(value) > 0);
  if (!positive || !Number.isInteger(Number(form.wheelCount)) || Number(form.wheelCount) < 0) return "Kích thước, trọng lượng, dung tích và số bánh phải là số hợp lệ.";
  if (![form.shellMaterial, form.wheelType, form.lockType, form.handleType, form.waterResistance, form.warranty].every((value) => value.trim())) return "Các thuộc tính travel-luggage bắt buộc chưa đầy đủ.";
  if (!form.availableColors.split(/\n|,/).some((value) => value.trim())) return "Cần ít nhất một màu đang bán.";
  return null;
}
