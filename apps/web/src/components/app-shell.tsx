"use client";

import type { components } from "@studio/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  BarChart3,
  BriefcaseBusiness,
  ChevronRight,
  Clapperboard,
  DatabaseZap,
  FolderKanban,
  LayoutDashboard,
  LibraryBig,
  LogOut,
  Menu,
  Megaphone,
  PackageSearch,
  Palette,
  PanelsTopLeft,
  ShieldCheck,
  UserCog,
  UsersRound,
  X,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { CurrentUserProvider, type CurrentUser } from "@/components/auth-context";
import { useStudioScope } from "@/components/studio-scope";
import { api } from "@/lib/api";
import { cn } from "@/lib/cn";
import { apiError } from "@/lib/problem";
import { canonicalLegacyRoute, studioRoutes, workspaceDestination } from "@/lib/studio-routes";

type Client = components["schemas"]["Client"];
type Workspace = components["schemas"]["Workspace"];
type NavItem = {
  href: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  exact?: boolean;
  aliases?: string[];
};

function useScopeClients() {
  return useQuery({
    queryKey: ["scope-clients"],
    queryFn: async () => {
      const { data, error } = await api.GET("/clients", { params: { query: { page: 1, pageSize: 100 } } });
      if (error || !data) throw apiError(error, "Không thể tải phạm vi khách hàng.");
      return data.items;
    },
  });
}

function useScopeWorkspaces(clientId: string) {
  return useQuery({
    queryKey: ["scope-workspaces", clientId],
    enabled: Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/workspaces", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải phạm vi workspace.");
      return data.items;
    },
  });
}

function NavLink({ item, onNavigate }: { item: NavItem; onNavigate?: () => void }) {
  const pathname = usePathname();
  const aliases = item.aliases ?? [];
  const active = item.exact
    ? pathname === item.href || aliases.some((alias) => pathname === alias || pathname.startsWith(`${alias}/`))
    : pathname === item.href || pathname.startsWith(`${item.href}/`) || aliases.some((alias) => pathname === alias || pathname.startsWith(`${alias}/`));
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      className={cn(
        "group flex min-h-11 items-center gap-3 rounded-2xl px-3 text-sm font-semibold transition",
        active ? "bg-[var(--lime)] text-[var(--ink)]" : "text-[#5e685f] hover:bg-white/75 hover:text-[var(--ink)]",
      )}
      aria-current={active ? "page" : undefined}
    >
      <Icon className="size-4.5 shrink-0" />
      <span className="min-w-0 flex-1 truncate">{item.label}</span>
      {active ? <ChevronRight className="size-4 shrink-0" /> : null}
    </Link>
  );
}

function NavSection({ label, items, onNavigate }: { label: string; items: NavItem[]; onNavigate: () => void }) {
  if (!items.length) return null;
  return (
    <section className="mt-5">
      <h2 className="mb-2 px-3 text-[0.65rem] font-black uppercase tracking-[0.16em] text-[var(--muted)]">{label}</h2>
      <nav aria-label={label} className="grid gap-1">
        {items.map((item) => <NavLink key={item.href} item={item} onNavigate={onNavigate} />)}
      </nav>
    </section>
  );
}

