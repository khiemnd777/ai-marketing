"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Lightbulb, RefreshCw, X } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";
import { useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass } from "@/components/ui";
import { api } from "@/lib/api";
import { includeCurrentOption } from "@/lib/form-options";
import { apiError } from "@/lib/problem";

type Recommendation = components["schemas"]["AnalyticsRecommendation"];

const day = (value: Date) => value.toISOString().slice(0, 10);
const today = day(new Date());
const thirtyDaysAgo = day(new Date(new Date().getTime() - 29 * 86_400_000));
const number = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 2 });
const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 4 });

export default function AnalyticsPage() {
  return <Suspense fallback={<SkeletonRows />}><AnalyticsContent /></Suspense>;
}

function AnalyticsContent() {
  const params = useSearchParams();
  const { clientId, workspaceId } = useStudioScope();
  const [from, setFrom] = useState(thirtyDaysAgo);
  const [to, setTo] = useState(today);
  const [campaignId, setCampaignId] = useState(params.get("campaignId") ?? "");
  const scope = { clientId, workspaceId };
  const query = { from, to, campaignId: campaignId || undefined };
  const qc = useQueryClient();
  const campaigns = useQuery({
    queryKey: ["campaigns", clientId, workspaceId, "analytics-filter"],
    enabled: !!clientId && !!workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns", { params: { path: scope, query: {} } });
      if (error || !data) throw apiError(error, "Không thể tải campaign để lọc analytics.");
      return data;
    },
  });
  const summary = useQuery({
    queryKey: ["analytics-summary", clientId, workspaceId, from, to, campaignId],
    enabled: !!clientId && !!workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/analytics/summary", { params: { path: scope, query } });
      if (error || !data) throw apiError(error, "Không thể tổng hợp analytics.");
      return data;
    },
  });
  const recommendations = useQuery({
    queryKey: ["analytics-recommendations", clientId, workspaceId, campaignId],
    enabled: !!clientId && !!workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/analytics/recommendations", { params: { path: scope, query: { campaignId: campaignId || undefined } } });
      if (error || !data) throw apiError(error, "Không thể tải recommendation.");
      return data;
    },
  });
  const generate = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/analytics/recommendations", { params: { path: scope, query: { campaignId: campaignId || undefined } } });
      if (error || !data) throw apiError(error, "Không thể tạo recommendation.");
      return data;
    },
    onSuccess: async () => qc.invalidateQueries({ queryKey: ["analytics-recommendations", clientId, workspaceId] }),
  });
  const maxDaily = useMemo(() => Math.max(1, ...(summary.data?.daily.map((item) => item.adImpressions + item.socialImpressions) ?? [1])), [summary.data]);
  if (!clientId || !workspaceId) return <><PageHeader eyebrow="Measurement" title="Analytics & Learning" description="Chi phí, chất lượng video, hiệu quả social và Ads được tổng hợp từ số đếm gốc." /><StatePanel title="Chưa chọn workspace">Mở một workspace để xem dữ liệu có scope.</StatePanel></>;
  return <>
    <PageHeader eyebrow="Measurement" title="Analytics & Learning" description="Derived metrics luôn được tính từ raw counts. Recommendation chỉ là bằng chứng để con người review, không bao giờ tự đổi budget hay trạng thái Ads." action={<Button disabled={generate.isPending} onClick={() => generate.mutate()}><Lightbulb className="mr-2 size-4" />{generate.isPending ? "Đang phân tích…" : "Tạo recommendation"}</Button>} />
    <Card className="mb-6 grid gap-4 p-5 md:grid-cols-3">
      <Field label="Từ ngày"><input className={inputClass} type="date" value={from} onChange={(event) => setFrom(event.target.value)} /></Field>
      <Field label="Đến ngày"><input className={inputClass} type="date" value={to} onChange={(event) => setTo(event.target.value)} /></Field>
      <Field label="Campaign (tùy chọn)"><select className={inputClass} value={campaignId} disabled={campaigns.isLoading} onChange={(event) => setCampaignId(event.target.value)}><option value="">Tất cả campaign</option>{includeCurrentOption((campaigns.data?.items ?? []).map((item) => ({ value: item.id, label: `${item.name} · ${item.status}` })), campaignId).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>{campaigns.error ? <span role="alert" className="text-xs font-semibold text-[var(--coral)]">{campaigns.error.message}</span> : null}</Field>
    </Card>
    {summary.isLoading ? <SkeletonRows /> : summary.error ? <StatePanel title="Không thể tải analytics" tone="danger">{summary.error.message}</StatePanel> : summary.data ? <>
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="Chi phí provider" value={usd.format(summary.data.totalCostUsd)} detail={`${summary.data.costs.length} nhóm chi phí`} />
        <Metric label="Ads CTR" value={`${number.format(summary.data.ads.ctr)}%`} detail={`${number.format(summary.data.ads.clicks)} / ${number.format(summary.data.ads.impressions)} clicks`} />
        <Metric label="ROAS" value={`${number.format(summary.data.ads.roas)}×`} detail={`${number.format(summary.data.ads.conversions)} conversions`} />
        <Metric label="Video attempts" value={`${number.format(summary.data.video.attemptFactor)}×`} detail={`${summary.data.video.rejectedScenes} rejected · ${summary.data.video.regeneratedScenes} regenerated`} />
      </div>
      <Card className="mb-6 p-5"><div className="flex flex-wrap items-center gap-3"><div className="mr-auto"><h2 className="font-serif text-xl font-bold">Production cost composition</h2><p className="mt-1 text-xs text-[var(--muted)]">LLM, Seedance, transcription, render và storage từ usage ledger; không gồm media spend.</p></div>{summary.data.costs.length ? summary.data.costs.map((item) => <span key={item.category} className="rounded-2xl bg-[#edf0e7] px-4 py-3 text-sm"><strong>{item.category}</strong> · {usd.format(item.usd)}</span>) : <Badge>Chưa có cost record</Badge>}</div><p className="mt-4 text-xs text-[var(--muted)]">Generation trung bình {number.format(summary.data.video.averageGenerationSeconds)}s · review {number.format(summary.data.video.averageReviewSeconds)}s · template success {summary.data.video.templateSuccessRate === null ? "chưa đủ mẫu" : `${number.format(summary.data.video.templateSuccessRate)}%`}</p></Card>
      <div className="mb-6 grid gap-6 xl:grid-cols-[1.25fr_.75fr]">
        <Card className="p-6"><h2 className="font-serif text-xl font-bold">Xu hướng theo ngày</h2><p className="mt-1 text-xs text-[var(--muted)]">Độ dài thanh = social + Ads impressions. Không nội suy ngày thiếu dữ liệu.</p><div className="mt-5 grid gap-3">{summary.data.daily.map((item) => <div key={item.date} className="grid grid-cols-[6.5rem_1fr_auto] items-center gap-3 text-xs"><span className="font-semibold">{item.date}</span><span className="h-3 overflow-hidden rounded-full bg-[#edf0e7]"><span className="block h-full rounded-full bg-[var(--moss)]" style={{ width: `${Math.max(1, ((item.adImpressions + item.socialImpressions) / maxDaily) * 100)}%` }} /></span><span className="text-right text-[var(--muted)]">{number.format(item.adImpressions + item.socialImpressions)} imp · {usd.format(item.costUsd)}</span></div>)}</div></Card>
        <Card className="p-6"><h2 className="font-serif text-xl font-bold">Funnel thật</h2><dl className="mt-5 grid gap-3">{[["Reach", summary.data.ads.reach], ["Impressions", summary.data.ads.impressions], ["Clicks", summary.data.ads.clicks], ["Conversions", summary.data.ads.conversions], ["Purchases", summary.data.ads.purchases]].map(([label, value]) => <div key={String(label)} className="flex justify-between border-b border-[var(--line)] pb-3 text-sm"><dt className="text-[var(--muted)]">{label}</dt><dd className="font-bold">{number.format(Number(value))}</dd></div>)}</dl></Card>
      </div>
      <Card className="mb-6 overflow-hidden"><div className="p-6"><h2 className="font-serif text-xl font-bold">Creative comparison</h2><p className="mt-1 text-sm text-[var(--muted)]">So sánh format, CTA và duration bằng cùng raw delivery window.</p></div><div className="overflow-x-auto"><table className="w-full min-w-[760px] text-left text-sm"><thead className="bg-[#edf0e7] text-xs uppercase tracking-wide text-[var(--muted)]"><tr>{["Format", "CTA", "Duration", "Campaigns", "Impressions", "CTR", "ROAS"].map((label) => <th key={label} className="px-5 py-3">{label}</th>)}</tr></thead><tbody>{summary.data.creativeComparisons.map((item) => <tr key={`${item.videoFormat}-${item.cta}-${item.durationSeconds}`} className="border-t border-[var(--line)]"><td className="px-5 py-4 font-semibold">{item.videoFormat}</td><td className="px-5 py-4">{item.cta}</td><td className="px-5 py-4">{item.durationSeconds}s</td><td className="px-5 py-4">{item.campaigns}</td><td className="px-5 py-4">{number.format(item.impressions)}</td><td className="px-5 py-4">{number.format(item.ctr)}%</td><td className="px-5 py-4">{number.format(item.roas)}×</td></tr>)}</tbody></table></div></Card>
    </> : null}
    <section><div className="mb-4 flex items-end justify-between"><div><h2 className="font-serif text-2xl font-bold">Learning recommendations</h2><p className="mt-1 text-sm text-[var(--muted)]">Review lưu quyết định; action Ads vẫn cần workflow và confirmation riêng.</p></div><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void recommendations.refetch()}><RefreshCw className="mr-2 size-4" />Làm mới</Button></div>{generate.error ? <StatePanel title="Không thể phân tích" tone="danger">{generate.error.message}</StatePanel> : recommendations.isLoading ? <SkeletonRows /> : recommendations.data?.items.length ? <div className="grid gap-4">{recommendations.data.items.map((item) => <RecommendationCard key={item.id} item={item} clientId={clientId} workspaceId={workspaceId} />)}</div> : <StatePanel title="Chưa có recommendation">Chỉ tạo recommendation khi dữ liệu đạt ngưỡng tối thiểu; hệ thống không bịa tỷ lệ từ mẫu thiếu.</StatePanel>}</section>
  </>;
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <Card className="p-5"><p className="text-xs font-bold uppercase tracking-[0.14em] text-[var(--muted)]">{label}</p><p className="mt-3 font-serif text-3xl font-bold">{value}</p><p className="mt-2 text-xs text-[var(--muted)]">{detail}</p></Card>;
}

