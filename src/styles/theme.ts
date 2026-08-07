import type { GlobalThemeOverrides } from 'naive-ui'

/**
 * Naive UI 主题覆盖。
 * 注意:naive-ui 内部用 seemly 解析颜色派生(如 rgba()),**不支持 CSS 变量字符串**,
 * 因此亮/暗各提供一套真实色值,与 src/styles/tokens.css 保持一致(双源同步)。
 */

const base: GlobalThemeOverrides = {
  common: {
    borderRadius: '10px',
    borderRadiusSmall: '8px',
    fontFamily: 'Inter, "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
  Button: {
    borderRadiusMedium: '10px',
    borderRadiusLarge: '10px',
  },
  Card: {
    borderRadius: '16px',
  },
  Modal: {
    borderRadius: '16px',
  },
  Input: {
    borderRadius: '10px',
  },
  Pagination: {
    borderRadius: '999px',
    itemColorActive: '#6558F5',
    itemTextColorActive: '#fff',
  },
  Tag: {
    borderRadius: '999px',
  },
  Switch: {
    railColorActive: '#6558F5',
  },
  Message: {
    borderRadius: '10px',
  },
  Dialog: {
    borderRadius: '16px',
  },
}

/** 浅色主题覆盖(与 tokens.css :root 同步) */
export const lightThemeOverrides: GlobalThemeOverrides = {
  ...base,
  common: {
    ...base.common,
    primaryColor: '#6558F5',
    primaryColorHover: '#7A6EF7',
    primaryColorPressed: '#4B3ED6',
    primaryColorSuppl: '#6558F5',
    textColorBase: '#1F2430',
    textColor1: '#1F2430',
    textColor2: '#1F2430',
    textColor3: '#8A8FA3',
    bodyColor: '#F5F6FB',
    cardColor: '#FFFFFF',
    modalColor: '#FFFFFF',
    popoverColor: '#FFFFFF',
    tableColor: '#FFFFFF',
    tableHeaderColor: '#F2F3FA',
    borderColor: '#EBECF4',
    dividerColor: '#EBECF4',
    successColor: '#5BA829',
    warningColor: '#D98E04',
    errorColor: '#E5484D',
    infoColor: '#6558F5',
  },
}

/** 暗色主题覆盖(与 tokens.css [data-theme='dark'] 同步) */
export const darkThemeOverrides: GlobalThemeOverrides = {
  ...base,
  common: {
    ...base.common,
    primaryColor: '#7C72FF',
    primaryColorHover: '#8F86FF',
    primaryColorPressed: '#A79DFF',
    primaryColorSuppl: '#7C72FF',
    textColorBase: '#E8EAF2',
    textColor1: '#E8EAF2',
    textColor2: '#E8EAF2',
    textColor3: '#9BA1B7',
    bodyColor: '#0F1117',
    cardColor: '#171B26',
    modalColor: '#171B26',
    popoverColor: '#171B26',
    tableColor: '#171B26',
    tableHeaderColor: '#1E2331',
    borderColor: 'rgba(255,255,255,0.07)',
    dividerColor: 'rgba(255,255,255,0.07)',
    successColor: '#7BD88F',
    warningColor: '#F0B24A',
    errorColor: '#F16A6E',
    infoColor: '#7C72FF',
  },
}
