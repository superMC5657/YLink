<script setup lang="ts">
/**
 * 管理端侧边栏:唯一菜单源 ADMIN_NAV_GROUPS(13 项),底部「返回用户中心」。
 * fill 模式用于移动端独立抽屉(撑满容器、固定展开、隐藏折叠钮)。
 * 规范见 docs/frontend/pages.md §2.4。
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useConfigStore } from '@/stores/config'
import { ADMIN_NAV_GROUPS } from '@/router/nav'

const props = defineProps<{ fill?: boolean }>()

const app = useAppStore()
const config = useConfigStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

/** 抽屉(fill)内固定展开;桌面受全局折叠状态控制 */
const collapsed = computed(() => !props.fill && app.sidebarCollapsed)
const width = computed(() => (props.fill ? '100%' : collapsed.value ? '72px' : '200px'))

function isActive(path: string): boolean {
  return route.path.startsWith(path)
}

function go(path: string) {
  router.push(path)
}

function goUserCenter() {
  router.push('/dashboard')
}
</script>

<template>
  <aside
    class="flex h-full shrink-0 flex-col border-r border-r-solid border-[var(--c-card-border)] bg-[var(--c-bg-card)] transition-[width] duration-300"
    :style="{ width }"
  >
    <!-- Logo 区 -->
    <div
      class="flex h-16 shrink-0 items-center gap-3 px-5"
      :class="collapsed ? 'justify-center px-0' : ''"
    >
      <span
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-white"
        style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
      >
        <AppIcon name="zap" :size="20" />
      </span>
      <span v-if="!collapsed" class="truncate text-16 font-700 text-[var(--c-text)]">
        {{ config.siteName }}
      </span>
    </div>

    <!-- 管理菜单(仅 ADMIN_NAV_GROUPS,不含任何用户端菜单) -->
    <nav class="flex-1 overflow-y-auto px-3 py-2">
      <div v-for="group in ADMIN_NAV_GROUPS" :key="group.label" class="mb-4">
        <div
          v-if="!collapsed"
          class="mb-1.5 px-3 text-14 uppercase tracking-wider text-[var(--c-text-sub)] opacity-70"
        >
          {{ t(group.label) }}
        </div>
        <div class="space-y-1">
          <button
            v-for="item in group.items"
            :key="item.path"
            class="group flex h-10 w-full cursor-pointer items-center gap-3 rounded-[var(--r-control)] px-3 text-14 transition-all duration-[var(--t-fast)]"
            :class="
              isActive(item.path)
                ? 'bg-[var(--c-primary-soft)] text-[var(--c-primary-text)] font-500'
                : 'text-[var(--c-text-sub)] hover:bg-[var(--c-bg-hover)] hover:text-[var(--c-text)]'
            "
            :title="collapsed ? t(item.name) : undefined"
            @click="go(item.path)"
          >
            <AppIcon :name="item.icon" :size="19" class="shrink-0" />
            <span v-if="!collapsed" class="truncate">{{ t(item.name) }}</span>
          </button>
        </div>
      </div>
    </nav>

    <!-- 底部:返回用户中心(对称互切,A3) + 折叠钮 -->
    <div class="shrink-0 border-t border-[var(--c-border)] p-3">
      <button
        class="flex h-10 w-full cursor-pointer items-center justify-center gap-2 rounded-[var(--r-control)] text-14 text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
        :title="collapsed ? t('nav.backToUser') : undefined"
        @click="goUserCenter"
      >
        <AppIcon name="arrow-left" :size="18" />
        <span v-if="!collapsed" class="truncate">{{ t('nav.backToUser') }}</span>
      </button>
      <button
        v-if="!fill"
        class="mt-1 flex h-10 w-full cursor-pointer items-center justify-center gap-2 rounded-[var(--r-control)] text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
        @click="app.toggleSidebar"
      >
        <AppIcon :name="collapsed ? 'chevron-right' : 'chevron-down'" :size="18" />
        <span v-if="!collapsed" class="text-14">{{ t('nav.collapse') }}</span>
      </button>
    </div>
  </aside>
</template>
