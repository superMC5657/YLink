# 前端开发文档 · 路由与页面拆解

> 对照 7 张截图逐页拆解组件树、数据来源与交互。接口路径引用见 [../api/README.md](../api/README.md)；视觉规范见 [design-system.md](design-system.md)。

## 1. 路由表

| 路径 | 页面 | 布局 | 鉴权 | 移动端导航归属 |
|---|---|---|---|---|
| `/login` | 登录 | AuthLayout | 游客 | — |
| `/register` | 注册 | AuthLayout | 游客 | — |
| `/forgot` | 找回密码 | AuthLayout | 游客 | — |
| `/dashboard` | 仪表板（首页） | MainLayout | 登录 | 底栏 Tab 1 |
| `/docs` | 使用文档 | MainLayout | 登录 | 抽屉-基础 |
| `/docs/:id` | 文档详情 | MainLayout | 登录 | — |
| `/orders` | 我的订单 | MainLayout | 登录 | 抽屉-财务 |
| `/invite` | 邀请赚钱 | MainLayout | 登录 | 抽屉-财务 |
| `/agent` | 申请代理 | MainLayout | 登录 | 抽屉-财务 |
| `/plans` | 购买订阅 | MainLayout | 登录 | 底栏 Tab 2 |
| `/nodes` | 节点状态 | MainLayout | 登录 | 抽屉-订阅 |
| `/profile` | 个人信息 | MainLayout | 登录 | 底栏 Tab 4（我的） |
| `/tickets` | 我的工单 | MainLayout | 登录 | 底栏 Tab 3 |
| `/tickets/:id` | 工单详情 | MainLayout | 登录 | — |
| `/traffic` | 流量明细 | MainLayout | 登录 | 抽屉-用户 |
| `/portal` | 门户分流页（用户中心/管理后台二选一） | 无（独立全屏页） | 管理员(role=1) | — |
| `/admin` | → redirect `/admin/overview` | AdminLayout | 管理员 | — |
| `/admin/overview` | 管理后台·总览 | AdminLayout | 管理员(role=1) | 管理端侧边栏-管理后台 |
| `/admin/users` | 管理后台·用户 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/audit-logs` | 管理后台·审计日志（F08，只读查询） | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/plans` | 管理后台·套餐 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/nodes` | 管理后台·节点 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/orders` | 管理后台·订单 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/tickets` | 管理后台·工单 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/coupons` | 管理后台·优惠券 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/notices` | 管理后台·公告 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/knowledges` | 管理后台·知识库 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/agent-applies` | 管理后台·代理审批 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/commission-logs` | 管理后台·佣金日志 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/traffic-import` | 管理后台·流量导入 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/admin/settings` | 管理后台·站点设置 | AdminLayout | 管理员 | 管理端侧边栏-管理后台 |
| `/:pathMatch(.*)*` | 404 | MainLayout | — | — |

守卫规则：
- `meta.guest` 页面：已登录访问 → 管理员重定向 `/portal`，普通用户重定向 `/dashboard`。
- 其余页面：未登录跳转 `/login?redirect=<原路径>`，登录成功后回跳；登录提交成功若无 `redirect` 参数，管理员 → `/portal`，普通用户 → `/dashboard`（见 LoginView.vue）。
- `meta.admin` 页面（`/portal` 与 `/admin/*`，父记录 AdminLayout 挂 `meta.admin`，vue-router 父子 meta 自动合并）：非管理员（role≠1）访问重定向 `/dashboard`（见 [progress.md](progress.md) M8）。
- Token 过期由 http 层静默刷新；刷新失败才清会话跳登录（见 [data-layer.md](data-layer.md)）。

## 2. 布局

### 2.1 MainLayout（桌面 ≥1024px）

```
┌────────────┬──────────────────────────────────────────┐
│ AppSidebar │ AppHeader（折叠/站点名｜主题/语言/用户chip）│
│  分组菜单   ├──────────────────────────────────────────┤
│  基础       │                                          │
│  财务       │            <router-view>                 │
│  订阅       │         （滚动容器，浅灰底）              │
│  用户       │                                          │
└────────────┴──────────────────────────────────────────┘
                                    右下角：客服浮球（可配置外链）
```

