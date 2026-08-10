<script setup lang="ts">
/**
 * 管理后台 · 订单管理:状态筛选 / 分页 / 退款。
 * 数据:GET /admin/orders、POST /admin/orders/{no}/refund(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminOrderItem, OrderStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminOrderItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | OrderStatus>('')

const STATUS_TEXT: Record<
  OrderStatus,
  { text: string; type: 'warning' | 'success' | 'neutral' | 'danger' }
> = {
  0: { text: '待支付', type: 'warning' },
  1: { text: '已完成', type: 'success' },
  2: { text: '已取消', type: 'neutral' },
  3: { text: '已退款', type: 'danger' },
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
    message.warning('仅已完成订单可退款')
    return
  }
  dialog.warning({
    title: '订单退款',
    content: `确定对订单 ${o.order_no} 退款 ¥${o.pay_amount.toFixed(2)} 吗?余额将退回、佣金回滚并写入审计。`,
    positiveText: '退款',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.refund(o.order_no, { remark: '管理后台手动退款' })
      message.success('退款成功')
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="订单管理" subtitle="全量订单查询与退款(退款含佣金回滚 + 审计)">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">全部</n-radio-button>
            <n-radio-button :value="0">待支付</n-radio-button>
            <n-radio-button :value="1">已完成</n-radio-button>
            <n-radio-button :value="2">已取消</n-radio-button>
            <n-radio-button :value="3">已退款</n-radio-button>
          </n-radio-group>
          <button class="btn-ghost h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[900px]">
          <thead>
            <tr>
              <th>订单号</th>
              <th>用户</th>
              <th>套餐</th>
              <th>周期</th>
              <th>金额(元)</th>
              <th>实付(元)</th>
              <th>状态</th>
              <th>支付方式</th>
              <th>创建时间</th>
              <th class="text-right">操作</th>
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
              <td>
                <StatusBadge :type="STATUS_TEXT[o.status]?.type ?? 'neutral'">
                  {{ STATUS_TEXT[o.status]?.text ?? o.status }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ o.pay_method ?? '-' }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ o.created_at.slice(0, 19).replace('T', ' ') }}
              </td>
              <td>
                <div class="flex justify-end">
                  <button
                    class="btn-ghost h-7 px-3 text-14"
                    :disabled="o.status !== 1"
                    :class="o.status !== 1 ? 'opacity-40' : ''"
                    @click="refund(o)"
                  >
                    退款
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="10"><EmptyState text="暂无订单" /></td>
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
