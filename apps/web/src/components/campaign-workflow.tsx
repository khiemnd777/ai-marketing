"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, Check, CircleDollarSign, RefreshCw, Send } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, type ReactNode } from "react";
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

type CampaignProgressStepKey = components["schemas"]["CampaignProgressStep"]["key"];

const workflowTabs = [
  { key: "BRIEF", suffix: "", label: "Brief" },
  { key: "CONCEPT", suffix: "/concepts", label: "Concept" },
  { key: "CONTENT", suffix: "/content", label: "Nội dung" },
  { key: "SCRIPT", suffix: "/script", label: "Kịch bản" },
  { key: "SCENES", suffix: "/scenes", label: "Cảnh quay" },
  { key: "QUALITY", suffix: "/quality", label: "Duyệt take" },
  { key: "COMPOSER", suffix: "/composer", label: "Dựng & duyệt final" },
] as const;

const distributionTabs = [
  { key: "PUBLISHING", suffix: "/publishing", label: "Xuất bản" },
  { key: "ADS", suffix: "/ads", label: "Meta Ads" },
] as const;

const tabs = [...workflowTabs, ...distributionTabs] as const;

export function campaignProgressQueryKey({ campaignId, clientId, workspaceId }: { campaignId: string; clientId: string; workspaceId: string }) {
  return ["campaign-progress", clientId, workspaceId, campaignId] as const;
}

export function CampaignTabs({ campaignId, clientId, workspaceId, active, completedSteps = new Set() }: { campaignId: string; clientId: string; workspaceId: string; active: string; completedSteps?: ReadonlySet<CampaignProgressStepKey> }) {
  const activeIndex = tabs.findIndex(({ suffix }) => suffix === active);
  const currentStepRef = useRef<HTMLAnchorElement>(null);

  useEffect(() => {
    currentStepRef.current?.scrollIntoView?.({ block: "nearest", inline: "center" });
  }, [active]);

  return <nav className="mb-7 overflow-x-auto pb-2" aria-label="Tiến trình campaign">
    <ol className="flex min-w-[68rem] items-center px-0.5">
      {workflowTabs.map(({ key, suffix, label }, index) => {
        const isCurrent = index === activeIndex;
        const isPast = activeIndex >= 0 && index < activeIndex;
        const isCompleted = completedSteps.has(key);

        return <li key={suffix} className="relative min-w-0 flex-1">
          <Link
            ref={isCurrent ? currentStepRef : undefined}
            href={studioRoutes.campaign(clientId, workspaceId, campaignId, suffix)}
            aria-current={isCurrent ? "step" : undefined}
            data-completed={isCompleted ? "true" : "false"}
            className={cn(
              "group relative z-10 mx-auto flex min-h-[4.75rem] w-fit min-w-20 flex-col items-center gap-1.5 rounded-2xl px-2 py-1.5 text-center text-sm font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ink)] focus-visible:ring-offset-2",
              isCurrent && "bg-white text-[var(--ink)] ring-1 ring-[var(--line)]",
              !isCurrent && isCompleted && "text-[var(--moss)] hover:bg-[#edf0e7]",
              !isCurrent && !isCompleted && "text-[var(--muted)] hover:bg-white hover:text-[var(--ink)]",
            )}
          >
            <span className={cn(
              "grid size-8 shrink-0 place-items-center rounded-full border text-xs",
              isCompleted && "border-[var(--moss)] bg-[var(--moss)] text-white",
              isCurrent && !isCompleted && "border-[var(--ink)] bg-[var(--lime)] text-[var(--ink)]",
              !isCurrent && !isCompleted && "border-[var(--line)] bg-white text-[var(--muted)] group-hover:border-[var(--muted)]",
            )} aria-hidden="true">
              {isCompleted ? <Check className="size-4 stroke-[2.5]" /> : index + 1}
            </span>
            <span className="whitespace-nowrap">{label}</span>
            {isCompleted ? <span className="sr-only">Đã hoàn tất</span> : null}
            {isPast ? <span className="sr-only">Bước trước</span> : null}
            {isCurrent ? <span className="sr-only">Bước hiện tại</span> : null}
          </Link>
          <span aria-hidden="true" className={cn("absolute left-[calc(50%+1rem)] top-[1.3rem] h-0.5 w-[calc(100%-2rem)] rounded-full", isCompleted ? "bg-[var(--moss)]" : "bg-[var(--line)]")} />
        </li>;
      })}
      <li className="relative w-[18rem] shrink-0 pl-3">
        <div className="rounded-2xl border border-[var(--line)] bg-white/55 p-2">
          <span className="mb-1.5 block text-center text-[10px] font-bold uppercase tracking-[0.12em] text-[var(--muted)]">Bước 8 · Phân phối</span>
          <div className="grid grid-cols-2 gap-2" role="group" aria-label="Kênh phân phối">
            {distributionTabs.map(({ key, suffix, label }) => {
              const isCurrent = suffix === active;
              const isCompleted = completedSteps.has(key);
              const isOptional = key === "ADS";
              const Icon = key === "PUBLISHING" ? Send : CircleDollarSign;

              return <Link
                key={suffix}
                ref={isCurrent ? currentStepRef : undefined}
                href={studioRoutes.campaign(clientId, workspaceId, campaignId, suffix)}
                aria-current={isCurrent ? "step" : undefined}
                data-completed={isCompleted ? "true" : "false"}
                className={cn(
                  "group flex min-h-[4.75rem] flex-col items-center gap-1 rounded-xl px-2 py-1.5 text-center text-sm font-bold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ink)] focus-visible:ring-offset-2",
                  isCurrent && "bg-white text-[var(--ink)] ring-1 ring-[var(--line)]",
                  !isCurrent && isCompleted && "text-[var(--moss)] hover:bg-[#edf0e7]",
                  !isCurrent && !isCompleted && "text-[var(--muted)] hover:bg-white hover:text-[var(--ink)]",
                  isOptional && !isCurrent && "ring-1 ring-dashed ring-[var(--line)]",
                )}
              >
                <span className={cn(
                  "grid size-8 shrink-0 place-items-center rounded-full border",
                  isCompleted && "border-[var(--moss)] bg-[var(--moss)] text-white",
                  isCurrent && !isCompleted && "border-[var(--ink)] bg-[var(--lime)] text-[var(--ink)]",
                  !isCurrent && !isCompleted && "border-[var(--line)] bg-white text-[var(--muted)] group-hover:border-[var(--muted)]",
                )} aria-hidden="true">
                  {isCompleted ? <Check className="size-4 stroke-[2.5]" /> : <Icon className="size-4" />}
                </span>
                <span className="whitespace-nowrap">{label}</span>
                {isOptional ? <span className={cn("rounded-full px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide", isCurrent ? "bg-[var(--ink)]/10 text-[var(--ink)]" : "bg-[#edf0e7] text-[var(--muted)]")}>Tùy chọn</span> : null}
                {isCompleted ? <span className="sr-only">Đã hoàn tất</span> : null}
                {isCurrent ? <span className="sr-only">Bước hiện tại</span> : null}
              </Link>;
            })}
          </div>
        </div>
      </li>
    </ol>
  </nav>;
}

