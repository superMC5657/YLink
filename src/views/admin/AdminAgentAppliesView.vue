<script setup lang="ts">
/**
 * 管理后台 · 代理审批:状态筛选 / 分页 / 通过 / 拒绝。
 * 数据:GET /admin/agent/applies、POST /admin/agent/applies/{id}/approve|reject(docs/api/README.md §16.1)
 */
import { onMounted, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminAgentApplyItem, AdminApplyStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminAgentApplyItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const statusFilter = ref<'' | AdminApplyStatus>('')

const STATUS_TEXT: Record<
  AdminApplyStatus,
  { text: string; type: 'warning' | 'success' | 'danger' }
> = {
  0: { text: '待审核', type: 'warning' },
  1: { text: '已通过', type: 'success' },
  2: { text: '已拒绝', type: 'danger' },
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
    title: approve ? '通过申请' : '拒绝申请',
    content: `确定${approve ? '通过' : '拒绝'}用户 ${a.user_email} 的代理申请吗?${
      approve ? '通过后该用户立即成为代理商。' : ''
    }`,
    positiveText: approve ? '通过' : '拒绝',
    negativeText: '取消',
    onPositiveClick: async () => {
      const remark = approve ? '管理后台审核通过' : '管理后台审核拒绝'
      if (approve) {
        await apiAdmin.approveAgent(a.id, { remark })
      } else {
        await apiAdmin.rejectAgent(a.id, { remark })
      }
      message.success(approve ? '已通过' : '已拒绝')
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="代理审批" subtitle="用户代理申请审核(通过后自动升级为代理商)">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-radio-group v-model:value="statusFilter" @update:value="onFilter">
            <n-radio-button :value="''">全部</n-radio-button>
            <n-radio-button :value="0">待审核</n-radio-button>
            <n-radio-button :value="1">已通过</n-radio-button>
            <n-radio-button :value="2">已拒绝</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
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
              <th>用户</th>
              <th>有效邀请</th>
              <th>状态</th>
              <th>申请时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in list" :key="a.id">
              <td class="num-font">{{ a.id }}</td>
              <td class="text-14 text-[var(--c-text)]">{{ a.user_email }}</td>
              <td class="num-font">{{ a.valid_invites }}</td>
              <td>
                <StatusBadge :type="STATUS_TEXT[a.status]?.type ?? 'neutral'">
                  {{ STATUS_TEXT[a.status]?.text ?? a.status }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(a.created_at) }}</td>
              <td>
                <div v-if="a.status === 0" class="flex gap-2">
                  <button class="btn-soft-success h-7 px-3 text-14" @click="review(a, true)">
                    通过
                  </button>
                  <button class="btn-soft-danger h-7 px-3 text-14" @click="review(a, false)">
                    拒绝
                  </button>
                </div>
                <span v-else class="text-14 text-[var(--c-text-sub)]">已处理</span>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="6"><EmptyState text="暂无代理申请" /></td>
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
