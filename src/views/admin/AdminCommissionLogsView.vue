<script setup lang="ts">
/**
 * 管理后台 · 佣金日志:状态筛选 / 分页。
 * 数据:GET /admin/commission-logs(docs/api/README.md §16.1)
 */
import { onMounted, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminCommissionItem, CommissionStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { formatMoney, formatTime } from '@/utils/format'

const loading = ref(false)
const list = ref<AdminCommissionItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | CommissionStatus>('')

const STATUS_TEXT: Record<
  CommissionStatus,
  { text: string; type: 'warning' | 'success' | 'danger' }
> = {
  0: { text: '确认中', type: 'warning' },
  1: { text: '已发放', type: 'success' },
  2: { text: '已撤销', type: 'danger' },
}

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.commissionLogs({
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

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="佣金日志" subtitle="邀请佣金的发放与回滚记录(含邀请人与被邀请人)">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">全部</n-radio-button>
            <n-radio-button :value="0">确认中</n-radio-button>
            <n-radio-button :value="1">已发放</n-radio-button>
            <n-radio-button :value="2">已撤销</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[960px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>邀请人</th>
              <th>被邀请人</th>
              <th>订单号</th>
              <th>订单金额(元)</th>
              <th>比例</th>
              <th>佣金(元)</th>
              <th>状态</th>
              <th>确认时间</th>
              <th>创建时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in list" :key="c.id">
              <td class="num-font">{{ c.id }}</td>
              <td class="text-14">{{ c.invite_email }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ c.from_email }}</td>
              <td class="num-font text-14">{{ c.order_no }}</td>
              <td class="num-font">{{ c.order_amount.toFixed(2) }}</td>
              <td class="num-font">{{ c.rate }}%</td>
              <td class="num-font font-600 text-[var(--c-pink)]">
                {{ formatMoney(c.amount) }}
              </td>
              <td>
                <StatusBadge :type="STATUS_TEXT[c.status]?.type ?? 'neutral'">
                  {{ STATUS_TEXT[c.status]?.text ?? c.status }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ c.confirmed_at ? formatTime(c.confirmed_at) : '-' }}
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(c.created_at) }}</td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="10"><EmptyState text="暂无佣金记录" /></td>
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
