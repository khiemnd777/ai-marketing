import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { CampaignTabs } from "./campaign-workflow";

const scope = {
  clientId: "client-1",
  workspaceId: "workspace-1",
  campaignId: "campaign-1",
};

afterEach(cleanup);

describe("CampaignTabs", () => {
  it("does not check prior route steps unless persisted progress says they are complete", () => {
    render(<CampaignTabs {...scope} active="/concepts" />);

    expect(screen.getByRole("navigation", { name: "Tiến trình campaign" })).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(8);
    expect(screen.getByRole("link", { name: /Concept/ })).toHaveAttribute("aria-current", "step");
    expect(screen.getByRole("link", { name: /Brief/ })).toHaveAccessibleName("Brief Bước trước");
    expect(screen.getByRole("link", { name: /Brief/ })).toHaveAttribute("data-completed", "false");
    expect(screen.getByRole("link", { name: /Nội dung/ })).not.toHaveAttribute("aria-current");
  });

  it("checks only steps confirmed complete by persisted progress", () => {
    render(<CampaignTabs {...scope} active="/scenes" completedSteps={new Set(["BRIEF", "CONCEPT"])} />);

    expect(screen.getByRole("link", { name: /Brief/ })).toHaveAttribute("data-completed", "true");
    expect(screen.getByRole("link", { name: /Concept/ })).toHaveAttribute("data-completed", "true");
    expect(screen.getByRole("link", { name: /Nội dung/ })).toHaveAttribute("data-completed", "false");
    expect(screen.getByRole("link", { name: /Kịch bản/ })).toHaveAttribute("data-completed", "false");
    expect(screen.getByRole("link", { name: /Cảnh quay/ })).toHaveAttribute("data-completed", "false");
    expect(screen.getByRole("link", { name: /Duyệt take/ })).toHaveAttribute("data-completed", "false");
    expect(screen.getByRole("link", { name: /Dựng & duyệt final/ })).toHaveAttribute("data-completed", "false");
  });

  it("presents publishing and optional Ads as sibling distribution channels", () => {
    render(<CampaignTabs {...scope} active="/publishing" />);

    const distribution = screen.getByRole("group", { name: "Kênh phân phối" });
    const publishingStep = within(distribution).getByRole("link", { name: /Xuất bản/ });
    const adsStep = within(distribution).getByRole("link", { name: /Meta Ads Tùy chọn/ });
    expect(screen.getByText("Bước 8 · Phân phối")).toBeInTheDocument();
    expect(publishingStep).toHaveAttribute("aria-current", "step");
    expect(publishingStep).toHaveAttribute("href", "/clients/client-1/workspaces/workspace-1/campaigns/campaign-1/publishing");
    expect(adsStep).toHaveTextContent("Tùy chọn");
    expect(adsStep).toHaveAttribute("href", "/clients/client-1/workspaces/workspace-1/campaigns/campaign-1/ads");
  });
});
