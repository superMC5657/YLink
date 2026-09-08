<script setup lang="ts">
/**
 * 一键导入客户端选择弹窗:按客户端生成 scheme 唤起,提供「复制订阅链接」兜底。
 * 规范见 docs/frontend/data-layer.md §6。
 */
import { ref } from 'vue'
import { CLIENT_OPTIONS, importToClient } from '@/utils/deeplink'
import type { ClientKind } from '@/utils/deeplink'
import { useUserStore } from '@/stores/user'
import { useConfigStore } from '@/stores/config'
import { useMessage } from 'naive-ui'
import { copyText } from '@/utils/platform'
import { maskUrl } from '@/utils/format'
import { useI18n } from 'vue-i18n'

const show = defineModel<boolean>('show', { default: false })

const user = useUserStore()
const config = useConfigStore()
const message = useMessage()
const { t } = useI18n()
const copied = ref(false)
/** 订阅链接默认掩码展示,显式点击后才显示原文(令牌类敏感信息不进 DOM) */
const revealUrl = ref(false)

function onPick(client: ClientKind) {
  const url = user.subscribe?.subscribe_url
  if (!url) {
    message.warning(t('common.noSubscribeLink'))
    return
  }
  const ok = importToClient(client, url, config.siteName)
  if (!ok) {
    message.info(t('common.clientManualImport'))
    void doCopy()
  }
}

async function doCopy() {
  const url = user.subscribe?.subscribe_url
  if (!url) return
  const ok = await copyText(url)
  if (ok) {
    copied.value = true
    message.success(t('common.copied'))
    setTimeout(() => (copied.value = false), 1500)
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="t('dashboard.oneClickImport')"
    class="max-w-120"
  >
    <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
      <button
        v-for="opt in CLIENT_OPTIONS"
        :key="opt.kind"
        class="flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-[var(--c-border)] py-4 transition-all duration-[var(--t-fast)] hover:border-[var(--c-primary)] hover:bg-[var(--c-primary-soft)]"
        @click="onPick(opt.kind)"
      >
        <span
          class="flex h-11 w-11 items-center justify-center rounded-full"
          style="background: var(--c-primary-soft); color: var(--c-primary-text)"
        >
          <AppIcon
            :name="
              opt.kind.includes('clash') ? 'zap' : opt.kind === 'sing-box' ? 'box' : 'download'
            "
            :size="22"
          />
        </span>
        <span class="text-14 font-500 text-[var(--c-text)]">{{ opt.name }}</span>
        <span class="text-14 text-[var(--c-text-sub)]">{{ opt.platforms.join(' / ') }}</span>
      </button>
    </div>

    <div
      class="mt-4 flex items-center justify-between gap-2 rounded-xl p-3"
      style="background-color: var(--c-bg-hover)"
    >
      <span class="num min-w-0 flex-1 truncate text-14 text-[var(--c-text-sub)]">
        {{
          user.subscribe?.subscribe_url
            ? revealUrl
              ? user.subscribe.subscribe_url
              : maskUrl(user.subscribe.subscribe_url)
            : t('common.noSubscribe')
        }}
      </span>
      <button
        v-if="user.subscribe?.subscribe_url"
        class="btn-soft-neutral h-8 shrink-0 px-3 text-14"
        @click="revealUrl = !revealUrl"
      >
        {{ revealUrl ? t('common.hide') : t('common.show') }}
      </button>
      <button class="btn-soft-blue h-8 shrink-0 px-3 text-14" @click="doCopy">
        <AppIcon :name="copied ? 'check' : 'copy'" :size="14" />
        {{ copied ? t('common.copied') : t('common.copy') }}
      </button>
    </div>
  </n-modal>
</template>
