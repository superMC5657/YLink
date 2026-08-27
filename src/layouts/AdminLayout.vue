<script setup lang="ts">
/**
 * 管理端布局:桌面(≥1024px)AdminSidebar+顶栏;手机(<768)顶栏汉堡+独立抽屉菜单。
 * 与用户端 MainLayout 的差异:不渲染 MobileTabBar、不渲染客服浮球、
 * 不接 usePullToRefresh 等用户端设施。规范见 docs/frontend/pages.md §2.4。
 */
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMediaQuery } from '@vueuse/core'
import AdminSidebar from '@/components/app/AdminSidebar.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import { useAppStore } from '@/stores/app'
import { useConfigStore } from '@/stores/config'

const isDesktop = useMediaQuery('(min-width: 1024px)')
const isMobile = useMediaQuery('(max-width: 767px)')
const app = useAppStore()
const config = useConfigStore()
const route = useRoute()

const drawerVisible = ref(false)

onMounted(() => {
  void config.fetchConfig()
  app.initSystemThemeListener()
  // 与 MainLayout 一致:1024-1279px 窄桌面默认折叠侧边栏
  if (window.innerWidth >= 1024 && window.innerWidth < 1280) {
    app.sidebarCollapsed = true
  }
})

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
    <!-- 桌面/平板:管理侧边栏 -->
    <AdminSidebar v-if="!isMobile" class="hidden md:flex" />

    <div class="flex min-w-0 flex-1 flex-col">
      <AppHeader admin @toggle-drawer="drawerVisible = true" />

      <main
        class="relative flex-1 overflow-y-auto overflow-x-hidden"
        :class="isMobile ? 'px-4 pb-8 pt-4' : 'px-6 py-6 md:px-8'"
      >
        <div class="mx-auto w-full" :class="isDesktop ? 'max-w-[1440px]' : 'max-w-none'">
          <router-view v-slot="{ Component }">
            <transition name="fade-slide" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </main>
    </div>

    <!-- 移动端:独立管理抽屉(内容与桌面侧边栏一致,仅管理菜单) -->
    <n-drawer
      :show="drawerVisible"
      :width="280"
      placement="left"
      @update:show="(v: boolean) => (drawerVisible = v)"
    >
      <div class="h-full"><AdminSidebar fill /></div>
    </n-drawer>
  </div>
</template>
