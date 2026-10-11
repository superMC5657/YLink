import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

/**
 * 提现/订阅关键路径 E2E(F02 提现闭环 + F10 订阅模板):
 * - 提交→审核→不可直接关单:全程行为断言(toast/按钮显隐/内容渲染),
 *   不直断 14003 错误码(死码断言)。
 * Mock 数据见 mock/business.ts(id=42 提现工单)与 mock/admin.ts(id=3 提现工单)。
 */
async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/#/login')
  await page.getByPlaceholder('邮箱').fill(email)
  await page.getByPlaceholder('密码').fill(password)
  await page.locator('button', { hasText: '登录' }).first().click()
  await expect(page).toHaveURL(/#\/portal/)
}

async function enterPortal(page: Page, entryText: string, urlPattern: RegExp) {
  await page.getByText(entryText).click()
  await expect(page).toHaveURL(urlPattern)
}

async function openAdminMenu(page: Page, menuText: string) {
  if ((page.viewportSize()?.width ?? 1440) < 768) {
    await page.locator('header button').first().click()
  }
  await page.getByText(menuText).first().click()
}

test.describe('佣金提现(提交→审核→不可直接关单)', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('用户提交佣金提现成功,提现工单用户侧不可关闭', async ({ authedPage }) => {
    await authedPage.goto('/#/invite')
    await expect(authedPage.getByRole('heading', { name: '提现记录' })).toBeVisible()
    // 提交:打开提现弹窗并填写
    await authedPage.getByText('申请提现').click()
    await expect(authedPage.getByText('佣金提现申请')).toBeVisible()
    await authedPage.locator('.n-modal input').first().fill('10')
    await authedPage.getByPlaceholder('支付宝账号').fill('demo@example.com')
    await authedPage.locator('.n-modal button', { hasText: '提交' }).click()
    // 行为断言:提交成功 toast(不直断业务码)
    await expect(authedPage.getByText('提现申请已提交')).toBeVisible()

    // 不可直接关单:提现工单详情隐藏关闭/重开入口(后端 14003 的前端行为表现)
    await authedPage.goto('/#/tickets/42')
    await expect(authedPage.getByText('佣金提现申请')).toBeVisible()
    await expect(authedPage.getByRole('button', { name: '关闭工单' })).toBeHidden()
    await expect(authedPage.getByRole('button', { name: '重新打开' })).toBeHidden()
  })

  test('管理员对提现工单确认打款完成审核', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    await openAdminMenu(page, '工单管理')
    await expect(page).toHaveURL(/#\/admin\/tickets/)
    // 提现行打标(行为:列表透出提现身份)
    await expect(page.locator('tbody tr', { hasText: '佣金提现申请' }).first()).toBeVisible()
    await page
      .locator('tbody tr', { hasText: '佣金提现申请' })
      .first()
      .locator('button', { hasText: '查看' })
      .click()
    // 审核入口:提现信息卡 + 确认打款/拒绝退回(行为:须经审核闭环,非直接关单)
    await expect(page.getByText('提现申请信息')).toBeVisible()
    await expect(page.getByText('确认打款')).toBeVisible()
    await page.getByText('确认打款').click()
    await expect(page.getByText('确认已完成线下打款')).toBeVisible()
    await page.locator('.n-modal button', { hasText: '确认' }).last().click()
    await expect(page.getByText('已确认打款')).toBeVisible()
  })
})

test.describe('订阅模板(预览与保存)', () => {
  test.use({ viewport: { width: 1440, height: 900 } })

  test('订阅模板预览渲染示例内容,编辑保存成功', async ({ page }) => {
    await loginAs(page, 'admin@example.com', 'Admin@123456')
    await enterPortal(page, '管理后台', /#\/admin\/overview/)
    await openAdminMenu(page, '订阅模板')
    await expect(page).toHaveURL(/#\/admin\/subscription-templates/)
    await expect(page.locator('tbody tr', { hasText: 'clash' }).first()).toBeVisible()
    // 预览:示例数据渲染(行为:出现渲染后的 clash 配置片段)
    await page
      .locator('tbody tr', { hasText: 'clash' })
      .first()
      .locator('button', { hasText: '预览' })
      .click()
    await expect(page.locator('.n-modal pre', { hasText: 'mixed-port' })).toBeVisible()
    await page.keyboard.press('Escape')
    // 编辑保存:内容可改并保存成功(行为:toast,不直断接口码)
    await page
      .locator('tbody tr', { hasText: 'clash' })
      .first()
      .locator('button', { hasText: '编辑' })
      .click()
    await expect(page.getByText('编辑订阅模板')).toBeVisible()
    const editor = page.locator('.n-modal textarea').first()
    await editor.fill('mixed-port: 7890\nmode: rule\nproxies:\n{{.NodeBlock}}')
    await expect(editor).toHaveValue(/mixed-port/)
    await page.locator('.n-modal button', { hasText: '保存' }).click()
    await expect(page.getByText('模板已保存')).toBeVisible()
  })
})
