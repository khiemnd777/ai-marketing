"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

type Campaign = components["schemas"]["Campaign"];
type CampaignInput = components["schemas"]["CampaignInput"];

function toInput(item: Campaign): CampaignInput { return { brandId: item.brandId, productId: item.productId, name: item.name, objective: item.objective, targetAudience: item.targetAudience, market: item.market, country: item.country, language: item.language, socialPlatformTargets: item.socialPlatformTargets, videoFormat: item.videoFormat, durationSeconds: item.durationSeconds, aspectRatio: item.aspectRatio, tone: item.tone, offer: item.offer, cta: item.cta, plannedAdsBudget: item.plannedAdsBudget, budgetCurrency: item.budgetCurrency, startsOn: item.startsOn, endsOn: item.endsOn, changeSummary: "Cập nhật brief", version: item.version }; }

export default function CampaignBriefPage() { return <Suspense fallback={<SkeletonRows />}><CampaignBrief /></Suspense>; }

function CampaignBrief() {
  const scope = useCampaignRoute();
  const qc = useQueryClient();
  const campaign = useQuery({ queryKey: ["campaign", scope.clientId, scope.workspaceId, scope.campaignId], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải brief."); return data; } });
  const characters = useQuery({ queryKey: ["characters", scope.clientId, scope.workspaceId], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/characters", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải nhân vật."); return data; } });
  const selection = useQuery({ queryKey: ["campaign-characters", scope.campaignId], enabled: !!scope.clientId && !!scope.workspaceId, retry: false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/characters", { params: { path: scope } }); if (error || !data) throw apiError(error, "Campaign chưa chọn đủ hai nhân vật."); return data; } });
  const [formOverride, setFormOverride] = useState<CampaignInput | null>(null);
  const [primaryOverride, setPrimary] = useState<string | null>(null);
  const [listenerOverride, setListener] = useState<string | null>(null);
  const form = formOverride ?? (campaign.data ? toInput(campaign.data) : null);
  const primary = primaryOverride ?? selection.data?.primary.id ?? "";
  const listener = listenerOverride ?? selection.data?.listener.id ?? "";
  const update = useMutation({ mutationFn: async (body: CampaignInput) => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}", { params: { path: scope }, body }); if (error || !data) throw apiError(error, "Không thể lưu brief."); return data; }, onSuccess: async () => { setFormOverride(null); await qc.invalidateQueries({ queryKey: ["campaign", scope.clientId, scope.workspaceId, scope.campaignId] }); } });
  const select = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/characters", { params: { path: scope }, body: { primaryCharacterId: primary, listenerCharacterId: listener } }); if (error || !data) throw apiError(error, "Không thể lưu cặp nhân vật."); return data; }, onSuccess: async () => { setPrimary(null); setListener(null); await qc.invalidateQueries({ queryKey: ["campaign-characters", scope.campaignId] }); } });
  const change = <K extends keyof CampaignInput>(key: K, value: CampaignInput[K]) => setFormOverride((current) => ({ ...(current ?? toInput(campaign.data!)), [key]: value }));
  return <CampaignHeader active="" title="Campaign brief" description="Brief là đầu vào có phiên bản. Mỗi thay đổi sẽ vô hiệu hóa concept, nội dung, script và cảnh đã duyệt ở phía sau.">
    {campaign.isLoading || !form ? <SkeletonRows /> : campaign.error ? <StatePanel title="Không thể tải brief" tone="danger">{campaign.error.message}</StatePanel> : <div className="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,.6fr)]">
      <Card className="p-6"><div className="mb-5 flex items-center justify-between"><h2 className="font-serif text-2xl font-bold">Thông tin chiến dịch</h2><Badge tone="warn">v{campaign.data!.currentVersion}</Badge></div><div className="grid gap-4 md:grid-cols-2">
        <Field label="Tên"><input className={inputClass} value={form.name} onChange={(e) => change("name", e.target.value)} /></Field>
        <Field label="Mục tiêu"><select className={inputClass} value={form.objective} onChange={(e) => change("objective", e.target.value as CampaignInput["objective"])}>{["PRODUCT_INTRODUCTION", "AWARENESS", "ENGAGEMENT", "WEBSITE_TRAFFIC", "LEAD_GENERATION", "SALES", "PROMOTION"].map((value) => <option key={value}>{value}</option>)}</select></Field>
        <Field label="Format"><select className={inputClass} value={form.videoFormat} onChange={(e) => change("videoFormat", e.target.value as CampaignInput["videoFormat"])}><option value="INTERVIEW_REVIEW">Interview Review</option><option value="PROBLEM_SOLUTION">Problem Solution</option></select></Field>
        <Field label="Thời lượng"><select className={inputClass} value={form.durationSeconds} onChange={(e) => change("durationSeconds", Number(e.target.value) as 30 | 45)}><option value={30}>30 giây</option><option value={45}>45 giây</option></select></Field>
        <Field label="Thị trường"><input className={inputClass} value={form.market} onChange={(e) => change("market", e.target.value)} /></Field>
        <Field label="Ngôn ngữ"><select className={inputClass} value={form.language} onChange={(e) => change("language", e.target.value as "vi" | "en")}><option value="vi">Tiếng Việt</option><option value="en">English</option></select></Field>
        <div className="md:col-span-2"><Field label="Đối tượng"><textarea className={textareaClass} value={form.targetAudience} onChange={(e) => change("targetAudience", e.target.value)} /></Field></div>
        <Field label="Tone"><textarea className={textareaClass} value={form.tone} onChange={(e) => change("tone", e.target.value)} /></Field>
        <Field label="Offer"><textarea className={textareaClass} value={form.offer} onChange={(e) => change("offer", e.target.value)} /></Field>
        <div className="md:col-span-2"><Field label="CTA"><input className={inputClass} value={form.cta} onChange={(e) => change("cta", e.target.value)} /></Field></div>
      </div>{update.error ? <p className="mt-4 text-sm font-semibold text-[var(--coral)]">{update.error.message}</p> : null}<Button className="mt-5" disabled={update.isPending} onClick={() => update.mutate(form)}>{update.isPending ? "Đang lưu…" : "Lưu phiên bản brief"}</Button></Card>
      <Card className="h-fit p-6"><h2 className="font-serif text-2xl font-bold">Hai nhân vật</h2><p className="mt-2 text-sm leading-6 text-[var(--muted)]">Seedance luôn dùng một người nói và một người nghe khác nhau. Nhân vật thật phải có consent APPROVED.</p><div className="mt-5 grid gap-4"><Field label="Người nói"><select className={inputClass} value={primary} onChange={(e) => setPrimary(e.target.value)}><option value="">Chọn nhân vật</option>{characters.data?.items.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.consentStatus}</option>)}</select></Field><Field label="Người nghe"><select className={inputClass} value={listener} onChange={(e) => setListener(e.target.value)}><option value="">Chọn nhân vật</option>{characters.data?.items.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.consentStatus}</option>)}</select></Field></div>{select.error ? <p className="mt-3 text-sm text-[var(--coral)]">{select.error.message}</p> : null}<Button className="mt-5 w-full" disabled={select.isPending || !primary || !listener || primary === listener} onClick={() => select.mutate()}>{select.isPending ? "Đang lưu…" : "Khóa cặp nhân vật"}</Button></Card>
    </div>}
  </CampaignHeader>;
}
