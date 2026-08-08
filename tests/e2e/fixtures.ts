import { test as base, expect } from '@playwright/test'
import type { Page } from '@playwright/test'

/**
 * 注入 Mock 会话到 localStorage(persistedstate key 为 'app:auth',带 app: 前缀)。
 * Mock token 无状态校验:任意 'Bearer mock-access-*' 有效。
 */
export async function injectAuth(page: Page): Promise<void> {
  await page.goto('/#/login', { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => {
    localStorage.setItem(
      'app:auth',
      JSON.stringify({
        accessToken: 'mock-access-e2e-' + Date.now(),
        refreshToken: 'mock-refresh-e2e',
        user: { id: 10086, email: 'e2e@test.com', role: 0 },
      }),
    )
  })
  // bootstrap 是异步的(main.ts),重载让 auth.restore() 读到刚写入的会话
  await page.reload({ waitUntil: 'domcontentloaded' })
}

export const test = base.extend<{ authedPage: Page }>({
  authedPage: async ({ page }, use) => {
    await injectAuth(page)
    await use(page)
  },
})

export { expect }
