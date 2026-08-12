import { expect, test } from './fixtures'

/**
 * 移动端冒烟(390×844 视口):仪表板单列布局 + 底部标签栏导航。
 * 对应文档 frontend/README.md §9「移动端视口冒烟」。
 */
test.use({ viewport: { width: 390, height: 844 } })

test('移动端仪表板渲染 + 底栏导航到购买订阅', async ({ authedPage }) => {
  await authedPage.goto('/#/dashboard')
  await expect(authedPage.getByText('钱包余额')).toBeVisible()
  await expect(authedPage.getByText('购买订阅').first()).toBeVisible()

  // 底栏 Tab 跳转
  await authedPage.getByText('购买订阅').first().click()
  await expect(authedPage).toHaveURL(/#\/plans/)
  await expect(authedPage.getByText('白羊座').first()).toBeVisible()

  // 抽屉菜单
  await authedPage.goto('/#/dashboard')
  // 主题按钮 title 三态:浅色 / 深色 / 跟随系统(默认 system)
  await expect(
    authedPage
      .locator('button[title*="浅色"], button[title*="深色"], button[title*="跟随系统"]')
      .first(),
  ).toBeVisible()
})
