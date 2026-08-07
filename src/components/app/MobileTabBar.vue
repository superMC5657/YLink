<script setup lang="ts">
/**
 * 移动端底部标签栏:4 个高频入口,激活主色,safe-area 适配。
 * 规范见 docs/frontend/pages.md §2.2。
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MOBILE_TABS } from '@/router/nav'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const activeIndex = computed(() => {
  const idx = MOBILE_TABS.findIndex((t) => route.path.startsWith(t.path))
  return idx >= 0 ? idx : 0
})

function go(path: string) {
  if (route.path === path) return
  router.push(path)
}
</script>

<template>
  <nav
    class="fixed inset-x-0 bottom-0 z-30 flex border-t border-[var(--c-border)] backdrop-blur-12"
    style="
      background: color-mix(in srgb, var(--c-bg-card) 90%, transparent);
      padding-bottom: env(safe-area-inset-bottom);
    "
  >
    <button
      v-for="(tab, i) in MOBILE_TABS"
      :key="tab.path"
      class="flex h-14 min-w-0 flex-1 cursor-pointer flex-col items-center justify-center gap-0.5 transition-colors"
      :style="{
        color: i === activeIndex ? 'var(--c-primary)' : 'var(--c-text-sub)',
      }"
      @click="go(tab.path)"
    >
      <span class="relative flex h-6 items-center">
        <AppIcon :name="tab.icon" :size="21" :stroke-width="i === activeIndex ? 2.4 : 2" />
        <span
          v-if="i === activeIndex"
          class="absolute -bottom-1 left-1/2 h-1 w-1 -translate-x-1/2 rounded-full"
          style="background: var(--c-primary)"
        />
      </span>
      <span class="text-14" :style="{ fontWeight: i === activeIndex ? 600 : 400 }">{{
        t(tab.name)
      }}</span>
    </button>
  </nav>
</template>
