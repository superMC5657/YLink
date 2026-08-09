<script setup lang="ts">
/**
 * 可复制文本:省略显示 + 复制按钮 + 成功反馈。
 */
import { ref } from 'vue'
import { copyText } from '@/utils/platform'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    text: string
    /** 展示文本,缺省用 text */
    display?: string
    /** 最大展示宽度下省略 */
    ellipsis?: boolean
    maxChars?: number
    silent?: boolean
  }>(),
  { ellipsis: true, maxChars: 24, silent: false },
)

const message = useMessage()
const { t } = useI18n()
const copied = ref(false)

async function onCopy() {
  const ok = await copyText(props.text)
  if (ok) {
    copied.value = true
    if (!props.silent) message.success(t('common.copied'))
    setTimeout(() => (copied.value = false), 1500)
  } else if (!props.silent) {
    message.error('复制失败')
  }
}

const shown = computedDisplay()

function computedDisplay(): string {
  if (!props.ellipsis) return props.text
  const base = props.display ?? props.text
  return base.length > props.maxChars ? base.slice(0, props.maxChars - 1) + '…' : base
}
</script>

<template>
  <span class="inline-flex max-w-full items-center gap-1.5 align-middle">
    <span class="num break-all text-14 text-[var(--c-text-sub)]" :title="text">{{ shown }}</span>
    <button
      class="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center rounded transition-colors hover:bg-[var(--c-bg-hover)]"
      :style="{ color: copied ? 'var(--c-success)' : 'var(--c-text-sub)' }"
      @click="onCopy"
    >
      <AppIcon :name="copied ? 'check' : 'copy'" :size="14" />
    </button>
  </span>
</template>
