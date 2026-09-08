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

- 无重复 `use`；resize/dispose 正常；`pnpm lint/typecheck` 通过。

## Comments
