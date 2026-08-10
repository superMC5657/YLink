<script setup lang="ts">
/**
 * 管理后台 · 节点管理:节点分组 CRUD + 节点 CRUD。
 * 数据:/admin/servers、/admin/server-groups(docs/api/README.md §16)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type {
  AdminServerGroupItem,
  AdminServerItem,
  AdminServerReq,
  ServerStatus,
} from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

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

const STATUS_LABEL: Record<
  ServerStatus,
  { text: string; type: 'success' | 'warning' | 'neutral' }
> = {
  1: { text: '正常', type: 'success' },
  2: { text: '拥挤', type: 'warning' },
  3: { text: '维护', type: 'neutral' },
}

const groupNameOf = computed(() => {
  const map = new Map(groups.value.map((g) => [g.id, g.name]))
  return (id: number) => map.get(id) ?? `分组#${id}`
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
    message.warning('请选择节点分组')
    return
  }
  if (!nodeForm.name.trim() || !nodeForm.host.trim() || !nodeForm.config.trim()) {
    message.warning('名称 / 地址 / 协议配置必填')
    return
  }
  nodeSaving.value = true
  try {
    if (editingNode.value === null) {
      await apiAdmin.createServer({ ...nodeForm })
      message.success('节点已创建')
    } else {
      await apiAdmin.updateServer(editingNode.value, { ...nodeForm })
      message.success('节点已更新')
    }
    nodeModal.value = false
    void load()
  } finally {
    nodeSaving.value = false
  }
}

function removeNode(srv: AdminServerItem) {
  dialog.warning({
    title: '删除节点',
    content: `确定删除节点「${srv.name}」吗?`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.deleteServer(srv.id)
      message.success('已删除')
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
    message.warning('请输入分组名称')
    return
  }
  groupSaving.value = true
  try {
    if (editingGroup.value === null) {
      await apiAdmin.createServerGroup({ name: groupName.value, sort: groupSort.value })
      message.success('分组已创建')
    } else {
      await apiAdmin.updateServerGroup(editingGroup.value, {
        name: groupName.value,
        sort: groupSort.value,
      })
      message.success('分组已更新')
    }
    groupModal.value = false
    void load()
  } finally {
    groupSaving.value = false
  }
}

function removeGroup(g: AdminServerGroupItem) {
  dialog.warning({
    title: '删除分组',
    content: `确定删除分组「${g.name}」吗?组内节点需先移除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.deleteServerGroup(g.id)
      message.success('已删除')
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="节点管理" subtitle="节点分组与节点 CRUD(配置将下发到订阅)">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
          <button class="btn-soft-primary h-9 px-4 text-14" @click="openGroupCreate">新建分组</button>
          <button class="btn-primary h-9 px-4 text-14" @click="openNodeCreate">
            <AppIcon name="plus" :size="15" /> 新建节点
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[880px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>分组</th>
              <th>协议</th>
              <th>地址</th>
              <th>倍率</th>
              <th>状态</th>
              <th>展示</th>
              <th>排序</th>
              <th>操作</th>
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
                <StatusBadge :type="STATUS_LABEL[s.status]?.type ?? 'neutral'">
                  {{ STATUS_LABEL[s.status]?.text ?? s.status }}
                </StatusBadge>
              </td>
              <td>
                <StatusBadge :type="s.is_show ? 'success' : 'neutral'">
                  {{ s.is_show ? '展示' : '隐藏' }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ s.sort }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openNodeEdit(s)">编辑</button>
                  <button class="btn-danger h-7 px-3 text-14" @click="removeNode(s)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && servers.length === 0">
              <td colspan="10"><EmptyState text="暂无节点" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 分组列表 -->
    <div class="card-base mt-5 p-5">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-16 font-600 text-[var(--c-text)]">节点分组</h2>
        <button class="btn-soft-primary h-8 px-3 text-14" @click="openGroupCreate">新建分组</button>
      </div>
      <div v-if="groups.length === 0" class="py-4"><EmptyState text="暂无分组" /></div>
      <div v-else class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
        <div
          v-for="g in groups"
          :key="g.id"
          class="flex items-center justify-between rounded-[var(--r-card)] border border-[var(--c-border)] px-4 py-3"
        >
          <div>
            <div class="text-14 font-500 text-[var(--c-text)]">{{ g.name }}</div>
            <div class="text-14 text-[var(--c-text-sub)]">
              {{ servers.filter((s) => s.group_id === g.id).length }} 个节点
            </div>
          </div>
          <div class="flex gap-2">
            <button class="btn-soft-primary h-7 px-2.5 text-14" @click="openGroupEdit(g)">编辑</button>
            <button class="btn-danger h-7 px-2.5 text-14" @click="removeGroup(g)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 节点弹窗 -->
    <n-modal
      v-model:show="nodeModal"
      preset="card"
      :title="editingNode === null ? '新建节点' : '编辑节点'"
      style="width: 560px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="名称">
            <n-input v-model:value="nodeForm.name" placeholder="节点名称" />
          </n-form-item>
          <n-form-item label="分组">
            <n-select
              v-model:value="nodeForm.group_id"
              :options="groups.map((g) => ({ label: g.name, value: g.id }))"
            />
          </n-form-item>
          <n-form-item label="协议">
            <n-select v-model:value="nodeForm.type" :options="PROTOCOLS" />
          </n-form-item>
          <n-form-item label="倍率">
            <n-input-number v-model:value="nodeForm.rate" :min="0.1" :step="0.1" class="w-full" />
          </n-form-item>
          <n-form-item label="地址 Host">
            <n-input v-model:value="nodeForm.host" placeholder="example.com 或 IP" />
          </n-form-item>
          <n-form-item label="端口">
            <n-input-number v-model:value="nodeForm.port" :min="1" :max="65535" class="w-full" />
          </n-form-item>
          <n-form-item label="状态">
            <n-select
              v-model:value="nodeForm.status"
              :options="[
                { label: '正常', value: 1 },
                { label: '拥挤', value: 2 },
                { label: '维护', value: 3 },
              ]"
            />
          </n-form-item>
          <n-form-item label="排序">
            <n-input-number v-model:value="nodeForm.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item label="标签">
          <n-select
            v-model:value="nodeForm.tags"
            multiple
            filterable
            tag
            :options="[]"
            placeholder="回车添加标签,如:旗舰 / 中转"
          />
        </n-form-item>
        <n-form-item label="协议配置(JSON)">
          <n-input
            v-model:value="nodeForm.config"
            type="textarea"
            :rows="4"
            placeholder='{"password":"...","method":"aes-256-gcm"}'
          />
        </n-form-item>
        <n-form-item label="展示">
          <n-switch v-model:value="nodeForm.is_show" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="nodeModal = false">取消</button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="nodeSaving" @click="saveNode">
            保存
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 分组弹窗 -->
    <n-modal
      v-model:show="groupModal"
      preset="card"
      :title="editingGroup === null ? '新建分组' : '编辑分组'"
      style="width: 360px"
    >
      <n-form label-placement="top">
        <n-form-item label="分组名称">
          <n-input v-model:value="groupName" placeholder="如:香港 / 美国" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="groupSort" class="w-full" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="groupModal = false">取消</button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="groupSaving" @click="saveGroup">
            保存
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
