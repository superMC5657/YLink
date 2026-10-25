# [P1] 前端正确性批修（mock 契约 / 回落色 / 日期 / 支付方式 / 图标）

Status: resolved
Type: task

## Finding

1. `mock/admin.ts:756-761`：node settings `command_template` 手写默认值生成 `--endpoint {api_url} --node-id {node_id} --node-key {node_key}`——`deploy/install-agent.sh:36-44` 不识别 `--node-id/--node-key` 会 exit 1，且占位符应为 `{{x}}`。后端 0009 规范默认是 `--endpoint {{endpoint}} --key {{node_key}}`。mock 必须镜像契约（README §7.1）。
2. `views/traffic/TrafficView.vue:120`、`views/admin/AdminReportsView.vue:306`：`cssVar` 回落值与令牌不符（`--c-pink` 回落 `#8B5CF6` 是渐变紫，正确为 `#C2487B`；Reports 的 olive/warning/blue 回落也各偏色）。
3. `views/admin/AdminTrafficImportView.vue:68`：重置行默认日期用 `toISOString().slice(0,10)`（UTC），同文件 27/33 用 `localDateKey`（本地）——东八区 0:00-7:59 差一天；`:321` `toLocaleString()` 绕过 `formatTime`。
4. `components/business/OrderDetailModal.vue:159-163`：支付方式硬编码（无视 `config.paymentMethods`，禁用的余额仍显示、启用的渠道缺失）；同文件 133-139 与 `PaymentModal.vue:46-53` 各自重复实现 `payMethodLabel`（正源 `utils/format.ts:126`）。OrderConfirmModal.vue:59-62 已是配置驱动。
5. `components/business/ImportClientSheet.vue:71`：`opt.kind === 'sing-box' ? 'box' : …`，但 AppIcon PATHS 无 `box` key → sing-box 回落 info 圆圈图标。

## Plan

1. mock `command_template` 对齐 0009 规范值；mock 安装命令相关测试同步。
2. `cssVar` 回落值全部改为对应令牌字面量（pink `#C2487B`、olive `#7C9A3D`、warning `#D98E04`、blue `#4b8fe5`，以 tokens.css 为准）。
3. `AdminTrafficImportView` 日期统一 `localDateKey`；`:321` 改 `formatTime`。
4. OrderDetailModal/PaymentModal/OrderConfirmModal 抽共享 `PaymentMethodPicker`（配置驱动 + 复用 format.ts 的 `payMethodLabel`），三处接入。
5. ImportClientSheet 的 sing-box 图标改映射到已有 icon（如 `server`）或给 AppIcon 补 `box` path。

## Acceptance

- mock 生成的安装命令可被 install-agent.sh 接受（`sh -c` 干跑参数解析）。
- 断网/异常时图表与文案不再出现偏色的回落色。
- 流量导入在 UTC+8 早 8 点前默认日期正确；三个支付入口的渠道列表均来自 config。
- `pnpm test` + e2e（withdraw/admin）全绿；check-error-toast / check-admin-contract 通过。

## Comments

- 2026-10-25: 已修复——① mock node 设置整键对齐 0009 契约：`install_script_url`/`install_command_template`（原 mock 用陈旧键名 `deploy_script_url`/`command_template`，对话框根本读不到）+ 默认模板逐字取 0009 种子值（`{{}}` 占位符、`--endpoint/--key` 参数，可被 install-agent.sh 参数解析接受）；② cssVar 回落色对齐令牌字面量（pink `#C2487B`×2、olive `#7C9A3D`、warning `#D98E04`、blue `#4B8FE5`）；③ AdminTrafficImportView 重置行默认日期改 `localDateKey`、重置记录时间改 `formatTime`；④ 新增共享 `PaymentMethodPicker`（渠道唯一来源 config.payment_methods、余额按应付额禁用+差额提示），OrderConfirmModal/OrderDetailModal 接入（OrderDetailModal 此前硬编码三渠道），PaymentModal 与 OrderDetailModal 的展示标签统一走 `utils/format.payMethodLabel`；⑤ ImportClientSheet 的 sing-box 图标 `box`（PATHS 无此键）改映射 `server`。docs/api §16 node 设置键陈旧描述同步为 install_* 形态；`pnpm lint`+`pnpm test`（239）+ e2e withdraw/admin（34）全绿。
- 2026-10-25: 范围外发现（未修）：OrderConfirmModal/OrderDetailModal 的 payMethod 缺省值硬编码 `'epay_alipay'`，若该渠道被禁用则无默认选中项；建议后续改为「首个可用渠道」。
