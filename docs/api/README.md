# 接口契约 · API v1（前后端唯一事实来源）

> 本文档是用户端 App（Vue/Tauri）与 Go/Gin 服务端之间的**契约**。前端 Mock、后端实现、Swagger 注解都必须与此一致；变更流程见文末。Base URL：`{host}/api/v1`。

## 1. 通用约定

### 1.1 响应信封（Envelope）

所有业务接口（除订阅下发与支付回调外）统一返回：

```json
{ "code": 0, "message": "ok", "data": {} }
```

- `code=0` 成功，`data` 为业务数据；失败时 `code` 为业务错误码，`message` 为用户可读文案（前端可直接 toast），`data` 为 `null`。
- HTTP 状态码语义化使用：成功 200；参数/业务错误 400；未认证 401；无权限 403；不存在 404；冲突 409；限流 429；服务异常 500。

### 1.2 鉴权

- 请求头：`Authorization: Bearer <access_token>`。
- access_token 有效期 2h，refresh_token 14d；401 时前端用 refresh_token 调 `POST /auth/refresh` 静默换新（旋转机制，旧 refresh 立即作废）。
- 免鉴权接口：`POST /captcha/email`、`POST /auth/*`（login/register/forgot/refresh）、`GET /config`、`GET /notices`、`GET /knowledges`、`GET /knowledges/{id}`、`GET /client/subscribe/{token}`（token 在路径中）、`POST /payment/notify/{method}`。

### 1.3 分页

请求：`page`（从 1 开始，默认 1）、`page_size`（默认 10，最大 50）。响应：

```json
{ "code": 0, "message": "ok", "data": { "list": [], "total": 37, "page": 1, "page_size": 10 } }
```

### 1.4 单位与格式

| 类型 | 约定 |
|---|---|
| 金额 | number，单位「元」，两位小数（如 `10.00`）；服务端内部与数据库存「分」 |
| 流量 | integer，单位「字节」（int64） |
| 速率 | integer，单位 Mbps |
| 时间 | 字符串，RFC3339 带时区（如 `2026-06-24T00:53:35+08:00`）；纯日期用 `YYYY-MM-DD` |
| 语言 | `zh-CN` / `en-US`，请求头 `Accept-Language` 或显式 `language` 参数 |

### 1.5 幂等

`POST /orders` 支持请求头 `Idempotency-Key: <uuid>`：24h 内同 Key 重复提交返回首次创建结果（HTTP 200 + 相同 data），不重复建单。前端在打开下单弹窗时生成 Key，弹窗生命周期内复用。

### 1.6 通用请求头

| 头 | 说明 |
|---|---|
| `Accept-Language` | 影响多语言内容（公告/知识库）与错误文案 |
| `X-Client` | `web / tauri-windows / tauri-macos / tauri-linux`，用于统计 |

---

## 2. 错误码

| code | HTTP | 含义 |
|---|---|---|
| 0 | 200 | 成功 |
| 40000 | 400 | 参数校验失败（message 含字段信息） |
| 40100 | 401 | 未登录或 token 失效 |
| 40101 | 401 | 密码错误 / 凭据无效 |
| 40300 | 403 | 无权限 / 账号被封禁 |
| 40400 | 404 | 资源不存在 |
| 40900 | 409 | 状态冲突（如重复操作） |
| 42900 | 429 | 触发限流 |
| 50000 | 500 | 服务器内部错误 |
| 10001 | 400 | 邮箱已注册 |
| 10002 | 400 | 验证码错误或已过期 |
| 10003 | 429 | 验证码发送过于频繁 |
| 10004 | 400 | 邀请码无效 |
| 11001 | 400 | 套餐不存在或未上架 |
| 11002 | 400 | 套餐不支持所选周期 |
| 11003 | 409 | 订单状态不允许该操作（如已取消再支付） |
| 11004 | 400 | 余额不足 |
| 11005 | 400 | 支付渠道不可用 |
| 12001 | 400 | 优惠券无效/已过期/不满足条件 |
| 13001 | 400 | 邀请码数量已达上限 |
| 13002 | 400 | 可划转佣金不足 |
| 14001 | 409 | 工单已关闭 |
| 15001 | 400 | 不满足代理申请条件 |
| 15002 | 409 | 代理申请审核中，请勿重复提交 |

