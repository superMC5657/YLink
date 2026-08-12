<script setup lang="ts">
/**
 * 桌面端更新卡片:启动时静默检查(desktop-tauri.md §5),发现新版本右下角浮出;
 * 也监听 `app:check-update` 事件(设置页「检查更新」入口触发)。
 * Web 端不渲染(desktop-tauri.md §8:Web 隐藏桌面专属入口)。
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { isTauri } from '@/utils/platform'
import { checkForUpdate, downloadAndInstall } from '@/utils/updater'
import type { AppUpdateInfo } from '@/utils/updater'

const { t } = useI18n()
const message = useMessage()

const visible = ref(false)
const info = ref<AppUpdateInfo | null>(null)
const downloading = ref(false)
const failed = ref(false)
const received = ref(0)
const total = ref<number | undefined>(undefined)

const percent = computed(() => {
  if (!total.value) return 0
  return Math.min(100, Math.round((received.value / total.value) * 100))
})

async function doCheck() {
  if (!isTauri() || visible.value || downloading.value) return
  const update = await checkForUpdate()
  if (!update) return
  info.value = update
  failed.value = false
  visible.value = true
}

async function onInstall() {
  if (!info.value || downloading.value) return
  downloading.value = true
  failed.value = false
  received.value = 0
  total.value = undefined
  const ok = await downloadAndInstall((r, totalBytes) => {
    received.value = r
    total.value = totalBytes
  })
  if (!ok) {
    downloading.value = false
    failed.value = true
    message.error(t('update.installFailed'))
  }
  // 成功路径:downloadAndInstall 内部已 relaunch,进程即将退出
}

function onDismiss() {
  if (downloading.value) return
  visible.value = false
}

function onCheckEvent() {
  void doCheck()
}

onMounted(() => {
  window.addEventListener('app:check-update', onCheckEvent)
  // 启动静默检查:失败静默忽略,不打扰用户
  void doCheck()
})

onBeforeUnmount(() => {
  window.removeEventListener('app:check-update', onCheckEvent)
})
</script>

<template>
  <Transition name="fade-slide">
    <div
      v-if="visible && info"
      class="fixed right-4 bottom-20 z-50 w-[19rem] md:right-6 md:bottom-6"
    >
      <div class="card-base p-5 shadow-xl">
        <div class="mb-3 flex items-start justify-between gap-2">
          <div class="flex items-center gap-2.5">
            <span
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full"
              style="background: var(--c-primary-soft); color: var(--c-primary-text)"
            >
              <AppIcon name="download" :size="17" />
            </span>
            <div>
              <div class="text-15 font-600 text-[var(--c-text)]">{{ t('update.available') }}</div>
              <div class="text-13 text-[var(--c-text-sub)]">
                {{ info.currentVersion }}
                <span class="mx-0.5 text-[var(--c-text-sub)]">→</span>
                <span class="num font-500 text-[var(--c-primary-text)]">v{{ info.version }}</span>
              </div>
            </div>
          </div>
          <button
            class="cursor-pointer text-[var(--c-text-sub)] transition-colors hover:text-[var(--c-text)]"
            :disabled="downloading"
            @click="onDismiss"
          >
            <AppIcon name="close" :size="16" />
          </button>
        </div>

        <p
          v-if="info.body"
          class="mb-3 max-h-24 overflow-y-auto rounded-lg p-3 text-13 leading-5 whitespace-pre-wrap text-[var(--c-text-sub)]"
          style="background-color: var(--c-bg-hover)"
        >
          {{ info.body }}
        </p>

        <!-- 下载进度 -->
        <div v-if="downloading" class="mb-3">
          <div
            class="h-1.5 w-full overflow-hidden rounded-full"
            style="background: var(--c-bg-hover)"
          >
            <div
              class="h-full rounded-full transition-all"
              :style="{
                width: total ? percent + '%' : '40%',
                background: 'linear-gradient(90deg,#6558F5,#8B5CF6)',
              }"
            />
          </div>
          <div class="mt-1.5 text-13 text-[var(--c-text-sub)]">
            {{ t('update.downloading') }}<template v-if="total"> · {{ percent }}%</template>
          </div>
        </div>

        <p v-else-if="failed" class="mb-3 text-13 text-[var(--c-danger)]">
          {{ t('update.installFailedTip') }}
        </p>

        <div v-if="!downloading" class="flex gap-2">
          <button class="btn-primary h-9 flex-1 text-14" @click="onInstall">
            <AppIcon name="download" :size="14" />
            {{ t('update.install') }}
          </button>
          <button class="btn-soft-neutral h-9 shrink-0 px-4 text-14" @click="onDismiss">
            {{ t('update.later') }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>
