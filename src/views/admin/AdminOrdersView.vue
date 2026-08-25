<script setup lang="ts">
/**
 * 管理后台 · 订单管理:状态筛选 / 分页 / 退款。
 * 数据:GET /admin/orders、POST /admin/orders/{no}/refund(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminOrderItem, OrderStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminOrderItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | OrderStatus>('')

const STATUS_KEY: Record<OrderStatus, string> = {
  0: 'common.pending',
  1: 'common.completed',
  2: 'common.cancelled',
  3: 'common.refunded',
}

function statusText(status: OrderStatus): string {
  return t(STATUS_KEY[status])
}

function statusType(status: OrderStatus): 'warning' | 'success' | 'neutral' | 'danger' {
  return status === 0 ? 'warning' : status === 1 ? 'success' : status === 2 ? 'neutral' : 'danger'
}

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.orders({
      page: page.value,
      page_size: pageSize,
      status: statusFilter.value,
    })
    list.value = res.list
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function onFilter() {
  page.value = 1
  void load()
}

function onPageChange(p: number) {
  page.value = p
  void load()
}

function refund(o: AdminOrderItem) {
  if (o.status !== 1) {
    message.warning(t('adminOrders.onlyCompletedRefundable'))
    return
  }
  dialog.warning({
    title: t('adminOrders.refundTitle'),
    content: t('adminOrders.refundConfirm', {
      no: o.order_no,
      amount: o.pay_amount.toFixed(2),
    }),
    positiveText: t('adminOrders.refundBtn'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await apiAdmin.refund(o.order_no, { remark: t('adminOrders.refundRemark') })
        message.success(t('adminOrders.refundSuccess'))
        void load()
      } catch (e) {
        message.error((e as Error).message)
        void load()
        throw e
      }
    },
  })
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
        message.error((e as Error).message)
        void load()
        throw e
      }
    },
  })
}

onMounted(() => void load())
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
              <td class="text-14">{{ o.period }}</td>
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
              <td class="text-14 text-[var(--c-text-sub)]">{{ o.pay_method ?? '-' }}</td>
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
  </div>
</template>
