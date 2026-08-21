import type { components } from "@studio/api-client";
import { describe, expect, it } from "vitest";
import { isEligibleBrandLogo, makePrimaryLogo } from "./brand-logo-panel";

type Asset = components["schemas"]["MediaAsset"];

const asset: Asset = {
  id: "11111111-1111-4111-8111-111111111111",
  clientId: "22222222-2222-4222-8222-222222222222",
  workspaceId: "33333333-3333-4333-8333-333333333333",
  assetType: "LOGO",
  category: "BRAND_LOGO",
  name: "Logo xanh",
  folder: "brands/logos",
  status: "APPROVED",
  usageRights: "Client owned",
  tags: ["brand-logo"],
  mimeType: "image/png",
  readyForUse: true,
  version: 2,
  createdAt: "2026-08-21T00:00:00Z",
  updatedAt: "2026-08-21T00:00:00Z",
};

describe("brand logo selection", () => {
  it("accepts only approved, processed, unexpired web images", () => {
    expect(isEligibleBrandLogo(asset, Date.parse("2026-08-21T00:00:00Z"))).toBe(true);
    expect(isEligibleBrandLogo({ ...asset, status: "DRAFT" })).toBe(false);
    expect(isEligibleBrandLogo({ ...asset, readyForUse: false })).toBe(false);
    expect(isEligibleBrandLogo({ ...asset, expiresAt: "2026-08-20T00:00:00Z" }, Date.parse("2026-08-21T00:00:00Z"))).toBe(false);
  });

  it("moves the chosen alternate to the primary position without duplicating it", () => {
    expect(makePrimaryLogo(["primary", "alternate", "mono"], "alternate")).toEqual(["alternate", "primary", "mono"]);
  });
});
