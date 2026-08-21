"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, ExternalLink, ImageIcon, Star, Trash2, Upload, X } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { MediaAssetPreview } from "@/components/media-asset-preview";
import { MediaUploader, type MediaScope } from "@/components/media-uploader";
import { Badge, Button, Card, SkeletonRows, StatePanel } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";
import { studioRoutes } from "@/lib/studio-routes";

type Asset = components["schemas"]["MediaAsset"];
type AssetStatus = components["schemas"]["ContentStatus"];

export function isEligibleBrandLogo(asset: Asset, now = Date.now()) {
  return (asset.assetType === "LOGO" || asset.assetType === "IMAGE")
    && asset.status === "APPROVED"
    && asset.readyForUse
    && Boolean(asset.mimeType?.match(/^image\/(jpeg|png|webp)$/))
    && (!asset.expiresAt || new Date(asset.expiresAt).getTime() > now);
}

export function makePrimaryLogo(ids: string[], id: string) {
  return ids.includes(id) ? [id, ...ids.filter((current) => current !== id)] : ids;
}

export function BrandLogoPanel({ brandId, scope, value, onChange, onEligibilityChange }: { brandId: string; scope: MediaScope; value: string[]; onChange: (ids: string[]) => void; onEligibilityChange: (valid: boolean) => void }) {
  const { canOperate, canReview } = usePermissions();
  const qc = useQueryClient();
  const [showUpload, setShowUpload] = useState(false);
  const [now] = useState(() => Date.now());
  const library = useQuery({
    queryKey: ["media", scope.clientId, scope.workspaceId, "brand-logo-candidates"],
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể tải logo từ Media Library.");
      return data;
    },
  });
  const refresh = useCallback(async () => qc.invalidateQueries({ queryKey: ["media", scope.clientId, scope.workspaceId] }), [qc, scope.clientId, scope.workspaceId]);
  const status = useMutation({
    mutationFn: async ({ asset, next }: { asset: Asset; next: AssetStatus }) => {
      const { data, error } = await api.PATCH("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/status", { params: { path: { ...scope, assetId: asset.id } }, body: { status: next, version: asset.version } });
      if (error || !data) throw apiError(error, "Không thể đổi trạng thái logo; asset có thể đang được sử dụng hoặc đã thay đổi.");
      return data;
    },
    onSuccess: refresh,
  });
  const candidates = useMemo(() => library.data?.items.filter((asset) => {
    const visual = asset.assetType === "LOGO" || asset.assetType === "IMAGE";
    const availableScope = !asset.productId && !asset.campaignId && (!asset.brandId || asset.brandId === brandId);
    return value.includes(asset.id) || (visual && availableScope);
  }) ?? [], [brandId, library.data, value]);
  const byId = useMemo(() => new Map(candidates.map((asset) => [asset.id, asset])), [candidates]);
  const selected = value.map((id) => byId.get(id)).filter((asset): asset is Asset => Boolean(asset));
  const available = candidates.filter((asset) => !value.includes(asset.id));
  const missingCount = value.length - selected.length;
  const valid = missingCount === 0 && selected.every((asset) => isEligibleBrandLogo(asset, now));

  useEffect(() => onEligibilityChange(value.length === 0 || (!library.isLoading && !library.error && valid)), [library.error, library.isLoading, onEligibilityChange, valid, value.length]);

  const toggle = (asset: Asset) => {
    if (!canOperate || (!value.includes(asset.id) && !isEligibleBrandLogo(asset, now))) return;
    onChange(value.includes(asset.id) ? value.filter((id) => id !== asset.id) : [...value, asset.id].slice(0, 20));
  };

  return <section aria-labelledby="brand-logo-title" className="mb-6">
    <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
      <div><h2 id="brand-logo-title" className="font-serif text-2xl font-bold">Logo thương hiệu</h2><p className="mt-1 max-w-3xl text-sm text-[var(--muted)]">Logo chính đứng đầu danh sách và được renderer sử dụng. File vẫn được quản lý, duyệt và lưu một lần trong Media Library.</p></div>
      <div className="flex flex-wrap gap-2"><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={library.isFetching} onClick={() => void library.refetch()}>Làm mới</Button><Link className="inline-flex min-h-11 items-center rounded-full bg-white px-4 text-sm font-bold ring-1 ring-[var(--line)]" href={studioRoutes.media(scope.clientId, scope.workspaceId)}>Media Library<ExternalLink className="ml-2 size-4" /></Link>{canOperate ? <Button onClick={() => setShowUpload((current) => !current)}>{showUpload ? <X className="mr-2 size-4" /> : <Upload className="mr-2 size-4" />}{showUpload ? "Đóng upload" : "Upload logo"}</Button> : null}</div>
    </div>
    {showUpload ? <div className="mb-5"><MediaUploader key={`brand-logo:${brandId}`} scope={scope} brandId={brandId} logoOnly onUploaded={refresh} /></div> : null}
    <Card className="mb-5 p-5">
      <div className="flex flex-wrap items-center gap-3"><div className="flex-1"><h3 className="font-bold">Logo đã chọn</h3><p className="mt-1 text-xs text-[var(--muted)]">Chọn tối đa 20 biến thể. Thay đổi chỉ có hiệu lực sau khi lưu phiên bản Brand.</p></div><Badge tone={valid ? "good" : "warn"}>{value.length === 0 ? "Chưa chọn logo" : valid ? `${value.length} logo sẵn sàng` : "Cần xử lý"}</Badge></div>
      {missingCount > 0 ? <div role="alert" className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[#e8c98f] bg-[#fff8e8] p-3 text-sm"><span>{missingCount} logo tham chiếu không còn khả dụng trong Media Library.</span>{canOperate ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => onChange(selected.map((asset) => asset.id))}>Gỡ tham chiếu lỗi</Button> : null}</div> : null}
      {selected.length === 0 ? <p className="mt-4 rounded-xl border border-dashed border-[var(--line)] p-4 text-sm text-[var(--muted)]">Chưa có logo trong phiên bản này. Upload hoặc chọn một asset đã duyệt bên dưới.</p> : <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">{selected.map((asset, index) => <LogoCard key={asset.id} asset={asset} scope={scope} now={now} selected primary={index === 0} canOperate={canOperate} canReview={canReview} busy={status.isPending} onToggle={() => toggle(asset)} onPrimary={() => onChange(makePrimaryLogo(value, asset.id))} onStatus={(next) => status.mutate({ asset, next })} />)}</div>}
    </Card>
    {library.isLoading ? <SkeletonRows /> : library.error ? <StatePanel title="Không thể tải Media Library" tone="danger"><p>{library.error.message}</p><Button className="mt-3" onClick={() => void library.refetch()}>Thử lại</Button></StatePanel> : available.length === 0 ? <StatePanel title={selected.length > 0 ? "Không còn logo khả dụng khác" : "Chưa có ảnh logo"}>{selected.length > 0 ? "Các logo phù hợp hiện đã được chọn. Bạn có thể upload thêm biến thể." : "Upload PNG, WebP hoặc JPEG tại đây. Logo mới cần được Reviewer/Admin duyệt trước khi chọn."}</StatePanel> : <div><h3 className="mb-3 font-bold">Chọn từ Media Library</h3><div className="grid max-h-[42rem] gap-4 overflow-auto pr-1 md:grid-cols-2 xl:grid-cols-3">{available.map((asset) => <LogoCard key={asset.id} asset={asset} scope={scope} now={now} selected={false} primary={false} canOperate={canOperate} canReview={canReview} busy={status.isPending} onToggle={() => toggle(asset)} onPrimary={() => onChange(makePrimaryLogo(value, asset.id))} onStatus={(next) => status.mutate({ asset, next })} />)}</div></div>}
    {status.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{status.error.message}</p> : null}
  </section>;
}

function LogoCard({ asset, scope, now, selected, primary, canOperate, canReview, busy, onToggle, onPrimary, onStatus }: { asset: Asset; scope: MediaScope; now: number; selected: boolean; primary: boolean; canOperate: boolean; canReview: boolean; busy: boolean; onToggle: () => void; onPrimary: () => void; onStatus: (status: AssetStatus) => void }) {
  const eligible = isEligibleBrandLogo(asset, now);
  const tone = asset.status === "APPROVED" ? "good" : asset.status === "REJECTED" ? "danger" : "warn";
  return <Card className={`overflow-hidden ${selected ? "border-[var(--moss)] ring-2 ring-[#26664f22]" : ""}`}>
    <MediaAssetPreview asset={asset} scope={scope} fit="contain" className="aspect-[3/2] bg-white" />
    <div className="p-4"><div className="flex items-start gap-2"><span className="grid size-9 shrink-0 place-items-center rounded-xl bg-[#edf0e7]"><ImageIcon className="size-4" /></span><div className="min-w-0 flex-1"><h4 className="truncate font-bold">{asset.name}</h4><p className="mt-1 truncate text-xs text-[var(--muted)]">{asset.category || "BRAND_LOGO"}</p></div><Badge tone={tone}>{asset.status}</Badge></div>
      <div className="mt-3 flex flex-wrap gap-1">{primary ? <Badge tone="good">Logo chính</Badge> : selected ? <Badge>Logo thay thế</Badge> : null}{!asset.readyForUse ? <Badge tone="warn">Đang xử lý</Badge> : null}{asset.expiresAt ? <Badge tone={new Date(asset.expiresAt).getTime() > now ? "neutral" : "danger"}>Hạn {new Date(asset.expiresAt).toLocaleDateString("vi-VN")}</Badge> : null}</div>
      <div className="mt-4 flex flex-wrap gap-2">{canOperate ? <Button aria-pressed={selected} className={selected ? "bg-white px-3 text-[var(--coral)] ring-1 ring-[var(--line)]" : "px-3"} disabled={!selected && !eligible} onClick={onToggle}>{selected ? <Trash2 className="mr-1 size-4" /> : <Check className="mr-1 size-4" />}{selected ? "Gỡ" : "Chọn"}</Button> : null}{canOperate && selected && !primary ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={onPrimary}><Star className="mr-1 size-4" />Đặt làm chính</Button> : null}{canReview && asset.status !== "APPROVED" ? <Button className="px-3" disabled={busy || !asset.readyForUse} onClick={() => onStatus("APPROVED")}>Duyệt</Button> : null}{canReview && !selected && asset.status !== "REJECTED" ? <Button className="bg-[var(--coral)] px-3" disabled={busy} onClick={() => onStatus("REJECTED")}>Từ chối</Button> : null}</div>
    </div>
  </Card>;
}
