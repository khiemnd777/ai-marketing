"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Link2, RefreshCw, Unplug } from "lucide-react";
import { Suspense } from "react";
import { useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, PageHeader, SkeletonRows, StatePanel } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

function MetaSettingsContent() {
  const { clientId, workspaceId } = useStudioScope();
  const scope = { clientId, workspaceId };
  const queryClient = useQueryClient();
  const connection = useQuery({
    queryKey: ["meta-connection", clientId, workspaceId],
    enabled: !!clientId && !!workspaceId,
    retry: false,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/meta/connection", { params: { path: scope } });
      if (error || !data) return null;
      return data;
    },
  });
  const refresh = async () => queryClient.invalidateQueries({ queryKey: ["meta-connection", clientId, workspaceId] });
  const connect = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/meta/oauth/start", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể bắt đầu Meta OAuth.");
      return data;
    },
    onSuccess: (data) => window.location.assign(data.authorizationUrl),
  });
  const sync = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/meta/sync", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể đồng bộ Meta assets.");
      return data;
    },
    onSuccess: refresh,
  });
  const disconnect = useMutation({
    mutationFn: async () => {
      if (!window.confirm("Ngắt Meta sẽ xóa token đã mã hóa và dừng publishing/Ads mới. Tiếp tục?")) return false;
      const { error } = await api.DELETE("/clients/{clientId}/workspaces/{workspaceId}/meta/connection", { params: { path: scope } });
      if (error) throw apiError(error, "Không thể ngắt Meta.");
      return true;
    },
    onSuccess: async (changed) => { if (changed) await refresh(); },
  });
  if (!clientId || !workspaceId) return <><PageHeader eyebrow="Distribution" title="Kết nối Meta" description="OAuth, Page, Instagram và tài sản Ads được cô lập theo workspace." /><StatePanel title="Chưa chọn workspace">Mở Settings từ workspace với clientId và workspaceId.</StatePanel></>;
  if (connection.isLoading) return <SkeletonRows />;
  const item = connection.data;
  const expiry = item?.tokenExpiresAt ? new Date(item.tokenExpiresAt) : null;
  const expiring = item?.status === "EXPIRING";
  return <>
    <PageHeader eyebrow="Distribution" title="Kết nối Meta" description="Direct Graph API connection. Token chỉ được giải mã trong worker và không bao giờ trả về trình duyệt." action={<div className="flex flex-wrap gap-2">{item ? <><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={sync.isPending} onClick={() => sync.mutate()}><RefreshCw className="mr-2 size-4" />Đồng bộ</Button><Button className="bg-[var(--coral)]" disabled={disconnect.isPending} onClick={() => disconnect.mutate()}><Unplug className="mr-2 size-4" />Ngắt</Button></> : <Button disabled={connect.isPending} onClick={() => connect.mutate()}><Link2 className="mr-2 size-4" />Kết nối Meta</Button>}</div>} />
    {(connect.error || sync.error || disconnect.error) ? <StatePanel title="Meta operation thất bại" tone="danger">{connect.error?.message ?? sync.error?.message ?? disconnect.error?.message}</StatePanel> : null}
    {!item ? <StatePanel title="Chưa kết nối">Dùng OAuth chính thức của Meta để khám phá Page, Instagram Business, Business Manager, Ad Account, Pixel và Audience.</StatePanel> : <div className="grid gap-5">
      <Card className="p-6"><div className="flex flex-wrap items-center gap-2"><h2 className="font-serif text-2xl font-bold">{item.displayName}</h2><Badge tone={item.status === "CONNECTED" ? "good" : "warn"}>{item.status}</Badge><Badge>{item.apiVersion}</Badge></div><p className="mt-3 text-sm text-[var(--muted)]">Scopes: {item.scopes.join(", ")}</p>{expiry ? <p className="mt-2 text-sm text-[var(--muted)]">Token hết hạn: {expiry.toLocaleString("vi-VN")}</p> : null}{expiring ? <p className="mt-3 flex items-center gap-2 text-sm font-semibold text-[#79580b]"><AlertTriangle className="size-4" />Token sắp hết hạn; hãy kết nối lại trước khi publishing bị dừng.</p> : null}</Card>
      <Resource title="Page & Instagram" items={item.accounts.map((account) => `${account.platform} · ${account.name}${account.username ? ` · @${account.username}` : ""} · ${account.status}`)} />
      <Resource title="Business & Ad Account" items={[...item.businesses.map((business) => `Business · ${business.name}`), ...item.adAccounts.map((account) => `Ad Account · ${account.name} · ${account.currency} · ${account.timezoneName}`)]} />
      <Resource title="Pixel & Audience" items={[...item.pixels.map((pixel) => `Pixel · ${pixel.name}`), ...item.audiences.map((audience) => `Audience · ${audience.name}${audience.approximateCount ? ` · ~${audience.approximateCount.toLocaleString("vi-VN")}` : ""}`)]} />
    </div>}
  </>;
}

function Resource({ title, items }: { title: string; items: string[] }) {
  return <Card className="p-6"><h2 className="font-serif text-xl font-bold">{title}</h2>{items.length ? <ul className="mt-4 grid gap-2 text-sm text-[var(--muted)]">{items.map((item) => <li key={item} className="rounded-2xl bg-white px-4 py-3 ring-1 ring-[var(--line)]">{item}</li>)}</ul> : <p className="mt-3 text-sm text-[var(--muted)]">Không tìm thấy resource phù hợp.</p>}</Card>;
}

export default function MetaSettingsPage() {
  return <Suspense fallback={<SkeletonRows />}><MetaSettingsContent /></Suspense>;
}
