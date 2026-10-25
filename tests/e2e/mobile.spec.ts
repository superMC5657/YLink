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

test('顶栏响应式分支随视口宽度切换(汉堡钮 ↔ 折叠钮)', async ({ authedPage }) => {
  await authedPage.goto('/#/dashboard')
  await expect(authedPage.getByText('钱包余额')).toBeVisible()
  // 起始视口 390(<768):移动分支,顶栏汉堡菜单图标(AppIcon menu 三横 path)
  const hamburger = authedPage.locator('header path[d="M3 12h18M3 6h18M3 18h18"]')
  await expect(hamburger).toBeVisible()

  // 拉宽到桌面(≥768):切换为桌面折叠钮,汉堡图标消失
  // (冻结判定 window.innerWidth 在此残留,useMediaQuery 才会随宽度切换)
  await authedPage.setViewportSize({ width: 1280, height: 800 })
  await expect(hamburger).toHaveCount(0)

  // 再缩回移动:汉堡钮回来(双向响应)
  await authedPage.setViewportSize({ width: 390, height: 844 })
  await expect(hamburger).toBeVisible()
})