function RecommendationCard({ item, clientId, workspaceId }: { item: Recommendation; clientId: string; workspaceId: string }) {
  const qc = useQueryClient();
  const [notes, setNotes] = useState("");
  const review = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/analytics/recommendations/{recommendationId}/review", { params: { path: { clientId, workspaceId, recommendationId: item.id } }, body: { action, version: item.version, notes } }); if (error || !data) throw apiError(error, "Không thể lưu review."); return data; }, onSuccess: async () => qc.invalidateQueries({ queryKey: ["analytics-recommendations", clientId, workspaceId] }) });
  return <Card className="p-5"><div className="flex flex-wrap items-center gap-2"><Badge tone={item.status === "APPROVED" ? "good" : item.status === "REJECTED" ? "danger" : "warn"}>{item.status}</Badge><span className="text-xs font-bold text-[var(--moss)]">{item.type}</span><span className="ml-auto text-xs text-[var(--muted)]">{item.model}</span></div><h3 className="mt-4 font-serif text-xl font-bold">{item.output}</h3><p className="mt-2 text-sm leading-6 text-[var(--muted)]">{item.rationale}</p>{item.status === "DRAFT" ? <div className="mt-4 flex flex-col gap-3 md:flex-row"><input aria-label="Ghi chú review" className={inputClass} value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Ghi chú review (bắt buộc khi từ chối)" /><Button disabled={review.isPending} onClick={() => review.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt insight</Button><Button className="bg-[var(--coral)]" disabled={review.isPending || !notes.trim()} onClick={() => review.mutate("REJECT")}><X className="mr-2 size-4" />Từ chối</Button></div> : <p className="mt-4 rounded-2xl bg-[#edf0e7] p-3 text-xs text-[var(--muted)]">{item.reviewNotes || "Đã review"} · Không có action được tự động thực thi.</p>}{review.error ? <p className="mt-3 text-sm text-[var(--coral)]">{review.error.message}</p> : null}</Card>;
}
