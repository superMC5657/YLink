import type { MockMethod } from 'vite-plugin-mock'
import { verifyAdmin } from './auth'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

// 管理端 Mock(契约 docs/api/README.md §16);vite-plugin-mock 按文件独立编译,
// 模块级状态不跨文件共享,故数据为纯静态。
const overview = {
  user_count: 128,
  agent_count: 6,
  order_count: 342,
  completed_orders: 210,
  total_revenue: 8642.5,
  today_revenue: 128.6,
  plan_count: 4,
}

const users = [
  {
    id: 10086,
    email: '2734921923@qq.com',
    role: 0,
    balance: 198.5,
    commission_balance: 42.8,
    is_banned: false,
    invite_by_id: null,
    plan_id: 1,
    expired_at: '2026-09-01T00:00:00+08:00',
    transfer_enable: 107374182400,
    u: 21474836480,
    d: 10737418240,
    created_at: '2026-05-20T10:00:00+08:00',
  },
  {
    id: 1,
    email: 'admin@example.com',
    role: 1,
    balance: 0,
    commission_balance: 0,
    is_banned: false,
    invite_by_id: null,
    plan_id: null,
    expired_at: null,
    transfer_enable: 0,
    u: 0,
    d: 0,
    created_at: '2026-05-01T08:00:00+08:00',
  },
  {
    id: 2,
    email: 'agent@example.com',
    role: 2,
    balance: 320,
    commission_balance: 96.5,
    is_banned: false,
    invite_by_id: null,
    plan_id: 1,
    expired_at: '2026-12-31T00:00:00+08:00',
    transfer_enable: 536870912000,
    u: 107374182400,
    d: 53687091200,
    created_at: '2026-04-11T09:30:00+08:00',
  },
]

const plans = [
  {
    id: 1,
    name: '白羊座',
    content: '**入门之选** · 50GB 流量',
    month_price: 12,
    quarter_price: 32.4,
    half_year_price: 60,
    year_price: 110.4,
    onetime_price: null,
    traffic_gb: 50,
    speed_limit: 100,
    device_limit: 3,
    group_ids: [1, 2],
    is_show: true,
    sort: 1,
  },
  {
    id: 2,
    name: '金牛座',
    content: '**进阶之选** · 200GB 流量',
    month_price: 25,
    quarter_price: 67.5,
    half_year_price: 125,
    year_price: 230,
    onetime_price: null,
    traffic_gb: 200,
    speed_limit: null,
    device_limit: 5,
    group_ids: [1, 2, 3],
    is_show: true,
    sort: 2,
  },
  {
    id: 3,
    name: '隐藏套餐',
    content: '下架中的套餐',
    month_price: 99,
    quarter_price: null,
    half_year_price: null,
    year_price: null,
    onetime_price: null,
    traffic_gb: 500,
    speed_limit: null,
    device_limit: 10,
    group_ids: [],
    is_show: false,
    sort: 9,
  },
]

const serverGroups = [
  { id: 1, name: '香港', sort: 1 },
  { id: 2, name: '美国', sort: 2 },
  { id: 3, name: '日本', sort: 3 },
]

// ---------- F16 重置记录演示数据 ----------
const trafficResets = [
  {
    id: 1,
    user_id: 10086,
    user_email: '2734921923@qq.com',
    mode: 'clear_usage',
    before_u: 21474836480,
    before_d: 10737418240,
    before_transfer_enable: 107374182400,
    after_transfer_enable: 107374182400,
    created_at: '2026-08-28T10:00:00+08:00',
  },
  {
    id: 2,
    user_id: 2,
    user_email: 'agent@example.com',
    mode: 'reset_quota',
    before_u: 107374182400,
    before_d: 53687091200,
    before_transfer_enable: 536870912000,
    after_transfer_enable: 536870912000,
    created_at: '2026-08-27T15:30:00+08:00',
  },
]

