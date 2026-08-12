import type { MockMethod } from 'vite-plugin-mock'
import { verifyAccess } from './auth'
import type {
  AgentStatus,
  CommissionRecord,
  InviteCode,
  Ticket,
  TicketDetail,
} from '../src/types/api'
import dayjs from 'dayjs'

function ok(data: unknown) {
  return { code: 0, message: 'ok', data }
}

function unauthorized() {
  return { code: 40100, message: '未登录或 token 失效', data: null }
}

// ---------- 邀请与佣金 ----------

const inviteCodes: InviteCode[] = [
  { code: 'AB12CD34', used_count: 3, created_at: '2026-06-01T10:00:00+08:00' },
  { code: 'EF56GH78', used_count: 8, created_at: '2026-06-15T10:00:00+08:00' },
  { code: 'IJ90KL12', used_count: 1, created_at: '2026-07-01T10:00:00+08:00' },
]

function genCode(): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let s = ''
  for (let i = 0; i < 8; i++) s += chars[Math.floor(Math.random() * chars.length)]
  return s
}

const commissionRecords: CommissionRecord[] = [
  {
    order_no: '20260628000000000000021',
    amount: 4.0,
    rate: 40,
    status: 1,
    confirmed_at: '2026-06-28T02:00:00+08:00',
    created_at: '2026-06-24T00:56:00+08:00',
  },
  {
    order_no: '20260701000000000000022',
    amount: 8.0,
    rate: 40,
    status: 1,
    confirmed_at: '2026-07-01T12:00:00+08:00',
    created_at: '2026-06-30T10:00:00+08:00',
  },
  {
    order_no: '20260705000000000000023',
    amount: 38.4,
    rate: 40,
    status: 1,
    confirmed_at: '2026-07-05T18:00:00+08:00',
    created_at: '2026-07-05T16:00:00+08:00',
  },
  {
    order_no: '20260712000000000000024',
    amount: 6.0,
    rate: 40,
    status: 0,
    confirmed_at: null,
    created_at: '2026-07-12T10:00:00+08:00',
  },
]

// ---------- 代理商 ----------

const agentStatus: AgentStatus = {
  is_agent: true,
  apply_status: 'approved',
  qualified: true,
  valid_invites: 68,
  required_valid_invites: 50,
  conditions: [
    { met: true, text: '邀请有效用户:≥ 50 人,且没有过被禁封记录。' },
    { met: true, text: '当前有效人数:已邀请 68 人,已满足条件。' },
  ],
}

// ---------- 工单 ----------

const ticketMessages: Record<number, TicketDetail['messages']> = {
  7: [
    {
      id: 1,
      sender_type: 0,
      message: '无法连接香港 01 节点,一直超时。',
      created_at: '2026-06-30T09:00:00+08:00',
    },
    {
      id: 2,
      sender_type: 1,
      message: '你好,请尝试切换节点或更新订阅后再试。',
      created_at: '2026-06-30T10:30:00+08:00',
    },
  ],
  8: [
    {
      id: 1,
      sender_type: 0,
      message: '年付套餐可以开发票吗?',
      created_at: '2026-07-05T14:00:00+08:00',
    },
  ],
}

const tickets: Ticket[] = [
  {
    id: 7,
    subject: '无法连接香港节点',
    level: 1,
    status: 1,
    reopen_count: 0,
    last_reply_at: '2026-06-30T10:30:00+08:00',
    created_at: '2026-06-30T09:00:00+08:00',
  },
  {
    id: 8,
    subject: '咨询年付套餐发票事宜',
    level: 0,
    status: 0,
    reopen_count: 0,
    last_reply_at: null,
    created_at: '2026-07-05T14:00:00+08:00',
  },
  {
    id: 6,
    subject: '流量统计不准确',
    level: 2,
    status: 2,
    reopen_count: 0,
    last_reply_at: '2026-06-20T16:00:00+08:00',
    created_at: '2026-06-19T11:00:00+08:00',
  },
]

