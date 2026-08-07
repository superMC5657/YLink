# 前端开发文档 · 设计系统

> 目标：还原截图的「浅色柔和卡片风」并整体升级为更精致的视觉品质，同时提供一套完整、不刺眼的暗色模式，以及桌面/平板/手机三端一致的响应式体验。所有视觉决策以 **CSS 变量令牌** 落地，业务代码不写死任何颜色。

## 1. 设计原则

1. **柔和优先**：低对比背景 + 白色浮起卡片 + 极软阴影，信息层级靠留白与字重区分，不靠重边框。
2. **圆润一致**：卡片 16px 圆角、按钮/标签全圆角胶囊，全站统一。
3. **点缀式用色**：主紫只用于关键行动点与激活态；语义色（成功/警告/危险/营销红）小面积出现。
4. **微动效**：卡片 hover 上浮、页面切换淡入滑移、数字滚动，时长 150–250ms，克制不花哨。
5. **暗色即一等公民**：暗色不是简单反色，背景、阴影、插画、图表全部有暗色专属处理。

## 2. 色彩令牌

### 2.1 浅色主题（默认，取自截图）

```css
:root {
  /* 主色 */
  --c-primary: #6558F5;          /* 主紫：按钮、链接、进度条 */
  --c-primary-hover: #7A6EF7;
  --c-primary-soft: #EAE7FF;     /* 侧边栏激活胶囊底、选中底 */
  --c-primary-text: #4B3ED6;     /* 激活态文字 */

  /* 中性 */
  --c-bg-app: #F5F6FB;           /* 内容区底 */
  --c-bg-card: #FFFFFF;
  --c-bg-hover: #F2F3FA;
  --c-border: #EBECF4;
  --c-text: #1F2430;
  --c-text-sub: #8A8FA3;
  --c-text-inverse: #FFFFFF;

  /* 语义 */
  --c-success: #5BA829;          /* 余额数字、已完成徽章 */
  --c-success-bg: #DDF3C6;
  --c-warning: #D98E04;
  --c-warning-bg: #FCEFCE;
  --c-danger: #E5484D;           /* 过期标签、重置订阅按钮 */
  --c-danger-bg: #FDE3E4;
  --c-marketing: #E33D5B;        /* 套餐卡片中的营销红字 */
  --c-olive: #7C9A3D;            /* 保存/新增邀请码 橄榄绿按钮 */
  --c-pink: #C2487B;             /* 佣金数字 */
  --c-amber-icon: #F5A524;       /* 公告铃铛等图标点缀 */
}
```

### 2.2 暗色主题

```css
[data-theme='dark'] {
  --c-primary: #7C72FF;          /* 暗色下提亮主色保证对比度 */
  --c-primary-hover: #8F86FF;
  --c-primary-soft: rgba(124, 114, 255, .16);
  --c-primary-text: #A79DFF;

  --c-bg-app: #0F1117;           /* 近黑蓝灰，避免纯黑 */
  --c-bg-card: #171B26;
  --c-bg-hover: #1E2331;
  --c-border: rgba(255, 255, 255, .07);
  --c-text: #E8EAF2;
  --c-text-sub: #9BA1B7;

  --c-success: #7BD88F;
  --c-success-bg: rgba(123, 216, 143, .14);
  --c-warning: #F0B24A;
  --c-warning-bg: rgba(240, 178, 74, .14);
  --c-danger: #F16A6E;
  --c-danger-bg: rgba(241, 106, 110, .14);
  --c-marketing: #FF6B88;
  --c-olive: #9DBE5F;
  --c-pink: #E57BA6;
  --c-amber-icon: #F5A524;
}
```

对比度要求：正文/背景对比度 ≥ 7:1，辅助文字 ≥ 4.5:1；暗色下禁用纯黑 `#000` 与纯白 `#FFF` 大面积对撞。

### 2.3 渐变与氛围（美观增强）

