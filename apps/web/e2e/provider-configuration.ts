import { expect, type Page } from "@playwright/test";

const defaults = {
  OPENAI: {
    baseUrl: "https://api.openai.com/v1", model: "demo-openai", transcriptionModel: "demo-transcription",
    reasoningEffort: "medium", timeoutSeconds: 60, inputUsdPer1M: 0, outputUsdPer1M: 0,
  },
  SEEDANCE: {
    baseUrl: "https://ark.ap-southeast.bytepluses.com/api", model: "demo-seedance", apiVersion: "v3",
    resolution: "720p", aspectRatio: "9:16", callbackUrl: "", timeoutSeconds: 30,
    pollIntervalSeconds: 1, taskTimeoutSeconds: 1800, usdPerSecond: 0,
  },
  META: {
    appId: "demo-meta-app", apiVersion: "demo", redirectUrl: "http://127.0.0.1:8080/v1/meta/oauth/callback",
    graphBaseUrl: "https://graph.facebook.com", dialogBaseUrl: "https://www.facebook.com",
  },
};

export async function configureDemoProviders(page: Page, clientId: string) {
  const csrf = (await page.context().cookies()).find((cookie) => cookie.name === "studio_csrf")?.value;
  expect(csrf).toBeTruthy();
  const headers = { "x-csrf-token": csrf!, accept: "application/json", "content-type": "application/json" };
  const providers = [
    { provider: "OPENAI", settings: defaults.OPENAI, secrets: {} },
    { provider: "SEEDANCE", settings: defaults.SEEDANCE, secrets: {} },
    {
      provider: "R2",
      settings: { accountId: "e2e", bucket: process.env.STUDIO_TEST_R2_BUCKET ?? "e2e-media", endpoint: process.env.STUDIO_TEST_R2_ENDPOINT ?? "http://127.0.0.1:9000", publicBaseUrl: "" },
      secrets: { accessKeyId: process.env.STUDIO_TEST_R2_ACCESS_KEY_ID ?? "e2e-access-key", secretAccessKey: process.env.STUDIO_TEST_R2_SECRET_ACCESS_KEY ?? "e2e-secret-key" },
    },
    { provider: "META", settings: defaults.META, secrets: {} },
    { provider: "RENDERER", settings: { baseUrl: process.env.STUDIO_TEST_RENDERER_URL ?? "http://127.0.0.1:8090" }, secrets: {} },
  ] as const;
  for (const item of providers) {
    const response = await page.request.put(`/api/studio/clients/${clientId}/provider-configuration/${item.provider}`, {
      headers,
      data: { enabled: true, settings: item.settings, secrets: item.secrets, clearSecrets: [], version: 0 },
    });
    expect(response.status(), `${item.provider} configuration`).toBe(200);
  }
}