---

## 3. 站点与验证码

### 3.1 获取站点配置（免登录）

`GET /config`

```json
{
  "code": 0, "message": "ok",
  "data": {
    "site_name": "YLink",
    "site_logo": "https://.../logo.png",
    "site_description": "高速稳定的网络加速服务",
    "register_enabled": true,
    "invite_code_required": false,
    "app_downloads": { "windows": "https://...", "macos": "https://...", "android": "https://..." },
    "telegram": { "group_url": "https://t.me/xxx", "bot_url": "https://t.me/xxx_bot" },
    "customer_service_url": "https://t.me/xxx",
    "free_traffic_tips": "绑定 TG 机器人每天领取免费流量……",
    "agent_policy": {
      "required_valid_invites": 50,
      "commission_rate": 40,
      "benefits": ["佣金比例：40%（循环）", "套餐福利：赠送免费的年付订阅套餐", "订单推送：享受 bot 订单实时推送", "审验周期：12个月"],
      "notes": ["点击按钮申请代理权限，审核通过后将获得以上特权。", "..."]
    },
    "payment_methods": [
      { "code": "balance", "name": "余额支付", "icon": "wallet", "enabled": true },
      { "code": "epay_alipay", "name": "支付宝", "icon": "alipay", "enabled": true },
      { "code": "epay_wxpay", "name": "微信支付", "icon": "wechat", "enabled": true }
    ],
    "languages": ["zh-CN", "en-US"]
  }
}
```

### 3.2 发送邮箱验证码（免登录）

`POST /captcha/email`

请求：`{ "email": "a@b.com", "type": "register" }`（type：`register` / `forgot`）
响应：`{ "code": 0, "message": "发送成功", "data": { "expire_in": 600, "resend_after": 60 } }`
错误：10003 限频；register 时邮箱已存在返回 10001。

---

## 4. 认证

### 4.1 注册

`POST /auth/register`

```json
// 请求
{ "email": "a@b.com", "password": "Passw0rd", "email_code": "123456", "invite_code": "AB12CD34" }
// 响应 data（与登录同构）
{ "access_token": "eyJ...", "refresh_token": "eyJ...", "token_type": "Bearer", "expires_in": 7200,
  "user": { "id": 10086, "email": "a@b.com", "role": 0 } }
```

错误：10001 邮箱已注册 / 10002 验证码错误 / 10004 邀请码无效；`invite_code` 选填（站点开启强制邀请时必填，见 /config）。

### 4.2 登录

`POST /auth/login`

请求：`{ "email": "a@b.com", "password": "Passw0rd" }`；响应同 4.1。
错误：40101 凭据无效；40300 账号封禁；42900 连续失败锁定（message 含剩余锁定时长）。

### 4.3 刷新令牌

`POST /auth/refresh`（免 access 鉴权，body 鉴权）

请求：`{ "refresh_token": "eyJ..." }`；响应同 4.1 的 token 部分。旧 refresh 立即失效（旋转）。

### 4.4 找回密码

`POST /auth/forgot`

请求：`{ "email": "a@b.com", "email_code": "123456", "password": "NewPass1" }`；响应 `data: null`。成功后吊销该用户全部会话。

### 4.5 退出登录

`POST /auth/logout`（需鉴权）。吊销当前 refresh；响应 `data: null`。

---

## 5. 用户

### 5.1 用户信息与仪表板统计

`GET /user/stat`

```json
{ "code": 0, "message": "ok",
  "data": {
    "email": "2734921923@qq.com",
    "balance": 0.00,
    "commission_balance": 0.00,
    "pending_order_count": 0,
    "open_ticket_count": 0,
    "invited_count": 0,
    "is_agent": false
  } }
```

### 5.2 通知设置

`GET /user/profile` — 读取通知设置；响应：`data` 为 `{ "remind_expire": bool, "remind_traffic": bool }`。

