<script setup lang="ts">
/**
 * 主布局:桌面(≥1024px)侧边栏+顶栏;平板(768-1024)迷你侧边栏;
 * 手机(<768)顶栏+底部标签栏+抽屉菜单。
 * 规范见 docs/frontend/pages.md §2。
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import MobileTabBar from '@/components/app/MobileTabBar.vue'
import DrawerMenu from '@/components/app/DrawerMenu.vue'
import CustomerServiceFab from '@/components/app/CustomerServiceFab.vue'
import { useMediaQuery } from '@vueuse/core'
import { useAppStore } from '@/stores/app'
import { useConfigStore } from '@/stores/config'
import { useUserStore } from '@/stores/user'

const isDesktop = useMediaQuery('(min-width: 1024px)')
const isMobile = useMediaQuery('(max-width: 767px)')
const app = useAppStore()
const config = useConfigStore()
const user = useUserStore()
const route = useRoute()

const drawerVisible = ref(false)

onMounted(() => {
  // 进入主布局:拉取站点配置与用户数据
  void config.fetchConfig()
  void user.refreshDashboard()
  app.initSystemThemeListener()
  // 1024-1279px 窄桌面:默认折叠侧边栏,给内容区留更多横向空间
  if (window.innerWidth >= 1024 && window.innerWidth < 1280) {
    app.sidebarCollapsed = true
  }
  // 窗口聚焦静默刷新
  window.addEventListener('focus', onFocus)
  window.addEventListener('visibilitychange', onVisibility)
})

onBeforeUnmount(() => {
  window.removeEventListener('focus', onFocus)
  window.removeEventListener('visibilitychange', onVisibility)
})

function onFocus() {
  if (document.visibilityState === 'visible') void user.refreshDashboard()
}
function onVisibility() {
  if (document.visibilityState === 'visible') void user.refreshDashboard()
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
        class="flex-1 overflow-y-auto overflow-x-hidden"
        :class="isMobile ? 'px-4 pb-24 pt-4' : 'px-6 py-6 md:px-8'"
      >
        <div
          class="mx-auto w-full"
          :class="isDesktop ? 'max-w-[1440px]' : 'max-w-none'"
        >
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
