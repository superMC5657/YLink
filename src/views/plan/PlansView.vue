<script setup lang="ts">
/**
 * 购买订阅(截图6):套餐卡片网格 + 下单确认弹窗。
 * 数据:GET /plans(docs/api/README.md §8);页面:docs/frontend/pages.md §3.8
 */
import { computed, onMounted, ref } from 'vue'
import { usePlanStore } from '@/stores/plan'
import { useConfigStore } from '@/stores/config'
import { useMediaQuery } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import PlanCard from '@/components/business/PlanCard.vue'
import OrderConfirmModal from '@/components/business/OrderConfirmModal.vue'
import type { Plan, PlanPeriod } from '@/types/api'

const planStore = usePlanStore()
const config = useConfigStore()
const isDesktop = useMediaQuery('(min-width: 1024px)')
const isTablet = useMediaQuery('(min-width: 768px)')
const { t } = useI18n()

const showConfirm = ref(false)
const selectedPlan = ref<Plan | null>(null)

/** 每张卡默认周期(取第一个)与用户手动切换的周期 */
const selectedPeriods = ref<Record<number, PlanPeriod>>({})

function defaultPeriod(plan: Plan): PlanPeriod {
  return (Object.keys(plan.prices) as PlanPeriod[])[0]
}

const gridClass = computed(() => {
  if (isDesktop.value) return 'grid-cols-3'
  if (isTablet.value) return 'grid-cols-2'
  return 'grid-cols-1'
})

function onBuy(plan: Plan) {
  selectedPlan.value = plan
  showConfirm.value = true
}

function onPeriodChange(plan: Plan, period: PlanPeriod) {
  selectedPeriods.value[plan.id] = period
}

function periodOf(plan: Plan): PlanPeriod {
  return selectedPeriods.value[plan.id] ?? defaultPeriod(plan)
}

onMounted(() => {
  void planStore.fetch()
  void config.fetchConfig()
})
</script>

<template>
  <div>
    <div class="mb-5">
      <h1 class="text-20 font-600 text-[var(--c-text)]">{{ t('plan.title') }}</h1>
      <p class="mt-1 text-13 text-[var(--c-text-sub)]">{{ config.config?.site_description }}</p>
    </div>

    <n-spin :show="planStore.loading">
      <div class="grid gap-5" :class="gridClass">
        <PlanCard
          v-for="plan in planStore.list"
          :key="plan.id"
          :plan="plan"
          :period="periodOf(plan)"
          @buy="onBuy"
          @period-change="onPeriodChange"
        />
      </div>
      <EmptyState
        v-if="!planStore.loading && planStore.list.length === 0"
        :text="t('common.empty')"
        :icon="'zap'"
      />
    </n-spin>

    <OrderConfirmModal v-model:show="showConfirm" :plan="selectedPlan" />
  </div>
</template>
