"use client";

import type { components } from "@studio/api-client";
import { useQuery } from "@tanstack/react-query";
import { ImageIcon, Music2 } from "lucide-react";
import Image from "next/image";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";
import type { MediaScope } from "@/components/media-uploader";

type Asset = components["schemas"]["MediaAsset"];

export function MediaAssetPreview({ asset, scope, className = "aspect-video" }: { asset: Asset; scope: MediaScope; className?: string }) {
  const preview = useQuery({ queryKey: ["media-preview", asset.id, asset.version], enabled: Boolean(asset.mimeType && asset.status !== "REJECTED"), staleTime: 4 * 60 * 1000, queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { ...scope, assetId: asset.id } } }); if (error || !data) throw apiError(error, "Không thể tải preview."); return data; } });
  const isImage = asset.mimeType?.startsWith("image/");
  const isVideo = asset.mimeType?.startsWith("video/");
  return <div className={`relative grid place-items-center overflow-hidden bg-[var(--ink)] text-white/60 ${className}`}>{preview.data && isImage ? <Image fill unoptimized sizes="(min-width: 1280px) 33vw, (min-width: 768px) 50vw, 100vw" className="object-cover" src={preview.data.url} alt={`Preview ${asset.name}`} /> : preview.data && isVideo ? <video aria-label={`Preview ${asset.name}`} className="h-full w-full object-cover" src={preview.data.url} muted playsInline preload="metadata" /> : asset.assetType === "AUDIO" ? <Music2 className="size-10" /> : <ImageIcon className="size-10" />}</div>;
}
