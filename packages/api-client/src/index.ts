import createClient from "openapi-fetch";
import type { paths } from "./generated/schema";

export type { components, operations, paths } from "./generated/schema";

export function createStudioClient(baseUrl: string, fetchImplementation: typeof fetch = fetch) {
  return createClient<paths>({
    baseUrl,
    credentials: "include",
    fetch: fetchImplementation,
  });
}
