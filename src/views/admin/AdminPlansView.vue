<script setup lang="ts">
/**
 * 管理后台 · 套餐管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/plans + GET /admin/server-groups(docs/api/README.md §16)
 */
import { onMounted, reactive, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminPlanItem, AdminPlanReq, AdminServerGroupItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminPlanItem[]>([])
const groups = ref<AdminServerGroupItem[]>([])

// 新建/编辑弹窗
const modal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = reactive<AdminPlanReq>({
  name: '',
  content: '',
  month_price: null,
  quarter_price: null,
  half_year_price: null,
  year_price: null,
  onetime_price: null,
  traffic_gb: 0,
  speed_limit: null,
  device_limit: null,
  group_ids: [],
  is_show: true,
  sort: 0,
})

async function load() {
  loading.value = true
  try {
    const [planRes, groupRes] = await Promise.all([apiAdmin.plans(), apiAdmin.serverGroups()])
    list.value = planRes.list
    groups.value = groupRes.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, {
    name: '',
    content: '',
    month_price: null,
    quarter_price: null,
    half_year_price: null,
    year_price: null,
    onetime_price: null,
    traffic_gb: 0,
    speed_limit: null,
    device_limit: null,
    group_ids: [],
    is_show: true,
    sort: 0,
  })
  modal.value = true
}

function openEdit(p: AdminPlanItem) {
  editingId.value = p.id
  Object.assign(form, {
    name: p.name,
    content: p.content,
    month_price: p.month_price,
    quarter_price: p.quarter_price,
    half_year_price: p.half_year_price,
    year_price: p.year_price,
    onetime_price: p.onetime_price,
    traffic_gb: p.traffic_gb,
    speed_limit: p.speed_limit,
    device_limit: p.device_limit,
    group_ids: p.group_ids ?? [],
    is_show: p.is_show,
    sort: p.sort,
  })
  modal.value = true
}

async function save() {
  if (!form.name.trim()) {
    message.warning('请输入套餐名称')
    return
  }
  saving.value = true
  try {
    if (editingId.value === null) {
      await apiAdmin.createPlan({ ...form })
      message.success('套餐已创建')
    } else {
      await apiAdmin.updatePlan(editingId.value, { ...form })
      message.success('套餐已更新')
    }
    modal.value = false
    void load()
  } finally {
    saving.value = false
  }
}

function remove(p: AdminPlanItem) {
  dialog.warning({
    title: '删除套餐',
    content: `确定删除套餐「${p.name}」吗?该操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.deletePlan(p.id)
      message.success('已删除')
      void load()
    },
  })
}

function price(v: number | null): string {
  return v === null ? '-' : `¥${v}`
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="套餐管理" subtitle="上架/隐藏套餐与周期定价">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> 新建套餐
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[860px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>月付</th>
              <th>季付</th>
              <th>半年</th>
              <th>年付</th>
              <th>一次性</th>
              <th>流量</th>
              <th>设备数</th>
              <th>状态</th>
              <th>排序</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in list" :key="p.id">
              <td>{{ p.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ p.name }}</td>
              <td class="num-font">{{ price(p.month_price) }}</td>
              <td class="num-font">{{ price(p.quarter_price) }}</td>
              <td class="num-font">{{ price(p.half_year_price) }}</td>
              <td class="num-font">{{ price(p.year_price) }}</td>
              <td class="num-font">{{ price(p.onetime_price) }}</td>
              <td>{{ p.traffic_gb }} GB</td>
              <td>{{ p.device_limit ?? '不限' }}</td>
              <td>
                <StatusBadge :type="p.is_show ? 'success' : 'neutral'">
                  {{ p.is_show ? '上架' : '隐藏' }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ p.sort }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(p)">编辑</button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(p)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="12"><EmptyState text="暂无套餐" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? '新建套餐' : '编辑套餐'"
      style="width: 560px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="名称">
            <n-input v-model:value="form.name" placeholder="套餐名称" />
          </n-form-item>
          <n-form-item label="流量(GB)">
            <n-input-number v-model:value="form.traffic_gb" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="月付(元)">
            <n-input-number v-model:value="form.month_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="季付(元)">
            <n-input-number v-model:value="form.quarter_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="半年付(元)">
            <n-input-number v-model:value="form.half_year_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="年付(元)">
            <n-input-number v-model:value="form.year_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="一次性(元)">
            <n-input-number v-model:value="form.onetime_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="设备数限制">
            <n-input-number v-model:value="form.device_limit" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="限速(Mbps)">
            <n-input-number v-model:value="form.speed_limit" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="排序">
            <n-input-number v-model:value="form.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item label="可见节点分组">
          <n-select
            v-model:value="form.group_ids"
            multiple
            :options="groups.map((g) => ({ label: g.name, value: g.id }))"
            placeholder="不选则对全部用户开放"
          />
        </n-form-item>
        <n-form-item label="上架">
          <n-switch v-model:value="form.is_show" />
        </n-form-item>
        <n-form-item label="内容说明(Markdown)">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :rows="3"
            placeholder="套餐卖点,支持 Markdown"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="modal = false">取消</button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="save">
            保存
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
