<script setup lang="ts">
/**
 * 下单确认弹窗:周期选择(省 N%)→ 优惠券 → 支付方式 → 费用明细 → 提交。
 * 数据:POST /coupons/check、POST /orders、POST /orders/{no}/checkout(契约 §9/§10)
 * 页面:docs/frontend/pages.md §3.8
 */
import { computed, ref, watch } from 'vue'
import { useOrderStore } from '@/stores/order'
import { useUserStore } from '@/stores/user'
import { useConfigStore } from '@/stores/config'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { formatMoney, planSavePercent } from '@/utils/format'
import PaymentModal from './PaymentModal.vue'
import type { CouponItem, Plan, PlanPeriod } from '@/types/api'

const props = defineProps<{
  show: boolean
  plan: Plan | null
}>()

const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'paid'): void }>()

const orderStore = useOrderStore()
const userStore = useUserStore()
const config = useConfigStore()
const message = useMessage()
const { t } = useI18n()

const period = ref<PlanPeriod>('month')
const couponCode = ref('')
const couponChecked = ref(false)
const couponResult = ref<{ discount_amount: number; pay_amount: number } | null>(null)
const couponError = ref('')
const availableCoupons = ref<CouponItem[]>([])
const couponsLoading = ref(false)
const payMethod = ref('epay_alipay')
const submitting = ref(false)

const createdOrder = ref<Awaited<ReturnType<typeof orderStore.create>> | null>(null)
const showPayment = ref(false)

const periods = computed<PlanPeriod[]>(() =>
  props.plan ? (Object.keys(props.plan.prices) as PlanPeriod[]) : [],
)

const price = computed(() => props.plan?.prices[period.value] ?? 0)

const savePercent = computed(() => (props.plan ? planSavePercent(props.plan, period.value) : null))

const discount = computed(() =>
  couponChecked.value && couponResult.value ? couponResult.value.discount_amount : 0,
)
const payAmount = computed(() => Math.max(0, +(price.value - discount.value).toFixed(2)))

const balance = computed(() => userStore.balance)
const balanceEnough = computed(() => balance.value >= payAmount.value)

const availableMethods = computed(() => {
  const methods = config.paymentMethods.filter((m) => m.enabled)
  return methods.map((m) => ({ ...m, disabled: m.code === 'balance' && !balanceEnough.value }))
})

const idempotencyKey = ref('')

function reset() {
  period.value = 'month'
  couponCode.value = ''
  couponChecked.value = false
  couponResult.value = null
  couponError.value = ''
  availableCoupons.value = []
  couponsLoading.value = false
  payMethod.value = 'epay_alipay'
  createdOrder.value = null
  showPayment.value = false
  // 幂等键:弹窗生命周期内复用(契约 §1.5)
  idempotencyKey.value = crypto.randomUUID()
}

/** 拉取当前套餐+周期的可用优惠券（展示用，不校验） */
async function loadAvailableCoupons() {
  if (!props.plan) return
  couponsLoading.value = true
  try {
    availableCoupons.value = await orderStore.fetchAvailableCoupons(props.plan.id, period.value)
  } catch {
    availableCoupons.value = []
  } finally {
    couponsLoading.value = false
  }
}

watch(
  () => props.show,
  (v) => {
    if (v) {
      reset()
      void loadAvailableCoupons()
    }
  },
)

function selectPeriod(p: PlanPeriod) {
  period.value = p
  couponChecked.value = false
  couponResult.value = null
  couponError.value = ''
  // 切换周期后可用券集合可能变化，重新拉取
  void loadAvailableCoupons()
}

/** 点选可用券：填入输入框并立即试算 */
function applyCoupon(c: CouponItem) {
  couponCode.value = c.code
  void checkCoupon()
}

/** 券的展示描述：减 ¥x / -x% / 满减门槛 */
function couponDesc(c: CouponItem): string {
  const amount = c.type === 1 ? `${formatMoney(c.value)}` : `-${c.value}%`
  const gate =
    c.min_spend > 0 ? `(${t('plan.couponGate', { amount: formatMoney(c.min_spend) })})` : ''
  return `${c.code} ${amount}${gate}`
}

