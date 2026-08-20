"use client";

import type { components } from "@studio/api-client";
import { createContext, useContext, type ReactNode } from "react";

export type CurrentUser = components["schemas"]["InternalUser"];

const CurrentUserContext = createContext<CurrentUser | null>(null);

export function CurrentUserProvider({ user, children }: { user: CurrentUser; children: ReactNode }) {
  return <CurrentUserContext.Provider value={user}>{children}</CurrentUserContext.Provider>;
}

export function useCurrentUser() {
  const user = useContext(CurrentUserContext);
  if (!user) throw new Error("useCurrentUser must be used inside CurrentUserProvider");
  return user;
}

export function usePermissions() {
  const user = useCurrentUser();
  return {
    user,
    isAdmin: user.role === "ADMIN",
    canOperate: user.role === "ADMIN" || user.role === "OPERATOR",
    canReview: user.role === "ADMIN" || user.role === "REVIEWER",
  };
}
