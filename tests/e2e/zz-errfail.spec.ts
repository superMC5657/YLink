import { test } from '@playwright/test'

test('复现 ERR_FAILED:真实浏览器登录并抓请求失败原因', async ({ page }) => {
  page.on('console', (msg) => {
    if (/ERR_|CORS|网络异常|failed/i.test(msg.text())) {
      console.log('[console]', msg.type(), msg.text().slice(0, 300))
    }
  })
  page.on('requestfailed', (r) => {
    console.log('[requestfailed]', r.method(), r.url(), '=>', r.failure()?.errorText)
  })
  page.on('request', (r) => {
    if (r.url().includes('8080')) console.log('[request]', r.method(), r.url())
  })
  page.on('response', (r) => {
    if (r.url().includes('8080')) console.log('[response]', r.status(), r.url())
  })

  await page.goto('http://localhost:5173/#/login', { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)
  await page.getByPlaceholder('邮箱').first().fill('demo@test.com')
  await page.getByPlaceholder('密码').first().fill('Passw0rd')
  await page.getByRole('button', { name: /登\s*录/ }).first().click()
  await page.waitForTimeout(5000)
  console.log('[final url]', page.url())
})
