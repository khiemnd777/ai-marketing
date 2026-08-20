"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, Check, Download, Film, Save, Sparkles, Star } from "lucide-react";
import { Suspense, useState } from "react";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type ProjectInput = components["schemas"]["VideoProjectInput"];
type FinalRender = components["schemas"]["FinalRender"];
type Scope = ReturnType<typeof useCampaignRoute>;

const defaultProject: ProjectInput = {
  headline: "Video sản phẩm",
  lowerThird: "",
  showPrice: true,
  showDiscountCode: true,
  showCta: true,
  showWebsite: true,
  showPhone: true,
  showQrCode: true,
  showDisclaimer: true,
  burnCaptions: true,
  musicAssetId: null,
  musicGainDb: -18,
  dialogueDuckingDb: -9,
  changeSummary: "Thiết lập final composer",
  version: 1,
};

const activeStatuses = new Set(["QUEUED", "BUILDING_MANIFEST", "RENDERING", "VALIDATING", "UPLOADING"]);

export default function FinalPage() {
  return <Suspense fallback={<SkeletonRows />}><FinalComposer /></Suspense>;
}

function FinalComposer() {
  const scope = useCampaignRoute();
  const qc = useQueryClient();
  const project = useQuery({
    queryKey: ["video-project", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    retry: false,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/video-project", { params: { path: scope } });
      if (response.status === 404) return null;
      if (error || !data) throw apiError(error, "Không thể tải final project.");
      return data;
    },
  });
  const renders = useQuery({
    queryKey: ["final-renders", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    refetchInterval: (query) => query.state.data?.items.some((item) => activeStatuses.has(item.status)) ? 2000 : false,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể tải final renders.");
      return data;
    },
  });
  const [draftOverride, setDraft] = useState<ProjectInput | null>(null);
  const draft: ProjectInput = draftOverride ?? (project.data ? {
    headline: project.data.headline,
    lowerThird: project.data.lowerThird,
    showPrice: project.data.showPrice,
    showDiscountCode: project.data.showDiscountCode,
    showCta: project.data.showCta,
    showWebsite: project.data.showWebsite,
    showPhone: project.data.showPhone,
    showQrCode: project.data.showQrCode,
    showDisclaimer: project.data.showDisclaimer,
    burnCaptions: project.data.burnCaptions,
    musicAssetId: project.data.musicAssetId,
    musicGainDb: project.data.musicGainDb,
    dialogueDuckingDb: project.data.dialogueDuckingDb,
    changeSummary: project.data.changeSummary,
    version: project.data.version,
  } : defaultProject);
  const refresh = async () => {
    setDraft(null);
    await Promise.all([
      qc.invalidateQueries({ queryKey: ["video-project", scope.campaignId] }),
      qc.invalidateQueries({ queryKey: ["final-renders", scope.campaignId] }),
      qc.invalidateQueries({ queryKey: ["campaign", scope.clientId, scope.workspaceId, scope.campaignId] }),
    ]);
  };
  const save = useMutation({
    mutationFn: async () => {
      const body = { ...draft, version: project.data?.version ?? 1 };
      const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/video-project", { params: { path: scope }, body });
      if (error || !data) throw apiError(error, "Không thể lưu final composer.");
      return data;
    },
    onSuccess: refresh,
  });
  const start = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } } });
      if (error || !data) throw apiError(error, "Mỗi scene phải có một take đã duyệt và được chọn trước khi render.");
      return data;
    },
    onSuccess: refresh,
  });
  const change = <K extends keyof ProjectInput>(key: K, value: ProjectInput[K]) => setDraft((current) => ({ ...(current ?? draft), [key]: value }));
  const toggles: Array<[keyof ProjectInput, string]> = [["showPrice", "Giá"], ["showDiscountCode", "Mã giảm giá"], ["showCta", "CTA"], ["showWebsite", "Website"], ["showPhone", "Điện thoại"], ["showQrCode", "QR"], ["showDisclaimer", "Disclaimer"], ["burnCaptions", "Burned captions"]];
  const busy = renders.data?.items.some((item) => activeStatuses.has(item.status));

  return <CampaignHeader active="/final" title="Final Composer" description="Ghép các take đã duyệt với product media, logo, offer, CTA, QR, subtitle, nhạc và ducking thành MP4 dọc 1080×1920. Mỗi lần lưu tạo một version mới; render lại không gọi Seedance.">
    {project.isLoading ? <SkeletonRows /> : project.error ? <StatePanel title="Không thể tải final project" tone="danger">{project.error.message}</StatePanel> : <>
      <Card className="p-6">
        <div className="mb-5 flex flex-wrap items-center gap-2"><span className="grid size-10 place-items-center rounded-2xl bg-[#edf0e7]"><Film className="size-5" /></span><h2 className="font-serif text-xl font-bold">Scene-based composition</h2>{project.data ? <Badge>project v{project.data.currentVersion}</Badge> : <Badge tone="warn">Chưa lưu</Badge>}</div>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Headline"><input className={inputClass} value={draft.headline} onChange={(event) => change("headline", event.target.value)} /></Field>
          <Field label="Lower third"><input className={inputClass} value={draft.lowerThird} onChange={(event) => change("lowerThird", event.target.value)} /></Field>
          <Field label="Music media asset ID (tùy chọn)"><input className={inputClass} value={draft.musicAssetId ?? ""} onChange={(event) => change("musicAssetId", event.target.value || null)} /></Field>
          <Field label="Ghi chú version"><input className={inputClass} value={draft.changeSummary} onChange={(event) => change("changeSummary", event.target.value)} /></Field>
          <Field label={`Music gain: ${draft.musicGainDb} dB`}><input className="w-full accent-[var(--moss)]" type="range" min={-60} max={0} value={draft.musicGainDb} onChange={(event) => change("musicGainDb", Number(event.target.value))} /></Field>
          <Field label={`Dialogue ducking: ${draft.dialogueDuckingDb} dB`}><input className="w-full accent-[var(--moss)]" type="range" min={-30} max={0} value={draft.dialogueDuckingDb} onChange={(event) => change("dialogueDuckingDb", Number(event.target.value))} /></Field>
        </div>
        <div className="mt-5 flex flex-wrap gap-3">{toggles.map(([key, label]) => <label key={key} className="flex items-center gap-2 rounded-full bg-[#f0f2ea] px-4 py-2 text-sm font-semibold"><input type="checkbox" checked={Boolean(draft[key])} onChange={(event) => change(key, event.target.checked as ProjectInput[typeof key])} />{label}</label>)}</div>
        {(save.error || start.error) ? <p className="mt-4 text-sm font-semibold text-[var(--coral)]">{save.error?.message ?? start.error?.message}</p> : null}
        <div className="mt-6 flex flex-wrap gap-3"><Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={save.isPending} onClick={() => save.mutate()}><Save className="mr-2 size-4" />Lưu composition version</Button><Button disabled={start.isPending || busy} onClick={() => start.mutate()}><Sparkles className="mr-2 size-4" />{busy ? "Đang render" : "Render final MP4"}</Button></div>
      </Card>
      <section className="mt-7"><h2 className="mb-4 font-serif text-2xl font-bold">Render history</h2>{renders.isLoading ? <SkeletonRows /> : renders.error ? <StatePanel title="Không thể tải render" tone="danger">{renders.error.message}</StatePanel> : renders.data?.items.length ? <div className="grid gap-4">{renders.data.items.map((item) => <RenderCard key={item.id} item={item} scope={scope} refresh={refresh} />)}</div> : <StatePanel title="Chưa có final render">Lưu composition và chọn take cho mọi scene trước khi render.</StatePanel>}</section>
    </>}
  </CampaignHeader>;
}

