/**
 * 适配缝（C3 transport-core 收敛）：既有引入路径与导出名保持可用，实现归属各域传输核。
 * 新调用方直接用 `@/stores/admin/<domain>` 传输核或组合式函数；本文件不新增逻辑。
 * 各域归属：overview/users/plans/nodes/orders/tickets(admin)/content/agent/system，决策见 ADR-0003。
 */
import { overviewTransport } from '@/stores/admin/overview'
import { usersTransport } from '@/stores/admin/users'
import { plansTransport } from '@/stores/admin/plans'
import { nodesTransport } from '@/stores/admin/nodes'
import { ordersTransport } from '@/stores/admin/orders'
import { adminTicketsTransport } from '@/stores/admin/tickets'
import { contentTransport } from '@/stores/admin/content'
import { agentAdminTransport } from '@/stores/admin/agent'
import { systemTransport } from '@/stores/admin/system'

/** 管理端 API(role=admin,契约见 docs/api/README.md 第 16 节) */
export const apiAdmin = {
  // 总览
  overview: overviewTransport.overview,

  // 用户
  users: usersTransport.users,
  updateUser: usersTransport.updateUser,
  adjustBalance: usersTransport.adjustBalance,

  // 用户管理增强(F05)
  exportUsersCSV: usersTransport.exportUsersCSV,
  batchUsers: usersTransport.batchUsers,
  sendMail: usersTransport.sendMail,
  resetUserSubToken: usersTransport.resetUserSubToken,

  // 套餐
  plans: plansTransport.plans,
  createPlan: plansTransport.createPlan,
  updatePlan: plansTransport.updatePlan,
  deletePlan: plansTransport.deletePlan,

  // 节点
  servers: nodesTransport.servers,
  createServer: nodesTransport.createServer,
  updateServer: nodesTransport.updateServer,
  deleteServer: nodesTransport.deleteServer,
  resetServerNodeKey: nodesTransport.resetServerNodeKey,

  // 节点批量 / 复制 / 排序（F09）
  batchServers: nodesTransport.batchServers,
  copyServer: nodesTransport.copyServer,
  sortServers: nodesTransport.sortServers,
  serverGroups: nodesTransport.serverGroups,
  createServerGroup: nodesTransport.createServerGroup,
  updateServerGroup: nodesTransport.updateServerGroup,
  deleteServerGroup: nodesTransport.deleteServerGroup,

  // 订单
  orders: ordersTransport.orders,
  refund: ordersTransport.refund,
  closeOrder: ordersTransport.closeOrder,

  // 工单
  tickets: adminTicketsTransport.tickets,
  ticketDetail: adminTicketsTransport.ticketDetail,
  replyTicket: adminTicketsTransport.replyTicket,
  closeTicket: adminTicketsTransport.closeTicket,
  // 佣金提现审核（F02）
  withdrawPay: adminTicketsTransport.withdrawPay,
  withdrawReject: adminTicketsTransport.withdrawReject,

  // 优惠券
  coupons: contentTransport.coupons,
  createCoupon: contentTransport.createCoupon,
  updateCoupon: contentTransport.updateCoupon,
  deleteCoupon: contentTransport.deleteCoupon,

  // 公告
  notices: contentTransport.notices,
  createNotice: contentTransport.createNotice,
  sortNotices: contentTransport.sortNotices,
  updateNotice: contentTransport.updateNotice,
  deleteNotice: contentTransport.deleteNotice,

  // 知识库
  knowledges: contentTransport.knowledges,
  createKnowledge: contentTransport.createKnowledge,
  sortKnowledges: contentTransport.sortKnowledges,
  updateKnowledge: contentTransport.updateKnowledge,
  deleteKnowledge: contentTransport.deleteKnowledge,

  // 知识库分类（F15）
  knowledgeCategories: contentTransport.knowledgeCategories,
  createKnowledgeCategory: contentTransport.createKnowledgeCategory,
  updateKnowledgeCategory: contentTransport.updateKnowledgeCategory,
  deleteKnowledgeCategory: contentTransport.deleteKnowledgeCategory,

  // 代理审批
  agentApplies: agentAdminTransport.agentApplies,
  approveAgent: agentAdminTransport.approveAgent,
  rejectAgent: agentAdminTransport.rejectAgent,

  // 佣金日志
  commissionLogs: agentAdminTransport.commissionLogs,

  // 审计日志(F08 只读)
  auditLogs: systemTransport.auditLogs,

  // 流量导入(模式 B)
  importTraffic: systemTransport.importTraffic,

  // 流量重置(F16)
  resetTraffic: systemTransport.resetTraffic,
  trafficResets: systemTransport.trafficResets,

  // 统计报表(F04)
  statOrders: overviewTransport.statOrders,
  statUsers: overviewTransport.statUsers,
  statTraffic: overviewTransport.statTraffic,

  // 邮件模板（F11）
  mailTemplates: systemTransport.mailTemplates,
  saveMailTemplate: systemTransport.saveMailTemplate,
  resetMailTemplate: systemTransport.resetMailTemplate,
  testMailTemplate: systemTransport.testMailTemplate,

  // 订阅模板（F10）
  subscriptionTemplates: systemTransport.subscriptionTemplates,
  saveSubscriptionTemplate: systemTransport.saveSubscriptionTemplate,
  resetSubscriptionTemplate: systemTransport.resetSubscriptionTemplate,
  previewSubscriptionTemplate: systemTransport.previewSubscriptionTemplate,

  // Telegram（F12）
  setupTelegramWebhook: systemTransport.setupTelegramWebhook,

  // 版本检查（F20）
  version: systemTransport.version,

  // 站点设置
  settings: systemTransport.settings,
  saveSetting: systemTransport.saveSetting,
}
