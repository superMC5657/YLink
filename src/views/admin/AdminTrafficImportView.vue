<script setup lang="ts">
/**
 * 管理后台 · 流量管理(模式 B 手工导入 + F16 流量重置):
 *  - 导入:逐行录入用户流量,一次批量提交(覆盖同日记录);
 *  - 重置:按用户清零用量/重新给量(保留节点上报快照防重复计费,写重置记录);
 *  - 记录:重置历史分页查询。
 * 数据:POST /admin/traffic/import、POST /admin/traffic/reset、GET /admin/traffic/resets(docs/api/README.md §16.1)
 */
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminTrafficResetLogItem, TrafficImportItem, TrafficResetMode } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatBytes, localDateKey } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const activeTab = ref<'import' | 'reset' | 'logs'>('import')

// ---- 导入 ----

const rows = reactive<TrafficImportItem[]>([
  { user_id: 10086, date: localDateKey(new Date()), u: 0, d: 0 },
])

const submitting = ref(false)

function addRow() {
  rows.push({ user_id: 0, date: localDateKey(new Date()), u: 0, d: 0 })
}

function removeRow(index: number) {
  rows.splice(index, 1)
}

async function submit() {
  if (rows.length === 0) {
    message.warning(t('adminTrafficImport.needRows'))
    return
  }
  for (const r of rows) {
    if (!r.user_id) {
      message.warning(t('adminTrafficImport.needUserId'))
      return
    }
    if (!r.date) {
      message.warning(t('adminTrafficImport.needDate'))
      return
    }
  }
  submitting.value = true
  try {
    await apiAdmin.importTraffic({
      items: rows.map((r) => ({
        user_id: r.user_id,
        date: r.date,
        u: Math.max(0, Math.floor(Number(r.u) || 0)),
        d: Math.max(0, Math.floor(Number(r.d) || 0)),
      })),
    })
    message.success(t('adminTrafficImport.imported', { count: rows.length }))
    rows.splice(0, rows.length, {
      user_id: 0,
      date: new Date().toISOString().slice(0, 10),
      u: 0,
      d: 0,
    })
  } finally {
    submitting.value = false
  }
}

// ---- F16 重置 ----

const resetUserIds = ref<string>('')
const resetMode = ref<TrafficResetMode>('clear_usage')
const resetting = ref(false)

function parseResetIds(): number[] {
  return resetUserIds.value
    .split(/[,;\s]+/)
    .map((s) => Number.parseInt(s, 10))
    .filter((n) => Number.isInteger(n) && n > 0)
}

function submitReset() {
  const ids = parseResetIds()
  if (ids.length === 0) {
    message.warning(t('adminTrafficImport.resetNeedIds'))
    return
  }
  dialog.warning({
    title: t('adminTrafficImport.resetTitle'),
    content: t('adminTrafficImport.resetConfirm', { count: ids.length }),
    positiveText: t('adminTrafficImport.resetAction'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      resetting.value = true
      try {
        const resp = await apiAdmin.resetTraffic({ user_ids: ids, mode: resetMode.value })
        const title = t('adminTrafficImport.resetDone')
        if (resp.failed.length === 0) {
          message.success(`${title}: ${resp.success}`)
        } else {
          const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
          message.warning(
            `${title}: ${resp.success} / ${t('adminTrafficImport.resetFailed')} ${resp.failed.length} — ${detail}`,
          )
        }
        resetUserIds.value = ''
        void loadLogs()
      } finally {
        resetting.value = false
      }
    },
  })
}

// ---- F16 重置记录 ----

