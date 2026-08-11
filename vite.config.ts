import type { ConfigEnv, UserConfig } from 'vite'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { viteMockServe } from 'vite-plugin-mock'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = loadEnv(mode, process.cwd(), '')
  const useMock = env.VITE_USE_MOCK !== 'false'

  return {
    base: './',
    plugins: [
      vue(),
      UnoCSS(),
      viteMockServe({
        mockPath: 'mock',
        enable: useMock,
        logger: true,
      }),
      // PWA：manifest + Workbox 离线壳（手机端响应式 Web 可安装/离线；Tauri 不受影响）
      VitePWA({
        registerType: 'autoUpdate',
        // 外部脚本注册（injectRegister script）而非 inline，兼容 Tauri CSP script-src 'self'
        injectRegister: 'script',
        includeAssets: ['pwa-128.png', 'pwa-192.png', 'pwa-512.png'],
        manifest: {
          name: 'YLink 高速稳定代理订阅',
          short_name: 'YLink',
          description: '高速稳定的网络加速服务',
          lang: 'zh-CN',
          theme_color: '#6558F5',
          background_color: '#ffffff',
          display: 'standalone',
          start_url: './',
          scope: './',
          icons: [
            { src: 'pwa-128.png', sizes: '128x128', type: 'image/png' },
            { src: 'pwa-192.png', sizes: '192x192', type: 'image/png' },
            { src: 'pwa-512.png', sizes: '512x512', type: 'image/png' },
            { src: 'pwa-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
          ],
        },
        workbox: {
          // 离线壳：index.html + 构建产物(带 hash)预缓存；运行时请求网络优先 + 缓存兜底
          globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],
          navigateFallback: 'index.html',
        },
        devOptions: { enabled: false },
      }),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: 5174,
      host: true,
    },
    build: {
      target: 'es2020',
      chunkSizeWarningLimit: 1500,
    },
  }
})
