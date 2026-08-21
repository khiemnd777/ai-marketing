"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Download, Film, FolderOpen, ImageIcon, ListFilter, Music2, Pencil, Search, Trash2, X } from "lucide-react";
import { Suspense, useCallback, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { MediaAssetPreview } from "@/components/media-asset-preview";
import { MediaUploader, type MediaScope } from "@/components/media-uploader";
import { useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

type Asset = components["schemas"]["MediaAsset"];
type AssetType = components["schemas"]["MediaAssetType"];
type AssetStatus = components["schemas"]["ContentStatus"];

export default function MediaPage() {
  return <Suspense fallback={<SkeletonRows />}><MediaContent /></Suspense>;
}

function MediaContent() {
  const { clientId, workspaceId } = useStudioScope();
  const scope = useMemo(() => ({ clientId, workspaceId }), [clientId, workspaceId]);
  const { canOperate, canReview } = usePermissions();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [assetType, setAssetType] = useState<AssetType | "">("");
  const [status, setStatus] = useState<AssetStatus | "">("");
  const [productId, setProductId] = useState("");
  const [brandId, setBrandId] = useState("");
  const [editing, setEditing] = useState<Asset | null>(null);
  const products = useQuery({ queryKey: ["products", clientId, workspaceId], enabled: !!clientId && !!workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/products", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải danh sách sản phẩm."); return data; } });
  const brands = useQuery({ queryKey: ["brands", clientId, workspaceId], enabled: !!clientId && !!workspaceId, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/brands", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải danh sách thương hiệu."); return data; } });
  const assets = useQuery({
    queryKey: ["media", clientId, workspaceId, search, assetType, status, productId],
    enabled: !!clientId && !!workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets", { params: { path: scope, query: { search: search || undefined, assetType: assetType || undefined, status: status || undefined, productId: productId || undefined } } });
      if (error || !data) throw apiError(error, "Không thể tải media.");
      return data;
    },
  });
  const productNames = useMemo(() => new Map(products.data?.items.map((product) => [product.id, product.name]) ?? []), [products.data]);
  const brandNames = useMemo(() => new Map(brands.data?.items.map((brand) => [brand.id, brand.name]) ?? []), [brands.data]);
  const logoBrands = useMemo(() => {
    const result = new Map<string, string[]>();
    for (const brand of brands.data?.items ?? []) for (const assetId of brand.logoAssetIds) result.set(assetId, [...(result.get(assetId) ?? []), brand.name]);
    return result;
  }, [brands.data]);
  const visibleAssets = useMemo(() => assets.data?.items.filter((asset) => !brandId || asset.brandId === brandId || (brands.data?.items.find((brand) => brand.id === brandId)?.logoAssetIds ?? []).includes(asset.id)) ?? [], [assets.data, brandId, brands.data]);
  const refresh = useCallback(async () => qc.invalidateQueries({ queryKey: ["media", clientId, workspaceId] }), [qc, clientId, workspaceId]);

  if (!clientId || !workspaceId) return <><PageHeader eyebrow="Tài sản" title="Thư viện media" description="Chọn workspace để upload trực tiếp vào vùng lưu trữ được cô lập." /><StatePanel title="Chưa chọn workspace">Chọn client và workspace trong thanh điều hướng, sau đó quay lại Media.</StatePanel></>;
  return <>
    <PageHeader eyebrow="Kho dùng chung" title="Thư viện media" description="Một nguồn asset cho toàn workspace. Product Media chỉ là view theo sản phẩm; mọi thay đổi trạng thái và metadata đều đồng bộ tại đây." />
    {canOperate ? <MediaUploader key={`${clientId}:${workspaceId}`} scope={scope} onUploaded={refresh} /> : <StatePanel title="Chế độ chỉ xem">Reviewer có thể duyệt hoặc từ chối asset; chỉ Operator/Admin được upload và sửa metadata.</StatePanel>}
    <Card className="mb-6 mt-6 p-4"><div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3"><label className="flex min-h-11 items-center gap-3 rounded-xl border border-[var(--line)] bg-white px-4 md:col-span-2 xl:col-span-1"><Search className="size-4 text-[var(--muted)]" /><input aria-label="Tìm media" className="w-full bg-transparent text-sm outline-none" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tên hoặc tag" /></label><select aria-label="Lọc loại media" className={inputClass} value={assetType} onChange={(event) => setAssetType(event.target.value as AssetType | "")}><option value="">Mọi loại</option>{["IMAGE", "VIDEO", "AUDIO", "LOGO", "BROCHURE", "SCREENSHOT", "SCREEN_RECORDING"].map((value) => <option key={value}>{value}</option>)}</select><select aria-label="Lọc trạng thái media" className={inputClass} value={status} onChange={(event) => setStatus(event.target.value as AssetStatus | "")}><option value="">Mọi trạng thái</option>{["DRAFT", "APPROVED", "REJECTED", "ARCHIVED"].map((value) => <option key={value}>{value}</option>)}</select><select aria-label="Lọc theo sản phẩm" className={inputClass} value={productId} onChange={(event) => setProductId(event.target.value)}><option value="">Mọi sản phẩm</option>{products.data?.items.map((product) => <option key={product.id} value={product.id}>{product.name}</option>)}</select><select aria-label="Lọc theo thương hiệu" className={inputClass} value={brandId} onChange={(event) => setBrandId(event.target.value)}><option value="">Mọi thương hiệu</option>{brands.data?.items.map((brand) => <option key={brand.id} value={brand.id}>{brand.name}</option>)}</select><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void assets.refetch()}><ListFilter className="mr-2 size-4" />Làm mới</Button></div></Card>
    {editing ? <AssetEditor asset={editing} scope={scope} onClose={() => setEditing(null)} onSaved={async () => { setEditing(null); await refresh(); }} /> : null}
    {assets.isLoading ? <SkeletonRows /> : assets.error ? <StatePanel title="Không thể tải media" tone="danger">{assets.error.message}</StatePanel> : visibleAssets.length === 0 ? <StatePanel title="Không có asset phù hợp">Thay đổi bộ lọc hoặc thêm logo, hero image, packshot và product footage.</StatePanel> : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{visibleAssets.map((asset) => <AssetCard key={asset.id} asset={asset} productName={asset.productId ? productNames.get(asset.productId) : undefined} owningBrandName={asset.brandId ? brandNames.get(asset.brandId) : undefined} logoBrandNames={logoBrands.get(asset.id) ?? []} scope={scope} canOperate={canOperate} canReview={canReview} onEdit={() => setEditing(asset)} refresh={refresh} />)}</div>}
  </>;
}

