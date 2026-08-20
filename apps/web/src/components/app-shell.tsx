"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
  LogOut,
  Menu,
  Megaphone,
  PackageSearch,
  ShieldCheck,
  UserCog,
  UsersRound,
  X,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { CurrentUserProvider, type CurrentUser } from "@/components/auth-context";
import { cn } from "@/lib/cn";
import { api } from "@/lib/api";
import { apiError } from "@/lib/problem";

type Client = components["schemas"]["Client"];
type Workspace = components["schemas"]["Workspace"];
type NavItem = {
  href: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  workspaceScoped?: boolean;
  adminOnly?: boolean;
};

const primary: NavItem[] = [
  { href: "/clients", label: "Khách hàng", icon: BriefcaseBusiness },
  { href: "/products", label: "Sản phẩm", icon: PackageSearch, workspaceScoped: true },
  { href: "/media", label: "Thư viện media", icon: LibraryBig, workspaceScoped: true },
  { href: "/campaigns", label: "Chiến dịch", icon: FolderKanban, workspaceScoped: true },
  { href: "/analytics", label: "Phân tích", icon: ChartNoAxesCombined, workspaceScoped: true },
];

const secondary: NavItem[] = [
  { href: "/operations", label: "Vận hành", icon: Activity, adminOnly: true },
  { href: "/settings/characters", label: "Nhân vật", icon: UsersRound, workspaceScoped: true },
  { href: "/settings/providers", label: "Nhà cung cấp", icon: DatabaseZap, adminOnly: true },
  { href: "/settings/meta", label: "Kết nối Meta", icon: Megaphone, workspaceScoped: true },
  { href: "/internal-users", label: "Người dùng", icon: UserCog, adminOnly: true },
  { href: "/account", label: "Tài khoản", icon: ShieldCheck },
];

function hrefWithScope(item: NavItem, clientId: string, workspaceId: string) {
  if (!item.workspaceScoped) return item.href;
  const query = new URLSearchParams();
  if (clientId) query.set("clientId", clientId);
  if (workspaceId) query.set("workspaceId", workspaceId);
  const serialized = query.toString();
  return serialized ? `${item.href}?${serialized}` : item.href;
}

function NavLink({ item, clientId, workspaceId, onNavigate }: { item: NavItem; clientId: string; workspaceId: string; onNavigate?: () => void }) {
  const pathname = usePathname();
  const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
  const Icon = item.icon;
  return (
    <Link
      href={hrefWithScope(item, clientId, workspaceId)}
      onClick={onNavigate}
      className={cn(
        "group flex min-h-11 items-center gap-3 rounded-2xl px-3 text-sm font-semibold transition",
        active ? "bg-[var(--lime)] text-[var(--ink)]" : "text-[#5e685f] hover:bg-white/70 hover:text-[var(--ink)]",
      )}
      aria-current={active ? "page" : undefined}
    >
      <Icon className="size-4.5" />
      <span className="flex-1">{item.label}</span>
      {active ? <ChevronRight className="size-4" /> : null}
    </Link>
  );
}

