# [P0] 03 Markdown 单例+懒加载+XSS收敛

Status: ready-for-agent
Type: task

## Finding

- `NoticePanel.vue:22`（dashboard首屏）/`PlanCard.vue:25`（每卡 render）/`DocsDetailView.vue:21` 各 `new MarkdownIt()`，`markdown-it` ~130KB 常驻。
- sanitize 各自裸调，无集中策略；NoticePanel 先拼接 coupon-code 再 sanitize，linkify 的 `a` 无 rel guard。

## Task

1. 新建 `src/utils/markdown.ts: renderMarkdown()` 单例 + `await import('markdown-it')` 懒加载；配置 `html:false,linkify:true` + `DOMPurify.sanitize({ALLOWED_URI_REGEXP:/^(https?|mailto):/, ADD_ATTR:['target','rel'], FORBID_TAGS:['style','form','input']})` + 后处理 `a→rel=noopener target=_blank`。
2. 三处改调统一入口；PlanCard 用 `computed` 缓存避免列表 N 次解析。
3. coupon-code 改白名单校验 `/^[A-Z0-9-]{4,20}$/` 后拼接；补 XSS 单测（PlanCard.spec 同款覆盖三处）。

## Done

- dashboard/plans 首屏减小；`pnpm test` XSS 用例全绿；coupon 复制正常。

## Comments
