<script setup lang="ts">
/**
 * 快捷操作宫格:9 个入口(路由跳转 / 复制订阅 / 一键导入 / 免费流量说明 / APP 下载)。
 * 页面:docs/frontend/pages.md §3.3
 */
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useConfigStore } from '@/stores/config'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { copyText } from '@/utils/platform'
import { openExternal } from '@/utils/platform'
import ImportClientSheet from './ImportClientSheet.vue'

const router = useRouter()
const user = useUserStore()
const config = useConfigStore()
const message = useMessage()
const { t } = useI18n()

const showImport = ref(false)
const showFreeTraffic = ref(false)
const showDownload = ref(false)

interface Action {
  key: string
  icon: string
  label: string
  color: string
  bg: string
}

const actions: Action[] = [
  { key: 'docs', icon: 'book', label: t('dashboard.viewTutorial'), color: '#6558F5', bg: '#EAE7FF' },
  { key: 'invite', icon: 'gift', label: t('dashboard.inviteEarn'), color: '#C2487B', bg: '#FBE3EF' },
  { key: 'plans', icon: 'zap', label: t('dashboard.buyPlan'), color: '#D98E04', bg: '#FCEFCE' },
  { key: 'free-traffic', icon: 'sparkles', label: t('dashboard.freeTraffic'), color: '#5BA829', bg: '#DDF3C6' },
  { key: 'download', icon: 'download', label: t('dashboard.appDownload'), color: '#4B8FE5', bg: '#E1EEFB' },
  { key: 'tickets', icon: 'ticket', label: t('dashboard.myTickets'), color: '#7C9A3D', bg: '#EAF2D9' },
  { key: 'profile', icon: 'user', label: t('dashboard.profile'), color: '#C2487B', bg: '#FBE3EF' },
  { key: 'subscribe-link', icon: 'link', label: t('dashboard.subscribeLink'), color: '#6558F5', bg: '#EAE7FF' },
  { key: 'import', icon: 'download', label: t('dashboard.oneClickImport'), color: '#E5484D', bg: '#FDE3E4' },
]

function onAction(a: Action) {
  switch (a.key) {
    case 'docs':
      router.push('/docs')
      break
    case 'invite':
      router.push('/invite')
      break
    case 'plans':
      router.push('/plans')
      break
    case 'free-traffic':
      showFreeTraffic.value = true
      break
    case 'download':
      showDownload.value = true
      break
    case 'tickets':
      router.push('/tickets')
      break
    case 'profile':
      router.push('/profile')
      break
    case 'subscribe-link':
      void copySubscribe()
      break
    case 'import':
      showImport.value = true
      break
  }
}

async function copySubscribe() {
  const url = user.subscribe?.subscribe_url
  if (!url) {
    message.warning('暂无订阅链接,请先购买套餐')
    return
  }
  const ok = await copyText(url)
  if (ok) message.success(t('common.copied'))
}

function openDownload(kind: 'windows' | 'macos' | 'android') {
  const url = config.config?.app_downloads?.[kind]
  if (url) openExternal(url)
}
</script>

<template>
  <div class="card-base p-5 md:p-6">
    <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('dashboard.quickActions') }}</h3>
    <div class="grid grid-cols-4 gap-3 md:grid-cols-9">
      <button
        v-for="a in actions"
        :key="a.key"
        class="flex cursor-pointer flex-col items-center gap-1.5 rounded-xl py-3 transition-all duration-[var(--t-fast)] hover:-translate-y-0.5 active:scale-95"
        :style="{ backgroundColor: a.bg }"
        @click="onAction(a)"
      >
        <AppIcon :name="a.icon" :size="20" :style="{ color: a.color }" />
        <span class="text-11 leading-tight text-[var(--c-text)]">{{ a.label }}</span>
      </button>
    </div>

    <!-- 免费流量说明 -->
    <n-modal v-model:show="showFreeTraffic" preset="card" :title="t('dashboard.freeTraffic')" class="max-w-105">
      <div class="flex items-start gap-3">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full" style="background: var(--c-success-bg); color: var(--c-success)">
          <AppIcon name="sparkles" :size="20" />
        </span>
        <p class="text-14 leading-6 text-[var(--c-text)]">{{ config.config?.free_traffic_tips || '暂无说明' }}</p>
      </div>
    </n-modal>

    <!-- APP 下载 -->
    <n-modal v-model:show="showDownload" preset="card" :title="t('dashboard.appDownload')" class="max-w-105">
      <div class="grid grid-cols-3 gap-3">
        <button class="flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-[var(--c-border)] py-5 transition-colors hover:bg-[var(--c-bg-hover)]" @click="openDownload('windows')">
          <AppIcon name="download" :size="24" :style="{ color: 'var(--c-primary)' }" />
          <span class="text-13 font-500">Windows</span>
        </button>
        <button class="flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-[var(--c-border)] py-5 transition-colors hover:bg-[var(--c-bg-hover)]" @click="openDownload('macos')">
          <AppIcon name="download" :size="24" :style="{ color: 'var(--c-primary)' }" />
          <span class="text-13 font-500">macOS</span>
        </button>
        <button class="flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-[var(--c-border)] py-5 transition-colors hover:bg-[var(--c-bg-hover)]" @click="openDownload('android')">
          <AppIcon name="download" :size="24" :style="{ color: 'var(--c-primary)' }" />
          <span class="text-13 font-500">Android</span>
        </button>
      </div>
    </n-modal>

    <!-- 一键导入 -->
    <ImportClientSheet v-model:show="showImport" />
  </div>
</template>
