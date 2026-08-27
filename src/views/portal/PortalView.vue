<script setup lang="ts">
/**
 * 门户分流页:管理员登录后的二选一入口(用户中心/管理后台)。
 * 独立全屏页,不带任何侧边菜单;仅管理员可进(meta.admin 兜底,见 guards.ts)。
 * 视觉复用 AuthLayout 风格(氛围背景 + 居中卡片)。规范见 docs/frontend/pages.md §2.5。
 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'
import LanguageToggle from '@/components/ui/LanguageToggle.vue'

const auth = useAuthStore()
const config = useConfigStore()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

void config.fetchConfig()

/** 两张分流卡片:用户中心 → /dashboard,管理后台 → /admin/overview */
const entries = computed(() => [
  { title: t('portal.userCenter'), icon: 'home', path: '/dashboard' },
  { title: t('portal.adminConsole'), icon: 'sliders', path: '/admin/overview' },
])

function onLogout() {
  dialog.warning({
    title: t('common.logout'),
    content: t('common.logoutConfirm'),
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
  <div
    class="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-4 py-8"
    style="background-color: var(--c-bg-app)"
  >
    <!-- 氛围背景(与 AuthLayout 一致) -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -top-32 -left-32 h-96 w-96 rounded-full opacity-25 blur-3xl"
        style="background: radial-gradient(circle, #6558f5 0%, transparent 70%)"
      />
      <div
        class="absolute -right-32 -bottom-32 h-96 w-96 rounded-full opacity-20 blur-3xl"
        style="background: radial-gradient(circle, #8b5cf6 0%, transparent 70%)"
      />
    </div>

    <!-- 右上角语言/主题 -->
    <div class="absolute top-4 right-4 z-10 flex items-center gap-2">
      <LanguageToggle />
      <ThemeToggle />
    </div>

    <div class="relative z-1 w-full max-w-160">
      <!-- 品牌区 -->
      <div class="mb-8 flex flex-col items-center gap-2">
        <span
          class="flex h-14 w-14 items-center justify-center rounded-2xl text-white shadow-lg"
          style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
        >
          <AppIcon name="zap" :size="28" />
        </span>
        <h1 class="text-22 font-700 text-[var(--c-text)]">{{ config.siteName }}</h1>
        <p class="text-14 text-[var(--c-text-sub)]">
          {{ t('portal.welcome') }} · {{ auth.user?.email }}
        </p>
      </div>

      <!-- 分流双卡片 -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <button
          v-for="entry in entries"
          :key="entry.path"
          class="card-base flex cursor-pointer flex-col items-center gap-3 border border-solid border-[var(--c-card-border)] px-6 py-8 transition-all duration-[var(--t-fast)] hover:border-[var(--c-primary)] hover:shadow-lg"
          style="--s-card: var(--s-pop)"
          @click="router.push(entry.path)"
        >
          <span
            class="flex h-12 w-12 items-center justify-center rounded-2xl text-white"
            style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          >
            <AppIcon :name="entry.icon" :size="24" />
          </span>
          <span class="text-16 font-700 text-[var(--c-text)]">{{ entry.title }}</span>
        </button>
      </div>

      <!-- 退出登录 -->
      <div class="mt-6 flex justify-center">
        <button
          class="flex h-10 cursor-pointer items-center gap-2 rounded-[var(--r-control)] px-4 text-14 text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)] hover:text-[var(--c-danger)]"
          @click="onLogout"
        >
          <AppIcon name="log-out" :size="16" />
          <span>{{ t('common.logout') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
