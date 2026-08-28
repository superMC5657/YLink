<script setup lang="ts">
/**
 * 管理后台 · 佣金日志:状态筛选 / 分页。
 * 数据:GET /admin/commission-logs(docs/api/README.md §16.1)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminCommissionItem, CommissionStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { formatMoney, formatTime } from '@/utils/format'

const { t } = useI18n()
const loading = ref(false)
const list = ref<AdminCommissionItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | CommissionStatus>('')

const STATUS_KEY: Record<CommissionStatus, string> = {
  0: 'adminCommissionLogs.confirming',
  1: 'adminCommissionLogs.paid',
  2: 'adminCommissionLogs.revoked',
}

function statusText(status: CommissionStatus): string {
  return t(STATUS_KEY[status])
}

function statusType(status: CommissionStatus): 'warning' | 'success' | 'danger' {
  return status === 0 ? 'warning' : status === 1 ? 'success' : 'danger'
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
    <PageHeader
      :title="t('adminCommissionLogs.title')"
      :subtitle="t('adminCommissionLogs.subtitle')"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">{{ t('adminCommissionLogs.all') }}</n-radio-button>
            <n-radio-button :value="0">{{ t('adminCommissionLogs.confirming') }}</n-radio-button>
            <n-radio-button :value="1">{{ t('adminCommissionLogs.paid') }}</n-radio-button>
            <n-radio-button :value="2">{{ t('adminCommissionLogs.revoked') }}</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
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
              <th>{{ t('adminCommissionLogs.inviter') }}</th>
              <th>{{ t('adminCommissionLogs.invitee') }}</th>
              <th>{{ t('adminCommissionLogs.orderNo') }}</th>
              <th>{{ t('adminCommissionLogs.orderAmount') }}</th>
              <th>{{ t('adminCommissionLogs.rate') }}</th>
              <th>{{ t('adminCommissionLogs.amount') }}</th>
              <th>{{ t('adminCommissionLogs.status') }}</th>
              <th>{{ t('adminCommissionLogs.confirmedAt') }}</th>
              <th>{{ t('adminCommissionLogs.createdAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in list" :key="c.id">
              <td class="num-font">{{ c.id }}</td>
              <td class="text-14">{{ c.invite_email }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ c.from_email }}</td>
              <td class="num-font text-14">
                {{ c.order_no }}
                <StatusBadge v-if="c.type === 1" type="primary" class="ml-1">
                  {{ t('adminCommissionLogs.withdrawType') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ c.order_amount.toFixed(2) }}</td>
              <td class="num-font">{{ c.rate }}%</td>
              <td class="num-font font-600 text-[var(--c-pink)]">
                {{ formatMoney(c.amount) }}
              </td>
              <td>
                <StatusBadge :type="statusType(c.status)">
                  {{ statusText(c.status) }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ c.confirmed_at ? formatTime(c.confirmed_at) : '-' }}
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(c.created_at) }}</td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="10"><EmptyState :text="t('adminCommissionLogs.empty')" /></td>
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
