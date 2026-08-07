/**
 * 冒烟测试脚本(Playwright):登录 → 仪表板 → 套餐 → 下单 → 订单 → 各页面可达性。
 * 运行:node scripts/smoke.mjs
 */
import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const results = []
let failures = 0

function ok(name, cond, extra = '') {
  results.push(`${cond ? 'PASS' : 'FAIL'}  ${name}${extra ? '  → ' + extra : ''}`)
  if (!cond) failures++
}

const browser = await chromium.launch({ channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
const errors = []
page.on('pageerror', (e) => errors.push('pageerror: ' + e.message))
page.on('console', (m) => {
  if (m.type() === 'error') errors.push('console: ' + m.text())
})

// 1. 登录页渲染(hash 路由)
await page.goto(BASE + '/#/login', { waitUntil: 'networkidle' })
ok('login page title', (await page.title()).includes('NanoCloud'))
ok('login form visible', await page.locator('button:has-text("登录")').first().isVisible())

// 2. 登录(表单已预填 Mock 账号)
await page.locator('button:has-text("登录")').first().click()
await page.waitForURL('**/dashboard', { timeout: 8000 })
ok('redirect to dashboard after login', page.url().includes('/dashboard'))
await page.waitForSelector('text=钱包余额', { timeout: 8000 })
ok('dashboard shows balance', await page.locator('text=钱包余额').first().isVisible())
ok('dashboard shows subscribe card', await page.locator('text=当前订阅').first().isVisible())
ok('dashboard shows notices', await page.locator('text=公告').first().isVisible())
ok('dashboard shows quick actions', await page.locator('text=快捷操作').first().isVisible())

// 3. 购买订阅页
await page.goto(BASE + '/#/plans', { waitUntil: 'networkidle' })
await page.waitForSelector('text=白羊座', { timeout: 8000 })
ok('plans list rendered', await page.locator('text=射手座').first().isVisible())

// 4. 下单弹窗
await page.locator('button:has-text("立即购买")').first().click()
await page.waitForSelector('text=套餐周期', { timeout: 5000 })
ok('order confirm modal opens', await page.locator('text=套餐周期').isVisible())
ok('fee detail visible', await page.locator('text=费用明细').isVisible())
// 提交订单(余额支付,余额充足 → 直接完成,弹 toast 并关闭弹窗)
await page.locator('button:has-text("余额支付")').first().click()
await page.locator('button:has-text("提交订单")').first().click()
await page.waitForTimeout(2500)
ok('balance payment closes modal', !(await page.locator('text=费用明细').first().isVisible().catch(() => false)))

// 5. 订单页
await page.goto(BASE + '/#/orders', { waitUntil: 'networkidle' })
await page.waitForSelector('text=我的订单', { timeout: 8000 })
await page.waitForSelector('table tbody tr', { timeout: 8000 })
ok('orders page renders', await page.locator('text=白羊座').first().isVisible())
ok('order status badge', (await page.getByText('已完成').count()) > 0)

// 6. 其他页面可达性
const pages = [
  ['/docs', '使用文档'],
  ['/invite', '邀请赚钱'],
  ['/agent', '申请代理'],
  ['/nodes', '节点状态'],
  ['/profile', '个人信息'],
  ['/tickets', '我的工单'],
  ['/traffic', '流量明细'],
]
for (const [path, text] of pages) {
  await page.goto(BASE + '/#' + path, { waitUntil: 'networkidle' })
  const found = await page.getByText(text, { exact: false }).first().isVisible().catch(() => false)
  ok(`page ${path} renders "${text}"`, found)
}

// 7. 暗色模式切换(初始 system,点击循环 light→dark→system,最多点 3 次)
await page.goto(BASE + '/#/dashboard', { waitUntil: 'networkidle' })
const themeBtn = page.locator('button[title*="浅色"], button[title*="深色"], button[title*="跟随系统"]').first()
if (await themeBtn.isVisible()) {
  let theme = 'light'
  for (let i = 0; i < 3; i++) {
    await themeBtn.click()
    theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'))
    if (theme === 'dark') break
  }
  ok('dark theme applied', theme === 'dark', `theme=${theme}`)
} else {
  ok('theme toggle visible', false, 'toggle button not found')
}

// 8. 移动端视口冒烟(新 context 无会话,先注入 token)
const mobile = await browser.newPage({ viewport: { width: 390, height: 844 } })
mobile.on('pageerror', (e) => errors.push('mobile pageerror: ' + e.message))
await mobile.goto(BASE + '/#/login', { waitUntil: 'domcontentloaded' })
await mobile.evaluate(() => {
  localStorage.setItem(
    'app:auth',
    JSON.stringify({
      accessToken: 'mock-access-mobile-1',
      refreshToken: 'mock-refresh-mobile-1',
      user: { id: 10086, email: 'mobile@test.com', role: 0 },
    }),
  )
})
await mobile.goto(BASE + '/#/dashboard', { waitUntil: 'networkidle' })
await mobile.waitForSelector('text=钱包余额', { timeout: 8000 })
ok('mobile dashboard renders', await mobile.locator('text=钱包余额').first().isVisible())
ok('mobile tabbar visible', await mobile.locator('text=购买订阅').first().isVisible())
// 底栏跳转
await mobile.locator('text=购买订阅').first().click()
await mobile.waitForURL('**/plans', { timeout: 5000 })
ok('mobile tab navigates to plans', mobile.url().includes('/plans'))

console.log('\n==== SMOKE RESULTS ====')
results.forEach((r) => console.log(r))
console.log(`\n${results.length - failures}/${results.length} passed`)
if (errors.length) {
  console.log('\nBrowser errors captured:')
  errors.slice(0, 10).forEach((e) => console.log('  ' + e))
}
await browser.close()
process.exit(failures > 0 || errors.length > 0 ? 1 : 0)
