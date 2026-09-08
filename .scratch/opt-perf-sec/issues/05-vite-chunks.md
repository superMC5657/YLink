# [P0] 05 Vite 分包+压缩

Status: ready-for-agent
Type: task

## Finding

- `vite.config.ts:66-71` 现状 `target:es2020, chunkSizeWarningLimit:1500`，无 manualChunks/minify/sourcemap/compression/visualizer 声明。

## Task

1. 加 `manualChunks: vendor-vue/vendor-naive/vendor-echarts/vendor-md/admin-*`，首屏目标 <300KB gzip；`chunkSizeWarningLimit` 改回 500-800。
2. 加 `vite-plugin-compression`（gzip+brotli）+ `rollup-plugin-visualizer` 取数（先取数再定阈值）。
3. 显式 `minify:'esbuild', sourcemap:false, cssCodeSplit:true`；`target:es2020` 保留（WebView2 兼容）注明原因；`viteMockServe logger:true` 开发外关噪音。

## Done

- `pnpm build:web` 产物分包清晰；Caddy/Nginx 预压缩或插件压缩其一生效。

## Comments
