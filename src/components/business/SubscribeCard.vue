<script setup lang="ts">
/**
 * 当前订阅卡:名称、过期标签、用量进度条、五宫格数据。
 * 数据:GET /user/subscribe(docs/api/README.md §5.4);页面:docs/frontend/pages.md §3.3
 */
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
import { formatBytes, formatExpiry, formatPercent, formatSpeed, formatTime } from '@/utils/format'

const user = useUserStore()
const { t } = useI18n()

const sub = computed(() => user.subscribe)

const progress = computed(() => sub.value?.used_percent ?? 0)

const progressColor = computed(() => {
  if (progress.value >= 95) return 'var(--c-danger)'
  if (progress.value >= 80) return 'var(--c-warning)'
  return 'var(--c-primary)'
})

const expireBadge = computed(() => {
  const s = sub.value
  if (!s?.has_subscription) return { type: 'neutral' as const, text: t('dashboard.noSubscription') }
  if (s.is_expired) return { type: 'danger' as const, text: formatExpiry(s.expired_at, true) }
  const days = s.expired_days
  if (days <= 7) return { type: 'warning' as const, text: formatExpiry(s.expired_at, false) }
  return { type: 'success' as const, text: formatExpiry(s.expired_at, false) }
})
</script>

<template>
  <div class="card-base card-hoverable p-5 md:p-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <span
          class="flex h-10 w-10 items-center justify-center rounded-xl text-white"
          style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
        >
          <AppIcon name="zap" :size="20" />
        </span>
        <div>
          <div class="text-14 text-[var(--c-text-sub)]">
            {{ t('dashboard.currentSubscription') }}
          </div>
          <div class="text-16 font-600 text-[var(--c-text)]">
            {{ sub?.has_subscription ? sub?.plan?.name : t('dashboard.noSubscription') }}
          </div>
          <div class="text-14 text-[var(--c-text-sub)]">
            {{ t('dashboard.expireAt') }}:
            {{ sub?.expired_at ? formatTime(sub.expired_at, false) : '-' }}
          </div>
        </div>
      </div>
      <StatusBadge :type="expireBadge.type">{{ expireBadge.text }}</StatusBadge>
    </div>

    <!-- 用量进度 -->
    <div class="mt-5">
      <div class="mb-1.5 flex items-center justify-between text-14">
        <span class="text-[var(--c-text-sub)]">
          {{ t('dashboard.transferUsed') }}
          <span class="num font-600 text-[var(--c-text)]">{{
            formatBytes(sub?.u && sub?.d ? sub.u + sub.d : 0)
          }}</span>
          /
          <span class="num font-600 text-[var(--c-text)]">{{
            formatBytes(sub?.transfer_enable)
          }}</span>
        </span>
        <span class="num font-600" :style="{ color: progressColor }">
          {{ t('dashboard.usedPercent', { percent: formatPercent(progress) }) }}
        </span>
      </div>
      <div
        class="h-2.5 w-full overflow-hidden rounded-full"
        style="background-color: var(--c-bg-hover)"
      >
        <div
          class="h-full rounded-full transition-all duration-500"
          :style="{
            width: `${Math.min(100, progress)}%`,
            background: `linear-gradient(90deg, ${progressColor}, ${progressColor}cc)`,
          }"
        />
      </div>
    </div>

    <!-- 五宫格 -->
    <div class="mt-5 grid grid-cols-5 gap-2">
      <div
        class="flex flex-col items-center gap-0.5 rounded-xl py-2.5"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="text-14 text-[var(--c-text-sub)]">{{ t('dashboard.transferRemaining') }}</span>
        <span class="num text-14 font-600 text-[var(--c-text)]">{{
          formatBytes(sub?.remaining)
        }}</span>
      </div>
      <div
        class="flex flex-col items-center gap-0.5 rounded-xl py-2.5"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="text-14 text-[var(--c-text-sub)]">{{ t('dashboard.speedLimit') }}</span>
        <span class="num text-14 font-600 text-[var(--c-text)]">{{
          formatSpeed(sub?.speed_limit)
        }}</span>
      </div>
      <div
        class="flex flex-col items-center gap-0.5 rounded-xl py-2.5"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="text-14 text-[var(--c-text-sub)]">{{ t('dashboard.deviceLimit') }}</span>
        <span class="num text-14 font-600 text-[var(--c-text)]">{{
          sub?.device_limit ?? '-'
        }}</span>
      </div>
      <div
        class="flex flex-col items-center gap-0.5 rounded-xl py-2.5"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="text-14 text-[var(--c-text-sub)]">{{ t('dashboard.planName') }}</span>
        <span class="text-14 font-600 text-[var(--c-text)]">{{ sub?.plan?.name ?? '-' }}</span>
      </div>
      <div
        class="flex flex-col items-center gap-0.5 rounded-xl py-2.5"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="text-14 text-[var(--c-text-sub)]">{{ t('common.expired') }}</span>
        <span
          class="text-14 font-600"
          :style="{ color: sub?.is_expired ? 'var(--c-danger)' : 'var(--c-success)' }"
        >
          {{ sub?.is_expired ? '是' : '否' }}
        </span>
      </div>
    </div>
  </div>
</template>
