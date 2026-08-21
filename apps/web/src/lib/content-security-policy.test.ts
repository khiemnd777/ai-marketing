import { describe, expect, it } from "vitest";
import { contentSecurityPolicy, parseBrowserStorageOrigins } from "./content-security-policy";

describe("contentSecurityPolicy", () => {
  it("allows configured object storage for uploads and signed previews", () => {
    const policy = contentSecurityPolicy("http://localhost:9100 https://*.r2.cloudflarestorage.com");
    for (const directive of ["connect-src", "img-src", "media-src"]) {
      expect(policy).toMatch(new RegExp(`${directive}[^;]*http://localhost:9100`));
      expect(policy).toMatch(new RegExp(`${directive}[^;]*https://\\*\\.r2\\.cloudflarestorage\\.com`));
    }
  });

  it("deduplicates origins and rejects values that could inject a directive", () => {
    expect(parseBrowserStorageOrigins("https://storage.example https://storage.example")).toEqual(["https://storage.example"]);
    expect(() => parseBrowserStorageOrigins("https://storage.example; script-src *")).toThrow(/Invalid BROWSER_STORAGE_ORIGINS/);
    expect(() => parseBrowserStorageOrigins("https://user:pass@storage.example")).toThrow(/Invalid BROWSER_STORAGE_ORIGINS/);
    expect(() => parseBrowserStorageOrigins("https://storage.example/upload")).toThrow(/Invalid BROWSER_STORAGE_ORIGINS/);
  });

  it("remains self-only when no browser storage origin is configured", () => {
    expect(contentSecurityPolicy(undefined)).toContain("connect-src 'self'");
  });
});
