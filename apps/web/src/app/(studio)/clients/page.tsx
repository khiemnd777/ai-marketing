"use client";

import type { components } from "@studio/api-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, ArrowLeft, ArrowRight, Building2, Mail, MapPin, Plus, RotateCcw, Search, X } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { usePermissions } from "@/components/auth-context";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { clientFormSchema, type ClientFormValues as Values } from "@/lib/client-form";
import { includeCurrentOption, industryOptions, marketOptions } from "@/lib/form-options";
import { apiError, newIdempotencyKey } from "@/lib/problem";
import { studioRoutes } from "@/lib/studio-routes";

type Client = components["schemas"]["Client"];
type LifecycleStatus = components["schemas"]["LifecycleStatus"];

export default function ClientsPage() {
  return <Suspense fallback={<SkeletonRows />}><ClientsContent /></Suspense>;
}

function ClientsContent() {
  const { canOperate, isAdmin } = usePermissions();
  const queryClient = useQueryClient();
  const pathname = usePathname();
  const router = useRouter();
  const params = useSearchParams();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const initialPage = Math.max(1, Number(params.get("page") ?? "1") || 1);
  const initialStatus = params.get("status");
  const [search, setSearch] = useState(params.get("search") ?? "");
  const [debouncedSearch, setDebouncedSearch] = useState(search);
  const [statusFilter, setStatusFilter] = useState<"" | LifecycleStatus>(initialStatus === "ACTIVE" || initialStatus === "ARCHIVED" ? initialStatus : "");
  const [page, setPage] = useState(initialPage);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      setDebouncedSearch(search.trim());
      setPage(1);
    }, 300);
    return () => window.clearTimeout(timeout);
  }, [search]);

  useEffect(() => {
    const next = new URLSearchParams(params.toString());
    if (debouncedSearch) next.set("search", debouncedSearch); else next.delete("search");
    if (statusFilter) next.set("status", statusFilter); else next.delete("status");
    if (page > 1) next.set("page", String(page)); else next.delete("page");
    const serialized = next.toString();
    const current = params.toString();
    if (serialized !== current) router.replace(serialized ? `${pathname}?${serialized}` : pathname, { scroll: false });
  }, [debouncedSearch, page, params, pathname, router, statusFilter]);

  const query = useQuery({
    queryKey: ["clients", debouncedSearch, statusFilter, page],
    queryFn: async () => {
      const { data, error } = await api.GET("/clients", { params: { query: { search: debouncedSearch || undefined, status: statusFilter || undefined, page, pageSize: 20 } } });
      if (error || !data) throw apiError(error, "Không thể tải khách hàng.");
      return data;
    },
  });
  const clientStatus = useMutation({
    mutationFn: async ({ item, next }: { item: Client; next: LifecycleStatus }) => {
      const { data, error } = await api.PATCH("/clients/{clientId}/status", { params: { path: { clientId: item.id } }, body: { status: next, version: item.version! } });
      if (error || !data) throw apiError(error, "Không thể đổi trạng thái. Dữ liệu có thể đã được cập nhật ở nơi khác.");
      return data;
    },
    onSuccess: async () => Promise.all([
      queryClient.invalidateQueries({ queryKey: ["clients"] }),
      queryClient.invalidateQueries({ queryKey: ["scope-clients"] }),
    ]),
  });

  const changeStatus = (item: Client) => {
    const next: LifecycleStatus = item.status === "ACTIVE" ? "ARCHIVED" : "ACTIVE";
    const message = next === "ARCHIVED"
      ? `Lưu trữ “${item.companyName}”? Người dùng vẫn có thể xem dữ liệu nhưng không nên bắt đầu công việc mới trong phạm vi này.`
      : `Kích hoạt lại “${item.companyName}”?`;
    if (window.confirm(message)) clientStatus.mutate({ item, next });
  };

  const pageMeta = query.data?.page;
  const items = query.data?.items ?? [];
  return <>
    <PageHeader
      eyebrow="Danh mục nội bộ"
      title="Khách hàng"
      description="Điểm bắt đầu để quản lý hồ sơ khách hàng, workspace và toàn bộ hoạt động marketing liên quan."
      action={canOperate ? <Button ref={triggerRef} onClick={() => setShowCreate(true)}><Plus className="mr-2 size-4" />Thêm khách hàng</Button> : undefined}
    />

    <Card className="mb-5 grid gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_14rem]">
      <label className="flex min-h-11 items-center gap-3 rounded-2xl border border-[var(--line)] bg-white px-4">
        <Search className="size-4 shrink-0 text-[var(--muted)]" />
        <span className="sr-only">Tìm khách hàng</span>
        <input className="min-w-0 flex-1 bg-transparent text-sm outline-none" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Tìm công ty hoặc người liên hệ" />
      </label>
      <label className="grid">
        <span className="sr-only">Lọc trạng thái khách hàng</span>
        <select className={inputClass} value={statusFilter} onChange={(event) => { setStatusFilter(event.target.value as "" | LifecycleStatus); setPage(1); }}>
          <option value="">Tất cả trạng thái</option>
          <option value="ACTIVE">Đang hoạt động</option>
          <option value="ARCHIVED">Đã lưu trữ</option>
        </select>
      </label>
    </Card>

    <div className="mb-3 flex items-center justify-between gap-4">
      <p className="text-sm font-semibold text-[var(--muted)]">{pageMeta ? `${pageMeta.totalItems.toLocaleString("vi-VN")} khách hàng` : "Đang tải danh mục"}</p>
      {clientStatus.error ? <p role="alert" className="text-sm font-semibold text-[var(--coral)]">{clientStatus.error.message}</p> : null}
    </div>

    {query.isLoading ? <SkeletonRows /> : query.error ? (
      <StatePanel title="Không thể tải khách hàng" tone="danger"><p role="alert">{query.error.message}</p><Button className="mt-4" onClick={() => query.refetch()}>Thử lại</Button></StatePanel>
    ) : items.length === 0 ? (
      <StatePanel title={debouncedSearch || statusFilter ? "Không tìm thấy khách hàng" : "Chưa có khách hàng"}>
        {debouncedSearch || statusFilter ? "Thử từ khóa khác hoặc bỏ bộ lọc trạng thái." : "Tạo khách hàng đầu tiên để mở workspace và hồ sơ thương hiệu."}
      </StatePanel>
    ) : <>
      <div className="mb-2 hidden grid-cols-[minmax(0,1.4fr)_minmax(12rem,.8fr)_minmax(11rem,.7fr)_auto] gap-5 px-5 text-xs font-black uppercase tracking-[0.12em] text-[var(--muted)] lg:grid">
        <span>Khách hàng</span><span>Liên hệ</span><span>Cập nhật</span><span className="w-28">Trạng thái</span>
      </div>
      <div className="grid gap-3">
        {items.map((item) => <Card key={item.id} className="grid gap-4 rounded-2xl p-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(12rem,.8fr)_minmax(11rem,.7fr)_auto] lg:items-center lg:gap-5 lg:shadow-none">
          <div className="flex min-w-0 items-start gap-4">
            <span className="grid size-12 shrink-0 place-items-center rounded-2xl bg-[#edf0e7] text-sm font-black text-[var(--moss)]">{initials(item.companyName)}</span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <Link className="font-serif text-xl font-bold hover:text-[var(--moss)]" href={studioRoutes.client(item.id)}>{item.companyName}</Link>
                <Badge tone={item.status === "ACTIVE" ? "good" : "neutral"}>{item.status === "ACTIVE" ? "Đang hoạt động" : "Đã lưu trữ"}</Badge>
              </div>
              <p className="mt-1 flex flex-wrap items-center gap-1.5 text-sm text-[var(--muted)]"><Building2 className="size-3.5" />{item.industry || "Chưa xác định ngành"}<span>·</span><MapPin className="size-3.5" />{item.market || "Chưa xác định thị trường"}</p>
            </div>
          </div>
          <div className="text-sm">
            <p className="font-semibold">{item.contactName || "Chưa có người liên hệ"}</p>
            <p className="mt-1 flex items-center gap-1.5 truncate text-xs text-[var(--muted)]"><Mail className="size-3.5" />{item.contactEmail || "Chưa có email"}</p>
          </div>
          <div className="text-sm text-[var(--muted)]">
            <span className="lg:hidden">Cập nhật </span>{new Date(item.updatedAt).toLocaleDateString("vi-VN")}
          </div>
          <div className="flex min-w-28 flex-wrap items-center justify-end gap-2">
            {isAdmin ? <button type="button" className="grid size-11 place-items-center rounded-full text-[var(--muted)] hover:bg-[#edf0e7] hover:text-[var(--ink)]" aria-label={item.status === "ACTIVE" ? `Lưu trữ ${item.companyName}` : `Kích hoạt ${item.companyName}`} disabled={clientStatus.isPending} onClick={() => changeStatus(item)}>{item.status === "ACTIVE" ? <Archive className="size-4" /> : <RotateCcw className="size-4" />}</button> : null}
            <Link href={studioRoutes.client(item.id)} className="inline-flex min-h-11 items-center gap-2 rounded-full px-3 text-sm font-bold text-[var(--moss)] hover:bg-[#edf0e7]">Mở <ArrowRight className="size-4" /></Link>
          </div>
        </Card>)}
      </div>
    </>}

    {pageMeta && pageMeta.totalPages > 1 ? <nav aria-label="Phân trang khách hàng" className="mt-6 flex items-center justify-center gap-3">
      <Button className="bg-white px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-[#edf0e7]" disabled={page <= 1 || query.isFetching} onClick={() => setPage((current) => Math.max(1, current - 1))}><ArrowLeft className="mr-2 size-4" />Trước</Button>
      <span className="text-sm font-semibold text-[var(--muted)]">Trang {pageMeta.number}/{pageMeta.totalPages}</span>
      <Button className="bg-white px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-[#edf0e7]" disabled={page >= pageMeta.totalPages || query.isFetching} onClick={() => setPage((current) => current + 1)}>Sau<ArrowRight className="ml-2 size-4" /></Button>
    </nav> : null}

    {showCreate ? <CreateClientDrawer onClose={() => { setShowCreate(false); triggerRef.current?.focus(); }} /> : null}
  </>;
}