- 主按钮/关键数字可用主色渐变：`linear-gradient(135deg, #6558F5, #8B5CF6)`（暗色 `#7C72FF → #A78BFA`）。
- 仪表板 Banner 卡保留背景插画；暗色模式叠加 `rgba(10,12,20,.45)` 遮罩保证文字可读，或使用暗色专属插画。
- 内容区底可加极低透明度（3%–5%）的品牌纹理/几何插画，暗色下同图降透明度至 2%。
- 卡片可选 `backdrop-filter: blur(12px)` 毛玻璃（仅顶栏、抽屉、移动底栏等浮层使用，普通卡片保持实色以保性能）。

## 3. 字体与排版

| 用途 | 字体 | 说明 |
|---|---|---|
| 界面文字 | `Inter, "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif` | 中西文混排清晰 |
| 数字展示（价格/流量） | `"DIN Alternate", "Bahnschrift", Inter, sans-serif` + `font-feature-settings: "tnum"` | 大数字等宽对齐，还原截图的展示感 |
| 字号阶梯 | 12 / 13 / 14(正文) / 16 / 18 / 20 / 24 / 28 / 32 | 套餐价格等大数字用 32–40 |
| 字重 | 400 正文 / 500 强调 / 600 标题 / 700 大数字 | — |
| 行高 | 正文 1.6，标题 1.3 | — |

## 4. 形状、阴影、间距、动效

```css
:root {
  --r-card: 16px;  --r-control: 10px;  --r-pill: 999px;
  --s-card: 0 8px 24px rgba(23, 25, 66, .06);
  --s-card-hover: 0 12px 32px rgba(23, 25, 66, .10);
  --s-pop: 0 16px 48px rgba(23, 25, 66, .16);
  --t-fast: .15s ease;  --t-base: .22s ease;
}
[data-theme='dark'] {
  --s-card: 0 8px 24px rgba(0, 0, 0, .35);          /* 暗色阴影更深更软 */
  --s-card-hover: 0 12px 32px rgba(0, 0, 0, .45);
  --s-pop: 0 16px 48px rgba(0, 0, 0, .55);
}
```

- 间距系统：4px 基准（4/8/12/16/24/32/48），卡片内边距桌面 24px、移动端 16px。
- 动效规范：hover 上浮 `translateY(-2px)` + 阴影加深；路由切换 200ms 淡入+8px 上移；数字变化 300ms 滚动动画；骨架屏闪烁 1.2s。尊重 `prefers-reduced-motion`。

## 5. 暗色模式实现

1. **机制**：`<html data-theme="light|dark">` + CSS 变量切换；Naive UI 同步传入 `darkTheme` 与 `themeOverrides`（主色/圆角/卡片底均引用同一套令牌，避免两套视觉漂移）。
2. **三种模式**：跟随系统 / 浅色 / 深色，设置持久化（Tauri Store / localStorage）。跟随系统用 `prefers-color-scheme` 监听，系统切换时实时响应。
3. **防闪烁**：SPA 入口 `index.html` 内联一段首帧脚本，在渲染前读持久化值并写入 `data-theme`，避免亮暗闪烁（FOUC）。
4. **联动项**：
   - ECharts：按主题切换 `light/dark` 主题并刷新实例；
   - 图片：Banner 插画暗色加遮罩层；纯黑描边图标一律使用 Iconify 线性图标（随 `currentColor` 变色），不使用位图图标；
   - 编辑器/代码块（知识库 Markdown）：暗色使用 `github-dark` 高亮主题；
   - Tauri 桌面端：窗口标题栏跟随主题（`set_theme`），托盘图标提供亮暗两套。
5. **验收**：所有页面截图走查一遍亮/暗两套；禁止出现写死颜色导致的「暗色下白块」。

## 6. 响应式与移动端适配

### 6.1 断点

| 断点 | 范围 | 布局模式 |
|---|---|---|
| `sm` | < 768px（手机） | 顶栏 + 单列内容 + **底部标签栏**（仪表板/购买/工单/我的）+ 左上角抽屉菜单承载完整导航 |
| `md` | 768–1024px（平板） | 折叠为图标的迷你侧边栏 + 两列卡片 |
| `lg` | > 1024px（桌面） | 完整分组侧边栏 + 顶栏 + 多列卡片 |

