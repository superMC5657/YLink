<script setup lang="ts">
/**
 * 管理后台 · 节点管理:节点分组 CRUD + 节点 CRUD + 批量操作/复制/排序(F09)。
 * 数据:/admin/servers、/admin/server-groups、/admin/servers/batch|sort|{id}/copy(docs/api/README.md §16)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type {
  AdminBatchServerResp,
  AdminServerGroupItem,
  AdminServerItem,
  AdminServerReq,
  ServerStatus,
} from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const servers = ref<AdminServerItem[]>([])
const groups = ref<AdminServerGroupItem[]>([])

// 节点新建/编辑
const nodeModal = ref(false)
const editingNode = ref<number | null>(null)
const nodeSaving = ref(false)
const nodeForm = reactive<AdminServerReq>({
  group_id: 0,
  name: '',
  type: 'shadowsocks',
  host: '',
  port: 8080,
  config: '',
  rate: 1,
  tags: [],
  status: 1,
  is_show: true,
  sort: 0,
})

// 分组新建/编辑
const groupModal = ref(false)
const editingGroup = ref<number | null>(null)
const groupSaving = ref(false)
const groupName = ref('')
const groupSort = ref(0)

const PROTOCOLS = [
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Hysteria2', value: 'hysteria2' },
  { label: 'Tuic', value: 'tuic' },
]

const STATUS_KEY: Record<ServerStatus, string> = {
  1: 'node.normal',
  2: 'node.busy',
  3: 'node.maintenance',
}

function statusText(status: ServerStatus): string {
  return t(STATUS_KEY[status])
}

function statusType(status: ServerStatus): 'success' | 'warning' | 'neutral' {
  return status === 1 ? 'success' : status === 2 ? 'warning' : 'neutral'
}

const groupNameOf = computed(() => {
  const map = new Map(groups.value.map((g) => [g.id, g.name]))
  return (id: number) => map.get(id) ?? t('adminNodes.groupFallback', { id })
})

async function load() {
  loading.value = true
  try {
    const [s, g] = await Promise.all([apiAdmin.servers(), apiAdmin.serverGroups()])
    servers.value = s.list
    groups.value = g.list
  } finally {
    loading.value = false
  }
}

function openNodeCreate() {
  editingNode.value = null
  Object.assign(nodeForm, {
    group_id: groups.value[0]?.id ?? 0,
    name: '',
    type: 'shadowsocks',
    host: '',
    port: 8080,
    config: '',
    rate: 1,
    tags: [],
    status: 1,
    is_show: true,
    sort: 0,
  })
  nodeModal.value = true
}

function openNodeEdit(srv: AdminServerItem) {
  editingNode.value = srv.id
  Object.assign(nodeForm, {
    group_id: srv.group_id,
    name: srv.name,
    type: srv.type,
    host: srv.host,
    port: srv.port,
    config: srv.config,
    rate: srv.rate,
    tags: srv.tags ?? [],
    status: srv.status,
    is_show: srv.is_show,
    sort: srv.sort,
  })
  nodeModal.value = true
}

async function saveNode() {
  if (!nodeForm.group_id) {
    message.warning(t('adminNodes.selectGroup'))
    return
  }
  if (!nodeForm.name.trim() || !nodeForm.host.trim() || !nodeForm.config.trim()) {
    message.warning(t('adminNodes.requiredFields'))
    return
  }
  nodeSaving.value = true
  try {
    if (editingNode.value === null) {
      await apiAdmin.createServer({ ...nodeForm })
      message.success(t('adminNodes.nodeCreated'))
    } else {
      await apiAdmin.updateServer(editingNode.value, { ...nodeForm })
      message.success(t('adminNodes.nodeUpdated'))
    }
    nodeModal.value = false
    void load()
  } finally {
    nodeSaving.value = false
  }
}

function removeNode(srv: AdminServerItem) {
  dialog.warning({
    title: t('adminNodes.deleteNodeTitle'),
    content: t('adminNodes.deleteNodeConfirm', { name: srv.name }),
    positiveText: t('adminNodes.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.deleteServer(srv.id)
      message.success(t('adminNodes.deleted'))
      void load()
    },
  })
}

function resetNodeKey(srv: AdminServerItem) {
  dialog.warning({
    title: t('adminNodes.resetKeyTitle'),
    content: t('adminNodes.resetKeyConfirm', { name: srv.name }),
    positiveText: t('adminNodes.resetKeyBtn'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const { node_key: key } = await apiAdmin.resetServerNodeKey(srv.id)
      message.success(t('adminNodes.resetKeySuccess', { key }))
      void load()
    },
  })
}

// ---- F09: 多选 / 批量操作 / 复制 / 排序 ----

const selectedIds = ref<number[]>([])

function isSelected(id: number): boolean {
  return selectedIds.value.includes(id)
}

function toggleRow(id: number, checked: boolean) {
  if (checked) {
    if (!selectedIds.value.includes(id)) selectedIds.value = [...selectedIds.value, id]
  } else {
    selectedIds.value = selectedIds.value.filter((v) => v !== id)
  }
}

const allChecked = computed(
  () => servers.value.length > 0 && servers.value.every((s) => selectedIds.value.includes(s.id)),
)

function toggleAll(checked: boolean) {
  selectedIds.value = checked ? servers.value.map((s) => s.id) : []
}

function clearSelection() {
  selectedIds.value = []
}

/** 批量结果汇总提示:成功 n,失败列出原因(与用户端批量同风格) */
function reportBatch(resp: AdminBatchServerResp) {
  const title = t('adminNodes.batchDone')
  if (resp.failed.length === 0) {
    message.success(`${title}: ${resp.success}`)
  } else {
    const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
    message.warning(
      `${title}: ${resp.success} / ${t('adminNodes.batchFailed')} ${resp.failed.length} — ${detail}`,
    )
  }
}