function AssetCard({ asset, productName, owningBrandName, logoBrandNames, scope, canOperate, canReview, onEdit, refresh }: { asset: Asset; productName?: string; owningBrandName?: string; logoBrandNames: string[]; scope: MediaScope; canOperate: boolean; canReview: boolean; onEdit: () => void; refresh: () => Promise<unknown> }) {
  const status = useMutation({ mutationFn: async (next: AssetStatus) => { const { data, error } = await api.PATCH("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/status", { params: { path: { ...scope, assetId: asset.id } }, body: { status: next, version: asset.version } }); if (error || !data) throw apiError(error, "Asset đã thay đổi; hãy làm mới trước khi duyệt."); return data; }, onSuccess: refresh });
  const remove = useMutation({ mutationFn: async () => { if (!window.confirm(`Xóa mềm asset “${asset.name}”?`)) return false; const { error } = await api.DELETE("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}", { params: { path: { ...scope, assetId: asset.id } }, body: { version: asset.version } }); if (error) throw apiError(error, "Không thể xóa asset đang được sử dụng hoặc đã thay đổi."); return true; }, onSuccess: (deleted) => deleted ? refresh() : undefined });
  const download = async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { ...scope, assetId: asset.id } } }); if (error || !data) throw apiError(error, "Không thể tạo link tải."); window.open(data.url, "_blank", "noopener,noreferrer"); };
  const tone = asset.status === "APPROVED" ? "good" : asset.status === "REJECTED" ? "danger" : asset.status === "DRAFT" ? "warn" : "neutral";
  return <Card className="overflow-hidden"><MediaAssetPreview asset={asset} scope={scope} fit={asset.assetType === "LOGO" ? "contain" : "cover"} /><div className="p-5"><div className="flex items-start gap-3"><span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]">{asset.assetType === "VIDEO" || asset.assetType === "SCREEN_RECORDING" ? <Film className="size-5" /> : asset.assetType === "AUDIO" ? <Music2 className="size-5" /> : <ImageIcon className="size-5" />}</span><div className="min-w-0 flex-1"><h2 className="truncate font-serif text-lg font-bold">{asset.name}</h2><p className="mt-1 text-xs text-[var(--muted)]">{asset.category || "Chưa phân loại"} · {asset.fileSizeBytes ? formatBytes(asset.fileSizeBytes) : "Đang xử lý"}</p></div><Badge tone={tone}>{asset.status}</Badge></div><div className="mt-3 flex flex-wrap gap-1">{productName ? <Badge tone="good">Sản phẩm: {productName}</Badge> : <Badge>Chưa gắn sản phẩm</Badge>}{owningBrandName ? <Badge>Thương hiệu: {owningBrandName}</Badge> : null}{logoBrandNames.map((name) => <Badge key={name} tone="good">Logo của: {name}</Badge>)}{!asset.readyForUse ? <Badge tone="warn">Đang xử lý</Badge> : null}{asset.folder ? <Badge><FolderOpen className="mr-1 inline size-3" />{asset.folder}</Badge> : null}{asset.tags.map((tag) => <Badge key={tag}>{tag}</Badge>)}</div>{asset.expiresAt ? <p className="mt-3 text-xs text-[var(--muted)]">Hết hạn {new Date(asset.expiresAt).toLocaleDateString("vi-VN")}</p> : null}{(status.error || remove.error) ? <p role="alert" className="mt-3 text-xs text-[var(--coral)]">{status.error?.message ?? remove.error?.message}</p> : null}<div className="mt-4 flex flex-wrap gap-2">{asset.mimeType ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void download()}><Download className="mr-1 size-4" />Mở</Button> : null}{canOperate ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={onEdit}><Pencil className="mr-1 size-4" />Sửa</Button> : null}{canReview && asset.status !== "APPROVED" ? <Button className="px-3" disabled={status.isPending || !asset.readyForUse} onClick={() => status.mutate("APPROVED")}><Check className="mr-1 size-4" />Duyệt</Button> : null}{canReview && logoBrandNames.length === 0 && asset.status !== "REJECTED" ? <Button className="bg-[var(--coral)] px-3" disabled={status.isPending} onClick={() => status.mutate("REJECTED")}><X className="mr-1 size-4" />Từ chối</Button> : null}{canOperate ? <Button aria-label={`Xóa ${asset.name}`} className="bg-white px-3 text-[var(--coral)] ring-1 ring-[var(--line)]" disabled={remove.isPending} onClick={() => remove.mutate()}><Trash2 className="size-4" /></Button> : null}</div></div></Card>;
}

