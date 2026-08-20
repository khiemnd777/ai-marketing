"use client";

import type { components } from "@studio/api-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueries, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, ArrowRight, Boxes, CheckCircle2, CircleDashed, Clapperboard, ImageIcon, Mail, MapPin, PackageSearch, Palette, Pencil, Phone, Plus, Save, UsersRound } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { usePermissions } from "@/components/auth-context";
import { useStudioScope } from "@/components/studio-scope";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { clientFormSchema, type ClientFormValues } from "@/lib/client-form";
import { apiError, newIdempotencyKey } from "@/lib/problem";
import { studioRoutes } from "@/lib/studio-routes";

type Client = components["schemas"]["Client"];

export default function ClientDetailPage() {
  const { canOperate, isAdmin } = usePermissions();
  const { clientId } = useStudioScope();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const view = pathname.endsWith("/profile") ? "profile" : pathname.endsWith("/workspaces") ? "workspaces" : "overview";
  const [showCreate, setShowCreate] = useState(false);
  const [editing, setEditing] = useState(false);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState("");
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [timezone, setTimezone] = useState("Asia/Ho_Chi_Minh");

  const client = useQuery({
    queryKey: ["client", clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải khách hàng.");
      return data;
    },
  });
  const workspaces = useQuery({
    queryKey: ["workspaces", clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải workspace.");
      return data;
    },
  });
  const provider = useQuery({
    queryKey: ["provider-configuration", clientId],
    enabled: isAdmin && Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/provider-configuration", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải provider mode.");
      return data;
    },
  });
  const workspaceId = selectedWorkspaceId || workspaces.data?.items.find((item) => item.status === "ACTIVE")?.id || workspaces.data?.items[0]?.id || "";
  const stats = useWorkspaceStats(clientId, workspaceId);
  const create = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/clients/{clientId}/workspaces", { params: { path: { clientId }, header: { "Idempotency-Key": newIdempotencyKey() } }, body: { name, slug, timezone } });
      if (error || !data) throw apiError(error, "Không thể tạo workspace.");
      return data;
    },
    onSuccess: async (created) => {
      setShowCreate(false);
      setName("");
      setSlug("");
      setSelectedWorkspaceId(created.id);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["workspaces", clientId] }),
        queryClient.invalidateQueries({ queryKey: ["scope-workspaces", clientId] }),
      ]);
    },
  });

  if (!clientId) return <StatePanel title="Thiếu phạm vi khách hàng">Mở khách hàng từ danh mục để giữ đúng ranh giới dữ liệu.</StatePanel>;
  if (client.isLoading) return <SkeletonRows />;
  if (client.error) return <StatePanel title="Không thể tải khách hàng" tone="danger"><p role="alert">{client.error.message}</p><Button className="mt-4" onClick={() => client.refetch()}>Thử lại</Button></StatePanel>;
  const item = client.data!;

  return <>
    <Link href={studioRoutes.clients} className="mb-5 inline-flex min-h-11 items-center gap-2 text-sm font-bold text-[var(--moss)]"><ArrowLeft className="size-4" />Tất cả khách hàng</Link>
    {item.status === "ARCHIVED" ? <div role="status" className="mb-5 flex items-start gap-3 rounded-2xl border border-[#e7d29a] bg-[#fff7dc] p-4 text-sm"><AlertTriangle className="mt-0.5 size-5 shrink-0 text-[#79580b]" /><div><strong>Khách hàng đã được lưu trữ.</strong><p className="mt-1 text-[var(--muted)]">Dữ liệu vẫn có thể xem; hãy kích hoạt lại từ danh mục trước khi bắt đầu công việc mới.</p></div></div> : null}

    <Card className="mb-7 overflow-hidden">
      <div className="flex flex-col gap-5 p-6 md:flex-row md:items-center">
        <span className="grid size-16 shrink-0 place-items-center rounded-3xl bg-[var(--ink)] text-xl font-black text-[var(--lime)]">{initials(item.companyName)}</span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2"><h1 className="font-serif text-3xl font-bold tracking-tight md:text-4xl">{item.companyName}</h1><Badge tone={item.status === "ACTIVE" ? "good" : "neutral"}>{item.status === "ACTIVE" ? "Đang hoạt động" : "Đã lưu trữ"}</Badge></div>
          <p className="mt-2 flex flex-wrap items-center gap-2 text-sm text-[var(--muted)]"><span>{item.industry || "Chưa xác định ngành"}</span><span>·</span><MapPin className="size-4" /><span>{item.market || "Chưa xác định thị trường"}</span></p>
          <p className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-[var(--muted)]"><span className="flex items-center gap-1.5"><Mail className="size-4" />{item.contactEmail || "Chưa có email"}</span><span className="flex items-center gap-1.5"><Phone className="size-4" />{item.phone || "Chưa có điện thoại"}</span></p>
        </div>
        {canOperate ? <div className="flex flex-wrap gap-2"><Link href={studioRoutes.clientProfile(clientId)} className="inline-flex min-h-11 items-center justify-center rounded-full bg-white px-5 text-sm font-semibold text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-[#edf0e7]"><Pencil className="mr-2 size-4" />Chỉnh sửa hồ sơ</Link><Button onClick={() => setShowCreate((current) => !current)}><Plus className="mr-2 size-4" />Thêm workspace</Button></div> : null}
      </div>
      {isAdmin && provider.data ? <div className="border-t border-[var(--line)] bg-[#f7f7f1] px-6 py-3 text-xs font-semibold text-[var(--muted)]">Provider của khách hàng: <Link className="font-bold text-[var(--moss)] hover:underline" href={studioRoutes.clientProviders(clientId)}>{provider.data.demoMode ? "DEMO MODE" : "LIVE MODE"} · {provider.data.providers.filter((entry) => entry.configured).length}/{provider.data.providers.length} đã cấu hình</Link></div> : null}
    </Card>

    {showCreate && canOperate ? <WorkspaceCreateCard name={name} slug={slug} timezone={timezone} setName={setName} setSlug={setSlug} setTimezone={setTimezone} create={create} onClose={() => setShowCreate(false)} /> : null}

    {view === "profile" ? <ProfileView client={item} editing={editing} setEditing={setEditing} canOperate={canOperate} /> : null}
    {view === "workspaces" ? <WorkspacesView clientId={clientId} query={workspaces} onCreate={() => setShowCreate(true)} canOperate={canOperate} /> : null}
    {view === "overview" ? <OverviewView clientId={clientId} workspaceId={workspaceId} workspaces={workspaces} selectedWorkspaceId={selectedWorkspaceId} setSelectedWorkspaceId={setSelectedWorkspaceId} stats={stats} /> : null}
  </>;
}

