<script setup lang="ts">
/**
 * 工单详情:对话气泡流(用户右/客服左),底部回复输入框,支持关闭/重新打开。
 * 数据:GET /tickets/{id}、POST /tickets/{id}/reply、/close(契约 §13.2)
 */
import { computed, onMounted, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTicketStore } from '@/stores/ticket'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { formatTime, ticketLevelLabel, ticketStatusLabel } from '@/utils/format'

const ticket = useTicketStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const loading = ref(true)
const replyText = ref('')
const sending = ref(false)
const listRef = ref<HTMLElement | null>(null)

const detail = computed(() => ticket.detail)
const isClosed = computed(() => detail.value?.status === 2)

function scrollToBottom() {
  void nextTick(() => {
    listRef.value?.scrollTo({ top: listRef.value.scrollHeight, behavior: 'smooth' })
  })
}

async function sendReply() {
  if (sending.value || !replyText.value.trim() || isClosed.value) return
  sending.value = true
  try {
    await ticket.reply(replyText.value.trim())
    replyText.value = ''
    scrollToBottom()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    sending.value = false
  }
}

function onClose() {
  dialog.warning({
    title: t('ticket.closeTicket'),
    content: t('ticket.closeConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await ticket.close()
      message.success(t('ticket.closed'))
    },
  })
}

onMounted(async () => {
  const id = Number(route.params.id)
  try {
    await ticket.fetchDetail(id)
    scrollToBottom()
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="flex h-[calc(100vh-13rem)] flex-col">
    <!-- 头部 -->
    <div class="mb-4 flex items-center gap-3">
      <button
        class="flex h-9 w-9 cursor-pointer items-center justify-center rounded-full text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
        @click="router.back()"
      >
        <AppIcon name="arrow-left" :size="18" />
      </button>
      <div class="min-w-0 flex-1">
        <h1 class="truncate text-16 font-600 text-[var(--c-text)]">{{ detail?.subject }}</h1>
        <div class="flex items-center gap-2 text-12 text-[var(--c-text-sub)]">
          <span>{{ t('ticket.level') }}:{{ ticketLevelLabel(detail?.level ?? 0) }}</span>
          <StatusBadge v-if="detail" :type="detail.status === 2 ? 'neutral' : detail.status === 1 ? 'success' : 'warning'">
            {{ ticketStatusLabel(detail.status) }}
          </StatusBadge>
        </div>
      </div>
      <button
        v-if="detail && !isClosed"
        class="btn-ghost h-8 shrink-0 px-3 text-12 text-[var(--c-danger)]"
        @click="onClose"
      >
        {{ t('ticket.closeTicket') }}
      </button>
    </div>

    <!-- 消息流 -->
    <div ref="listRef" class="card-base flex-1 space-y-4 overflow-y-auto p-4 md:p-6">
      <n-spin :show="loading">
        <div v-if="detail" class="space-y-4">
          <div
            v-for="m in detail.messages"
            :key="m.id"
            class="flex items-end gap-2.5"
            :class="m.sender_type === 0 ? 'flex-row-reverse' : ''"
          >
            <span
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-white"
              :style="m.sender_type === 0 ? 'background: linear-gradient(135deg,#6558F5,#8B5CF6)' : 'background: var(--c-bg-hover); color: var(--c-text-sub)'"
            >
              <AppIcon :name="m.sender_type === 0 ? 'user' : 'headset'" :size="16" />
            </span>
            <div class="flex max-w-[75%] flex-col" :class="m.sender_type === 0 ? 'items-end' : 'items-start'">
              <div
                class="rounded-2xl px-4 py-2.5 text-13 leading-6 whitespace-pre-wrap"
                :style="
                  m.sender_type === 0
                    ? 'background: linear-gradient(135deg,#6558F5,#8B5CF6); color: #fff; border-bottom-right-radius: 4px'
                    : 'background: var(--c-bg-hover); color: var(--c-text); border-bottom-left-radius: 4px'
                "
              >
                {{ m.message }}
              </div>
              <span class="mt-1 text-11 text-[var(--c-text-sub)]">{{ formatTime(m.created_at) }}</span>
            </div>
          </div>
        </div>
      </n-spin>
    </div>

    <!-- 回复区 -->
    <div class="mt-4">
      <div v-if="isClosed" class="mb-3 flex items-center justify-center gap-2 rounded-xl p-3 text-13" style="background: var(--c-bg-hover); color: var(--c-text-sub)">
        <AppIcon name="alert" :size="15" />
        {{ t('ticket.closedTip') }}
      </div>
      <div class="flex items-end gap-2">
        <textarea
          v-model="replyText"
          rows="2"
          :placeholder="isClosed ? t('ticket.closedTip') : t('ticket.replyPlaceholder')"
          :disabled="isClosed"
          class="flex-1 resize-none rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] p-3 text-13 text-[var(--c-text)] outline-none transition-colors placeholder:text-[var(--c-text-sub)] focus:border-[var(--c-primary)] disabled:opacity-50"
          @keydown.enter.exact.prevent="sendReply"
        />
        <button
          class="btn-primary h-11 shrink-0 px-5 text-13"
          :disabled="isClosed || sending || !replyText.trim()"
          @click="sendReply"
        >
          <AppIcon name="send" :size="15" />
          {{ t('ticket.send') }}
        </button>
      </div>
    </div>
  </div>
</template>
