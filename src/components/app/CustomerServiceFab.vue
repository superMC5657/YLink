<script setup lang="ts">
/**
 * 客服浮球:右下角外链客服(TG/网页),地址取站点配置。
 */
import { ref, watch } from 'vue'
import { useConfigStore } from '@/stores/config'
import { openExternal } from '@/utils/platform'

const config = useConfigStore()
const show = ref(false)

const serviceUrl = ref('')

watch(
  () => config.config?.customer_service_url,
  (url) => {
    serviceUrl.value = url ?? ''
  },
  { immediate: true },
)

function open() {
  if (serviceUrl.value) openExternal(serviceUrl.value)
}
</script>

<template>
  <div v-if="serviceUrl" class="fixed bottom-20 right-4 z-20 md:bottom-6 md:right-6">
    <button
      class="group relative flex h-13 w-13 cursor-pointer items-center justify-center rounded-full text-white shadow-lg transition-transform hover:scale-105 active:scale-95"
      style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
      @mouseenter="show = true"
      @mouseleave="show = false"
      @click="open"
    >
      <AppIcon name="headset" :size="24" />
      <transition name="fade">
        <span
          v-if="show"
          class="absolute right-full mr-3 whitespace-nowrap rounded-full bg-[var(--c-bg-card)] px-3 py-1 text-12 text-[var(--c-text)] shadow"
          style="--s-card: var(--s-pop)"
        >
          在线客服
        </span>
      </transition>
    </button>
  </div>
</template>
