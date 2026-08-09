/**
 * 桌面分辨率适配诊断 v2:注入 Mock 会话后,多分辨率下测量
 * 1) 页面是否横向溢出
 * 2) 内容区实际宽度 vs 视口(宽屏利用率)
 * 3) 关键组件宽度(表格/卡片/宫格)
 */
import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const RESOLUTIONS = [
  { name: '1024x768', width: 1024, height: 768 },
  { name: '1280x800', width: 1280, height: 800 },
  { name: '1440x900', width: 1440, height: 900 },
  { name: '1920x1080', width: 1920, height: 1080 },
  { name: '2560x1440', width: 2560, height: 1440 },
]

const ROUTES = ['/#/dashboard', '/#/plans', '/#/orders', '/#/invite', '/#/agent', '/#/nodes', '/#/profile', '/#/tickets', '/#/traffic', '/#/docs', '/#/tickets/7', '/#/docs/31']

const browser = await chromium.launch({ channel: 'chrome' })

async function injectAuth(page) {
  await page.goto(BASE + '/#/login', { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => {
    localStorage.setItem(
      'app:auth',
      JSON.stringify({
        accessToken: 'mock-access-diag-' + Date.now(),
        refreshToken: 'mock-refresh-diag',
        user: { id: 10086, email: 'diag@test.com', role: 0 },
      }),
    )
  })
  await page.reload({ waitUntil: 'domcontentloaded' })
}

async function measure(page, route) {
  await page.goto(BASE + route, { waitUntil: 'networkidle' })
  await page.waitForTimeout(700)
  return page.evaluate(() => {
    const doc = document.documentElement
    const main = document.querySelector('main')
    const content = main?.firstElementChild ?? null
    const sidebar = document.querySelector('aside')
    const overflowEls = []
    const isClipped = (el) => {
      // 存在 overflow-hidden 祖先的装饰性元素视觉上已被裁剪,不视为溢出
      let p = el.parentElement
      while (p) {
        const cs = getComputedStyle(p)
        if (cs.overflowX === 'hidden' || cs.overflow === 'hidden') return true
        p = p.parentElement
      }
      return false
    }
    document.querySelectorAll('body *').forEach((el) => {
      if (!(el instanceof HTMLElement)) return
      const r = el.getBoundingClientRect()
      if (r.width <= 0) return
      if (r.right > doc.clientWidth + 2 || r.left < -2) {
        const cs = getComputedStyle(el)
        if (cs.position === 'fixed') return
        if (el.closest('.n-modal') || el.closest('.n-drawer')) return
        if (isClipped(el)) return
        overflowEls.push({
          tag: el.tagName.toLowerCase(),
          cls: (el.className || '').toString().slice(0, 50),
          right: Math.round(r.right),
          left: Math.round(r.left),
        })
      }
    })
    return {
      viewport: doc.clientWidth,
      scrollWidth: doc.scrollWidth,
      horizontalOverflow: doc.scrollWidth > doc.clientWidth,
      mainWidth: main ? Math.round(main.getBoundingClientRect().width) : 0,
      contentWidth: content ? Math.round(content.getBoundingClientRect().width) : 0,
      contentMaxWidth: content ? Math.round(parseFloat(getComputedStyle(content).maxWidth) || 0) : 0,
      sidebarWidth: sidebar ? Math.round(sidebar.getBoundingClientRect().width) : 0,
      overflowEls: overflowEls.slice(0, 6),
      tables: Array.from(document.querySelectorAll('table')).map((t) => {
        const r = t.getBoundingClientRect()
        const wrap = t.parentElement ?? t
        const wrapR = wrap.getBoundingClientRect()
        const wrapCls = (wrap.className || '').toString()
        return {
          w: Math.round(r.width),
          parentW: Math.round(wrapR.width),
          inScrollWrap: wrapR.width < r.width && wrapCls.includes('overflow-x-auto'),
        }
      }),
      grids: Array.from(document.querySelectorAll('[class*="grid-cols-"]')).map((g) => {
        const r = g.getBoundingClientRect()
        return { w: Math.round(r.width), cls: (g.className || '').toString().slice(0, 45) }
      }),
    }
  })
}

let failed = 0
for (const res of RESOLUTIONS) {
  const page = await browser.newPage({ viewport: { width: res.width, height: res.height } })
  await injectAuth(page)
  console.log(`\n===== ${res.name} =====`)
  for (const route of ROUTES) {
    try {
      const m = await measure(page, route)
      const issues = []
      if (m.horizontalOverflow)
        issues.push(`⚠横向溢出 scrollW=${m.scrollWidth} viewport=${m.viewport}`)
      if (m.viewport >= 1440 && m.contentWidth < Math.max(960, m.viewport * 0.55))
        issues.push(`内容窄 ${m.contentWidth}px/${m.viewport}px`)
      m.tables.forEach((t, i) => {
        // 表格在 overflow-x-auto 容器内滚动属设计内行为,不视为页面级溢出
        if (t.w > t.parentW + 2 && !t.inScrollWrap) issues.push(`表格#${i}溢出 ${t.w}>${t.parentW}`)
      })
      m.overflowEls.slice(0, 2).forEach((o) =>
        issues.push(`溢出<${o.tag}> ${o.cls} right=${o.right}`),
      )
      const status = issues.length ? issues.join(' | ') : '✓'
      console.log(`  ${route.padEnd(14)} 内容=${String(m.contentWidth).padStart(5)}px ${status}`)
      if (issues.length) failed++
    } catch (e) {
      console.log(`  ${route} ERR ${String(e.message).slice(0, 60)}`)
    }
  }
  await page.close()
}

console.log(`\n===== 异常计数: ${failed} =====`)
await browser.close()
