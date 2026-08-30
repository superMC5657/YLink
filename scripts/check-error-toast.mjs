#!/usr/bin/env node
/**
 * 错误 toast 单一出口守护(2026-08-30,经全局 25 处双 toast 修复沉淀)
 * 约定文档:docs/frontend/data-layer.md §7「错误处理与用户反馈 · 调用方约定」
 *
 * 背景:http 层(utils/http.ts)对业务错误默认统一 toast(未传 silent 时,经
 * ToastBridge 注入的 naive message 弹出)。组件 catch 里再手动
 * message.error(错误对象.message) 会让同一错误弹两次 —— 2026-08-30 前该
 * 反模式在全项目累积 25 处(详见 docs/frontend/progress.md 同日小节)。
 *
 * 本脚本扫描并拦截该反模式,已串入 `pnpm lint`(本地与 CI 同时覆盖):
 *   禁止  message.error((e as Error).message) / message.error(e.message)
 *         / message.error(err.message) 等转发异常对象 message 的调用;
 *   放行  message.error(t('…')) —— 本地错误(剪贴板/canvas/前端校验等,
 *         非 http 层管辖)用 i18n 文案自行提示是合法做法。
 *
 * 例外文件:ToastBridge.vue(toast provider 注入本体)。
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

const ROOT = join(process.cwd(), 'src')
const EXCLUDE = new Set([join(ROOT, 'components', 'app', 'ToastBridge.vue')])
const EXTS = new Set(['.vue', '.ts'])
// message.error( ((xx) as Error | e | err ...).message ) —— 转发异常对象 message
const PATTERN = /\bmessage\.error\(\s*(?:\(\s*\w+\s+as\s+Error\s*\)|\w+)\s*\.\s*message\b/

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) {
      if (name === '__tests__' || name === 'node_modules') continue
      yield* walk(p)
    } else if (EXTS.has(name.slice(name.lastIndexOf('.'))) && !EXCLUDE.has(p)) {
      yield p
    }
  }
}

const hits = []
for (const file of walk(ROOT)) {
  const lines = readFileSync(file, 'utf8').split('\n')
  lines.forEach((line, i) => {
    if (PATTERN.test(line)) {
      hits.push(`${relative(process.cwd(), file).split(sep).join('/')}:${i + 1}: ${line.trim()}`)
    }
  })
}

if (hits.length > 0) {
  console.error(
    [
      '',
      '✖ 错误 toast 双弹反模式(组件层转发 http 错误对象的 message):',
      ...hits.map((h) => `  ${h}`),
      '',
      '  http 层已对业务错误统一 toast(未传 silent 时),catch 里禁止再弹,',
      '  只做状态恢复/流程控制;确需自定义提示请给请求传 silent: true。',
      "  本地错误(非 http)请用 message.error(t('…')) i18n 文案。",
      '  约定详见 docs/frontend/data-layer.md §7。',
      '',
    ].join('\n'),
  )
  process.exit(1)
}

console.log('check-error-toast: OK(http 层错误 toast 单一出口,无组件层重复弹提示)')
