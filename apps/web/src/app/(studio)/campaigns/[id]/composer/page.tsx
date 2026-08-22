"use client";

import type { components } from "@studio/api-client";
import { closestCenter, DndContext, KeyboardSensor, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, Check, Download, Film, GripVertical, Save, Sparkles, Star } from "lucide-react";
import { Suspense, useEffect, useMemo, useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { CampaignHeader, useCampaignRoute } from "@/components/campaign-workflow";
import { Badge, Button, Card, Field, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type ProjectInput = components["schemas"]["VideoProjectInput"];
type Scene = components["schemas"]["CampaignScene"];
type Generation = components["schemas"]["SceneGeneration"];
type MediaAsset = components["schemas"]["MediaAsset"];
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

const activeRenderStatuses = new Set(["QUEUED", "BUILDING_MANIFEST", "RENDERING", "VALIDATING", "UPLOADING"]);

function projectInput(project: components["schemas"]["VideoProject"] | null | undefined): ProjectInput {
  if (!project) return defaultProject;
  return {
    headline: project.headline,
    lowerThird: project.lowerThird,
    showPrice: project.showPrice,
    showDiscountCode: project.showDiscountCode,
    showCta: project.showCta,
    showWebsite: project.showWebsite,
    showPhone: project.showPhone,
    showQrCode: project.showQrCode,
    showDisclaimer: project.showDisclaimer,
    burnCaptions: project.burnCaptions,
    musicAssetId: project.musicAssetId,
    musicGainDb: project.musicGainDb,
    dialogueDuckingDb: project.dialogueDuckingDb,
    changeSummary: project.changeSummary,
    version: project.version,
  };
}

export default function ComposerPage() {
  return <Suspense fallback={<SkeletonRows />}><ComposerWorkspace /></Suspense>;
}

function ComposerWorkspace() {
  const scope = useCampaignRoute();
  const { canOperate } = usePermissions();
  const qc = useQueryClient();
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const project = useQuery({
    queryKey: ["video-project", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    retry: false,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/video-project", { params: { path: scope } });
      if (response.status === 404) return null;
      if (error || !data) throw apiError(error, "Không thể tải composition.");
      return data;
    },
  });
  const scenes = useQuery({
    queryKey: ["campaign-scenes", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể tải timeline scene.");
      return data;
    },
  });
  const media = useQuery({
    queryKey: ["media", scope.clientId, scope.workspaceId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets", { params: { path: { clientId: scope.clientId, workspaceId: scope.workspaceId }, query: {} } });
      if (error || !data) throw apiError(error, "Không thể tải media picker.");
      return data;
    },
  });
  const renders = useQuery({
    queryKey: ["final-renders", scope.campaignId],
    enabled: !!scope.clientId && !!scope.workspaceId,
    refetchInterval: (query) => query.state.data?.items.some((item) => activeRenderStatuses.has(item.status)) ? 2000 : false,
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope } });
      if (error || !data) throw apiError(error, "Không thể tải render history.");
      return data;
    },
  });
  const generationQueries = useQueries({
    queries: (scenes.data?.items ?? []).map((scene) => ({
      queryKey: ["scene-generations", scope.campaignId, scene.id],
      queryFn: async () => {
        const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/{sceneId}/generations", { params: { path: { ...scope, sceneId: scene.id } } });
        if (error || !data) throw apiError(error, `Không thể tải take cho ${scene.sceneKey}.`);
        return data;
      },
    })),
  });
  const sourceDraft = useMemo(() => projectInput(project.data), [project.data]);
  const [draftOverride, setDraft] = useState<ProjectInput | null>(null);
  const draft = draftOverride ?? sourceDraft;
  const dirty = draftOverride !== null && JSON.stringify(draftOverride) !== JSON.stringify(sourceDraft);
  const save = useMutation({
    mutationFn: async (submitted: ProjectInput) => {
      const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/video-project", { params: { path: scope }, body: { ...submitted, version: project.data?.version ?? 1 } });
      if (error || !data) throw apiError(error, "Không thể lưu composition; hãy tải lại nếu version đã thay đổi.");
      return data;
    },
    onSuccess: (data, submitted) => {
      qc.setQueryData(["video-project", scope.campaignId], data);
      setDraft((current) => current && JSON.stringify(current) !== JSON.stringify(submitted) ? current : null);
    },
  });
  const reorder = useMutation({
    mutationFn: async (sceneIds: string[]) => {
      const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/reorder", { params: { path: scope }, body: { sceneIds } });
      if (error || !data) throw apiError(error, "Không thể lưu thứ tự timeline.");
      return data;
    },
    onSuccess: (data) => qc.setQueryData(["campaign-scenes", scope.campaignId], data),
  });
  const startRender = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders", { params: { path: scope, header: { "Idempotency-Key": newIdempotencyKey() } } });
      if (error || !data) throw apiError(error, "Mỗi scene phải có một take đã duyệt và được chọn trước khi render.");
      return data;
    },
    onSuccess: async () => qc.invalidateQueries({ queryKey: ["final-renders", scope.campaignId] }),
  });

  useEffect(() => {
    if (!dirty || !canOperate || save.isPending) return;
    const timer = window.setTimeout(() => save.mutate(draft), 1200);
    return () => window.clearTimeout(timer);
  }, [canOperate, dirty, draft, save]);
  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (dirty) event.preventDefault(); };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const change = <K extends keyof ProjectInput>(key: K, value: ProjectInput[K]) => setDraft((current) => ({ ...(current ?? draft), [key]: value }));
  const handleDragEnd = (event: DragEndEvent) => {
    if (!canOperate || !scenes.data || !event.over || event.active.id === event.over.id) return;
    const ids = scenes.data.items.map((scene) => scene.id);
    const oldIndex = ids.indexOf(String(event.active.id));
    const newIndex = ids.indexOf(String(event.over.id));
    if (oldIndex < 0 || newIndex < 0) return;
    reorder.mutate(arrayMove(ids, oldIndex, newIndex));
  };
  const selectedByScene = new Map<string, Generation>();
  generationQueries.forEach((query, index) => {
    const scene = scenes.data?.items[index];
    const selected = query.data?.items.find((item) => item.selected);
    if (scene && selected) selectedByScene.set(scene.id, selected);
  });
  const selectedRender = renders.data?.items.find((item) => item.selected && item.outputAssetId)
    ?? renders.data?.items.find((item) => item.status === "APPROVED" && item.outputAssetId);
  const firstSelectedTake = [...selectedByScene.values()].find((item) => item.outputAssetId);
  const previewAssetId = selectedRender?.outputAssetId ?? firstSelectedTake?.outputAssetId ?? null;
  const musicAssets = media.data?.items.filter((item) => item.assetType === "AUDIO" && item.status !== "ARCHIVED") ?? [];
  const replacementAssets = media.data?.items.filter((item) => ["IMAGE", "VIDEO", "SCREEN_RECORDING"].includes(item.assetType) && item.status !== "ARCHIVED") ?? [];
  const busy = renders.data?.items.some((item) => activeRenderStatuses.has(item.status));
  const readyScenes = scenes.data?.items.filter((scene) => selectedByScene.has(scene.id)).length ?? 0;

  return <CampaignHeader active="/composer" title="Dựng & duyệt final" description="Dựng video từ các take đã chọn, render MP4, sau đó duyệt và chọn campaign output trước khi chuyển sang các kênh phân phối.">
    {project.isLoading || scenes.isLoading ? <SkeletonRows /> : project.error || scenes.error ? <StatePanel title="Không thể tải Composer" tone="danger">{project.error?.message ?? scenes.error?.message}</StatePanel> : <>
      <div className="mb-6 grid gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(300px,.55fr)]">
        <Card className="p-6">
          <div className="mb-5 flex flex-wrap items-center gap-2"><Film className="size-5" /><h2 className="font-serif text-xl font-bold">Thiết lập composition</h2>{project.data ? <Badge>v{project.data.currentVersion}</Badge> : <Badge tone="warn">Chưa lưu</Badge>}<span aria-live="polite" className="ml-auto text-xs font-semibold text-[var(--muted)]">{save.isPending ? "Đang tự lưu…" : save.isError ? "Tự lưu thất bại" : dirty ? "Chờ tự lưu" : "Đã lưu"}</span></div>
          <fieldset disabled={!canOperate} className="grid gap-4 md:grid-cols-2">
            <Field label="Headline"><input className={inputClass} value={draft.headline} onChange={(event) => change("headline", event.target.value)} /></Field>
            <Field label="Lower third"><input className={inputClass} value={draft.lowerThird} onChange={(event) => change("lowerThird", event.target.value)} /></Field>
            <Field label="Nhạc nền"><select className={inputClass} value={draft.musicAssetId ?? ""} onChange={(event) => change("musicAssetId", event.target.value || null)}><option value="">Không dùng nhạc</option>{musicAssets.map((asset) => <option key={asset.id} value={asset.id}>{asset.name}</option>)}</select></Field>
            <Field label="Ghi chú version"><input className={inputClass} value={draft.changeSummary} onChange={(event) => change("changeSummary", event.target.value)} /></Field>
            <Field label={`Music gain: ${draft.musicGainDb} dB`}><input className="w-full accent-[var(--moss)]" type="range" min={-60} max={0} value={draft.musicGainDb} onChange={(event) => change("musicGainDb", Number(event.target.value))} /></Field>
            <Field label={`Dialogue ducking: ${draft.dialogueDuckingDb} dB`}><input className="w-full accent-[var(--moss)]" type="range" min={-30} max={0} value={draft.dialogueDuckingDb} onChange={(event) => change("dialogueDuckingDb", Number(event.target.value))} /></Field>
          </fieldset>
          <div className="mt-5 flex flex-wrap gap-3">{([['showPrice', 'Giá'], ['showDiscountCode', 'Mã giảm giá'], ['showCta', 'CTA'], ['showWebsite', 'Website'], ['showPhone', 'Điện thoại'], ['showQrCode', 'QR'], ['showDisclaimer', 'Disclaimer'], ['burnCaptions', 'Burned captions']] as Array<[keyof ProjectInput, string]>).map(([key, label]) => <label key={key} className="flex items-center gap-2 rounded-full bg-[#f0f2ea] px-4 py-2 text-sm font-semibold"><input disabled={!canOperate} type="checkbox" checked={Boolean(draft[key])} onChange={(event) => change(key, event.target.checked as ProjectInput[typeof key])} />{label}</label>)}</div>
          {save.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{save.error.message}</p> : null}
          {canOperate ? <Button className="mt-5 bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" disabled={!dirty || save.isPending} onClick={() => save.mutate(draft)}><Save className="mr-2 size-4" />Lưu ngay</Button> : null}
        </Card>
        <Card className="p-5">
          <div className="mb-4 flex items-center justify-between"><h2 className="font-serif text-xl font-bold">Preview</h2><Badge tone={readyScenes === (scenes.data?.items.length ?? 0) && readyScenes > 0 ? "good" : "warn"}>{readyScenes}/{scenes.data?.items.length ?? 0} scene sẵn sàng</Badge></div>
          <div className="mx-auto aspect-[9/16] max-h-[520px] overflow-hidden rounded-3xl bg-[var(--ink)]"><AssetVideo assetId={previewAssetId} scope={scope} /></div>
          <p className="mt-3 text-xs leading-5 text-[var(--muted)]">Preview ưu tiên campaign output đã chọn, sau đó dùng take đã chọn đầu tiên. Exact text vẫn được render từ structured data.</p>
        </Card>
      </div>

      <section>
        <div className="mb-4 flex flex-wrap items-end justify-between gap-3"><div><h2 className="font-serif text-2xl font-bold">Timeline theo scene</h2><p className="mt-1 text-sm text-[var(--muted)]">Kéo thả bằng chuột hoặc bàn phím. Thứ tự được lưu ngay và làm mất hiệu lực approval downstream theo quy tắc backend.</p></div>{reorder.isPending ? <Badge tone="warn">Đang lưu thứ tự…</Badge> : null}</div>
        {scenes.data?.items.length ? <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}><SortableContext items={scenes.data.items.map((scene) => scene.id)} strategy={verticalListSortingStrategy}><div className="grid gap-4">{scenes.data.items.map((scene) => <SortableScene key={scene.id} scene={scene} selected={selectedByScene.get(scene.id)} scope={scope} assets={replacementAssets} canOperate={canOperate} refresh={async () => { await Promise.all([qc.invalidateQueries({ queryKey: ["campaign-scenes", scope.campaignId] }), qc.invalidateQueries({ queryKey: ["scene-generations", scope.campaignId, scene.id] })]); }} />)}</div></SortableContext></DndContext> : <StatePanel title="Chưa có timeline">Tạo và duyệt scenes trước khi mở Composer.</StatePanel>}
      </section>

      <section className="mt-8"><div className="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h2 className="font-serif text-2xl font-bold">Render & duyệt final</h2><p className="mt-1 text-sm text-[var(--muted)]">Render lại chỉ dùng manifest đã lưu, không gọi lại Seedance. Final MP4 phải được duyệt và chọn làm campaign output tại đây.</p></div>{canOperate ? <Button disabled={startRender.isPending || busy || dirty || readyScenes !== (scenes.data?.items.length ?? 0)} onClick={() => startRender.mutate()}><Sparkles className="mr-2 size-4" />{busy ? "Đang render" : dirty ? "Đợi tự lưu" : "Render MP4"}</Button> : null}</div>{startRender.error ? <StatePanel title="Không thể render" tone="danger">{startRender.error.message}</StatePanel> : renders.isLoading ? <SkeletonRows /> : renders.error ? <StatePanel title="Không thể tải render" tone="danger">{renders.error.message}</StatePanel> : renders.data?.items.length ? <div className="grid gap-4">{renders.data.items.map((item) => <RenderCard key={item.id} item={item} scope={scope} refresh={async () => qc.invalidateQueries({ queryKey: ["final-renders", scope.campaignId] })} />)}</div> : <StatePanel title="Chưa có final render">Chọn một take đã duyệt cho mọi scene, chờ autosave rồi render.</StatePanel>}</section>
    </>}
  </CampaignHeader>;
}

