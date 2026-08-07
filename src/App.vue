<script setup lang="ts">
/**
 * 根组件:Naive Provider(亮/暗主题与 overrides 联动)+ ToastBridge。
 */
import { computed } from 'vue'
import { darkTheme } from 'naive-ui'
import type { GlobalTheme } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { darkThemeOverrides, lightThemeOverrides } from '@/styles/theme'
import ToastBridge from '@/components/app/ToastBridge.vue'

const app = useAppStore()

const theme = computed<GlobalTheme | null>(() => (app.isDark ? darkTheme : null))
const themeOverrides = computed(() => (app.isDark ? darkThemeOverrides : lightThemeOverrides))
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <ToastBridge>
          <router-view />
        </ToastBridge>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
