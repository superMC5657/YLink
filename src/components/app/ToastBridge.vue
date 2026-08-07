<script setup lang="ts">
/**
 * 桥接组件:位于 message/dialog provider 内部。
 * 1) 把 naive message 注入 http 层(全局 toast);
 * 2) 监听网络状态,显示离线/恢复提示。
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { setToastProvider } from '@/utils/http'
import { useI18n } from 'vue-i18n'

declare global {
  interface Window {
    $dialog?: ReturnType<typeof useDialog>
  }
}

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
const wasOffline = ref(false)

setToastProvider((msg, type = 'error') => {
  if (type === 'success') message.success(msg)
  else if (type === 'warning') message.warning(msg)
  else message.error(msg)
})

// 暴露全局 dialog(供非组件上下文使用)
window.$dialog = dialog

onMounted(() => {
  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
})

onBeforeUnmount(() => {
  window.removeEventListener('online', handleOnline)
  window.removeEventListener('offline', handleOffline)
})

function handleOnline() {
  if (wasOffline.value) {
    wasOffline.value = false
    message.success(t('network.online'))
  }
}

function handleOffline() {
  wasOffline.value = true
  message.warning(t('network.offline'), { duration: 0 })
}
</script>

<template>
  <slot />
</template>
