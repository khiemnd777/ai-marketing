import { createHmac, timingSafeEqual } from "node:crypto";

export function signManifest(body: string, secret: string): string {
  return createHmac("sha256", secret).update(body).digest("hex");
}

export function verifyManifestSignature(body: string, supplied: string, secret: string): boolean {
  if (!/^[a-f0-9]{64}$/.test(supplied)) return false;
  const expected = Buffer.from(signManifest(body, secret), "hex");
  const actual = Buffer.from(supplied, "hex");
  return expected.length === actual.length && timingSafeEqual(expected, actual);
}