const logs = ref<AdminTrafficResetLogItem[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logPageSize = 10
const logUserId = ref<number | null>(null)
const logsLoading = ref(false)

async function loadLogs() {
  logsLoading.value = true
  try {
    const res = await apiAdmin.trafficResets({
      page: logPage.value,
      page_size: logPageSize,
      user_id: logUserId.value ?? '',
    })
    // 后端旧版本空列表可能返回 list:null，直接赋值会让模板判空抛 TypeError 卡死页面
    logs.value = res.list ?? []
    logTotal.value = res.total
  } finally {
    logsLoading.value = false
  }
}

function searchLogs() {
  logPage.value = 1
  void loadLogs()
}

function modeText(mode: TrafficResetMode): string {
  return mode === 'reset_quota'
    ? t('adminTrafficImport.modeResetQuota')
    : t('adminTrafficImport.modeClearUsage')
}

void loadLogs()
</script>

<template>
  <div>
    <PageHeader
      :title="t('adminTrafficImport.pageTitle')"
      :subtitle="t('adminTrafficImport.subtitle')"
    />

    <n-tabs v-model:value="activeTab" type="line" animated>
      <!-- 导入 -->
      <n-tab-pane name="import" :tab="t('adminTrafficImport.tabImport')">
        <div class="card-base overflow-x-auto">
          <div class="mb-3 flex items-center gap-2">
            <button class="btn-soft-neutral h-9 px-3 text-14" @click="addRow">
              <AppIcon name="plus" :size="15" /> {{ t('adminTrafficImport.addRow') }}
            </button>
            <button class="btn-primary h-9 px-4 text-14" :disabled="submitting" @click="submit">
              {{ t('adminTrafficImport.batchImport') }}
            </button>
          </div>
          <n-table :bordered="false" :single-line="false" class="min-w-[760px]">
            <thead>
              <tr>
                <th>#</th>
                <th>{{ t('adminTrafficImport.userId') }}</th>
                <th>{{ t('adminTrafficImport.date') }}</th>
                <th>{{ t('adminTrafficImport.upload') }}</th>
                <th>{{ t('adminTrafficImport.download') }}</th>
                <th>{{ t('adminTrafficImport.totalPreview') }}</th>
                <th>{{ t('common.action') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(r, i) in rows" :key="i">
                <td class="num-font text-[var(--c-text-sub)]">{{ i + 1 }}</td>
                <td>
                  <n-input-number
                    v-model:value="r.user_id"
                    :min="1"
                    :placeholder="t('adminTrafficImport.userIdPlaceholder')"
                    class="w-32"
                  />
                </td>
                <td>
                  <n-date-picker
                    v-model:formatted-value="r.date"
                    value-format="yyyy-MM-dd"
                    type="date"
                    class="w-36"
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="r.u"
                    :min="0"
                    :placeholder="t('adminTrafficImport.uploadPlaceholder')"
                    class="w-44"
                  />
                </td>
                <td>
                  <n-input-number
                    v-model:value="r.d"
                    :min="0"
                    :placeholder="t('adminTrafficImport.downloadPlaceholder')"
                    class="w-44"
                  />
                </td>
                <td class="num-font text-14 text-[var(--c-text-sub)]">
                  {{ formatBytes(Number(r.u) + Number(r.d)) }}
                </td>
                <td>
                  <button class="btn-soft-danger h-7 px-3 text-14" @click="removeRow(i)">
                    {{ t('adminTrafficImport.remove') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </n-table>

          <div
            class="mt-4 rounded-lg bg-[var(--c-warning-bg)] px-4 py-3 text-14 text-[var(--c-warning)]"
          >
            <AppIcon name="info" :size="15" class="mr-1 inline" />
            {{ t('adminTrafficImport.tip1') }}
            {{ t('adminTrafficImport.tip2') }}
          </div>
        </div>
      </n-tab-pane>

      <!-- F16 重置 -->
      <n-tab-pane name="reset" :tab="t('adminTrafficImport.tabReset')">
        <div class="card-base max-w-[560px]">
          <n-form label-placement="top">
            <n-form-item :label="t('adminTrafficImport.resetIdsLabel')">
              <n-input
                v-model:value="resetUserIds"
                type="textarea"
                :rows="2"
                :placeholder="t('adminTrafficImport.resetIdsPlaceholder')"
              />
            </n-form-item>
            <n-form-item :label="t('adminTrafficImport.resetModeLabel')">
              <n-radio-group v-model:value="resetMode">
                <n-radio value="clear_usage">{{ t('adminTrafficImport.modeClearUsage') }}</n-radio>
                <n-radio value="reset_quota">{{ t('adminTrafficImport.modeResetQuota') }}</n-radio>
              </n-radio-group>
            </n-form-item>
            <div
              class="rounded-lg bg-[var(--c-warning-bg)] px-4 py-3 text-14 text-[var(--c-warning)]"
            >
              <AppIcon name="info" :size="15" class="mr-1 inline" />
              {{ t('adminTrafficImport.resetTipSnapshot') }}
            </div>
            <div class="mt-4">
              <button
                class="btn-danger h-9 px-4 text-14"
                :disabled="resetting"
                @click="submitReset"
              >
                {{ t('adminTrafficImport.resetAction') }}
              </button>
            </div>
          </n-form>
        </div>
      </n-tab-pane>

      <!-- F16 重置记录 -->
      <n-tab-pane name="logs" :tab="t('adminTrafficImport.tabLogs')">
        <div class="card-base overflow-x-auto">
          <div class="mb-3 flex flex-wrap items-center gap-2">
            <n-input-number
              v-model:value="logUserId"
              clearable
              :min="1"
              :placeholder="t('adminTrafficImport.logFilterPlaceholder')"
              class="w-44"
            />
            <button class="btn-soft-primary h-8 px-3 text-14" @click="searchLogs">
              {{ t('adminTrafficImport.logSearch') }}
            </button>
          </div>
          <n-spin :show="logsLoading">
            <n-table :bordered="false" :single-line="false" class="min-w-[860px]">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>{{ t('adminTrafficImport.userId') }}</th>
                  <th>{{ t('adminTrafficImport.logEmail') }}</th>
                  <th>{{ t('adminTrafficImport.resetModeLabel') }}</th>
                  <th>{{ t('adminTrafficImport.logBeforeUsed') }}</th>
                  <th>{{ t('adminTrafficImport.logBeforeQuota') }}</th>
                  <th>{{ t('adminTrafficImport.logAfterQuota') }}</th>
                  <th>{{ t('adminTrafficImport.logAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="l in logs" :key="l.id">
                  <td class="num-font">{{ l.id }}</td>
                  <td class="num-font">{{ l.user_id }}</td>
                  <td class="text-14">{{ l.user_email }}</td>
                  <td>
                    <StatusBadge :type="l.mode === 'reset_quota' ? 'primary' : 'warning'">
                      {{ modeText(l.mode) }}
                    </StatusBadge>
                  </td>
                  <td class="num-font text-14">
                    {{ formatBytes(l.before_u + l.before_d) }}
                  </td>
                  <td class="num-font text-14">{{ formatBytes(l.before_transfer_enable) }}</td>
                  <td class="num-font text-14">{{ formatBytes(l.after_transfer_enable) }}</td>
                  <td class="text-14 text-[var(--c-text-sub)]">
                    {{ new Date(l.created_at).toLocaleString() }}
                  </td>
                </tr>
                <tr v-if="!logsLoading && logs.length === 0">
                  <td colspan="8"><EmptyState :text="t('adminTrafficImport.logEmpty')" /></td>
                </tr>
              </tbody>
            </n-table>
          </n-spin>
          <div class="mt-4 flex justify-end">
            <n-pagination
              :page="logPage"
              :page-size="logPageSize"
              :item-count="logTotal"
              @update:page="
                (p: number) => {
                  logPage = p
                  void loadLogs()
                }
              "
            />
          </div>
        </div>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>
