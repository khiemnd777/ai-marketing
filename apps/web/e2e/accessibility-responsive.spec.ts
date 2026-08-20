import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const email = process.env.STUDIO_TEST_EMAIL;
const password = process.env.STUDIO_TEST_PASSWORD;

async function assertWcag(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(results.violations, results.violations.map((violation) => `${violation.id}: ${violation.help}`).join("\n")).toEqual([]);
}

async function login(page: Page) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email!);
  await page.getByLabel("Mật khẩu").fill(password!);
  await page.getByRole("button", { name: "Đăng nhập" }).click();
  await expect(page).toHaveURL(/\/clients(?:\?|$)/);
}

test.beforeEach(() => {
  test.skip(!email || !password, "STUDIO_TEST_EMAIL and STUDIO_TEST_PASSWORD are required");
});

test("login and authenticated shell satisfy automated WCAG A/AA checks", async ({ page }) => {
  await page.goto("/login");
  await assertWcag(page);
  await login(page);
  await expect(page.getByRole("heading", { name: "Khách hàng", exact: true })).toBeVisible();
  await assertWcag(page);
});

test("client portfolio exposes a recoverable API failure state", async ({ page }) => {
  await login(page);
  await page.route("**/api/studio/clients?**", async (route) => route.fulfill({
    status: 503,
    contentType: "application/problem+json",
    body: JSON.stringify({ title: "Temporarily unavailable", status: 503 }),
  }));
  await page.goto("/clients?search=failure-state");
  await expect(page.getByRole("heading", { name: "Không thể tải khách hàng" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  await assertWcag(page);
});

test.describe("mobile application shell", () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test("uses a keyboard-accessible drawer without horizontal overflow", async ({ page }) => {
    await login(page);
    await expect(page.getByRole("navigation", { name: "Danh mục" })).toHaveCount(0);
    const menuButton = page.getByRole("button", { name: "Mở điều hướng" });
    await expect(menuButton).toBeVisible();
    const menuBox = await menuButton.boundingBox();
    expect(menuBox?.width).toBeGreaterThanOrEqual(44);
    expect(menuBox?.height).toBeGreaterThanOrEqual(44);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await assertWcag(page);

    await menuButton.click();
    const drawer = page.getByRole("dialog", { name: "Điều hướng ứng dụng" });
    await expect(drawer).toBeVisible();
    await expect(drawer.getByRole("button", { name: "Đóng điều hướng" })).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(drawer).toHaveCount(0);
    await expect(menuButton).toBeFocused();
  });

  test("uses a full-height customer drawer with dirty-state protection", async ({ page }) => {
    await login(page);
    await page.getByRole("button", { name: "Thêm khách hàng" }).click();
    const drawer = page.getByRole("dialog", { name: "Thêm khách hàng" });
    await expect(drawer).toBeVisible();
    const companyName = drawer.getByLabel("Tên công ty");
    await expect(companyName).toBeFocused();
    await companyName.fill("Bản nháp chưa lưu");
    page.once("dialog", (dialog) => dialog.accept());
    await page.keyboard.press("Escape");
    await expect(drawer).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Thêm khách hàng" })).toBeFocused();
  });
});
