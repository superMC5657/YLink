<script setup lang="ts">
/**
 * 管理后台 · 邮件模板(F11):内置模板列表 / 编辑(subject+body,Go template 语法) /
 * 恢复默认 / 测试发送(走真实 SMTP)。
 * 数据:GET/PUT/DELETE /admin/mail-templates、POST /admin/mail-templates/{name}/test
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminMailTemplateItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminMailTemplateItem[]>([])

// 编辑弹窗
const modal = ref(false)
const editingName = ref('')
const saving = ref(false)
const draft = ref({ subject: '', body: '' })

// 测试发送弹窗
const testModal = ref(false)
const testName = ref('')
const testEmail = ref('')
const testSending = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.mailTemplates()
    list.value = res.list
  } finally {
    loading.value = false
  }
}

function openEdit(item: AdminMailTemplateItem) {
  editingName.value = item.name
  draft.value = { subject: item.subject, body: item.body }
  modal.value = true
}

async function save() {
  if (!draft.value.subject.trim() || !draft.value.body.trim() || saving.value) return
  saving.value = true
  try {
    await apiAdmin.saveMailTemplate(editingName.value, { ...draft.value })
    message.success(t('adminMailTemplates.saved'))
    modal.value = false
    void load()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

function reset(item: AdminMailTemplateItem) {
  dialog.warning({
    title: t('adminMailTemplates.resetTitle'),
    content: t('adminMailTemplates.resetConfirm', { name: item.name }),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.resetMailTemplate(item.name)
      message.success(t('adminMailTemplates.resetDone'))
      void load()
    },
  })
}

function openTest(item: AdminMailTemplateItem) {
  testName.value = item.name
  testEmail.value = ''
  testModal.value = true
}

async function sendTest() {
  if (!testEmail.value.trim() || testSending.value) return
  testSending.value = true
  try {
    await apiAdmin.testMailTemplate(testName.value, { to_email: testEmail.value.trim() })
    message.success(t('adminMailTemplates.testSent'))
    testModal.value = false
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    testSending.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminMailTemplates.title')" :subtitle="t('adminMailTemplates.subtitle')">
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
              <th>{{ t('adminMailTemplates.name') }}</th>
              <th>{{ t('adminMailTemplates.subject') }}</th>
              <th>{{ t('adminMailTemplates.remark') }}</th>
              <th>{{ t('adminMailTemplates.placeholders') }}</th>
              <th>{{ t('adminMailTemplates.state') }}</th>
              <th>{{ t('adminMailTemplates.updatedAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.name">
              <td class="num-font font-500 text-[var(--c-text)]">{{ item.name }}</td>
              <td class="max-w-52 truncate text-14 text-[var(--c-text)]">{{ item.subject }}</td>
              <td class="max-w-48 truncate text-14 text-[var(--c-text-sub)]">
                {{ item.remark }}
              </td>
              <td class="max-w-56">
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="p in item.placeholders"
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
                      ? t('adminMailTemplates.custom')
                      : t('adminMailTemplates.default')
                  }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ item.updated_at ? formatTime(item.updated_at) : '-' }}
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(item)">
                    {{ t('adminMailTemplates.edit') }}
                  </button>
                  <button class="btn-soft-blue h-7 px-3 text-14" @click="openTest(item)">
                    {{ t('adminMailTemplates.test') }}
                  </button>
                  <button
                    v-if="item.is_custom"
                    class="btn-soft-neutral h-7 px-3 text-14"
                    @click="reset(item)"
                  >
                    {{ t('adminMailTemplates.reset') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="7"><EmptyState :text="t('adminMailTemplates.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="t('adminMailTemplates.editTitle', { name: editingName })"
      style="width: 640px"
    >
      <n-form label-placement="top">
        <n-form-item :label="t('adminMailTemplates.subject')">
          <n-input v-model:value="draft.subject" />
        </n-form-item>
        <n-form-item :label="t('adminMailTemplates.body')">
          <n-input
            v-model:value="draft.body"
            type="textarea"
            :rows="8"
            class="font-mono"
            :placeholder="t('adminMailTemplates.bodyPlaceholder')"
          />
        </n-form-item>
        <p class="text-13 text-[var(--c-text-sub)]">{{ t('adminMailTemplates.syntaxTip') }}</p>
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

    <!-- 测试发送弹窗 -->
    <n-modal
      v-model:show="testModal"
      preset="card"
      :title="t('adminMailTemplates.testTitle', { name: testName })"
      style="width: 460px"
    >
      <div class="space-y-4">
        <n-input v-model:value="testEmail" :placeholder="t('adminMailTemplates.testTo')" />
        <p class="text-13 text-[var(--c-text-sub)]">{{ t('adminMailTemplates.testTip') }}</p>
        <button
          class="btn-primary h-10 w-full text-14"
          :disabled="testSending || !testEmail.trim()"
          @click="sendTest"
        >
          {{ t('adminMailTemplates.testSend') }}
        </button>
      </div>
    </n-modal>
  </div>
</template>
