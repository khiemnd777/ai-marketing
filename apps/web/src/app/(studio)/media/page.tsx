"use client";

import type { Body, Meta, UppyFile } from "@uppy/core";
import Uppy from "@uppy/core";
import GoldenRetriever from "@uppy/golden-retriever";
import Dashboard from "@uppy/react/dashboard";
import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Download, Film, FolderOpen, ImageIcon, ListFilter, Music2, Pencil, Search, Trash2, X } from "lucide-react";
import Image from "next/image";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type Asset = components["schemas"]["MediaAsset"];
type AssetType = components["schemas"]["MediaAssetType"];
type AssetStatus = components["schemas"]["ContentStatus"];
type UploadSession = components["schemas"]["MediaUploadSession"];
type PresignedRequest = components["schemas"]["PresignedRequest"];
type Scope = { clientId: string; workspaceId: string };
type UploadedPart = { partNumber: number; etag: string };
type ResumeState = { session: UploadSession; completedParts: UploadedPart[] };
type MediaMeta = Meta & { displayName?: string; folder?: string; usageRights?: string; tags?: string; uploadKey?: string; resumeState?: string };
type MediaBody = Body & { assetId?: string };

const chunkSize = 10 * 1024 * 1024;
const acceptedTypes = ["image/jpeg", "image/png", "image/webp", "video/mp4", "video/quicktime", "audio/mpeg", "audio/wav", "audio/x-wav", "application/pdf"];

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
  const [editing, setEditing] = useState<Asset | null>(null);
  const assets = useQuery({
    queryKey: ["media", scope.clientId, scope.workspaceId, search, assetType, status],
    enabled: !!scope.clientId && !!scope.workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets", { params: { path: scope, query: { search: search || undefined, assetType: assetType || undefined, status: status || undefined } } });
      if (error || !data) throw apiError(error, "Không thể tải media.");
      return data;
    },
  });
  const refresh = useCallback(async () => qc.invalidateQueries({ queryKey: ["media", scope.clientId, scope.workspaceId] }), [qc, scope.clientId, scope.workspaceId]);

  if (!scope.clientId || !scope.workspaceId) return <><PageHeader eyebrow="Tài sản" title="Thư viện media" description="Chọn workspace để upload trực tiếp vào vùng lưu trữ được cô lập." /><StatePanel title="Chưa chọn workspace">Chọn client và workspace trong thanh điều hướng, sau đó quay lại Media.</StatePanel></>;
  return <>
    <PageHeader eyebrow="R2 trực tiếp" title="Thư viện media" description="Queue đa tệp dùng Uppy; file đi thẳng tới object storage bằng URL ký, multipart có retry từng phần, sau đó server mới xác minh metadata." />
    {canOperate ? <MediaUploader key={`${scope.clientId}:${scope.workspaceId}`} scope={scope} onUploaded={refresh} /> : <StatePanel title="Chế độ chỉ xem">Reviewer có thể duyệt hoặc từ chối asset; chỉ Operator/Admin được upload và sửa metadata.</StatePanel>}
    <Card className="mb-6 mt-6 p-4"><div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_220px_220px_auto]"><label className="flex min-h-11 items-center gap-3 rounded-xl border border-[var(--line)] bg-white px-4"><Search className="size-4 text-[var(--muted)]" /><input aria-label="Tìm media" className="w-full bg-transparent text-sm outline-none" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tên hoặc tag" /></label><select aria-label="Lọc loại media" className={inputClass} value={assetType} onChange={(event) => setAssetType(event.target.value as AssetType | "")}><option value="">Mọi loại</option>{["IMAGE", "VIDEO", "AUDIO", "LOGO", "BROCHURE", "SCREENSHOT", "SCREEN_RECORDING"].map((value) => <option key={value}>{value}</option>)}</select><select aria-label="Lọc trạng thái media" className={inputClass} value={status} onChange={(event) => setStatus(event.target.value as AssetStatus | "")}><option value="">Mọi trạng thái</option>{["DRAFT", "APPROVED", "REJECTED", "ARCHIVED"].map((value) => <option key={value}>{value}</option>)}</select><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void assets.refetch()}><ListFilter className="mr-2 size-4" />Làm mới</Button></div></Card>
    {editing ? <AssetEditor asset={editing} scope={scope} onClose={() => setEditing(null)} onSaved={async () => { setEditing(null); await refresh(); }} /> : null}
    {assets.isLoading ? <SkeletonRows /> : assets.error ? <StatePanel title="Không thể tải media" tone="danger">{assets.error.message}</StatePanel> : assets.data?.items.length === 0 ? <StatePanel title="Không có asset phù hợp">Thay đổi bộ lọc hoặc thêm hero image, packshot và product footage.</StatePanel> : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{assets.data?.items.map((asset) => <AssetCard key={asset.id} asset={asset} scope={scope} canOperate={canOperate} canReview={canReview} onEdit={() => setEditing(asset)} refresh={refresh} />)}</div>}
  </>;
}

