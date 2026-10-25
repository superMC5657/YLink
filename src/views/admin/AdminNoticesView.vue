<script setup lang="ts">
/**
 * 管理后台 · 公告管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/notices(docs/api/README.md §16.1)
 * CRUD 脚手架归 useAdminCrud,排序弹窗归 useSortDialog(呈现 SortDialog)。
 */
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminNoticeItem, AdminNoticeReq } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import SortDialog from '@/components/admin/SortDialog.vue'
import { useMessage } from 'naive-ui'
import { formatTime } from '@/utils/format'
import { useAdminCrud } from '@/composables/useAdminCrud'
import { useSortDialog } from '@/composables/useSortDialog'

const { t } = useI18n()
const message = useMessage()

const { loading, list, modal, editingId, saving, form, load, openCreate, openEdit, save, remove } =
  useAdminCrud<AdminNoticeItem, AdminNoticeReq>({
    fetch: () => apiAdmin.notices(),
    emptyForm: () => ({ title: '', content: '', is_show: true, sort: 0 }),
    toForm: (n) => ({ title: n.title, content: n.content, is_show: n.is_show, sort: n.sort }),
    create: (body) => apiAdmin.createNotice(body),
    update: (id, body) => apiAdmin.updateNotice(id, body),
    validate: (f) => {
      if (!f.title.trim()) {
        message.warning(t('adminNotices.enterTitle'))
        return false
      }
      if (!f.content.trim()) {
        message.warning(t('adminNotices.enterContent'))
        return false
      }
      return true
    },
    remove: (id) => apiAdmin.deleteNotice(id),
    removeTexts: {
      titleKey: 'adminNotices.deleteTitle',
      content: (n) => t('adminNotices.deleteConfirm', { title: n.title }),
      positiveKey: 'adminNotices.delete',
    },
    messages: {
      created: 'adminNotices.created',
      updated: 'adminNotices.updated',
      deleted: 'adminNotices.deleted',
    },
  })

// ---------- F15 排序弹窗(数据域归 useSortDialog) ----------
const { sortModal, sortItems, sortSaving, openSortModal, moveSortItem, saveSort } =
  useSortDialog<AdminNoticeItem>({
    source: () => list.value,
    nameOf: (n) => n.title,
    save: (items) => apiAdmin.sortNotices({ items }),
    savedKey: 'adminNotices.sortSaved',
    reload: load,
  })

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminNotices.pageTitle')" :subtitle="t('adminNotices.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="openSortModal">
            <AppIcon name="sliders" :size="15" /> {{ t('adminNotices.sort') }}
          </button>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminNotices.publish') }}
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
              <th>{{ t('adminNotices.title') }}</th>
              <th>{{ t('adminNotices.content') }}</th>
              <th>{{ t('adminNotices.status') }}</th>
              <th>{{ t('adminNotices.sort') }}</th>
              <th>{{ t('adminNotices.createdAt') }}</th>
              <th>{{ t('common.action') }}</th>
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
                  {{ n.is_show ? t('adminNotices.showing') : t('adminNotices.hidden') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ n.sort }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ formatTime(n.created_at) }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(n)">
                    {{ t('adminNotices.edit') }}
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(n)">
                    {{ t('adminNotices.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="7"><EmptyState :text="t('adminNotices.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? t('adminNotices.createTitle') : t('adminNotices.editTitle')"
      style="width: 600px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminNotices.title')">
            <n-input v-model:value="form.title" :placeholder="t('adminNotices.titlePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('adminNotices.sort')">
            <n-input-number v-model:value="form.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminNotices.contentLabel')">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :rows="6"
            :placeholder="t('adminNotices.contentPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('adminNotices.show')">
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
      :title="t('adminNotices.sortTitle')"
      :hint="t('adminNotices.sortHint')"
      :save-text="t('adminNotices.sortSave')"
      @move="moveSortItem"
      @save="saveSort"
    />
  </div>
</template>
