# ADR-0006 · 用户端 api 模块 seam 方向：契约名出口 + store 持有传输核

> 状态：已接受（2026-10-25，review-fixes-3 P2-16）
> 相关：`src/api/*.ts`、`src/stores/*.ts`、c47bfb0（ica-E 薄透传收敛）、ADR-0003（管理端同型决策）、docs/frontend/data-layer.md §1.2/§2

## 背景

README §3 与 data-layer.md §1.2 原文规定「views → stores → api/（纯函数集合）→ http」，
api 模块持有 transport 实现。c47bfb0（ica-E）后实际代码反转：用户端
`api/{agent,invite,knowledge,notice,order,plan,server,ticket,user}.ts` 九个模块全部收薄为
「从 `@/stores/*` re-export 传输核」的适配缝（如 `api/user.ts` 出口
`userTransport`/`captchaTransport`/`configTransport`），URL 与响应形态唯一归属 store 内
传输核；`api/auth.ts` 保留旧方向（自持实现，`stores/auth.ts` 反向引用）。代码与文档
两种分层方向并存且无 ADR 记录（review-round-09 P2-16）。

## 决定

1. **代码为准，保留 seam 现状**（用户拍板 2026-10-25）：api 模块 = 契约名出口——既有
   import 路径与导出名稳定，transport 实现唯一归属 `src/stores/<domain>.ts` 传输核。
   收益：契约名不随实现搬迁漂移；改 URL/响应形态只动传输核一处，seam 与 store 动作
   同源。api 模块不新增逻辑，新调用方一律学 store 接口。
2. **用户端视图/composable 一律经 store**：业务视图不 import api 模块；一次性动作
   （改密、重置订阅、会话管理、Telegram 绑定）在 store 提供 actions，结果由页面局部
   持有；后台状态轮询（本地通知）走 store 的快照动作，不写列表态。
3. **直调豁免两处**：管理后台视图直调 `apiAdmin`（纯适配缝，实现归 `stores/admin/*`
   各域传输核，ADR-0003）——管理端数据一次性拉取、无跨页共享状态；认证视图（注册/
   找回的验证码、忘记密码）直调 `apiCaptcha`/`apiAuth`——会话建立前的引导页，无
   业务 store 状态可挂靠。
4. **auth 域保留旧方向（唯一例外）**：`api/auth.ts` 自持实现，`stores/auth.ts` 与
   ForgotView 双消费；方法仅 5 个、无传输核化收益，接受与 seam 模式并存，不强行对称。
5. **文档随改**：data-layer.md §1.2/§2 与 frontend README §3 按本 ADR 改写——以
   「store 是唯一 transport 持有者」替代「store 是唯一允许调用 api 模块的地方」，
   删除不存在的 `useSubscribeStore` 幽灵条目（订阅信息实际在 `useUserStore`）。

## 后果

- 新用户端端点：类型进 `types/api.d.ts` → 传输核方法（store）→ 需要保留既有引入
  路径时在 api seam 挂一行 re-export；视图消费走 store。
- 分层方向单一化：transport 实现只在 stores（auth 例外）；「页面不直接发请求」约束
  不变，落点从 api 模块移到 store 传输核。
- seam 文件允许零直接消费方（如 apiAgent/apiPlan 等已由 store 全量接管的域）；是否
  清理归死代码批次（review-fixes-3 ticket 21）评估，不在本 ADR 范围。