- AppSidebar：宽 240px，折叠后 72px（仅图标 + tooltip）；激活项为 `--c-primary-soft` 底胶囊 + `--c-primary-text` 文字；分组标题小字 `--c-text-sub`。菜单源仅 `NAV_GROUPS`（基础/财务/订阅/用户 4 组 10 项），**不含管理菜单**；管理员在底部额外有「进入管理后台」按钮（`v-if="auth.isAdmin"`，跳 `/admin/overview`）。
- AppHeader：吸顶，毛玻璃（`backdrop-filter`），含侧边栏折叠钮、站点名、主题切换、语言下拉、用户邮箱 chip（点击进 `/profile`，下拉含退出登录）；管理员下拉首项为「进入管理后台」（对称互切，见 A4）。

### 2.2 MainLayout（手机 <768px）

```
┌──────────────────────────┐
│ AppHeader（☰ 站点名 主题 头像）│
├──────────────────────────┤
│        <router-view>      │
│      （单列，16px 内边距） │
├──────────────────────────┤
│ MobileTabBar（4 Tab + safe-area）│
└──────────────────────────┘
```

- 抽屉菜单 DrawerMenu：左滑出，内容与桌面侧边栏一致（仅用户端菜单，**不含管理菜单**），底部附用户信息与退出。
- TabBar：仪表板 / 购买订阅 / 我的工单 / 我的（`/profile`），图标 + 文字，激活主色。

### 2.3 AuthLayout

居中卡片（420px）+ 品牌区插画背景；移动端全屏卡片贴顶。登录/注册/找回三页共享，右上角放语言切换与主题切换。

### 2.4 AdminLayout（管理端独立布局，`/admin/*`）

```
┌────────────┬──────────────────────────────────────────┐
│AdminSidebar│ AppHeader(admin)（折叠/站点名/管理后台徽标｜主题/语言/用户）│
│  管理后台   ├──────────────────────────────────────────┤
│ 13 项菜单   │            <router-view>                 │
│(仅此一个源) │         （滚动容器）                      │
│            │                                          │
│ 返回用户中心 │                                          │
└────────────┴──────────────────────────────────────────┘
```

- AdminSidebar（`src/components/app/AdminSidebar.vue`）：唯一菜单源 `ADMIN_NAV_GROUPS`（13 项），底部固定「返回用户中心」按钮（回 `/dashboard`）；`fill` 模式（宽度 100%、固定展开、隐藏折叠钮）供移动端抽屉复用。
- AppHeader 传 `admin` prop：站点名旁显示「管理后台」徽标，用户下拉首项为「返回用户中心」（与用户端「进入管理后台」对称互切）。
- 手机 <768px：顶栏汉堡打开**独立抽屉**（内容与桌面侧边栏一致，仅管理菜单）；**不渲染 MobileTabBar、不渲染客服浮球**。
- 不接 `usePullToRefresh` 等用户端设施。

### 2.5 门户分流页 `/portal`（独立全屏页）

- `src/views/portal/PortalView.vue`：视觉复用 AuthLayout 风格（氛围背景 + 居中内容 + 右上语言/主题），不带任何侧边菜单。
- 品牌 Logo + 站点名 + 欢迎语（含当前登录邮箱）；两张分流卡片：「用户中心 → /dashboard」「管理后台 → /admin/overview」；底部退出登录。
- 仅管理员可进（`meta.admin` 兜底）；管理员登录默认落点即此页。

## 3. 逐页拆解

### 3.1 登录 `/login`（截图未含，按风格补齐）

- 组件：`AuthCard`（站点 Logo + 欢迎语）、`NForm`（邮箱、密码 + 校验）、登录按钮（主色渐变）、注册/找回链接。
- 交互：提交中按钮 loading；失败 toast 显示后端 message；成功写 `useAuthStore` 并按 `redirect` 回跳；无 `redirect` 时管理员 → `/portal`、普通用户 → `/dashboard`。
- 数据：`POST /auth/login`。

### 3.2 注册 `/register` / 找回 `/forgot`

