"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Boxes, Plus } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { usePermissions } from "@/components/auth-context";
import { Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

export default function ClientDetailPage() {
  const { canOperate } = usePermissions();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState(""); const [slug, setSlug] = useState(""); const [timezone, setTimezone] = useState("Asia/Ho_Chi_Minh");
  const client = useQuery({ queryKey: ["client", id], queryFn: async()=>{const{data,error}=await api.GET("/clients/{clientId}",{params:{path:{clientId:id}}});if(error||!data)throw apiError(error,"Không thể tải khách hàng.");return data;} });
  const workspaces = useQuery({ queryKey: ["workspaces", id], queryFn: async()=>{const{data,error}=await api.GET("/clients/{clientId}/workspaces",{params:{path:{clientId:id}}});if(error||!data)throw apiError(error,"Không thể tải workspace.");return data;} });
  const create = useMutation({ mutationFn: async()=>{const{data,error}=await api.POST("/clients/{clientId}/workspaces",{params:{path:{clientId:id},header:{"Idempotency-Key":newIdempotencyKey()}},body:{name,slug,timezone}});if(error||!data)throw apiError(error,"Không thể tạo workspace.");return data;},onSuccess:async()=>{setShowCreate(false);setName("");setSlug("");await queryClient.invalidateQueries({queryKey:["workspaces",id]});} });
  if(client.isLoading)return <SkeletonRows/>;if(client.error)return <StatePanel title="Không thể tải khách hàng" tone="danger">{client.error.message}</StatePanel>;
  return <><Link href="/clients" className="mb-5 inline-flex items-center gap-2 text-sm font-bold text-[var(--moss)]"><ArrowLeft className="size-4"/>Tất cả khách hàng</Link><PageHeader eyebrow="Khách hàng" title={client.data!.companyName} description={`${client.data!.industry||"Chưa xác định ngành"} · ${client.data!.market||"Chưa xác định thị trường"}`} action={canOperate ? <Button onClick={()=>setShowCreate(v=>!v)}><Plus className="mr-2 size-4"/>Thêm workspace</Button> : undefined}/>
  {canOperate && showCreate?<Card className="mb-6 p-6"><div className="grid gap-4 md:grid-cols-3"><Field label="Tên workspace"><input className={inputClass} value={name} onChange={e=>setName(e.target.value)} /></Field><Field label="Slug"><input className={inputClass} value={slug} onChange={e=>setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,""))}/></Field><Field label="Múi giờ"><input className={inputClass} value={timezone} onChange={e=>setTimezone(e.target.value)}/></Field></div>{create.error?<p className="mt-3 text-sm text-[var(--coral)]">{create.error.message}</p>:null}<div className="mt-4 flex gap-3"><Button disabled={create.isPending||name.length<2||!slug} onClick={()=>create.mutate()}>{create.isPending?"Đang tạo…":"Tạo workspace"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={()=>setShowCreate(false)}>Hủy</Button></div></Card>:null}
  {workspaces.isLoading?<SkeletonRows/>:workspaces.error?<StatePanel title="Không thể tải workspace" tone="danger">{workspaces.error.message}</StatePanel>:workspaces.data?.items.length===0?<StatePanel title="Chưa có workspace">Mỗi workspace đại diện cho một thương hiệu hoặc đơn vị kinh doanh và là ranh giới cô lập dữ liệu.</StatePanel>:<div className="grid gap-4 md:grid-cols-2">{workspaces.data?.items.map(item=><Link key={item.id} href={`/workspaces/${item.id}?clientId=${id}`}><Card className="h-full p-5 transition hover:-translate-y-0.5 hover:border-[var(--moss)]"><span className="mb-4 grid size-11 place-items-center rounded-2xl bg-[#edf0e7]"><Boxes className="size-5 text-[var(--moss)]"/></span><h2 className="font-serif text-xl font-bold">{item.name}</h2><p className="mt-2 text-sm text-[var(--muted)]">{item.slug} · {item.timezone}</p></Card></Link>)}</div>}</>;
}
