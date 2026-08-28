<script setup lang="ts">
/**
 * 根组件:Naive Provider(亮/暗主题与 overrides 联动)+ ToastBridge。
 * F19 品牌配置:站点 primary_color / background_url 下发后联动
 * CSS 设计令牌(tokens.css 变量)与 Naive 主题主色。
 */
import { computed, watch } from 'vue'
import { darkTheme } from 'naive-ui'
import type { GlobalTheme } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { useConfigStore } from '@/stores/config'
import { darkThemeOverrides, lightThemeOverrides } from '@/styles/theme'
import { applyBackgroundImage, applyPrimaryColor, isHexColor, mixHex } from '@/utils/brand'
import ToastBridge from '@/components/app/ToastBridge.vue'
import UpdateCard from '@/components/app/UpdateCard.vue'

const app = useAppStore()
const config = useConfigStore()

const theme = computed<GlobalTheme | null>(() => (app.isDark ? darkTheme : null))

// F19 品牌主色:空/非法色值回退默认主题
const brandColor = computed(() => {
  const c = config.config?.primary_color?.trim()
  return isHexColor(c) ? c : ''
})

const themeOverrides = computed(() => {
  const base = app.isDark ? darkThemeOverrides : lightThemeOverrides
  const c = brandColor.value
  if (!c) return base
  return {
    ...base,
    common: {
      ...base.common,
      primaryColor: c,
      primaryColorHover: mixHex(c, '#ffffff', 0.14),
      primaryColorPressed: mixHex(c, '#000000', 0.2),
      primaryColorSuppl: c,
      infoColor: c,
    },
  }
})

watch(
  [brandColor, () => config.config?.background_url, () => app.isDark],
  ([color, bg]) => {
    applyPrimaryColor(color)
    applyBackgroundImage(bg)
  },
  { immediate: true },
)
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <ToastBridge>
          <router-view />
        </ToastBridge>
        <!-- 桌面端更新卡片(仅 Tauri;Web 端内部自动降级不渲染) -->
        <UpdateCard />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
