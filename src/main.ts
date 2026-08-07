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
import AppIcon from '@/components/ui/AppIcon.vue'
import StatusBadge from '@/components/ui/StatusBadge.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import CopyText from '@/components/ui/CopyText.vue'
import PriceText from '@/components/ui/PriceText.vue'
import StatNumber from '@/components/ui/StatNumber.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import UiCard from '@/components/ui/UiCard.vue'
import './styles/tokens.css'
import './styles/theme'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

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
  messages: { 'zh-CN': zhCN, 'en-US': enUS },
})

const app = createApp(App)
const pinia = createPinia()
pinia.use(piniaPluginPersistedstate as never)
app.use(pinia)
app.use(router)
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

// 恢复会话(持久化 token)与主题/语言
const auth = useAuthStore()
auth.restore()

const appStore = useAppStore()
appStore.applyTheme()
appStore.initSystemThemeListener()

const browserLang = navigator.language?.startsWith('en') ? 'en-US' : 'zh-CN'
const initialLang = appStore.language || browserLang
i18n.global.locale.value = initialLang as 'zh-CN' | 'en-US'
setHttpLanguage(initialLang)

setupGuards(router)

app.mount('#app')
