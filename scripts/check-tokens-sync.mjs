#!/usr/bin/env node
/**
 * tokens.css ↔ theme.ts 双源一致性校验。
 *
 * 背景:naive-ui 主题覆盖(theme.ts)不支持 CSS 变量字符串,必须写真实色值,
 * 与 src/styles/tokens.css 的设计令牌形成双源,历史上靠人工保持同步。
 * 本脚本把两边的映射固化为断言,漂移即报错退出(挂在 pnpm lint 链上)。
 *
 * 映射表(vars → theme.ts 的 GlobalThemeOverrides common 键,亮/暗各一套):
 *   --c-primary      → primaryColor / primaryColorSuppl / infoColor
 *   --c-primary-hover→ primaryColorHover
 *   --c-primary-text → primaryColorPressed(naive 无对应语义键,沿用主色深值)
 *   --c-text         → textColorBase / textColor1 / textColor2
 *   --c-text-sub     → textColor3
 *   --c-bg-card      → cardColor / modalColor / popoverColor / tableColor
 *   --c-bg-hover     → tableHeaderColor
 *   --c-border       → borderColor / dividerColor
 *   --c-success      → successColor
 *   --c-warning      → warningColor
 *   --c-danger       → errorColor
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const css = readFileSync(resolve(root, 'src/styles/tokens.css'), 'utf8')
const ts = readFileSync(resolve(root, 'src/styles/theme.ts'), 'utf8')

/** 从 CSS 文本截取一个块内的变量表(key 统一小写、值去空白) */
function parseVars(block) {
  const map = new Map()
  for (const m of block.matchAll(/--([\w-]+)\s*:\s*([^;]+);/g)) {
    map.set(m[1], m[2].trim().replace(/\s+/g, '').toLowerCase())
  }
  return map
}

const rootBlock = css.match(/:root\s*\{([\s\S]*?)\n\}/)?.[1] ?? ''
const darkBlock = css.match(/\[data-theme='dark'\]\s*\{([\s\S]*?)\n\}/)?.[1] ?? ''
if (!rootBlock || !darkBlock) {
  console.error('check-tokens-sync: 无法解析 tokens.css 的 :root / [data-theme=dark] 块')
  process.exit(1)
}

/** 从 theme.ts 截取一个 override 对象文本,提取 `key: '#xxx'` 键值对(key 保持原样) */
function parseOverrides(source, anchor) {
  const idx = source.indexOf(anchor)
  if (idx < 0) {
    console.error(`check-tokens-sync: theme.ts 中找不到 ${anchor}`)
    process.exit(1)
  }
  const body = source.slice(idx, source.indexOf('\n}', idx))
  const map = new Map()
  for (const m of body.matchAll(/(\w+):\s*'([^']+)'/g)) {
    map.set(m[1], m[2].trim().replace(/\s+/g, '').toLowerCase())
  }
  return map
}

const light = { vars: parseVars(rootBlock), theme: parseOverrides(ts, 'lightThemeOverrides') }
const dark = { vars: parseVars(darkBlock), theme: parseOverrides(ts, 'darkThemeOverrides') }

const MAPPING = {
  'c-primary': ['primaryColor', 'primaryColorSuppl', 'infoColor'],
  'c-primary-hover': ['primaryColorHover'],
  'c-primary-text': ['primaryColorPressed'],
  'c-text': ['textColorBase', 'textColor1', 'textColor2'],
  'c-text-sub': ['textColor3'],
  'c-bg-card': ['cardColor', 'modalColor', 'popoverColor', 'tableColor'],
  'c-bg-hover': ['tableHeaderColor'],
  'c-border': ['borderColor', 'dividerColor'],
  'c-success': ['successColor'],
  'c-warning': ['warningColor'],
  'c-danger': ['errorColor'],
}

const problems = []
for (const [label, { vars, theme }] of [
  ['light', light],
  ['dark', dark],
]) {
  for (const [cssVar, themeKeys] of Object.entries(MAPPING)) {
    const cssValue = vars.get(cssVar)
    if (!cssValue) {
      problems.push(`[${label}] tokens.css 缺少 --${cssVar}`)
      continue
    }
    for (const key of themeKeys) {
      const themeValue = theme.get(key)
      if (!themeValue) {
        problems.push(`[${label}] theme.ts 缺少 ${key}(应与 --${cssVar} 同值)`)
      } else if (themeValue !== cssValue) {
        problems.push(
          `[${label}] --${cssVar}=${cssValue} 与 theme.ts ${key}='${themeValue}' 不一致`,
        )
      }
    }
  }
}

if (problems.length > 0) {
  console.error('check-tokens-sync: 设计令牌漂移,请以 tokens.css 为源修正 theme.ts:')
  for (const p of problems) console.error(`  - ${p}`)
  process.exit(1)
}
console.log('check-tokens-sync: OK(tokens.css 与 theme.ts 双源一致)')
