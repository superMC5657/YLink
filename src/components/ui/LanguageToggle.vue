<script setup lang="ts">
/**
 * 语言切换:简体中文 ↔ English 滑动切换。
 * 滑块位移几何归 useSlideOffset(与主题切换共用)。
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocale } from '@/composables/useLocale'
import { useSlideOffset } from '@/composables/useSlideOffset'

const { t, locale } = useI18n()
const { switchLocale } = useLocale()

const langs: { value: string; labelKey: string; shortKey: string }[] = [
  { value: 'zh-CN', labelKey: 'lang.zhCN', shortKey: 'lang.zhShort' },
  { value: 'en-US', labelKey: 'lang.enUS', shortKey: 'lang.enShort' },
]

const currentIndex = computed(() => {
  const idx = langs.findIndex((l) => l.value === locale.value)
  return idx >= 0 ? idx : 0
})

const slideOffset = useSlideOffset(currentIndex, langs.length)

function cycle() {
  const next = langs[(currentIndex.value + 1) % langs.length]
  void switchLocale(next.value)
}
</script>

<template>
  <button
    class="relative flex h-8 w-14 cursor-pointer items-center rounded-[var(--r-pill)] border border-[var(--c-border)] bg-[var(--c-bg-hover)] px-1 transition-colors"
    :title="t(langs[currentIndex].labelKey)"
    @click="cycle"
  >
    <span
      class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-14 font-600 text-white shadow transition-transform duration-300"
      :style="{
        transform: `translateX(${slideOffset})`,
        background: 'linear-gradient(135deg,var(--c-primary),var(--c-primary-grad-end))',
      }"
    >
      {{ t(langs[currentIndex].shortKey) }}
    </span>
  </button>
</template>
