import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { csrfFetch } from "./api";

describe("csrfFetch", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    document.cookie = "studio_csrf=test-csrf-token; path=/";
  });

  afterEach(() => {
    document.cookie = "studio_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("preserves a Request method and content type while adding CSRF", async () => {
    const request = new Request("http://localhost/api/studio/auth/login", {
      method: "POST",
      headers: { "content-type": "application/json", "x-request-id": "request-1" },
      body: JSON.stringify({ email: "operator@example.com", password: "long-enough-password" }),
    });

    await csrfFetch(request);

    expect(fetchMock).toHaveBeenCalledOnce();
    const [input, init] = fetchMock.mock.calls[0]!;
    const forwarded = new Request(input, init);
    expect(forwarded.method).toBe("POST");
    expect(forwarded.headers.get("content-type")).toBe("application/json");
    expect(forwarded.headers.get("x-request-id")).toBe("request-1");
    expect(forwarded.headers.get("x-csrf-token")).toBe("test-csrf-token");
    expect(forwarded.headers.get("accept")).toBe("application/json, application/problem+json");
    expect(forwarded.credentials).toBe("include");
  });

  it("does not attach CSRF to safe methods", async () => {
    await csrfFetch(new Request("http://localhost/api/studio/clients"));

    expect(fetchMock).toHaveBeenCalledOnce();
    const [input, init] = fetchMock.mock.calls[0]!;
    const forwarded = new Request(input, init);
    expect(forwarded.method).toBe("GET");
    expect(forwarded.headers.has("x-csrf-token")).toBe(false);
  });
});
