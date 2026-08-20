"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Laptop, Pencil, Save, ShieldCheck, Smartphone } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass } from "@/components/ui";
import { useCurrentUser } from "@/components/auth-context";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

type Session = components["schemas"]["UserSession"];

function deviceLabel(userAgent: string) {
  if (/iphone|android|mobile/i.test(userAgent)) return "Thiết bị di động";
  if (/macintosh|windows|linux/i.test(userAgent)) return "Máy tính";
  return "Thiết bị không xác định";
}

export default function AccountPage() {
  const user = useCurrentUser();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [profile, setProfile] = useState({ displayName: user.displayName, email: user.email });
  const sessions = useQuery({
    queryKey: ["auth-sessions"],
    queryFn: async () => {
      const { data, error } = await api.GET("/auth/sessions");
      if (error || !data) throw apiError(error, "Không thể tải phiên đăng nhập.");
      return data.items;
    },
  });
  const revoke = useMutation({
    mutationFn: async (session: Session) => {
      const { error } = await api.DELETE("/auth/sessions/{sessionId}", { params: { path: { sessionId: session.id } } });
      if (error) throw apiError(error, "Không thể thu hồi phiên.");
      return session;
    },
    onSuccess: async (session) => {
      if (session.current) {
        queryClient.clear();
        router.replace("/login");
        router.refresh();
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["auth-sessions"] });
    },
  });
  const updateProfile = useMutation({
    mutationFn: async () => { const { data, error } = await api.PUT("/auth/me", { body: { ...profile, version: user.version } }); if (error || !data) throw apiError(error, "Không thể cập nhật tài khoản."); return data; },
    onSuccess: () => { setEditing(false); router.refresh(); },
  });

  return (
    <>
      <PageHeader eyebrow="Bảo mật" title="Tài khoản của tôi" description="Cập nhật thông tin cá nhân, đổi mật khẩu và thu hồi những phiên đăng nhập không còn sử dụng." action={<div className="flex flex-wrap gap-2"><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={()=>setEditing(value=>!value)}><Pencil className="mr-2 size-4"/>{editing?"Đóng chỉnh sửa":"Chỉnh sửa"}</Button><Button onClick={()=>router.push("/account/password?returnUrl=%2Faccount")}><ShieldCheck className="mr-2 size-4" />Đổi mật khẩu</Button></div>} />
      {editing?<Card className="mb-7 p-6"><h2 className="font-serif text-xl font-bold">Chỉnh sửa hồ sơ</h2><div className="mt-5 grid gap-4 md:grid-cols-2"><Field label="Họ tên"><input className={inputClass} value={profile.displayName} onChange={event=>setProfile({...profile,displayName:event.target.value})}/></Field><Field label="Email"><input className={inputClass} type="email" value={profile.email} onChange={event=>setProfile({...profile,email:event.target.value})}/></Field></div>{updateProfile.error?<p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{updateProfile.error.message}</p>:null}<div className="mt-5 flex gap-3"><Button disabled={updateProfile.isPending||profile.displayName.trim().length<2||!profile.email.includes("@")} onClick={()=>updateProfile.mutate()}><Save className="mr-2 size-4"/>{updateProfile.isPending?"Đang lưu…":"Lưu thay đổi"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={()=>{setEditing(false);setProfile({displayName:user.displayName,email:user.email})}}>Hủy</Button></div></Card>:null}
      <Card className="mb-7 grid gap-5 p-6 md:grid-cols-3">
        <div><p className="text-xs font-bold uppercase tracking-wider text-[var(--muted)]">Họ tên</p><p className="mt-2 font-serif text-xl font-bold">{user.displayName}</p></div>
        <div><p className="text-xs font-bold uppercase tracking-wider text-[var(--muted)]">Email</p><p className="mt-2 text-sm font-semibold">{user.email}</p></div>
        <div><p className="text-xs font-bold uppercase tracking-wider text-[var(--muted)]">Vai trò</p><div className="mt-2"><Badge tone="good">{user.role}</Badge></div></div>
      </Card>
      <section aria-labelledby="sessions-heading">
        <div className="mb-4"><h2 id="sessions-heading" className="font-serif text-2xl font-bold">Phiên đang hoạt động</h2><p className="mt-2 text-sm text-[var(--muted)]">Thu hồi ngay thiết bị bạn không nhận ra. Thu hồi phiên hiện tại sẽ đăng xuất thiết bị này.</p></div>
        {sessions.isLoading ? <SkeletonRows /> : sessions.error ? <StatePanel title="Không thể tải phiên" tone="danger">{sessions.error.message}</StatePanel> : sessions.data?.length === 0 ? <StatePanel title="Không có phiên hoạt động">Đăng nhập lại để tạo phiên mới.</StatePanel> : <div className="grid gap-4">{sessions.data?.map((session) => {
          const MobileIcon = /iphone|android|mobile/i.test(session.userAgent) ? Smartphone : Laptop;
          return <Card key={session.id} className="flex flex-col gap-4 p-5 md:flex-row md:items-center"><span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-[#edf0e7]"><MobileIcon className="size-5 text-[var(--moss)]" /></span><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h3 className="font-bold">{deviceLabel(session.userAgent)}</h3>{session.current ? <Badge tone="good">Phiên hiện tại</Badge> : null}</div><p className="mt-1 truncate text-xs text-[var(--muted)]">{session.userAgent || "Không có user agent"}</p><p className="mt-2 text-xs text-[var(--muted)]">IP {session.ipAddress || "không xác định"} · Hoạt động {new Date(session.lastSeenAt).toLocaleString("vi-VN")}</p></div><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" disabled={revoke.isPending} onClick={() => revoke.mutate(session)}>{session.current ? "Đăng xuất phiên này" : "Thu hồi"}</Button></Card>;
        })}</div>}
        {revoke.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{revoke.error.message}</p> : null}
      </section>
    </>
  );
}
