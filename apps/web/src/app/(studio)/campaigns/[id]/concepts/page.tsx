"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, LockKeyhole, Save, X } from "lucide-react";
import { Suspense, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { CampaignHeader, GenerationPanel, campaignProgressQueryKey, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { environmentOptions, productPlacementOptions } from "@/lib/form-options";
import { apiError } from "@/lib/problem";

type Concept = components["schemas"]["CampaignConcept"];
type Candidate = components["schemas"]["ConceptCandidate"];

export default function ConceptsPage() { return <Suspense fallback={<SkeletonRows />}><Concepts /></Suspense>; }

function Concepts() {
  const scope = useCampaignRoute();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: ["concepts", scope.campaignId], enabled: !!scope.clientId && !!scope.workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/concepts", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải concept."); return data; } });
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: ["concepts", scope.campaignId] }), qc.invalidateQueries({ queryKey: campaignProgressQueryKey(scope) })]);
  return <CampaignHeader active="/concepts" title="Concept Generator" description="So sánh các concept có cấu trúc, chỉnh sửa theo phiên bản, duyệt rồi khóa đúng một concept trước khi tạo content và script.">
    <datalist id="concept-environment-options">{environmentOptions.map((option) => <option key={option.value} value={option.value} />)}</datalist><datalist id="concept-product-placement-options">{productPlacementOptions.map((option) => <option key={option.value} value={option.value} />)}</datalist>
    <GenerationPanel operation="CONCEPTS" />
    {query.isLoading ? <SkeletonRows /> : query.error ? <StatePanel title="Không thể tải concept" tone="danger">{query.error.message}</StatePanel> : query.data?.items.length === 0 ? <StatePanel title="Chưa có concept">Chạy AI concepts để nhận cả Interview Review và Problem Solution.</StatePanel> : <div className="grid gap-5 xl:grid-cols-2">{query.data?.items.map((item) => <ConceptCard key={item.id} item={item} scope={scope} onChanged={refresh} />)}</div>}
  </CampaignHeader>;
}

function ConceptCard({ item, scope, onChanged }: { item: Concept; scope: ReturnType<typeof useCampaignRoute>; onChanged: () => Promise<unknown> }) {
  const { canOperate, canReview } = usePermissions();
  const [draft, setDraft] = useState<Candidate>(item.payload);
  const update = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/concepts/{conceptId}", { params: { path: { ...scope, conceptId: item.id } }, body: { payload: draft, version: item.version } }); if (error || !data) throw apiError(error, "Không thể lưu concept."); return data; }, onSuccess: onChanged });
  const decide = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT" | "LOCK") => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/concepts/{conceptId}/decision", { params: { path: { ...scope, conceptId: item.id } }, body: { action, version: item.version, notes: `Quyết định ${action.toLowerCase()} từ studio` } }); if (error || !data) throw apiError(error, "Không thể ghi quyết định concept."); return data; }, onSuccess: onChanged });
  const change = <K extends keyof Candidate>(key: K, value: Candidate[K]) => setDraft((current) => ({ ...current, [key]: value }));
  const locked = item.status === "LOCKED";
  return <Card className="p-6"><div className="mb-5 flex flex-wrap items-center gap-2"><Badge tone={locked ? "good" : item.status === "REJECTED" ? "danger" : "warn"}>{item.status}</Badge><Badge>{draft.videoFormat}</Badge><span className="ml-auto text-xs text-[var(--muted)]">v{item.currentVersion} · ${draft.estimatedCostUsd.toFixed(4)}</span></div>
    <fieldset disabled={!canOperate} className="grid gap-4"><Field label="Tên concept"><input className={inputClass} disabled={locked || !canOperate} value={draft.title} onChange={(e) => change("title", e.target.value)} /></Field><Field label="Hook"><textarea className={textareaClass} disabled={locked || !canOperate} value={draft.hook} onChange={(e) => change("hook", e.target.value)} /></Field><Field label="Thông điệp cốt lõi"><textarea className={textareaClass} disabled={locked || !canOperate} value={draft.coreMessage} onChange={(e) => change("coreMessage", e.target.value)} /></Field><Field label="Phù hợp khán giả"><textarea className={textareaClass} disabled={locked || !canOperate} value={draft.audienceFit} onChange={(e) => change("audienceFit", e.target.value)} /></Field><div className="grid grid-cols-2 gap-3"><Field label="Số cảnh"><input className={inputClass} type="number" min={2} max={8} disabled={locked || !canOperate} value={draft.expectedSceneCount} onChange={(e) => change("expectedSceneCount", Number(e.target.value))} /></Field><Field label="Giây Seedance"><input className={inputClass} type="number" min={6} disabled={locked || !canOperate} value={draft.expectedSeedanceSeconds} onChange={(e) => change("expectedSeedanceSeconds", Number(e.target.value))} /></Field></div><Field label="Bối cảnh"><input className={inputClass} list="concept-environment-options" disabled={locked || !canOperate} value={draft.environment} onChange={(e) => change("environment", e.target.value)} /></Field><Field label="Product placement"><input className={inputClass} list="concept-product-placement-options" disabled={locked || !canOperate} value={draft.productPlacement} onChange={(e) => change("productPlacement", e.target.value)} /></Field></fieldset>
    {(update.error || decide.error) ? <p className="mt-4 text-sm text-[var(--coral)]">{update.error?.message ?? decide.error?.message}</p> : null}<div className="mt-5 flex flex-wrap gap-2">{canOperate && !locked ? <Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={update.isPending} onClick={() => update.mutate()}><Save className="mr-2 size-4" />Lưu</Button> : null}{canReview && (item.status === "DRAFT" || item.status === "REJECTED") ? <Button disabled={decide.isPending} onClick={() => decide.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt</Button> : null}{canReview && item.status === "APPROVED" ? <Button disabled={decide.isPending} onClick={() => decide.mutate("LOCK")}><LockKeyhole className="mr-2 size-4" />Khóa concept</Button> : null}{canReview && !locked && item.status !== "REJECTED" ? <Button className="bg-[var(--coral)]" disabled={decide.isPending} onClick={() => decide.mutate("REJECT")}><X className="mr-2 size-4" />Từ chối</Button> : null}</div>
  </Card>;
}
