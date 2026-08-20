import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  // These tests share one stateful API and exercise authentication limits.
  // Retrying a single spec reuses that state and can turn the original failure
  // into a misleading 429, so the workflow should fail on the first signal.
  retries: 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [["line"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.STUDIO_WEB_URL ?? "http://127.0.0.1:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    { name: "setup", testMatch: /.*\.setup\.ts/, use: { ...devices["Desktop Chrome"] } },
    { name: "chromium", testIgnore: /.*\.setup\.ts/, dependencies: ["setup"], use: { ...devices["Desktop Chrome"] } },
  ],
  outputDir: "test-results/playwright",
});