function useWorkspaceStats(clientId: string, workspaceId: string) {
  const scope = { clientId, workspaceId };
  return useQueries({ queries: [
    { queryKey: ["brands", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/brands", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải thương hiệu."); return data; } },
    { queryKey: ["products", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/products", { params: { path: scope, query: {} } }); if (error || !data) throw apiError(error, "Không thể tải sản phẩm."); return data; } },
    { queryKey: ["media", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/media-assets", { params: { path: scope, query: {} } }); if (error || !data) throw apiError(error, "Không thể tải media."); return data; } },
    { queryKey: ["campaigns", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/campaigns", { params: { path: scope, query: {} } }); if (error || !data) throw apiError(error, "Không thể tải chiến dịch."); return data; } },
    { queryKey: ["characters", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), queryFn: async () => { const { data, error } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/characters", { params: { path: scope } }); if (error || !data) throw apiError(error, "Không thể tải nhân vật."); return data; } },
    { queryKey: ["meta-connection", clientId, workspaceId], enabled: Boolean(clientId && workspaceId), retry: false, queryFn: async () => { const { data } = await api.GET("/clients/{clientId}/workspaces/{workspaceId}/meta/connection", { params: { path: scope } }); return data ?? null; } },
  ] });
}

type WorkspaceStats = ReturnType<typeof useWorkspaceStats>;
type WorkspacesQuery = UseQueryResult<{ items: components["schemas"]["Workspace"][] }, Error>;

function OverviewView({ clientId, workspaceId, workspaces, selectedWorkspaceId, setSelectedWorkspaceId, stats }: { clientId: string; workspaceId: string; workspaces: WorkspacesQuery; selectedWorkspaceId: string; setSelectedWorkspaceId: (value: string) => void; stats: WorkspaceStats }) {
  if (workspaces.isLoading) return <SkeletonRows />;
  if (workspaces.error) return <StatePanel title="Không thể tải workspace" tone="danger">{workspaces.error.message}</StatePanel>;
  if (!workspaces.data?.items.length) return <StatePanel title="Chưa có workspace">Tạo workspace đầu tiên để mở hồ sơ thương hiệu, Product Truth, media và chiến dịch.</StatePanel>;
  const [brands, products, media, campaigns, characters, meta] = stats;
  const current = workspaces.data.items.find((entry) => entry.id === workspaceId)!;
  const loading = stats.some((query) => query.isLoading);
  const hasError = stats.some((query) => query.error);
  const activeBrands = brands.data?.items.filter((entry) => entry.status === "ACTIVE").length ?? 0;
  const approvedProducts = products.data?.items.filter((entry) => entry.status === "APPROVED").length ?? 0;
  const draftProducts = products.data?.items.filter((entry) => entry.status === "DRAFT").length ?? 0;
  const readyCharacters = characters.data?.items.filter((entry) => entry.status === "ACTIVE" && (entry.consentStatus === "APPROVED" || entry.consentStatus === "NOT_REQUIRED")).length ?? 0;

  return <>
    <section className="mb-7" aria-labelledby="active-workspace-title">
      <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div><p className="text-xs font-black uppercase tracking-[0.14em] text-[var(--moss)]">Workspace đang làm việc</p><h2 id="active-workspace-title" className="mt-1 font-serif text-2xl font-bold">{current.name}</h2><p className="mt-1 text-sm text-[var(--muted)]">{current.timezone} · {current.status === "ACTIVE" ? "Đang hoạt động" : "Đã lưu trữ"}</p></div>
        <div className="flex flex-col gap-2 sm:flex-row"><label><span className="sr-only">Chọn workspace để xem tổng quan</span><select className={`${inputClass} sm:min-w-64`} value={selectedWorkspaceId || workspaceId} onChange={(event) => setSelectedWorkspaceId(event.target.value)}>{workspaces.data.items.map((entry) => <option key={entry.id} value={entry.id}>{entry.name}</option>)}</select></label><Link href={studioRoutes.workspace(clientId, workspaceId)} className="inline-flex min-h-11 items-center justify-center rounded-full bg-[var(--ink)] px-5 text-sm font-semibold text-white hover:bg-[var(--moss)]">Mở workspace<ArrowRight className="ml-2 size-4" /></Link></div>
      </div>
      {loading ? <SkeletonRows /> : hasError ? <StatePanel title="Một phần tổng quan chưa tải được" tone="danger">Mở workspace để tải lại module tương ứng.</StatePanel> : <>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
          <Metric icon={Palette} label="Thương hiệu" value={brands.data?.items.length ?? 0} detail={`${activeBrands} đang active`} />
          <Metric icon={PackageSearch} label="Sản phẩm" value={products.data?.items.length ?? 0} detail={`${approvedProducts} đã duyệt`} />
          <Metric icon={ImageIcon} label="Media" value={media.data?.items.length ?? 0} detail={`${media.data?.items.filter((entry) => entry.status === "APPROVED").length ?? 0} đã duyệt`} />
          <Metric icon={Clapperboard} label="Chiến dịch" value={campaigns.data?.items.length ?? 0} detail={`${campaigns.data?.items.filter((entry) => entry.status !== "ARCHIVED").length ?? 0} đang theo dõi`} />
          <Metric icon={UsersRound} label="Nhân vật" value={characters.data?.items.length ?? 0} detail={`${readyCharacters} khả dụng`} />
        </div>
        <Card className="mt-6 p-6"><h2 className="font-serif text-xl font-bold">Sẵn sàng vận hành</h2><p className="mt-1 text-sm text-[var(--muted)]">Chỉ báo được tính từ dữ liệu đang lưu trong workspace này.</p><div className="mt-5 grid gap-3 md:grid-cols-2">
          <Readiness ready={activeBrands > 0} label={activeBrands > 0 ? "Có hồ sơ thương hiệu active" : "Chưa có hồ sơ thương hiệu active"} href={studioRoutes.brands(clientId, workspaceId)} />
          <Readiness ready={approvedProducts > 0} label={approvedProducts > 0 ? `Có ${approvedProducts} sản phẩm đã duyệt` : draftProducts > 0 ? `Có ${draftProducts} sản phẩm đang ở bản nháp` : "Chưa có Product Truth"} href={studioRoutes.products(clientId, workspaceId)} />
          <Readiness ready={readyCharacters >= 2} label={readyCharacters >= 2 ? `Có ${readyCharacters} nhân vật khả dụng` : "Cần ít nhất 2 nhân vật hợp lệ"} href={studioRoutes.characters(clientId, workspaceId)} />
          <Readiness ready={meta.data?.status === "CONNECTED"} label={meta.data ? `Meta: ${meta.data.status}` : "Meta chưa kết nối"} href={studioRoutes.meta(clientId, workspaceId)} />
        </div></Card>
      </>}
    </section>
    <section><div className="mb-4 flex items-center justify-between"><h2 className="font-serif text-2xl font-bold">Workspaces</h2><Link className="inline-flex min-h-11 items-center gap-2 text-sm font-bold text-[var(--moss)]" href={studioRoutes.clientWorkspaces(clientId)}>Xem tất cả<ArrowRight className="size-4" /></Link></div><WorkspaceCards clientId={clientId} items={workspaces.data.items.slice(0, 4)} /></section>
  </>;
}

function ProfileView({ client, editing, setEditing, canOperate }: { client: Client; editing: boolean; setEditing: (value: boolean) => void; canOperate: boolean }) {
  return <section><PageHeader headingLevel="h2" eyebrow="Hồ sơ khách hàng" title="Hồ sơ & liên hệ" description="Thông tin nội bộ dùng để nhận diện đúng khách hàng và phối hợp vận hành." action={canOperate && !editing ? <Button className="bg-white text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-[#edf0e7]" onClick={() => setEditing(true)}><Pencil className="mr-2 size-4" />Chỉnh sửa</Button> : undefined} />
    {editing ? <ClientEditor key={client.version} data={client} onClose={() => setEditing(false)} /> : <Card className="grid gap-5 p-6 md:grid-cols-2"><Info label="Tên công ty" value={client.companyName} /><Info label="Ngành" value={client.industry} /><Info label="Thị trường" value={client.market} /><Info label="Người liên hệ" value={client.contactName} /><Info label="Email" value={client.contactEmail} /><Info label="Điện thoại" value={client.phone} /><div className="md:col-span-2"><Info label="Ghi chú nội bộ" value={client.internalNotes} /></div></Card>}
  </section>;
}

function WorkspacesView({ clientId, query, onCreate, canOperate }: { clientId: string; query: WorkspacesQuery; onCreate: () => void; canOperate: boolean }) {
  return <section><PageHeader headingLevel="h2" eyebrow="Khách hàng" title="Workspaces" description="Mỗi workspace là một ranh giới dữ liệu độc lập cho thương hiệu hoặc đơn vị kinh doanh." action={canOperate ? <Button onClick={onCreate}><Plus className="mr-2 size-4" />Thêm workspace</Button> : undefined} />
    {query.isLoading ? <SkeletonRows /> : query.error ? <StatePanel title="Không thể tải workspace" tone="danger">{query.error.message}</StatePanel> : query.data?.items.length ? <WorkspaceCards clientId={clientId} items={query.data.items} /> : <StatePanel title="Chưa có workspace">Tạo workspace đầu tiên để bắt đầu cấu hình thương hiệu.</StatePanel>}
  </section>;
}

function WorkspaceCards({ clientId, items }: { clientId: string; items: components["schemas"]["Workspace"][] }) {
  return <div className="grid gap-4 md:grid-cols-2">{items.map((item) => <Link key={item.id} href={studioRoutes.workspace(clientId, item.id)}><Card className="flex h-full items-center gap-4 p-5 transition hover:-translate-y-0.5 hover:border-[var(--moss)]"><span className="grid size-12 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><Boxes className="size-5 text-[var(--moss)]" /></span><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h3 className="font-serif text-xl font-bold">{item.name}</h3><Badge tone={item.status === "ACTIVE" ? "good" : "neutral"}>{item.status === "ACTIVE" ? "Đang hoạt động" : "Đã lưu trữ"}</Badge></div><p className="mt-1 text-sm text-[var(--muted)]">{item.slug} · {item.timezone}</p></div><ArrowRight className="size-5 shrink-0 text-[var(--moss)]" /></Card></Link>)}</div>;
}

function Metric({ icon: Icon, label, value, detail }: { icon: typeof Palette; label: string; value: number; detail: string }) {
  return <Card className="flex items-center gap-4 p-5"><span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><Icon className="size-5 text-[var(--moss)]" /></span><span><span className="block font-serif text-2xl font-bold">{value}</span><span className="block text-sm font-semibold">{label}</span><span className="block text-xs text-[var(--muted)]">{detail}</span></span></Card>;
}

function Readiness({ ready, label, href }: { ready: boolean; label: string; href: string }) {
  return <Link href={href} className="flex min-h-12 items-center gap-3 rounded-2xl border border-[var(--line)] bg-white px-4 py-3 text-sm font-semibold hover:border-[var(--moss)]">{ready ? <CheckCircle2 className="size-5 shrink-0 text-[#28643c]" /> : <CircleDashed className="size-5 shrink-0 text-[#9a6811]" />}<span className="flex-1">{label}</span><ArrowRight className="size-4 text-[var(--muted)]" /></Link>;
}

function Info({ label, value }: { label: string; value?: string | null }) {
  return <div><p className="text-xs font-black uppercase tracking-[0.12em] text-[var(--muted)]">{label}</p><p className="mt-2 whitespace-pre-wrap text-sm leading-6">{value || "Chưa cập nhật"}</p></div>;
}

function WorkspaceCreateCard({ name, slug, timezone, setName, setSlug, setTimezone, create, onClose }: { name: string; slug: string; timezone: string; setName: (value: string) => void; setSlug: (value: string) => void; setTimezone: (value: string) => void; create: UseMutationResult<components["schemas"]["Workspace"], Error, void>; onClose: () => void }) {
  return <Card className="mb-7 p-6"><h2 className="font-serif text-xl font-bold">Thêm workspace</h2><p className="mt-1 text-sm text-[var(--muted)]">Workspace mới được cô lập theo khách hàng hiện tại.</p><div className="mt-5 grid gap-4 md:grid-cols-3"><Field label="Tên workspace"><input className={inputClass} value={name} onChange={(event) => { setName(event.target.value); if (!slug) setSlug(slugify(event.target.value)); }} /></Field><Field label="Slug"><input className={inputClass} value={slug} onChange={(event) => setSlug(slugify(event.target.value))} /></Field><Field label="Múi giờ"><input className={inputClass} value={timezone} onChange={(event) => setTimezone(event.target.value)} /></Field></div>{create.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{create.error.message}</p> : null}<div className="mt-5 flex flex-wrap gap-3"><Button disabled={create.isPending || name.trim().length < 2 || !slug} onClick={() => create.mutate()}>{create.isPending ? "Đang tạo…" : "Tạo workspace"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={onClose}>Hủy</Button></div></Card>;
}

function ClientEditor({ data, onClose }: { data: Client; onClose: () => void }) {
  const queryClient = useQueryClient();
  const form = useForm<ClientFormValues>({ resolver: zodResolver(clientFormSchema), defaultValues: { companyName: data.companyName, contactName: data.contactName, contactEmail: data.contactEmail ?? "", phone: data.phone ?? "", industry: data.industry, market: data.market, internalNotes: data.internalNotes } });
  const save = useMutation({
    mutationFn: async (values: ClientFormValues) => { const { data: updated, error } = await api.PUT("/clients/{clientId}", { params: { path: { clientId: data.id } }, body: { ...values, contactEmail: values.contactEmail || null, phone: values.phone || null, version: data.version } }); if (error || !updated) throw apiError(error, "Không thể cập nhật khách hàng. Dữ liệu có thể đã thay đổi; hãy tải lại trước khi lưu."); return updated; },
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ["client", data.id] }), queryClient.invalidateQueries({ queryKey: ["clients"] }), queryClient.invalidateQueries({ queryKey: ["scope-clients"] })]); onClose(); },
  });
  const close = () => { if (!form.formState.isDirty || window.confirm("Bỏ các thay đổi hồ sơ chưa lưu?")) onClose(); };
  return <Card className="p-6"><h2 className="font-serif text-xl font-bold">Chỉnh sửa khách hàng</h2><form noValidate onSubmit={form.handleSubmit((values) => save.mutate(values))}><div className="mt-5 grid gap-4 md:grid-cols-2"><Field label="Tên công ty" error={form.formState.errors.companyName?.message}><input className={inputClass} {...form.register("companyName")} aria-invalid={Boolean(form.formState.errors.companyName)} /></Field><Field label="Người liên hệ" error={form.formState.errors.contactName?.message}><input className={inputClass} {...form.register("contactName")} /></Field><Field label="Email" error={form.formState.errors.contactEmail?.message}><input className={inputClass} type="email" {...form.register("contactEmail")} aria-invalid={Boolean(form.formState.errors.contactEmail)} /></Field><Field label="Điện thoại" error={form.formState.errors.phone?.message}><input className={inputClass} {...form.register("phone")} /></Field><Field label="Ngành" error={form.formState.errors.industry?.message}><input className={inputClass} {...form.register("industry")} /></Field><Field label="Thị trường" error={form.formState.errors.market?.message}><input className={inputClass} {...form.register("market")} /></Field><div className="md:col-span-2"><Field label="Ghi chú nội bộ" error={form.formState.errors.internalNotes?.message}><textarea className={textareaClass} {...form.register("internalNotes")} /></Field></div></div>{save.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{save.error.message}</p> : null}<div className="mt-5 flex flex-wrap gap-3"><Button type="submit" disabled={save.isPending || !form.formState.isDirty}><Save className="mr-2 size-4" />{save.isPending ? "Đang lưu…" : "Lưu thay đổi"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={close}>Hủy</Button></div></form></Card>;
}

function initials(value: string) { return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join("") || "KH"; }
function slugify(value: string) { return value.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/đ/g, "d").replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""); }
