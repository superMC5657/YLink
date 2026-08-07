<script setup lang="ts">
/**
 * 价格展示:货币符号小字 + 整数大字 + 小数小字;支持划线原价。
 * 规范见 docs/frontend/design-system.md §3。
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 元(两位小数) */
    value: number
    size?: number
    /** 划线原价(元) */
    original?: number | null
    color?: string
  }>(),
  { size: 32, original: null, color: 'var(--c-text)' },
)

const parts = computed(() => {
  const fixed = props.value.toFixed(2)
  const [int, dec] = fixed.split('.')
  return { int, dec }
})
</script>

<template>
  <span class="inline-flex items-baseline gap-1">
    <span class="num text-14 font-500" :style="{ color: 'var(--c-text-sub)' }">¥</span>
    <span class="num font-700 leading-none" :style="{ fontSize: `${size}px`, color }">{{
      parts.int
    }}</span>
    <span class="num text-14 font-500" :style="{ color: 'var(--c-text-sub)' }"
      >.{{ parts.dec }}</span
    >
    <span v-if="original" class="num ml-1 text-14 text-[var(--c-text-sub)] line-through opacity-70"
      >¥{{ original.toFixed(2) }}</span
    >
  </span>
</template>
