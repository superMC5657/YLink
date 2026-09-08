# [P2] 11 PWA与图表体验

Status: done
Type: task

## Finding

- `vite.config.ts:52 navigateFallback:index.html` 无 denylist/runtimeCaching;PWA 图标复用 512 兼容 maskable,缺 144/384;`globPatterns` 缺 webmanifest/json/webp/woff。
- `AdminReportsView` 5 图同 mount 全量 render 无骨架/采样;全站无 webp。

## Task

1. 加 `navigateFallbackDenylist:[/^\/api/]` + `runtimeCaching GET /api → networkOnly`;补 maskable 安全边距图 + 144/384;glob 补 webmanifest/webp/woff。
2. AdminReports 折叠懒 mount + dataZoom + 200ms resize 防抖 + showLoading;离线写操作给"稍后重试"。

## Done

- 弱网不掩盖 502;大流量不卡主线程;PWA 可安装。

## Comments

- PWA:denylist + NetworkOnly(GET /api)已验证进产物 sw.js;glob 补 woff/webp/webmanifest/json;manifest 补 144/384 图标 + maskable 专用图(内容缩至 78% 安全区、theme_color 铺底,PIL 脚本一次性生成,源 pwa-512.png 不动)。
- AdminReports:「折叠懒 mount」由 04 单交付的视口懒初始化(IntersectionObserver)承载,不再引入 n-collapse(报表页折叠反而伤可用性);本单加 dataZoom(营收时序图 inside+slider)+ showLoading 占位(数据未到/chunk 首载时不再白卡)+ resize 200ms 防抖(在 useEcharts.resizeAll 收口,TrafficView 同步受益,DataZoomComponent 按需注册)。
- 离线写操作:既有网络错误 toast 文案即「请稍后再试」+ ToastBridge 离线横幅已覆盖引导,未重复造轮子;配合 NetworkOnly,离线时明确失败不返回旧数据。
- 门禁:lint/typecheck/79 单测/报表 e2e/build:web(PWA 89 entries 预缓存)全绿。
