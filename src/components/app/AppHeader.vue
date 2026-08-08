<script setup lang="ts">
/**
 * 顶栏:毛玻璃吸顶。桌面含折叠钮/站点名/主题/语言/用户;移动端含抽屉钮/站点名/主题/头像。
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useLocale } from '@/composables/useLocale'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'

const app = useAppStore()
const auth = useAuthStore()
const config = useConfigStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t, locale } = useI18n()
const { switchLocale } = useLocale()
const emit = defineEmits<{ (e: 'toggle-drawer'): void }>()

const isMobile = computed(() => window.innerWidth < 768)
const userEmail = computed(() => auth.user?.email ?? '')

const languageOptions = computed(() =>
  (config.config?.languages ?? ['zh-CN', 'en-US']).map((lang) => ({
    label: lang === 'zh-CN' ? '简体中文' : 'English',
    value: lang,
  })),
)

function onLanguageChange(value: string) {
  void switchLocale(value)
}

function goProfile() {
  router.push('/profile')
}

function onLogout() {
  dialog.warning({
    title: '退出登录',
    content: '确定要退出当前账号吗?',
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await auth.logout()
      message.success(t('common.logout'))
      router.push('/login')
    },
  })
}
</script>

<template>
  <header
    class="sticky top-0 z-20 flex h-16 shrink-0 items-center gap-3 border-b border-[var(--c-border)] px-4 backdrop-blur-12 md:px-6"
    style="background: color-mix(in srgb, var(--c-bg-card) 82%, transparent)"
  >
    <!-- 移动端:抽屉钮 -->
    <button
      v-if="isMobile"
      class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-[var(--c-text)] transition-colors hover:bg-[var(--c-bg-hover)]"
      @click="emit('toggle-drawer')"
    >
      <AppIcon name="menu" :size="22" />
    </button>

    <!-- 桌面:折叠钮 -->
    <button
      v-else
      class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
      @click="app.toggleSidebar"
    >
      <AppIcon :name="app.sidebarCollapsed ? 'chevron-right' : 'chevron-down'" :size="20" />
    </button>

    <!-- 站点名(移动端) -->
    <span v-if="isMobile" class="text-16 font-700 text-[var(--c-text)]">{{ config.siteName }}</span>
    <span v-else class="ml-1 flex-1 text-13 text-[var(--c-text-sub)]">
      {{ route.meta.title ? t(String(route.meta.title)) : '' }}
    </span>

    <div class="flex items-center gap-2">
      <!-- 语言 -->
      <n-dropdown trigger="click" :options="languageOptions" @select="onLanguageChange">
        <button
          class="flex h-9 cursor-pointer items-center gap-1 rounded-[var(--r-pill)] px-3 text-13 text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
        >
          <AppIcon name="globe" :size="16" />
          <span class="hidden sm:inline">{{
            languageOptions.find((o) => o.value === locale)?.label
          }}</span>
        </button>
      </n-dropdown>

      <!-- 主题 -->
      <ThemeToggle />

      <!-- 用户 -->
      <n-dropdown
        trigger="click"
        :options="[
          { label: '个人信息', key: 'profile' },
          { type: 'divider', key: 'd1' },
          { label: '退出登录', key: 'logout' },
        ]"
        @select="(k: string) => (k === 'logout' ? onLogout() : goProfile())"
      >
        <button
          class="flex cursor-pointer items-center gap-2 rounded-[var(--r-pill)] px-2 py-1 transition-colors hover:bg-[var(--c-bg-hover)]"
        >
          <span
            class="flex h-8 w-8 items-center justify-center rounded-full text-white"
            style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          >
            <AppIcon name="user" :size="17" />
          </span>
          <span class="hidden max-w-40 truncate text-13 text-[var(--c-text)] lg:inline">{{
            userEmail
          }}</span>
        </button>
      </n-dropdown>
    </div>
  </header>
</template>
