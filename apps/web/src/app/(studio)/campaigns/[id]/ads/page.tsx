"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, Ban, Check, CircleDollarSign, Pause, Play, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { Suspense, useState } from "react";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type MetaAdCampaign = components["schemas"]["MetaAdCampaign"];
type MetaAdAction = components["schemas"]["MetaAdAction"];
type MetaAdGuardrails = components["schemas"]["MetaAdGuardrails"];
type ActionKind = "ACTIVATE" | "RESUME" | "PAUSE" | "ARCHIVE" | "BUDGET_CHANGE";

function AdsContent() {
  const scope = useCampaignRoute();
  const queryClient = useQueryClient();
  const [name, setName] = useState("Campaign Meta mới");
  const [dailyBudget, setDailyBudget] = useState("100000");
  const [destinationUrl, setDestinationUrl] = useState("https://example.com/products");
  const [adAccountId, setAdAccountId] = useState("");
  const [socialAccountId, setSocialAccountId] = useState("");
  const [pixelId, setPixelId] = useState("");
  const [creativeAssetId, setCreativeAssetId] = useState("");
  const [primaryTexts, setPrimaryTexts] = useState("Khám phá sản phẩm ngay hôm nay\nSẵn sàng cho hành trình tiếp theo");
  const [headlines, setHeadlines] = useState("Tìm hiểu sản phẩm");
  const connection = useQuery({ queryKey: ["meta-connection", scope.clientId, scope.workspaceId], enabled: !!scope.clientId && !!scope.workspaceId, retry: false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/meta/connection", { params: { path: scope } }); if (error || !data) return null; return data; } });
  const guardrails = useQuery({ queryKey: ["meta-ad-guardrails", scope.clientId, scope.workspaceId], enabled: !!scope.clientId && !!scope.workspaceId, retry: false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/meta-ad-guardrails", { params: { path: scope } }); if (error || !data) return null; return data; } });
  const renders = useQuery({ queryKey: ["final-renders", scope], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải final render."); return data; } });
  const campaigns = useQuery({ queryKey: ["meta-ad-campaigns", scope], enabled: !!scope.clientId && !!scope.workspaceId, refetchInterval: (query) => query.state.data?.items.some((item) => ["APPROVED", "CREATING"].includes(item.status)) ? 2500 : false, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải Meta Ads."); return data; } });
  const selectedAsset = creativeAssetId || renders.data?.items.find((item) => item.selected)?.outputAssetId || "";
  const selectedAdAccount = adAccountId || connection.data?.adAccounts[0]?.id || "";
  const selectedSocial = socialAccountId || connection.data?.accounts.find((item) => item.status === "CONNECTED")?.id || "";
  const selectedPixel = pixelId || connection.data?.pixels.find((item) => item.metaAdAccountId === selectedAdAccount)?.id || "";
  const currentCampaignCap = guardrails.data?.defaultCampaignSpendCapMinor.toString() ?? "2000000";
  const refresh = async () => Promise.all([queryClient.invalidateQueries({ queryKey: ["meta-ad-campaigns", scope] }), queryClient.invalidateQueries({ queryKey: ["meta-ad-guardrails", scope.clientId, scope.workspaceId] })]);
  const create = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } }, body: {
    metaAdAccountId: selectedAdAccount, socialAccountId: selectedSocial, metaPixelId: selectedPixel || null, name, objective: "OUTCOME_TRAFFIC", dailyBudgetMinor: Number(dailyBudget), lifetimeBudgetMinor: null, campaignSpendCapMinor: Number(currentCampaignCap), currency: "VND", startsAt: null, endsAt: null,
    audience: { countries: ["VN"], ageMin: 25, ageMax: 45, genders: [], interests: ["travel", "lifestyle"], customAudienceIds: [], retargetingAudienceIds: connection.data?.audiences.map((item) => item.providerAudienceId) ?? [] }, placements: ["facebook_feed", "instagram_reels"], destinationUrl, utmParameters: { utm_source: "meta", utm_medium: "paid_social", utm_campaign: name.toLowerCase().replaceAll(" ", "_") }, conversionEvent: selectedPixel ? "PAGE_VIEW" : null,
    creative: { mediaAssetId: selectedAsset, thumbnailAssetId: null, primaryTextVariants: lines(primaryTexts), headlineVariants: lines(headlines), ctaVariants: ["LEARN_MORE"] },
  } }); if (error || !data) throw apiError(error, "Campaign bị chặn bởi final-render, connection hoặc budget guardrail."); return data; }, onSuccess: refresh });
  return <CampaignHeader active="/ads" title="Meta Ads có kiểm soát" description="Campaign luôn được tạo PAUSED. Kích hoạt và tăng budget cần người duyệt, xác nhận số tiền chính xác và audit trail.">
    {!connection.data ? <StatePanel title="Chưa kết nối Meta">Kết nối tại <Link className="font-bold text-[var(--moss)] underline" href={`/settings/meta?clientId=${scope.clientId}&workspaceId=${scope.workspaceId}`}>Settings → Meta</Link> để khám phá Ad Account, Pixel và Audience.</StatePanel> : <>
      <GuardrailCard key={guardrails.data?.version ?? "new"} initial={guardrails.data} scope={scope} onChanged={refresh} />
      <Card className="mt-5 p-6"><h2 className="font-serif text-2xl font-bold">Tạo campaign PAUSED</h2><div className="mt-5 grid gap-4 md:grid-cols-2">
        <Field label="Tên campaign"><input className={inputClass} value={name} onChange={(event) => setName(event.target.value)} /></Field><Field label="Daily budget · VND minor"><input className={inputClass} type="number" value={dailyBudget} onChange={(event) => setDailyBudget(event.target.value)} /></Field>
        <Field label="Ad Account"><select className={inputClass} value={selectedAdAccount} onChange={(event) => setAdAccountId(event.target.value)}>{connection.data.adAccounts.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.currency}</option>)}</select></Field><Field label="Page / Instagram"><select className={inputClass} value={selectedSocial} onChange={(event) => setSocialAccountId(event.target.value)}>{connection.data.accounts.map((item) => <option key={item.id} value={item.id}>{item.platform} · {item.name}</option>)}</select></Field>
        <Field label="Pixel"><select className={inputClass} value={selectedPixel} onChange={(event) => setPixelId(event.target.value)}><option value="">Không dùng Pixel</option>{connection.data.pixels.filter((item) => item.metaAdAccountId === selectedAdAccount).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></Field><Field label="Final creative asset"><input className={inputClass} value={selectedAsset} onChange={(event) => setCreativeAssetId(event.target.value)} /></Field>
        <Field label="Destination HTTPS URL"><input className={inputClass} type="url" value={destinationUrl} onChange={(event) => setDestinationUrl(event.target.value)} /></Field><Field label="Headlines · mỗi dòng một variant"><textarea className={textareaClass} value={headlines} onChange={(event) => setHeadlines(event.target.value)} /></Field>
        <div className="md:col-span-2"><Field label="Primary text · mỗi dòng một variant"><textarea className={textareaClass} value={primaryTexts} onChange={(event) => setPrimaryTexts(event.target.value)} /></Field></div>
      </div>{create.error ? <p className="mt-3 text-sm text-[var(--coral)]">{create.error.message}</p> : null}<Button className="mt-5" disabled={create.isPending || !guardrails.data || !selectedAsset || !selectedAdAccount || !selectedSocial} onClick={() => create.mutate()}><CircleDollarSign className="mr-2 size-4" />Tạo để duyệt</Button>{!guardrails.data ? <p className="mt-2 text-xs text-[var(--muted)]">Lưu guardrails trước khi tạo campaign.</p> : null}</Card>
    </>}
    <section className="mt-7"><h2 className="mb-4 font-serif text-2xl font-bold">Campaign & action history</h2>{campaigns.isLoading ? <SkeletonRows /> : campaigns.error ? <StatePanel title="Không thể tải Meta Ads" tone="danger">{campaigns.error.message}</StatePanel> : campaigns.data?.items.length ? <div className="grid gap-5">{campaigns.data.items.map((item) => <CampaignCard key={item.id} item={item} scope={scope} onChanged={refresh} />)}</div> : <StatePanel title="Chưa có Meta Ads campaign">Campaign đầu tiên sẽ được giữ ở approval required rồi tạo trên Meta với trạng thái PAUSED.</StatePanel>}</section>
  </CampaignHeader>;
}

