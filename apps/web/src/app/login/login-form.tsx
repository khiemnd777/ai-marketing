"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button, Field, inputClass } from "@/components/ui";
import { api } from "@/lib/api";

const schema = z.object({
  email: z.email("Email không hợp lệ").max(320),
  password: z.string().min(10, "Mật khẩu phải có ít nhất 10 ký tự").max(200),
});
type LoginValues = z.infer<typeof schema>;

export function LoginForm() {
  const router = useRouter();
  const form = useForm<LoginValues>({ resolver: zodResolver(schema), defaultValues: { email: "", password: "" } });
  const login = useMutation({
    mutationFn: async (values: LoginValues) => {
      const { data, error } = await api.POST("/auth/login", { body: values });
      if (error || !data) throw new Error(error?.detail ?? "Không thể đăng nhập. Vui lòng thử lại.");
      return data;
    },
    onSuccess: () => {
      router.replace("/clients");
      router.refresh();
    },
  });

  return (
    <form className="mt-8 grid gap-5" onSubmit={form.handleSubmit((values) => login.mutate(values))} noValidate>
      <Field label="Email" error={form.formState.errors.email?.message}>
        <input className={inputClass} type="email" autoComplete="username" placeholder="operator@company.com" {...form.register("email")} />
      </Field>
      <Field label="Mật khẩu" error={form.formState.errors.password?.message}>
        <input className={inputClass} type="password" autoComplete="current-password" {...form.register("password")} />
      </Field>
      {login.error ? <p role="alert" className="rounded-2xl bg-[#ffe5de] px-4 py-3 text-sm font-semibold text-[#853a2a]">{login.error.message}</p> : null}
      <Button className="mt-1 w-full" type="submit" disabled={login.isPending}>
        {login.isPending ? "Đang xác thực…" : "Đăng nhập"}
      </Button>
    </form>
  );
}
