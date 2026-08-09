import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright 配置:Mock 环境固定跑 E2E(文档 frontend/README.md §7.3 / §9)。
 * webServer 自动拉起 dev server(Mock 开启)。
 */
export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'desktop-chromium',
      use: { ...devices['Desktop Chrome'], channel: process.env.CI ? undefined : 'chrome' },
    },
    {
      name: 'mobile-chromium',
      use: { ...devices['Pixel 5'], channel: process.env.CI ? undefined : 'chrome' },
    },
  ],
  webServer: {
    // --mode e2e 强制使用 .env.e2e(Mock),避免 .env.development.local 联调覆盖影响测试稳定性
    command: 'pnpm dev --mode e2e',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
})