`PUT /user/profile`

请求：`{ "remind_expire": true, "remind_traffic": false }`；响应：`data` 为更新后的完整设置对象。

### 5.3 修改密码

`POST /user/password/change`

请求：`{ "old_password": "...", "new_password": "..." }`；错误 40101 旧密码错误。成功后吊销其他会话，当前会话保留。

### 5.4 当前订阅

`GET /user/subscribe`

```json
{ "code": 0, "message": "ok",
  "data": {
    "has_subscription": true,
    "plan": { "id": 4, "name": "猎户座" },
    "expired_at": "2026-07-24T00:53:35+08:00",
    "is_expired": true,
    "expired_days": 15,
    "transfer_enable": 107374182400,
    "u": 0, "d": 0,
    "remaining": 107374182400,
    "used_percent": 0,
    "speed_limit": 100,
    "device_limit": 2,
    "subscribe_url": "https://api.example.com/api/v1/client/subscribe/9f3b...-token"
  } }
```

无订阅时：`{ "has_subscription": false, "plan": null, ... }`（其余字段为 0/null）。

### 5.5 重置订阅信息

`POST /user/subscribe/reset`

请求：`{ "password": "当前登录密码" }`（二次确认）；响应：`{ "subscribe_url": "https://.../subscribe/<new_token>" }`。旧链接立即失效。

### 5.6 流量明细

`GET /user/traffic-logs?from=2026-07-01&to=2026-07-31`

```json
{ "code": 0, "message": "ok",
  "data": { "list": [ { "date": "2026-07-01", "u": 104857600, "d": 2147483648, "total": 2252341248 } ] } }
```

范围最大 90 天，按日期升序；无数据返回空 list。

---

## 6. 公告

`GET /notices?page=1&page_size=5`

```json
{ "code": 0, "message": "ok",
  "data": { "total": 5, "page": 1, "page_size": 5,
    "list": [ { "id": 12, "title": "紧急通知", "content": "## 正文(Markdown)", "created_at": "2026-03-28T00:38:55+08:00" } ] } }
```

按创建时间倒序；`content` 已做 XSS 清洗，前端用 markdown-it 渲染后仍需 DOMPurify 二次过滤。

---

## 7. 知识库（使用文档）

### 7.1 列表（按分类分组）

`GET /knowledges?language=zh-CN&keyword=YLink`

```json
{ "code": 0, "message": "ok",
  "data": {
    "groups": [
      { "category": "安卓配置教程",
        "items": [ { "id": 31, "title": "YLink (推荐使用)", "updated_at": "2026-08-04T23:51:53+08:00" },
                   { "id": 32, "title": "Clash Meta (备用)", "updated_at": "2026-08-05T19:25:25+08:00" } ] },
      { "category": "防失联", "items": [ ] }
    ] } }
```

`keyword` 匹配标题（模糊）；不传 `language` 默认 zh-CN。

### 7.2 详情

`GET /knowledges/{id}`

```json
{ "code": 0, "message": "ok",
  "data": { "id": 31, "category": "安卓配置教程", "title": "YLink (推荐使用)",
            "body": "## 第一步…(Markdown)", "language": "zh-CN", "updated_at": "2026-08-04T23:51:53+08:00" } }
```

---

## 8. 套餐

`GET /plans`

```json
{ "code": 0, "message": "ok",
  "data": {
    "list": [
      { "id": 1, "name": "白羊座",
        "prices": { "month": 10.00, "quarter": 27.00, "year": 96.00 },
        "traffic_gb": 300, "speed_limit": 300, "device_limit": 5,
        "content": "购买套餐后可能需要等待5分钟左右才能连接\n支持**5台**设备同时在线…",
        "sort": 1 },
      { "id": 3, "name": "射手座", "prices": { "month": 20.00 },
        "traffic_gb": 650, "speed_limit": null, "device_limit": 10, "content": "…", "sort": 3 }
    ] } }
```

- `prices` 只返回支持的周期（key：`month/quarter/half_year/year/onetime`）；`speed_limit: null` 表示不限制（前端显示「无限制」）。
- 列表已按 `sort` 排序且只含上架套餐；`content` 为 Markdown（已清洗），营销红字由前端按约定标记渲染。

