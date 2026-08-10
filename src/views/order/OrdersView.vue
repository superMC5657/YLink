<script setup lang="ts">
/**
 * 我的订单(截图3):表格/卡片双视图 + 分页 + 详情弹窗 + 支付入口 + 待支付轮询。
 * 数据:GET /orders(docs/api/README.md §10.2)
 */
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useOrderStore } from '@/stores/order'
import { useMediaQuery } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import PageHeader from '@/components/ui/PageHeader.vue'
import OrderTable from '@/components/business/OrderTable.vue'
import OrderCardList from '@/components/business/OrderCardList.vue'
import OrderDetailModal from '@/components/business/OrderDetailModal.vue'
import PaymentModal from '@/components/business/PaymentModal.vue'
import type { Order } from '@/types/api'

const orderStore = useOrderStore()
const isDesktop = useMediaQuery('(min-width: 1024px)')
const { t } = useI18n()

const viewMode = ref<'table' | 'card'>('table')
const currentPage = ref(1)
const pageSize = 10
const statusFilter = ref<'' | 0 | 1 | 2 | 3>('')

const modalVisible = ref(false)
const currentOrderNo = ref<string | null>(null)
const payOrder = ref<Order | null>(null)
const showPayment = ref(false)
const payMethod = ref('epay_alipay')

async function load() {
  await orderStore.fetch({
    page: currentPage.value,
    page_size: pageSize,
    status: statusFilter.value,
  })
  // 存在待支付订单时开启轮询
  if (orderStore.list.some((o) => o.status === 0)) {
    orderStore.startPolling(() => void orderStore.fetch())
  } else {
    orderStore.stopPolling()
  }
}

function onPageChange(page: number) {
  currentPage.value = page
  void load()
}

function onStatusChange() {
  currentPage.value = 1
  void load()
}

function viewDetail(order: Order) {
  currentOrderNo.value = order.order_no
  modalVisible.value = true
}

function goPay(order: Order) {
  payOrder.value = order
  payMethod.value = 'epay_alipay'
  showPayment.value = true
}

function onPaid() {
  showPayment.value = false
  void load()
}

onMounted(() => void load())
onBeforeUnmount(() => orderStore.stopPolling())
</script>

<template>
  <div>
    <PageHeader :title="t('order.title')">
      <template #actions>
        <!-- 状态筛选 -->
        <n-select
          :value="statusFilter"
          class="w-32"
          size="small"
          :options="[
            { label: t('common.pending'), value: 0 },
            { label: t('common.completed'), value: 1 },
            { label: t('common.cancelled'), value: 2 },
            { label: t('common.refunded'), value: 3 },
          ]"
          placeholder="全部"
          clearable
          @update:value="
            (v: number | null) => {
              statusFilter = v === null ? '' : (v as 0 | 1 | 2 | 3)
              onStatusChange()
            }
          "
        />
        <!-- 视图切换 -->
        <div class="flex rounded-[var(--r-control)] border border-[var(--c-border)] p-0.5">
          <button
            class="cursor-pointer rounded-[var(--r-control)] px-3 py-1 text-14 transition-colors"
            :class="
              viewMode === 'table'
                ? 'bg-[var(--c-primary-soft)] text-[var(--c-primary-text)]'
                : 'text-[var(--c-text-sub)]'
            "
            @click="viewMode = 'table'"
          >
            {{ t('order.tableView') }}
          </button>
          <button
            class="cursor-pointer rounded-[var(--r-control)] px-3 py-1 text-14 transition-colors"
            :class="
              viewMode === 'card'
                ? 'bg-[var(--c-primary-soft)] text-[var(--c-primary-text)]'
                : 'text-[var(--c-text-sub)]'
            "
            @click="viewMode = 'card'"
          >
            {{ t('order.cardView') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base p-4 md:p-6">
      <n-spin :show="orderStore.loading">
        <OrderTable
          v-if="viewMode === 'table' && isDesktop"
          :orders="orderStore.list"
          @view="viewDetail"
          @pay="goPay"
        />
        <OrderCardList v-else :orders="orderStore.list" @view="viewDetail" @pay="goPay" />
        <EmptyState
          v-if="!orderStore.loading && orderStore.list.length === 0"
          :text="t('order.noOrders')"
          :icon="'order'"
        />
      </n-spin>

      <div v-if="orderStore.total > pageSize" class="mt-5 flex justify-center">
        <n-pagination
          :page="currentPage"
          :page-size="pageSize"
          :item-count="orderStore.total"
          @update:page="onPageChange"
        />
      </div>
    </div>

    <!-- 详情弹窗 -->
    <OrderDetailModal v-model:show="modalVisible" :order-no="currentOrderNo" @changed="load" />

    <!-- 去支付收银台 -->
    <PaymentModal v-model:show="showPayment" :order="payOrder" :method="payMethod" @paid="onPaid" />
  </div>
</template>
