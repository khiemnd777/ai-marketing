"use client";

import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, ShieldCheck, XCircle } from "lucide-react";
import { Badge, Card, PageHeader, SkeletonRows, StatePanel } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

export default function ProvidersPage() {
  const status = useQuery({ queryKey: ["provider-status"], refetchInterval: 30_000, queryFn: async () => { const { data, error } = await api.GET("/operations/providers"); if (error || !data) throw apiError(error, "Không thể tải provider status."); return data; } });
  return <><PageHeader eyebrow="Admin only" title="Provider configuration" description="Trạng thái cấu hình an toàn cho OpenAI, Seedance, R2, Meta và renderer. API key, token và shared secret không được trả về trình duyệt." action={<Badge tone={status.data?.demoMode ? "warn" : "good"}>{status.data?.demoMode ? "DEMO MODE" : "LIVE MODE"}</Badge>} />
    {status.isLoading ? <SkeletonRows /> : status.error ? <StatePanel title="Không thể kiểm tra provider" tone="danger">{status.error.message}</StatePanel> : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{status.data?.providers.map((provider) => <Card key={provider.name} className="p-6"><div className="flex items-start justify-between"><span className="grid size-11 place-items-center rounded-2xl bg-[#edf0e7]">{provider.configured ? <CheckCircle2 className="size-5 text-[var(--moss)]" /> : <XCircle className="size-5 text-[var(--coral)]" />}</span><Badge tone={provider.configured ? "good" : "danger"}>{provider.configured ? "CONFIGURED" : "MISSING"}</Badge></div><h2 className="mt-5 font-serif text-2xl font-bold capitalize">{provider.name}</h2><dl className="mt-4 grid gap-2 text-sm">{provider.model ? <Row label="Model" value={provider.model} /> : null}{provider.apiVersion ? <Row label="API" value={provider.apiVersion} /> : null}{provider.bucket ? <Row label="Bucket" value={provider.bucket} /> : null}{provider.baseUrl ? <Row label="Endpoint" value={provider.baseUrl} /> : null}</dl></Card>)}</div>}
    <Card className="mt-6 flex gap-4 p-5"><ShieldCheck className="mt-0.5 size-5 shrink-0 text-[var(--moss)]" /><div><h2 className="font-semibold">Secret boundary</h2><p className="mt-1 text-sm leading-6 text-[var(--muted)]">Provider secrets chỉ được nạp qua environment của API/worker. Trang này cố ý chỉ hiển thị boolean, model/version, bucket và endpoint đã loại query/fragment.</p></div></Card>
  </>;
}

function Row({ label, value }: { label: string; value: string }) { return <div className="flex justify-between gap-4 border-b border-[var(--line)] pb-2"><dt className="text-[var(--muted)]">{label}</dt><dd className="max-w-[70%] truncate font-semibold" title={value}>{value}</dd></div>; }