---

## 9. 优惠券

`POST /coupons/check`

```json
// 请求
{ "code": "618SALE", "plan_id": 1, "period": "month" }
// 成功
{ "code": 0, "message": "ok", "data": { "valid": true, "discount_amount": 2.00, "pay_amount": 8.00 } }
// 失败（HTTP 400）
{ "code": 12001, "message": "优惠券已过期", "data": null }
```

纯试算不落库；下单时服务端重算。

`GET /coupons/available?plan_id=&period=`（需鉴权；`plan_id`/`period` 可选过滤）

返回当前用户可用的优惠券列表（启用 + 生效期内 + 总限量未满 + 每人限用未满；传了 `plan_id`/`period` 时额外过滤适用套餐/周期）：

```json
// 成功
{ "code": 0, "message": "ok", "data": { "list": [
  { "code": "618SALE", "type": 1, "value": 2.00, "min_spend": 0,
    "valid_periods": ["month","quarter","half_year","year"], "plan_ids": [],
    "started_at": null, "ended_at": null },
  { "code": "VIP50", "type": 2, "value": 10, "min_spend": 200,
    "valid_periods": ["year"], "plan_ids": [1,2], "started_at": null, "ended_at": "2026-12-31T23:59:59+08:00" }
] } }
```

- `type`：1=固定金额（`value` 为元）/ 2=百分比（`value` 为百分比数值，如 10 表示 10%）
- `min_spend` 单位为元；`valid_periods`/`plan_ids` 空数组表示不限
- 不返回 `total_limit`/`used_count`/`limit_per_user`（运营内部信息）
- 下单仍需在弹窗输入码（或点选后自动填入）经 `POST /coupons/check` 试算；服务端下单时重算，展示列表仅为便利

---

## 10. 订单与支付

### 10.1 创建订单

`POST /orders`（头：`Idempotency-Key: <uuid>`）

```json
// 请求
{ "plan_id": 1, "period": "month", "coupon_code": "618SALE" }
// 响应
{ "code": 0, "message": "ok",
  "data": { "order_no": "2026062400063525438887716", "plan_name": "白羊座", "period": "month",
            "amount": 10.00, "discount_amount": 2.00, "pay_amount": 8.00,
            "status": 0, "created_at": "2026-06-24T00:53:35+08:00" } }
```

错误：11001/11002 套餐与周期；12001 优惠券；`coupon_code` 可空。

### 10.2 订单列表

`GET /orders?status=&page=1&page_size=10`（status 可空；0=待支付 1=已完成 2=已取消 3=已退款）

```json
{ "code": 0, "message": "ok",
  "data": { "total": 1, "page": 1, "page_size": 10,
    "list": [ { "order_no": "2026062400063525438887716", "plan_name": "猎户座", "period": "month",
                "pay_amount": 1.00, "status": 1, "created_at": "2026-06-24T00:53:35+08:00" } ] } }
```

### 10.3 订单详情（兼支付轮询）

`GET /orders/{order_no}`

```json
{ "code": 0, "message": "ok",
  "data": { "order_no": "2026...", "plan_name": "猎户座", "period": "month",
            "amount": 1.00, "discount_amount": 0.00, "balance_used": 0.00, "pay_amount": 1.00,
            "coupon_code": null, "status": 1, "pay_method": "epay_alipay",
            "paid_at": "2026-06-24T00:55:10+08:00", "created_at": "2026-06-24T00:53:35+08:00" } }
```

前端支付轮询每 3s 调本接口，`status=1` 即成功；仅返回本人订单（否则 40400）。

### 10.4 取消订单

`POST /orders/{order_no}/cancel`；仅 status=0 可取消，成功返回更新后订单。错误 11003。

### 10.5 收银台（拉起支付）

`POST /orders/{order_no}/checkout`

