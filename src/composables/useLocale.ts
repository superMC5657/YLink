/**
 * 语言切换组合式函数:懒加载语言包 + 注册到全局 i18n + 更新 locale + 同步 http Accept-Language + 持久化。
 */
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { setHttpLanguage } from '@/utils/http'
import { loadLocaleMessages, type AppLocale } from '@/locales'

let switching: Promise<void> | null = null

export function useLocale() {
  const { locale, setLocaleMessage } = useI18n()
  const app = useAppStore()

  async function switchLocale(lang: string): Promise<void> {
    if (switching) return switching
    switching = (async () => {
      const target = lang as AppLocale
      const messages = await loadLocaleMessages(target)
      setLocaleMessage(target, messages)
      locale.value = target
      app.setLanguage(target)
      setHttpLanguage(target)
    })().finally(() => {
      switching = null
    })
    return switching
  }

  return { switchLocale }
}
