<script setup lang="ts">
/**
 * 管理后台 · 公告管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/notices(docs/api/README.md §16.1)
 */
import { onMounted, reactive, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminNoticeItem, AdminNoticeReq } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminNoticeItem[]>([])

// 新建/编辑弹窗
const modal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = reactive<AdminNoticeReq>({
  title: '',
  content: '',
  is_show: true,
  sort: 0,
})

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.notices()
    list.value = res.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, { title: '', content: '', is_show: true, sort: 0 })
  modal.value = true
}

function openEdit(n: AdminNoticeItem) {
  editingId.value = n.id
  Object.assign(form, { title: n.title, content: n.content, is_show: n.is_show, sort: n.sort })
  modal.value = true
}

async function save() {
  if (!form.title.trim()) {
    message.warning('请输入公告标题')
    return
  }
  if (!form.content.trim()) {
    message.warning('请输入公告内容')
    return
  }
  saving.value = true
  try {
    if (editingId.value === null) {
      await apiAdmin.createNotice({ ...form })
      message.success('公告已创建')
    } else {
      await apiAdmin.updateNotice(editingId.value, { ...form })
      message.success('公告已更新')
    }
    modal.value = false
    void load()
  } finally {
    saving.value = false
  }
}

function remove(n: AdminNoticeItem) {
  dialog.warning({
    title: '删除公告',
    content: `确定删除公告「${n.title}」吗?该操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.deleteNotice(n.id)
      message.success('已删除')
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="公告管理" subtitle="仪表板公告的发布、隐藏与删除">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> 发布公告
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
              <th>标题</th>
              <th>内容</th>
              <th>状态</th>
              <th>排序</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in list" :key="n.id">
              <td class="num-font">{{ n.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ n.title }}</td>
              <td class="max-w-[320px] truncate text-14 text-[var(--c-text-sub)]">
                {{ n.content }}
              </td>
              <td>
                <StatusBadge :type="n.is_show ? 'success' : 'neutral'">
                  {{ n.is_show ? '展示中' : '已隐藏' }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ n.sort }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(n.created_at) }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(n)">编辑</button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(n)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="7"><EmptyState text="暂无公告" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? '发布公告' : '编辑公告'"
      style="width: 600px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="标题">
            <n-input v-model:value="form.title" placeholder="公告标题" />
          </n-form-item>
          <n-form-item label="排序">
            <n-input-number v-model:value="form.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item label="内容(Markdown)">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :rows="6"
            placeholder="公告正文,支持 Markdown"
          />
        </n-form-item>
        <n-form-item label="展示">
          <n-switch v-model:value="form.is_show" />
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
