import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

/**
 * 角色区分 E2E(管理端/用户端分拆布局):
 * - 管理员登录落点为门户分流页 #/portal,双卡片二选一(用户中心/管理后台);
 * - 管理端(AdminLayout)侧边栏/移动抽屉只含 13 项管理菜单,底部「返回用户中心」;
 * - 用户端侧边栏/抽屉不再出现管理菜单,管理员通过底部按钮/顶栏下拉进入管理端;
 * - 普通用户不可见管理入口,访问 /admin/* 与 /portal 均重定向 /dashboard。
 * Mock 管理员:admin@example.com / Admin@123456(见 mock/auth.ts)。
 */
async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/#/login')
  await page.getByPlaceholder('邮箱').fill(email)
  await page.getByPlaceholder('密码').fill(password)
  await page.locator('button', { hasText: '登录' }).first().click()
  // 管理员登录落点:门户分流页(spec admin-console-split §3.4)
  await expect(page).toHaveURL(/#\/portal/)
}

/** 从门户页进入指定端(entryText = 卡片标题) */
async function enterPortal(page: Page, entryText: string, urlPattern: RegExp) {
  await page.getByText(entryText).click()
  await expect(page).toHaveURL(urlPattern)
}

test.describe('角色区分(管理员)', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('管理员登录落门户页并可进入管理后台查看用户管理', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    // 门户双卡片(用户中心/管理后台)
    await expect(page.getByText('用户中心')).toBeVisible()
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    // 管理端侧边栏只含管理菜单,无用户端菜单(A3)
    await expect(page.getByText('仪表板')).toBeHidden()
    await page.getByText('用户管理').first().click()
    await expect(page).toHaveURL(/#\/admin\/users/)
    // 用户列表渲染(Mock)
    await expect(page.locator('table tbody tr').first()).toBeVisible()
    await expect(page.getByText('admin@example.com').first()).toBeVisible()
  })

  test('管理端底部「返回用户中心」回到用户端,侧边栏底部按钮进入管理端', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    // 用户中心卡 → 用户端
    await enterPortal(page, '用户中心', /#\/dashboard/)
    // 用户端侧边栏底部「进入管理后台」→ 管理端
    await page.getByText('进入管理后台').click()
    await expect(page).toHaveURL(/#\/admin\/overview/)
    // 管理端侧边栏底部「返回用户中心」→ 用户端(A3)
    await page.getByText('返回用户中心').click()
    await expect(page).toHaveURL(/#\/dashboard/)
  })

  test('管理员顶栏下拉出现进入管理后台入口(用户端)', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '用户中心', /#\/dashboard/)
    await page
      .locator('button[title*="浅色"], button[title*="深色"], button[title*="跟随系统"]')
      .first()
      .waitFor()
    // 打开用户下拉(顶栏最右侧按钮)
    await page.locator('header button').last().click()
    await expect(page.locator('.n-dropdown-option', { hasText: '进入管理后台' })).toBeVisible()
  })

  test('两端顶栏下拉对称互切(A4)', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    // 管理端下拉:返回用户中心
    await page.locator('header button').last().click()
    await page.locator('.n-dropdown-option', { hasText: '返回用户中心' }).click()
    await expect(page).toHaveURL(/#\/dashboard/)
    // 用户端下拉:进入管理后台
    await page.locator('header button').last().click()
    await page.locator('.n-dropdown-option', { hasText: '进入管理后台' }).click()
    await expect(page).toHaveURL(/#\/admin\/overview/)
  })

  test('管理员可查看并回复用户工单', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
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

test.describe('角色区分(管理员·移动端)', () => {
  test('移动端用户端抽屉不含管理菜单(A5)', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '用户中心', /#\/dashboard/)
    // 打开用户端抽屉(顶栏汉堡钮),断言限定在抽屉(role=dialog)内
    await page.locator('header button').first().click()
    const drawer = page.getByRole('dialog')
    await expect(drawer.getByRole('button', { name: '仪表板' })).toBeVisible()
    // 用户端抽屉无管理菜单(分组标题/管理项均不可见)
    await expect(drawer.getByText('管理后台')).toBeHidden()
    await expect(drawer.getByText('用户管理')).toBeHidden()
  })

  test('移动端管理端独立抽屉只含管理菜单并可操作(A5)', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    // 打开管理端独立抽屉
    await page.locator('header button').first().click()
    const drawer = page.getByRole('dialog')
    await expect(drawer.getByRole('button', { name: '用户管理' })).toBeVisible()
    // 独立抽屉不含用户端菜单
    await expect(drawer.getByText('仪表板')).toBeHidden()
    // 可正常操作:进入用户管理
    await drawer.getByRole('button', { name: '用户管理' }).click()
    await expect(page).toHaveURL(/#\/admin\/users/)
    await expect(page.locator('table tbody tr').first()).toBeVisible()
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

  test('普通用户访问门户页被重定向到仪表板(A6)', async ({ authedPage }) => {
    await authedPage.goto('/#/portal')
    await expect(authedPage).toHaveURL(/#\/dashboard/)
  })
})

test.describe('邮件模板管理(F11 回归:locale 字面量花括号)', () => {
  /**
   * 回归背景:zh/en 语言包 adminMailTemplates.syntaxTip/bodyPlaceholder 含 Go template
   * 字面量 {{.site_name}},被 vue-i18n 当插值解析 → 编译错误在 dev 下抛出 →
   * 编辑/测试发送弹窗渲染中断,点击「编辑」无反应(见 docs/frontend/progress.md)。
   * 修复:文案改用字面量插值 {'{{.site_name}}'} 转义;配套 vitest 单测
   * src/locales/__tests__/i18n.spec.ts 锁全部消息可编译。
   */
  test('邮件模板列表可打开编辑弹窗并保存成功,全程无渲染错误', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push('pageerror: ' + e.message))
    page.on('console', (m) => {
      if (m.type() === 'error') errors.push('console.error: ' + m.text())
    })

    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    await page.getByText('邮件模板').first().click()
    await expect(page).toHaveURL(/#\/admin\/mail-templates/)
    await expect(page.locator('table tbody tr').first()).toBeVisible()

    // 编辑弹窗打开 + 输入可编辑 + 保存成功
    await page.locator('tbody tr').first().locator('button', { hasText: '编辑' }).click()
    await expect(page.getByText('编辑模板：')).toBeVisible()
    const subject = page.locator('.n-modal input')
    await subject.fill('回归测试主题')
    await expect(subject).toHaveValue('回归测试主题')
    await page.locator('.n-modal button', { hasText: '保存' }).click()
    await expect(page.getByText('模板已保存')).toBeVisible()

    // 「测试发送」弹窗同一渲染路径
    await page.locator('tbody tr').first().locator('button', { hasText: '测试发送' }).click()
    await expect(page.getByText('测试发送：')).toBeVisible()

    expect(errors).toEqual([])
  })
})

test.describe('总览快捷操作(按运营频率分组)', () => {
  test('总览快捷操作含两组共 8 个入口,分组标签与跳转正确', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)

    const quick = page.locator('.card-base', { hasText: '快捷操作' })
    await expect(quick.getByText('日常运营')).toBeVisible()
    await expect(quick.getByText('运营与配置')).toBeVisible()

    // 8 个入口齐全(链接元素)
    for (const label of [
      '用户管理',
      '订单管理',
      '工单管理',
      '代理审批',
      '公告管理',
      '优惠券管理',
      '流量管理',
      '节点管理',
    ]) {
      await expect(quick.locator('a', { hasText: label })).toBeVisible()
    }
    // 跳转抽查:高频组 → 用户管理,周期组 → 流量管理
    await expect(quick.locator('a', { hasText: '工单管理' })).toHaveAttribute(
      'href',
      /#\/admin\/tickets/,
    )
    await expect(quick.locator('a', { hasText: '流量管理' })).toHaveAttribute(
      'href',
      /#\/admin\/traffic-import/,
    )
  })
})