function SortableScene({ scene, selected, scope, assets, canOperate, refresh }: { scene: Scene; selected?: Generation; scope: Scope; assets: MediaAsset[]; canOperate: boolean; refresh: () => Promise<unknown> }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: scene.id, disabled: !canOperate });
  const style = { transform: transform ? `translate3d(${transform.x}px, ${transform.y}px, 0) scaleX(${transform.scaleX}) scaleY(${transform.scaleY})` : undefined, transition };
  return <div ref={setNodeRef} style={style}><Card className="p-5"><div className="flex flex-col gap-4 lg:flex-row lg:items-start"><button type="button" aria-label={`Sắp xếp ${scene.sceneKey}`} className="mt-1 cursor-grab rounded-xl p-2 text-[var(--muted)] hover:bg-[#edf0e7] disabled:cursor-default" disabled={!canOperate} {...attributes} {...listeners}><GripVertical className="size-5" /></button><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><span className="grid size-8 place-items-center rounded-full bg-[var(--ink)] text-sm font-bold text-white">{scene.order}</span><h3 className="font-serif text-lg font-bold">{scene.sceneKey}</h3><Badge tone={selected ? "good" : "warn"}>{selected ? "Đã chọn take" : "Thiếu take"}</Badge><span className="ml-auto text-xs text-[var(--muted)]">{scene.direction.durationSeconds}s</span></div><p className="mt-3 text-sm leading-6 text-[var(--muted)]">{scene.direction.dialogue}</p>{selected ? <SceneEditControls generation={selected} scope={scope} assets={assets} canOperate={canOperate} refresh={refresh} /> : <p className="mt-3 rounded-xl bg-[#fff8e8] p-3 text-xs font-semibold">Sang Duyệt take để duyệt và chọn một take cho scene này.</p>}</div>{selected?.outputAssetId ? <div className="aspect-video w-full overflow-hidden rounded-2xl bg-[var(--ink)] lg:w-64"><AssetVideo assetId={selected.outputAssetId} scope={scope} /></div> : null}</div></Card></div>;
}