const batchSaving = ref(false)

function batchDelete() {
  const ids = [...selectedIds.value]
  dialog.warning({
    title: t('adminNodes.batchDelete'),
    content: t('adminNodes.batchDeleteConfirm', { count: ids.length }),
    positiveText: t('adminNodes.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      batchSaving.value = true
      try {
        const resp = await apiAdmin.batchServers({ action: 'delete', ids })
        reportBatch(resp)
        clearSelection()
        void load()
      } finally {
        batchSaving.value = false
      }
    },
  })
}

async function batchShow(isShow: boolean) {
  batchSaving.value = true
  try {
    const resp = await apiAdmin.batchServers({
      action: 'update',
      ids: [...selectedIds.value],
      is_show: isShow,
    })
    reportBatch(resp)
    clearSelection()
    await load()
  } finally {
    batchSaving.value = false
  }
}

// 批量修改公共字段弹窗
const batchModal = ref(false)
const batchStatus = ref<ServerStatus | null>(null)
const batchGroup = ref<number | null>(null)
const batchRate = ref<number | null>(null)

function openBatchUpdate() {
  batchStatus.value = null
  batchGroup.value = null
  batchRate.value = null
  batchModal.value = true
}

async function saveBatchUpdate() {
  const payload: {
    action: 'update'
    ids: number[]
    status?: ServerStatus
    group_id?: number
    rate?: number
  } = {
    action: 'update',
    ids: [...selectedIds.value],
  }
  let hasAny = false
  if (batchStatus.value !== null) {
    payload.status = batchStatus.value
    hasAny = true
  }
  if (batchGroup.value !== null) {
    payload.group_id = batchGroup.value
    hasAny = true
  }
  if (batchRate.value !== null) {
    payload.rate = batchRate.value
    hasAny = true
  }
  if (!hasAny) {
    message.warning(t('adminNodes.batchUpdateEmpty'))
    return
  }
  batchSaving.value = true
  try {
    const resp = await apiAdmin.batchServers(payload)
    reportBatch(resp)
    batchModal.value = false
    clearSelection()
    await load()
  } finally {
    batchSaving.value = false
  }
}

function copyNode(srv: AdminServerItem) {
  dialog.warning({
    title: t('adminNodes.copyTitle'),
    content: t('adminNodes.copyConfirm', { name: srv.name }),
    positiveText: t('adminNodes.copy'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.copyServer(srv.id)
      message.success(t('adminNodes.copied', { name: srv.name }))
      void load()
    },
  })
}

