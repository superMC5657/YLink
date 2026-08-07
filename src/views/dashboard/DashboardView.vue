<script setup lang="ts">
/**
 * 仪表板(截图1):Banner 统计 + 当前订阅 + 公告 + 快捷操作。
 * 数据:GET /user/stat、GET /user/subscribe、GET /notices(docs/api/README.md §5/§6)
 */
import { onMounted, onBeforeUnmount } from 'vue'
import { useUserStore } from '@/stores/user'
import BannerStatCard from '@/components/business/BannerStatCard.vue'
import SubscribeCard from '@/components/business/SubscribeCard.vue'
import NoticePanel from '@/components/business/NoticePanel.vue'
import QuickActionGrid from '@/components/business/QuickActionGrid.vue'
import { useMediaQuery } from '@vueuse/core'

const user = useUserStore()
const isDesktop = useMediaQuery('(min-width: 1024px)')

onMounted(() => {
  void user.refreshDashboard()
})

onBeforeUnmount(() => {
  // 离开页面无轮询句柄需要清理(窗口聚焦刷新由 MainLayout 处理)
})
</script>

<template>
  <div class="space-y-5">
    <BannerStatCard />

    <!-- 桌面双列:左 订阅卡 | 右 公告 -->
    <div v-if="isDesktop" class="grid grid-cols-5 gap-5">
      <div class="col-span-3"><SubscribeCard /></div>
      <div class="col-span-2"><NoticePanel /></div>
    </div>
    <!-- 移动端依次堆叠 -->
    <template v-else>
      <SubscribeCard />
      <NoticePanel />
    </template>

    <QuickActionGrid />
  </div>
</template>
