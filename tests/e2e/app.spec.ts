import { expect, test } from './fixtures'

/**
 * E2E 主链路:登录 → 仪表板 → 购买套餐(Mock 支付)→ 订单详情。
 * 对应文档 frontend/README.md §9 E2E 场景。
 */
test.describe('登录', () => {
  test('登录页渲染并成功登录跳转仪表板', async ({ page }) => {
    await page.goto('/#/login')
    await expect(page).toHaveTitle(/YLink/)
    await expect(page.locator('button', { hasText: '登录' }).first()).toBeVisible()

    // 登录页不预填账号(Mock 固定演示账号,手动填写)
    await page.getByPlaceholder('邮箱').fill('2734921923@qq.com')
    await page.getByPlaceholder('密码').fill('Passw0rd')
    await page.locator('button', { hasText: '登录' }).first().click()
    await expect(page).toHaveURL(/#\/dashboard/)
    await expect(page.getByText('钱包余额')).toBeVisible()
  })

  test('已登录访问 guest 页重定向仪表板', async ({ authedPage }) => {
    await authedPage.goto('/#/login')
    await expect(authedPage).toHaveURL(/#\/dashboard/)
  })
})

test.describe('仪表板', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('展示余额/订阅/公告/快捷操作', async ({ authedPage }) => {
    await authedPage.goto('/#/dashboard')
    await expect(authedPage.getByText('钱包余额')).toBeVisible()
    await expect(authedPage.getByText('我的佣金')).toBeVisible()
    await expect(authedPage.getByText('当前订阅')).toBeVisible()
    await expect(authedPage.getByText('公告')).toBeVisible()
    await expect(authedPage.getByText('快捷操作')).toBeVisible()
  })
})

test.describe('购买套餐(交易闭环)', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('下单 → 余额支付 → 完成', async ({ authedPage }) => {
    await authedPage.goto('/#/plans')
    await expect(authedPage.getByText('白羊座').first()).toBeVisible()

    await authedPage.locator('button', { hasText: '立即购买' }).first().click()
    await expect(authedPage.getByText('套餐周期')).toBeVisible()
    await expect(authedPage.getByText('费用明细')).toBeVisible()

    // 余额支付(余额充足,直接完成)
    await authedPage.locator('button', { hasText: '余额支付' }).first().click()
    await authedPage.locator('button', { hasText: '提交订单' }).first().click()
    // 弹窗关闭即支付完成
    await expect(authedPage.getByText('费用明细')).toBeHidden({ timeout: 8000 })
  })

  test('优惠券校验失败显示错误', async ({ authedPage }) => {
    await authedPage.goto('/#/plans')
    await authedPage.locator('button', { hasText: '立即购买' }).first().click()
    await authedPage.locator('input[placeholder*="优惠码"]').fill('BADCODE')
    await authedPage.locator('button', { hasText: '校验' }).click()
    // 内联错误(限定弹窗内,避免与全局 toast 文案冲突)
    await expect(
      authedPage.getByRole('dialog').getByText('优惠券无效或已过期'),
    ).toBeVisible()
  })
})

test.describe('订单', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('订单列表渲染状态徽章与详情弹窗', async ({ authedPage }) => {
    await authedPage.goto('/#/orders')
    await expect(authedPage.locator('table tbody tr').first()).toBeVisible()
    await expect(authedPage.getByText('已完成').first()).toBeVisible()

    await authedPage.getByText('查看详情').first().click()
    await expect(authedPage.getByText('订单详情')).toBeVisible()
    await expect(authedPage.getByText('订单号').first()).toBeVisible()
  })
})

test.describe('页面可达性', () => {
  const routes: [string, string][] = [
    ['/docs', '使用文档'],
    ['/invite', '邀请赚钱'],
    ['/agent', '申请代理'],
    ['/nodes', '节点状态'],
    ['/profile', '个人信息'],
    ['/tickets', '我的工单'],
    ['/traffic', '流量明细'],
  ]

  for (const [path, heading] of routes) {
    test(`页面 ${path} 渲染`, async ({ authedPage }) => {
      await authedPage.goto('/#' + path)
      await expect(authedPage.getByText(heading).first()).toBeVisible()
    })
  }
})

test.describe('暗色模式', () => {
  test('三态切换到达 dark', async ({ authedPage }) => {
    await authedPage.goto('/#/dashboard')
    const toggle = authedPage.locator('button[title*="浅色"], button[title*="深色"], button[title*="跟随系统"]').first()
    let theme = 'light'
    for (let i = 0; i < 3; i++) {
      await toggle.click()
      theme = (await authedPage.evaluate(() => document.documentElement.getAttribute('data-theme'))) ?? 'light'
      if (theme === 'dark') break
    }
    expect(theme).toBe('dark')
  })
})
