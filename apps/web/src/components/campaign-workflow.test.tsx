import { cleanup, render, screen } from "@testing-library/react";
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
    expect(screen.getAllByRole("listitem")).toHaveLength(9);
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
  });

  it("marks Meta Ads as optional while keeping it navigable", () => {
    render(<CampaignTabs {...scope} active="/publishing" />);

    const adsStep = screen.getByRole("link", { name: "Meta Ads Tùy chọn" });
    expect(adsStep).toHaveTextContent("Tùy chọn");
    expect(adsStep).toHaveAttribute("href", "/clients/client-1/workspaces/workspace-1/campaigns/campaign-1/ads");
  });
});
