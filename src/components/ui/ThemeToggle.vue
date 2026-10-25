<script setup lang="ts">
/**
 * 主题切换:亮 / 暗 / 跟随系统 三态循环,线性动画切换。
 * 滑块位移几何归 useSlideOffset(与语言切换共用)。
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useSlideOffset } from '@/composables/useSlideOffset'
import type { ThemeMode } from '@/utils/storage'

const { t } = useI18n()
const app = useAppStore()

const modes: { value: ThemeMode; labelKey: string; icon: string }[] = [
  { value: 'light', labelKey: 'theme.light', icon: 'sun' },
  { value: 'dark', labelKey: 'theme.dark', icon: 'moon' },
  { value: 'system', labelKey: 'theme.system', icon: 'globe' },
]

const currentIndex = computed(() => {
  const idx = modes.findIndex((m) => m.value === app.themeMode)
  return idx >= 0 ? idx : 2
})

const slideOffset = useSlideOffset(currentIndex, modes.length)

function cycle() {
  const next = modes[(currentIndex.value + 1) % modes.length]
  app.setThemeMode(next.value)
}
</script>

<template>
  <button
    class="relative flex h-8 w-14 cursor-pointer items-center rounded-[var(--r-pill)] border border-[var(--c-border)] bg-[var(--c-bg-hover)] px-1 transition-colors"
    :title="t(modes[currentIndex].labelKey)"
    @click="cycle"
  >
    <span
      class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-white shadow transition-transform duration-300"
      :style="{
        transform: `translateX(${slideOffset})`,
        background: 'linear-gradient(135deg,var(--c-primary),var(--c-primary-grad-end))',
      }"
    >
      <AppIcon :name="modes[currentIndex].icon" :size="14" />
    </span>
  </button>
</template>