// 排序弹窗:上下移调整顺序,保存时按最终顺序写 0..n
const sortModal = ref(false)
const sortItems = ref<{ id: number; name: string }[]>([])
const sortSaving = ref(false)

function openSortModal() {
  sortItems.value = [...servers.value]
    .sort((a, b) => a.sort - b.sort || a.id - b.id)
    .map((s) => ({ id: s.id, name: s.name }))
  sortModal.value = true
}

function moveSortItem(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= sortItems.value.length) return
  const items = [...sortItems.value]
  ;[items[index], items[target]] = [items[target], items[index]]
  sortItems.value = items
}

async function saveSort() {
  sortSaving.value = true
  try {
    await apiAdmin.sortServers({
      items: sortItems.value.map((s, i) => ({ id: s.id, sort: i })),
    })
    message.success(t('adminNodes.sortSaved'))
    sortModal.value = false
    await load()
  } finally {
    sortSaving.value = false
  }
}

function openGroupCreate() {
  editingGroup.value = null
  groupName.value = ''
  groupSort.value = 0
  groupModal.value = true
}

function openGroupEdit(g: AdminServerGroupItem) {
  editingGroup.value = g.id
  groupName.value = g.name
  groupSort.value = g.sort
  groupModal.value = true
}

async function saveGroup() {
  if (!groupName.value.trim()) {
    message.warning(t('adminNodes.enterGroupName'))
    return
  }
  groupSaving.value = true
  try {
    if (editingGroup.value === null) {
      await apiAdmin.createServerGroup({ name: groupName.value, sort: groupSort.value })
      message.success(t('adminNodes.groupCreated'))
    } else {
      await apiAdmin.updateServerGroup(editingGroup.value, {
        name: groupName.value,
        sort: groupSort.value,
      })
      message.success(t('adminNodes.groupUpdated'))
    }
    groupModal.value = false
    void load()
  } finally {
    groupSaving.value = false
  }
}

