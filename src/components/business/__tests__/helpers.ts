/**
 * 组件测试共享辅助：加载 zh-CN 语言包到全局 i18n 实例。
 * 语言包懒加载（src/locales/index.ts），测试中手动 setLocaleMessage。
 */
import { i18n } from '@/i18n'
import { loadLocaleMessages } from '@/locales'

let loaded = false

export async function useZhCN(): Promise<void> {
  if (loaded) return
  const messages = await loadLocaleMessages('zh-CN')
  i18n.global.setLocaleMessage('zh-CN', messages as never)
  i18n.global.locale.value = 'zh-CN'
  loaded = true
}
