import type { ConfigEnv, UserConfig } from 'vite'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { viteMockServe } from 'vite-plugin-mock'
import { VitePWA } from 'vite-plugin-pwa'
import { compression } from 'vite-plugin-compression2'
import { visualizer } from 'rollup-plugin-visualizer'
import { fileURLToPath, URL } from 'node:url'

/**
 * manualChunks 分包家族(仅收敛已知依赖,其余走 Rollup 默认拆分):
 * - vendor-vue:vue 生态 + i18n + dayjs + tauri api(首屏同步)
 * - vendor-naive:naive-ui 及其传递依赖(首屏同步,随用到的组件 tree-shake)
 * - vendor-echarts / vendor-md:仅被动态 import(useEcharts/renderMarkdown),
 *   归入命名 chunk 后仍是异步加载,不进首屏
 * 不设 catch-all 兜底,避免把异步依赖(qrcode 等)拖进同步 chunk。
 */
function manualChunks(id: string): string | undefined {
  if (!id.includes('node_modules')) return undefined
  if (/[\\/]node_modules[\\/]zrender[\\/]/.test(id)) return 'vendor-zrender'
  if (/[\\/]node_modules[\\/]echarts[\\/]/.test(id)) return 'vendor-echarts'
  if (
    /[\\/]node_modules[\\/](markdown-it|linkify-it|mdurl|entities|uc\.micro|punycode|dompurify)[\\/]/.test(
      id,
    )
  )
    return 'vendor-md'
  if (
    /[\\/]node_modules[\\/](@css-render|css-render|seemly|vueuc|vooks|evtd|treemate|vdirs|date-fns|naive-ui|@floating-ui)[\\/]/.test(
      id,
    )
  )
    return 'vendor-naive'
  if (
    /[\\/]node_modules[\\/](vue|@vue|vue-router|pinia|pinia-plugin-persistedstate|vue-demi|vue-i18n|@intlify|dayjs|@tauri-apps)[\\/]/.test(
      id,
    )
  )
    return 'vendor-vue'
  return undefined
}

export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = loadEnv(mode, process.cwd(), '')
  const useMock = env.VITE_USE_MOCK !== 'false'

  return {
    root: 'src',
    base: './',
    publicDir: '../public',
    envDir: '..',
    plugins: [
      vue(),
      UnoCSS(),
      viteMockServe({
        mockPath: 'mock',
        enable: useMock,
        // 生成骨架(mock/*.generated.ts)非路由模块,排除以免被当 mock 文件加载
        ignore: /\.generated\.ts$/,
        // mock 请求日志仅在开发模式打印,build/preview 不出噪音
        logger: mode === 'development',
      }),
      // 构建产物预压缩:单实例同时产出 .gz + .br(compression2 默认双算法),
      // 配合 Nginx/Caddy static precompression 直发预压缩文件
      compression({ include: /\.(js|css|html|svg|json)$/, algorithms: ['gzip', 'brotliCompress'] }),
      // 产物体积分析:pnpm build --mode analyze → 生成 stats-report.html(取数后再定 warning 阈值)
      ...(mode === 'analyze'
        ? [
            visualizer({
              filename: fileURLToPath(new URL('./stats-report.html', import.meta.url)),
              gzipSize: true,
              brotliSize: true,
            }),
          ]
        : []),
      // PWA：manifest + Workbox 离线壳（手机端响应式 Web 可安装/离线；Tauri 不受影响）
      VitePWA({
        registerType: 'autoUpdate',
        // 外部脚本注册（injectRegister script）而非 inline，兼容 Tauri CSP script-src 'self'
        injectRegister: 'script',
        includeAssets: [
          'pwa-128.png',
          'pwa-144.png',
          'pwa-192.png',
          'pwa-384.png',
          'pwa-512.png',
          'pwa-maskable-512.png',
        ],
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
            { src: 'pwa-144.png', sizes: '144x144', type: 'image/png' },
            { src: 'pwa-192.png', sizes: '192x192', type: 'image/png' },
            { src: 'pwa-384.png', sizes: '384x384', type: 'image/png' },
            { src: 'pwa-512.png', sizes: '512x512', type: 'image/png' },
            {
              src: 'pwa-maskable-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable',
            },
          ],
        },
        workbox: {
          // 离线壳：index.html + 构建产物(带 hash)预缓存；运行时请求网络优先 + 缓存兜底
          globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2,woff,webp,webmanifest,json}'],
          navigateFallback: 'index.html',
          // API 导航不落 SPA 兜底壳(否则 /api/* 异常时返回 index.html 掩盖真实 502/错误)
          navigateFallbackDenylist: [/^\/api\//],
          // API 响应一律只走网络:面板数据必须实时,离线时明确失败而非返回旧数据
          runtimeCaching: [
            {
              urlPattern: ({ url }: { url: URL }) => url.pathname.startsWith('/api/'),
              handler: 'NetworkOnly',
              method: 'GET',
            },
          ],
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
      outDir: '../dist',
      emptyOutDir: true,
      // es2020 保留:覆盖 Tauri WebView2 与现代浏览器(可选链/空值合并/逻辑赋值),不再下探
      target: 'es2020',
      minify: 'esbuild',
      sourcemap: false,
      cssCodeSplit: true,
      // 阈值按 pnpm analyze(stats-report.html)取数后定:最大合法异步 chunk 为
      // vendor-echarts ~813KB(echarts core+按需注册,仅图表页加载),首屏最大 vendor-naive ~708KB
      chunkSizeWarningLimit: 900,
      rollupOptions: {
        output: {
          // Vite 6.4 起 build.manualChunks 不再生效,需挂在 rollupOptions.output
          manualChunks,
        },
      },
    },
  }
})
