#!/usr/bin/env node
/**
 * 品牌 hex 门禁（ticket 14 验收的常驻化，大小写不敏感）。
 *
 * 背景：tokens.css 是品牌色的唯一字面量源；JS 侧回落值单源于 src/styles/tokenValues.ts；
 * theme.ts（naive-ui 专属第二源，check-tokens-sync 门禁）与 index.html（meta/favicon，
 * 先于 CSS 加载）是仅存的字面量定义点。历史上验收用区分大小写的 grep，大写变体
 * （#6558F5）绕过了门禁——这里统一按 -i 扫描，杜绝大小写规避。
 *
 * 两段断言：
 * 1. tokenValues.ts ↔ tokens.css 同步：同名键大小写不敏感逐值相等，漂移即报错；
 * 2. 品牌 hex 全仓扫描（src/ + uno.config.ts，大小写不敏感）：命中文件必须全部在
 *    定义点白名单内，业务/视图代码出现任何大小写变体即失败。
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join, relative, sep } from 'node:path'

const scriptDir = dirname(fileURLToPath(import.meta.url))
// 仓库根(scripts/ 的上一级),与其他 check-* 脚本的 root 语义一致
const resolve = (...parts) => join(scriptDir, '..', ...parts)

const css = readFileSync(resolve('src/styles/tokens.css'), 'utf8')
const rootBlock = css.match(/:root\s*\{([\s\S]*?)\n\}/)?.[1] ?? ''
if (!rootBlock) {
  console.error('check-brand-hex: 无法解析 tokens.css 的 :root 块')
  process.exit(1)
}

/** 归一化:统一小写(十六进制大小写不敏感) */
const normalize = (v) => v.trim().toLowerCase()

const cssVars = new Map()
for (const m of rootBlock.matchAll(/--([\w-]+)\s*:\s*(#[0-9a-fA-F]{3,8})\s*;/g)) {
  cssVars.set(m[1], normalize(m[2]))
}

// ---- 断言 1:tokenValues.ts ↔ tokens.css 同步 ----
const tv = readFileSync(resolve('src/styles/tokenValues.ts'), 'utf8')
const TV_KEYS = {
  primary: 'c-primary',
  primaryGradEnd: 'c-primary-grad-end',
  olive: 'c-olive',
  danger: 'c-danger',
  pink: 'c-pink',
  warning: 'c-warning',
  blue: 'c-blue',
}
let drift = false
for (const [key, cssName] of Object.entries(TV_KEYS)) {
  const m = tv.match(new RegExp(`${key}:\\s*'(#[0-9a-fA-F]{3,8})'`))
  if (!m) {
    console.error(`check-brand-hex: tokenValues.ts 缺少 ${key}`)
    drift = true
    continue
  }
  const cssVal = cssVars.get(cssName)
  if (!cssVal) {
    console.error(`check-brand-hex: tokens.css 缺少 --${cssName}`)
    drift = true
    continue
  }
  if (normalize(m[1]) !== cssVal) {
    console.error(
      `check-brand-hex: tokenValues.${key}=${m[1]} 与 tokens.css --${cssName}=${cssVal} 漂移`,
    )
    drift = true
  }
}
if (drift) process.exit(1)

// ---- 断言 2:品牌 hex 大小写不敏感全仓扫描 ----
const BRAND_HEX = /#(?:6558f5|8b5cf6|7cfc9c|ffb3d9|f5a524|f7c948)/i
// 仅存的字面量定义点:tokens.css(正源)/theme.ts(naive-ui 第二源)/
// tokenValues.ts(JS 回落正源)/index.html(meta theme-color + favicon,先于 CSS)
const ALLOWLIST = new Set([
  'src/index.html',
  'src/styles/theme.ts',
  'src/styles/tokens.css',
  'src/styles/tokenValues.ts',
])

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) yield* walk(p)
    else yield p
  }
}

const scanned = [...walk(resolve('src')), resolve('uno.config.ts')]
const offenders = []
for (const p of scanned) {
  const rel = relative(resolve('.'), p).split(sep).join('/')
  if (ALLOWLIST.has(rel)) continue
  const text = readFileSync(p, 'utf8')
  const line = text.split('\n').findIndex((l) => BRAND_HEX.test(l))
  if (line >= 0) offenders.push(`${rel}:${line + 1}`)
}

if (offenders.length) {
  console.error('check-brand-hex: 品牌色禁止写死,以下文件出现六种品牌 hex 的大小写变体:')
  for (const o of offenders) console.error(`  ${o}`)
  console.error('改用 var(--c-*) 或从 @/styles/tokenValues 引入 TOKEN_VALUES。')
  process.exit(1)
}
console.log('check-brand-hex: tokenValues 与 tokens.css 一致;品牌 hex 仅存在于定义点。')
