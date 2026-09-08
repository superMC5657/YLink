# [P0] 06 Layouts 懒加载

Status: ready-for-agent
Type: task

## Finding

- `src/router/index.ts:3-5 MainLayout/AdminLayout/AuthLayout` 同步进首包；Admin 外壳让普通用户买单；33 view 已全懒加载（肯定），缺 chunk 命名/预加载。

## Task

1. layouts 改 `()=>import()`，admin 独立 chunk 加 `/* webpackChunkName:"admin-layout" */`；重页面 reports/users 加 prefetch 空闲加载。
2. 验证普通登录首屏不再含 AdminLayout。

## Done

- 首包减小；路由跳转正常；E2E 双 project 通过。

## Comments
