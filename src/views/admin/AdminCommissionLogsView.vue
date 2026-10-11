<script setup lang="ts">
/**
 * 管理后台 · 佣金日志:状态筛选 / 分页。
 * 数据:GET /admin/commission-logs(docs/api/README.md §16.1)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import { useAdminList } from '@/composables/useAdminList'
import type { AdminCommissionItem, CommissionStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import {
  commissionStatusBadge,
  commissionStatusLabel,
  formatMoney,
  formatTime,
} from '@/utils/format'

const { t } = useI18n()
const statusFilter = ref<'' | CommissionStatus>('')
// 列表骨架归 useAdminList(C3 收敛);筛选值留视图,模板不动
const { loading, list, total, page, pageSize, load, applyFilters, onPageChange } = useAdminList<
  AdminCommissionItem,
  { status: '' | CommissionStatus }
>({
  fetch: (q) => apiAdmin.commissionLogs(q),
})

function statusText(status: CommissionStatus): string {
  // 状态元深模块同源（文案与徽标同源，不跨域复用）
  return commissionStatusLabel(status)
}

function statusType(status: CommissionStatus): 'warning' | 'success' | 'danger' {
  return commissionStatusBadge(status)
}

function onFilter() {
  void applyFilters({ status: statusFilter.value })
}

onMounted(() => void applyFilters({ status: statusFilter.value }))
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
