<script setup lang="ts">
/**
 * 空态:居中插画 + 文案。
 */
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

withDefaults(
  defineProps<{
    text?: string
    icon?: string
    description?: string
  }>(),
  { icon: 'database', description: '' },
)
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-3 py-12 text-center">
    <span
      class="flex h-16 w-16 items-center justify-center rounded-full text-[var(--c-text-sub)] opacity-40"
      :style="{ backgroundColor: 'var(--c-bg-hover)' }"
    >
      <AppIcon :name="icon" :size="30" />
    </span>
    <div class="text-14 text-[var(--c-text-sub)]">{{ text || t('common.empty') }}</div>
    <div v-if="description" class="text-14 text-[var(--c-text-sub)] opacity-70">
      {{ description }}
    </div>
    <slot />
  </div>
</template>