function MediaUploader({ scope, onUploaded }: { scope: Scope; onUploaded: () => Promise<unknown> }) {
  const [uppy] = useState(() => {
    const instance = new Uppy<MediaMeta, MediaBody>({
      id: `studio-media-${scope.workspaceId}`,
      autoProceed: false,
      allowMultipleUploadBatches: true,
      restrictions: { allowedFileTypes: acceptedTypes, maxFileSize: 2 * 1024 * 1024 * 1024, maxNumberOfFiles: 20 },
    });
    if (typeof window !== "undefined") instance.use(GoldenRetriever, { expires: 30 * 60 * 1000 });
    return instance;
  });
  const [batchError, setBatchError] = useState<string | null>(null);

  useEffect(() => {
    const requests = new Map<string, XMLHttpRequest>();
    const fileAdded = (file: UppyFile<MediaMeta, MediaBody>) => {
      uppy.setFileMeta(file.id, { displayName: file.name, folder: "", usageRights: "Owned by client; approved for marketing use", tags: inferCategory(file.type).toLowerCase().replaceAll("_", "-"), uploadKey: newIdempotencyKey() });
    };
    const fileRemoved = (file: UppyFile<MediaMeta, MediaBody>) => requests.get(file.id)?.abort();
    const uploader = async (fileIDs: string[]) => {
      setBatchError(null);
      await Promise.all(fileIDs.map(async (fileID) => {
        const file = uppy.getFile(fileID);
        if (!file.data || file.isRemote || file.size === null) throw new Error("Uppy chỉ nhận local file có dung lượng xác định.");
        uppy.emit("upload-start", [file]);
        try {
          const asset = await uploadOne(
            scope,
            file,
            (request) => requests.set(file.id, request),
            (uploaded) => uppy.emit("upload-progress", uppy.getFile(file.id), { bytesUploaded: uploaded, bytesTotal: file.size, uploadStarted: file.progress.uploadStarted ?? Date.now() }),
            (state) => uppy.setFileMeta(file.id, { resumeState: JSON.stringify(state) }),
          );
          uppy.emit("upload-success", uppy.getFile(file.id), { status: 200, body: { assetId: asset.id }, uploadURL: asset.id });
        } catch (error) {
          const normalized = error instanceof Error ? error : new Error("Upload thất bại.");
          setBatchError(normalized.message);
          uppy.emit("upload-error", uppy.getFile(file.id), normalized);
        } finally {
          requests.delete(file.id);
        }
      }));
      await onUploaded();
    };
    uppy.on("file-added", fileAdded);
    uppy.on("file-removed", fileRemoved);
    uppy.addUploader(uploader);
    return () => {
      uppy.off("file-added", fileAdded);
      uppy.off("file-removed", fileRemoved);
      uppy.removeUploader(uploader);
      for (const request of requests.values()) request.abort();
    };
  }, [onUploaded, scope, uppy]);

  return <Card className="overflow-hidden"><div className="border-b border-[var(--line)] p-5"><div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="font-serif text-xl font-bold">Upload queue</h2><p className="mt-1 text-sm text-[var(--muted)]">Tối đa 20 file/batch · 2 GB/file · tự khôi phục queue; multipart tiếp tục từ phần gần nhất đã xác nhận.</p></div><Badge tone="good">Private bucket</Badge></div>{batchError ? <p role="alert" className="mt-3 text-sm font-semibold text-[var(--coral)]">{batchError}</p> : null}</div><Dashboard uppy={uppy} height={390} proudlyDisplayPoweredByUppy={false} hideProgressDetails={false} showRemoveButtonAfterComplete note="JPEG, PNG, WebP, MP4, MOV, MP3, WAV hoặc PDF" metaFields={[{ id: "displayName", name: "Tên hiển thị", placeholder: "Tên dễ tìm" }, { id: "folder", name: "Folder", placeholder: "campaign/summer" }, { id: "usageRights", name: "Quyền sử dụng", placeholder: "Nguồn và phạm vi được phép" }, { id: "tags", name: "Tags", placeholder: "hero, summer, packshot" }]} /></Card>;
}

