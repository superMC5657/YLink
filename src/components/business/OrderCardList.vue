<script setup lang="ts">
/**
 * 订单卡片列表(移动端/手动切换)。状态徽章 + 关键字段纵排。
 */
import {
  formatMoney,
  formatTime,
  orderStatusBadge,
  orderStatusLabel,
  periodLabel,
} from '@/utils/format'
import type { Order } from '@/types/api'

defineProps<{ orders: Order[] }>()
const emit = defineEmits<{
  (e: 'view', order: Order): void
  (e: 'pay', order: Order): void
  (e: 'cancel', order: Order): void
}>()

function statusType(status: Order['status']) {
  // 状态元深模块同源（管理端同形）
  return orderStatusBadge(status)
}
</script>

<template>
  <div class="space-y-3">
    <div v-for="o in orders" :key="o.order_no" class="card-base card-hoverable p-4">
      <div class="flex items-center justify-between">
        <span class="text-15 font-600 text-[var(--c-text)]">{{ o.plan_name }}</span>
        <StatusBadge :type="statusType(o.status)">{{ orderStatusLabel(o.status) }}</StatusBadge>
      </div>

      <div class="mt-3 space-y-1.5 text-14">
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ $t('order.orderNo') }}</span>
          <CopyText :text="o.order_no" :max-chars="14" />
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ $t('order.period') }}</span>
          <span class="text-[var(--c-text)]">{{ periodLabel(o.period) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ $t('order.payAmount') }}</span>
          <span class="num font-600 text-[var(--c-text)]">{{ formatMoney(o.pay_amount) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ $t('order.createdAt') }}</span>
          <span class="text-[var(--c-text)]">{{ formatTime(o.created_at, false) }}</span>
        </div>
      </div>

      <div class="mt-3 flex justify-end gap-2">
        <button
          v-if="o.status === 0"
          class="rounded-[var(--r-control)] px-4 py-1.5 text-14 font-500 text-white transition-colors hover:brightness-105"
          style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          @click="emit('pay', o)"
        >
          {{ $t('order.goPay') }}
        </button>
        <button class="btn-soft-blue h-8 px-4 text-14" @click="emit('view', o)">
          {{ $t('common.viewDetail') }}
        </button>
      </div>
    </div>
  </div>
</template>
