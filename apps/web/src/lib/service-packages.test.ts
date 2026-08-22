import { describe, expect, it } from "vitest";
import { CONTENT_VARIANTS_PER_CAMPAIGN, formatVnd, servicePackages } from "./service-packages";

describe("service packages", () => {
  it("keeps a single recommended middle package with increasing capacity and fees", () => {
    expect(servicePackages).toHaveLength(3);
    expect(servicePackages.filter((item) => item.recommended).map((item) => item.id)).toEqual(["growth"]);

    for (let index = 1; index < servicePackages.length; index += 1) {
      expect(servicePackages[index]!.monthlyFeeVnd).toBeGreaterThan(servicePackages[index - 1]!.monthlyFeeVnd);
      expect(servicePackages[index]!.campaignsPerMonth).toBeGreaterThan(servicePackages[index - 1]!.campaignsPerMonth);
      expect(servicePackages[index]!.finalVideosPerMonth).toBeGreaterThan(servicePackages[index - 1]!.finalVideosPerMonth);
    }
  });

  it("aligns package promises with the complete Phase 1 formats", () => {
    for (const item of servicePackages) {
      expect(item.onboardingFeeVnd).toBeLessThan(item.monthlyFeeVnd);
      expect(item.contentVariantsPerMonth).toBe(item.campaignsPerMonth * CONTENT_VARIANTS_PER_CAMPAIGN);
      expect(item.finalVideosPerMonth).toBeGreaterThanOrEqual(item.campaignsPerMonth);
      expect(item.videoDurations.every((duration) => duration === 30 || duration === 45)).toBe(true);
      expect(item.languages.every((language) => language === "vi" || language === "en")).toBe(true);
      expect(item.features.join(" ")).not.toMatch(/TikTok|YouTube|Zalo/i);
    }
  });

  it("keeps onboarding fees separate from monthly retainers and applies the requested offers", () => {
    const [starter, growth, scale] = servicePackages;

    expect([starter!.previousOnboardingFeeVnd, starter!.onboardingFeeVnd]).toEqual([2_900_000, 0]);
    expect([growth!.previousOnboardingFeeVnd, growth!.onboardingFeeVnd]).toEqual([5_900_000, 2_950_000]);
    expect(growth!.onboardingFeeVnd).toBe(growth!.previousOnboardingFeeVnd! * 0.5);
    expect(growth!.onboardingDiscountLabel).toBe("-50%");
    expect(scale!.onboardingFeeVnd).toBe(7_900_000);
    expect(scale!.previousOnboardingFeeVnd).toBeUndefined();
  });

  it("applies the requested promotional quotas without changing the scale package", () => {
    const starter = servicePackages[0]!;
    const growth = servicePackages[1]!;
    const scale = servicePackages[2]!;

    expect(starter.monthlyFeeVnd).toBe(5_290_000);
    expect([starter.previousCampaignsPerMonth, starter.campaignsPerMonth]).toEqual([2, 4]);
    expect([starter.previousFinalVideosPerMonth, starter.finalVideosPerMonth]).toEqual([4, 8]);
    expect(starter.videoDurations).toEqual([30]);
    expect(starter.metaAds).toBe("NOT_INCLUDED");
    expect([starter.reviewCadence, ...starter.features].join(" ")).not.toMatch(/báo cáo (hiệu quả )?hàng tháng/i);

    expect([growth.previousCampaignsPerMonth, growth.campaignsPerMonth]).toEqual([3, 5]);
    expect([growth.previousFinalVideosPerMonth, growth.finalVideosPerMonth]).toEqual([10, 15]);
    expect(growth.metaAds).toBe("PAUSED_SETUP");

    expect([scale.campaignsPerMonth, scale.finalVideosPerMonth]).toEqual([6, 20]);
    expect(scale.previousCampaignsPerMonth).toBeUndefined();
    expect(scale.previousFinalVideosPerMonth).toBeUndefined();
    expect(scale.metaAds).toBe("PAUSED_SETUP");
  });

  it("formats prices as Vietnamese dong without implying fractional currency", () => {
    expect(formatVnd(5_290_000)).toBe("5.290.000đ");
  });
});
