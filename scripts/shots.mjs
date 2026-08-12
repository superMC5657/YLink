// 截取关键页面供人工确认(保存到 .ui-shots/)
import { mkdirSync } from 'node:fs'
import { chromium } from 'playwright'

const BASE = 'http://localhost:5174'
const OUT = '.ui-shots'
mkdirSync(OUT, { recursive: true })

const browser = await chromium.launch({ channel: 'chrome' })

async function shot(page, route, name, width, height) {
  await page.goto(BASE + route, { waitUntil: 'networkidle' })
  await page.waitForTimeout(700)
  await page.screenshot({ path: `${OUT}/${name}.png` })
  console.log(`saved ${OUT}/${name}.png`)
}

// 桌面
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
await page.goto(BASE + '/#/login', { waitUntil: 'domcontentloaded' })
await page.evaluate(() => {
  localStorage.setItem(
    'app:auth',
    JSON.stringify({
      accessToken: 'mock-access-shot',
      refreshToken: 'r',
      user: { id: 1, email: 'a@b.com', role: 0 },
    }),
  )
})
await page.reload({ waitUntil: 'domcontentloaded' })
for (const [route, name] of [
  ['/#/dashboard', 'dashboard'],
  ['/#/plans', 'plans'],
  ['/#/orders', 'orders'],
  ['/#/invite', 'invite'],
  ['/#/profile', 'profile'],
  ['/#/traffic', 'traffic'],
]) {
  await shot(page, route, name, 1440, 900)
}
await page.close()

// 移动端
const mob = await browser.newPage({ viewport: { width: 390, height: 844 } })
await mob.goto(BASE + '/#/login', { waitUntil: 'domcontentloaded' })
await mob.evaluate(() => {
  localStorage.setItem(
    'app:auth',
    JSON.stringify({
      accessToken: 'mock-access-shot',
      refreshToken: 'r',
      user: { id: 1, email: 'a@b.com', role: 0 },
    }),
  )
})
await mob.reload({ waitUntil: 'domcontentloaded' })
await shot(mob, '/#/dashboard', 'mobile-dashboard', 390, 844)
await mob.close()

await browser.close()
