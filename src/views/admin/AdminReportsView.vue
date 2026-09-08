<script setup lang="ts">
/**
 * 管理后台 · 统计报表(F04):订单/营收趋势、注册趋势、套餐分布、用户/节点流量 TopN。
 * 数据:GET /admin/stat/orders|users|traffic?days=(docs/api/README.md §16)
 * 图表走 composables/useEcharts(echarts 动态加载);5 图按视口懒初始化(IntersectionObserver)。
 */
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { apiAdmin } from '@/api/admin'
import type { AdminStatOrdersResp, AdminStatTrafficResp, AdminStatUsersResp } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useEcharts } from '@/composables/useEcharts'
import { formatBytes } from '@/utils/format'

const app = useAppStore()
const { t } = useI18n()
const { getChart, cssVar, whenVisible, resizeAll } = useEcharts()

const loading = ref(false)
const days = ref<7 | 30 | 90>(30)
const orders = ref<AdminStatOrdersResp | null>(null)
const users = ref<AdminStatUsersResp | null>(null)
const traffic = ref<AdminStatTrafficResp | null>(null)

// 五张图挂载点;实例由 useEcharts 按 el 缓存,卸载时统一 dispose
const revChartRef = ref<HTMLDivElement | null>(null)
const regChartRef = ref<HTMLDivElement | null>(null)
const planChartRef = ref<HTMLDivElement | null>(null)
const userTopChartRef = ref<HTMLDivElement | null>(null)
const nodeTopChartRef = ref<HTMLDivElement | null>(null)

// 视口懒初始化:图表容器进入视口才允许 init/render,避免同屏 5 图全量渲染卡顿
const chartVisible = reactive({
  rev: false,
  reg: false,
  plan: false,
  userTop: false,
  nodeTop: false,
})

function baseTooltip(): Record<string, unknown> {
  const isDark = app.isDark
  return {
    trigger: 'axis',
    backgroundColor: cssVar('--c-bg-card', isDark ? '#171B26' : '#fff'),
    borderColor: cssVar('--c-border', isDark ? 'rgba(255,255,255,0.1)' : '#EBECF4'),
    textStyle: {
      color: cssVar('--c-text', isDark ? '#E8EAF2' : '#1F2430'),
      fontSize: 12,
    },
  }
}

function axisStyles() {
  const isDark = app.isDark
  return {
    axisColor: cssVar('--c-text-sub', isDark ? '#9BA1B7' : '#8A8FA3'),
    splitColor: cssVar('--c-border', isDark ? 'rgba(255,255,255,0.06)' : 'rgba(23,25,66,0.06)'),
  }
}

function renderAll() {
  void renderRevenue()
  void renderRegister()
  void renderPlan()
  void renderUserTop()
  void renderNodeTop()
}

async function renderRevenue() {
  if (!revChartRef.value || !orders.value || !chartVisible.rev) return
  const chart = await getChart(revChartRef.value)
  if (!chart) return
  const { axisColor, splitColor } = axisStyles()
  chart.setOption({
    backgroundColor: 'transparent',
    ...baseTooltip(),
    legend: {
      data: [
        t('adminReports.revenue'),
        t('adminReports.refunded'),
        t('adminReports.balanceUsed'),
        t('adminReports.balanceRefunded'),
      ],
      textStyle: { color: axisColor, fontSize: 12 },
      top: 0,
    },
    grid: { left: 8, right: 8, top: 36, bottom: 8, containLabel: true },
    xAxis: {
      type: 'category',
      data: orders.value.items.map((x) => x.date.slice(5)),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    series: [
      {
        name: t('adminReports.revenue'),
        type: 'line',
        smooth: true,
        data: orders.value.items.map((x) => x.revenue),
        itemStyle: { color: cssVar('--c-olive', '#7C9A5C') },
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.08 },
      },
      {
        name: t('adminReports.refunded'),
        type: 'line',
        smooth: true,
        data: orders.value.items.map((x) => x.refunded),
        itemStyle: { color: cssVar('--c-danger', '#E5484D') },
        lineStyle: { width: 2 },
      },
      {
        name: t('adminReports.balanceUsed'),
        type: 'line',
        smooth: true,
        data: orders.value.items.map((x) => x.balance_used),
        itemStyle: { color: cssVar('--c-primary', '#6558F5') },
        lineStyle: { width: 2, type: 'dashed' },
      },
      {
        name: t('adminReports.balanceRefunded'),
        type: 'line',
        smooth: true,
        data: orders.value.items.map((x) => x.balance_refunded),
        itemStyle: { color: cssVar('--c-pink', '#C2487B') },
        lineStyle: { width: 2, type: 'dashed' },
      },
    ],
  })
}

