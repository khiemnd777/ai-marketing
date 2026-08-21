import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("web container packaging", () => {
  it("copies vertical packs before compiling the Next.js application", () => {
    const dockerfile = readFileSync(resolve(process.cwd(), "../../infra/docker/web.Dockerfile"), "utf8");
    const copyVerticals = dockerfile.indexOf("COPY verticals verticals");
    const buildWeb = dockerfile.indexOf("RUN cd apps/web && node node_modules/next/dist/bin/next build --webpack");

    expect(copyVerticals).toBeGreaterThan(-1);
    expect(buildWeb).toBeGreaterThan(copyVerticals);
  });
});