- 注册字段：邮箱、邮箱验证码（60s 倒计时按钮）、密码、确认密码、邀请码（选填，URL `?code=` 自动填充）。
- 表单样式：与登录页一致——`n-form` 关闭 feedback 占位（`:show-feedback="false"`），`.n-form-item` 纵向间距收为 4px；注册按钮带 `mt-9`（36px），对齐登录页「忘记密码」行撑出的按钮上方净空（≈38px）（2026-08-30 对齐）。
- 找回字段：邮箱、验证码、新密码。
- 数据：`POST /captcha/email`、`POST /auth/register`、`POST /auth/forgot`。
- 校验：邮箱格式、密码 ≥8 位且含字母数字、两次一致；错误内联显示。

### 3.3 仪表板 `/dashboard`（截图1）

```
DashboardPage
├── BannerStatCard            # 背景插画 + 圆形Logo + 钱包余额(绿)/我的佣金(粉)
├── SubscribeCard             # 当前订阅：名称、过期标签、用量进度条、五宫格数据
├── NoticePanel               # 公告(5)：手风琴列表(图标/标题/时间/展开)
└── QuickActionGrid           # 9个快捷入口（宫格）
```

- 数据来源：`GET /user/stat`（余额/佣金）、`GET /user/subscribe`、`GET /notices?page_size=5`。
- 快捷操作映射：
  - 查看教程 → `/docs`；邀请赚钱 → `/invite`；购买订阅 → `/plans`；免费流量 → 说明弹窗（内容取站点配置）；APP 下载 → 下载页弹窗（桌面端展示本站下载地址）；我的工单 → `/tickets`；个人信息 → `/profile`；
  - **订阅链接** → 复制订阅 URL（`CopyText` + 成功 toast）；
  - **一键导入** → 弹出客户端选择（Clash / Clash Meta / sing-box / Shadowrocket / v2rayN），按选择打开对应 scheme（见 [data-layer.md](data-layer.md) 第 7 节）。
- SubscribeCard 细节：`已过期 N 天` 用 danger 徽章；剩余天数 >7 不提示、≤7 天 warning 徽章；流量百分比进度条主色渐变，已用 ≥80% 变 warning、≥95% 变 danger；窗口聚焦时静默刷新。
- 移动端顺序：Banner → SubscribeCard → NoticePanel → QuickActionGrid（4 列宫格）。

### 3.4 使用文档 `/docs`（截图2）

- 组件：搜索框（防抖 300ms，前端过滤当前列表 + 传 `keyword` 重新拉取）、`KnowledgeGroupCard` × N（分类：防失联/新手知识科普/安卓配置/苹果配置/Windows/MacOS…分类由后端返回动态渲染）。
- 语言：复用网站全局语言（顶栏语言切换），`app.language` 变化时自动重新拉取对应语言的文档，页面内不提供独立语言下拉。
- 文章行：标题 + 更新时间 + 「阅读 →」（主色文字链接）。
- 数据：`GET /knowledges?keyword=&language=`。
- 详情页 `/docs/:id`：Markdown 渲染（markdown-it + DOMPurify + 代码高亮），顶部返回 + 标题 + 更新时间；移动端全宽阅读，正文字号 16px。

### 3.5 我的订单 `/orders`（截图3）

- 组件：`PageHeader`（标题 + 表格/卡片视图切换）、`OrderTable`（桌面）/`OrderCardList`（移动或手动切换）、分页器。
- 列：产品名称、订单号（CopyText）、周期（月付/季付/年付/一次性）、订单金额、订单状态（StatusBadge：待支付/已完成/已取消）、创建时间、操作（查看详情）。
- 交互：
  - 查看详情 → `OrderDetailModal`：屏幕中央弹窗展示订单全字段 + 支付入口（待支付时）+ 取消按钮；
  - 待支付订单可「去支付」直接唤起收银台弹窗（复用购买页组件）；
  - 空态 EmptyState「暂无数据」。
- 数据：`GET /orders?status=&page=&page_size=`；取消 `POST /orders/{order_no}/cancel`。
- 轮询：存在待支付订单时每 5s 轮询一次状态（页面驻留才轮询，离开即停）。

### 3.6 邀请赚钱 `/invite`（截图4）