function ScopeSwitcher({
  clientId,
  workspaceId,
  clients,
  workspaces,
  loadingWorkspaces,
  providerMode,
  hasError,
}: {
  clientId: string;
  workspaceId: string;
  clients: Client[];
  workspaces: Workspace[];
  loadingWorkspaces: boolean;
  providerMode?: boolean;
  hasError: boolean;
}) {
  const pathname = usePathname();
  const router = useRouter();

  const selectClient = (nextClientId: string) => {
    if (!nextClientId) {
      router.push(studioRoutes.clients);
      return;
    }
    localStorage.setItem("studio:last-client", nextClientId);
    router.push(studioRoutes.client(nextClientId));
  };

  const selectWorkspace = (nextWorkspaceId: string) => {
    if (!clientId || !nextWorkspaceId) {
      router.push(clientId ? studioRoutes.client(clientId) : studioRoutes.clients);
      return;
    }
    localStorage.setItem("studio:last-scope", JSON.stringify({ clientId, workspaceId: nextWorkspaceId }));
    router.push(workspaceDestination(pathname, clientId, nextWorkspaceId));
  };

  return (
    <div className="rounded-3xl border border-[var(--line)] bg-white/75 p-3" aria-label="Phạm vi làm việc">
      <label className="grid gap-1.5 text-[0.65rem] font-black uppercase tracking-[0.14em] text-[var(--muted)]">
        Khách hàng
        <select
          className="min-h-11 w-full rounded-xl border border-[var(--line)] bg-white px-3 text-sm font-bold normal-case tracking-normal text-[var(--ink)]"
          value={clientId}
          onChange={(event) => selectClient(event.target.value)}
        >
          <option value="">Chọn khách hàng</option>
          {clients.map((item) => <option key={item.id} value={item.id}>{item.companyName}{item.status === "ARCHIVED" ? " · Đã lưu trữ" : ""}</option>)}
        </select>
      </label>
      <label className="mt-3 grid gap-1.5 text-[0.65rem] font-black uppercase tracking-[0.14em] text-[var(--muted)]">
        Workspace
        <select
          className="min-h-11 w-full rounded-xl border border-[var(--line)] bg-white px-3 text-sm font-bold normal-case tracking-normal text-[var(--ink)] disabled:opacity-50"
          value={workspaceId}
          disabled={!clientId || loadingWorkspaces}
          onChange={(event) => selectWorkspace(event.target.value)}
        >
          <option value="">{clientId ? "Chọn workspace" : "Chọn khách hàng trước"}</option>
          {workspaces.map((item) => <option key={item.id} value={item.id}>{item.name}{item.status === "ARCHIVED" ? " · Đã lưu trữ" : ""}</option>)}
        </select>
      </label>
      {clientId && providerMode !== undefined ? (
        <div className={cn("mt-3 flex items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold", providerMode ? "bg-[#fff0c9] text-[#79580b]" : "bg-[#dcefdc] text-[#28643c]") }>
          <DatabaseZap className="size-3.5" />{providerMode ? "Provider: Demo" : "Provider: Live"}
        </div>
      ) : null}
      {hasError ? <p role="alert" className="mt-3 text-xs font-semibold text-[var(--coral)]">Không thể tải đầy đủ phạm vi làm việc.</p> : null}
    </div>
  );
}

