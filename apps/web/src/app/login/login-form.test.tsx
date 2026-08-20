import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthEntry, LoginForm } from "./login-form";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), replace: vi.fn(), refresh: vi.fn() }));

vi.mock("next/navigation", () => ({ useRouter: () => ({ replace: mocks.replace, refresh: mocks.refresh }) }));
vi.mock("@/lib/api", () => ({ api: { GET: mocks.get, POST: mocks.post } }));

function renderWithQuery(ui: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

describe("LoginForm", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(cleanup);

  it("explains that the login is internal", () => {
    renderWithQuery(<LoginForm />);
    expect(screen.getByRole("button", { name: "Đăng nhập" })).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveAttribute("autocomplete", "username");
  });

  it("shows normal login when an Admin already exists", async () => {
    mocks.get.mockResolvedValue({ data: { required: false } });
    renderWithQuery(<AuthEntry />);
    expect(await screen.findByRole("button", { name: "Đăng nhập" })).toBeInTheDocument();
  });

  it("creates and signs in the first Admin from the bootstrap UI", async () => {
    mocks.get.mockResolvedValue({ data: { required: true } });
    mocks.post.mockResolvedValue({ data: { id: "admin-1", role: "ADMIN", requiresPasswordChange: false } });
    renderWithQuery(<AuthEntry returnUrl="/clients" />);

    expect(await screen.findByRole("heading", { name: "Khởi tạo quản trị viên" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Họ tên"), { target: { value: "Local Admin" } });
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "admin@local.test" } });
    fireEvent.change(screen.getByLabelText("Mật khẩu", { exact: true }), { target: { value: "Local-password-2026!" } });
    fireEvent.change(screen.getByLabelText("Xác nhận mật khẩu"), { target: { value: "Local-password-2026!" } });
    fireEvent.click(screen.getByRole("button", { name: "Tạo quản trị viên và tiếp tục" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/auth/bootstrap", {
      body: { displayName: "Local Admin", email: "admin@local.test", password: "Local-password-2026!" },
    }));
    await waitFor(() => expect(mocks.replace).toHaveBeenCalledWith("/clients"));
  });
});
