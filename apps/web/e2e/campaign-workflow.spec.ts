import { expect, test, type Page } from "@playwright/test";
import { configureDemoProviders } from "./provider-configuration";

const email = process.env.STUDIO_TEST_EMAIL;
const password = process.env.STUDIO_TEST_PASSWORD;

async function login(page: Page) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email!);
  await page.getByLabel("Mật khẩu").fill(password!);
  await page.getByRole("button", { name: "Đăng nhập" }).click();
  await expect(page).toHaveURL(/\/clients(?:\?|$)/);
}

async function waitForGeneration(page: Page, operation: string) {
  const responsePromise = page.waitForResponse((response) => response.url().endsWith("/generation-jobs") && response.request().method() === "POST");
  await page.getByRole("button", { name: `Tạo ${operation}` }).click();
  expect((await responsePromise).status()).toBe(202);
  await expect(page.getByText("SUCCEEDED", { exact: true }).first()).toBeVisible({ timeout: 20_000 });
}

async function clickAndExpect(page: Page, buttonName: string, path: RegExp, method: string, status: number) {
  const responsePromise = page.waitForResponse((response) => path.test(new URL(response.url()).pathname) && response.request().method() === method);
  await page.getByRole("button", { name: buttonName }).first().click();
  expect((await responsePromise).status()).toBe(status);
}

