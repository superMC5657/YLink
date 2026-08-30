<script setup lang="ts">
/**
 * 管理后台 · 订阅模板(F10):按客户端类型(clash/sing-box/v2ray)维护 Go text/template
 * 全文档订阅模板 / 预览(示例数据渲染) / 恢复内置生成器。
 * 数据:GET/PUT/DELETE /admin/subscription-templates、POST /{name}/preview
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminSubscriptionTemplateItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminSubscriptionTemplateItem[]>([])

// 编辑弹窗
const modal = ref(false)
const editingName = ref('')
const saving = ref(false)
const draft = ref({ content: '' })

// 预览弹窗
const previewModal = ref(false)
const previewName = ref('')
const previewLoading = ref(false)
const previewContent = ref('')

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.subscriptionTemplates()
    list.value = res.list
  } finally {
    loading.value = false
  }
}

function openEdit(item: AdminSubscriptionTemplateItem) {
  editingName.value = item.name
  draft.value = { content: item.content }
  modal.value = true
}

async function save() {
  if (!draft.value.content.trim() || saving.value) return
  saving.value = true
  try {
    await apiAdmin.saveSubscriptionTemplate(editingName.value, { ...draft.value })
    message.success(t('adminSubscriptionTemplates.saved'))
    modal.value = false
    void load()
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
  } finally {
    saving.value = false
  }
}

function reset(item: AdminSubscriptionTemplateItem) {
  dialog.warning({
    title: t('adminSubscriptionTemplates.resetTitle'),
    content: t('adminSubscriptionTemplates.resetConfirm', { name: item.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.resetSubscriptionTemplate(item.name)
      message.success(t('adminSubscriptionTemplates.resetDone'))
      void load()
    },
  })
}

async function openPreview(item: AdminSubscriptionTemplateItem) {
  previewName.value = item.name
  previewContent.value = ''
  previewModal.value = true
  previewLoading.value = true
  try {
    const res = await apiAdmin.previewSubscriptionTemplate(item.name)
    previewContent.value = res.content
  } catch {
    // 错误提示由 http 层统一 toast;失败时同样关闭预览弹窗
    previewModal.value = false
  } finally {
    previewLoading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader
      :title="t('adminSubscriptionTemplates.title')"
      :subtitle="t('adminSubscriptionTemplates.subtitle')"
    >
      <template #actions>
        <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
          <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
        </button>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[860px]">
          <thead>
            <tr>
              <th>{{ t('adminSubscriptionTemplates.name') }}</th>
              <th>{{ t('adminSubscriptionTemplates.remark') }}</th>
              <th>{{ t('adminSubscriptionTemplates.variables') }}</th>
              <th>{{ t('adminSubscriptionTemplates.state') }}</th>
              <th>{{ t('adminSubscriptionTemplates.updatedAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.name">
              <td class="num-font font-500 text-[var(--c-text)]">{{ item.name }}</td>
              <td class="max-w-56 text-14 text-[var(--c-text-sub)]">{{ item.remark }}</td>
              <td class="max-w-56">
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="p in item.variables"
                    :key="p"
                    class="rounded bg-[var(--c-primary-soft)] px-1.5 py-0.5 text-12 text-[var(--c-primary)]"
                  >
                    {{ p }}
                  </span>
                </div>
              </td>
              <td>
                <StatusBadge :type="item.is_custom ? 'warning' : 'neutral'">
                  {{
                    item.is_custom
                      ? t('adminSubscriptionTemplates.custom')
                      : t('adminSubscriptionTemplates.default')
                  }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ item.updated_at ? formatTime(item.updated_at) : '-' }}
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(item)">
                    {{ t('adminSubscriptionTemplates.edit') }}
                  </button>
                  <button class="btn-soft-blue h-7 px-3 text-14" @click="openPreview(item)">
                    {{ t('adminSubscriptionTemplates.preview') }}
                  </button>
                  <button
                    v-if="item.is_custom"
                    class="btn-soft-neutral h-7 px-3 text-14"
                    @click="reset(item)"
                  >
                    {{ t('adminSubscriptionTemplates.reset') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="6"><EmptyState :text="t('adminSubscriptionTemplates.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="t('adminSubscriptionTemplates.editTitle', { name: editingName })"
      style="width: 720px"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('adminSubscriptionTemplates.content')">
          <n-input
            v-model:value="draft.content"
            type="textarea"
            :rows="14"
            class="font-mono"
            :placeholder="t('adminSubscriptionTemplates.contentPlaceholder')"
          />
        </n-form-item>
        <p class="text-13 text-[var(--c-text-sub)]">
          {{ t('adminSubscriptionTemplates.syntaxTip') }}
        </p>
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

    <!-- 预览弹窗 -->
    <n-modal
      v-model:show="previewModal"
      preset="card"
      :title="t('adminSubscriptionTemplates.previewTitle', { name: previewName })"
      style="width: 720px"
    >
      <n-spin :show="previewLoading">
        <pre
          class="max-h-[60vh] overflow-auto rounded-lg bg-[var(--c-bg)] p-4 font-mono text-13 text-[var(--c-text)]"
          >{{ previewContent || ' ' }}</pre>
        <p class="mt-2 text-13 text-[var(--c-text-sub)]">
          {{ t('adminSubscriptionTemplates.previewTip') }}
        </p>
      </n-spin>
    </n-modal>
  </div>
</template>
