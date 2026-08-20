"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Laptop, ShieldCheck, Smartphone } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Badge, Button, Card, PageHeader, SkeletonRows, StatePanel } from "@/components/ui";
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

  return (
    <>
      <PageHeader eyebrow="Bảo mật" title="Tài khoản của tôi" description="Kiểm tra thông tin vai trò, đổi mật khẩu và thu hồi những phiên đăng nhập không còn sử dụng." action={<Link href="/account/password"><Button><ShieldCheck className="mr-2 size-4" />Đổi mật khẩu</Button></Link>} />
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
