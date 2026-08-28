/**
 * 品牌配置运行时应用（F19 品牌配置子集）。
 * 管理端在 site 配置中设置 primary_color / background_url,
 * /config 下发后在此统一应用:CSS 设计令牌覆盖(tokens.css 变量)+ Naive 主题主色。
 * 校验失败(非法颜色/空值)回退默认主题,不影响渲染。
 */

/** 校验 Hex 颜色(#RGB/#RRGGBB/#RRGGBBAA) */
export function isHexColor(value: string | undefined | null): value is string {
  return !!value && /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/.test(value)
}

function parseHex(hex: string): [number, number, number] {
  let h = hex.slice(1)
  if (h.length === 3)
    h = h
      .split('')
      .map((c) => c + c)
      .join('')
  if (h.length === 8) h = h.slice(0, 6)
  const n = parseInt(h, 16)
  return [(n >> 16) & 0xff, (n >> 8) & 0xff, n & 0xff]
}

function toHex(r: number, g: number, b: number): string {
  const c = (v: number) =>
    Math.round(Math.min(255, Math.max(0, v)))
      .toString(16)
      .padStart(2, '0')
  return `#${c(r)}${c(g)}${c(b)}`
}

/** 颜色线性插值:ratio=0 返回 a,ratio=1 返回 b */
export function mixHex(a: string, b: string, ratio: number): string {
  const [r1, g1, b1] = parseHex(a)
  const [r2, g2, b2] = parseHex(b)
  return toHex(r1 + (r2 - r1) * ratio, g1 + (g2 - g1) * ratio, b1 + (b2 - b1) * ratio)
}

/** CSS 设计令牌(与 tokens.css :root / [data-theme=dark] 变量同名) */
const PRIMARY_VARS = [
  '--c-primary',
  '--c-primary-hover',
  '--c-primary-soft',
  '--c-primary-text',
  '--c-primary-grad-end',
] as const

/**
 * 应用品牌主色到 CSS 变量。主色为空或非法时清除覆盖(回退 tokens.css 默认)。
 * soft/grad 派生用 color-mix 前的 JS 混色(与 Naive 主题共用同一组派生色,保持一致)。
 */
export function applyPrimaryColor(color: string | undefined | null): void {
  const el = document.documentElement
  if (!isHexColor(color)) {
    PRIMARY_VARS.forEach((k) => el.style.removeProperty(k))
    return
  }
  const soft =
    el.dataset.theme === 'dark' ? mixHex(color, '#7c72ff', 0.16) : mixHex(color, '#ffffff', 0.86)
  el.style.setProperty('--c-primary', color)
  el.style.setProperty('--c-primary-hover', mixHex(color, '#ffffff', 0.14))
  el.style.setProperty('--c-primary-soft', soft)
  el.style.setProperty('--c-primary-text', mixHex(color, '#000000', 0.2))
  el.style.setProperty('--c-primary-grad-end', mixHex(color, '#8b5cf6', 0.3))
}

/** 应用品牌背景图(body 层,空值清除) */
export function applyBackgroundImage(url: string | undefined | null): void {
  const body = document.body
  if (!url || (!/^(https?:)?\/\//i.test(url) && !url.startsWith('/'))) {
    body.style.backgroundImage = ''
    body.style.backgroundSize = ''
    body.style.backgroundAttachment = ''
    body.style.backgroundPosition = ''
    return
  }
  body.style.backgroundImage = `url(${url})`
  body.style.backgroundSize = 'cover'
  body.style.backgroundAttachment = 'fixed'
  body.style.backgroundPosition = 'center'
}