async function uploadOne(scope: Scope, file: UppyFile<MediaMeta, MediaBody>, registerRequest: (request: XMLHttpRequest) => void, progress: (bytes: number) => void, persist: (state: ResumeState) => void): Promise<Asset> {
  if (!(file.data instanceof Blob) || file.size === null) throw new Error("File không hợp lệ.");
  const bodyData = file.data;
  const meta = file.meta;
  const assetType = inferAssetType(file.type, file.name);
  const category = inferCategory(file.type, file.name);
  let restored = parseResumeState(meta.resumeState);
  if (restored && new Date(restored.session.expiresAt).getTime() <= Date.now() + 15_000) restored = null;
  let session = restored?.session;
  const parts = restored?.completedParts ?? [];
  if (!session) {
    const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/media-uploads", { params: { path: scope, header: { "Idempotency-Key": String(meta.uploadKey || newIdempotencyKey()) } }, body: { assetType, category, name: String(meta.displayName || file.name), folder: String(meta.folder || ""), filename: file.name, mimeType: file.type, sizeBytes: file.size, usageRights: String(meta.usageRights || ""), tags: String(meta.tags || "").split(",").map((tag) => tag.trim()).filter(Boolean), sourceMetadata: { lastModified: bodyData instanceof File ? bodyData.lastModified : null, uppyFileId: file.id } } });
    if (error || !data) throw apiError(error, "Không thể tạo phiên upload.");
    session = data;
    persist({ session, completedParts: parts });
  }
  if (session.multipart) {
    const count = Math.ceil(file.size / chunkSize);
    const completed = new Map(parts.map((part) => [part.partNumber, part]));
    progress(completedBytes(completed, file.size));
    for (let index = 0; index < count; index++) {
      const partNumber = index + 1;
      if (completed.has(partNumber)) continue;
      const { data: request, error: partError } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/media-uploads/{uploadId}/parts/{partNumber}", { params: { path: { ...scope, uploadId: session.id, partNumber } } });
      if (partError || !request) throw apiError(partError, `Không thể ký phần ${partNumber}.`);
      const body = bodyData.slice(index * chunkSize, Math.min(file.size, (index + 1) * chunkSize));
      const response = await putWithRetry(request, body, registerRequest, (loaded) => progress(Math.min(file.size!, index * chunkSize + loaded)));
      if (!response.etag) throw new Error("Object storage không trả ETag; kiểm tra CORS ExposeHeaders.");
      completed.set(partNumber, { partNumber, etag: response.etag });
      parts.splice(0, parts.length, ...Array.from(completed.values()).sort((a, b) => a.partNumber - b.partNumber));
      persist({ session, completedParts: parts });
    }
  } else {
    if (!session.request) throw new Error("Phiên upload thiếu URL ký.");
    await putWithRetry(session.request, bodyData, registerRequest, progress);
  }
  progress(file.size);
  const { data: asset, error: completeError } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/media-uploads/{uploadId}/complete", { params: { path: { ...scope, uploadId: session.id } }, body: { parts } });
  if (completeError || !asset) throw apiError(completeError, "Server không thể xác minh file đã upload.");
  return asset;
}

