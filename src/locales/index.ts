import type { Locale } from 'vue-i18n'

export type AppLocale = 'zh-CN' | 'en-US'

export const SUPPORTED_LOCALES: AppLocale[] = ['zh-CN', 'en-US']

/** 语言包懒加载:按需动态 import(文档 frontend/data-layer.md §5) */
export async function loadLocaleMessages(locale: AppLocale): Promise<Record<string, unknown>> {
  const mod = await import(`../locales/${locale}.ts`)
  return (mod.default ?? mod) as Record<string, unknown>
}

/** 浏览器语言 → 支持的 locale */
export function resolveBrowserLocale(): AppLocale {
  const lang = navigator.language ?? 'zh-CN'
  return lang.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
}

export function isSupported(locale: string): locale is AppLocale {
  return (SUPPORTED_LOCALES as string[]).includes(locale)
}

export function normalizeLocale(locale: string | undefined | null): AppLocale {
  if (locale && isSupported(locale)) return locale
  return resolveBrowserLocale()
}

export type { Locale }
