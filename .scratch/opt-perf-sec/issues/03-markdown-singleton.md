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

- [x] `src/utils/markdown.ts: renderMarkdown()` 单例:markdown-it+DOMPurify 双动态 import(模块级缓存),统一 `html:false/linkify/breaks` + 代码高亮。
- [x] XSS 集中策略:URI 白名单 `^(https?|mailto):`(i)、FORBID style/form/input、净化后 `a→target=_blank rel=noopener noreferrer`(后处理不可伪造)。
- [x] 三处改调统一入口:NoticePanel(展开时异步渲染+过期结果丢弃)、PlanCard(watch+immediate 按内容缓存,列表不重复解析)、DocsDetailView(watch article.body)。
- [x] coupon-code 白名单 `/^[A-Z0-9-]{4,20}$/` 校验后才拼 chip(白名单不含引号/尖括号,属性注入不可能),chip 在净化后追加,i18n title 转义引号。
- [x] XSS 单测 5 例(内联 HTML 转义/危险协议/链接加固/FORBID 标签/误杀回归)+ PlanCard 集成断言更新。
- [x] 门禁:`pnpm lint/typecheck` + vitest 72 全绿;build 产物确认首屏入口对 markdown chunk 零引用。

## Comments

- 实测 markdown-it `html:false` 对内联 HTML 是**转义**(输出 `&lt;script&gt;` 文本)而非删除,且默认拦截 `javascript:` 链接(退化为纯文本);单测按真实行为断言「不产生标签/不产出 href」,另加误杀回归保证 linkify 正常。
