<script setup lang="ts">
/**
 * 管理后台 · 知识库管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/knowledges(docs/api/README.md §16.1)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminKnowledgeItem, AdminKnowledgeReq } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminKnowledgeItem[]>([])
const langFilter = ref<'all' | 'zh-CN' | 'en-US'>('all')
const keyword = ref('')

const filtered = computed(() =>
  list.value.filter((k) => {
    if (langFilter.value !== 'all' && k.language !== langFilter.value) return false
    const kw = keyword.value.trim().toLowerCase()
    if (!kw) return true
    return (
      k.title.toLowerCase().includes(kw) ||
      k.category.toLowerCase().includes(kw) ||
      k.body.toLowerCase().includes(kw)
    )
  }),
)

// 新建/编辑弹窗
const modal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = reactive<AdminKnowledgeReq>({
  category: '',
  title: '',
  body: '',
  language: 'zh-CN',
  is_show: true,
  sort: 0,
})

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.knowledges()
    list.value = res.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, {
    category: '',
    title: '',
    body: '',
    language: langFilter.value === 'all' ? 'zh-CN' : langFilter.value,
    is_show: true,
    sort: 0,
  })
  modal.value = true
}

function openEdit(k: AdminKnowledgeItem) {
  editingId.value = k.id
  Object.assign(form, {
    category: k.category,
    title: k.title,
    body: k.body,
    language: k.language,
    is_show: k.is_show,
    sort: k.sort,
  })
  modal.value = true
}

async function save() {
  if (!form.category.trim()) {
    message.warning(t('adminKnowledges.enterCategory'))
    return
  }
  if (!form.title.trim()) {
    message.warning(t('adminKnowledges.enterTitle'))
    return
  }
  if (!form.body.trim()) {
    message.warning(t('adminKnowledges.enterBody'))
    return
  }
  saving.value = true
  try {
    if (editingId.value === null) {
      await apiAdmin.createKnowledge({ ...form })
      message.success(t('adminKnowledges.created'))
    } else {
      await apiAdmin.updateKnowledge(editingId.value, { ...form })
      message.success(t('adminKnowledges.updated'))
    }
    modal.value = false
    void load()
  } finally {
    saving.value = false
  }
}

function remove(k: AdminKnowledgeItem) {
  dialog.warning({
    title: t('adminKnowledges.deleteTitle'),
    content: t('adminKnowledges.deleteConfirm', { title: k.title }),
    positiveText: t('adminKnowledges.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.deleteKnowledge(k.id)
      message.success(t('adminKnowledges.deleted'))
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminKnowledges.pageTitle')" :subtitle="t('adminKnowledges.subtitle')">
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <n-input
            v-model:value="keyword"
            :placeholder="t('adminKnowledges.searchPlaceholder')"
            clearable
            class="w-52"
          />
          <n-radio-group v-model:value="langFilter">
            <n-radio-button value="all">{{ t('adminKnowledges.allLanguages') }}</n-radio-button>
            <n-radio-button value="zh-CN">{{ t('adminKnowledges.zh') }}</n-radio-button>
            <n-radio-button value="en-US">{{ t('adminKnowledges.en') }}</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminKnowledges.newDoc') }}
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
              <th>{{ t('adminKnowledges.category') }}</th>
              <th>{{ t('adminKnowledges.title') }}</th>
              <th>{{ t('adminKnowledges.language') }}</th>
              <th>{{ t('adminKnowledges.status') }}</th>
              <th>{{ t('adminKnowledges.sort') }}</th>
              <th>{{ t('adminKnowledges.updatedAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="k in filtered" :key="k.id">
              <td class="num-font">{{ k.id }}</td>
              <td>
                <span
                  class="rounded bg-[var(--c-primary-soft)] px-2 py-0.5 text-13 text-[var(--c-primary)]"
                >
                  {{ k.category }}
                </span>
              </td>
              <td class="font-500 text-[var(--c-text)]">{{ k.title }}</td>
              <td class="text-14">{{ k.language }}</td>
              <td>
                <StatusBadge :type="k.is_show ? 'success' : 'neutral'">
                  {{ k.is_show ? t('adminKnowledges.showing') : t('adminKnowledges.hidden') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ k.sort }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(k.updated_at) }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(k)">
                    {{ t('adminKnowledges.edit') }}
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(k)">
                    {{ t('adminKnowledges.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && filtered.length === 0">
              <td colspan="8"><EmptyState :text="t('adminKnowledges.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="
        editingId === null ? t('adminKnowledges.createTitle') : t('adminKnowledges.editTitle')
      "
      style="width: 640px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminKnowledges.category')">
            <n-input
              v-model:value="form.category"
              :placeholder="t('adminKnowledges.categoryPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('adminKnowledges.language')">
            <n-select
              v-model:value="form.language"
              :options="[
                { label: t('adminKnowledges.zhOption'), value: 'zh-CN' },
                { label: t('adminKnowledges.enOption'), value: 'en-US' },
              ]"
            />
          </n-form-item>
          <n-form-item :label="t('adminKnowledges.title')">
            <n-input
              v-model:value="form.title"
              :placeholder="t('adminKnowledges.titlePlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('adminKnowledges.sort')">
            <n-input-number v-model:value="form.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminKnowledges.bodyLabel')">
          <n-input
            v-model:value="form.body"
            type="textarea"
            :rows="8"
            :placeholder="t('adminKnowledges.bodyPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('adminKnowledges.show')">
          <n-switch v-model:value="form.is_show" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="modal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="save">
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
