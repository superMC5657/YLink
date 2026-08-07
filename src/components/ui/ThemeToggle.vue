<script setup lang="ts">
/**
 * 主题切换:亮 / 暗 / 跟随系统 三态循环,线性动画切换。
 */
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import type { ThemeMode } from '@/utils/storage'

const app = useAppStore()

const modes: { value: ThemeMode; label: string; icon: string }[] = [
  { value: 'light', label: '浅色', icon: 'sun' },
  { value: 'dark', label: '深色', icon: 'moon' },
  { value: 'system', label: '跟随系统', icon: 'globe' },
]

const currentIndex = computed(() => {
  const idx = modes.findIndex((m) => m.value === app.themeMode)
  return idx >= 0 ? idx : 2
})

function cycle() {
  const next = modes[(currentIndex.value + 1) % modes.length]
  app.setThemeMode(next.value)
}
</script>

<template>
  <button
    class="relative flex h-8 w-14 cursor-pointer items-center rounded-[var(--r-pill)] border border-[var(--c-border)] bg-[var(--c-bg-hover)] px-1 transition-colors"
    :title="modes[currentIndex].label"
    @click="cycle"
  >
    <span
      class="flex h-6 w-6 items-center justify-center rounded-full text-white shadow transition-transform duration-300"
      :style="{
        transform: `translateX(${currentIndex * 16}px)`,
        background: 'linear-gradient(135deg,#6558F5,#8B5CF6)',
      }"
    >
      <AppIcon :name="modes[currentIndex].icon" :size="14" />
    </span>
  </button>
</template>