```json
// 请求
{ "method": "epay_alipay" }
// 响应（跳转型）
{ "code": 0, "message": "ok", "data": { "type": "url", "content": "https://pay.example.com/submit/xxx", "expire_in": 1800 } }
// 响应（二维码型）
{ "code": 0, "message": "ok", "data": { "type": "qrcode", "content": "alipays://platformapi/startapp?...", "expire_in": 1800 } }
// method=balance（余额支付成功，直接完成）
{ "code": 0, "message": "ok", "data": { "type": "paid", "content": null, "expire_in": 0 } }
```

错误：11003 订单状态不允许 / 11004 余额不足 / 11005 渠道不可用。30 分钟内对同一订单重复 checkout 返回原支付单（服务端去重）。

### 10.6 支付异步通知（服务端间，免鉴权）

`POST /payment/notify/{method}`（如 `epay_alipay`）。报文格式随支付驱动（易支付为 form-urlencoded）；服务端验签 + 金额比对 + 幂等处理后按驱动要求响应（易支付返回纯文本 `success`）。**前端不调用本接口**。

---

## 11. 邀请与佣金

### 11.1 邀请总览

`GET /invite/summary`

```json
{ "code": 0, "message": "ok",
  "data": { "commission_balance": 0.00, "commission_rate": 40, "registered_count": 0,
            "total_commission": 0.00, "pending_commission": 0.00 } }
```

对应截图 5 张统计卡：我的佣金 / 佣金比例 / 已注册用户数 / 累计获得佣金 / 确认中的佣金。

### 11.2 邀请码列表 / 新增

`GET /invite/codes`

```json
{ "code": 0, "message": "ok",
  "data": { "list": [ { "code": "AB12CD34", "used_count": 3, "created_at": "2026-06-01T10:00:00+08:00" } ],
            "limit": 5, "register_url_prefix": "https://panel.example.com/register?code=" } }
```

`POST /invite/codes`（无 body）→ 返回新码对象；超限错误 13001。

`DELETE /invite/codes/:code` → 删除当前用户自己的邀请码；不存在错误 40400。

### 11.3 佣金发放记录

`GET /invite/records?page=1&page_size=10`

```json
{ "code": 0, "message": "ok",
  "data": { "total": 2, "page": 1, "page_size": 10,
    "list": [ { "order_no": "2026...", "amount": 4.00, "rate": 40, "status": 1,
                "confirmed_at": "2026-06-28T02:00:00+08:00", "created_at": "2026-06-24T00:56:00+08:00" } ] } }
```

status：0=确认中 1=已发放 2=已撤销；列表页只展示已发放（status=1），确认中计入 summary。

### 11.4 佣金划转余额

`POST /invite/transfer`

请求：`{ "amount": 20.00 }`；响应：`{ "balance": 20.00, "commission_balance": 0.00 }`；错误 13002 余额不足。

---

## 12. 代理商

### 12.1 代理状态

`GET /agent/status`

```json
{ "code": 0, "message": "ok",
  "data": { "is_agent": false, "apply_status": "none",
            "qualified": false, "valid_invites": 0, "required_valid_invites": 50,
            "conditions": [ { "met": true, "text": "邀请有效用户：≥ 50 人，且没有过被禁封记录。" },
                            { "met": false, "text": "当前有效人数：已邀请 0 人，还需邀请 50 人。" } ] } }
```

`apply_status`：`none / pending / approved / rejected`。

### 12.2 提交申请

`POST /agent/apply`（无 body）。错误：15001 条件不满足 / 15002 审核中重复提交。响应：`{ "apply_status": "pending" }`。

---

## 13. 工单

### 13.1 列表 / 创建

`GET /tickets?page=1&page_size=10`

```json
{ "code": 0, "message": "ok",
  "data": { "total": 1, "page": 1, "page_size": 10,
    "list": [ { "id": 7, "subject": "无法连接节点", "level": 1, "status": 1,
                "last_reply_at": "2026-07-01T12:00:00+08:00", "created_at": "2026-06-30T09:00:00+08:00" } ] } }
```

status：0=待回复 1=已回复 2=已关闭；level：0=低 1=中 2=高。

`POST /tickets`

