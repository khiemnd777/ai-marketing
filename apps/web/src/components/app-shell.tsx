"use client";

import {
  Activity,
  Boxes,
  BriefcaseBusiness,
  ChartNoAxesCombined,
  ChevronRight,
  Clapperboard,
  DatabaseZap,
  FolderKanban,
  LibraryBig,
  Megaphone,
  PackageSearch,
  UsersRound,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ComponentType, ReactNode } from "react";
import { cn } from "@/lib/cn";

type NavItem = { href: string; label: string; icon: ComponentType<{ className?: string }> };

const primary: NavItem[] = [
  { href: "/clients", label: "Khách hàng", icon: BriefcaseBusiness },
  { href: "/products", label: "Sản phẩm", icon: PackageSearch },
  { href: "/media", label: "Thư viện media", icon: LibraryBig },
  { href: "/campaigns", label: "Chiến dịch", icon: FolderKanban },
  { href: "/analytics", label: "Phân tích", icon: ChartNoAxesCombined },
];

const secondary: NavItem[] = [
  { href: "/operations", label: "Vận hành", icon: Activity },
  { href: "/settings/characters", label: "Nhân vật", icon: UsersRound },
  { href: "/settings/providers", label: "Nhà cung cấp", icon: DatabaseZap },
  { href: "/settings/meta", label: "Kết nối Meta", icon: Megaphone },
];

function NavLink({ item }: { item: NavItem }) {
  const pathname = usePathname();
  const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      className={cn(
        "group flex min-h-11 items-center gap-3 rounded-2xl px-3 text-sm font-semibold transition",
        active ? "bg-[var(--lime)] text-[var(--ink)]" : "text-[#687168] hover:bg-white/70 hover:text-[var(--ink)]",
      )}
      aria-current={active ? "page" : undefined}
    >
      <Icon className="size-4.5" />
      <span className="flex-1">{item.label}</span>
      {active ? <ChevronRight className="size-4" /> : null}
    </Link>
  );
}

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[272px_minmax(0,1fr)]">
      <aside className="border-b border-[var(--line)] bg-[#edf0e7] px-4 py-5 lg:sticky lg:top-0 lg:h-screen lg:border-b-0 lg:border-r">
        <Link href="/clients" className="mb-7 flex items-center gap-3 px-2">
          <span className="grid size-11 place-items-center rounded-2xl bg-[var(--ink)] text-[var(--lime)]">
            <Clapperboard className="size-5" />
          </span>
          <span>
            <span className="block text-xs font-bold uppercase tracking-[0.18em] text-[var(--moss)]">AI Studio</span>
            <span className="block font-serif text-lg font-bold text-[var(--ink)]">Marketing Ops</span>
          </span>
        </Link>
        <nav aria-label="Điều hướng chính" className="grid gap-1">
          {primary.map((item) => <NavLink key={item.href} item={item} />)}
        </nav>
        <div className="my-5 border-t border-[var(--line)]" />
        <nav aria-label="Cấu hình và vận hành" className="grid gap-1">
          {secondary.map((item) => <NavLink key={item.href} item={item} />)}
        </nav>
        <div className="mt-6 rounded-3xl bg-[var(--ink)] p-4 text-white">
          <div className="mb-3 flex items-center gap-2 text-sm font-bold"><Boxes className="size-4 text-[var(--lime)]" /> Chế độ demo</div>
          <p className="text-xs leading-5 text-white/65">Các tác vụ trả phí bị khóa. Mọi kết quả mô phỏng đều được gắn nhãn và lưu dấu vết.</p>
        </div>
      </aside>
      <main className="min-w-0 px-5 py-6 md:px-8 lg:px-10 lg:py-9">{children}</main>
    </div>
  );
}
