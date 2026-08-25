<script setup lang="ts">
/**
 * 管理后台 · 代理审批:状态筛选 / 分页 / 通过 / 拒绝。
 * 数据:GET /admin/agent/applies、POST /admin/agent/applies/{id}/approve|reject(docs/api/README.md §16.1)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminAgentApplyItem, AdminApplyStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminAgentApplyItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | AdminApplyStatus>('')

const STATUS_KEY: Record<AdminApplyStatus, string> = {
  0: 'adminAgentApplies.pending',
  1: 'adminAgentApplies.approved',
  2: 'adminAgentApplies.rejected',
}

function statusText(status: AdminApplyStatus): string {
  return t(STATUS_KEY[status])
}

function statusType(status: AdminApplyStatus): 'warning' | 'success' | 'danger' {
  return status === 0 ? 'warning' : status === 1 ? 'success' : 'danger'
}

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.agentApplies({
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

function review(a: AdminAgentApplyItem, approve: boolean) {
  dialog.warning({
    title: approve ? t('adminAgentApplies.approveTitle') : t('adminAgentApplies.rejectTitle'),
    content: t('adminAgentApplies.reviewConfirm', {
      action: approve ? t('adminAgentApplies.reviewApprove') : t('adminAgentApplies.reviewReject'),
      email: a.user_email,
      suffix: approve ? t('adminAgentApplies.approvedSuffix') : '',
    }),
    positiveText: approve
      ? t('adminAgentApplies.reviewApprove')
      : t('adminAgentApplies.reviewReject'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const remark = approve
        ? t('adminAgentApplies.remarkApprove')
        : t('adminAgentApplies.remarkReject')
      if (approve) {
        await apiAdmin.approveAgent(a.id, { remark })
      } else {
        await apiAdmin.rejectAgent(a.id, { remark })
      }
      message.success(
        approve ? t('adminAgentApplies.approvedMsg') : t('adminAgentApplies.rejectedMsg'),
      )
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader
      :title="t('adminAgentApplies.pageTitle')"
      :subtitle="t('adminAgentApplies.subtitle')"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">{{ t('adminAgentApplies.all') }}</n-radio-button>
            <n-radio-button :value="0">{{ t('adminAgentApplies.pending') }}</n-radio-button>
            <n-radio-button :value="1">{{ t('adminAgentApplies.approved') }}</n-radio-button>
            <n-radio-button :value="2">{{ t('adminAgentApplies.rejected') }}</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[760px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('adminAgentApplies.user') }}</th>
              <th>{{ t('adminAgentApplies.validInvites') }}</th>
              <th>{{ t('adminAgentApplies.status') }}</th>
              <th>{{ t('adminAgentApplies.appliedAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in list" :key="a.id">
              <td class="num-font">{{ a.id }}</td>
              <td class="text-14 text-[var(--c-text)]">{{ a.user_email }}</td>
              <td class="num-font">{{ a.valid_invites }}</td>
              <td>
                <StatusBadge :type="statusType(a.status)">
                  {{ statusText(a.status) }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(a.created_at) }}</td>
              <td>
                <div v-if="a.status === 0" class="flex gap-2">
                  <button class="btn-soft-success h-7 px-3 text-14" @click="review(a, true)">
                    {{ t('adminAgentApplies.approve') }}
                  </button>
                  <button class="btn-soft-danger h-7 px-3 text-14" @click="review(a, false)">
                    {{ t('adminAgentApplies.reject') }}
                  </button>
                </div>
                <span v-else class="text-14 text-[var(--c-text-sub)]">
                  {{ t('adminAgentApplies.processed') }}
                </span>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="6"><EmptyState :text="t('adminAgentApplies.empty')" /></td>
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
