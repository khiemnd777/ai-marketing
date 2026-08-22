"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Ban, Check, Eye, ShieldCheck, Star } from "lucide-react";
import { Suspense, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";
import { qualityNeedsAction } from "@/lib/quality";

type Scene = components["schemas"]["CampaignScene"];
type Generation = components["schemas"]["SceneGeneration"];
type Review = components["schemas"]["SceneGenerationReview"];
type Scope = ReturnType<typeof useCampaignRoute>;
type QueueItem = { scene: Scene; generation: Generation };
type Filter = "ACTION" | "ALL" | "PASSED";

const checklist = [
  ["duplicateCharacter", "Nhân vật bị nhân đôi"],
  ["duplicateProduct", "Sản phẩm bị nhân đôi"],
  ["productColorMismatch", "Sai màu sản phẩm"],
  ["blurOrLowQualityWarning", "Mờ hoặc chất lượng thấp"],
  ["cropWarning", "Crop không an toàn"],
  ["subtitleOverflow", "Subtitle tràn khung"],
  ["logoOverlap", "Logo bị chồng lấn"],
  ["ctaSafeZoneViolation", "CTA vi phạm safe zone"],
] as const;

export default function QualityPage() {
  return <Suspense fallback={<SkeletonRows />}><QualityWorkspace /></Suspense>;
}

function QualityWorkspace() {
  const scope = useCampaignRoute();
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("ACTION");
  const scenes = useQuery({
    queryKey: ["campaign-scenes", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể tải scenes để duyệt take.");
      return data;
    },
  });
  const generationQueries = useQueries({
    queries: (scenes.data?.items ?? []).map((scene) => ({
      queryKey: ["scene-generations", scope.campaignId, scene.id],
      refetchInterval: (query: { state: { data?: { items: Generation[] } } }) => query.state.data?.items.some((item) => ["QUEUED", "SUBMITTING", "PROVIDER_QUEUED", "PROVIDER_PROCESSING", "SUCCEEDED", "DOWNLOADING", "VALIDATING"].includes(item.status)) ? 2500 : false,
      queryFn: async () => {
        const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/{sceneId}/generations", { params: { path: { ...scope, sceneId: scene.id } } });
        if (error || !data) throw apiError(error, `Không thể tải take của ${scene.sceneKey}.`);
        return data;
      },
    })),
  });
  const queue = useMemo(() => (scenes.data?.items ?? []).flatMap((scene, index) => (generationQueries[index]?.data?.items ?? []).map((generation) => ({ scene, generation }))), [generationQueries, scenes.data?.items]);
  const visible = queue.filter((item) => filter === "ALL" || (filter === "ACTION" ? qualityNeedsAction(item.generation) : item.generation.status === "APPROVED"));
  const actionCount = queue.filter((item) => qualityNeedsAction(item.generation)).length;
  const approvedCount = queue.filter((item) => item.generation.status === "APPROVED").length;
  const selectedCount = queue.filter((item) => item.generation.selected).length;
  const refresh = async (sceneId: string) => qc.invalidateQueries({ queryKey: ["scene-generations", scope.campaignId, sceneId] });

  return <CampaignHeader active="/quality" title="Duyệt take" description="Kiểm tra transcript, deterministic QC, findings thị giác và human checklist, sau đó chọn đúng một take đã duyệt cho mỗi cảnh trước khi dựng final.">
    {scenes.isLoading ? <SkeletonRows /> : scenes.error ? <StatePanel title="Không thể tải trang duyệt take" tone="danger">{scenes.error.message}</StatePanel> : <>
      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Metric label="Tổng take" value={queue.length} icon={Eye} />
        <Metric label="Cần xử lý" value={actionCount} icon={AlertTriangle} tone="warn" />
        <Metric label="Đã duyệt" value={approvedCount} icon={ShieldCheck} tone="good" />
        <Metric label="Đã chọn" value={selectedCount} icon={Star} tone={selectedCount === (scenes.data?.items.length ?? 0) && selectedCount > 0 ? "good" : "warn"} />
      </div>
      <div className="mb-5 flex flex-wrap gap-2" role="group" aria-label="Lọc hàng đợi duyệt take">{([['ACTION', `Cần xử lý (${actionCount})`], ['ALL', `Tất cả (${queue.length})`], ['PASSED', `Đã duyệt (${approvedCount})`]] as Array<[Filter, string]>).map(([value, label]) => <button key={value} type="button" aria-pressed={filter === value} className={filter === value ? "rounded-full bg-[var(--ink)] px-4 py-2 text-sm font-bold text-white" : "rounded-full bg-white px-4 py-2 text-sm font-bold text-[var(--muted)] ring-1 ring-[var(--line)]"} onClick={() => setFilter(value)}>{label}</button>)}</div>
      {generationQueries.some((query) => query.isLoading) && queue.length === 0 ? <SkeletonRows /> : generationQueries.find((query) => query.error)?.error ? <StatePanel title="Không thể tải một số takes" tone="danger">{generationQueries.find((query) => query.error)?.error?.message}</StatePanel> : visible.length ? <div className="grid gap-5">{visible.map((item) => <QualityCard key={item.generation.id} item={item} scope={scope} refresh={() => refresh(item.scene.id)} />)}</div> : <StatePanel title={filter === "ACTION" ? "Hàng đợi đã sạch" : "Không có take phù hợp"}>{filter === "ACTION" ? "Không còn finding hoặc take chờ review trong campaign này." : "Hãy tạo take trong Scene Director."}</StatePanel>}
    </>}
  </CampaignHeader>;
}

function Metric({ label, value, icon: Icon, tone = "neutral" }: { label: string; value: number; icon: typeof Eye; tone?: "neutral" | "good" | "warn" }) {
  return <Card className="flex items-center gap-4 p-5"><span className="grid size-11 place-items-center rounded-2xl bg-[#edf0e7]"><Icon className="size-5 text-[var(--moss)]" /></span><div><span className="block text-2xl font-bold">{value}</span><span className="text-sm text-[var(--muted)]">{label}</span></div><span className="ml-auto"><Badge tone={tone}>{tone === "good" ? "PASS" : tone === "warn" ? "CHECK" : "INFO"}</Badge></span></Card>;
}

function QualityCard({ item, scope, refresh }: { item: QueueItem; scope: Scope; refresh: () => Promise<unknown> }) {
  const { canReview } = usePermissions();
  const generation = item.generation;
  const quality = generation.qualityCheck;
  const [notes, setNotes] = useState(generation.reviewNotes || "Đã kiểm tra dialogue, hai nhân vật, sản phẩm, crop, logo và CTA safe zone.");
  const [review, setReview] = useState<Omit<Review, "action" | "version" | "notes">>({
    characterCount: quality?.characterCountReview ?? 2,
    duplicateCharacter: quality?.duplicateCharacterReview ?? false,
    duplicateProduct: quality?.duplicateProductReview ?? false,
    productColorMismatch: quality?.productColorMismatch ?? false,
    blurOrLowQualityWarning: quality?.blurOrLowQualityWarning ?? false,
    cropWarning: quality?.cropWarning ?? false,
    subtitleOverflow: quality?.subtitleOverflow ?? false,
    logoOverlap: quality?.logoOverlap ?? false,
    ctaSafeZoneViolation: quality?.ctaSafeZoneViolation ?? false,
  });
  const decide = useMutation({
    mutationFn: async (action: "APPROVE" | "REJECT") => {
      const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/{sceneId}/generations/{generationId}/review", { params: { path: { ...scope, sceneId: item.scene.id, generationId: generation.id } }, body: { ...review, action, version: generation.version, notes } });
      if (error || !data) throw apiError(error, "Checklist chưa đầy đủ hoặc QC chưa sẵn sàng.");
      return data;
    },
    onSuccess: refresh,
  });
  const select = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/{sceneId}/generations/{generationId}/select", { params: { path: { ...scope, sceneId: item.scene.id, generationId: generation.id } } });
      if (error || !data) throw apiError(error, "Chỉ take đã duyệt mới được chọn.");
      return data;
    },
    onSuccess: refresh,
  });
  const blocking = quality?.deterministicPass === false || quality?.transcriptPass === false || quality?.videoDecodes === false || quality?.durationPass === false || quality?.resolutionPass === false;
  return <Card className="overflow-hidden"><div className="grid lg:grid-cols-[340px_minmax(0,1fr)]"><div className="min-h-52 bg-[var(--ink)]"><AssetVideo assetId={generation.outputAssetId} scope={scope} /></div><div className="p-5"><div className="flex flex-wrap items-center gap-2"><span className="grid size-8 place-items-center rounded-full bg-[var(--ink)] text-sm font-bold text-white">{item.scene.order}</span><h2 className="font-serif text-xl font-bold">{item.scene.sceneKey} · take #{generation.attemptNumber}</h2><Badge tone={generation.status === "APPROVED" ? "good" : generation.status === "FAILED" || generation.status === "REJECTED" ? "danger" : "warn"}>{generation.status}</Badge>{generation.selected ? <Badge tone="good">Đã chọn</Badge> : null}<span className="ml-auto text-xs text-[var(--muted)]">v{generation.version}</span></div>
        <p className="mt-3 rounded-xl bg-[#f5f6f0] p-3 text-sm"><strong>Approved dialogue:</strong> {item.scene.direction.dialogue}</p>
        {generation.transcription ? <p className="mt-3 text-sm"><strong>Transcript:</strong> {generation.transcription.transcript || generation.transcription.status}</p> : null}
        <div className="mt-4 flex flex-wrap gap-2"><CheckBadge label="Decode" value={quality?.videoDecodes} /><CheckBadge label="Duration" value={quality?.durationPass} /><CheckBadge label="Resolution" value={quality?.resolutionPass} /><CheckBadge label="Audio" value={quality?.audioStreamPresent} /><CheckBadge label="Transcript" value={quality?.transcriptPass} /><CheckBadge label="Deterministic" value={quality?.deterministicPass} /></div>
        {quality?.findings.length ? <div className="mt-4 rounded-2xl border border-[#f3c7bc] bg-[#fff4f0] p-4"><p className="text-sm font-bold text-[var(--coral)]">Findings cần xác minh</p><ul className="mt-2 list-disc space-y-1 pl-5 text-sm">{quality.findings.map((finding) => <li key={finding}>{finding}</li>)}</ul></div> : null}
        {canReview && generation.status === "REVIEW_REQUIRED" ? <div className="mt-5 grid gap-4"><Field label="Số nhân vật nhìn thấy"><input className={inputClass} type="number" min={0} value={review.characterCount} onChange={(event) => setReview((current) => ({ ...current, characterCount: Number(event.target.value) }))} /></Field><div className="grid gap-2 sm:grid-cols-2">{checklist.map(([key, label]) => <label key={key} className="flex min-h-11 items-center gap-2 rounded-xl bg-[#f5f6f0] px-3 text-xs font-semibold"><input type="checkbox" checked={review[key]} onChange={(event) => setReview((current) => ({ ...current, [key]: event.target.checked }))} />{label}</label>)}</div><Field label="Ghi chú review"><textarea className={textareaClass} value={notes} onChange={(event) => setNotes(event.target.value)} /></Field></div> : null}
        {(decide.error || select.error) ? <p role="alert" className="mt-3 text-sm text-[var(--coral)]">{decide.error?.message ?? select.error?.message}</p> : null}
        <div className="mt-4 flex flex-wrap gap-2">{canReview && generation.status === "REVIEW_REQUIRED" ? <><Button disabled={decide.isPending || blocking} title={blocking ? "Deterministic QC phải pass trước khi duyệt" : undefined} onClick={() => decide.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt take</Button><Button className="bg-[var(--coral)]" disabled={decide.isPending || !notes.trim()} onClick={() => decide.mutate("REJECT")}><Ban className="mr-2 size-4" />Từ chối</Button></> : null}{canReview && generation.status === "APPROVED" && !generation.selected ? <Button disabled={select.isPending} onClick={() => select.mutate()}><Star className="mr-2 size-4" />Chọn cho Composer</Button> : null}</div>
      </div></div></Card>;
}

function CheckBadge({ label, value }: { label: string; value: boolean | null | undefined }) {
  return <Badge tone={value === true ? "good" : value === false ? "danger" : "neutral"}>{label}: {value === true ? "pass" : value === false ? "fail" : "n/a"}</Badge>;
}

function AssetVideo({ assetId, scope }: { assetId: string | null | undefined; scope: Scope }) {
  const download = useQuery({
    queryKey: ["signed-download", scope.clientId, scope.workspaceId, assetId],
    enabled: !!assetId,
    staleTime: 4 * 60 * 1000,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { clientId: scope.clientId, workspaceId: scope.workspaceId, assetId: assetId! } } });
      if (error || !data) throw apiError(error, "Không thể tải preview.");
      return data;
    },
  });
  if (!assetId) return <div className="grid h-full place-items-center p-5 text-center text-sm text-white/65">Take chưa có output</div>;
  if (download.isLoading) return <div className="grid h-full place-items-center text-sm text-white/65">Đang tải preview…</div>;
  if (download.error) return <div className="grid h-full place-items-center p-4 text-center text-xs text-white/70">{download.error.message}</div>;
  return <video className="h-full w-full object-contain" controls playsInline preload="metadata" src={download.data?.url} />;
}