function RenderCard({ item, scope, refresh }: { item: FinalRender; scope: Scope; refresh: () => Promise<void> }) {
  const [notes, setNotes] = useState("");
  const review = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders/{renderJobId}/review", { params: { path: { ...scope, renderJobId: item.id } }, body: { action, version: item.version, notes } }); if (error || !data) throw apiError(error, "Không thể lưu review final render."); return data; }, onSuccess: refresh });
  const select = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders/{renderJobId}/select", { params: { path: { ...scope, renderJobId: item.id } } }); if (error || !data) throw apiError(error, "Chỉ render đã duyệt mới có thể chọn."); return data; }, onSuccess: refresh });
  const download = async () => { if (!item.outputAssetId) return; const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { clientId: scope.clientId, workspaceId: scope.workspaceId, assetId: item.outputAssetId } } }); if (error || !data) throw apiError(error, "Không thể mở final video."); window.open(data.url, "_blank", "noopener,noreferrer"); };
  const tone = item.status === "APPROVED" ? "good" : item.status === "FAILED" || item.status === "REJECTED" ? "danger" : item.status === "REVIEW_REQUIRED" ? "warn" : "neutral";
  return <Card className="p-5"><div className="flex flex-wrap items-center gap-2"><Badge tone={tone}>{item.status}</Badge>{item.selected ? <Badge tone="good">Campaign output</Badge> : null}{item.reused ? <Badge>Idempotent reuse</Badge> : null}<span className="text-xs text-[var(--muted)]">project v{item.videoProjectVersion} · job v{item.version}</span><span className="ml-auto font-mono text-xs text-[var(--muted)]">{item.id.slice(0, 8)}</span></div>{item.errorMessage ? <p className="mt-3 text-sm text-[var(--coral)]">{item.errorMessage}</p> : null}{item.outputHash ? <p className="mt-3 break-all font-mono text-xs text-[var(--muted)]">SHA-256 {item.outputHash}</p> : null}{item.status === "REVIEW_REQUIRED" ? <Field label="Ghi chú review final"><textarea className={`${textareaClass} mt-4`} value={notes} onChange={(event) => setNotes(event.target.value)} /></Field> : null}{(review.error || select.error) ? <p className="mt-3 text-sm text-[var(--coral)]">{review.error?.message ?? select.error?.message}</p> : null}<div className="mt-4 flex flex-wrap gap-2">{item.outputAssetId ? <Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void download()}><Download className="mr-2 size-4" />Mở MP4</Button> : null}{item.status === "REVIEW_REQUIRED" ? <><Button disabled={review.isPending} onClick={() => review.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt final</Button><Button className="bg-[var(--coral)]" disabled={review.isPending || !notes.trim()} onClick={() => review.mutate("REJECT")}><Ban className="mr-2 size-4" />Từ chối</Button></> : null}{item.status === "APPROVED" && !item.selected ? <Button disabled={select.isPending} onClick={() => select.mutate()}><Star className="mr-2 size-4" />Chọn campaign output</Button> : null}</div>{item.srtStorageKey && item.vttStorageKey ? <p className="mt-3 text-xs text-[var(--muted)]">Đã tạo SRT + VTT · thumbnail · normalized ffprobe metadata</p> : null}</Card>;
}
