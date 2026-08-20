import type { Metadata } from "next";
import { PasswordChangeForm } from "./password-change-form";

export const metadata: Metadata = { title: "Đổi mật khẩu" };

function safeReturnUrl(value: string | string[] | undefined) {
  const candidate = Array.isArray(value) ? value[0] : value;
  return candidate && candidate.startsWith("/") && !candidate.startsWith("//") && !candidate.startsWith("/account/password")
    ? candidate
    : "/clients";
}

export default async function PasswordPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const query = await searchParams;
  return (
    <main className="grid min-h-screen place-items-center px-5 py-10">
      <section className="w-full max-w-xl rounded-[2rem] border border-[var(--line)] bg-[var(--panel)] p-7 shadow-[0_30px_100px_rgba(21,43,31,0.12)] md:p-10">
        <p className="text-xs font-extrabold uppercase tracking-[0.2em] text-[var(--moss)]">Bảo mật tài khoản</p>
        <h1 className="mt-3 font-serif text-3xl font-black">Đổi mật khẩu</h1>
        <p className="mt-3 text-sm leading-6 text-[var(--muted)]">Mật khẩu mới phải khác mật khẩu hiện tại và có ít nhất 14 ký tự. Các phiên đăng nhập khác sẽ bị thu hồi.</p>
        <PasswordChangeForm returnUrl={safeReturnUrl(query.returnUrl)} />
      </section>
    </main>
  );
}
