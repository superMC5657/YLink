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
  <n-table :bordered="false" :single-line="false" class="w-full">
    <thead>
      <tr>
        <th class="text-13">{{ $t('order.productName') }}</th>
        <th class="text-13">{{ $t('order.orderNo') }}</th>
        <th class="text-13">{{ $t('order.period') }}</th>
        <th class="text-13">{{ $t('order.payAmount') }}</th>
        <th class="text-13">{{ $t('order.status') }}</th>
        <th class="text-13">{{ $t('order.createdAt') }}</th>
        <th class="text-13">{{ $t('order.action') }}</th>
      </tr>
    </thead>
    <tbody>
      <tr
        v-for="o in orders"
        :key="o.order_no"
        class="transition-colors hover:bg-[var(--c-bg-hover)]"
      >
        <td class="text-14 font-500 text-[var(--c-text)]">{{ o.plan_name }}</td>
        <td><CopyText :text="o.order_no" :max-chars="16" /></td>
        <td class="text-13 text-[var(--c-text-sub)]">{{ periodLabel(o.period) }}</td>
        <td class="num text-14 font-600 text-[var(--c-text)]">{{ formatMoney(o.pay_amount) }}</td>
        <td>
          <StatusBadge :type="statusType(o.status)">{{ orderStatusLabel(o.status) }}</StatusBadge>
        </td>
        <td class="text-13 text-[var(--c-text-sub)]">{{ formatTime(o.created_at, false) }}</td>
        <td>
          <div class="flex items-center gap-2">
            <button
              class="text-13 font-500 text-[var(--c-primary-text)] hover:underline"
              @click="emit('view', o)"
            >
              {{ $t('common.viewDetail') }}
            </button>
            <button
              v-if="o.status === 0"
              class="rounded-full px-3 py-1 text-12 font-500 text-white transition-colors hover:brightness-105"
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
</template>
