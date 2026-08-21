import travelLuggageAssetRequirements from "../../../../verticals/travel-luggage/asset-requirements.json";

export type FormOption<T extends string = string> = Readonly<{ value: T; label: string }>;

export function includeCurrentOption<T extends string>(options: readonly FormOption<T>[], current: T | null | undefined): FormOption<T>[] {
  if (!current || options.some((option) => option.value === current)) return [...options];
  return [...options, { value: current, label: `${current} · Giá trị hiện tại` }];
}

export const languageOptions = [
  { value: "vi", label: "Tiếng Việt" },
  { value: "en", label: "English" },
] as const;

export const characterProviderOptions = [
  { value: "seedance", label: "Seedance / ModelArk" },
  { value: "internal", label: "Nguồn nội bộ" },
] as const;

export const genderPresentationOptions = [
  { value: "", label: "Không xác định" },
  { value: "Nam tính", label: "Nam tính" },
  { value: "Nữ tính", label: "Nữ tính" },
  { value: "Trung tính", label: "Trung tính" },
  { value: "Linh hoạt", label: "Linh hoạt / khác" },
] as const;

export const ageRangeOptions = ["18-24", "25-35", "36-45", "46-55", "56-65", "65+"].map((value) => ({ value, label: value }));
export const gestureStyleOptions = ["Tự nhiên", "Chuyên nghiệp", "Thân thiện", "Năng động", "Điềm tĩnh"].map((value) => ({ value, label: value }));
export const characterRoleOptions = ["Người giới thiệu", "Người đánh giá", "Người phỏng vấn", "Khách du lịch", "Người lắng nghe"].map((value) => ({ value, label: value }));

export const industryOptions = [
  { value: "Du lịch & hành lý", label: "Du lịch & hành lý" },
  { value: "Du lịch", label: "Du lịch" },
  { value: "Software", label: "Phần mềm" },
  { value: "Bán lẻ", label: "Bán lẻ" },
  { value: "Thương mại điện tử", label: "Thương mại điện tử" },
] as const;

export const marketOptions = [
  { value: "Việt Nam", label: "Việt Nam" },
  { value: "Đông Nam Á", label: "Đông Nam Á" },
  { value: "Hoa Kỳ", label: "Hoa Kỳ" },
  { value: "Toàn cầu", label: "Toàn cầu" },
] as const;

const isoRegionCodes = "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW".split(" ");
const regionNames = typeof Intl.DisplayNames === "function" ? new Intl.DisplayNames(["vi"], { type: "region" }) : null;
export const countryOptions: FormOption[] = isoRegionCodes
  .map((value) => ({ value, label: regionNames?.of(value) ?? value }))
  .sort((left, right) => left.label.localeCompare(right.label, "vi"));

type SupportedValuesIntl = typeof Intl & { supportedValuesOf?: (key: "currency" | "timeZone") => string[] };
const supportedValues = Intl as SupportedValuesIntl;
const currencyNames = typeof Intl.DisplayNames === "function" ? new Intl.DisplayNames(["vi"], { type: "currency" }) : null;
export const currencyOptions: FormOption[] = (supportedValues.supportedValuesOf?.("currency") ?? ["VND", "USD"])
  .map((value) => ({ value, label: `${value} · ${currencyNames?.of(value) ?? value}` }))
  .sort((left, right) => left.value.localeCompare(right.value));

export const timeZoneOptions: FormOption[] = ["UTC", ...(supportedValues.supportedValuesOf?.("timeZone") ?? ["Asia/Ho_Chi_Minh"])]
  .filter((value, index, values) => values.indexOf(value) === index)
  .map((value) => ({ value, label: value }));

export const mediaCategoryOptions: FormOption[] = travelLuggageAssetRequirements.categories.map((value) => ({
  value,
  label: value.replaceAll("_", " "),
}));

export const luggageMaterialOptions = ["Polycarbonate", "Polypropylene", "ABS", "Aluminium", "Vải polyester", "Vải nylon"].map((value) => ({ value, label: value }));
export const luggageWheelTypeOptions = ["Spinner 360°", "Spinner", "Inline skate", "Bánh đôi", "Bánh đơn"].map((value) => ({ value, label: value }));
export const luggageWheelCountOptions = [0, 2, 4, 8].map((value) => ({ value: String(value), label: String(value) }));
export const luggageLockTypeOptions = ["TSA", "Khóa số", "Khóa chìa", "Không có khóa"].map((value) => ({ value, label: value }));
export const luggageHandleTypeOptions = ["Telescopic", "Tay kéo đơn", "Tay kéo đôi", "Tay xách"].map((value) => ({ value, label: value }));
export const luggageWaterResistanceOptions = ["Không công bố", "Chống bắn nước", "Chống nước nhẹ", "Chống nước"].map((value) => ({ value, label: value }));
export const luggageWarrantyOptions = ["Không bảo hành", "1 năm", "2 năm", "3 năm", "5 năm", "Bảo hành trọn đời"].map((value) => ({ value, label: value }));

export const productFactKeyOptions = [
  "external_dimensions", "weight", "capacity", "shell_material", "wheel_type", "wheel_count", "lock_type", "handle_type", "warranty", "available_colors", "price", "discount_code", "sku", "model", "product_name",
].map((value) => ({ value, label: value.replaceAll("_", " ") }));

export const cameraOptions = ["Cận cảnh", "Trung cảnh", "Toàn cảnh", "Tracking shot", "Static shot", "Dolly in", "Dolly out"].map((value) => ({ value, label: value }));
export const environmentOptions = ["Studio tối giản", "Sân bay", "Khách sạn", "Đường phố", "Phòng khách", "Cửa hàng"].map((value) => ({ value, label: value }));
export const productPlacementOptions = ["Cầm trên tay", "Đặt cạnh nhân vật", "Chính giữa khung hình", "Cận cảnh chi tiết", "Trong bối cảnh sử dụng"].map((value) => ({ value, label: value }));

export const providerSettingSuggestions: Record<string, readonly string[]> = {
  model: ["gpt-5.6-luna", "dreamina-seedance-2-0-260128"],
  transcriptionModel: ["gpt-4o-mini-transcribe"],
  apiVersion: ["v3"],
};
