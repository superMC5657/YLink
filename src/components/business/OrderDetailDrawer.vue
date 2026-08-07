<script setup lang="ts">
/**
 * 订单详情抽屉:全字段 + 支付入口(待支付)+ 取消按钮。
 * 数据:GET /orders/{no}、POST /orders/{no}/cancel(契约 §10.3/§10.4)
 */
import { computed, ref, watch } from 'vue'
import { useOrderStore } from '@/stores/order'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import {
  formatMoney,
  formatTime,
  orderStatusLabel,
  periodLabel,
} from '@/utils/format'
import PaymentModal from './PaymentModal.vue'
import type { Order } from '@/types/api'

const props = defineProps<{ show: boolean; orderNo: string | null }>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'changed'): void }>()

const orderStore = useOrderStore()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const detail = ref<Order | null>(null)
const loading = ref(false)
const showPayment = ref(false)
const payMethod = ref('epay_alipay')

watch(
  () => props.show,
  async (v) => {
    if (v && props.orderNo) {
      loading.value = true
      try {
        detail.value = await orderStore.fetchDetail(props.orderNo)
      } finally {
        loading.value = false
      }
    } else {
      detail.value = null
    }
  },
)

const statusType = computed(() => {
  const map: Record<Order['status'], 'success' | 'warning' | 'neutral' | 'danger'> = {
    0: 'warning',
    1: 'success',
    2: 'neutral',
    3: 'danger',
  }
  return map[(detail.value?.status ?? 2) as Order['status']]
})

async function onCancel() {
  if (!detail.value) return
  dialog.warning({
    title: t('order.cancelOrder'),
    content: t('order.cancelConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await orderStore.cancel(detail.value!.order_no)
      message.success(t('plan.orderCanceled'))
      detail.value = await orderStore.fetchDetail(detail.value!.order_no)
      emit('changed')
    },
  })
}

function onPaid() {
  emit('changed')
  void orderStore.fetchDetail(detail.value!.order_no).then((d) => (detail.value = d))
}

function openPayment(method: string) {
  payMethod.value = method
  showPayment.value = true
}
</script>

<template>
  <n-drawer :show="props.show" @update:show="(v: boolean) => emit('update:show', v)" :width="420" placement="right">
    <n-drawer-content :title="t('order.detail')" closable>
      <n-spin :show="loading">
        <template v-if="detail">
          <div class="flex items-center justify-between">
            <span class="text-16 font-600 text-[var(--c-text)]">{{ detail.plan_name }}</span>
            <StatusBadge :type="statusType">{{ orderStatusLabel(detail.status) }}</StatusBadge>
          </div>

          <div class="mt-5 space-y-3 rounded-xl p-4" style="background-color: var(--c-bg-hover)">
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.orderNo') }}</span>
              <CopyText :text="detail.order_no" :max-chars="22" :silent="true" />
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.period') }}</span>
              <span class="text-[var(--c-text)]">{{ periodLabel(detail.period) }}</span>
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.amount') }}</span>
              <span class="num text-[var(--c-text)]">{{ formatMoney(detail.amount) }}</span>
            </div>
            <div v-if="detail.discount_amount > 0" class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.discount') }}</span>
              <span class="num text-[var(--c-marketing)]">-{{ formatMoney(detail.discount_amount) }}</span>
            </div>
            <div v-if="detail.balance_used > 0" class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.balanceUsed') }}</span>
              <span class="num text-[var(--c-text)]">-{{ formatMoney(detail.balance_used) }}</span>
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.payAmount') }}</span>
              <span class="num text-16 font-700 text-[var(--c-success)]">{{ formatMoney(detail.pay_amount) }}</span>
            </div>
            <div v-if="detail.coupon_code" class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.coupon') }}</span>
              <span class="text-[var(--c-text)]">{{ detail.coupon_code }}</span>
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.payMethod') }}</span>
              <span class="text-[var(--c-text)]">
                {{ detail.pay_method ? (detail.pay_method === 'balance' ? '余额支付' : detail.pay_method === 'epay_alipay' ? '支付宝' : '微信支付') : '-' }}
              </span>
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.paidAt') }}</span>
              <span class="text-[var(--c-text)]">{{ formatTime(detail.paid_at) }}</span>
            </div>
            <div class="flex justify-between text-13">
              <span class="text-[var(--c-text-sub)]">{{ t('order.createdAt') }}</span>
              <span class="text-[var(--c-text)]">{{ formatTime(detail.created_at) }}</span>
            </div>
          </div>

          <!-- 待支付:支付方式 + 操作 -->
          <template v-if="detail.status === 0">
            <div class="mt-5">
              <div class="mb-2 text-13 font-500 text-[var(--c-text)]">{{ t('plan.payment') }}</div>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="m in [
                    { code: 'epay_alipay', name: '支付宝', icon: 'credit' },
                    { code: 'epay_wxpay', name: '微信支付', icon: 'credit' },
                    { code: 'balance', name: '余额支付', icon: 'wallet' },
                  ]"
                  :key="m.code"
                  class="flex cursor-pointer flex-col items-center gap-1 rounded-xl border py-3 transition-colors"
                  :class="payMethod === m.code ? 'border-[var(--c-primary)] bg-[var(--c-primary-soft)]' : 'border-[var(--c-border)] hover:bg-[var(--c-bg-hover)]'"
                  @click="payMethod = m.code"
                >
                  <AppIcon :name="m.icon" :size="18" :style="{ color: payMethod === m.code ? 'var(--c-primary)' : 'var(--c-text-sub)' }" />
                  <span class="text-12" :style="{ color: payMethod === m.code ? 'var(--c-primary-text)' : 'var(--c-text-sub)' }">{{ m.name }}</span>
                </button>
              </div>
            </div>

            <div class="mt-5 flex gap-2">
              <button class="btn-primary h-10 flex-1 text-14" @click="openPayment(payMethod)">
                {{ t('order.goPay') }} · {{ formatMoney(detail.pay_amount) }}
              </button>
              <button class="btn-ghost h-10 px-4 text-13 text-[var(--c-danger)]" @click="onCancel">
                {{ t('order.cancelOrder') }}
              </button>
            </div>
          </template>
        </template>
      </n-spin>
    </n-drawer-content>
  </n-drawer>

  <PaymentModal v-model:show="showPayment" :order="detail" :method="payMethod" @paid="onPaid" />
</template>