```json
// 请求
{ "subject": "无法连接节点", "level": 1, "message": "详细描述…" }
// 响应 data：创建后的工单对象（同列表项）
```

### 13.2 详情 / 回复 / 关闭

`GET /tickets/{id}`

```json
{ "code": 0, "message": "ok",
  "data": { "id": 7, "subject": "无法连接节点", "level": 1, "status": 1, "created_at": "…",
            "messages": [ { "id": 1, "sender_type": 0, "message": "详细描述…", "created_at": "…" },
                          { "id": 2, "sender_type": 1, "message": "请尝试切换节点…", "created_at": "…" } ] } }
```

`POST /tickets/{id}/reply`：`{ "message": "…" }`；已关闭返回 14001。回复后状态：用户回复→0，客服回复→1。
`POST /tickets/{id}/close`：返回更新后工单。

---

## 14. 节点状态

`GET /servers`

```json
{ "code": 0, "message": "ok",
  "data": { "groups": [
    { "group": "香港",
      "servers": [ { "id": 11, "name": "香港 01 | IPLC", "type": "trojan", "rate": 1.0,
                     "status": 1, "tags": ["IPLC", "流媒体"] } ] } ] } }
```

status：1=正常 2=拥挤 3=维护。**不返回** host/port/密码等连接参数；只返回当前用户套餐可见分组。

---

## 15. 订阅下发（代理客户端直连，免登录）

`GET /client/subscribe/{token}?flag=`

| 项 | 说明 |
|---|---|
| `flag` | `clash` / `sing-box` / `v2ray`；缺省按 User-Agent 嗅探，仍不识别人话返回 base64 分享链接 |
| 成功响应 | 对应格式配置正文：clash→`text/yaml`；sing-box→`application/json`；v2ray→`text/plain`（base64） |
| 响应头 | `subscription-userinfo: upload={bytes}; download={bytes}; total={bytes}; expire={unix}`、`profile-update-interval: 24`、`content-disposition: attachment; filename="YLink"` |
| 失败 | token 无效/用户封禁 → 401 纯文本；未购套餐 → 返回仅含提示节点的配置 |

本接口不走 envelope 格式；独立限流（如 10 次/分钟/token）。

---

## 16. 管理端 API 附录（`/api/v1/admin`，role=admin）

用户端 App 不调用；供管理后台使用（前端 13 模块全部实现：M8 核心 6 模块 总览/用户/套餐/节点/订单/工单 + M9 二期 7 模块 优惠券/公告/知识库/代理审批/佣金日志/流量导入/站点设置）。响应走统一信封。

| 模块 | 端点 |
|---|---|
| 仪表盘 | `GET /admin/stat/overview`（用户/订单/收入/在售套餐统计） |
| 用户 | `GET /admin/users`、`PUT /admin/users/{id}`（封禁/角色）、`POST /admin/users/{id}/balance`（调余额，审计） |
| 套餐 | `GET/POST/PUT/DELETE /admin/plans` |
| 节点 | `GET/POST/PUT/DELETE /admin/servers`、`/admin/server-groups` |
| 订单 | `GET /admin/orders`、`POST /admin/orders/{no}/refund`（审计 + 佣金回滚）、`POST /admin/orders/{no}/close`（关闭待支付订单，审计） |
| 优惠券 | `GET/POST/PUT/DELETE /admin/coupons` |
| 内容 | `GET/POST/PUT/DELETE /admin/notices`、`/admin/knowledges` |
| 工单 | `GET /admin/tickets`、`GET /admin/tickets/{id}`、`POST /admin/tickets/{id}/reply`、`/close` |
| 代理 | `GET /admin/agent/applies`、`POST /admin/agent/applies/{id}/approve|reject` |
| 佣金 | `GET /admin/commission-logs` |
| 流量 | `POST /admin/traffic/import`（一期模式 B 手工导入） |
| 配置 | `GET/PUT /admin/settings` |

### 16.1 管理端响应字段约定（2026-08-10 细化）

