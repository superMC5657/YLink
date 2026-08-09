<script setup lang="ts">
/**
 * 流量明细:时间范围选择(近 7 天/30 天/自定义)+ ECharts 柱状图 + 明细表。
 * 数据:GET /user/traffic-logs?from=&to=(契约 §5.6);页面:pages.md §3.12
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'
import { formatBytes } from '@/utils/format'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { ECharts } from 'echarts/core'
import dayjs from 'dayjs'

echarts.use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const user = useUserStore()
const app = useAppStore()
const { t } = useI18n()

type RangeKey = '7d' | '30d' | 'custom'
const rangeKey = ref<RangeKey>('30d')
const customRange = ref<[number, number] | null>(null)

const chartRef = ref<HTMLDivElement | null>(null)
let chart: ECharts | null = null

const dateRange = computed<{ from: string; to: string }>(() => {
  if (rangeKey.value === '7d') {
    return {
      from: dayjs().subtract(6, 'day').format('YYYY-MM-DD'),
      to: dayjs().format('YYYY-MM-DD'),
    }
  }
  if (rangeKey.value === '30d') {
    return {
      from: dayjs().subtract(29, 'day').format('YYYY-MM-DD'),
      to: dayjs().format('YYYY-MM-DD'),
    }
  }
  if (customRange.value) {
    return {
      from: dayjs(customRange.value[0]).format('YYYY-MM-DD'),
      to: dayjs(customRange.value[1]).format('YYYY-MM-DD'),
    }
  }
  return {
    from: dayjs().subtract(29, 'day').format('YYYY-MM-DD'),
    to: dayjs().format('YYYY-MM-DD'),
  }
})

const totalUpload = computed(() => user.trafficLogs.reduce((a, x) => a + x.u, 0))
const totalDownload = computed(() => user.trafficLogs.reduce((a, x) => a + x.d, 0))
const totalAll = computed(() => user.trafficLogs.reduce((a, x) => a + x.total, 0))

function renderChart() {
  if (!chartRef.value) return
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }
  const isDark = app.isDark
  const axisColor = isDark ? '#9BA1B7' : '#8A8FA3'
  const splitColor = isDark ? 'rgba(255,255,255,0.06)' : 'rgba(23,25,66,0.06)'
  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: isDark ? '#171B26' : '#fff',
      borderColor: isDark ? 'rgba(255,255,255,0.1)' : '#EBECF4',
      textStyle: { color: isDark ? '#E8EAF2' : '#1F2430', fontSize: 12 },
      formatter: (params: { seriesName: string; value: number; axisValue: string }[]) => {
        const lines = params.map((p) => `${p.seriesName}: ${formatBytes(p.value)}`).join('<br/>')
        return `${params[0].axisValue}<br/>${lines}`
      },
    },
    legend: {
      data: [t('traffic.upload'), t('traffic.download')],
      textStyle: { color: axisColor, fontSize: 12 },
      top: 0,
    },
    grid: { left: 8, right: 8, top: 36, bottom: 8, containLabel: true },
    xAxis: {
      type: 'category',
      data: user.trafficLogs.map((x) => x.date.slice(5)),
      axisLine: { lineStyle: { color: splitColor } },
      axisLabel: { color: axisColor, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: splitColor } },
      axisLabel: {
        color: axisColor,
        fontSize: 11,
        formatter: (v: number) => formatBytes(v),
      },
    },
    series: [
      {
        name: t('traffic.upload'),
        type: 'bar',
        stack: 'traffic',
        data: user.trafficLogs.map((x) => x.u),
        itemStyle: { color: '#6558F5', borderRadius: [3, 3, 0, 0] },
        barMaxWidth: 18,
      },
      {
        name: t('traffic.download'),
        type: 'bar',
        stack: 'traffic',
        data: user.trafficLogs.map((x) => x.d),
        itemStyle: { color: '#8B5CF6', borderRadius: [3, 3, 0, 0] },
        barMaxWidth: 18,
      },
    ],
  })
}

watch(
  () => [user.trafficLogs, app.isDark] as const,
  () => {
    renderChart()
  },
  { deep: true },
)

watch(
  dateRange,
  (range) => {
    void user.fetchTrafficLogs(range.from, range.to)
  },
  { immediate: true },
)

function onResize() {
  chart?.resize()
}

onMounted(() => {
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div>
    <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-20 font-600 text-[var(--c-text)]">{{ t('traffic.title') }}</h1>
      <div class="flex items-center gap-2">
        <div class="flex rounded-[var(--r-control)] border border-[var(--c-border)] p-0.5">
          <button
            v-for="r in [
              { key: '7d', label: t('traffic.last7d') },
              { key: '30d', label: t('traffic.last30d') },
              { key: 'custom', label: t('traffic.custom') },
            ]"
            :key="r.key"
            class="cursor-pointer rounded-[var(--r-control)] px-3 py-1 text-12 transition-colors"
            :class="
              rangeKey === r.key
                ? 'bg-[var(--c-primary-soft)] font-500 text-[var(--c-primary-text)]'
                : 'text-[var(--c-text-sub)] hover:text-[var(--c-text)]'
            "
            @click="rangeKey = r.key as RangeKey"
          >
            {{ r.label }}
          </button>
        </div>
        <n-date-picker
          v-if="rangeKey === 'custom'"
          v-model:value="customRange"
          type="daterange"
          size="small"
          :clearable="false"
        />
      </div>
    </div>

    <!-- 汇总卡 -->
    <div class="mb-5 grid grid-cols-3 gap-4">
      <div class="card-base flex flex-col items-center gap-1 p-4">
        <span class="text-12 text-[var(--c-text-sub)]">{{ t('traffic.upload') }}</span>
        <span class="num text-18 font-700 text-[var(--c-primary-text)]">{{
          formatBytes(totalUpload)
        }}</span>
      </div>
      <div class="card-base flex flex-col items-center gap-1 p-4">
        <span class="text-12 text-[var(--c-text-sub)]">{{ t('traffic.download') }}</span>
        <span class="num text-18 font-700 text-[var(--c-pink)]">{{
          formatBytes(totalDownload)
        }}</span>
      </div>
      <div class="card-base flex flex-col items-center gap-1 p-4">
        <span class="text-12 text-[var(--c-text-sub)]">{{ t('traffic.total') }}</span>
        <span class="num text-18 font-700 text-[var(--c-text)]">{{ formatBytes(totalAll) }}</span>
      </div>
    </div>

    <!-- 图表 -->
    <div class="card-base mb-5 p-4 md:p-6">
      <div ref="chartRef" class="h-70 w-full" />
    </div>

    <!-- 明细表 -->
    <div class="card-base p-4 md:p-6">
      <n-table :bordered="false" class="w-full">
        <thead>
          <tr>
            <th class="text-13">{{ t('traffic.date') }}</th>
            <th class="text-13">{{ t('traffic.upload') }}</th>
            <th class="text-13">{{ t('traffic.download') }}</th>
            <th class="text-13">{{ t('traffic.total') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="x in user.trafficLogs"
            :key="x.date"
            class="transition-colors hover:bg-[var(--c-bg-hover)]"
          >
            <td class="text-13 text-[var(--c-text)]">{{ x.date }}</td>
            <td class="num text-13 text-[var(--c-primary-text)]">{{ formatBytes(x.u) }}</td>
            <td class="num text-13 text-[var(--c-pink)]">{{ formatBytes(x.d) }}</td>
            <td class="num text-13 font-500 text-[var(--c-text)]">{{ formatBytes(x.total) }}</td>
          </tr>
        </tbody>
      </n-table>
      <EmptyState
        v-if="user.trafficLogs.length === 0"
        :text="t('traffic.empty')"
        :icon="'traffic'"
      />
    </div>
  </div>
</template>
