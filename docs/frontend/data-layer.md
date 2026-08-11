# 前端开发文档 · 数据层

> 覆盖 HTTP 封装、状态管理、持久化、格式化、i18n 与「一键导入」深链接。接口定义以 [../api/README.md](../api/README.md) 契约为准。

## 1. HTTP 客户端封装（`utils/http.ts`）

### 1.1 职责

1. 底层通道选择：Tauri 环境用 `@tauri-apps/plugin-http`（原生网络栈，无 CORS/CSP 限制）；浏览器用 `window.fetch`（依赖后端 CORS）。
2. 请求注入：`Authorization: Bearer <access_token>`、`Accept-Language`（当前语言）、`X-Client`（`web / tauri-windows / tauri-macos / tauri-linux`）。
3. 响应处理：按 HTTP 状态 + 业务码（envelope `{code, message, data}`）统一解包为 `data`，错误归一化为 `ApiError { code, message, status }`。
4. Token 生命周期：
   - 收到 401 且存在 refresh_token → 调 `POST /auth/refresh` 静默换新；
   - 刷新期间并发请求进入等待队列，只刷新一次（single-flight）；
   - 刷新失败 → 清空会话、跳转 `/login` 并带 `redirect`。
5. 副作用策略：默认错误 toast 由封装层弹出（可在单次请求 `silent: true` 关闭）；GET 请求支持 `AbortController` 随组件卸载取消。

### 1.2 使用约定

```ts
// api/order.ts —— api 模块是纯函数集合，返回解包后的数据
export const fetchOrders = (q: OrderQuery) => http.get<PageResult<Order>>('/orders', { query: q })
export const createOrder = (body: CreateOrderReq) => http.post<CreateOrderResp>('/orders', { body })
```

- api 模块只做参数拼装与类型标注，不写业务逻辑；
- 所有类型在 `types/api.d.ts` 定义，与契约文档保持同步（契约变更先改类型）。

## 2. 状态管理（Pinia）

| Store | 关键状态 | 持久化 |
|---|---|---|
| `useAuthStore` | access_token、refresh_token、登录/登出/刷新动作 | 是 |
| `useUserStore` | 用户信息、余额、佣金、通知开关 | 否（进入主布局时拉取） |
| `useConfigStore` | 站点配置（站点名、TG 链接、客服地址、注册开关、代理政策文案） | 是（24h 缓存） |
| `useSubscribeStore` | 当前订阅、流量统计、订阅链接 | 否（窗口聚焦刷新） |
| `useNoticeStore` | 公告列表、分页 | 否 |
| `usePlanStore` | 套餐列表 | 会话级缓存 |
| `useOrderStore` | 订单列表、分页、当前详情、轮询句柄 | 否 |
| `useInviteStore` | 邀请统计、邀请码、佣金记录 | 否 |
| `useKnowledgeStore` | 文档分类列表、当前文章 | 否 |
| `useTicketStore` | 工单列表、当前会话 | 否 |
| `useServerStore` | 节点分组与状态 | 否（60s 刷新） |
| `useAppStore` | 侧边栏折叠、语言、主题模式、后端地址 | 是 |

约定：
- store 是唯一允许调用 api 模块的地方；页面组件读写 store，不直接发请求。
- 轮询/定时器句柄存放在对应 store，路由离开时统一清理（`$reset` 或显式 stop）。
- 例外（务实决策）：管理后台视图（`views/admin/*`）直接调用 `apiAdmin` 不经 store 中转——管理端数据一次性拉取、无跨页共享状态；用户端业务视图一律经 store（Profile / 下单弹窗已按此约定收敛）。

## 3. 持久化适配（`utils/storage.ts`）

- 接口：`getItem / setItem / removeItem`，内部 Tauri → `@tauri-apps/plugin-store`（`app-settings.json`），浏览器 → `localStorage`。
- 持久化内容：token 对、主题模式、语言、侧边栏折叠、后端地址、站点配置缓存。
- 注意：token 不落盘日志、不参与 URL；桌面端如需更高安全可后续换 keyring 存储（接口保持兼容）。

