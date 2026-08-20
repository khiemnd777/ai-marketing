"use client";

import type { components } from "@studio/api-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, Building2, Plus, RotateCcw, Search } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { usePermissions } from "@/components/auth-context";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass, textareaClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type Client = components["schemas"]["Client"];
const schema = z.object({ companyName: z.string().min(2).max(200), contactName: z.string().max(160), contactEmail: z.union([z.email(), z.literal("")]), phone: z.string().max(40), industry: z.string().max(160), market: z.string().max(160), internalNotes: z.string().max(10000) });
type Values = z.infer<typeof schema>;

export default function ClientsPage() {
  const { canOperate, isAdmin } = usePermissions();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const query = useQuery({
    queryKey: ["clients", search],
    queryFn: async () => { const { data, error } = await api.GET("/clients", { params: { query: { search: search || undefined, page: 1, pageSize: 100 } } }); if (error || !data) throw apiError(error, "Không thể tải khách hàng."); return data; },
  });
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: { companyName: "", contactName: "", contactEmail: "", phone: "", industry: "", market: "Việt Nam", internalNotes: "" } });
  const create = useMutation({
    mutationFn: async (values: Values) => { const { data, error } = await api.POST("/clients", { params: { header: { "Idempotency-Key": newIdempotencyKey() } }, body: { ...values, contactEmail: values.contactEmail || null, phone: values.phone || null } }); if (error || !data) throw apiError(error, "Không thể tạo khách hàng."); return data; },
    onSuccess: async () => { form.reset(); setShowCreate(false); await queryClient.invalidateQueries({ queryKey: ["clients"] }); },
  });
  const status = useMutation({
    mutationFn: async ({ item, next }: { item: Client; next: "ACTIVE" | "ARCHIVED" }) => { const { data, error } = await api.PATCH("/clients/{clientId}/status", { params: { path: { clientId: item.id } }, body: { status: next, version: item.version! } }); if (error || !data) throw apiError(error, "Không thể đổi trạng thái."); return data; },
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: ["clients"] }),
  });
  return <>
    <PageHeader eyebrow="Danh mục nội bộ" title="Khách hàng" description="Quản lý đơn vị khách hàng và đi vào từng workspace được cô lập dữ liệu." action={canOperate ? <Button onClick={() => setShowCreate((value) => !value)}><Plus className="mr-2 size-4" />Thêm khách hàng</Button> : undefined} />
    {canOperate && showCreate ? <Card className="mb-6 p-6"><form className="grid gap-4 md:grid-cols-2" onSubmit={form.handleSubmit((v)=>create.mutate(v))}><Field label="Tên công ty"><input className={inputClass} {...form.register("companyName")} /></Field><Field label="Người liên hệ"><input className={inputClass} {...form.register("contactName")} /></Field><Field label="Email"><input className={inputClass} type="email" {...form.register("contactEmail")} /></Field><Field label="Điện thoại"><input className={inputClass} {...form.register("phone")} /></Field><Field label="Ngành"><input className={inputClass} {...form.register("industry")} /></Field><Field label="Thị trường"><input className={inputClass} {...form.register("market")} /></Field><div className="md:col-span-2"><Field label="Ghi chú nội bộ"><textarea className={textareaClass} {...form.register("internalNotes")} /></Field></div>{create.error ? <p className="text-sm text-[var(--coral)] md:col-span-2">{create.error.message}</p>:null}<div className="flex gap-3 md:col-span-2"><Button type="submit" disabled={create.isPending}>{create.isPending?"Đang lưu…":"Tạo khách hàng"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={()=>setShowCreate(false)}>Hủy</Button></div></form></Card>:null}
    <div className="mb-5 flex items-center gap-3 rounded-2xl border border-[var(--line)] bg-white px-4"><Search className="size-4 text-[var(--muted)]" /><input aria-label="Tìm khách hàng" className="min-h-11 w-full bg-transparent text-sm outline-none" value={search} onChange={(e)=>setSearch(e.target.value)} placeholder="Tìm theo công ty hoặc người liên hệ" /></div>
    {query.isLoading?<SkeletonRows />:query.error?<StatePanel title="Không thể tải khách hàng" tone="danger">{query.error.message}</StatePanel>:query.data?.items.length===0?<StatePanel title="Chưa có khách hàng">Tạo khách hàng đầu tiên để mở workspace và hồ sơ thương hiệu.</StatePanel>:<div className="grid gap-4">{query.data?.items.map((item)=><Card key={item.id} className="flex flex-col gap-4 p-5 md:flex-row md:items-center"><span className="grid size-12 place-items-center rounded-2xl bg-[#edf0e7]"><Building2 className="size-5 text-[var(--moss)]" /></span><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><Link className="font-serif text-xl font-bold hover:text-[var(--moss)]" href={`/clients/${item.id}`}>{item.companyName}</Link><Badge tone={item.status==="ACTIVE"?"good":"neutral"}>{item.status==="ACTIVE"?"Đang hoạt động":"Đã lưu trữ"}</Badge></div><p className="mt-1 truncate text-sm text-[var(--muted)]">{item.contactName||"Chưa có người liên hệ"} · {item.market||"Chưa xác định thị trường"}</p></div>{isAdmin ? <Button className="bg-transparent px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" disabled={status.isPending} onClick={()=>status.mutate({item,next:item.status==="ACTIVE"?"ARCHIVED":"ACTIVE"})}>{item.status==="ACTIVE"?<Archive className="mr-2 size-4"/>:<RotateCcw className="mr-2 size-4"/>}{item.status==="ACTIVE"?"Lưu trữ":"Kích hoạt"}</Button> : null}</Card>)}</div>}
  </>;
}
