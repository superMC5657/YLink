import { test as base, expect } from '@playwright/test'
import type { Page } from '@playwright/test'

/**
 * 注入 Mock 会话到 localStorage(持久化键 'app:auth',带 app: 前缀)。
 * 持久面最小化:仅 refreshToken + user(accessToken 只存内存,首个请求 401 后
 * 经 mock /auth/refresh 静默刷新补回,mock-refresh-* 一律放行)。
 * Mock token 无状态校验:任意 'Bearer mock-access-*' 有效。
 */
export async function injectAuth(page: Page): Promise<void> {
  await page.goto('/#/login', { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => {
    localStorage.setItem(
      'app:auth',
      JSON.stringify({
        refreshToken: 'mock-refresh-e2e',
        user: { id: 10086, email: 'e2e@test.com', role: 0 },
        // 会话恢复要求 savedAt 必填(缺失视为过期脏数据丢弃):E2E 预置须带当前时间
        savedAt: Date.now(),
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