- `GET /admin/plans` 返回 `{list: AdminPlanView[]}`：价格字段（`month_price`/`quarter_price`/`half_year_price`/`year_price`/`onetime_price`）**单位为元**（`null` 表示未开放该周期），并展开 `group_ids: number[]`、`is_show`、`sort`、`traffic_gb`、`speed_limit`、`device_limit`。请求体 `AdminPlanReq` 同字段，价格传元。
- `GET /admin/servers` 返回 `{list: AdminServerView[]}`：展开用户端隐藏的 `group_id`/`host`/`port`/`config`/`is_show`/`sort`，`tags: string[]`。请求体 `AdminServerReq` 中 `type ∈ {shadowsocks,vmess,vless,trojan,hysteria2,tuic}`、`status ∈ {1=正常,2=拥挤,3=维护}`、`config` 为协议私有参数 JSON 字符串。
- `GET /admin/users` 分页返回 `{list,total,page,page_size}`，`balance`/`commission_balance` 单位为元；`PUT /admin/users/{id}` 请求体 `{role?, banned?}`（`role ∈ {0,1,2}`）；`POST /admin/users/{id}/balance` 请求体 `{amount(元，可正可负), remark?}`。
- `GET /admin/orders` 分页返回 `{list,total,page,page_size}`，`status ∈ {0=待支付,1=已完成,2=已取消,3=已退款}`，金额单位为元。
- `GET /admin/coupons` 返回 `{list: AdminCouponView[]}`：展开 `type ∈ {1=固定金额,2=百分比}`、`value`（type=1 为元、type=2 为百分比数值，如 10 表示 10%）、`min_spend` 单位为元、`limit_per_user`/`total_limit`/`used_count`、`valid_periods: string[]`（仅限可用周期）、`plan_ids: number[]`（仅限可用套餐，空=全部）、`started_at`/`ended_at`（null=不限）、`is_enable`、`created_at`。请求体 `AdminCouponReq` 同字段（`valid_periods`/`plan_ids` 传数组）。
- `GET /admin/notices` 返回 `{list: AdminNoticeItem[]}`（含隐藏，倒序）：`id/title/content/is_show/sort/created_at`。请求体 `AdminNoticeReq`：`title/content` 必填、`is_show?`、`sort?`。
- `GET /admin/knowledges` 返回 `{list: AdminKnowledgeItem[]}`（含隐藏）：`id/category/title/body/language(zh-CN|en-US)/is_show/sort/updated_at`。请求体 `AdminKnowledgeReq`：`category/title/body` 必填、`language?`（缺省继承原语言，新建默认 zh-CN）、`is_show?`、`sort?`。
- `GET /admin/agent/applies` 分页返回 `{list,total,page,page_size}`，`status ∈ {0=待审核,1=通过,2=拒绝}`（`-1` 或缺省=全部）；列表项含 `user_email`/`valid_invites`。`POST /admin/agent/applies/{id}/approve|reject` 请求体 `{remark?}`，仅待审核可审（否则 409）。
- `GET /admin/commission-logs` 分页返回 `{list,total,page,page_size}`，`status ∈ {0=确认中,1=已发放,2=已撤销}`；列表项含 `invite_email`/`from_email`/`order_no`/`order_amount`/`rate`/`amount`（元）/`confirmed_at`/`created_at`。
- `POST /admin/traffic/import` 请求体 `{items: [{user_id, date(YYYY-MM-DD), u, d}]}`（至少 1 项，流量单位为字节），成功后写审计。
- `GET /admin/settings` 返回 `{list: [{key, value}]}`，`value` 为配置项 JSON 字符串（`site`/`payment`/`invite`/`agent`/`order`/`templates`）；`PUT /admin/settings` 请求体 `{key, value}`（单 key 保存，写后失效配置缓存）。

---

## 17. 契约变更管理

1. 任何端点新增/字段变更先提本文档 PR，标注 `Added/Changed/Deprecated`，双方评审通过后实现。
2. 破坏性变更（删字段、改语义）走版本化：并行暴露 `/api/v2/...`，v1 保留至少一个版本周期。
3. 后端 Swagger 注解与本文档同步更新；CI 校验 Swagger 可构建。前端 `types/api.d.ts` 与本文档同步更新。
