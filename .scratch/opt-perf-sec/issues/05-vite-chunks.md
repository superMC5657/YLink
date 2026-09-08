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

- [x] manualChunks(via `build.rollupOptions.output`):vendor-vue(192KB)/vendor-naive(708KB)/vendor-zrender(206KB)/vendor-echarts(813KB)/vendor-md(134KB);首屏 = entry 70KB + vendor-vue + vendor-naive ≈ **280KB gzip < 300KB 目标**;echarts/md 仍为异步 chunk 不进首屏;不设 catch-all,防异步依赖(qrcode 等)被拖进同步 chunk。
- [x] `vite-plugin-compression2` 单实例双算法(gzip+brotli),dist 产出 59×.gz / 60×.br;`rollup-plugin-visualizer` 挂 `--mode analyze`(`pnpm analyze` → stats-report.html,已 gitignore)。
- [x] 显式 `minify:'esbuild'/sourcemap:false/cssCodeSplit:true`;`target:es2020` 保留(WebView2)注明;`chunkSizeWarningLimit` 1500→900(按 stats 取数:最大合法异步 chunk vendor-echarts 813KB,首屏最大 vendor-naive 708KB);viteMockServe `logger` 仅 development。
- [x] 门禁:lint/typecheck + vitest 72 全绿;build 无 chunk-size 告警、无重复 emit;dev server 启动并响应正常。

## Comments

- 踩坑:Vite 6.4 起 `build.manualChunks` 不再生效(仅剩 splitVendorChunk 兼容路径),必须挂 `build.rollupOptions.output.manualChunks`——此前配置静默无效,已修正。
- compression2 单实例默认即同时产出 .gz+.br(`algorithms:['gzip','brotliCompress']`),配两个实例会双发 overwrite 警告。
- vendor-echarts 813KB 为 echarts core+按需注册的真实体积(此前散成多个 chunk 总量相同 ~1MB),已拆出 zrender 独立 chunk;进一步压缩需改用 echarts/lib 细粒度导入,超出本单范围。
- `admin-*` 未做强制合并 chunk:33 个 view 已按路由天然拆分,强制合并会把 18 个管理页合成单一大 chunk,与降 warning 阈值目标相悖。