function GuardrailCard({ initial, scope, onChanged }: { initial: MetaAdGuardrails | null | undefined; scope: ReturnType<typeof useCampaignRoute>; onChanged: () => Promise<unknown> }) {
  const [workspaceCap, setWorkspaceCap] = useState((initial?.workspaceSpendCapMinor ?? 10_000_000).toString());
  const [campaignCap, setCampaignCap] = useState((initial?.defaultCampaignSpendCapMinor ?? 2_000_000).toString());
  const [maxIncrease, setMaxIncrease] = useState((initial?.maximumBudgetIncreasePercent ?? 20).toString());
  const save = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/meta-ad-guardrails", { params: { path: scope }, body: { workspaceSpendCapMinor: Number(workspaceCap), defaultCampaignSpendCapMinor: Number(campaignCap), maximumBudgetIncreasePercent: Number(maxIncrease), currency: "VND", version: initial?.version ?? 0 } }); if (error || !data) throw apiError(error, "Guardrail không hợp lệ hoặc đã thay đổi."); return data; }, onSuccess: onChanged });
  return <Card className="p-6"><div className="flex items-center gap-3"><ShieldCheck className="size-6 text-[var(--moss)]" /><h2 className="font-serif text-2xl font-bold">Workspace budget guardrails</h2></div><div className="mt-5 grid gap-4 md:grid-cols-3"><Field label="Workspace cap · VND minor"><input className={inputClass} type="number" value={workspaceCap} onChange={(event) => setWorkspaceCap(event.target.value)} /></Field><Field label="Default campaign cap"><input className={inputClass} type="number" value={campaignCap} onChange={(event) => setCampaignCap(event.target.value)} /></Field><Field label="Tăng budget tối đa · %"><input className={inputClass} type="number" value={maxIncrease} onChange={(event) => setMaxIncrease(event.target.value)} /></Field></div>{save.error ? <p className="mt-3 text-sm text-[var(--coral)]">{save.error.message}</p> : null}<Button className="mt-5" disabled={save.isPending} onClick={() => save.mutate()}>Lưu guardrails</Button></Card>;
}