## 4. 格式化与工具（composables/utils）

| 工具 | 说明 |
|---|---|
| `useTrafficFormat` | 字节 → `100.00 G / 512.3 M`；速率 → `100 Mbps`；百分比计算 |
| `formatMoney` | 元 number → `¥10.00`（千分位、两位小数） |
| `formatTime` | RFC3339 → `YYYY/M/D HH:mm:ss`（截图风格）、相对时间（N 天前/已过期 N 天） |
| `useCountdown` | 验证码 60s、支付二维码有效期倒计时 |
| 轮询（手写） | 订单/节点轮询在对应 store 内手写实现（`setInterval`），未提供通用 `usePolling`；订单 5s、节点 60s，路由离开时统一 stop |
| `useMediaQuery` | 断点判断，驱动布局降级 |

## 5. 国际化（vue-i18n）

- 语言包为单文件扁平对象（`locales/zh-CN.ts` / `en-US.ts`），按需动态 import 懒加载（`src/i18n.ts`）。
- 初始语言：持久化值 → 浏览器 `navigator.language` → 回退 `zh-CN`。
- 后端多语言字段（知识库、公告）按当前语言参数请求，前端不做翻译。
- 顶栏语言下拉切换后：更新 `Accept-Language`、重拉带语言的数据、持久化选择。

## 6. 订阅链接与一键导入

### 6.1 订阅链接

- 来源：`GET /user/subscribe` 返回 `subscribe_url`（形如 `https://api.example.com/api/v1/client/subscribe/{token}`）。
- 「订阅链接」按钮 = 复制该 URL；「重置订阅信息」后旧链接立即失效，需重新复制/导入。

### 6.2 一键导入 scheme 表（`utils/deeplink.ts`）

| 客户端 | 导入 URL 规则 |
|---|---|
| Clash / Clash Meta / Clash Verge | `clash://install-config?url=<urlencoded subscribe_url>&name=<站点名>` |
| sing-box（桌面/移动） | `sing-box://import-remote-profile?url=<urlencoded>#<站点名>` |
| Shadowrocket（iOS） | `shadowrocket://add/sub://<base64(subscribe_url)>` |
| v2rayN / v2rayNG | 复制 base64 订阅链接后提示导入，或直接 `v2rayng://install-sub?url=` |
| Quantumult X / Surge / Loon | 复制订阅链接 + 打开对应 App 引导文案 |

实现要点：
- 统一入口 `importToClient(client: ClientKind, url: string)`，Tauri 用 opener 插件、浏览器用 `location.href` 触发 scheme；
- 唤起失败（未安装）无回调可感知，UI 上提供「复制订阅链接」兜底按钮与安装教程入口（跳 `/docs` 对应分类）。

## 7. 错误处理与用户反馈

| 场景 | 行为 |
|---|---|
| 业务错误（code != 0） | toast 显示后端 `message`；表单类错误优先内联 |
| 401 且刷新失败 | 清会话 → 跳登录，toast「登录已过期」 |
| 403 | toast「无权限」；邮箱未验证场景跳验证引导页 |
| 429 | toast「操作太频繁，请稍后再试」 |
| 5xx / 网络错误 | toast「服务异常，请稍后再试」；关键页（仪表板）显示重试按钮 |
| 支付/下单类写操作 | 按钮 loading + 防重复提交；创建订单支持幂等键（见契约） |

## 8. 缓存与刷新策略

| 数据 | 策略 |
|---|---|
| 站点配置 | 启动拉取 + 24h 本地缓存，设置页可手动刷新 |
| 仪表板统计/订阅信息 | 进入页面拉取；窗口/标签页重新聚焦时静默刷新 |
| 公告/文档/套餐 | 进入页面拉取，会话内缓存，手动下拉刷新失效重拉 |
| 订单列表 | 每次进入拉取；存在待支付订单时 5s 轮询 |
| 节点状态 | 进入拉取 + 60s 静默轮询 |