function parseResumeState(value: unknown): ResumeState | null {
  if (typeof value !== "string" || !value) return null;
  try {
    const parsed = JSON.parse(value) as ResumeState;
    if (!parsed.session?.id || !parsed.session.expiresAt || !Array.isArray(parsed.completedParts)) return null;
    return parsed;
  } catch {
    return null;
  }
}

function completedBytes(parts: Map<number, UploadedPart>, total: number) {
  let uploaded = 0;
  for (const partNumber of parts.keys()) uploaded += Math.min(chunkSize, Math.max(0, total - (partNumber - 1) * chunkSize));
  return Math.min(total, uploaded);
}

async function putWithRetry(request: PresignedRequest, body: Blob, registerRequest: (request: XMLHttpRequest) => void, progress: (bytes: number) => void) {
  let lastError = new Error("Object storage từ chối upload.");
  for (let attempt = 1; attempt <= 3; attempt++) {
    try {
      return await xhrPut(request, body, registerRequest, progress);
    } catch (error) {
      lastError = error instanceof Error ? error : lastError;
      if (attempt < 3) await new Promise((resolve) => window.setTimeout(resolve, attempt * 350));
    }
  }
  throw lastError;
}

function xhrPut(request: PresignedRequest, body: Blob, registerRequest: (request: XMLHttpRequest) => void, progress: (bytes: number) => void): Promise<{ etag: string | null }> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    registerRequest(xhr);
    xhr.open(request.method, request.url);
    for (const [key, value] of Object.entries(request.headers)) if (!forbiddenHeader(key)) xhr.setRequestHeader(key, value);
    xhr.upload.onprogress = (event) => progress(event.loaded);
    xhr.onerror = () => reject(new Error("Mất kết nối khi upload tới object storage."));
    xhr.onabort = () => reject(new Error("Upload đã bị hủy."));
    xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve({ etag: xhr.getResponseHeader("etag") }) : reject(new Error(`Object storage trả HTTP ${xhr.status}.`));
    xhr.send(body);
  });
}

function forbiddenHeader(key: string) {
  return ["host", "content-length"].includes(key.toLowerCase());
}

function inferAssetType(mime: string, name: string): AssetType {
  if (mime.startsWith("video/")) return name.toLowerCase().includes("screen") ? "SCREEN_RECORDING" : "VIDEO";
  if (mime.startsWith("audio/")) return "AUDIO";
  if (mime === "application/pdf") return "BROCHURE";
  if (name.toLowerCase().includes("logo")) return "LOGO";
  if (name.toLowerCase().includes("screen")) return "SCREENSHOT";
  return "IMAGE";
}

function inferCategory(mime: string, name = "") {
  const normalized = name.toLowerCase();
  if (normalized.includes("packshot")) return "PACKSHOT";
  if (normalized.includes("wheel")) return "WHEEL_DEMO";
  if (normalized.includes("interior")) return "INTERIOR_VIEW";
  if (mime.startsWith("video/")) return "LIFESTYLE";
  if (mime.startsWith("audio/")) return "BACKGROUND_MUSIC";
  return "HERO_IMAGE";
}

