import { expect, test as setup } from "@playwright/test";

const adminEmail = process.env.STUDIO_TEST_EMAIL;
const adminPassword = process.env.STUDIO_TEST_PASSWORD;

setup("bootstrap the first Admin through the UI when required", async ({ page }) => {
  setup.skip(!adminEmail || !adminPassword, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");
  await page.goto("/login");

  const bootstrapButton = page.getByRole("button", { name: "Tạo quản trị viên và tiếp tục" });
  const loginButton = page.getByRole("button", { name: "Đăng nhập" });
  await expect(bootstrapButton.or(loginButton)).toBeVisible();
  const bootstrapRequired = await bootstrapButton.isVisible();
  if (process.env.STUDIO_EXPECT_ADMIN_BOOTSTRAP === "true") expect(bootstrapRequired).toBe(true);
  if (!bootstrapRequired) return;

  await page.getByLabel("Họ tên").fill("E2E Administrator");
  await page.getByLabel("Email").fill(adminEmail!);
  await page.getByLabel("Mật khẩu", { exact: true }).fill(adminPassword!);
  await page.getByLabel("Xác nhận mật khẩu").fill(adminPassword!);
  const responsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/studio/auth/bootstrap") && response.request().method() === "POST",
  );
  await bootstrapButton.click();
  expect((await responsePromise).status()).toBe(201);
  await expect(page).toHaveURL(/\/clients(?:\?|$)/);
});
