"use client";

import type { Body, Meta, UppyFile } from "@uppy/core";
import Uppy from "@uppy/core";
import GoldenRetriever from "@uppy/golden-retriever";
import { useUppyState } from "@uppy/react";
import type { components } from "@studio/api-client";
import { AlertCircle, CheckCircle2, File as FileIcon, FileText, ImageIcon, Loader2, Music, Plus, Upload, Video, X } from "lucide-react";
import { type DragEvent, type ChangeEvent, useEffect, useId, useRef, useState } from "react";
import { Badge, Button, Card, inputClass } from "@/components/ui";
import { cn } from "@/lib/cn";
import { api } from "@/lib/api";
import { includeCurrentOption, mediaCategoryOptions } from "@/lib/form-options";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type Asset = components["schemas"]["MediaAsset"];
type AssetType = components["schemas"]["MediaAssetType"];
type UploadSession = components["schemas"]["MediaUploadSession"];
type PresignedRequest = components["schemas"]["PresignedRequest"];
export type MediaScope = { clientId: string; workspaceId: string };
type UploadedPart = { partNumber: number; etag: string };
type ResumeState = { session: UploadSession; completedParts: UploadedPart[] };
type MediaMeta = Meta & { displayName?: string; category?: string; folder?: string; usageRights?: string; tags?: string; uploadKey?: string; resumeState?: string };
type MediaBody = Body & { assetId?: string };
type UploaderFile = UppyFile<MediaMeta, MediaBody>;

const chunkSize = 10 * 1024 * 1024;
const allAcceptedTypes = ["image/jpeg", "image/png", "image/webp", "video/mp4", "video/quicktime", "audio/mpeg", "audio/wav", "audio/x-wav", "application/pdf"];
const visualAcceptedTypes = ["image/jpeg", "image/png", "image/webp", "video/mp4", "video/quicktime"];
const logoAcceptedTypes = ["image/jpeg", "image/png", "image/webp"];