// ---------- F08 审计日志演示数据(action/target 对齐后端真实写入格式,含 target 可读化字段) ----------
const auditLogs = [
  {
    id: 6,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'traffic_reset',
    target: '[10086 10087]',
    target_kind: 'users',
    target_display: 'alice@example.com, bob@example.com',
    detail: '{"mode":"clear_usage","success":2,"failed":0}',
    ip: '192.168.1.20',
    created_at: '2026-08-28T10:05:00+08:00',
  },
  {
    id: 5,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'adjust_balance',
    target: '10086',
    target_kind: 'user',
    target_display: 'alice@example.com',
    detail: '{"email":"alice@example.com","amount":1000,"remark":"活动补偿"}',
    ip: '192.168.1.20',
    created_at: '2026-08-27T18:40:00+08:00',
  },
  {
    id: 4,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'ban_user',
    target: '10087',
    target_kind: 'user',
    target_display: 'bob@example.com',
    detail: '{"email":"bob@example.com","banned":true}',
    ip: '10.0.0.8',
    created_at: '2026-08-26T09:12:00+08:00',
  },
  {
    id: 3,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'refund',
    target: 'ORD20260825123456789012',
    target_kind: 'order',
    target_display: 'ORD20260825123456789012',
    detail: '{"amount":15.00}',
    ip: '10.0.0.8',
    created_at: '2026-08-25T15:30:00+08:00',
  },
  {
    id: 2,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'agent_approve',
    target: '10088',
    target_kind: 'user',
    target_display: 'carol@example.com',
    detail: '{"apply_id":3}',
    ip: '192.168.1.20',
    created_at: '2026-08-21T11:00:00+08:00',
  },
  {
    id: 1,
    admin_id: 1,
    admin_email: 'admin@example.com',
    action: 'traffic_import',
    target: '',
    target_kind: null,
    target_display: null,
    detail: '{"count":3,"date":"2026-08-20"}',
    ip: '192.168.1.20',
    created_at: '2026-08-20T08:00:00+08:00',
  },
]

// ---------- F04 报表演示数据(近 30 天逐日) ----------
function lastDays(n: number): string[] {
  const out: string[] = []
  const base = new Date('2026-08-28T00:00:00+08:00')
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(base.getTime() - i * 86400000)
    out.push(d.toISOString().slice(0, 10))
  }
  return out
}

const statOrders = {
  days: 30,
  items: lastDays(30).map((date, i) => ({
    date,
    order_count: 3 + ((i * 7) % 11),
    completed_count: 2 + ((i * 5) % 9),
    revenue: Math.round((80 + ((i * 137) % 260)) * 100) / 100,
    refunded: i % 9 === 0 ? 12 : 0,
    balance_used: Math.round((20 + ((i * 53) % 90)) * 100) / 100,
    balance_refunded: i % 9 === 0 ? 8 : 0,
  })),
}

const statUsers = {
  days: 30,
  register_trend: lastDays(30).map((date, i) => ({
    date,
    count: (i * 3) % 8,
  })),
  plan_distribution: [
    { plan_id: 1, plan_name: '白羊座', users: 86 },
    { plan_id: 2, plan_name: '金牛座', users: 54 },
    { plan_id: 3, plan_name: '射手座', users: 23 },
  ],
}

const statTraffic = {
  days: 30,
  user_top: [
    { user_id: 2, email: 'agent@example.com', total_bytes: 536870912000 },
    { user_id: 10086, email: '2734921923@qq.com', total_bytes: 214748364800 },
    { user_id: 10087, email: 'user3@example.com', total_bytes: 107374182400 },
    { user_id: 10088, email: 'user6@example.com', total_bytes: 53687091200 },
    { user_id: 10089, email: 'user7@example.com', total_bytes: 32212254720 },
  ],
  node_top: [
    { server_id: 1, name: '香港 01', bytes: 640000000000 },
    { server_id: 2, name: '美国 01', bytes: 320000000000 },
  ],
}

const servers = [
  {
    id: 1,
    group_id: 1,
    name: '香港 01',
    type: 'shadowsocks',
    host: 'hk1.example.com',
    port: 443,
    config: '{"password":"test","method":"aes-256-gcm"}',
    rate: 1,
    tags: ['旗舰'],
    status: 1,
    is_show: true,
    sort: 1,
    node_key: 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6',
  },
  {
    id: 2,
    group_id: 2,
    name: '美国 01',
    type: 'vless',
    host: 'us1.example.com',
    port: 8443,
    config: '{"uuid":"test-uuid","network":"tcp"}',
    rate: 1.5,
    tags: null,
    status: 2,
    is_show: true,
    sort: 2,
    node_key: '0f1e2d3c4b5a69788796a5b4c3d2e1f0',
  },
]

