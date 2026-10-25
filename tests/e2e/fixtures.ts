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

/**
 * 角色区分链路共享步骤(admin.spec / withdraw.spec 共用,收敛前两份逐字重复):
 * - loginAs:表单登录,管理员落点为门户分流页 #/portal(spec admin-console-split §3.4);
 * - enterPortal:门户双卡片二选一(entryText = 卡片标题);
 * - openAdminMenu:desktop 侧边栏常驻直接点菜单;mobile(视口 < 768)菜单在抽屉里,先开抽屉。
 */
export async function loginAs(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/#/login')
  await page.getByPlaceholder('邮箱').fill(email)
  await page.getByPlaceholder('密码').fill(password)
  await page.locator('button', { hasText: '登录' }).first().click()
  // 管理员登录落点:门户分流页(spec admin-console-split §3.4)
  await expect(page).toHaveURL(/#\/portal/)
}

/** 从门户页进入指定端(entryText = 卡片标题) */
export async function enterPortal(
  page: Page,
  entryText: string,
  urlPattern: RegExp,
): Promise<void> {
  await page.getByText(entryText).click()
  await expect(page).toHaveURL(urlPattern)
}

/** 进入管理端页面(桌面点侧边栏菜单,移动先开抽屉) */
export async function openAdminMenu(page: Page, menuText: string): Promise<void> {
  if ((page.viewportSize()?.width ?? 1440) < 768) {
    await page.locator('header button').first().click()
  }
  await page.getByText(menuText).first().click()
}

export const test = base.extend<{ authedPage: Page }>({
  authedPage: async ({ page }, use) => {
    await injectAuth(page)
    await use(page)
  },
})

export { expect }
