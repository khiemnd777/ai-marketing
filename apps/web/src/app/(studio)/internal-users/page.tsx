"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Pencil, Plus, Save, UserCheck, UserX } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Badge, Button, Card, Field, PageHeader, SkeletonRows, StatePanel, inputClass } from "@/components/ui";
import { useCurrentUser } from "@/components/auth-context";
import { api } from "@/lib/api";
import { apiError, newIdempotencyKey } from "@/lib/problem";

type User = components["schemas"]["InternalUser"];
type Role = User["role"];

const initialCreate = { email: "", displayName: "", role: "OPERATOR" as Role, temporaryPassword: "" };

export default function InternalUsersPage() {
  const currentUser = useCurrentUser();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState(initialCreate);
  const [resetUser, setResetUser] = useState<User | null>(null);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editForm, setEditForm] = useState({ email: "", displayName: "", role: "OPERATOR" as Role });
  const [temporaryPassword, setTemporaryPassword] = useState("");
  const users = useQuery({
    queryKey: ["internal-users"],
    queryFn: async () => {
      const { data, error } = await api.GET("/internal-users", { params: { query: { page: 1, pageSize: 100 } } });
      if (error || !data) throw apiError(error, "Không thể tải người dùng nội bộ.");
      return data.items;
    },
  });
  const refresh = async () => queryClient.invalidateQueries({ queryKey: ["internal-users"] });
  const create = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/internal-users", { params: { header: { "Idempotency-Key": newIdempotencyKey() } }, body: createForm });
      if (error || !data) throw apiError(error, "Không thể tạo người dùng.");
      return data;
    },
    onSuccess: async () => { setCreateForm(initialCreate); setShowCreate(false); await refresh(); },
  });
  const reset = useMutation({
    mutationFn: async () => {
      if (!resetUser) throw new Error("Chưa chọn người dùng.");
      const { data, error } = await api.POST("/internal-users/{userId}/reset-password", { params: { path: { userId: resetUser.id } }, body: { temporaryPassword, version: resetUser.version } });
      if (error || !data) throw apiError(error, "Không thể reset mật khẩu.");
      return data;
    },
    onSuccess: async () => { setResetUser(null); setTemporaryPassword(""); await refresh(); },
  });
  const update = useMutation({
    mutationFn: async () => {
      if (!editingUser) throw new Error("Chưa chọn người dùng.");
      const { data, error } = await api.PUT("/internal-users/{userId}", { params: { path: { userId: editingUser.id } }, body: { ...editForm, version: editingUser.version } });
      if (error || !data) throw apiError(error, "Không thể cập nhật người dùng.");
      return data;
    },
    onSuccess: async (updated) => { setEditingUser(null); await refresh(); if (updated.id === currentUser.id) { router.refresh(); } },
  });
  const status = useMutation({
    mutationFn: async (user: User) => {
      const next = user.status === "ACTIVE" ? "DISABLED" : "ACTIVE";
      const { data, error } = await api.PATCH("/internal-users/{userId}/status", { params: { path: { userId: user.id } }, body: { status: next, version: user.version } });
      if (error || !data) throw apiError(error, "Không thể đổi trạng thái người dùng.");
      return data;
    },
    onSuccess: refresh,
  });

  return (
    <>
      <PageHeader eyebrow="Quản trị" title="Người dùng nội bộ" description="Cấp quyền theo vai trò, reset mật khẩu tạm thời và thu hồi toàn bộ session khi vô hiệu hóa tài khoản." action={<Button onClick={() => setShowCreate((value) => !value)}><Plus className="mr-2 size-4" />Thêm người dùng</Button>} />
      {showCreate ? <Card className="mb-6 p-6"><h2 className="font-serif text-xl font-bold">Tạo tài khoản</h2><div className="mt-5 grid gap-4 md:grid-cols-2"><Field label="Họ tên"><input className={inputClass} value={createForm.displayName} onChange={(event) => setCreateForm({ ...createForm, displayName: event.target.value })} /></Field><Field label="Email"><input className={inputClass} type="email" value={createForm.email} onChange={(event) => setCreateForm({ ...createForm, email: event.target.value })} /></Field><Field label="Vai trò"><select className={inputClass} value={createForm.role} onChange={(event) => setCreateForm({ ...createForm, role: event.target.value as Role })}><option value="OPERATOR">Operator</option><option value="REVIEWER">Reviewer</option><option value="ADMIN">Admin</option></select></Field><Field label="Mật khẩu tạm thời"><input className={inputClass} type="password" autoComplete="new-password" value={createForm.temporaryPassword} onChange={(event) => setCreateForm({ ...createForm, temporaryPassword: event.target.value })} /></Field></div><p className="mt-3 text-xs text-[var(--muted)]">Người dùng phải đổi mật khẩu ở lần đăng nhập đầu tiên.</p>{create.error ? <p role="alert" className="mt-3 text-sm font-semibold text-[var(--coral)]">{create.error.message}</p> : null}<div className="mt-5 flex gap-3"><Button disabled={create.isPending || createForm.displayName.trim().length < 2 || createForm.temporaryPassword.length < 14} onClick={() => create.mutate()}>{create.isPending ? "Đang tạo…" : "Tạo tài khoản"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => setShowCreate(false)}>Hủy</Button></div></Card> : null}
      {editingUser ? <Card className="mb-6 p-6"><h2 className="font-serif text-xl font-bold">Chỉnh sửa {editingUser.displayName}</h2><div className="mt-5 grid gap-4 md:grid-cols-3"><Field label="Họ tên"><input className={inputClass} value={editForm.displayName} onChange={(event)=>setEditForm({...editForm,displayName:event.target.value})}/></Field><Field label="Email"><input className={inputClass} type="email" value={editForm.email} onChange={(event)=>setEditForm({...editForm,email:event.target.value})}/></Field><Field label="Vai trò"><select className={inputClass} value={editForm.role} onChange={(event)=>setEditForm({...editForm,role:event.target.value as Role})}><option value="OPERATOR">Operator</option><option value="REVIEWER">Reviewer</option><option value="ADMIN">Admin</option></select></Field></div><p className="mt-3 text-xs text-[var(--muted)]">Hệ thống không cho hạ quyền quản trị viên đang hoạt động cuối cùng.</p>{update.error?<p role="alert" className="mt-3 text-sm font-semibold text-[var(--coral)]">{update.error.message}</p>:null}<div className="mt-5 flex gap-3"><Button disabled={update.isPending||editForm.displayName.trim().length<2||!editForm.email.includes("@")} onClick={()=>update.mutate()}><Save className="mr-2 size-4"/>{update.isPending?"Đang lưu…":"Lưu thay đổi"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={()=>setEditingUser(null)}>Hủy</Button></div></Card>:null}
      {resetUser ? <Card className="mb-6 border-[#e5c66a] bg-[#fffaf0] p-6"><h2 className="font-serif text-xl font-bold">Reset mật khẩu cho {resetUser.displayName}</h2><p className="mt-2 text-sm text-[var(--muted)]">Tất cả session hiện tại sẽ bị thu hồi. Chỉ chia sẻ mật khẩu tạm thời qua kênh bảo mật.</p><div className="mt-4 max-w-md"><Field label="Mật khẩu tạm thời"><input className={inputClass} type="password" autoComplete="new-password" value={temporaryPassword} onChange={(event) => setTemporaryPassword(event.target.value)} /></Field></div>{reset.error ? <p role="alert" className="mt-3 text-sm font-semibold text-[var(--coral)]">{reset.error.message}</p> : null}<div className="mt-5 flex gap-3"><Button disabled={reset.isPending || temporaryPassword.length < 14} onClick={() => reset.mutate()}>{reset.isPending ? "Đang reset…" : "Reset và thu hồi session"}</Button><Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)]" onClick={() => { setResetUser(null); setTemporaryPassword(""); }}>Hủy</Button></div></Card> : null}
      {users.isLoading ? <SkeletonRows /> : users.error ? <StatePanel title="Không thể tải người dùng" tone="danger">{users.error.message}</StatePanel> : <div className="grid gap-4">{users.data?.map((user) => <Card key={user.id} data-user-email={user.email} className="flex flex-col gap-4 p-5 lg:flex-row lg:items-center"><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h2 className="font-serif text-xl font-bold">{user.displayName}</h2><Badge tone={user.status === "ACTIVE" ? "good" : "neutral"}>{user.status === "ACTIVE" ? "Đang hoạt động" : "Đã vô hiệu hóa"}</Badge><Badge>{user.role}</Badge>{user.requiresPasswordChange ? <Badge tone="warn">Cần đổi mật khẩu</Badge> : null}</div><p className="mt-2 text-sm text-[var(--muted)]">{user.email} · Đăng nhập gần nhất {user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleString("vi-VN") : "chưa có"}</p></div><div className="flex flex-wrap gap-2"><Button className="bg-transparent px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={()=>{setEditingUser(user);setEditForm({email:user.email,displayName:user.displayName,role:user.role})}}><Pencil className="mr-2 size-4"/>Chỉnh sửa</Button><Button className="bg-transparent px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" onClick={() => { setResetUser(user); setTemporaryPassword(""); }}><KeyRound className="mr-2 size-4" />Reset mật khẩu</Button><Button className="bg-transparent px-4 text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" disabled={status.isPending || user.id === currentUser.id} onClick={() => status.mutate(user)}>{user.status === "ACTIVE" ? <UserX className="mr-2 size-4" /> : <UserCheck className="mr-2 size-4" />}{user.status === "ACTIVE" ? "Vô hiệu hóa" : "Kích hoạt"}</Button></div></Card>)}</div>}
      {status.error ? <p role="alert" className="mt-4 text-sm font-semibold text-[var(--coral)]">{status.error.message}</p> : null}
    </>
  );
}
