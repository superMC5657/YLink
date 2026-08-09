<script setup lang="ts">
/**
 * 套餐卡片:名称 → 价格区 → 流量/带宽 → 描述(Markdown)→ 立即购买。
 * 数据:GET /plans(契约 §8);页面:docs/frontend/pages.md §3.8
 */
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import { formatSpeed } from '@/utils/format'
import type { Plan, PlanPeriod } from '@/types/api'

const props = defineProps<{
  plan: Plan
  /** 当前选中周期(默认取第一个) */
  period?: PlanPeriod
}>()

const emit = defineEmits<{
  (e: 'buy', plan: Plan): void
  (e: 'period-change', plan: Plan, period: PlanPeriod): void
}>()

const { t } = useI18n()
const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

const periods = computed(() => Object.keys(props.plan.prices) as PlanPeriod[])
const currentPeriod = computed<PlanPeriod>(() => props.period ?? periods.value[0])

const price = computed(() => props.plan.prices[currentPeriod.value] ?? 0)

/** 计算选中周期相对月付的折扣文案(季/年) */
const savePercent = computed(() => {
  const month = props.plan.prices.month
  if (!month || currentPeriod.value === 'month') return null
  const periodMonths: Record<string, number> = { quarter: 3, half_year: 6, year: 12, onetime: 12 }
  const months = periodMonths[currentPeriod.value]
  if (!months) return null
  const raw = 1 - price.value / (month * months)
  const pct = Math.round(raw * 100)
  return pct > 0 ? pct : null
})

function renderContent(): string {
  return DOMPurify.sanitize(md.render(props.plan.content))
}

function periodLabel(p: PlanPeriod): string {
  const map: Record<PlanPeriod, string> = {
    month: '月付',
    quarter: '季付',
    half_year: '半年付',
    year: '年付',
    onetime: '一次性',
  }
  return map[p]
}
</script>

<template>
  <div class="card-base card-hoverable flex flex-col p-6">
    <!-- 名称 -->
    <h3 class="text-center text-18 font-600 text-[var(--c-text)]">{{ plan.name }}</h3>
    <p class="mt-0.5 text-center text-14 text-[var(--c-text-sub)]">
      {{ t('plan.traffic') }} {{ plan.traffic_gb }}G
    </p>

    <!-- 价格区 -->
    <div class="mt-4 flex items-baseline justify-center gap-1.5">
      <PriceText :value="price" :size="34" />
      <span class="text-14 text-[var(--c-text-sub)]">/{{ periodLabel(currentPeriod) }}</span>
      <StatusBadge v-if="savePercent" type="marketing" :dot="false">
        {{ t('plan.savePercent', { n: savePercent }) }}
      </StatusBadge>
    </div>

    <!-- 周期切换 -->
    <div class="mt-4 flex justify-center gap-1.5">
      <button
        v-for="p in periods"
        :key="p"
        class="cursor-pointer rounded-[var(--r-control)] px-3 py-1 text-14 transition-colors"
        :class="
          currentPeriod === p
            ? 'bg-[var(--c-primary-soft)] font-600 text-[var(--c-primary-text)]'
            : 'bg-[var(--c-bg-hover)] text-[var(--c-text-sub)] hover:text-[var(--c-text)]'
        "
        @click="emit('period-change', plan, p)"
      >
        {{ periodLabel(p) }}
      </button>
    </div>

    <!-- 流量/带宽 -->
    <div
      class="mt-5 flex justify-center gap-6 rounded-xl py-3"
      style="background-color: var(--c-bg-hover)"
    >
      <div class="text-center">
        <div class="num text-16 font-700 text-[var(--c-text)]">{{ plan.traffic_gb }}G</div>
        <div class="text-14 text-[var(--c-text-sub)]">{{ t('plan.traffic') }}</div>
      </div>
      <div class="w-px bg-[var(--c-border)]" />
      <div class="text-center">
        <div class="num text-16 font-700 text-[var(--c-text)]">
          {{ formatSpeed(plan.speed_limit) }}
        </div>
        <div class="text-14 text-[var(--c-text-sub)]">{{ t('plan.bandwidth') }}</div>
      </div>
      <div class="w-px bg-[var(--c-border)]" />
      <div class="text-center">
        <div class="num text-16 font-700 text-[var(--c-text)]">{{ plan.device_limit }}</div>
        <div class="text-14 text-[var(--c-text-sub)]">{{ t('plan.devices') }}</div>
      </div>
    </div>

    <!-- 描述 -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div
      class="plan-content mt-4 flex-1 text-14 leading-6 text-[var(--c-text-sub)]"
      v-html="renderContent()"
    />

    <!-- 购买 -->
    <button
      class="mt-5 h-11 w-full cursor-pointer rounded-[var(--r-control)] border border-[var(--c-primary)] text-14 font-500 text-[var(--c-primary-text)] transition-all duration-[var(--t-base)] hover:bg-[var(--c-primary)] hover:text-white active:scale-98"
      @click="emit('buy', plan)"
    >
      {{ t('plan.buyNow') }}
    </button>
  </div>
</template>

<style scoped>
.plan-content :deep(p) {
  margin: 4px 0;
}
.plan-content :deep(strong) {
  color: var(--c-marketing);
  font-weight: 600;
}
</style>
