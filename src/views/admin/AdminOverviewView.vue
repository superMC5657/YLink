<script setup lang="ts">
/**
 * 管理后台 · 总览:核心运营指标统计。
 * 数据:GET /admin/stat/overview(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminOverviewResp } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'

const loading = ref(false)
const data = ref<AdminOverviewResp | null>(null)

// 快捷操作入口:与仪表盘 QuickActionGrid 同风格——每个入口不同彩色浅底 + 彩色图标
const quickEntries = [
  {
    to: '/admin/users',
    icon: 'users',
    label: '用户管理',
    color: 'var(--c-primary)',
    bg: 'var(--c-primary-soft)',
  },
  {
    to: '/admin/plans',
    icon: 'zap',
    label: '套餐管理',
    color: 'var(--c-warning)',
    bg: 'var(--c-warning-bg)',
  },
  {
    to: '/admin/nodes',
    icon: 'server',
    label: '节点管理',
    color: 'var(--c-blue)',
    bg: 'var(--c-blue-bg)',
  },
  {
    to: '/admin/orders',
    icon: 'order',
    label: '订单管理',
    color: 'var(--c-olive)',
    bg: 'var(--c-olive-bg)',
  },
]

async function load() {
  loading.value = true
  try {
    data.value = await apiAdmin.overview()
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="运营总览" subtitle="平台用户 / 订单 / 收入核心指标" />

    <n-spin :show="loading">
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-4">
        <StatNumber label="注册用户" :value="data?.user_count ?? 0" icon="users" />
        <StatNumber
          label="代理商"
          :value="data?.agent_count ?? 0"
          icon="agent"
          icon-color="var(--c-marketing)"
        />
        <StatNumber label="全部订单" :value="data?.order_count ?? 0" icon="order" />
        <StatNumber
          label="已完成订单"
          :value="data?.completed_orders ?? 0"
          icon="check"
          icon-color="var(--c-success)"
        />
        <StatNumber
          label="累计收入"
          :value="data?.total_revenue ?? 0"
          unit="元"
          icon="wallet"
          icon-color="var(--c-olive)"
        />
        <StatNumber
          label="今日收入"
          :value="data?.today_revenue ?? 0"
          unit="元"
          icon="coins"
          icon-color="var(--c-marketing)"
        />
        <StatNumber label="在售套餐" :value="data?.plan_count ?? 0" icon="zap" />
      </div>

      <!-- 快速入口 -->
      <div class="card-base mt-5 p-5">
        <h2 class="mb-4 text-16 font-600 text-[var(--c-text)]">快捷操作</h2>
        <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <RouterLink
            v-for="e in quickEntries"
            :key="e.to"
            :to="e.to"
            class="flex cursor-pointer items-center justify-center gap-2 rounded-xl py-4 text-14 no-underline transition-all duration-[var(--t-fast)] hover:-translate-y-0.5 hover:shadow-[var(--s-card)] active:scale-95"
            :style="{ backgroundColor: e.bg }"
          >
            <AppIcon :name="e.icon" :size="18" :style="{ color: e.color }" />
            <span class="font-500 text-[var(--c-text)]">{{ e.label }}</span>
          </RouterLink>
        </div>
      </div>
    </n-spin>
  </div>
</template>