function CampaignCard({ item, scope, onChanged }: { item: MetaAdCampaign; scope: ReturnType<typeof useCampaignRoute>; onChanged: () => Promise<unknown> }) {
  const queryClient = useQueryClient();
  const budget = item.dailyBudgetMinor ?? item.lifetimeBudgetMinor ?? 0;
  const createPhrase = `CREATE PAUSED ${item.currency} ${budget}`;
  const [createConfirmation, setCreateConfirmation] = useState("");
  const [action, setAction] = useState<ActionKind>(item.status === "PAUSED" ? "ACTIVATE" : "PAUSE");
  const [requestedBudget, setRequestedBudget] = useState(budget.toString());
  const [actionConfirmation, setActionConfirmation] = useState("");
  const actionsKey = ["meta-ad-actions", item.id];
  const actions = useQuery({ queryKey: actionsKey, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns/{adCampaignId}/actions", { params: { path: { ...scope, adCampaignId: item.id } } }); if (error || !data) throw apiError(error, "Không thể tải Ads actions."); return data; } });
  const refresh = async () => { await queryClient.invalidateQueries({ queryKey: actionsKey }); await onChanged(); };
  const reviewCreate = useMutation({ mutationFn: async (decision: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns/{adCampaignId}/review", { params: { path: { ...scope, adCampaignId: item.id } }, body: { action: decision, version: item.version, notes: decision === "APPROVE" ? "Budget, audience, placements, URL và creative đã được kiểm tra." : "Campaign cần chỉnh sửa trước khi tạo.", confirmedBudgetMinor: budget, confirmationText: createConfirmation } }); if (error || !data) throw apiError(error, "Xác nhận budget không khớp hoặc campaign đã thay đổi."); return data; }, onSuccess: refresh });
  const requestAction = useMutation({ mutationFn: async () => { const amount = action === "BUDGET_CHANGE" ? Number(requestedBudget) : null; const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns/{adCampaignId}/actions", { params: { path: { ...scope, adCampaignId: item.id }, header: { "Idempotency-Key": newIdempotencyKey() } }, body: { action, requestedBudgetMinor: amount, confirmationText: actionConfirmation } }); if (error || !data) throw apiError(error, "Action bị chặn bởi trạng thái, confirmation hoặc budget guardrail."); return data; }, onSuccess: refresh });
  const expectedAction = action === "ACTIVATE" || action === "RESUME" ? `ACTIVATE ${item.currency} ${budget}` : action === "BUDGET_CHANGE" ? `BUDGET ${item.currency} ${requestedBudget}` : "Không cần confirmation cho pause/archive";
  return <Card className="p-6"><div className="flex flex-wrap items-center gap-2"><h3 className="font-serif text-xl font-bold">{item.name}</h3><Badge tone={item.status === "ACTIVE" ? "good" : item.status === "FAILED" ? "danger" : item.status === "APPROVAL_REQUIRED" ? "warn" : "neutral"}>{item.status}</Badge><span className="ml-auto text-sm font-bold">{item.currency} {budget.toLocaleString("vi-VN")}</span></div><p className="mt-3 text-sm text-[var(--muted)]">{item.objective} · {item.audience.countries.join(", ")} · {item.audience.ageMin}–{item.audience.ageMax} · cap {item.campaignSpendCapMinor.toLocaleString("vi-VN")}</p><p className="mt-2 break-all text-xs text-[var(--muted)]">{item.destinationUrl}</p>
    {item.status === "APPROVAL_REQUIRED" ? <div className="mt-5 rounded-2xl bg-[#fff8e8] p-4"><p className="text-sm font-bold">Gõ chính xác để tạo campaign PAUSED:</p><code className="mt-2 block text-sm">{createPhrase}</code><input aria-label="Xác nhận tạo campaign PAUSED" className={`${inputClass} mt-3`} value={createConfirmation} onChange={(event) => setCreateConfirmation(event.target.value)} /><div className="mt-3 flex gap-2"><Button disabled={reviewCreate.isPending || createConfirmation !== createPhrase} onClick={() => reviewCreate.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt tạo PAUSED</Button><Button className="bg-[var(--coral)]" disabled={reviewCreate.isPending} onClick={() => reviewCreate.mutate("REJECT")}><Ban className="mr-2 size-4" />Từ chối</Button></div></div> : null}
    {["PAUSED", "ACTIVE"].includes(item.status) ? <div className="mt-5 grid gap-3 rounded-2xl bg-white p-4 ring-1 ring-[var(--line)] md:grid-cols-[1fr_1fr_1fr_auto]"><Field label="Action"><select className={inputClass} value={action} onChange={(event) => setAction(event.target.value as ActionKind)}>{item.status === "PAUSED" ? <><option value="ACTIVATE">Activate</option><option value="RESUME">Resume</option></> : <option value="PAUSE">Pause</option>}<option value="BUDGET_CHANGE">Budget change</option><option value="ARCHIVE">Archive</option></select></Field><Field label="Budget mới"><input className={inputClass} type="number" disabled={action !== "BUDGET_CHANGE"} value={requestedBudget} onChange={(event) => setRequestedBudget(event.target.value)} /></Field><Field label={`Confirmation: ${expectedAction}`}><input className={inputClass} value={actionConfirmation} disabled={action === "PAUSE" || action === "ARCHIVE"} onChange={(event) => setActionConfirmation(event.target.value)} /></Field><Button className="self-end" disabled={requestAction.isPending || ((action === "ACTIVATE" || action === "RESUME" || action === "BUDGET_CHANGE") && actionConfirmation !== expectedAction)} onClick={() => requestAction.mutate()}>{action === "PAUSE" ? <Pause className="mr-2 size-4" /> : action === "ARCHIVE" ? <Archive className="mr-2 size-4" /> : <Play className="mr-2 size-4" />}Yêu cầu</Button></div> : null}
    {(reviewCreate.error || requestAction.error) ? <p className="mt-3 text-sm text-[var(--coral)]">{reviewCreate.error?.message ?? requestAction.error?.message}</p> : null}
    <div className="mt-5"><h4 className="text-sm font-bold">Action audit</h4>{actions.isLoading ? <p className="mt-2 text-sm text-[var(--muted)]">Đang tải…</p> : <div className="mt-2 grid gap-2">{actions.data?.items.map((entry) => <ActionRow key={entry.id} item={entry} scope={scope} adCampaignId={item.id} onChanged={refresh} />)}</div>}</div>
  </Card>;
}

function ActionRow({ item, scope, adCampaignId, onChanged }: { item: MetaAdAction; scope: ReturnType<typeof useCampaignRoute>; adCampaignId: string; onChanged: () => Promise<unknown> }) {
  const review = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/meta-ad-campaigns/{adCampaignId}/actions/{actionId}/review", { params: { path: { ...scope, adCampaignId, actionId: item.id } }, body: { action, version: item.version, notes: action === "APPROVE" ? "Human reviewer approved this exact action." : "Action rejected by human reviewer." } }); if (error || !data) throw apiError(error, "Không thể review action."); return data; }, onSuccess: onChanged });
  return <div className="flex flex-wrap items-center gap-2 rounded-2xl bg-[#f6f6f1] px-4 py-3 text-sm"><span className="font-bold">{item.action}</span><Badge tone={item.status === "SUCCEEDED" ? "good" : item.status === "FAILED" || item.status === "REJECTED" ? "danger" : item.status === "PENDING_APPROVAL" ? "warn" : "neutral"}>{item.status}</Badge>{item.requestedBudgetMinor ? <span>{item.requestedBudgetMinor.toLocaleString("vi-VN")}</span> : null}<span className="ml-auto text-xs text-[var(--muted)]">v{item.version}</span>{item.status === "PENDING_APPROVAL" ? <><Button disabled={review.isPending} onClick={() => review.mutate("APPROVE")}><Check className="mr-1 size-4" />Duyệt</Button><Button className="bg-[var(--coral)]" disabled={review.isPending} onClick={() => review.mutate("REJECT")}><Ban className="mr-1 size-4" />Từ chối</Button></> : null}{review.error ? <span className="w-full text-xs text-[var(--coral)]">{review.error.message}</span> : null}</div>;
}

function lines(value: string) { return value.split("\n").map((item) => item.trim()).filter(Boolean); }

export default function AdsPage() {
  return <Suspense fallback={<SkeletonRows />}><AdsContent /></Suspense>;
}
