<script setup lang="ts">
/**
 * 桥接组件(位于 message/dialog provider 内部):
 * 1) 把 naive message 注入 http 层(全局 toast);
 * 2) 暴露全局 dialog;
 * 3) 网络离线顶部横幅 + 恢复提示(页面:docs/frontend/pages.md §3.13)。
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
const offline = ref(false)

setToastProvider((msg, type = 'error') => {
  if (type === 'success') message.success(msg)
  else if (type === 'warning') message.warning(msg)
  else message.error(msg)
})

window.$dialog = dialog

function handleOnline() {
  if (offline.value) {
    offline.value = false
    message.success(t('network.online'))
  }
}

function handleOffline() {
  offline.value = true
}

onMounted(() => {
  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
})

onBeforeUnmount(() => {
  window.removeEventListener('online', handleOnline)
  window.removeEventListener('offline', handleOffline)
})
</script>

<template>
  <div class="pointer-events-none fixed inset-x-0 top-0 z-50">
    <transition name="fade-slide">
      <div
        v-if="offline"
        class="flex h-10 items-center justify-center gap-2 text-14 font-500 text-white"
        style="background: linear-gradient(90deg, #e5484d, #f16a6e)"
      >
        <span class="h-2 w-2 animate-pulse rounded-full bg-white" />
        {{ t('network.offline') }}
      </div>
    </transition>
  </div>
  <slot />
</template>
