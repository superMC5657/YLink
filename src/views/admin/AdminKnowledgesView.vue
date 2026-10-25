<script setup lang="ts">
/**
 * 管理后台 · 知识库管理:列表 / 新建 / 编辑 / 删除 + 分类管理。
 * 数据:GET/POST/PUT/DELETE /admin/knowledges(docs/api/README.md §16.1)
 * CRUD 脚手架归 useAdminCrud(分类联动走 onOpen/beforeSave 钩子),
 * 排序弹窗归 useSortDialog(呈现 SortDialog)。
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminKnowledgeCategoryItem, AdminKnowledgeItem, AdminKnowledgeReq } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import SortDialog from '@/components/admin/SortDialog.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'
import { useAdminCrud } from '@/composables/useAdminCrud'
import { useSortDialog } from '@/composables/useSortDialog'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const langFilter = ref<'all' | 'zh-CN' | 'en-US'>('all')
const keyword = ref('')

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

const { loading, list, modal, editingId, saving, form, load, openCreate, openEdit, save, remove } =
  useAdminCrud<AdminKnowledgeItem, AdminKnowledgeReq>({
    fetch: () => apiAdmin.knowledges(),
    emptyForm: () => ({
      category: '',
      title: '',
      body: '',
      language: 'zh-CN',
      is_show: true,
      sort: 0,
    }),
    toForm: (k) => ({
      category: k.category,
      title: k.title,
      body: k.body,
      language: k.language,
      is_show: k.is_show,
      sort: k.sort,
    }),
    create: (body) => apiAdmin.createKnowledge(body),
    update: (id, body) => apiAdmin.updateKnowledge(id, body),
    beforeSave: applyCategoryValue,
    validate: (f) => {
      if (!f.category.trim()) {
        message.warning(t('adminKnowledges.enterCategory'))
        return false
      }
      if (!f.title.trim()) {
        message.warning(t('adminKnowledges.enterTitle'))
        return false
      }
      if (!f.body.trim()) {
        message.warning(t('adminKnowledges.enterBody'))
        return false
      }
      return true
    },
    onOpen: (f, item) => {
      if (item === null) {
        // 新建态语言跟随当前筛选
        f.language = langFilter.value === 'all' ? 'zh-CN' : langFilter.value
        categoryValue.value = null
      } else {
        categoryValue.value = item.category_id ?? item.category
      }
      void loadCategoryOptions(f.language ?? 'zh-CN')
    },
    remove: (id) => apiAdmin.deleteKnowledge(id),
    removeTexts: {
      titleKey: 'adminKnowledges.deleteTitle',
      content: (k) => t('adminKnowledges.deleteConfirm', { title: k.title }),
      positiveKey: 'adminKnowledges.delete',
    },
    messages: {
      created: 'adminKnowledges.created',
      updated: 'adminKnowledges.updated',
      deleted: 'adminKnowledges.deleted',
    },
  })

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

// ---------- F15 排序弹窗(数据域归 useSortDialog) ----------
const { sortModal, sortItems, sortSaving, openSortModal, moveSortItem, saveSort } =
  useSortDialog<AdminKnowledgeItem>({
    source: () => filtered.value,
    nameOf: (k) => k.title,
    save: (items) => apiAdmin.sortKnowledges({ items }),
    savedKey: 'adminKnowledges.sortSaved',
    reload: load,
  })

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
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
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
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
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
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
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
      } catch {
        // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
      }
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
    <SortDialog
      v-model:show="sortModal"
      :items="sortItems"
      :saving="sortSaving"
      :title="t('adminKnowledges.sortTitle')"
      :hint="t('adminKnowledges.sortHint')"
      :save-text="t('adminKnowledges.sortSave')"
      @move="moveSortItem"
      @save="saveSort"
    />

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