async function renderRegister() {
  if (!regChartRef.value || !users.value || !chartVisible.reg) return
  const chart = await getChart(regChartRef.value)
  if (!chart) return
  const { axisColor, splitColor } = axisStyles()
  chart.setOption({
    backgroundColor: 'transparent',
    ...baseTooltip(),
    grid: { left: 8, right: 8, top: 16, bottom: 8, containLabel: true },
    xAxis: {
      type: 'category',
      data: users.value.register_trend.map((x) => x.date.slice(5)),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    series: [
      {
        name: t('adminReports.registers'),
        type: 'bar',
        data: users.value.register_trend.map((x) => x.count),
        itemStyle: { color: cssVar('--c-primary', '#6558F5'), borderRadius: [3, 3, 0, 0] },
        barMaxWidth: 18,
      },
    ],
  })
}

async function renderPlan() {
  if (!planChartRef.value || !users.value || !chartVisible.plan) return
  const chart = await getChart(planChartRef.value)
  if (!chart) return
  const { axisColor, splitColor } = axisStyles()
  const dist = [...users.value.plan_distribution].reverse() // 横向条形图自下而上
  chart.setOption({
    backgroundColor: 'transparent',
    ...baseTooltip(),
    grid: { left: 8, right: 24, top: 16, bottom: 8, containLabel: true },
    xAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    yAxis: {
      type: 'category',
      data: dist.map((x) => x.plan_name || t('adminReports.planDeleted')),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    series: [
      {
        name: t('adminReports.activeUsers'),
        type: 'bar',
        data: dist.map((x) => x.users),
        itemStyle: { color: cssVar('--c-warning', '#E6A23C'), borderRadius: [0, 3, 3, 0] },
        barMaxWidth: 18,
      },
    ],
  })
}

async function renderUserTop() {
  if (!userTopChartRef.value || !traffic.value || !chartVisible.userTop) return
  const chart = await getChart(userTopChartRef.value)
  if (!chart) return
  const { axisColor, splitColor } = axisStyles()
  const top = [...traffic.value.user_top].reverse()
  chart.setOption({
    backgroundColor: 'transparent',
    ...baseTooltip(),
    grid: { left: 8, right: 24, top: 16, bottom: 8, containLabel: true },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11, formatter: (v: number) => formatBytes(v) },
    },
    yAxis: {
      type: 'category',
      data: top.map((x) => x.email),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    series: [
      {
        name: t('adminReports.usedTraffic'),
        type: 'bar',
        data: top.map((x) => x.total_bytes),
        itemStyle: { color: cssVar('--c-blue', '#3B82F6'), borderRadius: [0, 3, 3, 0] },
        barMaxWidth: 18,
      },
    ],
  })
}

async function renderNodeTop() {
  if (!nodeTopChartRef.value || !traffic.value || !chartVisible.nodeTop) return
  const chart = await getChart(nodeTopChartRef.value)
  if (!chart) return
  const { axisColor, splitColor } = axisStyles()
  const top = [...traffic.value.node_top].reverse()
  chart.setOption({
    backgroundColor: 'transparent',
    ...baseTooltip(),
    grid: { left: 8, right: 24, top: 16, bottom: 8, containLabel: true },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11, formatter: (v: number) => formatBytes(v) },
    },
    yAxis: {
      type: 'category',
      data: top.map((x) => x.name),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    series: [
      {
        name: t('adminReports.reportedTraffic'),
        type: 'bar',
        data: top.map((x) => x.bytes),
        itemStyle: { color: cssVar('--c-pink', '#8B5CF6'), borderRadius: [0, 3, 3, 0] },
        barMaxWidth: 18,
      },
    ],
  })
}

