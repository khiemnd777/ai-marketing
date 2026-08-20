import { forwardRef, type ButtonHTMLAttributes, type HTMLAttributes, type ReactNode } from "react";
import { cn } from "@/lib/cn";

export const Button = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement>>(function Button({ className, type = "button", ...props }, ref) {
  return (
    <button
      ref={ref}
      type={type}
      className={cn(
        "inline-flex min-h-10 items-center justify-center rounded-full bg-[var(--ink)] px-5 text-sm font-semibold text-white transition hover:bg-[var(--moss)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--moss)] disabled:cursor-not-allowed disabled:opacity-45",
        className,
      )}
      {...props}
    />
  );
});

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-3xl border border-[var(--line)] bg-[var(--panel)] shadow-[0_18px_60px_rgba(29,48,37,0.06)]", className)} {...props} />;
}

export function Badge({ children, tone = "neutral" }: { children: ReactNode; tone?: "neutral" | "good" | "warn" | "danger" }) {
  const tones = {
    neutral: "bg-[#edf0e9] text-[#566058]",
    good: "bg-[#dcefdc] text-[#28643c]",
    warn: "bg-[#fff0c9] text-[#79580b]",
    danger: "bg-[#ffe0d9] text-[#8b3425]",
  };
  return <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-bold", tones[tone])}>{children}</span>;
}

export function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label className="grid gap-2 text-sm font-semibold text-[var(--ink)]">
      {label}
      {children}
      {error ? <span className="text-xs font-medium text-[var(--coral)]">{error}</span> : null}
    </label>
  );
}

export const inputClass =
  "min-h-11 w-full rounded-2xl border border-[var(--line)] bg-white px-4 text-sm text-[var(--ink)] outline-none transition placeholder:text-[#98a099] focus:border-[var(--moss)] focus:ring-4 focus:ring-[#26664f18]";

export function PageHeader({ eyebrow, title, description, action, headingLevel = "h1" }: { eyebrow: string; title: string; description: string; action?: ReactNode; headingLevel?: "h1" | "h2" }) {
  const Heading = headingLevel;
  return <header className="mb-7 flex flex-col gap-5 md:flex-row md:items-end md:justify-between"><div className="max-w-3xl"><p className="mb-2 text-xs font-bold uppercase tracking-[0.18em] text-[var(--moss)]">{eyebrow}</p><Heading className="font-serif text-3xl font-bold tracking-tight md:text-4xl">{title}</Heading><p className="mt-3 max-w-2xl text-sm leading-6 text-[var(--muted)]">{description}</p></div>{action}</header>;
}

export function StatePanel({ title, children, tone = "neutral" }: { title: string; children: ReactNode; tone?: "neutral" | "danger" }) {
  return <Card className={cn("p-6", tone === "danger" && "border-[#efb6aa] bg-[#fff4ef]")}><h2 className="font-serif text-xl font-bold">{title}</h2><div className="mt-2 text-sm leading-6 text-[var(--muted)]">{children}</div></Card>;
}

export function SkeletonRows() {
  return <div aria-label="Đang tải" className="grid gap-3">{[0,1,2].map((item)=><div key={item} className="h-20 animate-pulse rounded-2xl bg-[#e7e8df]" />)}</div>;
}

export const textareaClass = `${inputClass} min-h-28 py-3`;