export default [
  // ---------- 邀请 ----------
  {
    url: '/api/v1/invite/summary',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok({
        commission_balance: 88.6,
        commission_rate: 40,
        registered_count: 12,
        total_commission: 126.4,
        pending_commission: 6.0,
      })
    },
  },
  {
    url: '/api/v1/invite/codes',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok({
        list: inviteCodes,
        limit: 5,
        register_url_prefix: 'https://panel.example.com/register?code=',
      })
    },
  },
  {
    url: '/api/v1/invite/codes',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      if (inviteCodes.length >= 5) {
        return { code: 13001, message: '邀请码数量已达上限', data: null }
      }
      const code: InviteCode = {
        code: genCode(),
        used_count: 0,
        created_at: dayjs().toISOString(),
      }
      inviteCodes.unshift(code)
      return ok(code)
    },
  },
  {
    url: '/api/v1/invite/codes/:code',
    method: 'delete',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { code?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const code = query.code
      const idx = inviteCodes.findIndex((c) => c.code === code)
      if (idx === -1) return { code: 40400, message: '邀请码不存在', data: null }
      inviteCodes.splice(idx, 1)
      return ok(null)
    },
  },
  {
    url: '/api/v1/invite/records',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { page?: string; page_size?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const size = Number(query.page_size ?? 10)
      const paid = commissionRecords.filter((r) => r.status === 1)
      const start = (page - 1) * size
      return ok({
        total: paid.length,
        page,
        page_size: size,
        list: paid.slice(start, start + size),
      })
    },
  },
  {
    url: '/api/v1/invite/transfer',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { amount?: number }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const amount = Number(body?.amount ?? 0)
      if (amount > 88.6 || amount <= 0) {
        return { code: 13002, message: '可划转佣金不足', data: null }
      }
      return ok({ balance: 168.5 + amount, commission_balance: 88.6 - amount })
    },
  },
  // ---------- 代理商 ----------
  {
    url: '/api/v1/agent/status',
    method: 'get',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      return ok(agentStatus)
    },
  },
  {
    url: '/api/v1/agent/apply',
    method: 'post',
    response: ({ headers }: { headers: Record<string, string> }) => {
      if (!verifyAccess(headers)) return unauthorized()
      if (agentStatus.apply_status === 'pending') {
        return { code: 15002, message: '申请审核中,请勿重复提交', data: null }
      }
      agentStatus.apply_status = 'pending'
      agentStatus.is_agent = false
      return ok({ apply_status: 'pending' })
    },
  },
  // ---------- 工单 ----------
  {
    url: '/api/v1/tickets',
    method: 'get',
    response: ({
      headers,
      query,
    }: {
      headers: Record<string, string>
      query: { page?: string; page_size?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const page = Number(query.page ?? 1)
      const size = Number(query.page_size ?? 10)
      const start = (page - 1) * size
      return ok({
        total: tickets.length,
        page,
        page_size: size,
        list: tickets.slice(start, start + size),
      })
    },
  },
  {
    url: '/api/v1/tickets',
    method: 'post',
    response: ({
      headers,
      body,
    }: {
      headers: Record<string, string>
      body: { subject?: string; level?: number; message?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Math.max(0, ...tickets.map((t) => t.id)) + 1
      const ticket: Ticket = {
        id,
        subject: body?.subject ?? '未命名工单',
        level: (body?.level ?? 1) as Ticket['level'],
        status: 0,
        reopen_count: 0,
        last_reply_at: null,
        created_at: dayjs().toISOString(),
      }
      tickets.unshift(ticket)
      ticketMessages[id] = [
        { id: 1, sender_type: 0, message: body?.message ?? '', created_at: dayjs().toISOString() },
      ]
      return ok(ticket)
    },
  },
  {
    url: '/api/v1/tickets/:id',
    method: 'get',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Number(query.id)
      const t = tickets.find((x) => x.id === id)
      if (!t) return { code: 40400, message: '工单不存在', data: null }
      return ok({ ...t, messages: ticketMessages[id] ?? [] })
    },
  },
  {
    url: '/api/v1/tickets/:id/reply',
    method: 'post',
    response: ({
      headers,
      query,
      body,
    }: {
      headers: Record<string, string>
      query: { id?: string }
      body: { message?: string }
    }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Number(query.id)
      const t = tickets.find((x) => x.id === id)
      if (!t) return { code: 40400, message: '工单不存在', data: null }
      if (t.status === 2) return { code: 14001, message: '工单已关闭', data: null }
      const msgs = ticketMessages[id] ?? []
      msgs.push({
        id: msgs.length + 1,
        sender_type: 0,
        message: body?.message ?? '',
        created_at: dayjs().toISOString(),
      })
      t.status = 0
      t.last_reply_at = dayjs().toISOString()
      return ok({ ...t, messages: msgs })
    },
  },
  {
    url: '/api/v1/tickets/:id/close',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Number(query.id)
      const t = tickets.find((x) => x.id === id)
      if (!t) return { code: 40400, message: '工单不存在', data: null }
      t.status = 2
      return ok(t)
    },
  },
  {
    url: '/api/v1/tickets/:id/reopen',
    method: 'post',
    response: ({ headers, query }: { headers: Record<string, string>; query: { id?: string } }) => {
      if (!verifyAccess(headers)) return unauthorized()
      const id = Number(query.id)
      const t = tickets.find((x) => x.id === id)
      if (!t) return { code: 40400, message: '工单不存在', data: null }
      if (t.status !== 2) return { code: 40900, message: '状态冲突', data: null }
      if (t.reopen_count >= 1) return { code: 14002, message: '工单仅可重开一次', data: null }
      t.status = 0
      t.reopen_count += 1
      t.last_reply_at = dayjs().toISOString()
      return ok(t)
    },
  },
] as MockMethod[]
