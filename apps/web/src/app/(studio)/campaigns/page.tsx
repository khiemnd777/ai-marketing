"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Clapperboard, Copy, Plus, Search } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type CampaignInput = components["schemas"]["CampaignInput"];
const initial: CampaignInput = { brandId: "", productId: "", name: "", objective: "PRODUCT_INTRODUCTION", targetAudience: "Khách du lịch thường xuyên", market: "Việt Nam", country: "VN", language: "vi", socialPlatformTargets: ["FACEBOOK", "INSTAGRAM"], videoFormat: "INTERVIEW_REVIEW", durationSeconds: 30, aspectRatio: "9:16", tone: "Tin cậy, gần gũi", offer: "", cta: "Tìm hiểu ngay", changeSummary: "Brief đầu tiên" };

export default function CampaignsPage() { return <Suspense fallback={<SkeletonRows />}><CampaignsContent /></Suspense>; }

function CampaignsContent() {
  const { canOperate } = usePermissions();
  const query = useSearchParams();
  const clientId = query.get("clientId") ?? "";
  const workspaceId = query.get("workspaceId") ?? "";
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [show, setShow] = useState(false);
  const [form, setForm] = useState<CampaignInput>(initial);
  const scope = { clientId, workspaceId };
  const campaigns = useQuery({ queryKey: ["campaigns", clientId, workspaceId, search], enabled: !!clientId && !!workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns", { params: { path: scope, query: { search: search || undefined } } }); if (error || !data) throw apiError(error, "Không thể tải campaign."); return data; } });
  const brands = useQuery({ queryKey: ["brands", clientId, workspaceId], enabled: !!clientId && !!workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/brands", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải brand."); return data; } });
  const products = useQuery({ queryKey: ["products", clientId, workspaceId], enabled: !!clientId && !!workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/products", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải sản phẩm."); return data; } });
  const create = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } }, body: form }); if (error || !data) throw apiError(error, "Không thể tạo campaign."); return data; }, onSuccess: async () => { setForm(initial); setShow(false); await qc.invalidateQueries({ queryKey: ["campaigns", clientId, workspaceId] }); } });
  const duplicate = useMutation({ mutationFn: async (item: components["schemas"]["Campaign"]) => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/duplicate", { params: { path: { ...scope, campaignId: item.id }, header: { "Idempotency-Key": newIdempotencyKey() } }, body: { name: `${item.name} — bản sao` } }); if (error || !data) throw apiError(error, "Không thể nhân bản campaign."); return data; }, onSuccess: async () => qc.invalidateQueries({ queryKey: ["campaigns", clientId, workspaceId] }) });
  if (!clientId || !workspaceId) return <><PageHeader eyebrow="Campaign Builder" title="Chiến dịch" description="Mỗi campaign được gắn cứng với một brand, một product và Product Truth trong workspace." /><StatePanel title="Chưa chọn workspace">Mở khách hàng và workspace trước, rồi quay lại Campaign Builder.</StatePanel></>;
  const change = <K extends keyof CampaignInput>(key: K, value: CampaignInput[K]) => setForm((current) => ({ ...current, [key]: value }));
  return <><PageHeader eyebrow="Campaign Builder" title="Chiến dịch" description="Tạo brief có phiên bản, khóa concept, duyệt nội dung và đi tiếp qua script, nhân vật, cảnh quay." action={canOperate ? <Button onClick={() => setShow((value) => !value)}><Plus className="mr-2 size-4" />Campaign mới</Button> : undefined} />
    {canOperate && show ? <Card className="mb-6 p-6"><div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Field label="Tên campaign"><input className={inputClass} value={form.name} onChange={(e) => change("name", e.target.value)} /></Field>
      <Field label="Brand"><select className={inputClass} value={form.brandId} onChange={(e) => change("brandId", e.target.value)}><option value="">Chọn brand</option>{brands.data?.items.filter((item) => item.status === "ACTIVE").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></Field>
      <Field label="Sản phẩm"><select className={inputClass} value={form.productId} onChange={(e) => change("productId", e.target.value)}><option value="">Chọn sản phẩm</option>{products.data?.items.filter((item) => item.status !== "ARCHIVED").map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></Field>
      <Field label="Mục tiêu"><select className={inputClass} value={form.objective} onChange={(e) => change("objective", e.target.value as CampaignInput["objective"])}>{["PRODUCT_INTRODUCTION", "AWARENESS", "ENGAGEMENT", "WEBSITE_TRAFFIC", "LEAD_GENERATION", "SALES", "PROMOTION"].map((value) => <option key={value}>{value}</option>)}</select></Field>
      <Field label="Định dạng"><select className={inputClass} value={form.videoFormat} onChange={(e) => change("videoFormat", e.target.value as CampaignInput["videoFormat"])}><option value="INTERVIEW_REVIEW">Interview Review</option><option value="PROBLEM_SOLUTION">Problem Solution</option></select></Field>
      <Field label="Thời lượng"><select className={inputClass} value={form.durationSeconds} onChange={(e) => change("durationSeconds", Number(e.target.value) as 30 | 45)}><option value={30}>30 giây</option><option value={45}>45 giây</option></select></Field>
      <Field label="Thị trường"><input className={inputClass} value={form.market} onChange={(e) => change("market", e.target.value)} /></Field>
      <Field label="Quốc gia"><input className={inputClass} maxLength={2} value={form.country} onChange={(e) => change("country", e.target.value.toUpperCase())} /></Field>
      <Field label="Ngôn ngữ"><select className={inputClass} value={form.language} onChange={(e) => change("language", e.target.value as "vi" | "en")}><option value="vi">Tiếng Việt</option><option value="en">English</option></select></Field>
      <div className="md:col-span-2"><Field label="Đối tượng"><textarea className={textareaClass} value={form.targetAudience} onChange={(e) => change("targetAudience", e.target.value)} /></Field></div>
      <Field label="Tone"><textarea className={textareaClass} value={form.tone} onChange={(e) => change("tone", e.target.value)} /></Field>
      <Field label="Offer"><input className={inputClass} value={form.offer} onChange={(e) => change("offer", e.target.value)} /></Field>
      <Field label="CTA"><input className={inputClass} value={form.cta} onChange={(e) => change("cta", e.target.value)} /></Field>
      <Field label="Ngân sách ads"><input className={inputClass} type="number" min="0" value={form.plannedAdsBudget ?? ""} onChange={(e) => change("plannedAdsBudget", e.target.value ? Number(e.target.value) : null)} /></Field>
      <Field label="Tiền tệ"><input className={inputClass} maxLength={3} value={form.budgetCurrency ?? ""} onChange={(e) => change("budgetCurrency", e.target.value ? e.target.value.toUpperCase() : null)} /></Field>
    </div>{create.error ? <p className="mt-4 text-sm font-semibold text-[var(--coral)]">{create.error.message}</p> : null}<div className="mt-5 flex gap-3"><Button disabled={create.isPending || form.name.length < 2 || !form.brandId || !form.productId} onClick={() => create.mutate()}>{create.isPending ? "Đang tạo…" : "Tạo brief"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => setShow(false)}>Hủy</Button></div></Card> : null}
    <div className="mb-5 flex items-center gap-3 rounded-2xl border border-[var(--line)] bg-white px-4"><Search className="size-4 text-[var(--muted)]" /><input aria-label="Tìm campaign" className="min-h-11 w-full bg-transparent text-sm outline-none" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Tên campaign" /></div>
    {campaigns.isLoading ? <SkeletonRows /> : campaigns.error ? <StatePanel title="Không thể tải campaign" tone="danger">{campaigns.error.message}</StatePanel> : campaigns.data?.items.length === 0 ? <StatePanel title="Chưa có campaign">Tạo brief đầu tiên để bắt đầu concept và nội dung.</StatePanel> : <div className="grid gap-4 md:grid-cols-2">{campaigns.data?.items.map((item) => <Card key={item.id} className="flex h-full gap-4 p-5"><span className="grid size-12 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><Clapperboard className="size-5 text-[var(--moss)]" /></span><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><Link className="font-serif text-xl font-bold hover:text-[var(--moss)]" href={`/campaigns/${item.id}?clientId=${clientId}&workspaceId=${workspaceId}`}>{item.name}</Link><Badge tone={item.status === "APPROVED" || item.status === "READY_TO_PUBLISH" ? "good" : "warn"}>{item.status}</Badge></div><p className="mt-2 text-sm text-[var(--muted)]">{item.videoFormat} · {item.durationSeconds}s · {item.language}</p></div>{canOperate ? <button className="self-start rounded-full p-2 text-[var(--muted)] hover:bg-[#edf0e7]" aria-label="Nhân bản" disabled={duplicate.isPending} onClick={() => duplicate.mutate(item)}><Copy className="size-4" /></button> : null}</Card>)}</div>}
  </>;
}
