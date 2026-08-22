export type StudioScope = {
  clientId: string;
  workspaceId: string;
};

const encode = (value: string) => encodeURIComponent(value);

export const studioRoutes = {
  pricing: "/pricing",
  clients: "/clients",
  client: (clientId: string) => `/clients/${encode(clientId)}`,
  clientProfile: (clientId: string) => `/clients/${encode(clientId)}/profile`,
  clientWorkspaces: (clientId: string) => `/clients/${encode(clientId)}/workspaces`,
  clientProviders: (clientId: string) => `/clients/${encode(clientId)}/providers`,
  workspace: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}`,
  brands: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/brands`,
  brand: (clientId: string, workspaceId: string, brandId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/brands/${encode(brandId)}`,
  products: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/products`,
  product: (clientId: string, workspaceId: string, productId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/products/${encode(productId)}`,
  media: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/media`,
  characters: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/characters`,
  campaigns: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/campaigns`,
  campaign: (clientId: string, workspaceId: string, campaignId: string, suffix = "") => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/campaigns/${encode(campaignId)}${suffix}`,
  meta: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/meta`,
  analytics: (clientId: string, workspaceId: string) => `/clients/${encode(clientId)}/workspaces/${encode(workspaceId)}/analytics`,
} as const;

export function workspaceDestination(pathname: string, clientId: string, workspaceId: string) {
  if (pathname.includes("/products")) return studioRoutes.products(clientId, workspaceId);
  if (pathname.includes("/media")) return studioRoutes.media(clientId, workspaceId);
  if (pathname.includes("/characters")) return studioRoutes.characters(clientId, workspaceId);
  if (pathname.includes("/campaigns")) return studioRoutes.campaigns(clientId, workspaceId);
  if (pathname.includes("/meta")) return studioRoutes.meta(clientId, workspaceId);
  if (pathname.includes("/analytics")) return studioRoutes.analytics(clientId, workspaceId);
  if (pathname.includes("/brands")) return studioRoutes.brands(clientId, workspaceId);
  return studioRoutes.workspace(clientId, workspaceId);
}

export function legacyScopedHref(pathname: string, scope: StudioScope) {
  const query = new URLSearchParams();
  if (scope.clientId) query.set("clientId", scope.clientId);
  if (scope.workspaceId) query.set("workspaceId", scope.workspaceId);
  const serialized = query.toString();
  return serialized ? `${pathname}?${serialized}` : pathname;
}

export function canonicalLegacyRoute(pathname: string, scope: StudioScope) {
  if (!scope.clientId) return null;
  if (pathname === "/settings/providers") return studioRoutes.clientProviders(scope.clientId);
  if (!scope.workspaceId) return null;

  if (/^\/workspaces\/[^/]+$/.test(pathname)) return studioRoutes.workspace(scope.clientId, scope.workspaceId);
  if (pathname === "/products") return studioRoutes.products(scope.clientId, scope.workspaceId);
  const product = pathname.match(/^\/products\/([^/]+)$/);
  if (product) return studioRoutes.product(scope.clientId, scope.workspaceId, product[1]!);
  if (pathname === "/media") return studioRoutes.media(scope.clientId, scope.workspaceId);
  if (pathname === "/settings/characters") return studioRoutes.characters(scope.clientId, scope.workspaceId);
  if (pathname === "/settings/meta") return studioRoutes.meta(scope.clientId, scope.workspaceId);
  if (pathname === "/analytics") return studioRoutes.analytics(scope.clientId, scope.workspaceId);
  const brand = pathname.match(/^\/brands\/([^/]+)$/);
  if (brand) return studioRoutes.brand(scope.clientId, scope.workspaceId, brand[1]!);
  if (pathname === "/campaigns") return studioRoutes.campaigns(scope.clientId, scope.workspaceId);
  const campaign = pathname.match(/^\/campaigns\/([^/]+)(\/.*)?$/);
  if (campaign) return studioRoutes.campaign(scope.clientId, scope.workspaceId, campaign[1]!, campaign[2] ?? "");
  return null;
}