const orders = [
  {
    order_no: '20260628000000000000021',
    user_id: 10086,
    user_email: '2734921923@qq.com',
    plan_name: '白羊座',
    period: 'month',
    amount: 12,
    discount_amount: 0,
    balance_used: 12,
    pay_amount: 0,
    commission_amount: null, // 余额支付不产生佣金
    status: 1,
    pay_method: 'balance',
    paid_at: '2026-06-28T00:56:00+08:00',
    created_at: '2026-06-28T00:55:00+08:00',
  },
  {
    order_no: '20260701000000000000022',
    user_id: 10086,
    user_email: '2734921923@qq.com',
    plan_name: '金牛座',
    period: 'year',
    amount: 230,
    discount_amount: 20,
    balance_used: 210,
    pay_amount: 0,
    commission_amount: null, // 余额支付不产生佣金
    status: 3,
    pay_method: 'balance',
    paid_at: '2026-07-01T12:00:00+08:00',
    created_at: '2026-07-01T11:58:00+08:00',
  },
  {
    order_no: '20260715000000000000023',
    user_id: 10087,
    user_email: 'user3@example.com',
    plan_name: '射手座',
    period: 'quarter',
    amount: 54,
    discount_amount: 0,
    balance_used: 0,
    pay_amount: 54,
    commission_amount: 21.6, // 在线支付产生佣金(40%)
    status: 1,
    pay_method: 'epay_alipay',
    paid_at: '2026-07-15T18:30:00+08:00',
    created_at: '2026-07-15T18:28:00+08:00',
  },
  {
    order_no: '20260720000000000000024',
    user_id: 10088,
    user_email: 'user6@example.com',
    plan_name: '白羊座',
    period: 'year',
    amount: 96,
    discount_amount: 0,
    balance_used: 0,
    pay_amount: 96,
    commission_amount: 48, // 在线支付产生佣金(50% 代理)
    status: 1,
    pay_method: 'epay_wxpay',
    paid_at: '2026-07-20T09:10:00+08:00',
    created_at: '2026-07-20T09:08:00+08:00',
  },
]

const tickets = [
  {
    id: 1,
    user_id: 10086,
    user_email: '2734921923@qq.com',
    subject: '无法导入订阅链接',
    level: 1,
    status: 0,
    last_reply_at: '2026-07-10T09:30:00+08:00',
    created_at: '2026-07-10T09:28:00+08:00',
  },
  {
    id: 2,
    user_id: 10086,
    user_email: '2734921923@qq.com',
    subject: '流量统计不准',
    level: 0,
    status: 1,
    last_reply_at: '2026-07-12T14:00:00+08:00',
    created_at: '2026-07-12T13:50:00+08:00',
  },
]

const ticketDetail = {
  id: 1,
  subject: '无法导入订阅链接',
  level: 1,
  status: 0,
  created_at: '2026-07-10T09:28:00+08:00',
  messages: [
    {
      id: 1,
      sender_type: 0,
      message: '我在 Clash 里导入订阅链接一直失败,提示格式错误,麻烦看一下。',
      created_at: '2026-07-10T09:28:00+08:00',
    },
    {
      id: 2,
      sender_type: 1,
      message: '您好,请确认复制的是完整链接(以 https:// 开头),或提供报错截图。',
      created_at: '2026-07-10T09:30:00+08:00',
    },
  ],
}

// ---------- 二期模块(契约 docs/api/README.md §16.1) ----------
const coupons = [
  {
    id: 1,
    code: 'WELCOME10',
    type: 2,
    value: 10,
    min_spend: 0,
    limit_per_user: 1,
    total_limit: 100,
    used_count: 23,
    valid_periods: ['month', 'quarter', 'half_year', 'year'],
    plan_ids: [],
    started_at: null,
    ended_at: null,
    is_enable: true,
    created_at: '2026-06-01T10:00:00+08:00',
  },
  {
    id: 2,
    code: 'VIP50',
    type: 1,
    value: 50,
    min_spend: 200,
    limit_per_user: 0,
    total_limit: 0,
    used_count: 5,
    valid_periods: ['year'],
    plan_ids: [2],
    started_at: '2026-07-01T00:00:00+08:00',
    ended_at: '2026-12-31T23:59:59+08:00',
    is_enable: true,
    created_at: '2026-07-01T09:00:00+08:00',
  },
]

