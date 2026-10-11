import { describe, expect, it } from 'vitest'
import { apiAdmin } from '@/api/admin'
import { overviewTransport } from '../overview'
import { usersTransport } from '../users'
import { plansTransport } from '../plans'
import { nodesTransport } from '../nodes'
import { ordersTransport } from '../orders'
import { adminTicketsTransport } from '../tickets'
import { contentTransport } from '../content'
import { agentAdminTransport } from '../agent'
import { systemTransport } from '../system'

// claim:apiAdmin 是纯适配缝,导出名稳定、实现与各域传输核同一引用;
// 改名/漏接/手写重复实现都会红。最怕的失败:视图调着旧名,实现却分叉。
const transports = [
  overviewTransport,
  usersTransport,
  plansTransport,
  nodesTransport,
  ordersTransport,
  adminTicketsTransport,
  contentTransport,
  agentAdminTransport,
  systemTransport,
]

const expectedKeys = [
  'adjustBalance',
  'agentApplies',
  'approveAgent',
  'auditLogs',
  'batchServers',
  'batchUsers',
  'closeOrder',
  'closeTicket',
  'commissionLogs',
  'copyServer',
  'coupons',
  'createCoupon',
  'createKnowledge',
  'createKnowledgeCategory',
  'createNotice',
  'createPlan',
  'createServer',
  'createServerGroup',
  'deleteCoupon',
  'deleteKnowledge',
  'deleteKnowledgeCategory',
  'deleteNotice',
  'deletePlan',
  'deleteServer',
  'deleteServerGroup',
  'exportUsersCSV',
  'importTraffic',
  'knowledgeCategories',
  'knowledges',
  'mailTemplates',
  'notices',
  'orders',
  'overview',
  'plans',
  'previewSubscriptionTemplate',
  'refund',
  'rejectAgent',
  'replyTicket',
  'resetMailTemplate',
  'resetServerNodeKey',
  'resetSubscriptionTemplate',
  'resetTraffic',
  'resetUserSubToken',
  'saveMailTemplate',
  'saveSetting',
  'saveSubscriptionTemplate',
  'sendMail',
  'serverGroups',
  'servers',
  'settings',
  'setupTelegramWebhook',
  'sortKnowledges',
  'sortNotices',
  'sortServers',
  'statOrders',
  'statTraffic',
  'statUsers',
  'subscriptionTemplates',
  'testMailTemplate',
  'ticketDetail',
  'tickets',
  'trafficResets',
  'updateCoupon',
  'updateKnowledge',
  'updateKnowledgeCategory',
  'updateNotice',
  'updatePlan',
  'updateServer',
  'updateServerGroup',
  'updateUser',
  'users',
  'version',
  'withdrawPay',
  'withdrawReject',
]

describe('apiAdmin 适配缝', () => {
  it('导出名全量稳定(增删改名即红)', () => {
    expect(Object.keys(apiAdmin).sort()).toEqual(expectedKeys)
  })

  it('每个导出都是某传输核的方法(无手写分叉)', () => {
    const impls = new Set<unknown>(transports.flatMap((t) => Object.values(t)))
    for (const [key, fn] of Object.entries(apiAdmin)) {
      expect(impls.has(fn), key).toBe(true)
    }
  })

  it('传输核方法无遗漏(新增须显式接入)', () => {
    const total = transports.reduce((n, t) => n + Object.keys(t).length, 0)
    expect(Object.keys(apiAdmin)).toHaveLength(total)
  })
})
