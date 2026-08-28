<script setup lang="ts">
/**
 * 管理后台 · 审计日志(F08):只读查询,按操作人/动作/目标/时间范围筛选 + 分页 + 变更明细。
 * 数据:GET /admin/audit-logs(docs/api/README.md §16)。
 * 可读化:动作代码经 i18n 映射为文案,目标由后端解析为实体类型+名称(target_kind/target_display),
 * 失败或未收录时回退原始 action/target;筛选下拉与详情弹窗同规则。
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminAuditLogItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { formatTime, localDateKey } from '@/utils/format'

const { t, te } = useI18n()
const loading = ref(false)
const list = ref<AdminAuditLogItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const actions = ref<string[]>([])

/** 动作代码 → 人类可读文案(未收录的动作回退显示原始代码) */
function actionLabel(action: string): string {
  const key = `adminAuditLogs.actions.${action}`
  return te(key) ? t(key) : action
}

/**
 * 目标 → 人类可读文案:类型前缀 + 后端解析出的名称,如「用户 a@b.com」「节点 HK-01」;
 * 无类型(空 target/未收录动作)或名称解析失败时回退原始 target(裸 ID)。
 */
function targetLabel(log: AdminAuditLogItem): string {
  const raw = log.target ?? ''
  if (!log.target_kind) return raw || '-'
  const kindKey = `adminAuditLogs.targetKinds.${log.target_kind}`
  const kind = te(kindKey) ? t(kindKey) : log.target_kind
  return `${kind} ${log.target_display || `#${raw}`}`
}

// 筛选条件
const adminIdFilter = ref<number | null>(null)
const actionFilter = ref<string | null>(null)
const targetFilter = ref('')
const rangeFilter = ref<[number, number] | null>(null)

// 详情弹窗
const detailModal = ref(false)
const detailItem = ref<AdminAuditLogItem | null>(null)

/** detail jsonb 字符串美化(供详情弹窗展示) */
function prettyDetail(detail: string | null): string {
  if (!detail) return '-'
  try {
    return JSON.stringify(JSON.parse(detail), null, 2)
  } catch {
    return detail
  }
}

function openDetail(log: AdminAuditLogItem) {
  detailItem.value = log
  detailModal.value = true
}

function onPageSizeChange(ps: number) {
  pageSize.value = ps
  page.value = 1
  void load()
}

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.auditLogs({
      page: page.value,
      page_size: pageSize.value,
      admin_id: adminIdFilter.value ?? '',
      action: actionFilter.value ?? '',
      target: targetFilter.value || '',
      from: rangeFilter.value ? localDateKey(rangeFilter.value[0]) : '',
      to: rangeFilter.value ? localDateKey(rangeFilter.value[1]) : '',
    })
    list.value = res.list
    total.value = res.total
    if (res.actions?.length) actions.value = res.actions
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
    <PageHeader :title="t('adminAuditLogs.title')" :subtitle="t('adminAuditLogs.subtitle')">
      <template #actions>
        <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
          <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
        </button>
      </template>
    </PageHeader>

    <!-- 筛选栏 -->
    <div class="card-base mb-4 flex flex-wrap items-center gap-2 p-3">
      <n-input-number
        v-model:value="adminIdFilter"
        :placeholder="t('adminAuditLogs.adminId')"
        :show-button="false"
        class="w-40"
        clearable
        @update:value="onFilter"
      />
      <n-select
        v-model:value="actionFilter"
        :options="actions.map((a) => ({ label: actionLabel(a), value: a }))"
        :placeholder="t('adminAuditLogs.action')"
        class="w-52"
        clearable
        filterable
        tag
        @update:value="onFilter"
      />
      <n-input
        v-model:value="targetFilter"
        :placeholder="t('adminAuditLogs.target')"
        class="w-44!"
        clearable
        @keyup.enter="onFilter"
        @clear="onFilter"
      />
      <!-- w-44! 需 important:naive-ui 运行时注入的 .n-input{width:100%} 会覆盖同特异性的
           uno.css 宽度类,导致输入框撑满整行独占一行(n-input-number/select/date-picker 无此默认值) -->
      <n-date-picker
        v-model:value="rangeFilter"
        type="daterange"
        clearable
        class="w-72"
        @update:value="onFilter"
      />
      <button class="btn-primary h-9 px-4 text-14" @click="onFilter">
        {{ t('common.search') }}
      </button>
    </div>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[920px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('adminAuditLogs.operator') }}</th>
              <th>{{ t('adminAuditLogs.action') }}</th>
              <th>{{ t('adminAuditLogs.target') }}</th>
              <th>{{ t('adminAuditLogs.ip') }}</th>
              <th>{{ t('adminAuditLogs.createdAt') }}</th>
              <th>{{ t('adminAuditLogs.detail') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in list" :key="log.id">
              <td class="num-font">{{ log.id }}</td>
              <td class="text-14">
                <span class="text-[var(--c-text)]">{{ log.admin_email }}</span>
                <span class="ml-1 num-font text-12 text-[var(--c-text-sub)]"
                  >#{{ log.admin_id }}</span
                >
              </td>
              <td class="text-14" :title="log.action">{{ actionLabel(log.action) }}</td>
              <td class="text-14">{{ targetLabel(log) }}</td>
              <td class="num-font text-14 text-[var(--c-text-sub)]">{{ log.ip ?? '-' }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(log.created_at) }}</td>
              <td>
                <button
                  class="text-13 text-[var(--c-primary)] hover:underline"
                  :disabled="!log.detail"
                  @click="openDetail(log)"
                >
                  {{ t('adminAuditLogs.viewDetail') }}
                </button>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="7"><EmptyState :text="t('adminAuditLogs.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
      <div class="flex justify-end p-4">
        <n-pagination
          v-model:page="page"
          :item-count="total"
          :page-size="pageSize"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="onPageChange"
          @update:page-size="onPageSizeChange"
        />
      </div>
    </div>

    <!-- 变更明细弹窗 -->
    <n-modal
      v-model:show="detailModal"
      preset="card"
      :title="t('adminAuditLogs.detailTitle')"
      class="w-[560px] max-w-[92vw]"
    >
      <div v-if="detailItem" class="space-y-2 text-14">
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ t('adminAuditLogs.operator') }}</span>
          <span>{{ detailItem.admin_email }} #{{ detailItem.admin_id }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ t('adminAuditLogs.action') }}</span>
          <span :title="detailItem.action">{{ actionLabel(detailItem.action) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ t('adminAuditLogs.target') }}</span>
          <span>{{ targetLabel(detailItem) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ t('adminAuditLogs.ip') }}</span>
          <span class="num-font">{{ detailItem.ip ?? '-' }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-[var(--c-text-sub)]">{{ t('adminAuditLogs.createdAt') }}</span>
          <span>{{ formatTime(detailItem.created_at) }}</span>
        </div>
        <div>
          <div class="mb-1 text-[var(--c-text-sub)]">{{ t('adminAuditLogs.detail') }}</div>
          <pre class="overflow-x-auto rounded bg-[var(--c-bg-mute)] p-3 text-12 leading-relaxed">{{
            prettyDetail(detailItem.detail)
          }}</pre>
        </div>
      </div>
    </n-modal>
  </div>
</template>
