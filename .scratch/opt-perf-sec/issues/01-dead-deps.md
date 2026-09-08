# [P0] 01 Dead deps 清理

Status: ready-for-agent
Type: task

## Finding

- `package.json:47 @iconify/vue` src 零引用（已被 `src/components/ui/AppIcon.vue:15-89` 自研离线 SVG 替代）。
- `uno.config.ts:13-19 presetIcons({collections:{}})` 空配置残留。
- `package.json:49 @vueuse/core` src 未见 `from '@vueuse`（需 `rg` 二次确认）。

## Task

1. `rg -n "iconify|vueuse" src mock tests scripts` 双确认；确无引用则 `pnpm remove @iconify/vue`（@vueuse 同理，若有引用则本单只清 iconify 并在 Comments 注明）。
2. `uno.config.ts` 去 `presetIcons` 或注释为何保留。
3. 跑 `pnpm install --frozen-lockfile` 验证 lockfile 更新。

## Done

- [x] `@iconify/vue` 已移除（src 零引用）；`uno.config.ts` presetIcons 空配置已删。
- [x] `@vueuse/core` 有 6 处引用（useMediaQuery：AdminLayout/MainLayout/Dashboard/Invite/Orders/Plans），按约定保留。
- [x] `pnpm install --frozen-lockfile` + `pnpm lint` + `pnpm build:web` 通过；git status 仅 package.json/pnpm-lock/uno.config。

## Comments

- @vueuse/core 保留：src 存在真实引用，未达死依赖标准。
