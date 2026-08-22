export const CONTENT_VARIANTS_PER_CAMPAIGN = 14;

export type ServicePackage = {
  id: "starter" | "growth" | "scale";
  name: string;
  audience: string;
  monthlyFeeVnd: number;
  onboardingFeeVnd: number;
  recommended: boolean;
  promotionLabel?: string;
  workspaces: number;
  activeProducts: number;
  campaignsPerMonth: number;
  previousCampaignsPerMonth?: number;
  finalVideosPerMonth: number;
  previousFinalVideosPerMonth?: number;
  contentVariantsPerMonth: number;
  videoDurations: readonly (30 | 45)[];
  languages: readonly ("vi" | "en")[];
  metaAds: "NOT_INCLUDED" | "PAUSED_SETUP";
  reviewCadence: string;
  responseTime: string;
  features: readonly string[];
};

export const servicePackages: readonly ServicePackage[] = [
  {
    id: "starter",
    name: "Khởi động",
    audience: "Một sản phẩm chủ lực cần hiện diện đều đặn trên Facebook và Instagram.",
    monthlyFeeVnd: 5_290_000,
    onboardingFeeVnd: 5_290_000,
    recommended: false,
    promotionLabel: "Ưu đãi gấp đôi",
    workspaces: 1,
    activeProducts: 1,
    campaignsPerMonth: 4,
    previousCampaignsPerMonth: 2,
    finalVideosPerMonth: 8,
    previousFinalVideosPerMonth: 4,
    contentVariantsPerMonth: 4 * CONTENT_VARIANTS_PER_CAMPAIGN,
    videoDurations: [30],
    languages: ["vi"],
    metaAds: "NOT_INCLUDED",
    reviewCadence: "Không bao gồm báo cáo định kỳ",
    responseTime: "Phản hồi trong 2 ngày làm việc",
    features: [
      "Kiểm tra thông tin sản phẩm trước khi sản xuất",
      "4 bộ chiến dịch hoàn chỉnh, mỗi bộ gồm 14 nội dung",
      "8 video dọc hoàn chỉnh 30 giây đã qua duyệt",
      "Lên lịch và đăng nội dung lên Facebook, Instagram",
      "Một vòng chỉnh sửa nội dung và hình thức cho mỗi video",
    ],
  },
  {
    id: "growth",
    name: "Tăng trưởng",
    audience: "Thương hiệu cần sản xuất nội dung đều đặn và có hỗ trợ quảng cáo Facebook, Instagram.",
    monthlyFeeVnd: 12_900_000,
    onboardingFeeVnd: 12_900_000,
    recommended: true,
    promotionLabel: "Tăng số lượng ưu đãi",
    workspaces: 1,
    activeProducts: 3,
    campaignsPerMonth: 5,
    previousCampaignsPerMonth: 3,
    finalVideosPerMonth: 15,
    previousFinalVideosPerMonth: 10,
    contentVariantsPerMonth: 5 * CONTENT_VARIANTS_PER_CAMPAIGN,
    videoDurations: [30, 45],
    languages: ["vi", "en"],
    metaAds: "PAUSED_SETUP",
    reviewCadence: "Rà soát hiệu quả 2 lần/tháng",
    responseTime: "Phản hồi trong 1 ngày làm việc",
    features: [
      "Toàn bộ quyền lợi gói Khởi động",
      "5 bộ chiến dịch và 15 video dọc hoàn chỉnh mỗi tháng",
      "Chọn tiếng Việt hoặc tiếng Anh cho từng chiến dịch",
      "Hỗ trợ thiết lập quảng cáo Facebook & Instagram",
      "Phân tích và đề xuất cải thiện 2 lần/tháng",
    ],
  },
  {
    id: "scale",
    name: "Mở rộng",
    audience: "Đội marketing có nhiều dòng sản phẩm, cần nhịp sản xuất cao và ưu tiên vận hành.",
    monthlyFeeVnd: 24_900_000,
    onboardingFeeVnd: 24_900_000,
    recommended: false,
    workspaces: 2,
    activeProducts: 6,
    campaignsPerMonth: 6,
    finalVideosPerMonth: 20,
    contentVariantsPerMonth: 6 * CONTENT_VARIANTS_PER_CAMPAIGN,
    videoDurations: [30, 45],
    languages: ["vi", "en"],
    metaAds: "PAUSED_SETUP",
    reviewCadence: "Rà soát hiệu quả hàng tuần",
    responseTime: "Ưu tiên phản hồi trong 4 giờ làm việc",
    features: [
      "Toàn bộ quyền lợi gói Tăng trưởng",
      "6 bộ chiến dịch và 20 video dọc hoàn chỉnh mỗi tháng",
      "Quản lý tối đa 2 nhóm thương hiệu và 6 sản phẩm",
      "Rà soát hiệu quả, chi phí và đề xuất tối ưu hàng tuần",
      "Ưu tiên lịch duyệt, xuất video và hỗ trợ vận hành",
    ],
  },
] as const;

export const serviceAddOns = [
  { name: "Bộ chiến dịch và video bổ sung", price: "3.900.000đ / bộ", note: "Gồm 14 nội dung và 1 video hoàn chỉnh 30 hoặc 45 giây." },
  { name: "Phiên bản ngôn ngữ thứ hai", price: "1.900.000đ / bộ chiến dịch", note: "Biên tập lại nội dung và video; bản dịch được kiểm tra trước khi bàn giao." },
  { name: "Nhóm thương hiệu bổ sung", price: "3.000.000đ / lần", note: "Thiết lập thông tin sản phẩm, thương hiệu, tư liệu và quy trình duyệt ban đầu." },
  { name: "Vận hành quảng cáo Facebook & Instagram", price: "10% ngân sách quảng cáo", note: "Tối thiểu 5.000.000đ/tháng; tiền chạy quảng cáo thanh toán riêng." },
  { name: "Yêu cầu gấp", price: "+25% hạng mục", note: "Áp dụng khi thời hạn bàn giao dưới 2 ngày làm việc và còn năng lực nhận việc." },
] as const;

export const pricingTerms = [
  "Bảng giá chưa bao gồm VAT và được cập nhật theo từng quý.",
  "Chi phí công cụ, chiến lược, biên tập, kiểm duyệt và xuất video đã nằm trong phí gói.",
  "Ngân sách quảng cáo Facebook & Instagram, quay ngoại cảnh, người mẫu, bản quyền âm nhạc và nội dung bên thứ ba không nằm trong phí gói.",
  "Không tự động phát sinh thêm chi phí hoặc tăng ngân sách. Mọi khoản bổ sung cần khách hàng xác nhận trước.",
  "Số lượng trong mỗi gói là phạm vi dịch vụ dùng để lập báo giá.",
] as const;

export function formatVnd(value: number) {
  return new Intl.NumberFormat("vi-VN").format(value) + "đ";
}