function ShellContent({ user, children }: { user: CurrentUser; children: ReactNode }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { clientId, workspaceId } = useStudioScope();
  const [navigationOpen, setNavigationOpen] = useState(false);
  const navigationRef = useRef<HTMLElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const clients = useScopeClients();
  const workspaces = useScopeWorkspaces(clientId);
  const providerMode = useQuery({
    queryKey: ["provider-configuration", clientId],
    enabled: user.role === "ADMIN" && Boolean(clientId),
    queryFn: async () => {
      const { data, error } = await api.GET("/clients/{clientId}/provider-configuration", { params: { path: { clientId } } });
      if (error || !data) throw apiError(error, "Không thể tải provider mode.");
      return data;
    },
  });
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

  const clientItems = clients.data ?? [];
  const workspaceItems = workspaces.data ?? [];
  const selectedClient = clientItems.find((item) => item.id === clientId);
  const selectedWorkspace = workspaceItems.find((item) => item.id === workspaceId);
  const closeNavigation = () => setNavigationOpen(false);
  const clientItemsNav: NavItem[] = clientId ? [
    { href: studioRoutes.client(clientId), label: "Tổng quan", icon: LayoutDashboard, exact: true },
    { href: studioRoutes.clientProfile(clientId), label: "Hồ sơ & liên hệ", icon: BriefcaseBusiness, exact: true },
    { href: studioRoutes.clientWorkspaces(clientId), label: "Workspaces", icon: PanelsTopLeft, exact: true },
    ...(user.role === "ADMIN" ? [{ href: studioRoutes.clientProviders(clientId), label: "Nhà cung cấp", icon: DatabaseZap, exact: true } satisfies NavItem] : []),
  ] : [];
  const workspaceItemsNav: NavItem[] = clientId && workspaceId ? [
    { href: studioRoutes.workspace(clientId, workspaceId), label: "Tổng quan", icon: LayoutDashboard, exact: true, aliases: ["/workspaces"] },
    { href: studioRoutes.brands(clientId, workspaceId), label: "Thương hiệu", icon: Palette },
    { href: studioRoutes.products(clientId, workspaceId), label: "Sản phẩm & Product Truth", icon: PackageSearch, aliases: ["/products"] },
    { href: studioRoutes.media(clientId, workspaceId), label: "Thư viện media", icon: LibraryBig, aliases: ["/media"] },
    { href: studioRoutes.characters(clientId, workspaceId), label: "Nhân vật", icon: UsersRound, aliases: ["/settings/characters"] },
    { href: studioRoutes.campaigns(clientId, workspaceId), label: "Chiến dịch", icon: FolderKanban, aliases: ["/campaigns"] },
    { href: studioRoutes.meta(clientId, workspaceId), label: "Kết nối Meta", icon: Megaphone, aliases: ["/settings/meta"] },
    { href: studioRoutes.analytics(clientId, workspaceId), label: "Phân tích", icon: BarChart3, aliases: ["/analytics"] },
  ] : [];
  const systemItems: NavItem[] = [
    ...(user.role === "ADMIN" ? [
      { href: "/operations", label: "Vận hành", icon: Activity } satisfies NavItem,
      { href: "/internal-users", label: "Người dùng nội bộ", icon: UserCog } satisfies NavItem,
    ] : []),
    { href: "/account", label: "Tài khoản", icon: ShieldCheck },
  ];

  useEffect(() => {
    const canonical = canonicalLegacyRoute(pathname, { clientId, workspaceId });
    if (!canonical) return;
    const remaining = new URLSearchParams(searchParams.toString());
    remaining.delete("clientId");
    remaining.delete("workspaceId");
    const serialized = remaining.toString();
    router.replace(serialized ? `${canonical}?${serialized}` : canonical);
  }, [clientId, pathname, router, searchParams, workspaceId]);

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

  const contextTitle = selectedClient?.companyName ?? "Marketing Ops";
  const contextSubtitle = selectedWorkspace?.name ?? (clientId ? "Tổng quan khách hàng" : "AI Product Marketing Studio");

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[280px_minmax(0,1fr)]">
      <a href="#main-content" className="skip-link">Bỏ qua điều hướng</a>
      <header inert={navigationOpen ? true : undefined} aria-hidden={navigationOpen ? true : undefined} className="sticky top-0 z-30 flex min-h-16 items-center justify-between border-b border-[var(--line)] bg-[#edf0e7]/95 px-4 backdrop-blur lg:hidden">
        <Link href={clientId ? studioRoutes.client(clientId) : studioRoutes.clients} className="flex min-w-0 items-center gap-3" aria-label="Về trang tổng quan">
          <span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-[var(--ink)] text-[var(--lime)]"><Clapperboard className="size-5" /></span>
          <span className="min-w-0"><span className="block truncate text-sm font-bold text-[var(--ink)]">{contextTitle}</span><span className="block truncate text-xs text-[var(--muted)]">{contextSubtitle}</span></span>
        </Link>
        <button ref={menuButtonRef} type="button" className="grid size-11 shrink-0 place-items-center rounded-2xl border border-[var(--line)] bg-white" aria-label="Mở điều hướng" aria-controls="studio-navigation" aria-expanded={navigationOpen} onClick={() => setNavigationOpen(true)}><Menu className="size-5" /></button>
      </header>
      {navigationOpen ? <button type="button" tabIndex={-1} className="fixed inset-0 z-40 bg-[var(--ink)]/45 backdrop-blur-[1px] lg:hidden" aria-label="Đóng điều hướng" onClick={closeNavigation} /> : null}
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
          <Link href={studioRoutes.clients} className="flex min-w-0 flex-1 items-center gap-3 px-2" onClick={closeNavigation}>
            <span className="grid size-11 place-items-center rounded-2xl bg-[var(--ink)] text-[var(--lime)]"><Clapperboard className="size-5" /></span>
            <span><span className="block text-xs font-bold uppercase tracking-[0.18em] text-[var(--moss)]">AI Studio</span><span className="block font-serif text-lg font-bold text-[var(--ink)]">Marketing Ops</span></span>
          </Link>
          <button data-drawer-close type="button" className="grid size-11 shrink-0 place-items-center rounded-2xl hover:bg-white lg:hidden" aria-label="Đóng điều hướng" onClick={closeNavigation}><X className="size-5" /></button>
        </div>

        <ScopeSwitcher
          clientId={clientId}
          workspaceId={workspaceId}
          clients={clientItems}
          workspaces={workspaceItems}
          loadingWorkspaces={workspaces.isLoading}
          providerMode={providerMode.data?.demoMode}
          hasError={Boolean(clients.error || workspaces.error || providerMode.error)}
        />

        <NavSection label="Danh mục" items={[{ href: studioRoutes.clients, label: "Tất cả khách hàng", icon: BriefcaseBusiness, exact: true }]} onNavigate={closeNavigation} />
        <NavSection label={selectedClient?.companyName ?? "Khách hàng"} items={clientItemsNav} onNavigate={closeNavigation} />
        <NavSection label={selectedWorkspace?.name ?? "Workspace"} items={workspaceItemsNav} onNavigate={closeNavigation} />
        <NavSection label="Hệ thống" items={systemItems} onNavigate={closeNavigation} />

        <div className="mt-5 flex items-center gap-3 rounded-3xl border border-[var(--line)] bg-white/75 p-3 lg:mt-auto">
          <span className="grid size-10 shrink-0 place-items-center rounded-full bg-[var(--lime)] text-sm font-black text-[var(--ink)]">{user.displayName.slice(0, 1).toUpperCase()}</span>
          <Link href="/account" className="min-w-0 flex-1" onClick={closeNavigation}><span className="block truncate text-sm font-bold">{user.displayName}</span><span className="block text-xs text-[var(--muted)]">{user.role}</span></Link>
          <button className="grid size-11 place-items-center rounded-full text-[var(--muted)] hover:bg-white hover:text-[var(--ink)]" aria-label="Đăng xuất" disabled={logout.isPending} onClick={() => logout.mutate()}><LogOut className="size-4" /></button>
        </div>
        {logout.error ? <p role="alert" className="mt-2 text-xs font-semibold text-[var(--coral)]">{logout.error.message}</p> : null}
      </aside>
      <main id="main-content" tabIndex={-1} inert={navigationOpen ? true : undefined} aria-hidden={navigationOpen ? true : undefined} className="min-w-0 overflow-x-hidden px-4 py-6 sm:px-5 md:px-8 lg:px-10 lg:py-9">
        <div className="mx-auto w-full max-w-[1600px]">{children}</div>
      </main>
    </div>
  );
}

export function AppShell({ user, children }: { user: CurrentUser; children: ReactNode }) {
  return (
    <CurrentUserProvider user={user}>
      <Suspense fallback={<div className="min-h-screen bg-[#f6f5ee]" aria-label="Đang tải giao diện" />}>
        <ShellContent user={user}>{children}</ShellContent>
      </Suspense>
    </CurrentUserProvider>
  );
}
