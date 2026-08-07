/**
 * 字体比例诊断:测量各页面实际渲染的字号分布与关键文本字号。
 * 输出每个页面:字号分布(px → 计数)+ 超范围字号清单。
 */
import { chromium } from 'playwright'

const BASE = 'http://localhost:5174'
const ROUTES = ['/#/dashboard', '/#/plans', '/#/orders', '/#/invite', '/#/agent', '/#/nodes', '/#/profile', '/#/tickets', '/#/traffic', '/#/docs']

const browser = await chromium.launch({ channel: 'chrome' })

async function injectAuth(page) {
  await page.goto(BASE + '/#/login', { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => {
    localStorage.setItem(
      'app:auth',
      JSON.stringify({ accessToken: 'mock-access-font-' + Date.now(), refreshToken: 'r', user: { id: 1, email: 'a@b.com', role: 0 } }),
    )
  })
  await page.reload({ waitUntil: 'domcontentloaded' })
}

async function measure(page, route) {
  await page.goto(BASE + route, { waitUntil: 'networkidle' })
  await page.waitForTimeout(700)
  return page.evaluate(() => {
    const counts = new Map()
    const samples = [] // 记录每条文本:字号 + 文本前12字 + 类名
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT)
    let node
    while ((node = walker.nextNode())) {
      const text = (node.textContent || '').trim()
      if (!text || text.length < 2) continue
      const el = node.parentElement
      if (!el) continue
      if (el.closest('script, style, svg, .n-modal, .n-drawer, [class*="absolute"]')) continue
      const fs = parseFloat(getComputedStyle(el).fontSize)
      const rounded = Math.round(fs)
      counts.set(rounded, (counts.get(rounded) ?? 0) + 1)
      if (samples.length < 400) {
        samples.push({ fs: rounded, text: text.slice(0, 14), cls: (el.className || '').toString().slice(0, 30) })
      }
    }
    return {
      dist: Object.fromEntries([...counts.entries()].sort((a, b) => a[0] - b[0])),
      samples,
    }
  })
}

for (const route of ROUTES) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  await injectAuth(page)
  const m = await measure(page, route)
  console.log(`\n===== ${route} =====`)
  console.log('字号分布:', JSON.stringify(m.dist))
  // 显示几个有代表性的:最小/最大/最常见的 5 条样本
  const sorted = m.samples.slice().sort((a, b) => a.fs - b.fs)
  console.log('最小字号样本:')
  sorted.slice(0, 6).forEach((s) => console.log(`  ${s.fs}px [${s.cls}] ${s.text}`))
  console.log('最大字号样本:')
  sorted.slice(-6).forEach((s) => console.log(`  ${s.fs}px [${s.cls}] ${s.text}`))
  await page.close()
}

await browser.close()