function AssetCard({ asset, scope, canOperate, canReview, onEdit, refresh }: { asset: Asset; scope: Scope; canOperate: boolean; canReview: boolean; onEdit: () => void; refresh: () => Promise<unknown> }) {
  const status = useMutation({ mutationFn: async (next: AssetStatus) => { const { data, error } = await api.PATCH("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/status", { params: { path: { ...scope, assetId: asset.id } }, body: { status: next, version: asset.version } }); if (error || !data) throw apiError(error, "Asset đã thay đổi; hãy làm mới trước khi duyệt."); return data; }, onSuccess: refresh });
  const remove = useMutation({ mutationFn: async () => { if (!window.confirm(`Xóa mềm asset “${asset.name}”?`)) return false; const { error } = await api.DELETE("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}", { params: { path: { ...scope, assetId: asset.id } }, body: { version: asset.version } }); if (error) throw apiError(error, "Không thể xóa asset đang được sử dụng hoặc đã thay đổi."); return true; }, onSuccess: (deleted) => deleted ? refresh() : undefined });
  const download = async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { ...scope, assetId: asset.id } } }); if (error || !data) throw apiError(error, "Không thể tạo link tải."); window.open(data.url, "_blank", "noopener,noreferrer"); };
  const tone = asset.status === "APPROVED" ? "good" : asset.status === "REJECTED" ? "danger" : asset.status === "DRAFT" ? "warn" : "neutral";
  return <Card className="overflow-hidden"><AssetPreview asset={asset} scope={scope} /><div className="p-5"><div className="flex items-start gap-3"><span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]">{asset.assetType === "VIDEO" || asset.assetType === "SCREEN_RECORDING" ? <Film className="size-5" /> : asset.assetType === "AUDIO" ? <Music2 className="size-5" /> : <ImageIcon className="size-5" />}</span><div className="min-w-0 flex-1"><h2 className="truncate font-serif text-lg font-bold">{asset.name}</h2><p className="mt-1 text-xs text-[var(--muted)]">{asset.category || "Chưa phân loại"} · {asset.fileSizeBytes ? formatBytes(asset.fileSizeBytes) : "Đang xử lý"}</p></div><Badge tone={tone}>{asset.status}</Badge></div><div className="mt-3 flex flex-wrap gap-1">{asset.folder ? <Badge><FolderOpen className="mr-1 inline size-3" />{asset.folder}</Badge> : null}{asset.tags.map((tag) => <Badge key={tag}>{tag}</Badge>)}</div>{asset.expiresAt ? <p className="mt-3 text-xs text-[var(--muted)]">Hết hạn {new Date(asset.expiresAt).toLocaleDateString("vi-VN")}</p> : null}{(status.error || remove.error) ? <p role="alert" className="mt-3 text-xs text-[var(--coral)]">{status.error?.message ?? remove.error?.message}</p> : null}<div className="mt-4 flex flex-wrap gap-2">{asset.mimeType ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void download()}><Download className="mr-1 size-4" />Mở</Button> : null}{canOperate ? <Button className="bg-white px-3 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={onEdit}><Pencil className="mr-1 size-4" />Sửa</Button> : null}{canReview && asset.status !== "APPROVED" ? <Button className="px-3" disabled={status.isPending || !asset.mimeType} onClick={() => status.mutate("APPROVED")}><Check className="mr-1 size-4" />Duyệt</Button> : null}{canReview && asset.status !== "REJECTED" ? <Button className="bg-[var(--coral)] px-3" disabled={status.isPending} onClick={() => status.mutate("REJECTED")}><X className="mr-1 size-4" />Từ chối</Button> : null}{canOperate ? <Button aria-label={`Xóa ${asset.name}`} className="bg-white px-3 text-[var(--coral)] ring-1 ring-[var(--line)]" disabled={remove.isPending} onClick={() => remove.mutate()}><Trash2 className="size-4" /></Button> : null}</div></div></Card>;
}

