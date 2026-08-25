<script setup lang="ts">
/**
 * 管理后台 · 工单管理:全量工单列表(含用户邮箱)/ 详情消息流 / 客服回复 / 关闭。
 * 数据:GET /admin/tickets、GET /admin/tickets/{id}、POST reply|close(docs/api/README.md §16)
 */
import { onMounted, ref, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminTicketDetail, AdminTicketItem, TicketLevel, TicketStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminTicketItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10

// 详情弹窗
const modalVisible = ref(false)
const detail = ref<AdminTicketDetail | null>(null)
const detailLoading = ref(false)
const replyText = ref('')
const sending = ref(false)
const msgListRef = ref<HTMLElement | null>(null)

const LEVEL_KEY: Record<TicketLevel, string> = {
  0: 'adminTickets.low',
  1: 'adminTickets.medium',
  2: 'adminTickets.high',
}

const STATUS_KEY: Record<TicketStatus, string> = {
  0: 'adminTickets.pending',
  1: 'adminTickets.replied',
  2: 'adminTickets.closed',
}

function levelText(level: TicketLevel): string {
  return t(LEVEL_KEY[level])
}

function levelType(level: TicketLevel): 'neutral' | 'warning' | 'danger' {
  return level === 0 ? 'neutral' : level === 1 ? 'warning' : 'danger'
}

function statusText(status: TicketStatus): string {
  return t(STATUS_KEY[status])
}

function statusType(status: TicketStatus): 'warning' | 'primary' | 'neutral' {
  return status === 0 ? 'warning' : status === 1 ? 'primary' : 'neutral'
}

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.tickets({ page: page.value, page_size: pageSize })
    list.value = res.list
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  void load()
}

function scrollToBottom() {
  void nextTick(() => {
    msgListRef.value?.scrollTo({ top: msgListRef.value.scrollHeight })
  })
}

async function openDetail(t: AdminTicketItem) {
  modalVisible.value = true
  detailLoading.value = true
  replyText.value = ''
  try {
    detail.value = await apiAdmin.ticketDetail(t.id)
    scrollToBottom()
  } finally {
    detailLoading.value = false
  }
}

const isClosed = () => detail.value?.status === 2

async function sendReply() {
  if (sending.value || !detail.value || !replyText.value.trim() || isClosed()) return
  sending.value = true
  try {
    await apiAdmin.replyTicket(detail.value.id, { message: replyText.value.trim() })
    detail.value = await apiAdmin.ticketDetail(detail.value.id)
    replyText.value = ''
    scrollToBottom()
    message.success(t('adminTickets.replySuccess'))
    void load()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    sending.value = false
  }
}