function CreateClientDrawer({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const firstFieldRef = useRef<HTMLInputElement | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const form = useForm<Values>({
    resolver: zodResolver(clientFormSchema),
    defaultValues: { companyName: "", contactName: "", contactEmail: "", phone: "", industry: "", market: "Việt Nam", internalNotes: "" },
  });
  const selectedIndustry = useWatch({ control: form.control, name: "industry" });
  const selectedMarket = useWatch({ control: form.control, name: "market" });
  const create = useMutation({
    mutationFn: async (values: Values) => {
      const { data, error } = await api.POST("/clients", { params: { header: { "Idempotency-Key": newIdempotencyKey() } }, body: { ...values, contactEmail: values.contactEmail || null, phone: values.phone || null } });
      if (error || !data) throw apiError(error, "Không thể tạo khách hàng.");
      return data;
    },
    onSuccess: async (client) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["clients"] }),
        queryClient.invalidateQueries({ queryKey: ["scope-clients"] }),
      ]);
      router.push(studioRoutes.client(client.id));
    },
  });
  const close = () => {
    if (form.formState.isDirty && !window.confirm("Bỏ các thông tin khách hàng chưa lưu?")) return;
    onClose();
  };

  useEffect(() => {
    firstFieldRef.current?.focus();
  }, []);

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const keydown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        if (!form.formState.isDirty || window.confirm("Bỏ các thông tin khách hàng chưa lưu?")) onClose();
      }
      if (event.key !== "Tab" || !panelRef.current) return;
      const focusable = [...panelRef.current.querySelectorAll<HTMLElement>("button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled])")];
      const first = focusable[0];
      const last = focusable.at(-1);
      if (!first || !last) return;
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", keydown);
    return () => { document.body.style.overflow = previousOverflow; document.removeEventListener("keydown", keydown); };
  }, [form.formState.isDirty, onClose]);

  const company = form.register("companyName");
  return <div className="fixed inset-0 z-[70] flex justify-end bg-[var(--ink)]/45 backdrop-blur-[1px]" onMouseDown={(event) => { if (event.target === event.currentTarget) close(); }}>
    <div ref={panelRef} role="dialog" aria-modal="true" aria-labelledby="create-client-title" className="h-full w-full overflow-y-auto bg-[var(--panel)] shadow-2xl sm:max-w-xl">
      <div className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--line)] bg-[var(--panel)]/95 px-5 py-4 backdrop-blur sm:px-7">
        <div><p className="text-xs font-black uppercase tracking-[0.14em] text-[var(--moss)]">Khách hàng</p><h2 id="create-client-title" className="mt-1 font-serif text-2xl font-bold">Thêm khách hàng</h2></div>
        <button type="button" className="grid size-11 place-items-center rounded-full hover:bg-[#edf0e7]" aria-label="Đóng" onClick={close}><X className="size-5" /></button>
      </div>
      <form className="grid gap-6 p-5 sm:p-7" noValidate onSubmit={form.handleSubmit((values) => create.mutate(values))}>
        <fieldset className="grid gap-4"><legend className="mb-4 font-serif text-lg font-bold">Thông tin chính</legend>
          <Field label="Tên công ty" error={form.formState.errors.companyName?.message}><input className={inputClass} {...company} ref={(element) => { company.ref(element); firstFieldRef.current = element; }} aria-invalid={Boolean(form.formState.errors.companyName)} /></Field>
          <div className="grid gap-4 sm:grid-cols-2"><Field label="Ngành" error={form.formState.errors.industry?.message}><select className={inputClass} {...form.register("industry")}><option value="">Chưa chọn</option>{includeCurrentOption(industryOptions, selectedIndustry).map((option)=><option key={option.value} value={option.value}>{option.label}</option>)}</select></Field><Field label="Thị trường" error={form.formState.errors.market?.message}><select className={inputClass} {...form.register("market")}><option value="">Chưa chọn</option>{includeCurrentOption(marketOptions, selectedMarket).map((option)=><option key={option.value} value={option.value}>{option.label}</option>)}</select></Field></div>
        </fieldset>
        <fieldset className="grid gap-4"><legend className="mb-4 font-serif text-lg font-bold">Người liên hệ</legend>
          <Field label="Tên người liên hệ" error={form.formState.errors.contactName?.message}><input className={inputClass} {...form.register("contactName")} /></Field>
          <div className="grid gap-4 sm:grid-cols-2"><Field label="Email" error={form.formState.errors.contactEmail?.message}><input className={inputClass} type="email" {...form.register("contactEmail")} aria-invalid={Boolean(form.formState.errors.contactEmail)} /></Field><Field label="Điện thoại" error={form.formState.errors.phone?.message}><input className={inputClass} {...form.register("phone")} /></Field></div>
        </fieldset>
        <Field label="Ghi chú nội bộ" error={form.formState.errors.internalNotes?.message}><textarea className={textareaClass} {...form.register("internalNotes")} /></Field>
        {create.error ? <p role="alert" className="rounded-2xl bg-[#fff4ef] p-4 text-sm font-semibold text-[var(--coral)]">{create.error.message}</p> : null}
        <div className="sticky bottom-0 flex flex-col-reverse gap-3 border-t border-[var(--line)] bg-[var(--panel)] py-4 sm:flex-row sm:justify-end"><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={close}>Hủy</Button><Button type="submit" disabled={create.isPending}>{create.isPending ? "Đang tạo…" : "Tạo khách hàng"}</Button></div>
      </form>
    </div>
  </div>;
}

function initials(value: string) {
  return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join("") || "KH";
}
