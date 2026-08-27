<script setup lang="ts">
/**
 * 顶栏:毛玻璃吸顶。桌面含折叠钮/站点名/主题/语言/用户;移动端含抽屉钮/站点名/主题/头像。
 */
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useMessage, useDialog } from 'naive-ui'
import type { DropdownOption } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import ThemeToggle from '@/components/ui/ThemeToggle.vue'
import LanguageToggle from '@/components/ui/LanguageToggle.vue'

const app = useAppStore()
const auth = useAuthStore()
const config = useConfigStore()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()
/** admin 模式(AdminLayout):徽标 + 下拉「返回用户中心」;用户模式下拉「进入管理后台」(仅管理员) */
const props = defineProps<{ admin?: boolean }>()
const emit = defineEmits<{ (e: 'toggle-drawer'): void }>()

const isMobile = computed(() => window.innerWidth < 768)
const userEmail = computed(() => auth.user?.email ?? '')

const dropdownOptions = computed<DropdownOption[]>(() => [
  // 对称互切入口:管理端 → 返回用户中心;用户端(管理员) → 进入管理后台
  ...(props.admin || auth.isAdmin
    ? [
        { label: t(props.admin ? 'nav.backToUser' : 'nav.enterAdmin'), key: 'switch' },
        { type: 'divider' as const, key: 'd0' },
      ]
    : []),
  { label: t('nav.profile'), key: 'profile' },
  { type: 'divider' as const, key: 'd1' },
  { label: t('common.logout'), key: 'logout' },
])

function onDropdownSelect(key: string) {
  if (key === 'logout') onLogout()
  else if (key === 'switch') router.push(props.admin ? '/dashboard' : '/admin/overview')
  else goProfile()
}

function goProfile() {
  router.push('/profile')
}

function onLogout() {
  dialog.warning({
    title: t('common.logout'),
    content: t('common.logoutConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await auth.logout()
      message.success(t('common.logout'))
      router.push('/login')
    },
  })
}
</script>

<template>
  <header
    class="sticky top-0 z-20 flex h-16 shrink-0 items-center gap-3 border-b border-[var(--c-border)] px-4 backdrop-blur-12 md:px-6"
    style="background: color-mix(in srgb, var(--c-bg-card) 82%, transparent)"
  >
    <!-- 移动端:抽屉钮 -->
    <button
      v-if="isMobile"
      class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-[var(--c-text)] transition-colors hover:bg-[var(--c-bg-hover)]"
      @click="emit('toggle-drawer')"
    >
      <AppIcon name="menu" :size="22" />
    </button>

    <!-- 桌面:折叠钮 -->
    <button
      v-else
      class="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-[var(--c-text-sub)] transition-colors hover:bg-[var(--c-bg-hover)]"
      @click="app.toggleSidebar"
    >
      <AppIcon :name="app.sidebarCollapsed ? 'chevron-right' : 'chevron-down'" :size="20" />
    </button>

    <!-- 站点名(移动端)/管理端徽标 -->
    <template v-if="admin">
      <span class="truncate text-16 font-700 text-[var(--c-text)]">{{ config.siteName }}</span>
      <span
        class="hidden shrink-0 rounded-full px-2.5 py-0.5 text-12 md:inline-flex"
        style="background: var(--c-primary-soft); color: var(--c-primary-text)"
      >
        {{ t('nav.groupAdmin') }}
      </span>
      <span class="flex-1" />
    </template>
    <span v-else-if="isMobile" class="text-16 font-700 text-[var(--c-text)]">{{
      config.siteName
    }}</span>
    <!-- 桌面:占位撑开布局(原页面标题已移除) -->
    <span v-else class="flex-1" />

    <div class="flex items-center gap-2">
      <!-- 语言(滑动切换,仿主题切换) -->
      <LanguageToggle />

      <!-- 主题 -->
      <ThemeToggle />

      <!-- 用户 -->
      <n-dropdown trigger="click" :options="dropdownOptions" @select="onDropdownSelect">
        <button
          class="flex cursor-pointer items-center gap-2 rounded-[var(--r-control)] px-2 py-1 transition-colors hover:bg-[var(--c-bg-hover)]"
        >
          <span
            class="flex h-8 w-8 items-center justify-center rounded-full text-white"
            style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          >
            <AppIcon name="user" :size="17" />
          </span>
          <span class="hidden max-w-40 truncate text-14 text-[var(--c-text)] lg:inline">{{
            userEmail
          }}</span>
        </button>
      </n-dropdown>
    </div>
  </header>
</template>