function SceneEditControls({ generation, scope, assets, canOperate, refresh }: { generation: Generation; scope: Scope; assets: MediaAsset[]; canOperate: boolean; refresh: () => Promise<unknown> }) {
  const [edit, setEdit] = useState(generation.edit ?? { trimStartMs: 0, trimEndMs: null, muteAudio: false, transition: "CUT" as const, replacementAssetId: null, attachedProductAssetIds: [], subtitlePreview: true, version: 1 });
  const save = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/scenes/{sceneId}/generations/{generationId}/edit", { params: { path: { ...scope, sceneId: generation.sceneId, generationId: generation.id } }, body: edit });
      if (error || !data) throw apiError(error, "Không thể lưu edit metadata.");
      return data;
    },
    onSuccess: async (data) => { if (data.edit) setEdit(data.edit); await refresh(); },
  });
  if (!canOperate) return <p className="mt-3 text-xs text-[var(--muted)]">Edit metadata chỉ dành cho Operator/Admin.</p>;
  return <div className="mt-4 grid gap-3 rounded-2xl bg-[#f6f7f1] p-4 md:grid-cols-2 xl:grid-cols-4"><Field label="Trim đầu (ms)"><input className={inputClass} type="number" min={0} value={edit.trimStartMs} onChange={(event) => setEdit((current) => ({ ...current, trimStartMs: Number(event.target.value) }))} /></Field><Field label="Trim cuối (ms)"><input className={inputClass} type="number" min={1} value={edit.trimEndMs ?? ""} onChange={(event) => setEdit((current) => ({ ...current, trimEndMs: event.target.value ? Number(event.target.value) : null }))} /></Field><Field label="Transition"><select className={inputClass} value={edit.transition} onChange={(event) => setEdit((current) => ({ ...current, transition: event.target.value as typeof current.transition }))}><option value="CUT">Cut</option><option value="CROSSFADE">Crossfade</option><option value="FADE_TO_BLACK">Fade to black</option></select></Field><Field label="Media thay thế"><select className={inputClass} value={edit.replacementAssetId ?? ""} onChange={(event) => setEdit((current) => ({ ...current, replacementAssetId: event.target.value || null }))}><option value="">Dùng AI take</option>{assets.map((asset) => <option key={asset.id} value={asset.id}>{asset.name} · {asset.assetType}</option>)}</select></Field><label className="flex items-center gap-2 text-xs font-semibold"><input type="checkbox" checked={edit.muteAudio} onChange={(event) => setEdit((current) => ({ ...current, muteAudio: event.target.checked }))} />Tắt audio scene</label><label className="flex items-center gap-2 text-xs font-semibold"><input type="checkbox" checked={edit.subtitlePreview} onChange={(event) => setEdit((current) => ({ ...current, subtitlePreview: event.target.checked }))} />Hiện subtitle</label><div className="md:col-span-2 xl:col-span-2"><Button className="w-full" disabled={save.isPending} onClick={() => save.mutate()}><Save className="mr-2 size-4" />Lưu scene edit</Button>{save.error ? <p className="mt-2 text-xs text-[var(--coral)]">{save.error.message}</p> : null}</div></div>;
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
  if (!assetId) return <div className="grid h-full place-items-center p-5 text-center text-sm text-white/65">Chưa có video để preview</div>;
  if (download.isLoading) return <div className="grid h-full place-items-center text-sm text-white/65">Đang tải preview…</div>;
  if (download.error) return <div className="grid h-full place-items-center p-4 text-center text-xs text-white/70">{download.error.message}</div>;
  return <video className="h-full w-full object-contain" controls playsInline preload="metadata" src={download.data?.url} />;
}

