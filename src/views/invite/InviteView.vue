<script setup lang="ts">
/**
 * 邀请赚钱(截图4):左列表(邀请码 + 佣金记录)、右统计卡(5 张)+ 划转。
 * 数据:GET /invite/*(docs/api/README.md §11)
 */
import { onMounted, ref } from 'vue'
import { useInviteStore } from '@/stores/invite'
import { useUserStore } from '@/stores/user'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import { formatMoney, formatTime } from '@/utils/format'
import { copyText } from '@/utils/platform'
import StatNumber from '@/components/ui/StatNumber.vue'
import PageHeader from '@/components/ui/PageHeader.vue'

const invite = useInviteStore()
const user = useUserStore()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
const isDesktop = useMediaQuery('(min-width: 1024px)')

const showTransfer = ref(false)
const transferAmount = ref<number | null>(null)
const transferring = ref(false)

async function onCreateCode() {
  try {
    const code = await invite.createCode()
    message.success('邀请码已生成')
    void copyText(code.code)
  } catch (e) {
    message.error((e as Error).message)
  }
}

async function onTransfer() {
  if (transferring.value) return
  const amount = transferAmount.value
  if (!amount || amount <= 0) {
    message.warning('请输入划转金额')
    return
  }
  transferring.value = true
  try {
    const data = await invite.transfer(amount)
    message.success(t('invite.transferSuccess'))
    user.stat!.balance = data.balance
    showTransfer.value = false
    transferAmount.value = null
    await invite.fetchSummary()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    transferring.value = false
  }
}

async function copyCode(code: string) {
  await copyText(code)
  message.success(t('common.copied'))
}

function onDeleteCode(code: string) {
  dialog.warning({
    title: t('invite.deleteCode'),
    content: t('invite.deleteCodeConfirm', { code }),
    positiveText: t('invite.deleteCode'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await invite.deleteCode(code)
        message.success(t('invite.deleteCodeSuccess'))
      } catch (e) {
        message.error((e as Error).message)
      }
    },
  })
}

async function copyRegisterLink() {
  const prefix = invite.registerUrlPrefix
  const first = invite.codes[0]
  if (!prefix || !first) {
    message.warning('请先生成邀请码')
    return
  }
  await copyText(prefix + first.code)
  message.success(t('common.copied'))
}

onMounted(() => {
  void invite.refreshAll()
})
</script>

