import { describe, expect, it } from "vitest";
import { signManifest, verifyManifestSignature } from "../src/auth.js";

describe("manifest authentication", () => {
  it("accepts only the exact HMAC", () => {
    const signature = signManifest('{"renderId":"one"}', "renderer-secret-at-least-32-bytes");
    expect(verifyManifestSignature('{"renderId":"one"}', signature, "renderer-secret-at-least-32-bytes")).toBe(true);
    expect(verifyManifestSignature('{"renderId":"two"}', signature, "renderer-secret-at-least-32-bytes")).toBe(false);
  });
});
