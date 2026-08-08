<script setup lang="ts">
/**
 * 仪表板 Banner 卡:背景插画 + 圆形 Logo + 钱包余额(绿)/我的佣金(粉)。
 * 数据:GET /user/stat(docs/api/README.md §5.1);页面:docs/frontend/pages.md §3.3
 */
import { useUserStore } from '@/stores/user'
import { formatMoney } from '@/utils/format'
import { useI18n } from 'vue-i18n'

const user = useUserStore()
const { t } = useI18n()
</script>

<template>
  <div
    class="relative overflow-hidden rounded-[var(--r-card)] p-6 text-white md:p-8"
    style="
      background: linear-gradient(120deg, #5b4be0 0%, #7a5cf0 45%, #9a6cf8 100%);
      box-shadow: var(--s-card-hover);
    "
  >
    <!-- 氛围圆环 -->
    <div
      class="pointer-events-none absolute -right-16 -top-16 h-64 w-64 rounded-full border-2 border-white/10"
    />
    <div
      class="pointer-events-none absolute -right-8 -top-8 h-40 w-40 rounded-full border-2 border-white/15"
    />
    <div
      class="pointer-events-none absolute bottom-0 right-24 h-24 w-24 rounded-full bg-white/5 blur-2xl"
    />

    <div class="relative flex flex-wrap items-center gap-5">
      <!-- Logo -->
      <span
        class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-white/15 backdrop-blur"
      >
        <AppIcon name="zap" :size="28" />
      </span>

      <div class="min-w-0 flex-1">
        <div class="text-13 text-white/70">{{ t('dashboard.title') }}</div>
        <div class="truncate text-20 font-700">{{ user.stat?.email }}</div>
      </div>

      <!-- 余额 / 佣金 -->
      <div class="flex gap-6 md:gap-10">
        <div>
          <div class="flex items-center gap-1.5 text-13 text-white/70">
            <AppIcon name="wallet" :size="15" />
            {{ t('dashboard.balance') }}
          </div>
          <div class="num mt-1 text-28 font-700 text-[#7CFC9C] md:text-32">
            {{ formatMoney(user.balance) }}
          </div>
        </div>
        <div>
          <div class="flex items-center gap-1.5 text-13 text-white/70">
            <AppIcon name="coins" :size="15" />
            {{ t('dashboard.commission') }}
          </div>
          <div class="num mt-1 text-28 font-700 text-[#FFB3D9] md:text-32">
            {{ formatMoney(user.commissionBalance) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
