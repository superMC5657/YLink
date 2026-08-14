<script setup lang="ts">
/**
 * 主布局:桌面(≥1024px)侧边栏+顶栏;平板(768-1024)迷你侧边栏;
 * 手机(<768)顶栏+底部标签栏+抽屉菜单。
 * 规范见 docs/frontend/pages.md §2。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import MobileTabBar from '@/components/app/MobileTabBar.vue'
import DrawerMenu from '@/components/app/DrawerMenu.vue'
import CustomerServiceFab from '@/components/app/CustomerServiceFab.vue'
import { useMediaQuery } from '@vueuse/core'
import { useAppStore } from '@/stores/app'
import { useConfigStore } from '@/stores/config'
import { useUserStore } from '@/stores/user'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { useLocalNotifications } from '@/composables/useLocalNotifications'

const isDesktop = useMediaQuery('(min-width: 1024px)')
const isMobile = useMediaQuery('(max-width: 767px)')
const app = useAppStore()
const config = useConfigStore()
const user = useUserStore()
const route = useRoute()
const { t } = useI18n()
const { checkExpire, checkTickets } = useLocalNotifications()

const drawerVisible = ref(false)
const mainEl = ref<HTMLElement | null>(null)

// 本地通知轮询(desktop-tauri.md §4):工单已回复每 60s 检查一次
let notifyTimer: ReturnType<typeof setInterval> | null = null

// 移动端下拉刷新（pages.md §6.3）：与窗口聚焦刷新一致，静默刷新仪表板数据
const { pulling, refreshing, distance, threshold } = usePullToRefresh(mainEl, () =>
  user.refreshDashboard(),
)

onMounted(() => {
  // 进入主布局:拉取站点配置与用户数据
  void config.fetchConfig()
  void user.refreshDashboard().then(() => void checkExpire())
  void checkTickets()
  app.initSystemThemeListener()
  // 1024-1279px 窄桌面:默认折叠侧边栏,给内容区留更多横向空间
  if (window.innerWidth >= 1024 && window.innerWidth < 1280) {
    app.sidebarCollapsed = true
  }
  // 窗口聚焦静默刷新
  window.addEventListener('focus', onFocus)
  window.addEventListener('visibilitychange', onVisibility)
  // 工单已回复本地通知:60s 轮询(桌面端;Web 端内部自动降级)
  notifyTimer = setInterval(() => void checkTickets(), 60_000)
})

onBeforeUnmount(() => {
  window.removeEventListener('focus', onFocus)
  window.removeEventListener('visibilitychange', onVisibility)
  if (notifyTimer) {
    clearInterval(notifyTimer)
    notifyTimer = null
  }
})

function onFocus() {
  if (document.visibilityState === 'visible') {
    void user.refreshDashboard().then(() => void checkExpire())
    // 工单已回复通知:聚焦时立即检查,不必等 60s 轮询
    void checkTickets()
  }
}
function onVisibility() {
  if (document.visibilityState === 'visible') {
    void user.refreshDashboard().then(() => void checkExpire())
    void checkTickets()
  }
}

// 路由切换关闭抽屉
watch(
  () => route.path,
  () => {
    drawerVisible.value = false
  },
)
</script>

<template>
  <div class="flex h-screen w-full overflow-hidden">
    <!-- 桌面/平板:侧边栏 -->
    <AppSidebar v-if="!isMobile" class="hidden md:flex" />

    <div class="flex min-w-0 flex-1 flex-col">
      <AppHeader @toggle-drawer="drawerVisible = true" />

      <main
        ref="mainEl"
        class="relative flex-1 overflow-y-auto overflow-x-hidden"
        :class="isMobile ? 'px-4 pb-24 pt-4' : 'px-6 py-6 md:px-8'"
      >
        <!-- 移动端下拉刷新指示器（仅触屏下拉时出现） -->
        <transition name="fade">
          <div
            v-if="pulling || refreshing"
            class="pointer-events-none fixed inset-x-0 top-2 z-50 flex justify-center"
          >
            <div
              class="flex items-center gap-1.5 rounded-full bg-[var(--c-bg-elevated)] px-3 py-1 text-13 text-[var(--c-text-sub)] shadow-lg transition-transform"
              :style="{ transform: `translateY(${distance}px)` }"
            >
              <span
                class="h-3 w-3 rounded-full border-2 border-[var(--c-primary)] border-t-transparent"
                :class="{ 'animate-spin': refreshing }"
              />
              {{
                refreshing
                  ? t('common.refreshing')
                  : distance >= threshold
                    ? t('common.releaseToRefresh')
                    : t('common.pullToRefresh')
              }}
            </div>
          </div>
        </transition>

        <div class="mx-auto w-full" :class="isDesktop ? 'max-w-[1440px]' : 'max-w-none'">
          <router-view v-slot="{ Component }">
            <transition name="fade-slide" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </main>
    </div>

    <!-- 移动端:底部标签栏 + 抽屉 -->
    <MobileTabBar v-if="isMobile" />
    <DrawerMenu v-model:show="drawerVisible" />

    <!-- 客服浮球 -->
    <CustomerServiceFab />
  </div>
</template>
