import { describe, expect, it } from "vitest";
import { canonicalLegacyRoute, legacyScopedHref, studioRoutes, workspaceDestination } from "./studio-routes";

describe("studio routes", () => {
  it("builds hierarchical client and workspace routes", () => {
    expect(studioRoutes.product("client 1", "workspace/2", "product-3")).toBe(
      "/clients/client%201/workspaces/workspace%2F2/products/product-3",
    );
  });

  it("keeps the logical workspace module when the workspace changes", () => {
    expect(workspaceDestination("/clients/c1/workspaces/w1/products/p1", "c2", "w2")).toBe(
      "/clients/c2/workspaces/w2/products",
    );
    expect(workspaceDestination("/campaigns/legacy", "c2", "w2")).toBe(
      "/clients/c2/workspaces/w2/campaigns",
    );
  });

  it("falls back to a workspace overview and preserves legacy scope links", () => {
    expect(workspaceDestination("/account", "c1", "w1")).toBe("/clients/c1/workspaces/w1");
    expect(legacyScopedHref("/products", { clientId: "c1", workspaceId: "w1" })).toBe(
      "/products?clientId=c1&workspaceId=w1",
    );
  });

  it("canonicalizes legacy scoped frontend routes", () => {
    const scope = { clientId: "c1", workspaceId: "w1" };
    expect(canonicalLegacyRoute("/products/p1", scope)).toBe("/clients/c1/workspaces/w1/products/p1");
    expect(canonicalLegacyRoute("/campaigns/cp1/quality", scope)).toBe("/clients/c1/workspaces/w1/campaigns/cp1/quality");
    expect(canonicalLegacyRoute("/account", scope)).toBeNull();
  });
});
