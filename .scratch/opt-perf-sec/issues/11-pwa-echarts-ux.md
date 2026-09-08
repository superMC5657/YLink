# [P2] 11 PWA与图表体验

Status: ready-for-agent
Type: task

## Finding

- `vite.config.ts:52 navigateFallback:index.html` 无 denylist/runtimeCaching；PWA 图标复用 512 兼 maskable，缺 144/384；`globPatterns` 缺 webmanifest/json/webp/woff。
- `AdminReportsView` 5 图同 mount 全量 render 无骨架/采样；全站无 webp。

## Task

1. 加 `navigateFallbackDenylist:[/^\/api/]` + `runtimeCaching GET /api → networkOnly`；补 maskable 安全边距图 + 144/384；glob 补 webmanifest/webp/woff。
2. AdminReports 折叠懒 mount + dataZoom + 200ms resize 防抖 + showLoading；离线写操作给“稍后重试”。

## Done

- 弱网不掩盖 502；大流量不卡主线程；PWA 可安装。

## Comments
