<script setup lang="ts">
/**
 * 胶囊状态徽章。type: success / warning / danger / neutral / primary / marketing
 * 规范见 docs/frontend/design-system.md §7。
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    type?: 'success' | 'warning' | 'danger' | 'neutral' | 'primary' | 'marketing'
    dot?: boolean
  }>(),
  { type: 'neutral', dot: true },
)

const style = computed(() => {
  const map = {
    success: { color: 'var(--c-success)', bg: 'var(--c-success-bg)' },
    warning: { color: 'var(--c-warning)', bg: 'var(--c-warning-bg)' },
    danger: { color: 'var(--c-danger)', bg: 'var(--c-danger-bg)' },
    neutral: { color: 'var(--c-text-sub)', bg: 'var(--c-bg-hover)' },
    primary: { color: 'var(--c-primary-text)', bg: 'var(--c-primary-soft)' },
    marketing: { color: 'var(--c-marketing)', bg: 'var(--c-danger-bg)' },
  }
  return map[props.type]
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1 rounded-[var(--r-pill)] px-2.5 py-0.5 text-12 font-500 leading-5 whitespace-nowrap"
    :style="{ color: style.color, backgroundColor: style.bg }"
  >
    <span v-if="dot" class="h-1.5 w-1.5 rounded-full" :style="{ backgroundColor: style.color }" />
    <slot />
  </span>
</template>
