/**
 * E2E 前置守卫:确认 5174 上的 dev server 是 e2e mock 模式。
 * 背景:本地 reuseExistingServer 为 true,若本机已有非 e2e 模式 pnpm dev
 * 占用 5174,E2E 会复用错 server(Mock 关闭,登录直挂)。此处以公开的
 * GET /api/v1/config envelope 为探针,形态不对即 fail-fast,提示停掉本机 dev。
 * 生产代码零影响,仅起测前跑一次;CI(不复用,自起 --mode e2e)恒过。
 */
async function globalSetup(): Promise<void> {
  const base = 'http://localhost:5174'
  const deadline = Date.now() + 10_000
  let last: string
  for (;;) {
    try {
      const res = await fetch(`${base}/api/v1/config`)
      const json = (await res.json()) as { code?: number; data?: { site_name?: string } }
      if (res.ok && json?.code === 0 && json?.data?.site_name) return
      last = `HTTP ${res.status} body=${JSON.stringify(json).slice(0, 120)}`
    } catch (e) {
      last = String(e).slice(0, 120)
    }
    if (Date.now() > deadline) break
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(
    '[E2E] 5174 上的 dev server 非 e2e mock 模式(' +
      last +
      ')。请先停掉本机 pnpm dev,再重跑 E2E（E2E 会以 pnpm dev --mode e2e 自起 mock server；勿手动起非 e2e dev 占用 5174）。',
  )
}

export default globalSetup
