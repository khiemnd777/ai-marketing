import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ProvidersPage from "./page";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), searchGet: vi.fn() }));

vi.mock("next/navigation", () => ({
  useParams: () => ({}),
  usePathname: () => "/settings/providers",
  useSearchParams: () => ({ get: mocks.searchGet }),
}));
vi.mock("@/lib/api", () => ({ api: { GET: mocks.get, PUT: mocks.put } }));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><ProvidersPage /></QueryClientProvider>);
}

const profile = {
  clientId: "018f47a0-7b5f-7d5f-9d2a-c5939813086f",
  demoMode: true,
  version: 4,
  providers: [{
    provider: "OPENAI" as const,
    enabled: true,
    configured: true,
    settings: {
      baseUrl: "https://api.openai.com/v1",
      model: "gpt-5.6-luna",
      transcriptionModel: "gpt-4o-mini-transcribe",
      reasoningEffort: "medium",
      timeoutSeconds: 60,
      inputUsdPer1M: 0,
      outputUsdPer1M: 0,
    },
    configuredSecretFields: ["apiKey" as const],
    version: 2,
  }],
};

describe("ProvidersPage", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(cleanup);

  it("requires a client scope before reading configuration", () => {
    mocks.searchGet.mockReturnValue(null);
    renderPage();
    expect(screen.getByRole("heading", { name: "Chọn khách hàng" })).toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalled();
  });

  it("shows only secret presence and confirms before enabling live mode", async () => {
    mocks.searchGet.mockImplementation((name: string) => name === "clientId" ? profile.clientId : null);
    mocks.get.mockResolvedValue({ data: profile });
    mocks.put.mockResolvedValue({ data: { ...profile, demoMode: false, version: 5 } });
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true);
    renderPage();

    expect(await screen.findByRole("heading", { name: "OpenAI" })).toBeInTheDocument();
    expect(screen.getByLabelText("API key")).toHaveAttribute("placeholder", "Đã lưu — để trống để giữ nguyên");
    expect(document.body.textContent).not.toContain("sk-client-secret");

    fireEvent.click(screen.getByRole("button", { name: "Chuyển sang live" }));
    expect(mocks.put).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Chuyển sang live" }));
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith(
      "/clients/{clientId}/provider-configuration/mode",
      { params: { path: { clientId: profile.clientId } }, body: { demoMode: false, version: 4 } },
    ));
    confirm.mockRestore();
  });
});
