# [P0] 06 Layouts 懒加载

Status: ready-for-agent
Type: task

## Finding

- `src/router/index.ts:3-5 MainLayout/AdminLayout/AuthLayout` 同步进首包；Admin 外壳让普通用户买单；33 view 已全懒加载（肯定），缺 chunk 命名/预加载。

## Task

1. layouts 改 `()=>import()`，admin 独立 chunk 加 `/* webpackChunkName:"admin-layout" */`；重页面 reports/users 加 prefetch 空闲加载。
2. 验证普通登录首屏不再含 AdminLayout。

## Done

- [x] MainLayout/AdminLayout/AuthLayout 改 `()=>import()`,三壳各自独立 chunk(AdminLayout 1.9KB 仅管理员拉取);Vite/Rollup 按模块名自动产出 `AdminLayout-<hash>.js`,webpackChunkName 注释不适用(见 Comments)。
- [x] `prefetchAdminHeavyViews(isAdmin)`:`router.isReady()` 后 `requestIdleCallback`(无则 setTimeout 兜底)预取 reports/users chunk;普通用户跳过。
- [x] 普通登录首屏不再含 AdminLayout(entry 中仅剩 chunk 映射表字符串,代码本体在独立 chunk)。
- [x] E2E 双 project 56/56 通过;报表页用例改为滚动内部滚动容器(main.overflow-y-auto,window 不滚动)触发懒初始化后断言 5 canvas。

## Comments

- E2E 报表用例的两处适配:①ticket 04 视口懒初始化后,视口外图表不再 mount,断言前需逐屏滚动;②双布局均为 `h-screen + 内部 overflow-y-auto`,`window.scrollTo` 无效,需滚 `main` 容器。两处均为适配新交互,测试意图(5 图渲染+零报错)不变。