function AssetEditor({ asset, scope, onClose, onSaved }: { asset: Asset; scope: MediaScope; onClose: () => void; onSaved: () => Promise<unknown> }) {
  const [form, setForm] = useState({ name: asset.name, category: asset.category, folder: asset.folder, usageRights: asset.usageRights, tags: asset.tags.join(", "), expiresAt: asset.expiresAt ? asset.expiresAt.slice(0, 10) : "" });
  const save = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}", { params: { path: { ...scope, assetId: asset.id } }, body: { name: form.name, category: form.category.trim().toUpperCase(), folder: form.folder, usageRights: form.usageRights, tags: form.tags.split(",").map((tag) => tag.trim()).filter(Boolean), expiresAt: form.expiresAt ? new Date(`${form.expiresAt}T23:59:59Z`).toISOString() : null, version: asset.version } }); if (error || !data) throw apiError(error, "Metadata không hợp lệ hoặc asset đã thay đổi."); return data; }, onSuccess: onSaved });
  return <Card className="mb-6 border-[var(--moss)] p-6"><div className="mb-5 flex items-center justify-between"><div><h2 className="font-serif text-xl font-bold">Sửa metadata</h2><p className="mt-1 text-sm text-[var(--muted)]">{asset.name} · optimistic version {asset.version}</p></div><button aria-label="Đóng trình sửa" className="rounded-full p-2 hover:bg-[#edf0e7]" onClick={onClose}><X className="size-5" /></button></div><div className="grid gap-4 md:grid-cols-2"><Field label="Tên"><input className={inputClass} value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} /></Field><Field label="Danh mục"><input className={inputClass} value={form.category} onChange={(event) => setForm((current) => ({ ...current, category: event.target.value }))} /></Field><Field label="Folder"><input className={inputClass} value={form.folder} onChange={(event) => setForm((current) => ({ ...current, folder: event.target.value }))} /></Field><Field label="Tags (phân cách dấu phẩy)"><input className={inputClass} value={form.tags} onChange={(event) => setForm((current) => ({ ...current, tags: event.target.value }))} /></Field><Field label="Ngày hết hạn"><input className={inputClass} type="date" value={form.expiresAt} onChange={(event) => setForm((current) => ({ ...current, expiresAt: event.target.value }))} /></Field><div className="md:col-span-2"><Field label="Quyền sử dụng"><textarea className={textareaClass} value={form.usageRights} onChange={(event) => setForm((current) => ({ ...current, usageRights: event.target.value }))} /></Field></div></div>{save.error ? <p role="alert" className="mt-3 text-sm text-[var(--coral)]">{save.error.message}</p> : null}<div className="mt-5 flex gap-3"><Button disabled={save.isPending || !form.name.trim() || !form.usageRights.trim()} onClick={() => save.mutate()}><Check className="mr-2 size-4" />Lưu metadata</Button><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={onClose}>Hủy</Button></div></Card>;
}

function formatBytes(value: number) {
  if (value < 1024 * 1024) return `${Math.max(1, Math.round(value / 1024))} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
  return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`;
}
