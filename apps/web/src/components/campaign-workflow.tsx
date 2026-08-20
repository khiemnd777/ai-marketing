"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, CircleDollarSign, RefreshCw } from "lucide-react";
import Link from "next/link";
import { useEffect, type ReactNode } from "react";
import { usePermissions } from "@/components/auth-context";
import { useScopedEntityId, useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, PageHeader, StatePanel } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";
import { cn } from "@/lib/cn";
import { studioRoutes } from "@/lib/studio-routes";

export type GenerationOperation = components["schemas"]["GenerationOperation"];

export function useCampaignRoute() {
  const { clientId, workspaceId } = useStudioScope();
  return { campaignId: useScopedEntityId("campaignId"), clientId, workspaceId };
}

const tabs = [
  ["", "Brief"],
  ["/concepts", "Concept"],
  ["/content", "Nội dung"],
  ["/script", "Kịch bản"],
  ["/scenes", "Cảnh quay"],
  ["/quality", "Quality"],
  ["/composer", "Composer"],
  ["/publishing", "Xuất bản"],
  ["/ads", "Meta Ads"],
] as const;

export function CampaignTabs({ campaignId, clientId, workspaceId, active }: { campaignId: string; clientId: string; workspaceId: string; active: string }) {
  return <nav className="mb-7 flex gap-2 overflow-x-auto pb-1" aria-label="Campaign workflow">{tabs.map(([suffix, label]) => <Link key={suffix} href={studioRoutes.campaign(clientId, workspaceId, campaignId, suffix)} className={cn("whitespace-nowrap rounded-full px-4 py-2 text-sm font-bold", active === suffix ? "bg-[var(--lime)] text-[var(--ink)]" : "bg-white text-[var(--muted)] ring-1 ring-[var(--line)] hover:text-[var(--ink)]")}>{label}</Link>)}</nav>;
}

export function CampaignHeader({ active, title, description, children }: { active: string; title: string; description: string; children?: ReactNode }) {
  const scope = useCampaignRoute();
  const campaign = useQuery({
    queryKey: ["campaign", scope.clientId, scope.workspaceId, scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId && !!scope.campaignId,
    queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải campaign."); return data; },
  });
  if (!scope.clientId || !scope.workspaceId) return <StatePanel title="Chưa chọn workspace">Mở campaign từ danh sách trong một workspace để giữ đúng phạm vi dữ liệu.</StatePanel>;
  return <><PageHeader eyebrow={campaign.data ? `${campaign.data.name} · ${campaign.data.status}` : "Campaign Builder"} title={title} description={description} />
    <CampaignTabs {...scope} active={active} />
    {campaign.error ? <StatePanel title="Không thể tải campaign" tone="danger">{campaign.error.message}</StatePanel> : children}
  </>;
}

export function GenerationPanel({ operation }: { operation: GenerationOperation }) {
  const scope = useCampaignRoute();
  const { canOperate } = usePermissions();
  const queryClient = useQueryClient();
  const jobs = useQuery({
    queryKey: ["generation-jobs", scope.clientId, scope.workspaceId, scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    refetchInterval: (query) => (query.state.data?.items.some((job) => job.status === "QUEUED" || job.status === "RUNNING") ? 1500 : false),
    queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/generation-jobs", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải job AI."); return data; },
  });
  const estimate = useQuery({
    queryKey: ["cost-estimate", scope.clientId, scope.workspaceId, scope.campaignId, operation],
    enabled: !!scope.clientId && !!scope.workspaceId,
    staleTime: 20 * 60 * 1000,
    queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/cost-estimate", { params: { path: scope, query: { operation } } }); if (error || !data) throw apiError(error, "Không thể ước tính chi phí."); return data; },
  });
  const start = useMutation({
    mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/generation-jobs", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } }, body: { operation } }); if (error || !data) throw apiError(error, "Không thể bắt đầu job AI."); return data; },
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["generation-jobs", scope.clientId, scope.workspaceId, scope.campaignId] }); },
  });
  const latest = jobs.data?.items.find((job) => job.operation === operation);
  useEffect(() => {
    if (latest?.status !== "SUCCEEDED") return;
    const resultKey = {
      CONCEPTS: "concepts",
      CONTENT: "campaign-content",
      SCRIPT: "campaign-script",
      SCENES: "campaign-scenes",
    }[operation];
    void queryClient.invalidateQueries({ queryKey: [resultKey, scope.campaignId] });
  }, [latest?.id, latest?.status, operation, queryClient, scope.campaignId]);
  const running = latest?.status === "QUEUED" || latest?.status === "RUNNING";
  return <Card className="mb-6 flex flex-col gap-4 p-5 md:flex-row md:items-center">
    <span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><Bot className="size-5 text-[var(--moss)]" /></span>
    <div className="flex-1"><div className="flex flex-wrap items-center gap-2"><h2 className="font-serif text-lg font-bold">AI {operation.toLowerCase()}</h2>{latest ? <Badge tone={latest.status === "SUCCEEDED" ? "good" : latest.status === "FAILED" ? "danger" : "warn"}>{latest.status}</Badge> : null}</div><p className="mt-1 flex items-center gap-1 text-xs text-[var(--muted)]"><CircleDollarSign className="size-3.5" /> Ước tính {estimate.data ? `$${estimate.data.estimatedCost.toFixed(4)} USD` : "đang tính"}; số tiền thực tế được lưu theo provider trace.</p>{start.error ? <p className="mt-2 text-sm font-semibold text-[var(--coral)]">{start.error.message}</p> : null}{latest?.errorMessage ? <p className="mt-2 text-sm text-[var(--coral)]">{latest.errorMessage}</p> : null}</div>
    {canOperate ? <Button disabled={start.isPending || running} onClick={() => start.mutate()}>{running ? <><RefreshCw className="mr-2 size-4 animate-spin" />Đang xử lý</> : `Tạo ${operation.toLowerCase()}`}</Button> : <Badge>Chỉ xem</Badge>}
  </Card>;
}
