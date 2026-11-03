/**
 * JS 侧设计令牌字面量单源：canvas（SharePanel 海报）/ECharts（图表色）等
 * 读不到 CSS 变量的场景以这里为回落值。
 * 与 tokens.css 同名变量保持一致，漂移由 scripts/check-brand-hex.mjs 的同步断言拦截；
 * theme.ts 为 naive-ui 专属第二源（check-tokens-sync 门禁），本文件不重复其键。
 */
export const TOKEN_VALUES = {
  /** tokens.css --c-primary */
  primary: '#6558f5',
  /** tokens.css --c-primary-grad-end */
  primaryGradEnd: '#8b5cf6',
  /** tokens.css --c-olive */
  olive: '#7c9a3d',
  /** tokens.css --c-danger */
  danger: '#e5484d',
  /** tokens.css --c-pink */
  pink: '#c2487b',
  /** tokens.css --c-warning */
  warning: '#d98e04',
  /** tokens.css --c-blue */
  blue: '#4b8fe5',
} as const
