<script setup lang="ts">
/**
 * 404 页:插画 + 返回首页按钮。
 */
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
    <span class="num text-8xl font-700" style="background: linear-gradient(135deg, #6558f5, #8b5cf6); -webkit-background-clip: text; background-clip: text; color: transparent">
      404
    </span>
    <span class="flex h-16 w-16 items-center justify-center rounded-full text-[var(--c-text-sub)]" style="background-color: var(--c-bg-hover)">
      <AppIcon name="alert" :size="28" />
    </span>
    <div>
      <h1 class="text-18 font-600 text-[var(--c-text)]">{{ t('notFound.title') }}</h1>
      <p class="mt-1 text-13 text-[var(--c-text-sub)]">{{ t('notFound.desc') }}</p>
    </div>
    <button class="btn-primary h-10 px-6 text-14" @click="router.push('/dashboard')">
      <AppIcon name="home" :size="16" />
      {{ t('notFound.backHome') }}
    </button>
  </div>
</template>
