<script setup lang="ts">
/**
 * 大数字统计卡:数字字体 + 单位小字 + 说明文案,支持颜色语义。
 * 规范见 docs/frontend/design-system.md §7。
 */
withDefaults(
  defineProps<{
    label: string
    value: string | number
    unit?: string
    color?: string
    icon?: string
    iconColor?: string
  }>(),
  { unit: '', color: 'var(--c-text)', icon: '', iconColor: 'var(--c-primary)' },
)
</script>

<template>
  <div class="card-base card-hoverable flex flex-col gap-3 p-5">
    <div class="flex items-center justify-between">
      <span class="text-14 text-[var(--c-text-sub)]">{{ label }}</span>
      <span
        v-if="icon"
        class="flex h-9 w-9 items-center justify-center rounded-full"
        :style="{ color: iconColor, backgroundColor: 'var(--c-primary-soft)' }"
      >
        <AppIcon :name="icon" :size="18" />
      </span>
    </div>
    <div class="flex items-baseline gap-1">
      <span class="num text-28 font-700 leading-none" :style="{ color }">{{ value }}</span>
      <span v-if="unit" class="text-14 text-[var(--c-text-sub)]">{{ unit }}</span>
    </div>
  </div>
</template>