- 左列：`InviteCodeCard`（表格：邀请码/创建时间/操作=复制；右上「新增邀请码」橄榄绿胶囊按钮；注册链接 = 站点地址 + `?code=`）、`CommissionRecordCard`（佣金发放记录：发放时间/佣金，分页）。
- 右列 5 张 `StatCard`：我的佣金（含「划转到余额」按钮）、佣金比例、已注册用户数、累计获得佣金、确认中的佣金；各配不同色线性图标。
- 交互：划转弹窗输入金额（≤ 可划转余额），成功后刷新统计与余额；新增邀请码上限由后端控制，超限提示。
- 数据：`GET /invite/summary`、`GET /invite/codes`、`POST /invite/codes`、`GET /invite/records`、`POST /invite/transfer`。
- 移动端：统计卡两行网格置顶，表格转卡片。

### 3.7 申请代理 `/agent`（截图5）

- 左：`AgentApplyCard`——信息图标 + 「成为代理商」+ 描述 + 状态按钮（未满足条件=灰禁用 / 可申请=主色 / 审核中=warning / 已是代理=success 展示特权）。
- 右：`ConditionList`（加盟条件，逐项 ✓/✗ 图标 + 进度，如「已邀请 0 人，还需 50 人」）、`BenefitList`（佣金比例/套餐福利/订单推送/审验周期）、`NoticeListCard`（注意事项有序列表）。
- 数据：`GET /agent/status`、`POST /agent/apply`。
- 内容（条件、特权文案）从站点配置读取，运营可改。

### 3.8 购买订阅 `/plans`（截图6）

- 组件：`PlanCard` 网格（桌面 3–4 列、平板 2 列、手机单列）。
- PlanCard 结构：套餐名（居中 18px/600）→ 价格区（PriceText：`¥10.00 /月付`，多周期时展示当前选中周期价）→ 流量/带宽行（`流量: 300 G`、`带宽: 300 Mbps`）→ 描述富文本（Markdown，支持红色营销文案）→ 「立即购买」胶囊按钮（描边主色，hover 填充）。
- 购买弹窗 `OrderConfirmModal`（移动端为 Bottom Sheet）：
  1. 周期选择（单选胶囊组，价格实时联动，标出「省 N%」）；
  2. 优惠券输入 + 校验按钮（成功显示减免金额，失败红字提示）；
  3. 支付方式单选（余额 / 支付宝 / 微信…，余额不足置灰并提示差额）；
  4. 费用明细（套餐价 − 优惠 − 余额抵扣 = 应付）；
  5. 提交 → 余额支付直接完成；在线支付进入 `PaymentModal`（二维码/跳转链接 + 倒计时 + 每 3s 轮询订单状态，支付成功自动跳「支付成功」结果卡并刷新订阅信息）。
- 数据：`GET /plans`、`POST /coupons/check`、`POST /orders`、`POST /orders/{no}/checkout`、`GET /orders/{no}`（轮询）。

### 3.9 节点状态 `/nodes`（侧边栏入口，截图未含）

- 组件：`NodeGroupCard`（按分组展示）+ `NodeRow`（节点名、类型标签、倍率、状态点：正常=绿/拥挤=黄/维护=灰、延迟或负载二选一展示）。
- 安全约束：该接口**不返回** host/port 等连接信息，仅展示状态；连接信息只通过订阅下发。
- 数据：`GET /servers`；60s 静默刷新。

### 3.10 个人信息 `/profile`（截图7）

- 左列：复用 `BannerStatCard`；`NotifySettingCard`（到期邮件提醒/流量邮件提醒两个 Switch，切换即保存并 toast）；`TelegramCard`（加入群组=外链、机器人链接=外链，取站点配置）。
- 右列：`ResetPasswordCard`（旧密码/新密码/确认新密码，下划线输入风格；「清空表单」灰胶囊 +「保存」橄榄绿胶囊）；`ResetSubscribeCard`（警告说明条 + 红色「点击重置」，二次确认弹窗输入密码确认，成功后展示新订阅链接并提示重新导入）。
- 数据：`POST /user/password/change`、`PUT /user/profile`、`POST /user/subscribe/reset`。

### 3.11 我的工单 `/tickets` + 详情 `/tickets/:id`