function onClose() {
  if (!detail.value) return
  dialog.warning({
    title: t('adminTickets.closeTitle'),
    content: t('adminTickets.closeConfirm'),
    positiveText: t('adminTickets.closeBtn'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.closeTicket(detail.value!.id)
      message.success(t('adminTickets.closeSuccess'))
      detail.value = await apiAdmin.ticketDetail(detail.value!.id)
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminTickets.title')" :subtitle="t('adminTickets.subtitle')">
      <template #actions>
        <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
          <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
        </button>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[820px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('adminTickets.user') }}</th>
              <th>{{ t('adminTickets.subject') }}</th>
              <th>{{ t('adminTickets.level') }}</th>
              <th>{{ t('adminTickets.status') }}</th>
              <th>{{ t('adminTickets.lastReply') }}</th>
              <th>{{ t('adminTickets.createdAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tk in list" :key="tk.id">
              <td>{{ tk.id }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ tk.user_email }}</td>
              <td class="max-w-60 truncate font-500 text-[var(--c-text)]">{{ tk.subject }}</td>
              <td>
                <StatusBadge :type="levelType(tk.level)">
                  {{ levelText(tk.level) }}
                </StatusBadge>
              </td>
              <td>
                <StatusBadge :type="statusType(tk.status)">
                  {{ statusText(tk.status) }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ tk.last_reply_at ? tk.last_reply_at.slice(0, 16).replace('T', ' ') : '-' }}
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ tk.created_at.slice(0, 16).replace('T', ' ') }}
              </td>
              <td>
                <div class="flex">
                  <button class="btn-soft-blue h-7 px-3 text-14" @click="openDetail(tk)">
                    {{ t('adminTickets.view') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="8"><EmptyState :text="t('adminTickets.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
      <div class="flex justify-end p-4">
        <n-pagination
          v-model:page="page"
          :item-count="total"
          :page-size="pageSize"
          @update:page="onPageChange"
        />
      </div>
    </div>

    <!-- 工单详情弹窗(屏幕中央) -->
    <n-modal
      v-model:show="modalVisible"
      preset="card"
      style="width: 640px; max-width: calc(100vw - 32px)"
    >
      <template #header>
        <div class="min-w-0">
          <div class="truncate text-16 font-600 text-[var(--c-text)]">
            {{ detail?.subject }}
          </div>
          <div v-if="detail" class="mt-1 flex items-center gap-2">
            <StatusBadge :type="statusType(detail.status)">
              {{ statusText(detail.status) }}
            </StatusBadge>
            <span class="text-14 text-[var(--c-text-sub)]">
              {{
                t('adminTickets.created', {
                  time: detail.created_at.slice(0, 16).replace('T', ' '),
                })
              }}
            </span>
          </div>
        </div>
      </template>

      <n-spin :show="detailLoading">
        <div
          v-if="detail"
          ref="msgListRef"
          class="max-h-[40vh] space-y-3 overflow-y-auto px-3 py-2"
        >
          <div
            v-for="m in detail.messages"
            :key="m.id"
            class="flex gap-2"
            :class="m.sender_type === 0 ? 'flex-row' : 'flex-row-reverse'"
          >
            <span
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-white"
              :style="
                m.sender_type === 0
                  ? 'background: linear-gradient(135deg,#6558f5,#8b5cf6)'
                  : 'background: var(--c-olive)'
              "
            >
              <AppIcon :name="m.sender_type === 0 ? 'user' : 'headset'" :size="15" />
            </span>
            <div
              class="max-w-[75%] rounded-2xl px-4 py-2.5 text-14 leading-relaxed"
              :class="
                m.sender_type === 0
                  ? 'rounded-tl-sm bg-[var(--c-bg-hover)] text-[var(--c-text)]'
                  : 'rounded-tr-sm bg-[var(--c-primary-soft)] text-[var(--c-primary-text)]'
              "
            >
              <div class="mb-1 flex items-center justify-between gap-3">
                <span class="text-14 font-500">
                  {{
                    m.sender_type === 0 ? t('adminTickets.userRole') : t('adminTickets.agentRole')
                  }}
                </span>
                <span class="text-14 opacity-60">
                  {{ m.created_at.slice(0, 16).replace('T', ' ') }}
                </span>
              </div>
              <div class="whitespace-pre-wrap">{{ m.message }}</div>
            </div>
          </div>
          <EmptyState v-if="detail.messages.length === 0" :text="t('adminTickets.emptyMessages')" />
        </div>
      </n-spin>

      <!-- 回复区 -->
      <div v-if="detail" class="mt-4 border-t border-[var(--c-border)] pt-4">
        <n-input
          v-model:value="replyText"
          type="textarea"
          :rows="3"
          :disabled="isClosed()"
          :placeholder="
            isClosed() ? t('adminTickets.closedPlaceholder') : t('adminTickets.replyPlaceholder')
          "
        />
        <div class="mt-3 flex justify-end gap-2">
          <button class="btn-soft-danger h-9 px-4 text-14" :disabled="isClosed()" @click="onClose">
            {{ isClosed() ? t('adminTickets.closedBtn') : t('adminTickets.closeTicket') }}
          </button>
          <button
            class="btn-primary h-9 px-5 text-14"
            :disabled="sending || isClosed() || !replyText.trim()"
            @click="sendReply"
          >
            {{ t('adminTickets.reply') }}
          </button>
        </div>
      </div>
    </n-modal>
  </div>
</template>