// ---------- 知识库 ----------
const knowledges = [
  {
    id: 1,
    category: '入门指南',
    title: '如何导入订阅链接',
    body: '复制订阅链接后,在客户端中选择「从 URL 导入」并粘贴即可。',
    language: 'zh-CN',
    is_show: true,
    sort: 1,
    updated_at: '2026-06-15T10:00:00+08:00',
  },
  {
    id: 2,
    category: '常见问题',
    title: '连接不上节点怎么办',
    body: '1. 检查是否已到期或流量用尽;2. 尝试切换其他节点;3. 联系客服。',
    language: 'zh-CN',
    is_show: true,
    sort: 2,
    updated_at: '2026-06-20T14:00:00+08:00',
  },
  {
    id: 3,
    category: 'Getting Started',
    title: 'How to import subscription',
    body: 'Copy the subscription URL and paste it into your client via "Import from URL".',
    language: 'en-US',
    is_show: true,
    sort: 1,
    updated_at: '2026-06-15T10:00:00+08:00',
  },
]

const agentApplies = [
  {
    id: 1,
    user_id: 10086,
    user_email: '2734921923@qq.com',
    valid_invites: 52,
    status: 0,
    created_at: '2026-07-10T09:30:00+08:00',
  },
  {
    id: 2,
    user_id: 2,
    user_email: 'agent@example.com',
    valid_invites: 60,
    status: 1,
    created_at: '2026-06-01T09:00:00+08:00',
  },
  {
    id: 3,
    user_id: 9,
    user_email: 'rejected@example.com',
    valid_invites: 3,
    status: 2,
    created_at: '2026-05-01T09:00:00+08:00',
  },
]

const commissionLogs = [
  {
    id: 1,
    invite_user_id: 10086,
    invite_email: '2734921923@qq.com',
    from_user_id: 3,
    from_email: 'user3@example.com',
    order_no: '20260715000000000000023',
    order_amount: 54,
    rate: 40,
    amount: 21.6,
    status: 1,
    confirmed_at: '2026-07-16T02:00:00+08:00',
    created_at: '2026-07-15T18:30:00+08:00',
  },
  {
    id: 2,
    invite_user_id: 2,
    invite_email: 'agent@example.com',
    from_user_id: 6,
    from_email: 'user6@example.com',
    order_no: '20260601000000000000001',
    order_amount: 230,
    rate: 50,
    amount: 115,
    status: 0,
    confirmed_at: null,
    created_at: '2026-06-01T10:00:00+08:00',
  },
  {
    id: 3,
    invite_user_id: 10086,
    invite_email: '2734921923@qq.com',
    from_user_id: 4,
    from_email: 'user4@example.com',
    order_no: '20260615000000000000018',
    order_amount: 12,
    rate: 40,
    amount: 4.8,
    status: 2,
    confirmed_at: null,
    created_at: '2026-06-15T11:00:00+08:00',
  },
]

