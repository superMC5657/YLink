<script setup lang="ts">
/**
 * 语言切换:简体中文 ↔ English 滑动切换。
 * 滑块位移按轨道几何精确计算:两端对称,左右余量一致。
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocale } from '@/composables/useLocale'

const { locale } = useI18n()
const { switchLocale } = useLocale()

const langs: { value: string; label: string; short: string }[] = [
  { value: 'zh-CN', label: '简体中文', short: '中' },
  { value: 'en-US', label: 'English', short: 'EN' },
]

// 轨道几何:w-14=56px,滑块 w-6=24px,左右对称余量 5px(border 1px + padding 4px)
const TRACK_W = 56
const SLIDER_W = 24
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
    :title="langs[currentIndex].label"
    @click="cycle"
  >
    <span
      class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-10 font-600 text-white shadow transition-transform duration-300"
      :style="{
        transform: `translateX(${slideOffset})`,
        background: 'linear-gradient(135deg,#6558F5,#8B5CF6)',
      }"
    >
      {{ langs[currentIndex].short }}
    </span>
  </button>
</template>
