<script setup lang="ts">
/**
 * 主题切换:亮 / 暗 / 跟随系统 三态循环,线性动画切换。
 * 滑块位移按轨道几何精确计算:三态均分可用行程,第三态刚好贴右边缘,不超出胶囊。
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

// 轨道几何:w-14=56px,滑块 w-6=24px,左右对称余量 5px(border 1px + padding 4px)
const TRACK_W = 56
const SLIDER_W = 24
const EDGE_GAP = 5

const currentIndex = computed(() => {
  const idx = modes.findIndex((m) => m.value === app.themeMode)
  return idx >= 0 ? idx : 2
})

/** 滑块位移:3 态均分可用行程,首尾态分别贴左右边缘,中间态居中 */
const slideOffset = computed(() => {
  const usable = TRACK_W - EDGE_GAP * 2 - SLIDER_W
  const steps = modes.length - 1
  return ((usable / steps) * currentIndex.value).toFixed(1) + 'px'
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
      class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-white shadow transition-transform duration-300"
      :style="{
        transform: `translateX(${slideOffset})`,
        background: 'linear-gradient(135deg,#6558F5,#8B5CF6)',
      }"
    >
      <AppIcon :name="modes[currentIndex].icon" :size="14" />
    </span>
  </button>
</template>
