<script setup lang="ts">
/**
 * 支付方式选择器(下单确认/订单详情待支付两入口共享):
 * 渠道列表唯一来源为 config.payment_methods(禁用渠道不显示,启用渠道不缺失),
 * 余额渠道按「应付额 vs 用户余额」禁用并提示差额;展示名/图标沿用 config 下发值。
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '@/stores/config'
import { useUserStore } from '@/stores/user'
import { formatMoney } from '@/utils/format'
import AppIcon from '@/components/ui/AppIcon.vue'

const props = defineProps<{
  /** 当前选中渠道 code */
  modelValue: string
  /** 应付额(元),用于余额渠道可用性判断 */
  amount: number
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const config = useConfigStore()
const userStore = useUserStore()
const { t } = useI18n()

const balanceEnough = computed(() => userStore.balance >= props.amount)

/** 渠道来自 config(禁用不显示);余额不足时禁用而非隐藏,提示差额 */
const availableMethods = computed(() =>
  config.paymentMethods
    .filter((m) => m.enabled)
    .map((m) => ({ ...m, disabled: m.code === 'balance' && !balanceEnough.value })),
)
</script>

<template>
  <div>
    <div class="grid grid-cols-3 gap-2">
      <button
        v-for="m in availableMethods"
        :key="m.code"
        type="button"
        class="flex cursor-pointer flex-col items-center gap-1 rounded-xl border py-3 transition-colors"
        :class="[
          modelValue === m.code
            ? 'border-[var(--c-primary)] bg-[var(--c-primary-soft)]'
            : 'border-[var(--c-border)]',
          m.disabled ? 'cursor-not-allowed opacity-50' : 'hover:bg-[var(--c-bg-hover)]',
        ]"
        :disabled="m.disabled"
        @click="emit('update:modelValue', m.code)"
      >
        <AppIcon
          :name="m.icon === 'wallet' ? 'wallet' : 'credit'"
          :size="18"
          :style="{ color: modelValue === m.code ? 'var(--c-primary)' : 'var(--c-text-sub)' }"
        />
        <span
          class="text-14"
          :style="{ color: modelValue === m.code ? 'var(--c-primary-text)' : 'var(--c-text-sub)' }"
        >
          {{ m.name }}
        </span>
      </button>
    </div>
    <p
      v-if="modelValue === 'balance' && !balanceEnough"
      class="mt-1.5 text-14 text-[var(--c-danger)]"
    >
      {{ t('plan.balanceShort', { amount: formatMoney(Math.max(0, amount - userStore.balance)) }) }}
    </p>
  </div>
</template>
