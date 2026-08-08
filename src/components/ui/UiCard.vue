<script setup lang="ts">
/**
 * 通用卡片:白底/暗色实底、16px 圆角、极软阴影;可选 hover 上浮。
 * 规范见 docs/frontend/design-system.md §4 / §7。
 */
withDefaults(
  defineProps<{
    hoverable?: boolean
    padding?: string
  }>(),
  { hoverable: false, padding: '24px' },
)
</script>

<template>
  <div class="card-base w-full" :class="hoverable ? 'card-hoverable' : ''" :style="{ padding }">
    <slot />
  </div>
</template>
