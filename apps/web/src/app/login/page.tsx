import type { Metadata } from "next";
import { AuthEntry } from "./login-form";

export const metadata: Metadata = { title: "Truy cập nội bộ" };

function safeReturnUrl(value: string | string[] | undefined) {
  const candidate = Array.isArray(value) ? value[0] : value;
  return candidate && candidate.startsWith("/") && !candidate.startsWith("//") ? candidate : "/clients";
}

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const query = await searchParams;
  return (
    <main className="grid min-h-screen place-items-center px-5 py-10">
      <section className="grid w-full max-w-5xl overflow-hidden rounded-[2.25rem] border border-[var(--line)] bg-[var(--panel)] shadow-[0_30px_100px_rgba(21,43,31,0.12)] md:grid-cols-[1.1fr_0.9fr]">
        <div className="relative overflow-hidden bg-[var(--ink)] p-8 text-white md:p-12">
          <div className="absolute -right-20 -top-20 size-72 rounded-full border-[42px] border-[var(--lime)] opacity-80" />
          <p className="relative text-xs font-extrabold uppercase tracking-[0.22em] text-[var(--lime)]">Managed internal platform</p>
          <h1 className="relative mt-5 max-w-lg font-serif text-4xl font-black leading-[1.05] md:text-6xl">Từ dữ liệu đúng đến nội dung có thể kiểm chứng.</h1>
          <p className="relative mt-6 max-w-md text-sm leading-7 text-white/68">Điều hành Product Truth, nội dung AI, video hội thoại, xuất bản Meta và hiệu suất chiến dịch trong một luồng phê duyệt.</p>
          <div className="relative mt-12 grid grid-cols-3 gap-3 text-center text-xs font-bold text-white/70">
            <span className="rounded-2xl border border-white/15 p-3">Product Truth</span>
            <span className="rounded-2xl border border-white/15 p-3">Human review</span>
            <span className="rounded-2xl border border-white/15 p-3">Spend guardrails</span>
          </div>
        </div>
        <div className="p-8 md:p-12">
          <AuthEntry returnUrl={safeReturnUrl(query.returnUrl)} />
        </div>
      </section>
    </main>
  );
}