function ScopeSwitcher({ clientId, workspaceId }: { clientId: string; workspaceId: string }) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const clients = useQuery({
    queryKey: ["scope-clients"],
    queryFn: async () => {
      const { data, error } = await api.GET("/clients", { params: { query: { page: 1, pageSize: 100 } } });
      if (error || !data) throw apiError(error, "Không thể tải phạm vi khách hàng.");
      return data.items;
    },
  });
  const workspaces = useQuery({
    queryKey: ["scope-workspaces", clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải phạm vi workspace.");
      return data.items;
    },
  });

  const replaceScope = (nextClientId: string, nextWorkspaceId: string) => {
    const query = new URLSearchParams(searchParams.toString());
    if (nextClientId) query.set("clientId", nextClientId); else query.delete("clientId");
    if (nextWorkspaceId) query.set("workspaceId", nextWorkspaceId); else query.delete("workspaceId");
    const serialized = query.toString();
    router.replace(serialized ? `${pathname}?${serialized}` : pathname);
    if (nextClientId && nextWorkspaceId) {
      localStorage.setItem("studio:last-scope", JSON.stringify({ clientId: nextClientId, workspaceId: nextWorkspaceId }));
    }
  };

  useEffect(() => {
    if (clientId || workspaceId) return;
    try {
      const stored = JSON.parse(localStorage.getItem("studio:last-scope") ?? "null") as { clientId?: string; workspaceId?: string } | null;
      if (stored?.clientId && stored.workspaceId) replaceScope(stored.clientId, stored.workspaceId);
    } catch {
      localStorage.removeItem("studio:last-scope");
    }
    // Scope restoration intentionally runs only when URL scope is absent on entry.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const clientItems = clients.data ?? [];
  const workspaceItems = workspaces.data ?? [];
  return (
    <div className="mb-5 grid gap-2 rounded-2xl border border-[var(--line)] bg-white/70 p-3" aria-label="Phạm vi làm việc">
      <label className="grid gap-1 text-[0.65rem] font-bold uppercase tracking-[0.12em] text-[var(--muted)]">
        Khách hàng
        <select
          className="min-h-10 rounded-xl border border-[var(--line)] bg-white px-3 text-sm font-semibold normal-case tracking-normal text-[var(--ink)]"
          value={clientId}
          onChange={(event) => replaceScope(event.target.value, "")}
        >
          <option value="">Chọn khách hàng</option>
          {clientItems.map((item: Client) => <option key={item.id} value={item.id}>{item.companyName}</option>)}
        </select>
      </label>
      <label className="grid gap-1 text-[0.65rem] font-bold uppercase tracking-[0.12em] text-[var(--muted)]">
        Workspace
        <select
          className="min-h-10 rounded-xl border border-[var(--line)] bg-white px-3 text-sm font-semibold normal-case tracking-normal text-[var(--ink)] disabled:opacity-50"
          value={workspaceId}
          disabled={!clientId || workspaces.isLoading}
          onChange={(event) => replaceScope(clientId, event.target.value)}
        >
          <option value="">{clientId ? "Chọn workspace" : "Chọn khách hàng trước"}</option>
          {workspaceItems.map((item: Workspace) => <option key={item.id} value={item.id}>{item.name}</option>)}
        </select>
      </label>
      {clients.error || workspaces.error ? <p role="alert" className="text-xs font-semibold text-[var(--coral)]">Không thể tải phạm vi làm việc.</p> : null}
    </div>
  );
}

function ShellContent({ user, demoMode, children }: { user: CurrentUser; demoMode: boolean; children: ReactNode }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const navigationRef = useRef<HTMLElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const clientId = searchParams.get("clientId") ?? "";
  const workspaceId = searchParams.get("workspaceId") ?? "";
  const logout = useMutation({
    mutationFn: async () => {
      const { error } = await api.POST("/auth/logout");
      if (error) throw apiError(error, "Không thể đăng xuất.");
    },
    onSuccess: () => {
      queryClient.clear();
      router.replace("/login");
      router.refresh();
    },
  });
  const secondaryItems = secondary.filter((item) => !item.adminOnly || user.role === "ADMIN");

  useEffect(() => {
    if (!navigationOpen) return;
    const previousOverflow = document.body.style.overflow;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const menuButton = menuButtonRef.current;
    document.body.style.overflow = "hidden";
    navigationRef.current?.querySelector<HTMLElement>("[data-drawer-close]")?.focus();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setNavigationOpen(false);
      if (event.key !== "Tab" || !navigationRef.current) return;
      const focusable = [...navigationRef.current.querySelectorAll<HTMLElement>("button:not([disabled]), a[href], select:not([disabled])")];
      const first = focusable[0];
      const last = focusable.at(-1);
      if (!first || !last) return;
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleKeyDown);
      (previousFocus ?? menuButton)?.focus();
    };
  }, [navigationOpen]);

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[288px_minmax(0,1fr)]">
      <a href="#main-content" className="skip-link">Bỏ qua điều hướng</a>
      <header inert={navigationOpen ? true : undefined} aria-hidden={navigationOpen ? true : undefined} className="sticky top-0 z-30 flex min-h-16 items-center justify-between border-b border-[var(--line)] bg-[#edf0e7]/95 px-4 backdrop-blur lg:hidden">
        <Link href="/clients" className="flex items-center gap-3" aria-label="AI Studio — về trang khách hàng">
          <span className="grid size-10 place-items-center rounded-2xl bg-[var(--ink)] text-[var(--lime)]"><Clapperboard className="size-5" /></span>
          <span className="font-serif text-base font-bold text-[var(--ink)]">Marketing Ops</span>
        </Link>
        <button ref={menuButtonRef} type="button" className="grid size-11 place-items-center rounded-2xl border border-[var(--line)] bg-white" aria-label="Mở điều hướng" aria-controls="studio-navigation" aria-expanded={navigationOpen} onClick={() => setNavigationOpen(true)}><Menu className="size-5" /></button>
      </header>
      {navigationOpen ? <button type="button" tabIndex={-1} className="fixed inset-0 z-40 bg-[var(--ink)]/45 backdrop-blur-[1px] lg:hidden" aria-label="Đóng điều hướng" onClick={() => setNavigationOpen(false)} /> : null}
      <aside
        id="studio-navigation"
        ref={navigationRef}
        role={navigationOpen ? "dialog" : undefined}
        aria-modal={navigationOpen ? true : undefined}
        aria-label={navigationOpen ? "Điều hướng ứng dụng" : undefined}
        className={cn(
          "fixed inset-y-0 left-0 z-50 w-[min(88vw,22rem)] flex-col overflow-y-auto border-r border-[var(--line)] bg-[#edf0e7] px-4 py-5 shadow-2xl",
          navigationOpen ? "flex" : "hidden",
          "lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-auto lg:shadow-none",
        )}
      >
        <div className="mb-5 flex items-center gap-2">
          <Link href="/clients" className="flex min-w-0 flex-1 items-center gap-3 px-2" onClick={() => setNavigationOpen(false)}>
          <span className="grid size-11 place-items-center rounded-2xl bg-[var(--ink)] text-[var(--lime)]"><Clapperboard className="size-5" /></span>
          <span><span className="block text-xs font-bold uppercase tracking-[0.18em] text-[var(--moss)]">AI Studio</span><span className="block font-serif text-lg font-bold text-[var(--ink)]">Marketing Ops</span></span>
          </Link>
          <button data-drawer-close type="button" className="grid size-11 shrink-0 place-items-center rounded-2xl hover:bg-white lg:hidden" aria-label="Đóng điều hướng" onClick={() => setNavigationOpen(false)}><X className="size-5" /></button>
        </div>
        <ScopeSwitcher clientId={clientId} workspaceId={workspaceId} />
        <nav aria-label="Điều hướng chính" className="grid gap-1">{primary.map((item) => <NavLink key={item.href} item={item} clientId={clientId} workspaceId={workspaceId} onNavigate={() => setNavigationOpen(false)} />)}</nav>
        <div className="my-4 border-t border-[var(--line)]" />
        <nav aria-label="Cấu hình và vận hành" className="grid gap-1">{secondaryItems.map((item) => <NavLink key={item.href} item={item} clientId={clientId} workspaceId={workspaceId} onNavigate={() => setNavigationOpen(false)} />)}</nav>
        <div className="mt-5 rounded-3xl bg-[var(--ink)] p-4 text-white">
          <div className="mb-2 flex items-center gap-2 text-sm font-bold"><Boxes className="size-4 text-[var(--lime)]" />{demoMode ? "Chế độ demo" : "Môi trường live"}</div>
          <p className="text-xs leading-5 text-white/65">{demoMode ? "Tác vụ trả phí bị khóa; kết quả mô phỏng được gắn nhãn và lưu dấu vết." : "Tác vụ provider có thể phát sinh chi phí và luôn tuân theo approval guardrails."}</p>
        </div>
        <div className="mt-4 flex items-center gap-3 rounded-3xl border border-[var(--line)] bg-white/70 p-3 lg:mt-auto">
          <span className="grid size-10 shrink-0 place-items-center rounded-full bg-[var(--lime)] text-sm font-black text-[var(--ink)]">{user.displayName.slice(0, 1).toUpperCase()}</span>
          <Link href="/account" className="min-w-0 flex-1"><span className="block truncate text-sm font-bold">{user.displayName}</span><span className="block text-xs text-[var(--muted)]">{user.role}</span></Link>
          <button className="grid size-10 place-items-center rounded-full text-[var(--muted)] hover:bg-white hover:text-[var(--ink)]" aria-label="Đăng xuất" disabled={logout.isPending} onClick={() => logout.mutate()}><LogOut className="size-4" /></button>
        </div>
        {logout.error ? <p role="alert" className="mt-2 text-xs font-semibold text-[var(--coral)]">{logout.error.message}</p> : null}
      </aside>
      <main id="main-content" tabIndex={-1} inert={navigationOpen ? true : undefined} aria-hidden={navigationOpen ? true : undefined} className="min-w-0 overflow-x-hidden px-4 py-6 sm:px-5 md:px-8 lg:px-10 lg:py-9">{children}</main>
    </div>
  );
}

export function AppShell({ user, demoMode, children }: { user: CurrentUser; demoMode: boolean; children: ReactNode }) {
  return (
    <CurrentUserProvider user={user}>
      <Suspense fallback={<div className="min-h-screen bg-[#f6f5ee]" aria-label="Đang tải giao diện" />}>
        <ShellContent user={user} demoMode={demoMode}>{children}</ShellContent>
      </Suspense>
    </CurrentUserProvider>
  );
}
