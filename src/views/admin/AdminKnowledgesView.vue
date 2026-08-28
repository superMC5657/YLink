<script setup lang="ts">
/**
 * 管理后台 · 知识库管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/knowledges(docs/api/README.md §16.1)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminKnowledgeCategoryItem, AdminKnowledgeItem, AdminKnowledgeReq } from '@/types/api'
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

// ---------- F15 分类选择（下拉选已有分类或输入新分类） ----------
const categoryValue = ref<number | string | null>(null)
const categoryOptions = ref<{ label: string; value: number }[]>([])

async function loadCategoryOptions(language: string) {
  try {
    const res = await apiAdmin.knowledgeCategories({ language })
    categoryOptions.value = res.list.map((c) => ({ label: c.name, value: c.id }))
  } catch {
    categoryOptions.value = []
  }
}

function onFormLanguageChange(lang: string | undefined) {
  form.language = lang
  categoryValue.value = null
  void loadCategoryOptions(lang ?? 'zh-CN')
}

/** 保存前把分类选择归一为 category / category_id */
function applyCategoryValue() {
  const v = categoryValue.value
  if (v === null || v === '') {
    form.category = ''
    form.category_id = null
    return
  }
  if (typeof v === 'number') {
    form.category = categoryOptions.value.find((o) => o.value === v)?.label ?? ''
    form.category_id = v
  } else {
    form.category = v
    form.category_id = null
  }
}

// ---------- F15 排序弹窗 ----------
const sortModal = ref(false)
const sortItems = ref<{ id: number; name: string }[]>([])
const sortSaving = ref(false)

function openSortModal() {
  sortItems.value = [...filtered.value]
    .sort((a, b) => a.sort - b.sort || a.id - b.id)
    .map((k) => ({ id: k.id, name: k.title }))
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
    await apiAdmin.sortKnowledges({ items: sortItems.value.map((k, i) => ({ id: k.id, sort: i })) })
    message.success(t('adminKnowledges.sortSaved'))
    sortModal.value = false
    await load()
  } finally {
    sortSaving.value = false
  }
}

// ---------- F15 分类管理弹窗 ----------
const catModal = ref(false)
const catLoading = ref(false)
const catSaving = ref(false)
const cats = ref<AdminKnowledgeCategoryItem[]>([])
const catNewName = ref('')
const catNewLanguage = ref<'zh-CN' | 'en-US'>('zh-CN')
/** 行内改名草稿:catId -> name */
const catRenameDrafts = ref<Record<number, string>>({})

async function openCatModal() {
  catModal.value = true
  catLoading.value = true
  try {
    const res = await apiAdmin.knowledgeCategories({})
    cats.value = res.list
  } finally {
    catLoading.value = false
  }
}

async function createCategory() {
  if (!catNewName.value.trim() || catSaving.value) return
  catSaving.value = true
  try {
    await apiAdmin.createKnowledgeCategory({
      language: catNewLanguage.value,
      name: catNewName.value.trim(),
    })
    message.success(t('adminKnowledges.catCreated'))
    catNewName.value = ''
    await openCatModal()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    catSaving.value = false
  }
}

async function renameCategory(c: AdminKnowledgeCategoryItem) {
  const name = (catRenameDrafts.value[c.id] ?? '').trim()
  if (!name || name === c.name || catSaving.value) return
  catSaving.value = true
  try {
    await apiAdmin.updateKnowledgeCategory(c.id, { name, sort: c.sort })
    message.success(t('adminKnowledges.catUpdated'))
    await openCatModal()
    await load()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    catSaving.value = false
  }
}

/** 分类上/下移（与同语言相邻分类交换 sort 值后立即保存） */
async function moveCategory(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= cats.value.length) return
  const a = cats.value[index]
  const b = cats.value[target]
  if (a.language !== b.language) return
  catSaving.value = true
  try {
    await Promise.all([
      apiAdmin.updateKnowledgeCategory(a.id, { name: a.name, sort: b.sort }),
      apiAdmin.updateKnowledgeCategory(b.id, { name: b.name, sort: a.sort }),
    ])
    await openCatModal()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    catSaving.value = false
  }
}

function removeCategory(c: AdminKnowledgeCategoryItem) {
  dialog.warning({
    title: t('adminKnowledges.catDeleteTitle'),
    content: t('adminKnowledges.catDeleteConfirm', { name: c.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await apiAdmin.deleteKnowledgeCategory(c.id)
        message.success(t('adminKnowledges.catDeleted'))
        await openCatModal()
      } catch (e) {
        message.error((e as Error).message)
      }
    },
  })
}

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
  categoryValue.value = null
  void loadCategoryOptions(form.language ?? 'zh-CN')
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
  categoryValue.value = k.category_id ?? k.category
  void loadCategoryOptions(k.language)
  modal.value = true
}

