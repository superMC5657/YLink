<script setup lang="ts">
/**
 * 移动端抽屉菜单:左滑出,内容与桌面侧边栏一致,底部附用户信息与退出。
 */
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { NAV_GROUPS } from '@/router/nav'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const auth = useAuthStore()
const config = useConfigStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

function isActive(path: string): boolean {
  if (path === '/dashboard') return route.path === '/dashboard'
  return route.path.startsWith(path)
}

function go(path: string) {
  emit('update:show', false)
  router.push(path)
}

function onLogout() {
  dialog.warning({
    title: '退出登录',
    content: '确定要退出当前账号吗?',
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await auth.logout()
      emit('update:show', false)
      message.success(t('common.logout'))
      router.push('/login')
    },
  })
}
</script>

<template>
  <n-drawer
    :show="props.show"
    :width="280"
    placement="left"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <div class="flex h-full flex-col">
      <!-- Logo -->
      <div class="flex h-16 shrink-0 items-center gap-3 border-b border-[var(--c-border)] px-5">
        <span
          class="flex h-9 w-9 items-center justify-center rounded-xl text-white"
          style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
        >
          <AppIcon name="zap" :size="20" />
        </span>
        <span class="text-16 font-700 text-[var(--c-text)]">{{ config.siteName }}</span>
      </div>

      <!-- 菜单 -->
      <nav class="flex-1 overflow-y-auto px-3 py-3">
        <div v-for="group in NAV_GROUPS" :key="group.label" class="mb-4">
          <div
            class="mb-1.5 px-3 text-11 uppercase tracking-wider text-[var(--c-text-sub)] opacity-70"
          >
            {{ group.label }}
          </div>
          <div class="space-y-1">
            <button
              v-for="item in group.items"
              :key="item.path"
              class="flex h-11 w-full cursor-pointer items-center gap-3 rounded-[var(--r-pill)] px-3 text-14 transition-colors"
              :class="
                isActive(item.path)
                  ? 'bg-[var(--c-primary-soft)] font-500 text-[var(--c-primary-text)]'
                  : 'text-[var(--c-text-sub)] hover:bg-[var(--c-bg-hover)]'
              "
              @click="go(item.path)"
            >
              <AppIcon :name="item.icon" :size="20" />
              <span>{{ item.name }}</span>
            </button>
          </div>
        </div>
      </nav>

      <!-- 底部用户 -->
      <div class="shrink-0 border-t border-[var(--c-border)] p-4">
        <div class="flex items-center gap-3">
          <span
            class="flex h-10 w-10 items-center justify-center rounded-full text-white"
            style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          >
            <AppIcon name="user" :size="18" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="truncate text-14 font-500 text-[var(--c-text)]">{{ auth.user?.email }}</div>
            <div class="text-12 text-[var(--c-text-sub)]">ID: {{ auth.user?.id ?? '-' }}</div>
          </div>
        </div>
        <button
          class="mt-4 flex h-10 w-full cursor-pointer items-center justify-center gap-2 rounded-[var(--r-pill)] text-14 text-[var(--c-danger)] transition-colors hover:bg-[var(--c-danger-bg)]"
          @click="onLogout"
        >
          <AppIcon name="log-out" :size="17" />
          <span>{{ t('common.logout') }}</span>
        </button>
      </div>
    </div>
  </n-drawer>
</template>