function AssetPreview({ asset, scope }: { asset: Asset; scope: Scope }) {
  const preview = useQuery({ queryKey: ["media-preview", asset.id], enabled: Boolean(asset.mimeType && asset.status !== "REJECTED"), staleTime: 4 * 60 * 1000, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { ...scope, assetId: asset.id } } }); if (error || !data) throw apiError(error, "Không thể tải preview."); return data; } });
  const isImage = asset.mimeType?.startsWith("image/");
  const isVideo = asset.mimeType?.startsWith("video/");
  return <div className="relative grid aspect-video place-items-center overflow-hidden bg-[var(--ink)] text-white/60">{preview.data && isImage ? <Image fill unoptimized sizes="(min-width: 1280px) 33vw, (min-width: 768px) 50vw, 100vw" className="object-cover" src={preview.data.url} alt={`Preview ${asset.name}`} /> : preview.data && isVideo ? <video className="h-full w-full object-cover" src={preview.data.url} muted playsInline preload="metadata" /> : asset.assetType === "AUDIO" ? <Music2 className="size-10" /> : <ImageIcon className="size-10" />}</div>;
}

function AssetEditor({ asset, scope, onClose, onSaved }: { asset: Asset; scope: Scope; onClose: () => void; onSaved: () => Promise<unknown> }) {
  const [form, setForm] = useState({ name: asset.name, category: asset.category, folder: asset.folder, usageRights: asset.usageRights, tags: asset.tags.join(", "), expiresAt: asset.expiresAt ? asset.expiresAt.slice(0, 10) : "" });
  const save = useMutation({ mutationFn: async () => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}", { params: { path: { ...scope, assetId: asset.id } }, body: { name: form.name, category: form.category, folder: form.folder, usageRights: form.usageRights, tags: form.tags.split(",").map((tag) => tag.trim()).filter(Boolean), expiresAt: form.expiresAt ? new Date(`${form.expiresAt}T23:59:59Z`).toISOString() : null, version: asset.version } }); if (error || !data) throw apiError(error, "Metadata không hợp lệ hoặc asset đã thay đổi."); return data; }, onSuccess: onSaved });
  return <Card className="mb-6 border-[var(--moss)] p-6"><div className="mb-5 flex items-center justify-between"><div><h2 className="font-serif text-xl font-bold">Sửa metadata</h2><p className="mt-1 text-sm text-[var(--muted)]">{asset.name} · optimistic version {asset.version}</p></div><button aria-label="Đóng trình sửa" className="rounded-full p-2 hover:bg-[#edf0e7]" onClick={onClose}><X className="size-5" /></button></div><div className="grid gap-4 md:grid-cols-2"><Field label="Tên"><input className={inputClass} value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} /></Field><Field label="Danh mục"><input className={inputClass} value={form.category} onChange={(event) => setForm((current) => ({ ...current, category: event.target.value }))} /></Field><Field label="Folder"><input className={inputClass} value={form.folder} onChange={(event) => setForm((current) => ({ ...current, folder: event.target.value }))} /></Field><Field label="Tags (phân cách dấu phẩy)"><input className={inputClass} value={form.tags} onChange={(event) => setForm((current) => ({ ...current, tags: event.target.value }))} /></Field><Field label="Ngày hết hạn"><input className={inputClass} type="date" value={form.expiresAt} onChange={(event) => setForm((current) => ({ ...current, expiresAt: event.target.value }))} /></Field><div className="md:col-span-2"><Field label="Quyền sử dụng"><textarea className={textareaClass} value={form.usageRights} onChange={(event) => setForm((current) => ({ ...current, usageRights: event.target.value }))} /></Field></div></div>{save.error ? <p role="alert" className="mt-3 text-sm text-[var(--coral)]">{save.error.message}</p> : null}<div className="mt-5 flex gap-3"><Button disabled={save.isPending || !form.name.trim() || !form.usageRights.trim()} onClick={() => save.mutate()}><Check className="mr-2 size-4" />Lưu metadata</Button><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={onClose}>Hủy</Button></div></Card>;
}

function formatBytes(value: number) {
  if (value < 1024 * 1024) return `${Math.max(1, Math.round(value / 1024))} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
  return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`;
}
