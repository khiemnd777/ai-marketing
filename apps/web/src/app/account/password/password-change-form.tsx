"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button, Field, inputClass } from "@/components/ui";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

const schema = z.object({
  currentPassword: z.string().min(10, "Nhập mật khẩu hiện tại").max(200),
  newPassword: z.string().min(14, "Mật khẩu mới phải có ít nhất 14 ký tự").max(200),
  confirmation: z.string(),
}).refine((value) => value.newPassword === value.confirmation, { path: ["confirmation"], message: "Mật khẩu xác nhận chưa khớp" })
  .refine((value) => value.currentPassword !== value.newPassword, { path: ["newPassword"], message: "Mật khẩu mới phải khác mật khẩu hiện tại" });

type Values = z.infer<typeof schema>;

export function PasswordChangeForm({ returnUrl }: { returnUrl: string }) {
  const router = useRouter();
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: { currentPassword: "", newPassword: "", confirmation: "" } });
  const change = useMutation({
    mutationFn: async (values: Values) => {
      const { error } = await api.POST("/auth/change-password", { body: { currentPassword: values.currentPassword, newPassword: values.newPassword } });
      if (error) throw apiError(error, "Không thể đổi mật khẩu.");
    },
    onSuccess: () => {
      router.replace(returnUrl);
      router.refresh();
    },
  });
  const logout = useMutation({
    mutationFn: async () => {
      const { error } = await api.POST("/auth/logout");
      if (error) throw apiError(error, "Không thể đăng xuất.");
    },
    onSettled: () => router.replace("/login"),
  });

  return (
    <form className="mt-7 grid gap-5" onSubmit={form.handleSubmit((values) => change.mutate(values))} noValidate>
      <Field label="Mật khẩu hiện tại" error={form.formState.errors.currentPassword?.message}>
        <input className={inputClass} type="password" autoComplete="current-password" {...form.register("currentPassword")} />
      </Field>
      <Field label="Mật khẩu mới" error={form.formState.errors.newPassword?.message}>
        <input className={inputClass} type="password" autoComplete="new-password" {...form.register("newPassword")} />
      </Field>
      <Field label="Xác nhận mật khẩu mới" error={form.formState.errors.confirmation?.message}>
        <input className={inputClass} type="password" autoComplete="new-password" {...form.register("confirmation")} />
      </Field>
      {change.error ? <p role="alert" className="rounded-2xl bg-[#ffe5de] px-4 py-3 text-sm font-semibold text-[#853a2a]">{change.error.message}</p> : null}
      <Button type="submit" disabled={change.isPending}>{change.isPending ? "Đang cập nhật…" : "Đổi mật khẩu và tiếp tục"}</Button>
      <Button className="bg-transparent text-[var(--ink)] ring-1 ring-[var(--line)] hover:bg-white" disabled={logout.isPending} onClick={() => logout.mutate()}>Đăng xuất</Button>
    </form>
  );
}
