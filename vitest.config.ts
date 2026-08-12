import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    // 统一东八区:formatTime 等按本地时区格式化,测试断言以北京时间为准,
    // 避免 CI(Ubuntu UTC)与本机(Windows UTC+8)时区不一致导致失败
    env: {
      TZ: 'Asia/Shanghai',
    },
    include: ['src/**/*.spec.ts', 'src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/utils/**', 'src/stores/**', 'src/composables/**'],
    },
  },
})
