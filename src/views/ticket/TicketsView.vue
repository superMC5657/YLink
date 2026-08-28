<script setup lang="ts">
/**
 * 我的工单:卡片式表格 + 新建弹窗(主题/优先级/内容)。
 * 数据:GET /tickets、POST /tickets(契约 §13.1)
 */
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTicketStore } from '@/stores/ticket'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { formatTime, ticketLevelLabel, ticketStatusLabel } from '@/utils/format'
import PageHeader from '@/components/ui/PageHeader.vue'
import type { TicketLevel } from '@/types/api'

const ticket = useTicketStore()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const showCreate = ref(false)
const creating = ref(false)
const form = ref({ subject: '', level: 1 as TicketLevel, message: '' })

const levelOptions = [
  { label: t('ticket.low'), value: 0 },
  { label: t('ticket.medium'), value: 1 },
  { label: t('ticket.high'), value: 2 },
]

function statusType(status: number) {
  const map: Record<number, 'success' | 'warning' | 'neutral'> = {
    0: 'warning',
    1: 'success',
    2: 'neutral',
  }
  return map[status] ?? 'neutral'
}

const canCreate = computed(
  () => form.value.subject.trim().length > 0 && form.value.message.trim().length > 0,
)

async function onCreate() {
  if (creating.value || !canCreate.value) return
  creating.value = true
  try {
    const ticketCreated = await ticket.create({ ...form.value })
    message.success(t('ticket.created'))
    showCreate.value = false
    form.value = { subject: '', level: 1, message: '' }
    router.push(`/tickets/${ticketCreated.id}`)
  } finally {
    creating.value = false
  }
}

function goDetail(id: number) {
  router.push(`/tickets/${id}`)
}

onMounted(() => void ticket.fetch())
</script>

<template>
  <div>
    <PageHeader :title="t('ticket.title')">
      <template #actions>
        <button class="btn-primary h-9 px-4 text-14" @click="showCreate = true">
          <AppIcon name="plus" :size="15" />
          {{ t('ticket.newTicket') }}
        </button>
      </template>
    </PageHeader>

    <div class="card-base p-4 md:p-6">
      <n-spin :show="ticket.loading">
        <div class="space-y-3">
          <button
            v-for="tk in ticket.list"
            :key="tk.id"
            class="flex w-full cursor-pointer items-center gap-4 rounded-xl border border-[var(--c-border)] p-4 text-left transition-all hover:border-[var(--c-primary)] hover:bg-[var(--c-bg-hover)]"
            @click="goDetail(tk.id)"
          >
            <span
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full"
              :style="
                tk.level === 2
                  ? { backgroundColor: 'var(--c-danger-bg)', color: 'var(--c-danger)' }
                  : tk.level === 1
                    ? { backgroundColor: 'var(--c-warning-bg)', color: 'var(--c-warning)' }
                    : { backgroundColor: 'var(--c-bg-hover)', color: 'var(--c-text-sub)' }
              "
            >
              <AppIcon name="ticket" :size="18" />
            </span>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="truncate text-14 font-500 text-[var(--c-text)]">{{ tk.subject }}</span>
                <StatusBadge v-if="tk.type === 1" type="primary">
                  {{ t('ticket.withdrawType') }}
                </StatusBadge>
                <span class="shrink-0 text-14 text-[var(--c-text-sub)]"
                  >{{ t('ticket.level') }}:{{ ticketLevelLabel(tk.level) }}</span
                >
              </div>
              <div class="mt-1 text-14 text-[var(--c-text-sub)]">
                {{ t('ticket.lastReplyAt') }}:{{ formatTime(tk.last_reply_at) }}
              </div>
            </div>

            <div class="flex shrink-0 flex-col items-end gap-1.5">
              <StatusBadge :type="statusType(tk.status)">{{
                ticketStatusLabel(tk.status)
              }}</StatusBadge>
              <span class="text-14 text-[var(--c-text-sub)]">{{
                formatTime(tk.created_at, false)
              }}</span>
            </div>
          </button>
        </div>

        <EmptyState
          v-if="!ticket.loading && ticket.list.length === 0"
          :text="t('ticket.empty')"
          :icon="'ticket'"
        />
      </n-spin>
    </div>

    <!-- 新建工单 -->
    <n-modal
      v-model:show="showCreate"
      preset="card"
      :title="t('ticket.createTitle')"
      class="max-w-105"
    >
      <div class="space-y-4">
        <div>
          <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{
            t('ticket.subject')
          }}</label>
          <input
            v-model="form.subject"
            type="text"
            :placeholder="t('ticket.subjectPlaceholder')"
            class="h-10 w-full rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] px-3 text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
          />
        </div>
        <div>
          <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{ t('ticket.level') }}</label>
          <n-radio-group v-model:value="form.level">
            <n-radio-button
              v-for="o in levelOptions"
              :key="o.value"
              :value="o.value"
              :label="o.label"
            />
          </n-radio-group>
        </div>
        <div>
          <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{
            t('ticket.messagePlaceholder')
          }}</label>
          <textarea
            v-model="form.message"
            rows="4"
            :placeholder="t('ticket.messagePlaceholder')"
            class="w-full resize-none rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] p-3 text-14 text-[var(--c-text)] outline-none transition-colors placeholder:text-[var(--c-text-sub)] focus:border-[var(--c-primary)]"
          />
        </div>
        <button
          class="btn-primary h-10 w-full text-14"
          :disabled="!canCreate || creating"
          @click="onCreate"
        >
          {{ t('common.submit') }}
        </button>
      </div>
    </n-modal>
  </div>
</template>
