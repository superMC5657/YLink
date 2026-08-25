<script setup lang="ts">
/**
 * 管理后台 · 节点管理:节点分组 CRUD + 节点 CRUD。
 * 数据:/admin/servers、/admin/server-groups(docs/api/README.md §16)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type {
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
          <button class="btn-primary h-9 px-4 text-14" @click="openNodeCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminNodes.newNode') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[980px]">
          <thead>
            <tr>
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
              <td colspan="11"><EmptyState :text="t('adminNodes.emptyNodes')" /></td>
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
  </div>
</template>
