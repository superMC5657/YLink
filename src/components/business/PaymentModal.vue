<script setup lang="ts">
/**
 * 收银台弹窗:二维码/跳转链接 + 倒计时 + 每 3s 轮询订单状态,支付成功展示结果卡。
 * 数据:POST /orders/{no}/checkout、GET /orders/{no}(docs/api/README.md §10.3/§10.5)
 */
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useOrderStore } from '@/stores/order'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
import { useCountdown } from '@/composables/useCountdown'
import { openExternal } from '@/utils/platform'
import { notify } from '@/utils/notify'
import { formatMoney, formatTime } from '@/utils/format'
import QRCode from 'qrcode'
import type { Order } from '@/types/api'

const props = defineProps<{
  show: boolean
  order: Order | null
  /** 支付方式 code */
  method: string
}>()

const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'paid'): void }>()

const orderStore = useOrderStore()
const userStore = useUserStore()
const { t } = useI18n()
const { remaining, start } = useCountdown(1800)

const phase = ref<'loading' | 'qrcode' | 'url' | 'paid' | 'error'>('loading')
const qrDataUrl = ref('')
const checkoutUrl = ref('')
const payMethodName = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null

/** 运行时读取 CSS 变量,避免硬编码主题色(设计规范 tokens.css) */
function resolveCssVar(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

const payMethodLabel = computed(() => {
  const map: Record<string, string> = {
    balance: t('order.payMethodBalance'),
    epay_alipay: t('order.payMethodAlipay'),
    epay_wxpay: t('order.payMethodWxpay'),
  }
  return map[props.method] ?? props.method
})

async function initCheckout() {
  if (!props.order) return
  phase.value = 'loading'
  try {
    const resp = await orderStore.checkout(props.order.order_no, { method: props.method })
    payMethodName.value = payMethodLabel.value
    if (resp.type === 'paid') {
      phase.value = 'paid'
      emit('paid')
      void notifyPaid()
      void userStore.refreshDashboard()
      return
    }
    if (resp.type === 'url') {
      checkoutUrl.value = resp.content ?? ''
      phase.value = 'url'
      start(resp.expire_in || 1800)
    } else {
      const dataUrl = await QRCode.toDataURL(resp.content ?? '', {
        width: 220,
        margin: 1,
        color: {
          dark: resolveCssVar('--c-text', '#1F2430'),
          light: resolveCssVar('--c-bg-card', '#FFFFFF'),
        },
      })
      qrDataUrl.value = dataUrl
      phase.value = 'qrcode'
      start(resp.expire_in || 1800)
    }
    startPolling()
  } catch {
    // 错误提示由 http 层统一 toast;这里仅切换到错误态 UI
    phase.value = 'error'
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    if (!props.order) return
    try {
      const fresh = await orderStore.fetchDetail(props.order.order_no)
      if (fresh.status === 1) {
        phase.value = 'paid'
        emit('paid')
        stopPolling()
        void notifyPaid()
        void userStore.refreshDashboard()
      }
    } catch {
      // 轮询失败忽略
    }
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function goUrl() {
  if (checkoutUrl.value) openExternal(checkoutUrl.value)
}

/** 支付成功本地通知(desktop-tauri.md §4) */
async function notifyPaid() {
  await notify(t('notify.paySuccess'), props.order?.order_no)
}

watch(
  () => props.show,
  (v) => {
    if (v) void initCheckout()
    else {
      stopPolling()
      phase.value = 'loading'
    }
  },
)

onBeforeUnmount(stopPolling)
</script>

<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="phase === 'paid' ? t('plan.paySuccess') : t('plan.payNow')"
    class="max-w-100"
    :mask-closable="phase !== 'paid'"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <!-- 加载 -->
    <div v-if="phase === 'loading'" class="flex flex-col items-center gap-4 py-10">
      <span
        class="h-8 w-8 animate-spin rounded-full border-3 border-[var(--c-border)] border-t-[var(--c-primary)]"
      />
      <span class="text-14 text-[var(--c-text-sub)]">{{ t('plan.orderCreating') }}</span>
    </div>

    <!-- 二维码 -->
    <div v-else-if="phase === 'qrcode'" class="flex flex-col items-center gap-4 py-2">
      <div class="rounded-2xl border border-[var(--c-border)] p-3">
        <img :src="qrDataUrl" alt="qr" class="h-55 w-55" />
      </div>
      <p class="text-14 text-[var(--c-text-sub)]">{{ t('plan.qrcodeTip') }}</p>
      <div
        class="flex items-center gap-1.5 rounded-full px-4 py-1.5"
        style="background-color: var(--c-bg-hover)"
      >
        <AppIcon name="clock" :size="15" :style="{ color: 'var(--c-warning)' }" />
        <span class="num text-14 font-600 text-[var(--c-text)]">
          {{ Math.floor(remaining / 60) }}:{{ String(remaining % 60).padStart(2, '0') }}
        </span>
      </div>
    </div>

    <!-- 跳转 -->
    <div v-else-if="phase === 'url'" class="flex flex-col items-center gap-4 py-4">
      <span
        class="flex h-16 w-16 items-center justify-center rounded-full"
        style="background: var(--c-primary-soft); color: var(--c-primary-text)"
      >
        <AppIcon name="external-link" :size="28" />
      </span>
      <p class="text-14 text-[var(--c-text)]">{{ t('plan.urlTip') }}</p>
      <button class="btn-primary h-10 px-6 text-14" @click="goUrl">
        {{ t('plan.payNow') }} · {{ payMethodName }}
      </button>
      <div
        class="flex items-center gap-1.5 rounded-full px-4 py-1.5"
        style="background-color: var(--c-bg-hover)"
      >
        <AppIcon name="clock" :size="15" :style="{ color: 'var(--c-warning)' }" />
        <span class="num text-14 font-600 text-[var(--c-text)]">
          {{ Math.floor(remaining / 60) }}:{{ String(remaining % 60).padStart(2, '0') }}
        </span>
      </div>
    </div>

    <!-- 成功 -->
    <div v-else-if="phase === 'paid'" class="flex flex-col items-center gap-3 py-6 text-center">
      <span
        class="flex h-18 w-18 items-center justify-center rounded-full"
        style="background: var(--c-success-bg); color: var(--c-success)"
      >
        <AppIcon name="check" :size="36" :stroke-width="3" />
      </span>
      <h3 class="text-20 font-700 text-[var(--c-text)]">{{ t('plan.paySuccess') }}</h3>
      <p class="text-14 text-[var(--c-text-sub)]">{{ t('plan.paySuccessTip') }}</p>
      <div class="mt-2 w-full rounded-xl p-4 text-left" style="background-color: var(--c-bg-hover)">
        <div class="flex justify-between text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('order.orderNo') }}</span>
          <span class="num text-[var(--c-text)]">{{ props.order?.order_no }}</span>
        </div>
        <div class="mt-1.5 flex justify-between text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('order.payAmount') }}</span>
          <span class="num font-600 text-[var(--c-success)]">{{
            formatMoney(props.order?.pay_amount)
          }}</span>
        </div>
        <div class="mt-1.5 flex justify-between text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('order.paidAt') }}</span>
          <span class="text-[var(--c-text)]">{{ formatTime(props.order?.paid_at) }}</span>
        </div>
      </div>
      <button class="btn-primary mt-2 h-10 px-8 text-14" @click="emit('update:show', false)">
        {{ t('common.confirm') }}
      </button>
    </div>

    <!-- 失败 -->
    <div v-else class="flex flex-col items-center gap-3 py-6">
      <span
        class="flex h-16 w-16 items-center justify-center rounded-full"
        style="background: var(--c-danger-bg); color: var(--c-danger)"
      >
        <AppIcon name="alert" :size="28" />
      </span>
      <p class="text-14 text-[var(--c-text)]">{{ t('plan.payExpired') }}</p>
      <button class="btn-soft-neutral h-9 px-5 text-14" @click="emit('update:show', false)">
        {{ t('common.close') }}
      </button>
    </div>
  </n-modal>
</template>
