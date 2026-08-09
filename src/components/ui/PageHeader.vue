<script setup lang="ts">
/**
 * 页面头部:标题 + 副标题 + 右侧操作区。
 */
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-20 font-600 text-[var(--c-text)]">{{ title }}</h1>
      <p v-if="subtitle" class="mt-1 text-14 text-[var(--c-text-sub)]">{{ subtitle }}</p>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