async function save() {
  applyCategoryValue()
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
            class="w-52!"
          />
          <n-radio-group v-model:value="langFilter">
            <n-radio-button value="all">{{ t('adminKnowledges.allLanguages') }}</n-radio-button>
            <n-radio-button value="zh-CN">{{ t('adminKnowledges.zh') }}</n-radio-button>
            <n-radio-button value="en-US">{{ t('adminKnowledges.en') }}</n-radio-button>
          </n-radio-group>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="openCatModal">
            <AppIcon name="book" :size="15" /> {{ t('adminKnowledges.catManage') }}
          </button>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="openSortModal">
            <AppIcon name="sliders" :size="15" /> {{ t('adminKnowledges.sort') }}
          </button>
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
            <n-select
              v-model:value="categoryValue"
              filterable
              tag
              clearable
              :options="categoryOptions"
              :placeholder="t('adminKnowledges.categoryPlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('adminKnowledges.language')">
            <n-select
              :value="form.language"
              :options="[
                { label: t('adminKnowledges.zhOption'), value: 'zh-CN' },
                { label: t('adminKnowledges.enOption'), value: 'en-US' },
              ]"
              @update:value="onFormLanguageChange"
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

    <!-- F15 排序弹窗 -->
    <n-modal
      v-model:show="sortModal"
      preset="card"
      :title="t('adminKnowledges.sortTitle')"
      style="width: 420px"
    >
      <p class="mb-3 text-14 text-[var(--c-text-sub)]">{{ t('adminKnowledges.sortHint') }}</p>
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
            {{ t('adminKnowledges.sortSave') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- F15 分类管理弹窗 -->
    <n-modal
      v-model:show="catModal"
      preset="card"
      :title="t('adminKnowledges.catManage')"
      style="width: 560px"
    >
      <n-spin :show="catLoading">
        <!-- 新增分类 -->
        <div class="mb-4 flex items-center gap-2">
          <n-select
            v-model:value="catNewLanguage"
            class="w-32"
            :options="[
              { label: t('adminKnowledges.zhOption'), value: 'zh-CN' },
              { label: t('adminKnowledges.enOption'), value: 'en-US' },
            ]"
          />
          <n-input
            v-model:value="catNewName"
            class="flex-1"
            :placeholder="t('adminKnowledges.catNamePlaceholder')"
          />
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="catSaving"
            @click="createCategory"
          >
            {{ t('adminKnowledges.catAdd') }}
          </button>
        </div>

        <div class="space-y-2">
          <div
            v-for="(c, i) in cats"
            :key="c.id"
            class="flex items-center gap-2 rounded-lg border border-[var(--c-border)] px-3 py-2"
          >
            <StatusBadge type="neutral">{{ c.language }}</StatusBadge>
            <n-input
              v-model:value="catRenameDrafts[c.id]"
              size="small"
              class="flex-1"
              :placeholder="c.name"
            />
            <span class="num-font shrink-0 text-13 text-[var(--c-text-sub)]">
              {{ t('adminKnowledges.catDocCount', { count: c.knowledge_count }) }}
            </span>
            <button
              class="btn-soft-neutral h-7 px-2 text-14"
              :disabled="catSaving || i === 0 || cats[i - 1].language !== c.language"
              @click="moveCategory(i, -1)"
            >
              ↑
            </button>
            <button
              class="btn-soft-neutral h-7 px-2 text-14"
              :disabled="catSaving || i === cats.length - 1 || cats[i + 1].language !== c.language"
              @click="moveCategory(i, 1)"
            >
              ↓
            </button>
            <button
              class="btn-soft-blue h-7 px-2 text-14"
              :disabled="catSaving"
              @click="renameCategory(c)"
            >
              {{ t('common.save') }}
            </button>
            <button
              class="btn-soft-danger h-7 px-2 text-14"
              :disabled="catSaving || c.knowledge_count > 0"
              @click="removeCategory(c)"
            >
              {{ t('adminKnowledges.catDelete') }}
            </button>
          </div>
          <EmptyState
            v-if="!catLoading && cats.length === 0"
            :text="t('adminKnowledges.catEmpty')"
          />
        </div>
      </n-spin>
    </n-modal>
  </div>
</template>