export function MediaUploader({ scope, productId, brandId, visualOnly = false, logoOnly = false, onUploaded }: { scope: MediaScope; productId?: string; brandId?: string; visualOnly?: boolean; logoOnly?: boolean; onUploaded: () => Promise<unknown> }) {
  const [uppy] = useState(() => {
    const instance = new Uppy<MediaMeta, MediaBody>({
      id: `studio-media-${scope.workspaceId}-${brandId ? `brand-${brandId}` : productId ? `product-${productId}` : "library"}`,
      autoProceed: false,
      allowMultipleUploadBatches: true,
      restrictions: { allowedFileTypes: logoOnly ? logoAcceptedTypes : visualOnly ? visualAcceptedTypes : allAcceptedTypes, maxFileSize: logoOnly ? 25 * 1024 * 1024 : 2 * 1024 * 1024 * 1024, maxNumberOfFiles: 20 },
    });
    if (typeof window !== "undefined" && typeof window.indexedDB !== "undefined") instance.use(GoldenRetriever, { expires: 30 * 60 * 1000 });
    return instance;
  });
  const [batchError, setBatchError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const dragDepthRef = useRef(0);
  const formatDescriptionId = useId();
  const queueTitleId = useId();
  const fileMap = useUppyState(uppy, (state) => state.files);
  const files = Object.values(fileMap);
  const uploading = files.some((file) => Boolean(file.progress.uploadStarted) && !file.progress.uploadComplete && !file.error);
  const pendingCount = files.filter((file) => !file.progress.uploadStarted && !file.progress.uploadComplete).length;

  useEffect(() => {
    const requests = new Map<string, XMLHttpRequest>();
    const fileAdded = (file: UppyFile<MediaMeta, MediaBody>) => {
      const category = logoOnly ? "BRAND_LOGO" : inferCategory(file.type, file.name);
      uppy.setFileMeta(file.id, { displayName: file.name, category, folder: logoOnly ? "brands/logos" : productId ? "products" : "", usageRights: "Owned by client; approved for marketing use", tags: category.toLowerCase().replaceAll("_", "-"), uploadKey: newIdempotencyKey() });
    };
    const fileRemoved = (file: UppyFile<MediaMeta, MediaBody>) => requests.get(file.id)?.abort();
    const uploader = async (fileIDs: string[]) => {
      setBatchError(null);
      await Promise.all(fileIDs.map(async (fileID) => {
        const file = uppy.getFile(fileID);
        if (!file.data || file.isRemote || file.size === null) throw new Error("Uppy chỉ nhận local file có dung lượng xác định.");
        uppy.emit("upload-start", [file]);
        try {
          const asset = await uploadOne(scope, productId, brandId, logoOnly, file, (request) => requests.set(file.id, request), (uploaded) => uppy.emit("upload-progress", uppy.getFile(file.id), { bytesUploaded: uploaded, bytesTotal: file.size, uploadStarted: file.progress.uploadStarted ?? Date.now() }), (state) => uppy.setFileMeta(file.id, { resumeState: JSON.stringify(state) }));
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
  }, [brandId, logoOnly, onUploaded, productId, scope, uppy]);

  useEffect(() => () => uppy.destroy(), [uppy]);

  const acceptedNote = logoOnly ? "PNG, WebP hoặc JPEG · tối đa 25 MB/file" : visualOnly ? "JPEG, PNG, WebP, MP4 hoặc MOV" : "JPEG, PNG, WebP, MP4, MOV, MP3, WAV hoặc PDF";
  const acceptedTypes = logoOnly ? logoAcceptedTypes : visualOnly ? visualAcceptedTypes : allAcceptedTypes;

  const addFiles = (selected: File[]) => {
    setBatchError(null);
    try {
      uppy.addFiles(selected.map((file) => ({ name: file.name, type: file.type, data: file, source: "Local" })));
    } catch (error) {
      setBatchError(error instanceof Error ? error.message : "Không thể thêm file vào hàng đợi.");
    }
  };

  const handleInput = (event: ChangeEvent<HTMLInputElement>) => {
    addFiles(Array.from(event.target.files ?? []));
    event.target.value = "";
  };

  const handleDrop = (event: DragEvent<HTMLButtonElement>) => {
    event.preventDefault();
    dragDepthRef.current = 0;
    setDragActive(false);
    addFiles(Array.from(event.dataTransfer.files));
  };

  const clearQueue = () => {
    for (const file of files) uppy.removeFile(file.id);
    setBatchError(null);
  };

  return (
    <Card className="overflow-hidden">
      <div className="border-b border-[var(--line)] p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="font-serif text-xl font-bold">{logoOnly ? "Upload logo thương hiệu" : productId ? "Upload cho sản phẩm" : "Upload media"}</h2>
            <p className="mt-1 text-sm text-[var(--muted)]">{logoOnly ? "Logo được lưu vào Media Library và cần duyệt trước khi chọn." : productId ? "Asset được lưu trong Media Library và tự động gắn với sản phẩm này." : "Tối đa 20 file/batch · 2 GB/file · multipart có thể tiếp tục."}</p>
          </div>
          <Badge tone="good">Một nguồn dữ liệu</Badge>
        </div>
        {batchError ? <p role="alert" className="mt-3 flex items-start gap-2 text-sm font-semibold text-[var(--coral)]"><AlertCircle className="mt-0.5 size-4 shrink-0" />{batchError}</p> : null}
      </div>

      <div className="p-4 sm:p-5">
        <input ref={inputRef} className="sr-only" type="file" multiple accept={acceptedTypes.join(",")} onChange={handleInput} tabIndex={-1} aria-hidden="true" />
        <button
          type="button"
          className={cn(
            "group flex min-h-52 w-full flex-col items-center justify-center rounded-2xl border-2 border-dashed px-5 py-8 text-center transition focus-visible:outline-offset-4",
            dragActive ? "border-[var(--moss)] bg-[#eef5e6] shadow-[inset_0_0_0_1px_var(--moss)]" : "border-[#cfd5c9] bg-[#fbfaf4] hover:border-[var(--moss)] hover:bg-[#f5f7ee]",
            files.length > 0 && "min-h-36",
          )}
          onClick={() => inputRef.current?.click()}
          onDragEnter={(event) => { event.preventDefault(); dragDepthRef.current += 1; setDragActive(true); }}
          onDragOver={(event) => { event.preventDefault(); event.dataTransfer.dropEffect = "copy"; setDragActive(true); }}
          onDragLeave={(event) => { event.preventDefault(); dragDepthRef.current = Math.max(0, dragDepthRef.current - 1); if (dragDepthRef.current === 0) setDragActive(false); }}
          onDrop={handleDrop}
          aria-describedby={formatDescriptionId}
        >
          <span className={cn("mb-4 grid size-14 place-items-center rounded-2xl transition", dragActive ? "bg-[var(--moss)] text-white" : "bg-[#e7efdc] text-[var(--moss)] group-hover:bg-[var(--lime)]")}>
            <Upload className="size-6" aria-hidden="true" />
          </span>
          <span className="text-base font-bold sm:text-lg">{dragActive ? "Thả file để thêm vào hàng đợi" : "Kéo thả file vào đây"}</span>
          <span className="mt-1 text-sm text-[var(--muted)]">hoặc <span className="font-bold text-[var(--moss)] underline decoration-2 underline-offset-4">nhấn để chọn từ máy</span></span>
          <span id={formatDescriptionId} className="mt-4 text-xs leading-5 text-[var(--muted)]">{acceptedNote}</span>
        </button>

        {files.length > 0 ? (
          <section className="mt-5" aria-labelledby={queueTitleId}>
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 id={queueTitleId} className="font-bold">Xem trước &amp; hàng đợi</h3>
                <p className="mt-0.5 text-xs text-[var(--muted)]">{files.length}/20 file · kiểm tra nội dung trước khi tải lên</p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button className="min-h-11 bg-white px-4 text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => inputRef.current?.click()} disabled={files.length >= 20}>
                  <Plus className="mr-2 size-4" />Thêm file
                </Button>
                <Button className="min-h-11 bg-white px-4 text-[var(--coral)] ring-1 ring-[var(--line)] hover:bg-[#fff1ed]" onClick={clearQueue} disabled={uploading}>
                  Xóa tất cả
                </Button>
                <Button className="min-h-11 px-5" onClick={() => { setBatchError(null); void uppy.upload().catch((error: unknown) => setBatchError(error instanceof Error ? error.message : "Upload thất bại.")); }} disabled={pendingCount === 0 || uploading}>
                  {uploading ? <Loader2 className="mr-2 size-4 animate-spin" /> : <Upload className="mr-2 size-4" />}{uploading ? "Đang tải lên…" : `Tải lên ${pendingCount} file`}
                </Button>
              </div>
            </div>
            <div className="grid gap-4 xl:grid-cols-2">
              {files.map((file) => <FilePreviewCard key={file.id} file={file} uppy={uppy} disabled={uploading} logoOnly={logoOnly} />)}
            </div>
          </section>
        ) : null}
      </div>
    </Card>
  );
}

function FilePreviewCard({ file, uppy, disabled, logoOnly }: { file: UploaderFile; uppy: Uppy<MediaMeta, MediaBody>; disabled: boolean; logoOnly: boolean }) {
  const objectUrl = useObjectUrl(file);
  const percentage = file.progress.percentage ?? 0;
  const status = file.error ? "error" : file.progress.uploadComplete ? "complete" : file.progress.uploadStarted ? "uploading" : "ready";
  const statusLabel = status === "error" ? "Lỗi" : status === "complete" ? "Đã tải lên" : status === "uploading" ? `${Math.round(percentage)}%` : "Sẵn sàng";
  const statusTone = status === "error" ? "danger" : status === "complete" ? "good" : status === "uploading" ? "warn" : "neutral";
  const updateMeta = (key: keyof Pick<MediaMeta, "displayName" | "category" | "folder" | "usageRights" | "tags">, value: string) => uppy.setFileMeta(file.id, { [key]: value });

  return (
    <article className="overflow-hidden rounded-2xl border border-[var(--line)] bg-white">
      <div className="grid min-h-36 grid-cols-[7.5rem_1fr] sm:grid-cols-[9rem_1fr]">
        <FileVisual file={file} objectUrl={objectUrl} />
        <div className="min-w-0 p-4">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="truncate text-sm font-bold" title={file.name}>{file.meta.displayName || file.name}</p>
              <p className="mt-1 text-xs text-[var(--muted)]">{formatBytes(file.size)} · {file.type || "Không rõ định dạng"}</p>
            </div>
            <button type="button" className="grid size-10 shrink-0 place-items-center rounded-full text-[var(--muted)] transition hover:bg-[#f1f2ec] hover:text-[var(--coral)] disabled:opacity-40" onClick={() => uppy.removeFile(file.id)} disabled={disabled} aria-label={`Xóa ${file.name} khỏi hàng đợi`}>
              <X className="size-4" />
            </button>
          </div>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <Badge tone={statusTone}>{status === "complete" ? <CheckCircle2 className="mr-1 size-3.5" /> : status === "error" ? <AlertCircle className="mr-1 size-3.5" /> : null}{statusLabel}</Badge>
            <Badge>{String(file.meta.category || "CHƯA PHÂN LOẠI")}</Badge>
            {status === "error" ? <button type="button" className="min-h-8 rounded-full px-3 text-xs font-bold text-[var(--moss)] ring-1 ring-[var(--line)] hover:bg-[#f1f5ea]" onClick={() => void uppy.retryUpload(file.id)} disabled={disabled}>Thử lại</button> : null}
          </div>
          {status === "uploading" ? <div className="mt-4 h-2 overflow-hidden rounded-full bg-[#e8ebe4]" role="progressbar" aria-label={`Tiến độ tải ${file.name}`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(percentage)}><div className="h-full rounded-full bg-[var(--moss)] transition-[width]" style={{ width: `${percentage}%` }} /></div> : null}
          {file.error ? <p role="alert" className="mt-3 text-xs font-semibold text-[var(--coral)]">{String(file.error)}</p> : null}
        </div>
      </div>
      <details className="border-t border-[var(--line)] px-4 py-3">
        <summary className="min-h-8 cursor-pointer text-sm font-bold text-[var(--moss)]">Chỉnh thông tin file</summary>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <MetadataField label="Tên hiển thị" fileName={file.name}><input className={inputClass} value={String(file.meta.displayName || "")} onChange={(event) => updateMeta("displayName", event.target.value)} disabled={disabled} /></MetadataField>
          <MetadataField label="Danh mục" fileName={file.name}><select className={inputClass} value={String(file.meta.category || "")} onChange={(event) => updateMeta("category", event.target.value)} disabled={disabled || logoOnly}>{includeCurrentOption(mediaCategoryOptions,String(file.meta.category||"")).map((option)=><option key={option.value} value={option.value}>{option.label}</option>)}</select></MetadataField>
          <MetadataField label="Folder" fileName={file.name}><input className={inputClass} value={String(file.meta.folder || "")} onChange={(event) => updateMeta("folder", event.target.value)} disabled={disabled} /></MetadataField>
          <MetadataField label="Tags" fileName={file.name}><input className={inputClass} value={String(file.meta.tags || "")} onChange={(event) => updateMeta("tags", event.target.value)} disabled={disabled} placeholder="hero, summer, packshot" /></MetadataField>
          <div className="sm:col-span-2"><MetadataField label="Quyền sử dụng" fileName={file.name}><input className={inputClass} value={String(file.meta.usageRights || "")} onChange={(event) => updateMeta("usageRights", event.target.value)} disabled={disabled} /></MetadataField></div>
        </div>
      </details>
    </article>
  );
}

function MetadataField({ label, fileName, children }: { label: string; fileName: string; children: React.ReactElement<{ "aria-label"?: string }> }) {
  return <label className="grid gap-1.5 text-xs font-bold text-[var(--muted)]">{label}<span className="sr-only"> cho {fileName}</span>{children}</label>;
}

function FileVisual({ file, objectUrl }: { file: UploaderFile; objectUrl: string | null }) {
  const className = "h-full min-h-36 w-full bg-[#eef0e8] object-cover";
  // Blob URLs are ephemeral local previews and should not pass through Next's image optimizer.
  // eslint-disable-next-line @next/next/no-img-element
  if (objectUrl && file.type.startsWith("image/")) return <img className={className} src={objectUrl} alt={`Xem trước ${file.name}`} />;
  if (objectUrl && file.type.startsWith("video/")) return <video className={className} src={objectUrl} controls muted playsInline preload="metadata" aria-label={`Xem trước ${file.name}`} />;
  if (objectUrl && file.type.startsWith("audio/")) return <div className="flex min-h-36 flex-col items-center justify-center gap-3 bg-[#f1f4eb] p-3 text-[var(--moss)]"><Music className="size-9" /><audio className="h-9 w-full" src={objectUrl} controls aria-label={`Nghe thử ${file.name}`} /></div>;
  const Icon = file.type === "application/pdf" ? FileText : file.type.startsWith("image/") ? ImageIcon : file.type.startsWith("video/") ? Video : file.type.startsWith("audio/") ? Music : FileIcon;
  return <div className="grid min-h-36 place-items-center bg-[#f1f4eb] text-[var(--moss)]"><Icon className="size-10" aria-hidden="true" /></div>;
}

function useObjectUrl(file: UploaderFile) {
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  useEffect(() => {
    if (!(file.data instanceof Blob) || typeof URL.createObjectURL !== "function") return;
    const next = URL.createObjectURL(file.data);
    let disposed = false;
    let revoked = false;
    const revoke = () => {
      if (revoked) return;
      revoked = true;
      URL.revokeObjectURL(next);
    };
    queueMicrotask(() => disposed ? revoke() : setObjectUrl(next));
    return () => { disposed = true; revoke(); };
  }, [file.data]);
  return objectUrl;
}

function formatBytes(value: number | null) {
  if (value === null) return "Không rõ dung lượng";
  const units = ["B", "KB", "MB", "GB"];
  let amount = value;
  let index = 0;
  while (amount >= 1024 && index < units.length - 1) { amount /= 1024; index += 1; }
  return `${new Intl.NumberFormat("vi-VN", { maximumFractionDigits: index === 0 ? 0 : 1 }).format(amount)} ${units[index]}`;
}

async function uploadOne(scope: MediaScope, productId: string | undefined, brandId: string | undefined, logoOnly: boolean, file: UppyFile<MediaMeta, MediaBody>, registerRequest: (request: XMLHttpRequest) => void, progress: (bytes: number) => void, persist: (state: ResumeState) => void): Promise<Asset> {
  if (!(file.data instanceof Blob) || file.size === null) throw new Error("File không hợp lệ.");
  const bodyData = file.data;
  const meta = file.meta;
  const assetType = logoOnly ? "LOGO" : inferAssetType(file.type, file.name);
  const category = String(meta.category || (logoOnly ? "BRAND_LOGO" : inferCategory(file.type, file.name))).trim().toUpperCase();
  let restored = parseResumeState(meta.resumeState);
  if (restored && new Date(restored.session.expiresAt).getTime() <= Date.now() + 15_000) restored = null;
  let session = restored?.session;
  const parts = restored?.completedParts ?? [];
  if (!session) {
    const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/media-uploads", { params: { path: scope, header: { "Idempotency-Key": String(meta.uploadKey || newIdempotencyKey()) } }, body: { brandId: brandId ?? null, productId: productId ?? null, assetType, category, name: String(meta.displayName || file.name), folder: String(meta.folder || ""), filename: file.name, mimeType: file.type, sizeBytes: file.size, usageRights: String(meta.usageRights || ""), tags: String(meta.tags || "").split(",").map((tag) => tag.trim()).filter(Boolean), sourceMetadata: { lastModified: bodyData instanceof File ? bodyData.lastModified : null, uppyFileId: file.id } } });
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
    return parsed.session?.id && parsed.session.expiresAt && Array.isArray(parsed.completedParts) ? parsed : null;
  } catch { return null; }
}

function completedBytes(parts: Map<number, UploadedPart>, total: number) {
  let uploaded = 0;
  for (const partNumber of parts.keys()) uploaded += Math.min(chunkSize, Math.max(0, total - (partNumber - 1) * chunkSize));
  return Math.min(total, uploaded);
}

async function putWithRetry(request: PresignedRequest, body: Blob, registerRequest: (request: XMLHttpRequest) => void, progress: (bytes: number) => void) {
  let lastError = new Error("Object storage từ chối upload.");
  for (let attempt = 1; attempt <= 3; attempt++) {
    try { return await xhrPut(request, body, registerRequest, progress); }
    catch (error) {
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
    for (const [key, value] of Object.entries(request.headers ?? {})) if (!["host", "content-length"].includes(key.toLowerCase())) xhr.setRequestHeader(key, value);
    xhr.upload.onprogress = (event) => progress(event.loaded);
    xhr.onerror = () => reject(new Error("Mất kết nối khi upload tới object storage."));
    xhr.onabort = () => reject(new Error("Upload đã bị hủy."));
    xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve({ etag: xhr.getResponseHeader("etag") }) : reject(new Error(`Object storage trả HTTP ${xhr.status}.`));
    xhr.send(body);
  });
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
  if (normalized.includes("front")) return "FRONT_VIEW";
  if (normalized.includes("side")) return "SIDE_VIEW";
  if (normalized.includes("wheel")) return mime.startsWith("video/") ? "WHEEL_DEMO" : "WHEEL_CLOSE_UP";
  if (normalized.includes("interior")) return "INTERIOR_VIEW";
  if (mime.startsWith("video/")) return "LIFESTYLE";
  if (mime.startsWith("audio/")) return "BACKGROUND_MUSIC";
  return "HERO_IMAGE";
}