const settings = [
  {
    key: 'site',
    value: JSON.stringify({
      site_name: 'YLink',
      site_logo: '',
      site_description: '高速稳定的网络加速服务',
      register_enabled: true,
      invite_code_required: false,
      app_downloads: {},
      telegram: {},
      customer_service_url: '',
      free_traffic_tips: '绑定 TG 机器人每天领取免费流量',
      languages: ['zh-CN', 'en-US'],
    }),
  },
  {
    key: 'payment',
    value: JSON.stringify({
      methods: [
        { code: 'balance', name: '余额支付', icon: 'wallet', enabled: true },
        { code: 'epay_alipay', name: '支付宝', icon: 'alipay', enabled: true },
        { code: 'epay_wxpay', name: '微信支付', icon: 'wechat', enabled: true },
      ],
    }),
  },
  {
    key: 'invite',
    value: JSON.stringify({
      commission_rate: 40,
      agent_commission_rate: 50,
      commission_confirm_days: 3,
      invite_code_limit: 5,
    }),
  },
  {
    key: 'agent',
    value: JSON.stringify({
      required_valid_invites: 50,
      audit_months: 12,
      benefits: ['佣金比例：40%（循环）', '套餐福利：赠送免费的年付订阅套餐'],
      notes: ['点击按钮申请代理权限，审核通过后将获得以上特权。'],
    }),
  },
  {
    key: 'order',
    value: JSON.stringify({ expire_minutes: 30 }),
  },
  {
    key: 'templates',
    value: JSON.stringify({
      captcha: '您的验证码是 <b>{code}</b>，10 分钟内有效。',
      welcome: '欢迎注册 {site_name}！',
      expire: '您的订阅将于 {expired_at} 到期，请及时续费。',
      traffic: '您的流量已使用 {percent}%，请注意剩余流量。',
    }),
  },
]

function paged<T>(list: T[], page: number, pageSize: number) {
  const start = (page - 1) * pageSize
  return {
    list: list.slice(start, start + pageSize),
    total: list.length,
    page,
    page_size: pageSize,
  }
}

