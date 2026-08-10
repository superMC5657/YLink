<script setup lang="ts">
/**
 * 管理后台 · 工单管理:全量工单列表(含用户邮箱)/ 详情消息流 / 客服回复 / 关闭。
 * 数据:GET /admin/tickets、GET /admin/tickets/{id}、POST reply|close(docs/api/README.md §16)
 */
import { onMounted, ref, nextTick } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminTicketDetail, AdminTicketItem, TicketLevel, TicketStatus } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminTicketItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10

// 详情抽屉
const drawer = ref(false)
const detail = ref<AdminTicketDetail | null>(null)
const detailLoading = ref(false)
const replyText = ref('')
const sending = ref(false)
const msgListRef = ref<HTMLElement | null>(null)

const LEVEL_LABEL: Record<TicketLevel, { text: string; type: 'neutral' | 'warning' | 'danger' }> = {
  0: { text: '低', type: 'neutral' },
  1: { text: '中', type: 'warning' },
  2: { text: '高', type: 'danger' },
}

const STATUS_LABEL: Record<
  TicketStatus,
  { text: string; type: 'warning' | 'primary' | 'neutral' }
> = {
  0: { text: '待回复', type: 'warning' },
  1: { text: '已回复', type: 'primary' },
  2: { text: '已关闭', type: 'neutral' },
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
  drawer.value = true
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
    message.success('回复成功')
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
    title: '关闭工单',
    content: '确定关闭该工单吗?关闭后用户无法继续回复。',
    positiveText: '关闭',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.closeTicket(detail.value!.id)
      message.success('工单已关闭')
      detail.value = await apiAdmin.ticketDetail(detail.value!.id)
      void load()
    },
  })
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="工单管理" subtitle="全量工单(含发起用户)/ 客服回复 / 关闭">
      <template #actions>
        <button class="btn-ghost h-9 px-3 text-14" @click="load">
          <AppIcon name="refresh" :size="15" /> 刷新
        </button>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[820px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户</th>
              <th>主题</th>
              <th>优先级</th>
              <th>状态</th>
              <th>最后回复</th>
              <th>创建时间</th>
              <th class="text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in list" :key="t.id">
              <td>{{ t.id }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ t.user_email }}</td>
              <td class="max-w-60 truncate font-500 text-[var(--c-text)]">{{ t.subject }}</td>
              <td>
                <StatusBadge :type="LEVEL_LABEL[t.level]?.type ?? 'neutral'">
                  {{ LEVEL_LABEL[t.level]?.text ?? t.level }}
                </StatusBadge>
              </td>
              <td>
                <StatusBadge :type="STATUS_LABEL[t.status]?.type ?? 'neutral'">
                  {{ STATUS_LABEL[t.status]?.text ?? t.status }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ t.last_reply_at ? t.last_reply_at.slice(0, 16).replace('T', ' ') : '-' }}
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ t.created_at.slice(0, 16).replace('T', ' ') }}
              </td>
              <td>
                <div class="flex justify-end">
                  <button class="btn-ghost h-7 px-3 text-14" @click="openDetail(t)">查看</button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="8"><EmptyState text="暂无工单" /></td>
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

    <!-- 工单详情抽屉 -->
    <n-drawer v-model:show="drawer" :width="520">
      <n-drawer-content closable>
        <template #header>
          <div class="min-w-0">
            <div class="truncate text-16 font-600 text-[var(--c-text)]">
              {{ detail?.subject }}
            </div>
            <div v-if="detail" class="mt-1 flex items-center gap-2">
              <StatusBadge :type="STATUS_LABEL[detail.status]?.type ?? 'neutral'">
                {{ STATUS_LABEL[detail.status]?.text ?? detail.status }}
              </StatusBadge>
              <span class="text-14 text-[var(--c-text-sub)]">
                创建于 {{ detail.created_at.slice(0, 16).replace('T', ' ') }}
              </span>
            </div>
          </div>
        </template>

        <n-spin :show="detailLoading">
          <div v-if="detail" ref="msgListRef" class="max-h-[55vh] space-y-3 overflow-y-auto py-2">
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
                    {{ m.sender_type === 0 ? '用户' : '客服' }}
                  </span>
                  <span class="text-14 opacity-60">
                    {{ m.created_at.slice(0, 16).replace('T', ' ') }}
                  </span>
                </div>
                <div class="whitespace-pre-wrap">{{ m.message }}</div>
              </div>
            </div>
            <EmptyState v-if="detail.messages.length === 0" text="暂无消息" />
          </div>
        </n-spin>

        <!-- 回复区 -->
        <div v-if="detail" class="mt-4 border-t border-[var(--c-border)] pt-4">
          <n-input
            v-model:value="replyText"
            type="textarea"
            :rows="3"
            :disabled="isClosed()"
            :placeholder="isClosed() ? '工单已关闭,无法回复' : '输入回复内容…'"
          />
          <div class="mt-3 flex justify-end gap-2">
            <button class="btn-ghost h-9 px-4 text-14" :disabled="isClosed()" @click="onClose">
              {{ isClosed() ? '已关闭' : '关闭工单' }}
            </button>
            <button
              class="btn-primary h-9 px-5 text-14"
              :disabled="sending || isClosed() || !replyText.trim()"
              @click="sendReply"
            >
              回复
            </button>
          </div>
        </div>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>
