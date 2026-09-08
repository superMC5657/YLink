# [P0] 04 ECharts 共享+动态

Status: ready-for-agent
Type: task

## Finding

- `AdminReportsView.vue:12-19` 与 `TrafficView.vue:11-18` 重复 `echarts/core use([Bar,Line,Grid,Tooltip,Legend,Canvas])` 同步引入，无共享封装。

## Task

1. 新建 `src/composables/useEcharts.ts`（init/dispose/resize/cssVar 取色统一），内部 `import('echarts/core')` 动态。
2. 两处改调；Admin 5 图视口懒初始化（折叠/IntersectionObserver）。
3. Traffic tooltip formatter 复用 `format.ts:formatBytes`，数字外 HTML 转义。

## Done

- [x] `src/composables/useEcharts.ts`:echarts core/charts/components/renderers 动态 import(模块级缓存全应用单次注册 Bar/Line/Grid/Tooltip/Legend/Canvas),实例按 el 缓存防重复 init,dispose(卸载)/resizeAll/cssVar 统一收口。
- [x] TrafficView/AdminReportsView 改调 useEcharts;Admin 5 图 IntersectionObserver 视口懒初始化(不支持时降级立即渲染)。
- [x] Traffic tooltip formatter 补 escapeHtml(seriesName/axisValue 转义),数值仍走 formatBytes。
- [x] 门禁:`pnpm lint/typecheck` + vitest 72 全绿;build 产物:echarts 仅存在于 renderers/useEcharts 异步 chunk,首屏入口零引用。

## Comments