function RenderCard({ item, scope, refresh }: { item: FinalRender; scope: Scope; refresh: () => Promise<unknown> }) {
  const { canReview } = usePermissions();
  const [notes, setNotes] = useState("");
  const review = useMutation({ mutationFn: async (action: "APPROVE" | "REJECT") => { const { data, error } = await api.PUT("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders/{renderJobId}/review", { params: { path: { ...scope, renderJobId: item.id } }, body: { action, version: item.version, notes } }); if (error || !data) throw apiError(error, "Không thể lưu review final render."); return data; }, onSuccess: refresh });
  const select = useMutation({ mutationFn: async () => { const { data, error } = await api.POST("/clients/{clientId}/workspaces/{workspaceId}/campaigns/{campaignId}/final-renders/{renderJobId}/select", { params: { path: { ...scope, renderJobId: item.id } } }); if (error || !data) throw apiError(error, "Chỉ render đã duyệt mới có thể chọn."); return data; }, onSuccess: refresh });
  const download = async () => { if (!item.outputAssetId) return; const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets/{assetId}/download", { params: { path: { clientId: scope.clientId, workspaceId: scope.workspaceId, assetId: item.outputAssetId } } }); if (error || !data) throw apiError(error, "Không thể mở final video."); window.open(data.url, "_blank", "noopener,noreferrer"); };
  const tone = item.status === "APPROVED" ? "good" : item.status === "FAILED" || item.status === "REJECTED" ? "danger" : item.status === "REVIEW_REQUIRED" ? "warn" : "neutral";
  return <Card className="p-5"><div className="flex flex-wrap items-center gap-2"><Badge tone={tone}>{item.status}</Badge>{item.selected ? <Badge tone="good">Campaign output</Badge> : null}{item.reused ? <Badge>Idempotent reuse</Badge> : null}<span className="text-xs text-[var(--muted)]">project v{item.videoProjectVersion} · job v{item.version}</span><span className="ml-auto font-mono text-xs text-[var(--muted)]">{item.id.slice(0, 8)}</span></div>{item.errorMessage ? <p className="mt-3 text-sm text-[var(--coral)]">{item.errorMessage}</p> : null}{canReview && item.status === "REVIEW_REQUIRED" ? <Field label="Ghi chú review final"><textarea className={`${textareaClass} mt-4`} value={notes} onChange={(event) => setNotes(event.target.value)} /></Field> : null}{(review.error || select.error) ? <p className="mt-3 text-sm text-[var(--coral)]">{review.error?.message ?? select.error?.message}</p> : null}<div className="mt-4 flex flex-wrap gap-2">{item.outputAssetId ? <Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => void download()}><Download className="mr-2 size-4" />Mở MP4</Button> : null}{canReview && item.status === "REVIEW_REQUIRED" ? <><Button disabled={review.isPending} onClick={() => review.mutate("APPROVE")}><Check className="mr-2 size-4" />Duyệt final</Button><Button className="bg-[var(--coral)]" disabled={review.isPending || !notes.trim()} onClick={() => review.mutate("REJECT")}><Ban className="mr-2 size-4" />Từ chối</Button></> : null}{canReview && item.status === "APPROVED" && !item.selected ? <Button disabled={select.isPending} onClick={() => select.mutate()}><Star className="mr-2 size-4" />Chọn campaign output</Button> : null}</div></Card>;
}
