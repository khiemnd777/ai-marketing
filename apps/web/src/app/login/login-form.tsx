"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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

const bootstrapSchema = z.object({
  displayName: z.string().trim().min(2, "Họ tên phải có ít nhất 2 ký tự").max(120),
  email: z.email("Email không hợp lệ").max(320),
  password: z.string().min(14, "Mật khẩu phải có ít nhất 14 ký tự").max(200),
  confirmPassword: z.string(),
}).refine((value) => value.password === value.confirmPassword, {
  path: ["confirmPassword"],
  message: "Xác nhận mật khẩu chưa khớp",
});
type BootstrapValues = z.infer<typeof bootstrapSchema>;

function AuthHeading({ bootstrap }: { bootstrap: boolean }) {
  return (
    <>
      <p className="text-xs font-extrabold uppercase tracking-[0.2em] text-[var(--moss)]">AI Product Marketing Studio</p>
      <h2 className="mt-3 font-serif text-3xl font-black">{bootstrap ? "Khởi tạo quản trị viên" : "Đăng nhập nội bộ"}</h2>
      <p className="mt-2 text-sm leading-6 text-[var(--muted)]">
        {bootstrap
          ? "Chỉ tạo tài khoản ADMIN đầu tiên cho hệ thống này. Đây không phải đăng ký công khai."
          : "Tài khoản do quản trị viên cấp. Không có đăng ký công khai."}
      </p>
    </>
  );
}

export function AuthEntry({ returnUrl = "/clients" }: { returnUrl?: string }) {
  const status = useQuery({
    queryKey: ["admin-bootstrap-status"],
    queryFn: async () => {
      const { data, error } = await api.GET("/auth/bootstrap/status");
      if (error || !data) throw new Error(error?.detail ?? "Không thể kiểm tra trạng thái quản trị viên.");
      return data;
    },
  });

  if (status.isPending) {
    return (
      <div aria-live="polite">
        <AuthHeading bootstrap={false} />
        <div role="status" className="mt-8 h-44 animate-pulse rounded-3xl bg-[#edf0e9] p-5 text-sm font-semibold text-[var(--muted)]">Đang kiểm tra trạng thái hệ thống…</div>
      </div>
    );
  }
  if (status.error) {
    return (
      <div>
        <AuthHeading bootstrap={false} />
        <div role="alert" className="mt-8 rounded-2xl bg-[#ffe5de] px-4 py-4 text-sm font-semibold text-[#853a2a]">
          <p>{status.error.message}</p>
          <Button className="mt-4 bg-transparent text-[#853a2a] ring-1 ring-[#853a2a] hover:bg-[#fff4ef]" onClick={() => void status.refetch()}>Thử kiểm tra lại</Button>
        </div>
      </div>
    );
  }
  if (status.data.required) return <BootstrapAdminForm returnUrl={returnUrl} />;
  return <><AuthHeading bootstrap={false} /><LoginForm returnUrl={returnUrl} /></>;
}

export function LoginForm({ returnUrl = "/clients" }: { returnUrl?: string }) {
  const router = useRouter();
  const form = useForm<LoginValues>({ resolver: zodResolver(schema), defaultValues: { email: "", password: "" } });
  const login = useMutation({
    mutationFn: async (values: LoginValues) => {
      const { data, error } = await api.POST("/auth/login", { body: values });
      if (error || !data) throw new Error(error?.detail ?? "Không thể đăng nhập. Vui lòng thử lại.");
      return data;
    },
    onSuccess: (user) => {
      const destination = user.requiresPasswordChange
        ? `/account/password?returnUrl=${encodeURIComponent(returnUrl)}`
        : returnUrl;
      router.replace(destination);
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

function BootstrapAdminForm({ returnUrl }: { returnUrl: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const form = useForm<BootstrapValues>({
    resolver: zodResolver(bootstrapSchema),
    defaultValues: { displayName: "", email: "", password: "", confirmPassword: "" },
  });
  const bootstrap = useMutation({
    mutationFn: async (values: BootstrapValues) => {
      const { data, error } = await api.POST("/auth/bootstrap", {
        body: { displayName: values.displayName, email: values.email, password: values.password },
      });
      if (error || !data) throw new Error(error?.detail ?? "Không thể tạo quản trị viên. Vui lòng thử lại.");
      return data;
    },
    onSuccess: () => {
      queryClient.setQueryData(["admin-bootstrap-status"], { required: false });
      router.replace(returnUrl);
      router.refresh();
    },
    onError: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin-bootstrap-status"] });
    },
  });

  return (
    <>
      <AuthHeading bootstrap />
      <form className="mt-8 grid gap-5" onSubmit={form.handleSubmit((values) => bootstrap.mutate(values))} noValidate>
        <Field label="Họ tên" error={form.formState.errors.displayName?.message}>
          <input className={inputClass} autoComplete="name" placeholder="Quản trị viên hệ thống" {...form.register("displayName")} />
        </Field>
        <Field label="Email" error={form.formState.errors.email?.message}>
          <input className={inputClass} type="email" autoComplete="username" placeholder="admin@company.com" {...form.register("email")} />
        </Field>
        <Field label="Mật khẩu" error={form.formState.errors.password?.message}>
          <input className={inputClass} type="password" autoComplete="new-password" {...form.register("password")} />
        </Field>
        <Field label="Xác nhận mật khẩu" error={form.formState.errors.confirmPassword?.message}>
          <input className={inputClass} type="password" autoComplete="new-password" {...form.register("confirmPassword")} />
        </Field>
        {bootstrap.error ? <p role="alert" className="rounded-2xl bg-[#ffe5de] px-4 py-3 text-sm font-semibold text-[#853a2a]">{bootstrap.error.message}</p> : null}
        <Button className="mt-1 w-full" type="submit" disabled={bootstrap.isPending}>
          {bootstrap.isPending ? "Đang khởi tạo…" : "Tạo quản trị viên và tiếp tục"}
        </Button>
      </form>
    </>
  );
}
