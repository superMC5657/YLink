import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

/**
 * 角色区分 E2E:管理员登录可见管理后台入口并可访问 admin 页面;
 * 普通用户不可见管理菜单,直接访问 /admin/* 被重定向到 /dashboard。
 * Mock 管理员:admin@example.com / Admin@123456(见 mock/auth.ts)。
 */
async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/#/login')
  await page.getByPlaceholder('邮箱').fill(email)
  await page.getByPlaceholder('密码').fill(password)
  await page.locator('button', { hasText: '登录' }).first().click()
  await expect(page).toHaveURL(/#\/dashboard/)
}

test.describe('角色区分(管理员)', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('管理员登录后侧边栏显示管理后台菜单并可进入用户管理', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await expect(page.getByText('管理后台')).toBeVisible()
    await page.getByText('用户管理').first().click()
    await expect(page).toHaveURL(/#\/admin\/users/)
    // 用户列表渲染(Mock)
    await expect(page.locator('table tbody tr').first()).toBeVisible()
    await expect(page.getByText('admin@example.com').first()).toBeVisible()
  })

  test('移动端抽屉内显示管理后台菜单', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    // 打开抽屉菜单(顶栏汉堡钮)
    await page.locator('header button').first().click()
    await expect(page.getByText('管理后台')).toBeVisible()
    await page.getByText('用户管理').first().click()
    await expect(page).toHaveURL(/#\/admin\/users/)
  })

  test('管理员顶栏下拉出现管理后台入口', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await page
      .locator('button[title*="浅色"], button[title*="深色"], button[title*="跟随系统"]')
      .first()
      .waitFor()
    // 打开用户下拉(顶栏最右侧按钮)
    await page.locator('header button').last().click()
    await expect(page.getByText('管理后台').first()).toBeVisible()
  })

  test('管理员可查看并回复用户工单', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await page.getByText('工单管理').first().click()
    await expect(page).toHaveURL(/#\/admin\/tickets/)
    // 工单列表渲染(有数据行,含发起用户)
    await expect(page.locator('table tbody tr').first()).toBeVisible()
    // 打开详情弹窗,出现回复输入框
    await page.getByText('查看').first().click()
    await expect(page.getByPlaceholder(/输入回复内容/)).toBeVisible()
    // 回复成功
    await page.getByPlaceholder(/输入回复内容/).fill('已收到您的工单,我们会尽快处理')
    await page.locator('button', { hasText: '回复' }).click()
    await expect(page.getByText('回复成功')).toBeVisible()
  })
})

test.describe('角色区分(普通用户)', () => {
  test('普通用户看不到管理后台菜单', async ({ authedPage }) => {
    await authedPage.goto('/#/dashboard')
    await expect(authedPage.getByText('管理后台')).toBeHidden()
    await expect(authedPage.getByText('用户管理')).toBeHidden()
  })

  test('普通用户直接访问 admin 页面被重定向到仪表板', async ({ authedPage }) => {
    await authedPage.goto('/#/admin/overview')
    await expect(authedPage).toHaveURL(/#\/dashboard/)
  })
})