export default [
  {
    url: '/api/v1/admin/stat/overview',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(overview)
    },
  },
  {
    // F08 审计日志:筛选语义对齐后端(admin_id/action/target 精确,from<=created_at<to)
    url: '/api/v1/admin/audit-logs',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 20)
      const adminId = query.admin_id
      const action = query.action ?? ''
      const target = query.target ?? ''
      const from = query.from ?? ''
      const to = query.to ?? ''
      const filtered = auditLogs.filter((l) => {
        if (adminId !== undefined && adminId !== '' && String(l.admin_id) !== adminId) return false
        if (action && l.action !== action) return false
        if (target && l.target !== target) return false
        const day = l.created_at.slice(0, 10)
        if (from && day < from) return false
        if (to && day >= to) return false
        return true
      })
      const actions = [...new Set(auditLogs.map((l) => l.action))].sort()
      return ok({ ...paged(filtered, page, pageSize), actions })
    },
  },
  {
    url: '/api/v1/admin/users',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      const kw = (query.keyword ?? '').toLowerCase()
      const filtered = kw ? users.filter((u) => u.email.toLowerCase().includes(kw)) : users
      return ok(paged(filtered, page, pageSize))
    },
  },
  {
    url: '/api/v1/admin/plans',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: plans })
    },
  },
  {
    url: '/api/v1/admin/server-groups',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: serverGroups })
    },
  },
  {
    url: '/api/v1/admin/servers',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: servers })
    },
  },
  // F09 批量 / 复制 / 排序
  {
    url: '/api/v1/admin/servers/batch',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { action?: string; ids?: number[] }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const ids = body.ids ?? []
      if (!body.action || ids.length === 0) {
        return { code: 40000, message: '参数校验失败: action/ids 必填', data: null }
      }
      if (body.action === 'delete') {
        for (const id of ids) {
          const idx = servers.findIndex((s) => s.id === id)
          if (idx >= 0) servers.splice(idx, 1)
        }
      } else {
        for (const id of ids) {
          const s = servers.find((x) => x.id === id)
          if (s) Object.assign(s, body)
        }
      }
      return ok({ success: ids.length, failed: [] })
    },
  },
  {
    url: '/api/v1/admin/servers/sort',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { items?: { id: number; sort: number }[] }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      for (const it of body.items ?? []) {
        const s = servers.find((x) => x.id === it.id)
        if (s) s.sort = it.sort
      }
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/servers/:id/copy',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const src = servers.find((s) => s.id === Number(query.id))
      if (!src) return { code: 40400, message: '节点不存在', data: null }
      const arr = new Uint8Array(16)
      crypto.getRandomValues(arr)
      const copy = {
        ...src,
        id: Math.max(...servers.map((s) => s.id)) + 1,
        name: `${src.name}-copy`,
        node_key: Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join(''),
      }
      servers.push(copy)
      return ok(copy)
    },
  },
  {
    url: '/api/v1/admin/servers/:id/node-key/reset',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const srv = servers.find((s) => s.id === Number(query.id))
      if (!srv) return { code: 40400, message: '节点不存在', data: null }
      const arr = new Uint8Array(16)
      crypto.getRandomValues(arr)
      srv.node_key = Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('')
      return ok({ node_key: srv.node_key })
    },
  },
  {
    url: '/api/v1/admin/orders',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      const status = query.status
      const filtered =
        status === undefined || status === ''
          ? orders
          : orders.filter((o) => String(o.status) === status)
      return ok(paged(filtered, page, pageSize))
    },
  },
  {
    url: '/api/v1/admin/orders/:order_no/refund',
    method: 'post',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { order_no?: string }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const order = orders.find((o) => o.order_no === query.order_no)
      if (!order) return { code: 40400, message: '订单不存在', data: null }
      if (order.status !== 1) return { code: 40000, message: '仅已完成订单可退款', data: null }
      order.status = 3
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/orders/:order_no/close',
    method: 'post',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { order_no?: string }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const order = orders.find((o) => o.order_no === query.order_no)
      if (!order) return { code: 40400, message: '订单不存在', data: null }
      if (order.status !== 0) return { code: 40000, message: '仅待支付订单可关闭', data: null }
      order.status = 2
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/tickets',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      return ok(paged(tickets, page, pageSize))
    },
  },
  {
    url: '/api/v1/admin/tickets/:id',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(ticketDetail)
    },
  },
  {
    url: '/api/v1/admin/tickets/:id/reply',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/tickets/:id/close',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },

  // ---------- 优惠券 ----------
  {
    url: '/api/v1/admin/coupons',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: coupons })
    },
  },
  {
    url: '/api/v1/admin/coupons',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: Record<string, unknown>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      if (!body.code || typeof body.value !== 'number' || body.value <= 0) {
        return { code: 40000, message: '参数校验失败: code 必填且 value 须为正数', data: null }
      }
      const created = {
        id: Date.now(),
        code: body.code,
        type: body.type ?? 2,
        value: body.value,
        min_spend: body.min_spend ?? 0,
        limit_per_user: body.limit_per_user ?? 0,
        total_limit: body.total_limit ?? 0,
        used_count: 0,
        valid_periods: body.valid_periods ?? [],
        plan_ids: body.plan_ids ?? [],
        started_at: body.started_at ?? null,
        ended_at: body.ended_at ?? null,
        is_enable: body.is_enable ?? true,
        created_at: '2026-08-11T10:00:00+08:00',
      }
      coupons.unshift(created)
      return ok(created)
    },
  },
  {
    url: '/api/v1/admin/coupons/:id',
    method: 'put',
    response: ({
      headers,
      query,
      body,
    }: {
      headers: Record<string, string>
      query: { id?: string }
      body: Record<string, unknown>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const c = coupons.find((x) => String(x.id) === query.id)
      if (!c) return { code: 40400, message: '优惠券不存在', data: null }
      if (!body.code || typeof body.value !== 'number' || body.value <= 0) {
        return { code: 40000, message: '参数校验失败: code 必填且 value 须为正数', data: null }
      }
      Object.assign(c, {
        code: body.code ?? c.code,
        type: body.type ?? c.type,
        value: body.value ?? c.value,
        min_spend: body.min_spend ?? c.min_spend,
        limit_per_user: body.limit_per_user ?? c.limit_per_user,
        total_limit: body.total_limit ?? c.total_limit,
        valid_periods: body.valid_periods ?? c.valid_periods,
        plan_ids: body.plan_ids ?? c.plan_ids,
        started_at: body.started_at ?? c.started_at,
        ended_at: body.ended_at ?? c.ended_at,
        is_enable: body.is_enable ?? c.is_enable,
      })
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/coupons/:id',
    method: 'delete',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const idx = coupons.findIndex((x) => String(x.id) === query.id)
      if (idx < 0) return { code: 40400, message: '优惠券不存在', data: null }
      coupons.splice(idx, 1)
      return ok(null)
    },
  },

  // ---------- 知识库 ----------
  {
    url: '/api/v1/admin/knowledges',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: knowledges })
    },
  },
  {
    url: '/api/v1/admin/knowledges',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: {
        category?: string
        title?: string
        body?: string
        language?: string
        is_show?: boolean
        sort?: number
      }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      if (!body.category || !body.title || !body.body) {
        return { code: 40000, message: '参数校验失败: category/title/body 必填', data: null }
      }
      const created = {
        id: Date.now(),
        category: body.category,
        title: body.title,
        body: body.body,
        language: body.language ?? 'zh-CN',
        is_show: body.is_show ?? true,
        sort: body.sort ?? 0,
        updated_at: '2026-08-11T10:00:00+08:00',
      }
      knowledges.unshift(created)
      return ok(created)
    },
  },
  {
    url: '/api/v1/admin/knowledges/:id',
    method: 'put',
    response: ({
      headers,
      query,
      body,
    }: {
      headers: Record<string, string>
      query: { id?: string }
      body: {
        category?: string
        title?: string
        body?: string
        language?: string
        is_show?: boolean
        sort?: number
      }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const k = knowledges.find((x) => String(x.id) === query.id)
      if (!k) return { code: 40400, message: '知识库条目不存在', data: null }
      Object.assign(k, {
        category: body.category ?? k.category,
        title: body.title ?? k.title,
        body: body.body ?? k.body,
        language: body.language ?? k.language,
        is_show: body.is_show ?? k.is_show,
        sort: body.sort ?? k.sort,
      })
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/knowledges/:id',
    method: 'delete',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const idx = knowledges.findIndex((x) => String(x.id) === query.id)
      if (idx < 0) return { code: 40400, message: '知识库条目不存在', data: null }
      knowledges.splice(idx, 1)
      return ok(null)
    },
  },

  // ---------- 代理审批 ----------
  {
    url: '/api/v1/admin/agent/applies',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      const status = query.status
      const filtered =
        status === undefined || status === '' || status === '-1'
          ? agentApplies
          : agentApplies.filter((a) => String(a.status) === status)
      return ok(paged(filtered, page, pageSize))
    },
  },
  {
    url: '/api/v1/admin/agent/applies/:id/approve',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const a = agentApplies.find((x) => String(x.id) === query.id)
      if (!a) return { code: 40400, message: '申请不存在', data: null }
      if (a.status !== 0) return { code: 40900, message: '该申请已处理', data: null }
      a.status = 1
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/agent/applies/:id/reject',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const a = agentApplies.find((x) => String(x.id) === query.id)
      if (!a) return { code: 40400, message: '申请不存在', data: null }
      if (a.status !== 0) return { code: 40900, message: '该申请已处理', data: null }
      a.status = 2
      return ok(null)
    },
  },

  // ---------- 佣金日志 ----------
  {
    url: '/api/v1/admin/commission-logs',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      const status = query.status
      const filtered =
        status === undefined || status === ''
          ? commissionLogs
          : commissionLogs.filter((c) => String(c.status) === status)
      return ok(paged(filtered, page, pageSize))
    },
  },

  // ---------- 流量导入 / 重置(F16) / 报表(F04) ----------
  {
    url: '/api/v1/admin/traffic/import',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { items?: unknown[] }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      if (!body.items || body.items.length === 0) {
        return { code: 40000, message: '参数校验失败: items 至少 1 项', data: null }
      }
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/traffic/reset',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { user_ids?: number[]; mode?: string }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const ids = body.user_ids ?? []
      if (ids.length === 0 || !body.mode) {
        return { code: 40000, message: '参数校验失败: user_ids/mode 必填', data: null }
      }
      // 演示:最后一个 ID 记为失败,展示失败明细 UI
      trafficResets.unshift({
        id: trafficResets.length + 1,
        user_id: ids[0],
        user_email: users.find((u) => u.id === ids[0])?.email ?? `#${ids[0]}`,
        mode: body.mode,
        before_u: 21474836480,
        before_d: 10737418240,
        before_transfer_enable: 107374182400,
        after_transfer_enable: body.mode === 'reset_quota' ? 107374182400 : 107374182400,
        created_at: '2026-08-28T10:00:00+08:00',
      })
      return ok({
        success: Math.max(0, ids.length - 1),
        failed: [],
      })
    },
  },
  {
    url: '/api/v1/admin/traffic/resets',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: Record<string, string>
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const pageSize = Number(query.page_size ?? 10)
      const uid = query.user_id
      const filtered =
        uid === undefined || uid === ''
          ? trafficResets
          : trafficResets.filter((l) => String(l.user_id) === uid)
      return ok(paged(filtered, page, pageSize))
    },
  },
  {
    url: '/api/v1/admin/stat/orders',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(statOrders)
    },
  },
  {
    url: '/api/v1/admin/stat/users',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(statUsers)
    },
  },
  {
    url: '/api/v1/admin/stat/traffic',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(statTraffic)
    },
  },

  // ---------- 站点设置 ----------
  {
    url: '/api/v1/admin/settings',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({ list: settings })
    },
  },
  {
    url: '/api/v1/admin/settings',
    method: 'put',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { key?: string; value?: string }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      if (!body.key || typeof body.value !== 'string') {
        return { code: 40000, message: '参数校验失败: key/value 必填', data: null }
      }
      const s = settings.find((x) => x.key === body.key)
      if (s) {
        s.value = body.value
      } else {
        settings.push({ key: body.key as string, value: body.value as string })
      }
      return ok(null)
    },
  },

  // ---------- 佣金提现审核（F02） ----------
  {
    url: '/api/v1/admin/tickets/:id/withdraw/pay',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/tickets/:id/withdraw/reject',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },

  // ---------- 内容排序（F15） ----------
  {
    url: '/api/v1/admin/notices/sort',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/knowledges/sort',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },

  // ---------- 知识库分类（F15） ----------
  {
    url: '/api/v1/admin/knowledge-categories',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({
        list: [
          { id: 1, language: 'zh-CN', name: '入门指南', sort: 0, knowledge_count: 2 },
          { id: 2, language: 'zh-CN', name: '客户端配置', sort: 1, knowledge_count: 3 },
          { id: 3, language: 'en-US', name: 'Getting Started', sort: 0, knowledge_count: 1 },
        ],
      })
    },
  },
  {
    url: '/api/v1/admin/knowledge-categories',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { language?: string; name?: string }
    }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({
        id: Date.now(),
        language: body.language ?? 'zh-CN',
        name: body.name ?? '',
        sort: 0,
      })
    },
  },
  {
    url: '/api/v1/admin/knowledge-categories/:id',
    method: 'put',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/knowledge-categories/:id',
    method: 'delete',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },

  // ---------- 邮件模板（F11） ----------
  {
    url: '/api/v1/admin/mail-templates',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({
        list: [
          {
            name: 'captcha',
            subject: '[{{.site_name}}] 邮箱验证码',
            body: '您的验证码是 <b>{{.code}}</b>，10 分钟内有效。',
            is_custom: false,
            placeholders: ['{{.site_name}}', '{{.code}}'],
            remark: '注册/找回密码验证码邮件',
            updated_at: null,
          },
          {
            name: 'expire_remind',
            subject: '[{{.site_name}}] 订阅到期提醒',
            body: '您的订阅将于 <b>{{.expire_date}}</b> 到期，请及时续费。',
            is_custom: true,
            placeholders: ['{{.site_name}}', '{{.expire_date}}'],
            remark: '到期前 3 天与 1 天各发送一次（每日 10:00）',
            updated_at: '2026-08-28T09:00:00+08:00',
          },
          {
            name: 'traffic_remind',
            subject: '[{{.site_name}}] 流量使用提醒',
            body: '您的流量已使用 <b>{{.percent}}%</b>，请注意剩余流量。',
            is_custom: false,
            placeholders: ['{{.site_name}}', '{{.percent}}'],
            remark: '用量 ≥80% 时发送（每日 10:00）',
            updated_at: null,
          },
        ],
      })
    },
  },
  {
    url: '/api/v1/admin/mail-templates/:name',
    method: 'put',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/mail-templates/:name',
    method: 'delete',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },
  {
    url: '/api/v1/admin/mail-templates/:name/test',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok(null)
    },
  },

  // ---------- 版本检查（F20） ----------
  {
    url: '/api/v1/admin/version',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAdmin(headers)) return unauthorized()
      return ok({
        version: '0.4.1',
        latest: null,
        has_update: null,
        notes: null,
      })
    },
  },
] as MockMethod[]