- 列表：卡片式表格（主题、状态徽章：待回复/已回复/已关闭、更新时间、操作）；右上「新建工单」主色按钮 → 弹窗（主题、优先级、内容）。
- 详情：对话气泡流（用户右/客服左，头像 + 时间），底部回复输入框（已关闭置灰 + 「重新打开」入口），支持关闭工单。
- 移动端：详情页全屏，输入框贴底适配键盘。
- 数据：`GET /tickets`、`POST /tickets`、`GET /tickets/:id`、`POST /tickets/:id/reply`、`POST /tickets/:id/close`。

### 3.12 流量明细 `/traffic`

- 组件：时间范围选择（近 7 天/30 天/自定义）、ECharts 折线/柱状组合图（上行/下行双系列，暗色主题联动）、明细表（日期/上行/下行/合计，移动端转卡片）。
- 数据：`GET /user/traffic-logs?from=&to=`；空数据展示 EmptyState。

### 3.13 404 与全局异常

- 404 页：插画 + 返回首页按钮；接口异常统一 toast；网络断开顶部横幅提示（`navigator.onLine` 监听），恢复自动重试关键数据。

### 3.14 管理后台 `/admin/*`（M8 + M9，13 模块，role=1）

- 布局：独立 `AdminLayout`（§2.4），侧边栏/移动抽屉只含 13 项管理菜单；管理员经门户分流页 `/portal`（§2.5）或用户端底部按钮/顶栏下拉进入；`meta.admin` 挂 AdminLayout 父记录。

M8 核心 6 模块：

- 总览 `/admin/overview`：7 项运营统计卡（用户/代理/订单/收入/在售套餐）+ 快捷操作（按运营频率分组两组共 8 入口：日常运营=用户/订单/工单/代理审批，运营与配置=公告/优惠券/流量/节点）。
- 用户 `/admin/users`：搜索/分页、封禁/角色调整、调余额（均写审计）。
- 套餐 `/admin/plans`：CRUD（周期定价元/流量/设备/限速/节点分组/上架/排序）。
- 节点 `/admin/nodes`：分组 CRUD + 节点 CRUD（6 协议/地址/配置 JSON/倍率/状态/标签）。
- 订单 `/admin/orders`：状态筛选/分页、退款（余额退回+佣金回滚）、关闭待支付订单。
- 工单 `/admin/tickets`：列表/详情、客服回复、关闭。

M9 二期 7 模块（2026-08-11）：

- 优惠券 `/admin/coupons`：列表/新建/编辑/删除（固定金额/百分比、面值/最低消费/限用/适用周期/套餐/生效时间/启停），一键公告（生成公告草稿，优惠码反引号包裹）。
- 公告 `/admin/notices`：列表/发布/编辑/删除（Markdown 正文、展示开关、排序）。
- 知识库 `/admin/knowledges`：列表（语言筛选+搜索）/新建/编辑/删除（分类/正文/语言/展示/排序）。
- 代理审批 `/admin/agent-applies`：状态筛选/分页、通过/拒绝（通过后升级代理商）。
- 佣金日志 `/admin/commission-logs`：状态筛选/分页，展示邀请人/被邀请人/订单/比例/佣金。
- 流量导入 `/admin/traffic-import`：模式 B 手工导入（user_id/date/u/d 字节，批量提交写审计）。
- 站点设置 `/admin/settings`：按 key（site/payment/invite/agent/order/templates）编辑配置 JSON，保存后缓存失效。

- 数据来源：`api/admin.ts` 封装 13 组端点（M8 6 组 + M9 7 组，契约第 16 节）；管理后台视图直调 `apiAdmin` 不经 store（文档化例外，见 data-layer.md §2）。

## 4. 弹窗与全局组件清单

| 名称 | 触发 | 说明 |
|---|---|---|
| `OrderConfirmModal` | 套餐卡「立即购买」 | 下单确认 + 优惠券 + 支付方式 |
| `PaymentModal` | 提交订单后 | 二维码/跳转 + 轮询 + 成功结果 |
| `OrderDetailModal` | 订单列表「查看详情」 | 屏幕中央弹窗 |
| `ImportClientSheet` | 快捷操作「一键导入」 | 客户端选择 + scheme 唤起 |
| `TransferModal` | 邀请页「划转」 | 佣金转余额 |
| `CustomerServiceFab` | 全局右下角 | 外链客服（TG/网页），地址取站点配置 |
