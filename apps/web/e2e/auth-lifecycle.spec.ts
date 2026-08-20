import { expect, test, type Page } from "@playwright/test";

const adminEmail = process.env.STUDIO_TEST_EMAIL;
const adminPassword = process.env.STUDIO_TEST_PASSWORD;

async function login(page: Page, email: string, password: string) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Mật khẩu").fill(password);
  await page.getByRole("button", { name: "Đăng nhập" }).click();
}

async function changePassword(page: Page, currentPassword: string, newPassword: string) {
  await page.getByLabel("Mật khẩu hiện tại").fill(currentPassword);
  await page.getByLabel("Mật khẩu mới", { exact: true }).fill(newPassword);
  await page.getByLabel("Xác nhận mật khẩu mới").fill(newPassword);
  const responsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/studio/auth/change-password") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Đổi mật khẩu và tiếp tục" }).click();
  expect((await responsePromise).status()).toBe(204);
}

test.beforeEach(() => {
  test.skip(!adminEmail || !adminPassword, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");
});

test("auth gate, forced password change, admin reset/status, session revocation, and workspace scope", async ({ page, browser }) => {
  const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const operatorEmail = `operator-${suffix}@example.com`;
  const temporaryPassword = `Temporary-${suffix}-A1!`;
  const changedPassword = `Changed-${suffix}-B2!`;
  const resetPassword = `Reset-${suffix}-C3!`;
  const finalPassword = `Final-${suffix}-D4!`;
  const clientName = `Auth Client ${suffix}`;
  const workspaceName = `Auth Workspace ${suffix}`;

  await page.goto("/clients");
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fclients/);
  await expect(page.getByRole("navigation", { name: "Danh mục" })).toHaveCount(0);

  await login(page, adminEmail!, adminPassword!);
  await expect(page).toHaveURL(/\/clients(?:\?|$)/);
  await page.getByRole("button", { name: "Thêm khách hàng" }).click();
  await page.getByLabel("Tên công ty").fill(clientName);
  await page.getByLabel("Người liên hệ").fill("Auth Administrator");
  await page.getByLabel("Email", { exact: true }).fill(`auth-${suffix}@example.com`);
  await page.getByLabel("Ngành").fill("Software");
  await page.getByRole("button", { name: "Tạo khách hàng" }).click();
  await expect(page.getByRole("heading", { name: clientName })).toBeVisible();
  await page.getByRole("button", { name: "Thêm workspace" }).click();
  await page.getByLabel("Tên workspace").fill(workspaceName);
  await page.getByLabel("Slug").fill(`auth-${suffix}`.toLowerCase());
  await page.getByRole("button", { name: "Tạo workspace" }).click();
  await expect(page.getByRole("link").filter({ hasText: workspaceName })).toBeVisible();

  await page.goto("/internal-users");
  await page.getByRole("button", { name: "Thêm người dùng" }).click();
  await page.getByLabel("Họ tên").fill(`Operator ${suffix}`);
  await page.getByLabel("Email", { exact: true }).fill(operatorEmail);
  await page.getByLabel("Vai trò").selectOption("OPERATOR");
  await page.getByLabel("Mật khẩu tạm thời").fill(temporaryPassword);
  const createResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/api/studio/internal-users") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Tạo tài khoản" }).click();
  expect((await createResponsePromise).status()).toBe(201);
  const userCard = page.locator(`[data-user-email="${operatorEmail}"]`);
  await expect(userCard).toBeVisible();
  await expect(userCard).toContainText("Cần đổi mật khẩu");

  const operatorContext = await browser.newContext();
  const operatorPage = await operatorContext.newPage();
  await login(operatorPage, operatorEmail, temporaryPassword);
  await expect(operatorPage).toHaveURL(/\/account\/password/);
  await operatorPage.goto("/clients");
  await expect(operatorPage).toHaveURL(/\/account\/password/);
  await changePassword(operatorPage, temporaryPassword, changedPassword);
  await expect(operatorPage).toHaveURL(/\/clients(?:\?|$)/);
  await expect(operatorPage.getByRole("link", { name: "Người dùng" })).toHaveCount(0);
  await expect(operatorPage.getByRole("link", { name: "Vận hành" })).toHaveCount(0);

  const clientSelect = operatorPage.getByRole("combobox").nth(0);
  await expect(clientSelect).toBeVisible();
  await expect.poll(async () => clientSelect.locator("option").count()).toBeGreaterThan(1);
  await clientSelect.selectOption({ index: 1 });
  const workspaceSelect = operatorPage.getByRole("combobox").nth(1);
  await expect(workspaceSelect).toBeEnabled();
  await expect.poll(async () => workspaceSelect.locator("option").count()).toBeGreaterThan(1);
  await workspaceSelect.selectOption({ index: 1 });
  const productLink = operatorPage.getByRole("link", { name: "Sản phẩm & Product Truth" });
  await expect(productLink).toHaveAttribute("href", /\/clients\/.+\/workspaces\/.+\/products/);
  await productLink.click();
  await expect(operatorPage).toHaveURL(/\/clients\/.+\/workspaces\/.+\/products/);
  await expect(operatorPage.getByText("Chưa chọn workspace")).toHaveCount(0);

  await page.reload();
  await expect(userCard).toBeVisible();
  await userCard.getByRole("button", { name: "Reset mật khẩu" }).click();
  await page.getByLabel("Mật khẩu tạm thời").fill(resetPassword);
  const resetResponsePromise = page.waitForResponse(
    (response) => response.url().includes("/reset-password") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Reset và thu hồi session" }).click();
  expect((await resetResponsePromise).status()).toBe(200);

  await operatorPage.goto("/account");
  await expect(operatorPage).toHaveURL(/\/login/);
  await login(operatorPage, operatorEmail, resetPassword);
  await expect(operatorPage).toHaveURL(/\/account\/password/);
  await changePassword(operatorPage, resetPassword, finalPassword);
  await expect(operatorPage).toHaveURL(/\/clients(?:\?|$)/);

  await page.reload();
  await expect(userCard).toBeVisible();
  const disableResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/status") && response.request().method() === "PATCH",
  );
  await userCard.getByRole("button", { name: "Vô hiệu hóa" }).click();
  expect((await disableResponsePromise).status()).toBe(200);
  await operatorPage.goto("/clients");
  await expect(operatorPage).toHaveURL(/\/login/);

  const enableResponsePromise = page.waitForResponse(
    (response) => response.url().endsWith("/status") && response.request().method() === "PATCH",
  );
  await userCard.getByRole("button", { name: "Kích hoạt" }).click();
  expect((await enableResponsePromise).status()).toBe(200);

  await login(operatorPage, operatorEmail, finalPassword);
  await expect(operatorPage).toHaveURL(/\/clients(?:\?|$)/);
  await operatorPage.goto("/account");
  await expect(operatorPage.getByText("Phiên hiện tại")).toBeVisible();
  await operatorPage.getByRole("button", { name: "Đăng xuất phiên này" }).click();
  await expect(operatorPage).toHaveURL(/\/login/);
  await operatorContext.close();
});
