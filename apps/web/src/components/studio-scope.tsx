"use client";

import { useParams, usePathname, useSearchParams } from "next/navigation";
import type { StudioScope } from "@/lib/studio-routes";

function stringParam(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] ?? "" : value ?? "";
}

export function useStudioScope(): StudioScope {
  const params = useParams<Record<string, string | string[]>>();
  const pathname = usePathname();
  const query = useSearchParams();
  const pathClientId = pathname.startsWith("/clients/") ? stringParam(params.id) : stringParam(params.clientId);
  const legacyWorkspaceId = pathname.startsWith("/workspaces/") ? stringParam(params.id) : "";

  return {
    clientId: query.get("clientId") ?? pathClientId,
    workspaceId: query.get("workspaceId") ?? (stringParam(params.workspaceId) || legacyWorkspaceId),
  };
}

export function useScopedEntityId(name: "workspaceId" | "brandId" | "productId" | "campaignId") {
  const params = useParams<Record<string, string | string[]>>();
  return stringParam(params[name]) || stringParam(params.id);
}
