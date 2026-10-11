import { beforeEach, describe, expect, it, vi } from 'vitest'
import { download, http } from '@/utils/http'
import { overviewTransport } from '../overview'
import { usersTransport } from '../users'
import { plansTransport } from '../plans'
import { nodesTransport } from '../nodes'
import { ordersTransport } from '../orders'
import { adminTicketsTransport } from '../tickets'
import { contentTransport } from '../content'
import { agentAdminTransport } from '../agent'
import { systemTransport } from '../system'

vi.mock('@/utils/http', () => ({
  http: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  download: vi.fn(),
}))

const fns = {
  get: vi.mocked(http.get),
  post: vi.mocked(http.post),
  put: vi.mocked(http.put),
  delete: vi.mocked(http.delete),
  download: vi.mocked(download),
} as const

type Via = keyof typeof fns

interface Case {
  name: string
  run: () => unknown
  via: Via
  url: string
  arg?: unknown
}

// claim:各域传输核拥有本域 URL+形态;改 URL 只改一处,调参形状由本表锁定。
// 最怕的失败:路径段/方法拼错致管理端整域 404,适配缝还绿着。
const cases: Case[] = [
  // 总览
  {
    name: 'overview',
    run: () => overviewTransport.overview(),
    via: 'get',
    url: '/admin/stat/overview',
  },
  {
    name: 'statOrders',
    run: () => overviewTransport.statOrders({ days: 30 }),
    via: 'get',
    url: '/admin/stat/orders',
    arg: { query: { days: 30 } },
  },
  {
    name: 'statUsers',
    run: () => overviewTransport.statUsers({ days: 7 }),
    via: 'get',
    url: '/admin/stat/users',
    arg: { query: { days: 7 } },
  },
  {
    name: 'statTraffic',
    run: () => overviewTransport.statTraffic({}),
    via: 'get',
    url: '/admin/stat/traffic',
    arg: { query: {} },
  },
  // 用户
  {
    name: 'users',
    run: () => usersTransport.users({ page: 1, page_size: 10, keyword: 'a' }),
    via: 'get',
    url: '/admin/users',
    arg: { query: { page: 1, page_size: 10, keyword: 'a' } },
  },
  {
    name: 'users 越界钳制',
    run: () => usersTransport.users({ page: 0, page_size: 99 }),
    via: 'get',
    url: '/admin/users',
    arg: { query: { page: 1, page_size: 10 } },
  },
  {
    name: 'updateUser',
    run: () => usersTransport.updateUser(7, { banned: true }),
    via: 'put',
    url: '/admin/users/7',
    arg: { body: { banned: true } },
  },
  {
    name: 'adjustBalance',
    run: () => usersTransport.adjustBalance(7, { amount: 1.5, remark: 'x' }),
    via: 'post',
    url: '/admin/users/7/balance',
    arg: { body: { amount: 1.5, remark: 'x' } },
  },
  {
    name: 'exportUsersCSV',
    run: () => usersTransport.exportUsersCSV({ keyword: 'a' }),
    via: 'download',
    url: '/admin/users/export',
    arg: { keyword: 'a' },
  },
  {
    name: 'batchUsers',
    run: () => usersTransport.batchUsers({ action: 'ban', ids: [1, 2] }),
    via: 'post',
    url: '/admin/users/batch',
    arg: { body: { action: 'ban', ids: [1, 2] } },
  },
  {
    name: 'sendMail',
    run: () => usersTransport.sendMail({ ids: [1], subject: 's', body: 'b' }),
    via: 'post',
    url: '/admin/users/mail',
    arg: { body: { ids: [1], subject: 's', body: 'b' } },
  },
  {
    name: 'resetUserSubToken',
    run: () => usersTransport.resetUserSubToken(7),
    via: 'post',
    url: '/admin/users/7/sub-token/reset',
  },
  // 套餐
  { name: 'plans', run: () => plansTransport.plans(), via: 'get', url: '/admin/plans' },
  {
    name: 'createPlan',
    run: () => plansTransport.createPlan({ name: 'n', traffic_gb: 10 }),
    via: 'post',
    url: '/admin/plans',
    arg: { body: { name: 'n', traffic_gb: 10 } },
  },
  {
    name: 'updatePlan',
    run: () => plansTransport.updatePlan(3, { name: 'n', traffic_gb: 10 }),
    via: 'put',
    url: '/admin/plans/3',
    arg: { body: { name: 'n', traffic_gb: 10 } },
  },
  {
    name: 'deletePlan',
    run: () => plansTransport.deletePlan(3),
    via: 'delete',
    url: '/admin/plans/3',
  },
  // 节点
  { name: 'servers', run: () => nodesTransport.servers(), via: 'get', url: '/admin/servers' },
  {
    name: 'createServer',
    run: () =>
      nodesTransport.createServer({
        group_id: 1,
        name: 'n',
        type: 'vless',
        host: 'h',
        port: 443,
        config: '{}',
      }),
    via: 'post',
    url: '/admin/servers',
    arg: {
      body: { group_id: 1, name: 'n', type: 'vless', host: 'h', port: 443, config: '{}' },
    },
  },
  {
    name: 'updateServer',
    run: () =>
      nodesTransport.updateServer(5, {
        group_id: 1,
        name: 'n',
        type: 'vless',
        host: 'h',
        port: 443,
        config: '{}',
      }),
    via: 'put',
    url: '/admin/servers/5',
    arg: {
      body: { group_id: 1, name: 'n', type: 'vless', host: 'h', port: 443, config: '{}' },
    },
  },
  {
    name: 'deleteServer',
    run: () => nodesTransport.deleteServer(5),
    via: 'delete',
    url: '/admin/servers/5',
  },
  {
    name: 'resetServerNodeKey',
    run: () => nodesTransport.resetServerNodeKey(5),
    via: 'post',
    url: '/admin/servers/5/node-key/reset',
  },
  {
    name: 'batchServers',
    run: () => nodesTransport.batchServers({ action: 'delete', ids: [1] }),
    via: 'post',
    url: '/admin/servers/batch',
    arg: { body: { action: 'delete', ids: [1] } },
  },
  {
    name: 'copyServer',
    run: () => nodesTransport.copyServer(5),
    via: 'post',
    url: '/admin/servers/5/copy',
  },
  {
    name: 'sortServers',
    run: () => nodesTransport.sortServers({ items: [{ id: 1, sort: 0 }] }),
    via: 'post',
    url: '/admin/servers/sort',
    arg: { body: { items: [{ id: 1, sort: 0 }] } },
  },
  {
    name: 'serverGroups',
    run: () => nodesTransport.serverGroups(),
    via: 'get',
    url: '/admin/server-groups',
  },
  {
    name: 'createServerGroup',
    run: () => nodesTransport.createServerGroup({ name: 'g' }),
    via: 'post',
    url: '/admin/server-groups',
    arg: { body: { name: 'g' } },
  },
  {
    name: 'updateServerGroup',
    run: () => nodesTransport.updateServerGroup(2, { name: 'g' }),
    via: 'put',
    url: '/admin/server-groups/2',
    arg: { body: { name: 'g' } },
  },
  {
    name: 'deleteServerGroup',
    run: () => nodesTransport.deleteServerGroup(2),
    via: 'delete',
    url: '/admin/server-groups/2',
  },
  // 订单
  {
    name: 'orders',
    run: () => ordersTransport.orders({ page: 1, page_size: 10, status: 1 }),
    via: 'get',
    url: '/admin/orders',
    arg: { query: { page: 1, page_size: 10, status: 1 } },
  },
  {
    name: 'refund',
    run: () => ordersTransport.refund('NO1', { remark: 'r' }),
    via: 'post',
    url: '/admin/orders/NO1/refund',
    arg: { body: { remark: 'r' } },
  },
  {
    name: 'closeOrder',
    run: () => ordersTransport.closeOrder('NO1', { remark: 'r' }),
    via: 'post',
    url: '/admin/orders/NO1/close',
    arg: { body: { remark: 'r' } },
  },
  // 工单
  {
    name: 'tickets',
    run: () => adminTicketsTransport.tickets({ page: 2, page_size: 10 }),
    via: 'get',
    url: '/admin/tickets',
    arg: { query: { page: 2, page_size: 10 } },
  },
  {
    name: 'ticketDetail',
    run: () => adminTicketsTransport.ticketDetail(9),
    via: 'get',
    url: '/admin/tickets/9',
  },
  {
    name: 'replyTicket',
    run: () => adminTicketsTransport.replyTicket(9, { message: 'm' }),
    via: 'post',
    url: '/admin/tickets/9/reply',
    arg: { body: { message: 'm' } },
  },
  {
    name: 'closeTicket',
    run: () => adminTicketsTransport.closeTicket(9),
    via: 'post',
    url: '/admin/tickets/9/close',
  },
  {
    name: 'withdrawPay',
    run: () => adminTicketsTransport.withdrawPay(9, { remark: 'ok' }),
    via: 'post',
    url: '/admin/tickets/9/withdraw/pay',
    arg: { body: { remark: 'ok' } },
  },
  {
    name: 'withdrawReject',
    run: () => adminTicketsTransport.withdrawReject(9, { remark: 'no' }),
    via: 'post',
    url: '/admin/tickets/9/withdraw/reject',
    arg: { body: { remark: 'no' } },
  },
  // 内容
  { name: 'coupons', run: () => contentTransport.coupons(), via: 'get', url: '/admin/coupons' },
  {
    name: 'createCoupon',
    run: () => contentTransport.createCoupon({ code: 'C', type: 1, value: 5 }),
    via: 'post',
    url: '/admin/coupons',
    arg: { body: { code: 'C', type: 1, value: 5 } },
  },
  {
    name: 'updateCoupon',
    run: () => contentTransport.updateCoupon(1, { code: 'C', type: 1, value: 5 }),
    via: 'put',
    url: '/admin/coupons/1',
    arg: { body: { code: 'C', type: 1, value: 5 } },
  },
  {
    name: 'deleteCoupon',
    run: () => contentTransport.deleteCoupon(1),
    via: 'delete',
    url: '/admin/coupons/1',
  },
  { name: 'notices', run: () => contentTransport.notices(), via: 'get', url: '/admin/notices' },
  {
    name: 'createNotice',
    run: () => contentTransport.createNotice({ title: 't', content: 'c' }),
    via: 'post',
    url: '/admin/notices',
    arg: { body: { title: 't', content: 'c' } },
  },
  {
    name: 'sortNotices',
    run: () => contentTransport.sortNotices({ items: [{ id: 1, sort: 0 }] }),
    via: 'post',
    url: '/admin/notices/sort',
    arg: { body: { items: [{ id: 1, sort: 0 }] } },
  },
  {
    name: 'updateNotice',
    run: () => contentTransport.updateNotice(1, { title: 't', content: 'c' }),
    via: 'put',
    url: '/admin/notices/1',
    arg: { body: { title: 't', content: 'c' } },
  },
  {
    name: 'deleteNotice',
    run: () => contentTransport.deleteNotice(1),
    via: 'delete',
    url: '/admin/notices/1',
  },
  {
    name: 'knowledges',
    run: () => contentTransport.knowledges(),
    via: 'get',
    url: '/admin/knowledges',
  },
  {
    name: 'createKnowledge',
    run: () => contentTransport.createKnowledge({ category: 'c', title: 't', body: 'b' }),
    via: 'post',
    url: '/admin/knowledges',
    arg: { body: { category: 'c', title: 't', body: 'b' } },
  },
  {
    name: 'sortKnowledges',
    run: () => contentTransport.sortKnowledges({ items: [{ id: 1, sort: 0 }] }),
    via: 'post',
    url: '/admin/knowledges/sort',
    arg: { body: { items: [{ id: 1, sort: 0 }] } },
  },
  {
    name: 'updateKnowledge',
    run: () => contentTransport.updateKnowledge(1, { category: 'c', title: 't', body: 'b' }),
    via: 'put',
    url: '/admin/knowledges/1',
    arg: { body: { category: 'c', title: 't', body: 'b' } },
  },
  {
    name: 'deleteKnowledge',
    run: () => contentTransport.deleteKnowledge(1),
    via: 'delete',
    url: '/admin/knowledges/1',
  },
  {
    name: 'knowledgeCategories',
    run: () => contentTransport.knowledgeCategories({ language: 'zh-CN' }),
    via: 'get',
    url: '/admin/knowledge-categories',
    arg: { query: { language: 'zh-CN' } },
  },
  {
    name: 'createKnowledgeCategory',
    run: () => contentTransport.createKnowledgeCategory({ language: 'zh-CN', name: 'n' }),
    via: 'post',
    url: '/admin/knowledge-categories',
    arg: { body: { language: 'zh-CN', name: 'n' } },
  },
  {
    name: 'updateKnowledgeCategory',
    run: () => contentTransport.updateKnowledgeCategory(1, { name: 'n' }),
    via: 'put',
    url: '/admin/knowledge-categories/1',
    arg: { body: { name: 'n' } },
  },
  {
    name: 'deleteKnowledgeCategory',
    run: () => contentTransport.deleteKnowledgeCategory(1),
    via: 'delete',
    url: '/admin/knowledge-categories/1',
  },
  // 代理与佣金
  {
    name: 'agentApplies',
    run: () => agentAdminTransport.agentApplies({ page: 1, page_size: 10, status: 0 }),
    via: 'get',
    url: '/admin/agent/applies',
    arg: { query: { page: 1, page_size: 10, status: 0 } },
  },
  {
    name: 'approveAgent',
    run: () => agentAdminTransport.approveAgent(4, { remark: 'ok' }),
    via: 'post',
    url: '/admin/agent/applies/4/approve',
    arg: { body: { remark: 'ok' } },
  },
  {
    name: 'rejectAgent',
    run: () => agentAdminTransport.rejectAgent(4, { remark: 'no' }),
    via: 'post',
    url: '/admin/agent/applies/4/reject',
    arg: { body: { remark: 'no' } },
  },
  {
    name: 'commissionLogs',
    run: () => agentAdminTransport.commissionLogs({ page: 1, page_size: 10, status: '' }),
    via: 'get',
    url: '/admin/commission-logs',
    arg: { query: { page: 1, page_size: 10, status: '' } },
  },
  // 系统
  {
    name: 'auditLogs',
    run: () =>
      systemTransport.auditLogs({ page: 1, page_size: 20, admin_id: 1, action: 'ban_user' }),
    via: 'get',
    url: '/admin/audit-logs',
    arg: { query: { page: 1, page_size: 20, admin_id: 1, action: 'ban_user' } },
  },
  {
    name: 'importTraffic',
    run: () =>
      systemTransport.importTraffic({ items: [{ user_id: 1, date: '2026-01-01', u: 1, d: 2 }] }),
    via: 'post',
    url: '/admin/traffic/import',
    arg: { body: { items: [{ user_id: 1, date: '2026-01-01', u: 1, d: 2 }] } },
  },
  {
    name: 'resetTraffic',
    run: () => systemTransport.resetTraffic({ user_ids: [1], mode: 'clear_usage' }),
    via: 'post',
    url: '/admin/traffic/reset',
    arg: { body: { user_ids: [1], mode: 'clear_usage' } },
  },
  {
    name: 'trafficResets',
    run: () => systemTransport.trafficResets({ page: 1, page_size: 10, user_id: '' }),
    via: 'get',
    url: '/admin/traffic/resets',
    arg: { query: { page: 1, page_size: 10, user_id: '' } },
  },
  {
    name: 'mailTemplates',
    run: () => systemTransport.mailTemplates(),
    via: 'get',
    url: '/admin/mail-templates',
  },
  {
    name: 'saveMailTemplate',
    run: () => systemTransport.saveMailTemplate('captcha', { subject: 's', body: 'b' }),
    via: 'put',
    url: '/admin/mail-templates/captcha',
    arg: { body: { subject: 's', body: 'b' } },
  },
  {
    name: 'resetMailTemplate',
    run: () => systemTransport.resetMailTemplate('captcha'),
    via: 'delete',
    url: '/admin/mail-templates/captcha',
  },
  {
    name: 'testMailTemplate',
    run: () => systemTransport.testMailTemplate('captcha', { to_email: 'a@b.c' }),
    via: 'post',
    url: '/admin/mail-templates/captcha/test',
    arg: { body: { to_email: 'a@b.c' } },
  },
  {
    name: 'subscriptionTemplates',
    run: () => systemTransport.subscriptionTemplates(),
    via: 'get',
    url: '/admin/subscription-templates',
  },
  {
    name: 'saveSubscriptionTemplate',
    run: () => systemTransport.saveSubscriptionTemplate('clash', { content: 'c' }),
    via: 'put',
    url: '/admin/subscription-templates/clash',
    arg: { body: { content: 'c' } },
  },
  {
    name: 'resetSubscriptionTemplate',
    run: () => systemTransport.resetSubscriptionTemplate('clash'),
    via: 'delete',
    url: '/admin/subscription-templates/clash',
  },
  {
    name: 'previewSubscriptionTemplate',
    run: () => systemTransport.previewSubscriptionTemplate('clash'),
    via: 'post',
    url: '/admin/subscription-templates/clash/preview',
  },
  {
    name: 'setupTelegramWebhook',
    run: () => systemTransport.setupTelegramWebhook(),
    via: 'post',
    url: '/admin/telegram/webhook/setup',
  },
  { name: 'version', run: () => systemTransport.version(), via: 'get', url: '/admin/version' },
  { name: 'settings', run: () => systemTransport.settings(), via: 'get', url: '/admin/settings' },
  {
    name: 'saveSetting',
    run: () => systemTransport.saveSetting({ key: 'site', value: '{}' }),
    via: 'put',
    url: '/admin/settings',
    arg: { body: { key: 'site', value: '{}' } },
  },
]

describe('admin transports', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    for (const fn of Object.values(fns)) fn.mockResolvedValue(null as never)
  })

  it.each(cases)('$name → $via $url', async ({ run, via, url, arg }) => {
    await run()
    const fn = fns[via]
    if (arg === undefined) expect(fn).toHaveBeenCalledWith(url)
    else expect(fn).toHaveBeenCalledWith(url, arg)
  })
})
