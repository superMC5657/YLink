<script setup lang="ts">
/**
 * 认证布局:居中卡片(420px)+ 品牌氛围背景;右上语言/主题切换。
 * 规范见 docs/frontend/pages.md §2.3。
 */
import { useConfigStore } from '@/stores/config'
import { useAppStore } from '@/stores/app'
import { setHttpLanguage } from '@/utils/http'
import { useI18n } from 'vue-i18n'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'

const config = useConfigStore()
const app = useAppStore()
const { locale } = useI18n()

void config.fetchConfig()

function onLanguageChange(value: string) {
  locale.value = value
  setHttpLanguage(value)
  app.setLanguage(value)
}

const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' },
]
</script>

<template>
  <div
    class="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-8"
    style="background-color: var(--c-bg-app)"
  >
    <!-- 氛围背景 -->
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
      <n-dropdown trigger="click" :options="languageOptions" @select="onLanguageChange">
        <button
          class="flex h-9 cursor-pointer items-center gap-1 rounded-full border border-[var(--c-border)] bg-[var(--c-bg-card)] px-3 text-13 text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
        >
          <AppIcon name="globe" :size="16" />
          <span>{{ languageOptions.find((o) => o.value === locale)?.label }}</span>
        </button>
      </n-dropdown>
      <ThemeToggle />
    </div>

    <!-- 卡片 -->
    <div class="relative z-1 w-full max-w-105">
      <div class="mb-6 flex flex-col items-center gap-3">
        <span
          class="flex h-14 w-14 items-center justify-center rounded-2xl text-white shadow-lg"
          style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
        >
          <AppIcon name="zap" :size="28" />
        </span>
        <h1 class="text-22 font-700 text-[var(--c-text)]">{{ config.siteName }}</h1>
      </div>

      <div class="card-base p-6 md:p-8" style="--s-card: var(--s-pop)">
        <router-view />
      </div>

      <p class="mt-5 text-center text-12 text-[var(--c-text-sub)] opacity-70">
        {{ config.config?.site_description }}
      </p>
    </div>
  </div>
</template>
