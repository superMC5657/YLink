<script setup lang="ts">
/**
 * 倒计时胶囊(收银台二维码/跳转两态共用):时钟图标 + mm:ss。
 */
import { computed } from 'vue'

const props = defineProps<{ remaining: number }>()

const text = computed(
  () => `${Math.floor(props.remaining / 60)}:${String(props.remaining % 60).padStart(2, '0')}`,
)
</script>

<template>
  <div
    class="flex items-center gap-1.5 rounded-full px-4 py-1.5"
    style="background-color: var(--c-bg-hover)"
  >
    <AppIcon name="clock" :size="15" :style="{ color: 'var(--c-warning)' }" />
    <span class="num text-14 font-600 text-[var(--c-text)]">{{ text }}</span>
  </div>
</template>
