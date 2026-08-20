import { expect, test } from "@playwright/test";

const email = process.env.STUDIO_TEST_EMAIL;
const password = process.env.STUDIO_TEST_PASSWORD;

test.beforeEach(() => {
  test.skip(!email || !password, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");
});

test("login, CSRF mutation, client creation, and workspace creation work through the production proxy", async ({ page }) => {
  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const companyName = `E2E Northstar ${suffix}`;
  const workspaceName = `E2E Workspace ${suffix}`;
  const workspaceSlug = `e2e-${suffix}`.toLowerCase();

  await page.goto("/login");
  await page.getByLabel("Email").fill(email!);
  await page.getByLabel("Mật khẩu").fill(password!);

  const loginResponsePromise = page.waitForResponse(
    (response) => response.url().includes("/api/studio/auth/login") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Đăng nhập" }).click();
  const loginResponse = await loginResponsePromise;
  expect(loginResponse.status()).toBe(200);
  expect(loginResponse.request().headers()["content-type"]).toContain("application/json");
  await expect(page).toHaveURL(/\/clients$/);
  await expect(page.getByRole("heading", { name: "Khách hàng" })).toBeVisible();

  await page.getByRole("button", { name: "Thêm khách hàng" }).click();
  await page.getByLabel("Tên công ty").fill(companyName);
  await page.getByLabel("Người liên hệ").fill("E2E Operator");
  await page.getByLabel("Email", { exact: true }).fill(`e2e-${suffix}@example.com`);
  await page.getByLabel("Ngành").fill("Travel");

  const clientResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/studio/clients") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Tạo khách hàng" }).click();
  const clientResponse = await clientResponsePromise;
  expect(clientResponse.status()).toBe(201);
  expect(clientResponse.request().headers()["x-csrf-token"]).toBeTruthy();
  expect(clientResponse.request().headers()["idempotency-key"]).toBeTruthy();
  await expect(page.getByRole("link", { name: companyName })).toBeVisible();

  await page.getByRole("link", { name: companyName }).click();
  await page.getByRole("button", { name: "Thêm workspace" }).click();
  await page.getByLabel("Tên workspace").fill(workspaceName);
  await page.getByLabel("Slug").fill(workspaceSlug);

  const workspaceResponsePromise = page.waitForResponse(
    (response) => /\/api\/studio\/clients\/[^/]+\/workspaces$/.test(new URL(response.url()).pathname)
      && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Tạo workspace" }).click();
  const workspaceResponse = await workspaceResponsePromise;
  expect(workspaceResponse.status()).toBe(201);
  expect(workspaceResponse.request().headers()["x-csrf-token"]).toBeTruthy();
  const workspaceLink = page.getByRole("link").filter({ hasText: workspaceName });
  await expect(workspaceLink).toBeVisible();

  const clientId = new URL(page.url()).pathname.split("/").at(-1)!;
  const workspaceHref = await workspaceLink.getAttribute("href");
  const workspaceId = new URL(workspaceHref!, page.url()).pathname.split("/").at(-1)!;
  await page.goto(`/media?clientId=${clientId}&workspaceId=${workspaceId}`);
  await expect(page.getByRole("heading", { name: "Thư viện media" })).toBeVisible();
  await expect(page.locator(".uppy-Dashboard")).toBeVisible();

  await page.goto(`/analytics?clientId=${clientId}&workspaceId=${workspaceId}`);
  await expect(page.getByRole("heading", { name: "Analytics & Learning" })).toBeVisible();

  const recommendationResponsePromise = page.waitForResponse(
    (response) => /\/analytics\/recommendations$/.test(new URL(response.url()).pathname)
      && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Tạo recommendation" }).click();
  const recommendationResponse = await recommendationResponsePromise;
  expect(recommendationResponse.status()).toBe(201);

  const missingCampaignId = "00000000-0000-4000-8000-000000000000";
  await page.goto(`/campaigns/${missingCampaignId}/composer?clientId=${clientId}&workspaceId=${workspaceId}`);
  await expect(page.getByRole("heading", { name: "Final Composer" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Quality" })).toBeVisible();
  await page.getByRole("link", { name: "Quality" }).click();
  await expect(page.getByRole("heading", { name: "Quality & Review" })).toBeVisible();
});