async function checkCoupon() {
  if (!couponCode.value.trim()) {
    couponError.value = t('plan.couponInvalid')
    couponChecked.value = false
    return
  }
  try {
    const resp = await orderStore.checkCoupon({
      code: couponCode.value.trim(),
      plan_id: props.plan!.id,
      period: period.value,
    })
    couponChecked.value = true
    couponResult.value = { discount_amount: resp.discount_amount, pay_amount: resp.pay_amount }
    couponError.value = ''
  } catch (e) {
    couponChecked.value = false
    couponResult.value = null
    couponError.value = (e as Error).message
  }
}

async function submit() {
  if (submitting.value || !props.plan) return
  submitting.value = true
  try {
    const order = await orderStore.create(
      {
        plan_id: props.plan.id,
        period: period.value,
        coupon_code: couponChecked.value ? couponCode.value.trim() : null,
      },
      idempotencyKey.value,
    )
    createdOrder.value = order
    message.success(t('plan.orderCreated'))

    // 余额支付:直接完成
    if (payMethod.value === 'balance') {
      const resp = await orderStore.checkout(order.order_no, { method: 'balance' })
      if (resp.type === 'paid') {
        message.success(t('plan.paySuccess'))
        void userStore.refreshDashboard()
        emit('paid')
        emit('update:show', false)
        return
      }
    }
    // 在线支付:进入收银台
    showPayment.value = true
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="props.plan?.name"
    class="max-w-110"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <template v-if="props.plan">
      <!-- 1. 周期 -->
      <div class="mb-2 text-14 font-500 text-[var(--c-text)]">{{ t('plan.period') }}</div>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="p in periods"
          :key="p"
          class="flex cursor-pointer items-center gap-1.5 rounded-[var(--r-control)] border px-4 py-2 text-14 transition-colors"
          :class="
            period === p
              ? 'border-[var(--c-primary)] bg-[var(--c-primary-soft)] font-500 text-[var(--c-primary-text)]'
              : 'border-[var(--c-border)] text-[var(--c-text-sub)] hover:border-[var(--c-primary)]'
          "
          @click="selectPeriod(p)"
        >
          <span>
            {{
              p === 'month'
                ? '月付'
                : p === 'quarter'
                  ? '季付'
                  : p === 'half_year'
                    ? '半年付'
                    : p === 'year'
                      ? '年付'
                      : '一次性'
            }}
          </span>
          <span class="num">{{ formatMoney(props.plan.prices[p] ?? 0) }}</span>
          <span v-if="period === p && savePercent" class="text-14 text-[var(--c-marketing)]">
            {{ t('plan.savePercent', { n: savePercent }) }}
          </span>
        </button>
      </div>

      <!-- 2. 优惠券 -->
      <div class="mt-5 mb-2 text-14 font-500 text-[var(--c-text)]">{{ t('plan.coupon') }}</div>
      <div class="flex gap-2">
        <input
          v-model="couponCode"
          type="text"
          :placeholder="t('plan.couponPlaceholder')"
          class="h-10 flex-1 rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] px-3 text-14 text-[var(--c-text)] outline-none transition-colors placeholder:text-[var(--c-text-sub)] focus:border-[var(--c-primary)]"
        />
        <button class="btn-soft-blue h-10 px-4 text-14" @click="checkCoupon">
          {{ t('plan.couponCheck') }}
        </button>
      </div>
      <p v-if="couponChecked && couponResult" class="mt-1.5 text-14 text-[var(--c-success)]">
        {{ t('plan.couponApplied', { amount: formatMoney(couponResult.discount_amount) }) }}
      </p>
      <p v-if="couponError" class="mt-1.5 text-14 text-[var(--c-danger)]">{{ couponError }}</p>

      <!-- 可用优惠券（GET /coupons/available）：点选自动填入并试算 -->
      <div v-if="couponsLoading" class="mt-2 text-13 text-[var(--c-text-sub)]">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="availableCoupons.length" class="mt-2">
        <div class="mb-1.5 text-13 text-[var(--c-text-sub)]">{{ t('plan.availableCoupons') }}</div>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="c in availableCoupons"
            :key="c.code"
            class="cursor-pointer rounded-[var(--r-pill)] border border-[var(--c-border)] bg-[var(--c-bg-card)] px-2.5 py-1 text-13 text-[var(--c-text-sub)] transition-colors hover:border-[var(--c-primary)] hover:text-[var(--c-primary-text)]"
            :class="
              couponCode === c.code ? 'border-[var(--c-primary)] text-[var(--c-primary-text)]' : ''
            "
            @click="applyCoupon(c)"
          >
            {{ couponDesc(c) }}
          </button>
        </div>
      </div>

      <!-- 3. 支付方式 -->
      <div class="mt-5 mb-2 text-14 font-500 text-[var(--c-text)]">{{ t('plan.payment') }}</div>
      <div class="grid grid-cols-3 gap-2">
        <button
          v-for="m in availableMethods"
          :key="m.code"
          class="flex cursor-pointer flex-col items-center gap-1 rounded-xl border py-3 transition-colors"
          :class="[
            payMethod === m.code
              ? 'border-[var(--c-primary)] bg-[var(--c-primary-soft)]'
              : 'border-[var(--c-border)]',
            m.disabled ? 'cursor-not-allowed opacity-50' : 'hover:bg-[var(--c-bg-hover)]',
          ]"
          :disabled="m.disabled"
          @click="payMethod = m.code"
        >
          <AppIcon
            :name="m.icon === 'wallet' ? 'wallet' : 'credit'"
            :size="18"
            :style="{ color: payMethod === m.code ? 'var(--c-primary)' : 'var(--c-text-sub)' }"
          />
          <span
            class="text-14"
            :style="{ color: payMethod === m.code ? 'var(--c-primary-text)' : 'var(--c-text-sub)' }"
          >
            {{ m.name }}
          </span>
        </button>
      </div>
      <p
        v-if="payMethod === 'balance' && !balanceEnough"
        class="mt-1.5 text-14 text-[var(--c-danger)]"
      >
        {{ t('plan.balanceShort', { amount: formatMoney(Math.max(0, payAmount - balance)) }) }}
      </p>

      <!-- 4. 费用明细 -->
      <div class="mt-5 rounded-xl p-4" style="background-color: var(--c-bg-hover)">
        <div class="mb-2 text-14 font-500 text-[var(--c-text)]">{{ t('plan.feeDetail') }}</div>
        <div class="flex justify-between text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('plan.planPrice') }}</span>
          <span class="num text-[var(--c-text)]">{{ formatMoney(price) }}</span>
        </div>
        <div v-if="discount > 0" class="mt-1.5 flex justify-between text-14">
          <span class="text-[var(--c-text-sub)]">{{ t('order.discount') }}</span>
          <span class="num text-[var(--c-marketing)]">-{{ formatMoney(discount) }}</span>
        </div>
        <div class="mt-3 flex justify-between border-t border-[var(--c-border)] pt-3">
          <span class="text-14 font-500 text-[var(--c-text)]">{{ t('plan.total') }}</span>
          <PriceText :value="payAmount" :size="24" />
        </div>
      </div>

      <!-- 5. 提交 -->
      <button class="btn-primary mt-5 h-11 w-full text-15" :disabled="submitting" @click="submit">
        <span
          v-if="submitting"
          class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
        />
        {{ t('plan.submitOrder') }} · {{ formatMoney(payAmount) }}
      </button>
    </template>

    <!-- 收银台 -->
    <PaymentModal
      v-model:show="showPayment"
      :order="createdOrder"
      :method="payMethod"
      @paid="emit('paid')"
    />
  </n-modal>
</template>
