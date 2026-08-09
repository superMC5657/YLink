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
import { useI18n } from 'vue-i18n'

const show = defineModel<boolean>('show', { default: false })

const user = useUserStore()
const config = useConfigStore()
const message = useMessage()
const { t } = useI18n()
const copied = ref(false)

function onPick(client: ClientKind) {
  const url = user.subscribe?.subscribe_url
  if (!url) {
    message.warning('暂无订阅链接,请先购买套餐')
    return
  }
  const ok = importToClient(client, url, config.siteName)
  if (!ok) {
    message.info('该客户端不支持自动唤起,请复制订阅链接后手动导入')
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
      class="mt-4 flex items-center justify-between rounded-xl p-3"
      style="background-color: var(--c-bg-hover)"
    >
      <span class="min-w-0 flex-1 truncate text-14 text-[var(--c-text-sub)]">
        {{ user.subscribe?.subscribe_url ?? '暂无订阅' }}
      </span>
      <button class="btn-ghost ml-3 h-8 shrink-0 px-3 text-14" @click="doCopy">
        <AppIcon :name="copied ? 'check' : 'copy'" :size="14" />
        {{ copied ? t('common.copied') : t('common.copy') }}
      </button>
    </div>
  </n-modal>
</template>
