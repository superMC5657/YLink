<script setup lang="ts">
/**
 * 管理后台 · 订单管理:状态筛选 / 分页 / 退款。
 * 数据:GET /admin/orders、POST /admin/orders/{no}/refund(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import { useAdminList } from '@/composables/useAdminList'
import type { AdminOrderItem, OrderStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { orderStatusBadge, orderStatusLabel, payMethodLabel, periodLabel } from '@/utils/format'
import { useMessage, useDialog } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const statusFilter = ref<'' | OrderStatus>('')
// 列表骨架归 useAdminList(C3 收敛);筛选值留视图,模板不动
const { loading, list, total, page, pageSize, load, applyFilters, onPageChange } = useAdminList<
  AdminOrderItem,
  { status: '' | OrderStatus }
>({
  fetch: (q) => apiAdmin.orders(q),
})

function statusText(status: OrderStatus): string {
  // 状态元深模块同源（用户端同形，不跨域复用键）
  return orderStatusLabel(status)
}

function statusType(status: OrderStatus): 'warning' | 'success' | 'neutral' | 'danger' {
  return orderStatusBadge(status)
}

function onFilter() {
  void applyFilters({ status: statusFilter.value })
}

// 退款弹窗状态:退款仅余额原路退回;现金部分需线下在网关侧退款并回填外部单号(审计)
const refundModal = ref(false)
const refundOrder = ref<AdminOrderItem | null>(null)
const refundGatewayId = ref('')
const refunding = ref(false)

function refund(o: AdminOrderItem) {
  if (o.status !== 1) {
    message.warning(t('adminOrders.onlyCompletedRefundable'))
    return
  }
  refundOrder.value = o
  refundGatewayId.value = ''
  refundModal.value = true
}

async function confirmRefund() {
  const o = refundOrder.value
  if (!o || refunding.value) return
  refunding.value = true
  try {
    await apiAdmin.refund(o.order_no, {
      remark: t('adminOrders.refundRemark'),
      gateway_refund_id: refundGatewayId.value.trim() || undefined,
    })
    message.success(t('adminOrders.refundSuccess'))
    refundModal.value = false
    void load()
  } finally {
    refunding.value = false
  }
}

function closeOrder(o: AdminOrderItem) {
  if (o.status !== 0) {
    message.warning(t('adminOrders.onlyPendingClosable'))
    return
  }
  dialog.warning({
    title: t('adminOrders.closeTitle'),
    content: t('adminOrders.closeConfirm', { no: o.order_no }),
    positiveText: t('adminOrders.closeBtn'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await apiAdmin.closeOrder(o.order_no, { remark: t('adminOrders.closeRemark') })
        message.success(t('adminOrders.closeSuccess'))
        void load()
      } catch (e) {
        // 错误提示由 http 层统一 toast;这里刷新列表后继续抛出,供 dialog 感知失败
        void load()
        throw e
      }
    },
  })
}

onMounted(() => void applyFilters({ status: statusFilter.value }))
</script>

<template>
  <div>
    <PageHeader :title="t('adminOrders.title')" :subtitle="t('adminOrders.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">{{ t('adminOrders.all') }}</n-radio-button>
            <n-radio-button :value="0">{{ t('common.pending') }}</n-radio-button>
            <n-radio-button :value="1">{{ t('common.completed') }}</n-radio-button>
            <n-radio-button :value="2">{{ t('common.cancelled') }}</n-radio-button>
            <n-radio-button :value="3">{{ t('common.refunded') }}</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[900px]">
          <thead>
            <tr>
              <th>{{ t('adminOrders.orderNo') }}</th>
              <th>{{ t('adminOrders.user') }}</th>
              <th>{{ t('adminOrders.plan') }}</th>
              <th>{{ t('adminOrders.period') }}</th>
              <th>{{ t('adminOrders.amount') }}</th>
              <th>{{ t('adminOrders.payAmount') }}</th>
              <th>{{ t('adminOrders.commission') }}</th>
              <th>{{ t('adminOrders.status') }}</th>
              <th>{{ t('adminOrders.payMethod') }}</th>
              <th>{{ t('adminOrders.createdAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in list" :key="o.order_no">
              <td class="num-font text-14">{{ o.order_no }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ o.user_email }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ o.plan_name }}</td>
              <td class="text-14">{{ periodLabel(o.period) }}</td>
              <td class="num-font">{{ o.amount.toFixed(2) }}</td>
              <td class="num-font font-600 text-[var(--c-olive)]">{{ o.pay_amount.toFixed(2) }}</td>
              <td
                class="num-font"
                :class="
                  o.commission_amount ? 'text-[var(--c-primary-text)]' : 'text-[var(--c-text-sub)]'
                "
              >
                {{ o.commission_amount != null ? o.commission_amount.toFixed(2) : '-' }}
              </td>
              <td>
                <StatusBadge :type="statusType(o.status)">
                  {{ statusText(o.status) }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ payMethodLabel(o.pay_method) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ o.created_at.slice(0, 19).replace('T', ' ') }}
              </td>
              <td>
                <div class="flex gap-2">
                  <button
                    v-if="o.status === 0"
                    class="btn-soft-danger h-7 px-3 text-14"
                    @click="closeOrder(o)"
                  >
                    {{ t('adminOrders.close') }}
                  </button>
                  <button
                    class="btn-soft-warning h-7 px-3 text-14"
                    :disabled="o.status !== 1"
                    :class="o.status !== 1 ? 'opacity-40' : ''"
                    @click="refund(o)"
                  >
                    {{ t('adminOrders.refund') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="11"><EmptyState :text="t('adminOrders.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
      <div class="flex justify-end p-4">
        <n-pagination
          v-model:page="page"
          :item-count="total"
          :page-size="pageSize"
          @update:page="onPageChange"
        />
      </div>
    </div>

    <!-- 退款确认弹窗:明示退款语义(仅余额原路;现金需线下网关退),外部单号回填审计 -->
    <n-modal
      v-model:show="refundModal"
      preset="card"
      :title="t('adminOrders.refundTitle')"
      class="max-w-95"
    >
      <div class="space-y-4">
        <p class="text-14 text-[var(--c-text-sub)]">
          {{
            t('adminOrders.refundConfirm', {
              no: refundOrder?.order_no ?? '',
              amount: (refundOrder?.pay_amount ?? 0).toFixed(2),
            })
          }}
        </p>
        <div
          class="flex items-start gap-2.5 rounded-xl p-3.5"
          style="background: var(--c-warning-bg)"
        >
          <AppIcon
            name="alert"
            :size="17"
            class="mt-0.5 shrink-0"
            :style="{ color: 'var(--c-warning)' }"
          />
          <p class="text-14 leading-5 text-[var(--c-text)]">
            {{ t('adminOrders.refundScopeTip') }}
          </p>
        </div>
        <div>
          <label class="mb-1 block text-14 text-[var(--c-text-sub)]">
            {{ t('adminOrders.gatewayRefundId') }}
          </label>
          <input
            v-model="refundGatewayId"
            class="h-10 w-full rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] px-3 text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
          />
        </div>
        <button class="btn-danger h-10 w-full text-14" :disabled="refunding" @click="confirmRefund">
          {{ t('adminOrders.refundBtn') }}
        </button>
      </div>
    </n-modal>
  </div>
</template>
