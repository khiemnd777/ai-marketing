import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { proxyToStudioApi } from "./proxy";

describe("studio API runtime proxy", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubEnv("API_URL", "http://api.internal:8080");
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("resolves API_URL at request time and preserves mutation headers", async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ id: "client-1" }), {
      status: 201,
      headers: { "content-type": "application/json", "set-cookie": "studio_session=session; HttpOnly; Path=/" },
    }));
    const request = new Request("http://studio.local/api/studio/clients?source=e2e", {
      method: "POST",
      headers: {
        "content-type": "application/json",
        cookie: "studio_csrf=csrf-token",
        "x-csrf-token": "csrf-token",
        "idempotency-key": "idempotency-1",
      },
      body: JSON.stringify({ companyName: "Northstar" }),
    });

    const response = await proxyToStudioApi(request, { params: Promise.resolve({ path: ["clients"] }) });

    expect(response.status).toBe(201);
    expect(fetchMock).toHaveBeenCalledOnce();
    const [target, init] = fetchMock.mock.calls[0]!;
    expect(String(target)).toBe("http://api.internal:8080/v1/clients?source=e2e");
    expect(init?.method).toBe("POST");
    expect(new Headers(init?.headers).get("content-type")).toBe("application/json");
    expect(new Headers(init?.headers).get("x-csrf-token")).toBe("csrf-token");
    expect(new Headers(init?.headers).get("idempotency-key")).toBe("idempotency-1");
    expect(response.headers.getSetCookie()).toEqual(["studio_session=session; HttpOnly; Path=/"]);
  });

  it("returns a problem response instead of leaking network errors", async () => {
    fetchMock.mockRejectedValue(new Error("getaddrinfo ENOTFOUND api.internal"));

    const response = await proxyToStudioApi(
      new Request("http://studio.local/api/studio/clients"),
      { params: Promise.resolve({ path: ["clients"] }) },
    );

    expect(response.status).toBe(502);
    await expect(response.json()).resolves.toMatchObject({ title: "API unavailable", status: 502 });
  });
});