test.beforeEach(() => {
  test.skip(!email || !password, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");
});

test("complete no-cost product truth to analytics journey", async ({ page }) => {
  test.setTimeout(12 * 60_000);
  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const clientName = `Journey Client ${suffix}`;
  const workspaceName = `Journey Workspace ${suffix}`;
  const brandName = `Journey Brand ${suffix}`;
  const productName = `Cabin Journey ${suffix}`;
  const campaignName = `Launch Journey ${suffix}`;
  const primaryName = `Host ${suffix}`;
  const listenerName = `Traveler ${suffix}`;

  await login(page);
  await page.getByRole("button", { name: "Thêm khách hàng" }).click();
  await page.getByLabel("Tên công ty").fill(clientName);
  await page.getByLabel("Người liên hệ").fill("Journey Operator");
  await page.getByLabel("Email", { exact: true }).fill(`journey-${suffix}@example.com`);
  await page.getByLabel("Ngành").fill("Travel");
  await page.getByRole("button", { name: "Tạo khách hàng" }).click();
  await expect(page).toHaveURL(/\/clients\/[0-9a-f-]+$/);

  await page.getByRole("button", { name: "Thêm workspace" }).click();
  await page.getByLabel("Tên workspace").fill(workspaceName);
  await page.getByLabel("Slug").fill(`journey-${suffix}`.toLowerCase());
  await page.getByRole("button", { name: "Tạo workspace" }).click();
  const workspaceLink = page.getByRole("link").filter({ hasText: workspaceName });
  await expect(workspaceLink).toBeVisible();
  const workspaceHref = await workspaceLink.getAttribute("href");
  const workspaceUrl = new URL(workspaceHref!, page.url());
  const workspaceParts = workspaceUrl.pathname.split("/");
  const clientId = workspaceParts[2]!;
  const workspaceId = workspaceParts[4]!;
  await configureDemoProviders(page, clientId);
  await workspaceLink.click();

  await page.getByRole("button", { name: "Thêm thương hiệu" }).click();
  await page.getByLabel("Tên thương hiệu").fill(brandName);
  await page.getByLabel("Giọng điệu").fill("Tin cậy, thực tế và rõ ràng");
  await page.getByRole("button", { name: "Tạo thương hiệu" }).click();
  await expect(page.getByRole("link").filter({ hasText: brandName })).toBeVisible();

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/products`);
  await page.getByRole("button", { name: "Thêm vali" }).click();
  await page.getByLabel("Tên sản phẩm").fill(productName);
  await page.getByLabel("SKU").fill(`SKU-${suffix}`.toUpperCase());
  await page.getByLabel("Model").fill("Cabin 20");
  await page.getByRole("button", { name: "Tạo bản nháp" }).click();
  const productLink = page.getByRole("link").filter({ hasText: productName });
  await expect(productLink).toBeVisible();
  await productLink.click();

  await page.getByRole("button", { name: "Thêm fact" }).click();
  await page.getByLabel("Fact key").fill("external_dimensions");
  await page.getByLabel("Nhãn").fill("Kích thước ngoài");
  await page.getByLabel("Giá trị chính xác").fill("55 x 36 x 23 cm");
  await page.getByLabel("Nguồn", { exact: true }).fill("Product specification");
  await page.getByLabel("Trích đoạn nguồn").fill("External dimensions: 55 x 36 x 23 cm");
  await page.getByRole("button", { name: "Lưu fact nháp" }).click();
  await page.getByRole("button", { name: "Duyệt & khóa" }).click();
  await expect(page.getByText("APPROVED", { exact: true }).first()).toBeVisible();

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/characters`);
  for (const [name, description] of [[primaryName, "Professional product host"], [listenerName, "Frequent traveler listening naturally"]] as const) {
    await page.getByRole("button", { name: "Thêm nhân vật" }).click();
    await page.getByLabel("Tên", { exact: true }).fill(name);
    await page.getByLabel("Mô tả ngoại hình").fill(description);
    await page.getByRole("button", { name: "Tạo nhân vật" }).click();
    await expect(page.getByRole("heading", { name })).toBeVisible();
  }

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/campaigns`);
  await page.getByRole("button", { name: "Campaign mới" }).click();
  await page.getByLabel("Tên campaign").fill(campaignName);
  await page.getByLabel("Brand").selectOption({ label: brandName });
  await page.getByLabel("Sản phẩm").selectOption({ label: productName });
  await page.getByRole("button", { name: "Tạo brief" }).click();
  await page.getByRole("link", { name: campaignName }).click();
  await expect(page).toHaveURL(/\/clients\/[0-9a-f-]+\/workspaces\/[0-9a-f-]+\/campaigns\/[0-9a-f-]+$/);
  const campaignId = new URL(page.url()).pathname.split("/").at(-1)!;
  expect(campaignId).toMatch(/^[0-9a-f-]+$/);
  await page.getByLabel("Người nói").selectOption({ label: `${primaryName} · NOT_REQUIRED` });
  await page.getByLabel("Người nghe").selectOption({ label: `${listenerName} · NOT_REQUIRED` });
  await page.getByRole("button", { name: "Khóa cặp nhân vật" }).click();

  await page.getByRole("link", { name: "Concept" }).click();
  await waitForGeneration(page, "concepts");
  await expect(page.getByRole("button", { name: "Duyệt" }).first()).toBeVisible();
  await page.getByRole("button", { name: "Duyệt" }).first().click();
  await page.getByRole("button", { name: "Khóa concept" }).first().click();

  await page.getByRole("link", { name: "Nội dung" }).click();
  await waitForGeneration(page, "content");
  await expect(page.locator("textarea")).toHaveCount(14);
  await page.getByRole("button", { name: "Duyệt" }).first().click();

  await page.getByRole("link", { name: "Kịch bản" }).click();
  await waitForGeneration(page, "script");
  await page.getByRole("button", { name: "Duyệt script" }).click();
  await expect(page.getByText("APPROVED", { exact: true }).first()).toBeVisible();

  await page.getByRole("link", { name: "Cảnh quay" }).click();
  await waitForGeneration(page, "scenes");
  await expect(page.getByRole("button", { name: "Duyệt" })).toHaveCount(4);
  for (let remaining = 4; remaining > 0; remaining -= 1) {
    await clickAndExpect(page, "Duyệt", /\/scenes\/[^/]+\/approve$/, "POST", 200);
    await expect(page.getByRole("button", { name: "Duyệt" })).toHaveCount(remaining - 1);
  }

  await expect(page.getByRole("button", { name: "Tạo take · 720p" })).toHaveCount(4);
  for (let index = 0; index < 4; index += 1) {
    const responsePromise = page.waitForResponse((response) => /\/scenes\/[^/]+\/generations$/.test(new URL(response.url()).pathname) && response.request().method() === "POST");
    await page.getByRole("button", { name: "Tạo take · 720p" }).nth(index).click();
    expect((await responsePromise).status()).toBe(202);
  }
  await expect(page.getByText("REVIEW_REQUIRED", { exact: true })).toHaveCount(4, { timeout: 3 * 60_000 });

  await page.getByRole("link", { name: "Quality" }).click();
  await expect(page.getByRole("heading", { name: "Quality & Review" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Duyệt take" })).toHaveCount(4);
  for (let remaining = 4; remaining > 0; remaining -= 1) {
    await clickAndExpect(page, "Duyệt take", /\/generations\/[^/]+\/review$/, "PUT", 200);
    await expect(page.getByRole("button", { name: "Duyệt take" })).toHaveCount(remaining - 1);
  }
  await expect(page.getByRole("button", { name: "Chọn cho Composer" })).toHaveCount(4);
  for (let remaining = 4; remaining > 0; remaining -= 1) {
    await clickAndExpect(page, "Chọn cho Composer", /\/generations\/[^/]+\/select$/, "POST", 200);
    await expect(page.getByRole("button", { name: "Chọn cho Composer" })).toHaveCount(remaining - 1);
  }

  await page.getByRole("link", { name: "Composer" }).click();
  await expect(page.getByText("4/4 scene sẵn sàng", { exact: true })).toBeVisible();
  await clickAndExpect(page, "Render MP4", /\/final-renders$/, "POST", 202);
  await expect(page.getByText("REVIEW_REQUIRED", { exact: true })).toBeVisible({ timeout: 6 * 60_000 });

  await page.getByRole("link", { name: "Quality" }).click();
  await clickAndExpect(page, "Duyệt final", /\/final-renders\/[^/]+\/review$/, "PUT", 200);
  await clickAndExpect(page, "Chọn output", /\/final-renders\/[^/]+\/select$/, "POST", 200);
  await expect(page.getByText("Campaign output", { exact: true })).toBeVisible();

  await page.goto(`/settings/meta?clientId=${clientId}&workspaceId=${workspaceId}`);
  await page.getByRole("button", { name: "Kết nối Meta" }).click();
  await expect(page).toHaveURL(/\/settings\/meta\?.*connected=1/, { timeout: 30_000 });
  await expect(page.getByRole("heading", { name: "Demo Meta Operator" })).toBeVisible();
  await expect(page.getByText("CONNECTED", { exact: true }).first()).toBeVisible();

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/campaigns/${campaignId}/publishing`);
  await expect(page.getByRole("heading", { name: "Tạo publishing request" })).toBeVisible();
  await clickAndExpect(page, "Gửi để duyệt", /\/social-posts$/, "POST", 201);
  await clickAndExpect(page, "Duyệt publish", /\/social-posts\/[^/]+\/review$/, "PUT", 200);
  await expect(page.getByText("PUBLISHED", { exact: true })).toBeVisible({ timeout: 30_000 });

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/campaigns/${campaignId}/ads`);
  await clickAndExpect(page, "Lưu guardrails", /\/meta-ad-guardrails$/, "PUT", 200);
  await clickAndExpect(page, "Tạo để duyệt", /\/meta-ad-campaigns$/, "POST", 201);
  await page.getByLabel("Xác nhận tạo campaign PAUSED").fill("CREATE PAUSED VND 100000");
  await clickAndExpect(page, "Duyệt tạo PAUSED", /\/meta-ad-campaigns\/[^/]+\/review$/, "PUT", 200);
  await expect(page.getByText("PAUSED", { exact: true }).first()).toBeVisible({ timeout: 30_000 });

  await page.goto(`/clients/${clientId}/workspaces/${workspaceId}/analytics?campaignId=${campaignId}`);
  await expect(page.getByRole("heading", { name: "Analytics & Learning" })).toBeVisible();
  await expect(page.getByText("Chi phí provider", { exact: true })).toBeVisible();
  await expect.poll(async () => {
    await page.reload();
    return page.getByText("Ads CTR", { exact: true }).locator("..").innerText();
  }, { timeout: 30_000 }).not.toContain("0 / 0 clicks");
  await clickAndExpect(page, "Tạo recommendation", /\/analytics\/recommendations$/, "POST", 201);
});