function removeGroup(g: AdminServerGroupItem) {
  dialog.warning({
    title: t('adminNodes.deleteGroupTitle'),
    content: t('adminNodes.deleteGroupConfirm', { name: g.name }),
    positiveText: t('adminNodes.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.deleteServerGroup(g.id)
      message.success(t('adminNodes.deleted'))
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminNodes.pageTitle')" :subtitle="t('adminNodes.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-soft-primary h-9 px-4 text-14" @click="openGroupCreate">
            {{ t('adminNodes.newGroup') }}
          </button>
          <button class="btn-soft-primary h-9 px-4 text-14" @click="openSortModal">
            {{ t('adminNodes.sortTitle') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openNodeCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminNodes.newNode') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <!-- F09 批量操作工具栏(选中 > 0 时显示) -->
    <div v-if="selectedIds.length > 0" class="card-base mb-4 flex flex-wrap items-center gap-2 p-3">
      <span class="text-14 text-[var(--c-text-sub)]">
        {{ t('adminNodes.selectedCount', { count: selectedIds.length }) }}
      </span>
      <button
        class="btn-soft-success h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchShow(true)"
      >
        {{ t('adminNodes.batchShow') }}
      </button>
      <button
        class="btn-soft-warning h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchShow(false)"
      >
        {{ t('adminNodes.batchHide') }}
      </button>
      <button
        class="btn-soft-primary h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="openBatchUpdate"
      >
        {{ t('adminNodes.batchUpdate') }}
      </button>
      <button class="btn-soft-danger h-8 px-3 text-14" :disabled="batchSaving" @click="batchDelete">
        {{ t('adminNodes.batchDelete') }}
      </button>
      <button class="btn-soft-neutral h-8 px-3 text-14" @click="clearSelection">
        {{ t('adminNodes.clearSelection') }}
      </button>
    </div>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[1060px]">
          <thead>
            <tr>
              <th class="w-10">
                <n-checkbox :checked="allChecked" @update:checked="toggleAll" />
              </th>
              <th>ID</th>
              <th>{{ t('adminNodes.name') }}</th>
              <th>{{ t('adminNodes.group') }}</th>
              <th>{{ t('adminNodes.protocol') }}</th>
              <th>{{ t('adminNodes.host') }}</th>
              <th>{{ t('adminNodes.rate') }}</th>
              <th>{{ t('adminNodes.status') }}</th>
              <th>{{ t('adminNodes.show') }}</th>
              <th>{{ t('adminNodes.sort') }}</th>
              <th>{{ t('adminNodes.nodeKey') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in servers" :key="s.id">
              <td>
                <n-checkbox
                  :checked="isSelected(s.id)"
                  @update:checked="(v: boolean) => toggleRow(s.id, v)"
                />
              </td>
              <td>{{ s.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ s.name }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ groupNameOf(s.group_id) }}</td>
              <td class="text-14">{{ s.type }}</td>
              <td class="num-font text-14 text-[var(--c-text-sub)]">{{ s.host }}:{{ s.port }}</td>
              <td class="num-font">{{ s.rate }}</td>
              <td>
                <StatusBadge :type="statusType(s.status)">
                  {{ statusText(s.status) }}
                </StatusBadge>
              </td>
              <td>
                <StatusBadge :type="s.is_show ? 'success' : 'neutral'">
                  {{ s.is_show ? t('adminNodes.show') : t('adminNodes.hidden') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ s.sort }}</td>
              <td>
                <CopyText v-if="s.node_key" :text="s.node_key" :max-chars="10" />
                <span v-else class="text-14 text-[var(--c-text-sub)]">—</span>
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openNodeEdit(s)">
                    {{ t('adminNodes.edit') }}
                  </button>
                  <button class="btn-soft-neutral h-7 px-3 text-14" @click="copyNode(s)">
                    {{ t('adminNodes.copy') }}
                  </button>
                  <button class="btn-soft-neutral h-7 px-3 text-14" @click="resetNodeKey(s)">
                    {{ t('adminNodes.resetKey') }}
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="removeNode(s)">
                    {{ t('adminNodes.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && servers.length === 0">
              <td colspan="12"><EmptyState :text="t('adminNodes.emptyNodes')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 分组列表 -->
    <div class="card-base mt-5 p-5">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-16 font-600 text-[var(--c-text)]">{{ t('adminNodes.groupTitle') }}</h2>
        <button class="btn-soft-primary h-8 px-3 text-14" @click="openGroupCreate">
          {{ t('adminNodes.newGroup') }}
        </button>
      </div>
      <div v-if="groups.length === 0" class="py-4">
        <EmptyState :text="t('adminNodes.emptyGroups')" />
      </div>
      <div v-else class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
        <div
          v-for="g in groups"
          :key="g.id"
          class="flex items-center justify-between rounded-[var(--r-card)] border border-[var(--c-border)] px-4 py-3"
        >
          <div>
            <div class="text-14 font-500 text-[var(--c-text)]">{{ g.name }}</div>
            <div class="text-14 text-[var(--c-text-sub)]">
              {{
                t('adminNodes.nodeCount', {
                  count: servers.filter((s) => s.group_id === g.id).length,
                })
              }}
            </div>
          </div>
          <div class="flex gap-2">
            <button class="btn-soft-primary h-7 px-2.5 text-14" @click="openGroupEdit(g)">
              {{ t('adminNodes.edit') }}
            </button>
            <button class="btn-danger h-7 px-2.5 text-14" @click="removeGroup(g)">
              {{ t('adminNodes.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 节点弹窗 -->
    <n-modal
      v-model:show="nodeModal"
      preset="card"
      :title="editingNode === null ? t('adminNodes.createNode') : t('adminNodes.editNode')"
      style="width: 560px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminNodes.name')">
            <n-input v-model:value="nodeForm.name" :placeholder="t('adminNodes.namePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('adminNodes.group')">
            <n-select
              v-model:value="nodeForm.group_id"
              :options="groups.map((g) => ({ label: g.name, value: g.id }))"
            />
          </n-form-item>
          <n-form-item :label="t('adminNodes.protocol')">
            <n-select v-model:value="nodeForm.type" :options="PROTOCOLS" />
          </n-form-item>
          <n-form-item :label="t('adminNodes.rate')">
            <n-input-number v-model:value="nodeForm.rate" :min="0.1" :step="0.1" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminNodes.hostLabel')">
            <n-input v-model:value="nodeForm.host" :placeholder="t('adminNodes.hostPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('adminNodes.portLabel')">
            <n-input-number v-model:value="nodeForm.port" :min="1" :max="65535" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminNodes.status')">
            <n-select
              v-model:value="nodeForm.status"
              :options="[
                { label: t('node.normal'), value: 1 },
                { label: t('node.busy'), value: 2 },
                { label: t('node.maintenance'), value: 3 },
              ]"
            />
          </n-form-item>
          <n-form-item :label="t('adminNodes.sort')">
            <n-input-number v-model:value="nodeForm.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminNodes.tagsLabel')">
          <n-select
            v-model:value="nodeForm.tags"
            multiple
            filterable
            tag
            :options="[]"
            :placeholder="t('adminNodes.tagsPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.configLabel')">
          <n-input
            v-model:value="nodeForm.config"
            type="textarea"
            :rows="4"
            placeholder='{"password":"...","method":"aes-256-gcm"}'
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.showLabel')">
          <n-switch v-model:value="nodeForm.is_show" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="nodeModal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="nodeSaving" @click="saveNode">
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 分组弹窗 -->
    <n-modal
      v-model:show="groupModal"
      preset="card"
      :title="editingGroup === null ? t('adminNodes.createGroup') : t('adminNodes.editGroup')"
      style="width: 360px"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('adminNodes.groupNameLabel')">
          <n-input v-model:value="groupName" :placeholder="t('adminNodes.groupNamePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.sortLabel')">
          <n-input-number v-model:value="groupSort" class="w-full" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="groupModal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="groupSaving" @click="saveGroup">
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- F09 批量修改公共字段弹窗 -->
    <n-modal
      v-model:show="batchModal"
      preset="card"
      :title="t('adminNodes.batchUpdate')"
      style="width: 420px"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('adminNodes.batchUpdateHint')">
          <span class="text-14 text-[var(--c-text-sub)]">
            {{ t('adminNodes.selectedCount', { count: selectedIds.length }) }}
          </span>
        </n-form-item>
        <n-form-item :label="t('adminNodes.status')">
          <n-select
            v-model:value="batchStatus"
            clearable
            :options="[
              { label: t('node.normal'), value: 1 },
              { label: t('node.busy'), value: 2 },
              { label: t('node.maintenance'), value: 3 },
            ]"
            :placeholder="t('adminNodes.batchUnchanged')"
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.group')">
          <n-select
            v-model:value="batchGroup"
            clearable
            :options="groups.map((g) => ({ label: g.name, value: g.id }))"
            :placeholder="t('adminNodes.batchUnchanged')"
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.rate')">
          <n-input-number
            v-model:value="batchRate"
            clearable
            :min="0.1"
            :step="0.1"
            class="w-full"
            :placeholder="t('adminNodes.batchUnchanged')"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="batchModal = false">
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="batchSaving"
            @click="saveBatchUpdate"
          >
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- F09 排序弹窗 -->
    <n-modal
      v-model:show="sortModal"
      preset="card"
      :title="t('adminNodes.sortTitle')"
      style="width: 420px"
    >
      <p class="mb-3 text-14 text-[var(--c-text-sub)]">{{ t('adminNodes.sortHint') }}</p>
      <div class="flex flex-col gap-2">
        <div
          v-for="(item, i) in sortItems"
          :key="item.id"
          class="flex items-center justify-between rounded-lg border border-[var(--c-border)] px-3 py-2"
        >
          <span class="num-font mr-2 text-14 text-[var(--c-text-sub)]">{{ i + 1 }}</span>
          <span class="flex-1 truncate text-14 font-500 text-[var(--c-text)]">{{ item.name }}</span>
          <div class="flex gap-1">
            <button
              class="btn-soft-neutral h-7 px-2 text-14"
              :disabled="i === 0"
              @click="moveSortItem(i, -1)"
            >
              ↑
            </button>
            <button
              class="btn-soft-neutral h-7 px-2 text-14"
              :disabled="i === sortItems.length - 1"
              @click="moveSortItem(i, 1)"
            >
              ↓
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="sortModal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="sortSaving" @click="saveSort">
            {{ t('adminNodes.sortSave') }}
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