export function CampaignHeader({ active, title, description, children }: { active: string; title: string; description: string; children?: ReactNode }) {
  const scope = useCampaignRoute();
  const campaign = useQuery({
    queryKey: ["campaign", scope.clientId, scope.workspaceId, scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId && !!scope.campaignId,
    queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải campaign."); return data; },
  });
  const progress = useQuery({
    queryKey: campaignProgressQueryKey(scope),
    enabled: !!scope.clientId && !!scope.workspaceId && !!scope.campaignId,
    queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/progress", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể kiểm tra tiến trình campaign."); return data; },
    refetchInterval: (query) => {
      const currentKey = tabs.find(({ suffix }) => suffix === active)?.key;
      const needsAsyncRefresh = currentKey && ["SCENES", "QUALITY", "COMPOSER", "PUBLISHING", "ADS"].includes(currentKey);
      const currentCompleted = query.state.data?.steps.find((step) => step.key === currentKey)?.completed;
      return needsAsyncRefresh && !currentCompleted ? 2500 : false;
    },
  });
  const completedSteps = progress.data && !progress.error
    ? new Set(progress.data.steps.filter((step) => step.completed).map((step) => step.key))
    : new Set<CampaignProgressStepKey>();
  if (!scope.clientId || !scope.workspaceId) return <StatePanel title="Chưa chọn workspace">Mở campaign từ danh sách trong một workspace để giữ đúng phạm vi dữ liệu.</StatePanel>;
  return <><PageHeader eyebrow={campaign.data ? `${campaign.data.name} · ${campaign.data.status}` : "Campaign Builder"} title={title} description={description} />
    <CampaignTabs {...scope} active={active} completedSteps={completedSteps} />
    {progress.isPending ? <span className="sr-only" role="status">Đang kiểm tra tiến trình campaign.</span> : null}
    {progress.error ? <span className="sr-only" role="status">Không thể kiểm tra tiến trình; chưa đánh dấu bước nào hoàn tất.</span> : null}
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
    void Promise.all([
      queryClient.invalidateQueries({ queryKey: [resultKey, scope.campaignId] }),
      queryClient.invalidateQueries({ queryKey: campaignProgressQueryKey({ campaignId: scope.campaignId, clientId: scope.clientId, workspaceId: scope.workspaceId }) }),
    ]);
  }, [latest?.id, latest?.status, operation, queryClient, scope.campaignId, scope.clientId, scope.workspaceId]);
  const running = latest?.status === "QUEUED" || latest?.status === "RUNNING";
  return <Card className="mb-6 flex flex-col gap-4 p-5 md:flex-row md:items-center">
    <span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><Bot className="size-5 text-[var(--moss)]" /></span>
    <div className="flex-1"><div className="flex flex-wrap items-center gap-2"><h2 className="font-serif text-lg font-bold">AI {operation.toLowerCase()}</h2>{latest ? <Badge tone={latest.status === "SUCCEEDED" ? "good" : latest.status === "FAILED" ? "danger" : "warn"}>{latest.status}</Badge> : null}</div><p className="mt-1 flex items-center gap-1 text-xs text-[var(--muted)]"><CircleDollarSign className="size-3.5" /> Ước tính {estimate.data ? `$${estimate.data.estimatedCost.toFixed(4)} USD` : "đang tính"}; số tiền thực tế được lưu theo provider trace.</p>{start.error ? <p className="mt-2 text-sm font-semibold text-[var(--coral)]">{start.error.message}</p> : null}{latest?.errorMessage ? <p className="mt-2 text-sm text-[var(--coral)]">{latest.errorMessage}</p> : null}</div>
    {canOperate ? <Button disabled={start.isPending || running} onClick={() => start.mutate()}>{running ? <><RefreshCw className="mr-2 size-4 animate-spin" />Đang xử lý</> : `Tạo ${operation.toLowerCase()}`}</Button> : <Badge>Chỉ xem</Badge>}
  </Card>;
}
