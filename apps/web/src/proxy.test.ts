import { NextRequest } from "next/server";
import { afterEach, describe, expect, it } from "vitest";
import { proxy } from "./proxy";

const originalStorageOrigins = process.env.BROWSER_STORAGE_ORIGINS;

afterEach(() => {
  if (originalStorageOrigins === undefined) delete process.env.BROWSER_STORAGE_ORIGINS;
  else process.env.BROWSER_STORAGE_ORIGINS = originalStorageOrigins;
});

describe("web proxy security headers", () => {
  it("allows the configured storage origin on login and authenticated pages", () => {
    process.env.BROWSER_STORAGE_ORIGINS = "http://localhost:9100";

    const login = proxy(new NextRequest("http://localhost:3300/login"));
    expect(login.headers.get("content-security-policy")).toContain("connect-src 'self' http://localhost:9100");

    const request = new NextRequest("http://localhost:3300/clients", { headers: { cookie: "studio_session=test" } });
    const authenticated = proxy(request);
    expect(authenticated.headers.get("content-security-policy")).toContain("img-src 'self' data: blob: http://localhost:9100");
  });
});
