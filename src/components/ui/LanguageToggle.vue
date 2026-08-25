<script setup lang="ts">
/**
 * 语言切换:简体中文 ↔ English 滑动切换。
 * 滑块位移按轨道几何精确计算:两端对称,左右余量一致。
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocale } from '@/composables/useLocale'

const { t, locale } = useI18n()
const { switchLocale } = useLocale()

const langs: { value: string; labelKey: string; shortKey: string }[] = [
  { value: 'zh-CN', labelKey: 'lang.zhCN', shortKey: 'lang.zhShort' },
  { value: 'en-US', labelKey: 'lang.enUS', shortKey: 'lang.enShort' },
]

// 轨道几何:w-14=56px,滑块 w-7=28px(容纳 14px 文字),左右对称余量 5px(border 1px + padding 4px)
const TRACK_W = 56
const SLIDER_W = 28
const EDGE_GAP = 5

const currentIndex = computed(() => {
  const idx = langs.findIndex((l) => l.value === locale.value)
  return idx >= 0 ? idx : 0
})

/** 滑块位移:2 态均分可用行程,两端分别贴左右边缘 */
const slideOffset = computed(() => {
  const usable = TRACK_W - EDGE_GAP * 2 - SLIDER_W
  const steps = langs.length - 1
  return ((usable / steps) * currentIndex.value).toFixed(1) + 'px'
})

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
        background: 'linear-gradient(135deg,#6558F5,#8B5CF6)',
      }"
    >
      {{ t(langs[currentIndex].shortKey) }}
    </span>
  </button>
</template>
