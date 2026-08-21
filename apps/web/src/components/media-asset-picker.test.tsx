import type { components } from "@studio/api-client";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MediaAssetPicker } from "./media-asset-picker";

type Asset = components["schemas"]["MediaAsset"];

const asset: Asset = {
  id: "11111111-1111-4111-8111-111111111111",
  clientId: "22222222-2222-4222-8222-222222222222",
  workspaceId: "33333333-3333-4333-8333-333333333333",
  assetType: "IMAGE",
  category: "HERO_IMAGE",
  name: "Vali hero xanh navy",
  folder: "products",
  status: "APPROVED",
  usageRights: "Client owned",
  tags: ["hero"],
  readyForUse: true,
  version: 2,
  createdAt: "2026-08-21T00:00:00Z",
  updatedAt: "2026-08-21T00:00:00Z",
};

describe("MediaAssetPicker", () => {
  it("shows human-readable asset details and returns the selected id", () => {
    const onChange = vi.fn();
    render(<MediaAssetPicker assets={[asset]} value={[]} onChange={onChange} label="Product Media tham chiếu" />);

    expect(screen.getByText("Vali hero xanh navy")).toBeInTheDocument();
    expect(screen.getByText("HERO_IMAGE")).toBeInTheDocument();
    expect(screen.queryByText(asset.id)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("checkbox", { name: /Vali hero xanh navy/i }));
    expect(onChange).toHaveBeenCalledWith([asset.id]);
  });
});
