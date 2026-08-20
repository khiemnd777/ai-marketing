import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PasswordChangeForm } from "./password-change-form";

const mocks = vi.hoisted(() => ({ post: vi.fn(), push: vi.fn(), replace: vi.fn(), refresh: vi.fn() }));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mocks.push, replace: mocks.replace, refresh: mocks.refresh }),
}));
vi.mock("@/lib/api", () => ({ api: { POST: mocks.post } }));

function renderForm(forced: boolean) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PasswordChangeForm returnUrl="/account" forced={forced} />
    </QueryClientProvider>,
  );
}

describe("PasswordChangeForm", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(cleanup);

  it("lets a user return without changing a voluntary password", () => {
    renderForm(false);

    fireEvent.click(screen.getByRole("button", { name: "Quay lại" }));

    expect(mocks.push).toHaveBeenCalledWith("/account");
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("does not bypass a password change required by policy", () => {
    renderForm(true);

    expect(screen.queryByRole("button", { name: "Quay lại" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Đổi mật khẩu và tiếp tục" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Đăng xuất" })).toBeInTheDocument();
  });
});
