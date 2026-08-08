import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import { createI18n } from 'vue-i18n'
import {
  create,
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NPagination,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSpin,
  NSwitch,
  NTable,
  NDatePicker,
} from 'naive-ui'
import 'uno.css'

import App from './App.vue'
import router from './router'
import { setupGuards } from './router/guards'
import { useAppStore } from './stores/app'
import { useAuthStore } from './stores/auth'
import { setHttpLanguage } from './utils/http'
import { onDeepLink } from './utils/platform'
import AppIcon from '@/components/ui/AppIcon.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import CopyText from '@/components/ui/CopyText.vue'
import PriceText from '@/components/ui/PriceText.vue'
import StatNumber from '@/components/ui/StatNumber.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import UiCard from '@/components/ui/UiCard.vue'
import { loadLocaleMessages, normalizeLocale } from '@/locales'
import './styles/tokens.css'
import './styles/theme'

/** naive-ui 按需注册(与 create({}) 默认空注册不同,必须显式列出) */
const naive = create({
  components: [
    NConfigProvider,
    NMessageProvider,
    NDialogProvider,
    NDrawer,
    NDrawerContent,
    NDropdown,
    NForm,
    NFormItem,
    NInput,
    NInputNumber,
    NModal,
    NPagination,
    NRadioButton,
    NRadioGroup,
    NSelect,
    NSpin,
    NSwitch,
    NTable,
    NDatePicker,
  ],
})

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {},
})

const app = createApp(App)
const pinia = createPinia()
pinia.use(piniaPluginPersistedstate as never)
app.use(pinia)
app.use(i18n)
app.use(naive)

// 全局注册基础 UI 组件(业务组件大量直接使用而未局部 import)
app.component('AppIcon', AppIcon)
app.component('StatusBadge', StatusBadge)
app.component('EmptyState', EmptyState)
app.component('CopyText', CopyText)
app.component('PriceText', PriceText)
app.component('StatNumber', StatNumber)
app.component('PageHeader', PageHeader)
app.component('UiCard', UiCard)

/**
 * 异步引导:语言包懒加载完成后再挂载,避免首帧缺文案。
 * 语言解析:持久化值 → 浏览器语言 → zh-CN。
 * 注意:router 在此安装,确保守卫先于初始导航注册。
 */
async function bootstrap() {
  const auth = useAuthStore()
  auth.restore()

  const appStore = useAppStore()
  appStore.applyTheme()
  appStore.initSystemThemeListener()

  const initialLang = normalizeLocale(appStore.language)
  const messages = await loadLocaleMessages(initialLang)
  i18n.global.setLocaleMessage(initialLang, messages)
  i18n.global.locale.value = initialLang
  setHttpLanguage(initialLang)

  setupGuards(router)
  app.use(router)

  // Tauri 深链接:nanocloud://plans → 路由 /plans(desktop-tauri.md §4)
  onDeepLink((url) => {
    try {
      const path = new URL(url).pathname
      if (path && path !== '/') void router.push(path)
    } catch {
      void router.push(url.replace(/^[a-z]+:\/\//, '/'))
    }
  })

  app.mount('#app')
}

void bootstrap()
