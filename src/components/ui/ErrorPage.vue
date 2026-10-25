<script setup lang="ts">
/**
 * 错误页骨架(404/403 共用):渐变大数字 + 图标 + 标题/描述 + 返回仪表板按钮。
 * 仅文案与数字渐变配色随场景不同,经 props 注入。
 */
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

defineProps<{
  /** 展示的大数字(如 '404') */
  code: string
  /** 大数字渐变背景(随场景:404 主色渐变 / 403 警示色渐变) */
  gradient: string
  titleKey: string
  descKey: string
  backKey: string
}>()

const router = useRouter()
const { t } = useI18n()
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
    <span
      class="num text-8xl font-700"
      :style="{
        background: gradient,
        '-webkit-background-clip': 'text',
        'background-clip': 'text',
        color: 'transparent',
      }"
    >
      {{ code }}
    </span>
    <span
      class="flex h-16 w-16 items-center justify-center rounded-full text-[var(--c-text-sub)]"
      style="background-color: var(--c-bg-hover)"
    >
      <AppIcon name="alert" :size="28" />
    </span>
    <div>
      <h1 class="text-18 font-600 text-[var(--c-text)]">{{ t(titleKey) }}</h1>
      <p class="mt-1 text-14 text-[var(--c-text-sub)]">{{ t(descKey) }}</p>
    </div>
    <button class="btn-primary h-10 px-6 text-14" @click="router.push('/dashboard')">
      <AppIcon name="home" :size="16" />
      {{ t(backKey) }}
    </button>
  </div>
</template>