<template>
  <div>
    <PageHeader :title="t('invite.title')" />

    <div class="flex flex-col gap-5 lg:flex-row">
      <!-- 左列:邀请码 + 佣金记录 -->
      <div class="min-w-0 flex-1 space-y-5">
        <!-- 邀请码卡 -->
        <div class="card-base p-5 md:p-6">
          <div class="mb-4 flex items-center justify-between">
            <h3 class="text-16 font-600 text-[var(--c-text)]">{{ t('invite.inviteCodes') }}</h3>
            <button class="btn-olive h-9 px-4 text-14" @click="onCreateCode">
              <AppIcon name="plus" :size="15" />
              {{ t('invite.createCode') }}
            </button>
          </div>

          <div
            class="mb-4 flex flex-wrap items-center gap-2 rounded-xl p-3"
            style="background-color: var(--c-bg-hover)"
          >
            <span class="text-14 text-[var(--c-text-sub)]">{{ t('invite.registerLink') }}:</span>
            <span class="num min-w-0 flex-1 truncate text-14 text-[var(--c-text)]">
              {{ invite.registerUrlPrefix }}{{ invite.codes[0]?.code ?? '------' }}
            </span>
            <button class="btn-soft-blue h-7 px-3 text-14" @click="copyRegisterLink">
              <AppIcon name="copy" :size="13" />
              {{ t('common.copy') }}
            </button>
          </div>

          <div v-if="isDesktop" class="w-full overflow-x-auto">
            <n-table :bordered="false" class="w-full">
              <thead>
                <tr>
                  <th class="text-14">{{ t('invite.code') }}</th>
                  <th class="text-14">{{ t('invite.usedCount') }}</th>
                  <th class="text-14">{{ t('invite.createdAt') }}</th>
                  <th class="text-14">{{ t('common.action') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="c in invite.codes"
                  :key="c.code"
                  class="transition-colors hover:bg-[var(--c-bg-hover)]"
                >
                  <td class="num font-600 text-[var(--c-text)]">{{ c.code }}</td>
                  <td class="num text-[var(--c-text)]">{{ c.used_count }}</td>
                  <td class="text-14 text-[var(--c-text-sub)]">
                    {{ formatTime(c.created_at, false) }}
                  </td>
                  <td>
                    <div class="flex items-center gap-3">
                      <button
                        class="btn-soft-blue h-7 px-2.5 text-14"
                        @click="copyCode(c.code)"
                      >
                        <AppIcon name="copy" :size="13" />
                        {{ t('common.copy') }}
                      </button>
                      <button
                        class="btn-soft-danger h-7 px-2.5 text-14"
                        @click="onDeleteCode(c.code)"
                      >
                        <AppIcon name="trash" :size="13" />
                        {{ t('invite.deleteCode') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </n-table>
          </div>

          <!-- 移动端卡片 -->
          <div v-else class="space-y-2">
            <div
              v-for="c in invite.codes"
              :key="c.code"
              class="flex items-center justify-between rounded-xl p-3"
              style="background-color: var(--c-bg-hover)"
            >
              <div>
                <div class="num text-14 font-600 text-[var(--c-text)]">{{ c.code }}</div>
                <div class="text-14 text-[var(--c-text-sub)]">
                  {{ t('invite.usedCount') }}:{{ c.used_count }} ·
                  {{ formatTime(c.created_at, false) }}
                </div>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button class="btn-soft-blue h-8 px-3 text-14" @click="copyCode(c.code)">
                  <AppIcon name="copy" :size="13" />
                  {{ t('common.copy') }}
                </button>
                <button class="btn-soft-danger h-8 px-2.5 text-14" @click="onDeleteCode(c.code)">
                  <AppIcon name="trash" :size="13" />
                </button>
              </div>
            </div>
            <EmptyState
              v-if="invite.codes.length === 0"
              :text="t('invite.emptyCodes')"
              :icon="'gift'"
            />
          </div>

          <p class="mt-3 text-14 text-[var(--c-text-sub)]">
            {{ t('invite.codeLimit', { limit: invite.codeLimit }) }}
          </p>
        </div>

        <!-- 佣金记录 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">
            {{ t('invite.commissionRecords') }}
          </h3>
          <div v-if="isDesktop" class="w-full overflow-x-auto">
            <n-table :bordered="false" class="w-full">
              <thead>
                <tr>
                  <th class="text-14">{{ t('order.orderNo') }}</th>
                  <th class="text-14">{{ t('invite.amount') }}</th>
                  <th class="text-14">{{ t('invite.rate') }}</th>
                  <th class="text-14">{{ t('invite.recordTime') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="r in invite.records"
                  :key="r.order_no"
                  class="transition-colors hover:bg-[var(--c-bg-hover)]"
                >
                  <td><CopyText :text="r.order_no" :max-chars="14" /></td>
                  <td class="num font-600 text-[var(--c-pink)]">+{{ formatMoney(r.amount) }}</td>
                  <td class="num text-14 text-[var(--c-text-sub)]">{{ r.rate }}%</td>
                  <td class="text-14 text-[var(--c-text-sub)]">
                    {{ formatTime(r.confirmed_at, false) }}
                  </td>
                </tr>
              </tbody>
            </n-table>
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="r in invite.records"
              :key="r.order_no"
              class="flex items-center justify-between rounded-xl p-3"
              style="background-color: var(--c-bg-hover)"
            >
              <div class="min-w-0">
                <div class="num text-14 font-600 text-[var(--c-pink)]">
                  +{{ formatMoney(r.amount) }}
                </div>
                <div class="truncate text-14 text-[var(--c-text-sub)]">{{ r.order_no }}</div>
              </div>
              <div class="text-right">
                <div class="text-14 text-[var(--c-text)]">{{ r.rate }}%</div>
                <div class="text-14 text-[var(--c-text-sub)]">
                  {{ formatTime(r.confirmed_at, false) }}
                </div>
              </div>
            </div>
            <EmptyState
              v-if="invite.records.length === 0"
              :text="t('invite.emptyRecords')"
              :icon="'coins'"
            />
          </div>
        </div>
      </div>

      <!-- 右列:统计卡 -->
      <div class="w-full shrink-0 space-y-4 lg:w-80">
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-1">
          <StatNumber
            :label="t('invite.myCommission')"
            :value="formatMoney(invite.summary?.commission_balance ?? 0)"
            color="var(--c-pink)"
            icon="coins"
            icon-color="var(--c-pink)"
          />
          <StatNumber
            :label="t('invite.commissionRate')"
            :value="`${invite.summary?.commission_rate ?? 0}%`"
            icon="agent"
            icon-color="var(--c-primary)"
          />
          <StatNumber
            :label="t('invite.registeredUsers')"
            :value="invite.summary?.registered_count ?? 0"
            icon="users"
            icon-color="var(--c-success)"
          />
          <StatNumber
            :label="t('invite.totalCommission')"
            :value="formatMoney(invite.summary?.total_commission ?? 0)"
            color="var(--c-warning)"
            icon="award"
            icon-color="var(--c-warning)"
          />
          <StatNumber
            :label="t('invite.pendingCommission')"
            :value="formatMoney(invite.summary?.pending_commission ?? 0)"
            color="var(--c-text-sub)"
            icon="clock"
            icon-color="var(--c-text-sub)"
          />
        </div>

        <!-- 划转按钮 -->
        <button class="btn-primary h-11 w-full text-14" @click="showTransfer = true">
          <AppIcon name="refresh" :size="16" />
          {{ t('invite.transfer') }}
        </button>
      </div>
    </div>

    <!-- 划转弹窗 -->
    <n-modal
      v-model:show="showTransfer"
      preset="card"
      :title="t('invite.transferModalTitle')"
      class="max-w-95"
    >
      <div class="space-y-4">
        <div>
          <div class="mb-1.5 text-14 text-[var(--c-text-sub)]">
            {{
              t('invite.transferable', {
                amount: formatMoney(invite.summary?.commission_balance ?? 0),
              })
            }}
          </div>
          <n-input-number
            v-model:value="transferAmount"
            :min="0.01"
            :max="invite.summary?.commission_balance ?? 0"
            :step="1"
            placeholder="0.00"
            class="w-full"
          />
        </div>
        <button
          class="btn-primary h-10 w-full text-14"
          :disabled="transferring"
          @click="onTransfer"
        >
          <span
            v-if="transferring"
            class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
          />
          {{ t('common.confirm') }}
        </button>
      </div>
    </n-modal>
  </div>
</template>
