<script setup lang="ts">
/**
 * 订单表格(桌面视图)。数据:GET /orders(契约 §10.2)
 */
import { formatMoney, formatTime, orderStatusLabel, periodLabel } from '@/utils/format'
import type { Order } from '@/types/api'

defineProps<{ orders: Order[] }>()
const emit = defineEmits<{
  (e: 'view', order: Order): void
  (e: 'pay', order: Order): void
  (e: 'cancel', order: Order): void
}>()

function statusType(status: Order['status']) {
  const map: Record<Order['status'], 'success' | 'warning' | 'neutral' | 'danger'> = {
    0: 'warning',
    1: 'success',
    2: 'neutral',
    3: 'danger',
  }
  return map[status]
}
</script>

<template>
  <div class="w-full overflow-x-auto">
    <n-table :bordered="false" :single-line="false" class="w-full">
      <thead>
        <tr>
          <th class="text-14">{{ $t('order.productName') }}</th>
          <th class="text-14">{{ $t('order.orderNo') }}</th>
          <th class="text-14">{{ $t('order.period') }}</th>
          <th class="text-14">{{ $t('order.payAmount') }}</th>
          <th class="text-14">{{ $t('order.status') }}</th>
          <th class="text-14">{{ $t('order.createdAt') }}</th>
          <th class="text-14">{{ $t('order.action') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="o in orders"
          :key="o.order_no"
          class="transition-colors hover:bg-[var(--c-bg-hover)]"
        >
          <td class="whitespace-nowrap text-14 font-500 text-[var(--c-text)]">{{ o.plan_name }}</td>
          <td class="whitespace-nowrap"><CopyText :text="o.order_no" :max-chars="12" /></td>
          <td class="whitespace-nowrap text-14 text-[var(--c-text-sub)]">{{ periodLabel(o.period) }}</td>
          <td class="num whitespace-nowrap text-14 font-600 text-[var(--c-text)]">{{ formatMoney(o.pay_amount) }}</td>
          <td class="whitespace-nowrap">
            <StatusBadge :type="statusType(o.status)">{{ orderStatusLabel(o.status) }}</StatusBadge>
          </td>
          <td class="whitespace-nowrap text-14 text-[var(--c-text-sub)]">{{ formatTime(o.created_at, false) }}</td>
          <td class="whitespace-nowrap">
            <div class="flex items-center gap-2">
              <button
                class="text-14 font-500 text-[var(--c-primary-text)] hover:underline"
                @click="emit('view', o)"
              >
                {{ $t('common.viewDetail') }}
              </button>
              <button
                v-if="o.status === 0"
                class="rounded-[var(--r-control)] px-3 py-1 text-14 font-500 text-white transition-colors hover:brightness-105"
                style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
                @click="emit('pay', o)"
              >
                {{ $t('order.goPay') }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </n-table>
  </div>
</template>

<style scoped>
/* 桌面表格紧凑化:窄屏下减小横向占用,宽屏下更协调 */
:deep(.n-table th),
:deep(.n-table td) {
  padding: 10px 12px;
  font-size: 13px;
}
</style>