async function load() {
  loading.value = true
  try {
    const query = { days: days.value }
    const [o, u, tr] = await Promise.all([
      apiAdmin.statOrders(query),
      apiAdmin.statUsers(query),
      apiAdmin.statTraffic(query),
    ])
    orders.value = o
    users.value = u
    traffic.value = tr
  } finally {
    loading.value = false
  }
  renderAll()
}

watch(days, () => void load())
watch(
  () => app.isDark,
  () => renderAll(),
)

function onResize() {
  resizeAll()
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  // 视口内才初始化对应图表(进入视口即渲染,即使数据未到也会在 load 完成后补渲染)
  const observe = (
    elRef: typeof revChartRef,
    key: keyof typeof chartVisible,
    render: () => void,
  ) => {
    const el = elRef.value
    if (!el) return
    whenVisible(el, () => {
      chartVisible[key] = true
      render()
    })
  }
  observe(revChartRef, 'rev', renderRevenue)
  observe(regChartRef, 'reg', renderRegister)
  observe(planChartRef, 'plan', renderPlan)
  observe(userTopChartRef, 'userTop', renderUserTop)
  observe(nodeTopChartRef, 'nodeTop', renderNodeTop)
  void load()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
})
</script>

<template>
  <div>
    <PageHeader :title="t('adminReports.title')" :subtitle="t('adminReports.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <div class="flex rounded-[var(--r-control)] border border-[var(--c-border)] p-0.5">
            <button
              v-for="d in [7, 30, 90]"
              :key="d"
              class="h-7 rounded-[var(--r-control)] px-3 text-14 transition-colors"
              :class="
                days === d
                  ? 'bg-[var(--c-primary)] text-white'
                  : 'text-[var(--c-text-sub)] hover:text-[var(--c-text)]'
              "
              @click="days = d as 7 | 30 | 90"
            >
              {{ t('adminReports.lastNDays', { n: d }) }}
            </button>
          </div>
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <n-spin :show="loading">
      <div class="card-base p-5">
        <h2 class="mb-3 text-16 font-600 text-[var(--c-text)]">
          {{ t('adminReports.revenueTitle') }}
        </h2>
        <p class="mb-2 text-14 text-[var(--c-text-sub)]">{{ t('adminReports.revenueHint') }}</p>
        <div ref="revChartRef" class="h-72 w-full" />
      </div>

      <div class="mt-5 grid grid-cols-1 gap-5 xl:grid-cols-2">
        <div class="card-base p-5">
          <h2 class="mb-3 text-16 font-600 text-[var(--c-text)]">
            {{ t('adminReports.registerTitle') }}
          </h2>
          <div ref="regChartRef" class="h-64 w-full" />
        </div>
        <div class="card-base p-5">
          <h2 class="mb-3 text-16 font-600 text-[var(--c-text)]">
            {{ t('adminReports.planTitle') }}
          </h2>
          <div ref="planChartRef" class="h-64 w-full" />
        </div>
        <div class="card-base p-5">
          <h2 class="mb-1 text-16 font-600 text-[var(--c-text)]">
            {{ t('adminReports.userTopTitle') }}
          </h2>
          <p class="mb-2 text-14 text-[var(--c-text-sub)]">{{ t('adminReports.userTopHint') }}</p>
          <div ref="userTopChartRef" class="h-64 w-full" />
        </div>
        <div class="card-base p-5">
          <h2 class="mb-1 text-16 font-600 text-[var(--c-text)]">
            {{ t('adminReports.nodeTopTitle') }}
          </h2>
          <p class="mb-2 text-14 text-[var(--c-text-sub)]">{{ t('adminReports.nodeTopHint') }}</p>
          <div ref="nodeTopChartRef" class="h-64 w-full" />
        </div>
      </div>
    </n-spin>
  </div>
</template>
