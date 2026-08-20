import type { components } from "@studio/api-client";
import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { AppShell } from "@/components/app-shell";

type CurrentUser = components["schemas"]["InternalUser"];

function safeReturnUrl(value: string | null) {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/clients";
}

async function loadCurrentUser() {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.getAll().map(({ name, value }) => `${name}=${value}`).join("; ");
  const apiBase = process.env.API_URL ?? "http://127.0.0.1:8080";
  const response = await fetch(new URL("/v1/auth/me", apiBase), {
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (response.status === 401) return null;
  if (!response.ok) throw new Error(`Authentication service returned ${response.status}`);
  return response.json() as Promise<CurrentUser>;
}

export default async function StudioLayout({ children }: { children: ReactNode }) {
  const requestHeaders = await headers();
  const returnUrl = safeReturnUrl(requestHeaders.get("x-studio-return-url"));
  const user = await loadCurrentUser();
  if (!user) redirect(`/login?returnUrl=${encodeURIComponent(returnUrl)}`);
  if (user.requiresPasswordChange) redirect(`/account/password?returnUrl=${encodeURIComponent(returnUrl)}`);
  return <AppShell user={user} demoMode={process.env.DEMO_MODE === "true"}>{children}</AppShell>;
}