断点与 UnoCSS 对齐（`sm/md/lg`），组件内逻辑分支用 `useMediaQuery`。

### 6.2 导航结构降级

- 桌面：侧边栏分组（基础/财务/订阅/用户），激活项胶囊高亮，可折叠。
- 手机：底部标签栏只放 4 个高频入口；完整菜单收进抽屉（左侧滑出，分组结构与桌面一致）；顶栏保留站点名、主题切换、用户入口。
- 路由切换时抽屉自动关闭；底栏激活态与路由同步。

### 6.3 组件级降级规则

| 桌面形态 | 移动端形态 |
|---|---|
| 订单表格（N 列） | 订单卡片列表（状态徽章 + 关键字段纵排 + 操作按钮），保留「表格/卡片」手动切换 |
| 套餐 4 列卡片网格 | 单列全宽卡片，价格区置顶，横滑可选 |
| 仪表板双列（左 Banner+快捷 / 右公告+订阅） | 依次单列堆叠：Banner → 订阅卡 → 公告 → 快捷操作（宫格 4 列） |
| 邀请页左表格右统计 | 统计卡两行网格置顶，表格转卡片 |
| 表格分页器 | 「加载更多」按钮 + 下拉刷新（可选），保留页码版于平板以上 |
| 弹窗 Modal | 底部弹出抽屉（Bottom Sheet），全屏宽，圆角顶部 |

### 6.4 移动端细节

- viewport：`width=device-width, initial-scale=1, viewport-fit=cover`；底部标签栏与抽屉适配 `env(safe-area-inset-bottom)`。
- 触控目标 ≥ 44×44px；列表行高 ≥ 52px；点击态用 `:active` 缩放 0.98 反馈，禁用移动端 300ms 点击延迟（`touch-action: manipulation`）。
- 禁止水平滚动：所有表格/代码块容器 `overflow-x: auto` 并带滚动阴影提示。
- 长数字（订单号）等宽字体 + 断行 `word-break: break-all` + 一键复制按钮。
- 输入框字号 ≥ 16px，避免 iOS 聚焦自动放大。
- 横屏不单独适配，保证不破碎即可。

## 7. 核心组件规范

| 组件 | 规范要点 |
|---|---|
| `Card` | 白底/暗色实底、`--r-card`、`--s-card`、hover 可选上浮；标题区 16px/600 |
| `StatNumber` | 大数字（数字字体）+ 单位小字 + 下方说明文案，支持颜色语义（余额绿/佣金粉） |
| `StatusBadge` | 胶囊徽章：已完成=success、待支付=warning、已取消=neutral、已过期=danger |
| `PriceText` | 货币符号小字 + 整数大字 + 小数字字，营销划线价支持 |
| `EmptyState` | 居中插画 + 「暂无数据」，暗色配专属插画透明度 |
| `Skeleton` | 卡片/列表/表格三种骨架，首屏加载统一使用 |
| `PageHeader` | 页面标题 + 右侧操作区（移动端操作收进「…」菜单） |
| `Pagination` | Naive 分页皮肤覆盖：圆角页码、主色激活、移动端简化 |
| `CopyText` | 订单号/订阅链接/邀请码：省略显示 + 复制按钮 + 成功反馈 |
| `ThemeToggle` | 亮/暗/跟随系统三态，图标线性动画切换 |

## 8. 图标与插画

- 图标统一 Iconify：界面用 MingCute Line，语义图标用 Solar Line；尺寸 16/20/24 三档，`currentColor` 着色。
- 空态/登录页插画：自绘或选用统一风格的开源插画集（如 unDraw），暗色下统一降透明度至 80% 并叠冷色调。
- Banner 插画：1920×480 @2x，WebP；明暗两套资源命名 `banner-light.webp / banner-dark.webp`。
